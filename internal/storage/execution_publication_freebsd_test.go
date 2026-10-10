// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Only owned native fixtures transfer an initialized original receipt. Production
// OpenExecutionDomainLeases never initializes or transfers such a capability.
func journaledFixture(t *testing.T, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error)) *ExecutionDomainLeases {
	t.Helper()
	for _, path := range []string{journal, ledger} {
		if os.Mkdir(path, 0700) != nil || os.WriteFile(filepath.Join(path, stateWriterName), nil, 0600) != nil {
			t.Fatal("owned journaled pair")
		}
	}
	j, e := openExecutionJournalStore(journal)
	if e != nil {
		t.Fatal(e)
	}
	l, e := openStateDirectory(ledger, false)
	if e != nil {
		t.Fatal(e)
	}
	p := &ExecutionDomainLeases{native: &executionDomainLeaseNative{j, l}}
	p.origin = p
	t.Cleanup(func() { _ = p.Close() })
	observed, e := p.Observe(context.Background(), strings.Repeat("1", 32), read)
	if e != nil {
		t.Fatal(e)
	}
	record := executionJournalFixture()
	record.Binding = observed.binding
	record.Census = observed.census
	if j.initialize(record, nil) != nil {
		t.Fatal("owned original journal acknowledgement")
	}
	return p
}
func TestNativeJournaledPublicationUFS(t *testing.T) { nativeJournaledPublication(t, "ufs") }
func TestNativeJournaledPublicationZFS(t *testing.T) { nativeJournaledPublication(t, "zfs") }
func nativeJournaledPublication(t *testing.T, kind string) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("owned root state VM required")
	}
	f := newRestoreFixture(t, kind)
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("journaled publication assertion", name)
		}
		markers = append(markers, name+"=pass")
	}
	ctx := context.Background()
	fixture := func(name string) *ExecutionDomainLeases {
		return journaledFixture(t, filepath.Join(f.journal, name), filepath.Join(f.ledger, name), f.read)
	}
	policy, e := DecodeOwnershipRecord(ownershipGolden())
	if e != nil {
		t.Fatal(e)
	}
	p := fixture("positive")
	initial := p.native.journal.ack
	copied := &ExecutionDomainLeases{origin: p, native: p.native, binding: p.binding, bound: true}
	check("copied_owner_denial", errors.Is(copied.publishJournaledOwnership(ctx, policy, 0, f.read, nil), ErrDenied))
	check("nil_owner_denial", errors.Is((*ExecutionDomainLeases)(nil).publishJournaledOwnership(ctx, policy, 0, f.read, nil), ErrDenied))
	check("nil_context_denial", errors.Is(p.publishJournaledOwnership(nil, policy, 0, f.read, nil), ErrDenied))
	check("nil_reader_denial", errors.Is(p.publishJournaledOwnership(ctx, policy, 0, nil, nil), ErrDenied))
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	check("initial_context_denial", errors.Is(p.publishJournaledOwnership(cancelled, policy, 0, f.read, nil), context.Canceled) && !p.blocked)
	var trace []string
	wrap := func(stage string, tx ownershipPublication) ownershipPublication {
		sf, sd := tx.syncFile, tx.syncDirectory
		tx.syncFile = func() error { trace = append(trace, stage+"-file"); return sf() }
		tx.syncDirectory = func() error { trace = append(trace, stage+"-directory"); return sd() }
		if stage == "ledger" {
			original := tx.write
			tx.write = func() error {
				check("pending_ack_before_ledger", p.native.journal.ack.Metadata.Pending && p.native.journal.ack.Census == initial.Census)
				if x, e := openStateDirectory(filepath.Join(f.ledger, "positive"), false); x != nil || !errors.Is(e, ErrStateLocked) {
					if x != nil {
						_ = x.Close()
					}
					return ErrDenied
				}
				if x, e := openExecutionJournalStore(filepath.Join(f.journal, "positive")); x != nil || !errors.Is(e, ErrStateLocked) {
					if x != nil {
						_ = x.close()
					}
					return ErrDenied
				}
				check("both_lifetime_locks_held", true)
				return original()
			}
		}
		if stage == "confirmation" {
			original := tx.write
			tx.write = func() error {
				data, e := os.ReadFile(filepath.Join(f.ledger, "positive", "ownership-100.json"))
				if e != nil || !bytes.Equal(data, ownershipGolden()) {
					return ErrDrift
				}
				check("ledger_bytes_before_confirmation", true)
				check("pending_retained_until_confirmation", p.native.journal.ack.Metadata.Pending)
				return original()
			}
		}
		return tx
	}
	check("ownership_creation", p.publishJournaledOwnership(ctx, policy, 0, f.read, wrap) == nil)
	check("actual_sync_order", reflect.DeepEqual(trace, []string{"pending-file", "pending-directory", "ledger-file", "ledger-directory", "confirmation-file", "confirmation-directory"}))
	confirmed := p.native.journal.ack
	_, census, e := p.native.ledger.inspectExecutionCensus(p.binding, defaultStateCatalogLimits, nil)
	check("confirmed_actual_census", e == nil && confirmed.Census == census && !confirmed.Metadata.Pending && confirmed.Generation == initial.Generation+2)
	policy.Revision++
	check("ownership_successor", p.publishJournaledOwnership(ctx, policy, 1, f.read, nil) == nil)
	check("successor_advances_twice", p.native.journal.ack.Generation == initial.Generation+4)
	job := nativePropertyJobFixture(t)
	check("job_creation", p.publishJournaledPropertyJob(ctx, job, 0, f.read, nil) == nil)
	pending := p.native.journal.ack
	pending.Generation++
	pending.Native = journalNativeFixture(pending)
	check("fixture_native_pending_ack", p.native.journal.publish(pending, nil) == nil)
	cancelledJob, e := TransitionPropertyJob(job, JobCancelled, job.job.History[0].At+1, "operator-cancelled", "")
	if e != nil {
		t.Fatal(e)
	}
	check("job_successor", p.publishJournaledPropertyJob(ctx, cancelledJob, 1, f.read, nil) == nil)
	check("native_pending_preserved", p.native.journal.ack.Native == pending.Native && p.native.journal.ack.Native.Pending && !p.native.journal.ack.Metadata.Pending)
	wire, e := os.ReadFile(filepath.Join(f.ledger, "positive", "job-"+job.job.ID+".json"))
	if e != nil {
		t.Fatal(e)
	}
	reservations, e := BuildJobReservations([][]byte{wire})
	check("terminal_hold_preserved", e == nil && len(reservations.Pools()) == 1 && len(reservations.Pools()[0].Jobs) == 1)
	saved := restoreJournalSnapshot(t, filepath.Join(f.journal, "positive"))
	check("original_close", p.Close() == nil)
	reopened, e := OpenExecutionDomainLeases(ctx, filepath.Join(f.journal, "positive"), filepath.Join(f.ledger, "positive"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	_, e = reopened.Observe(ctx, strings.Repeat("1", 32), f.read)
	check("reopen_observation", e == nil)
	nextPolicy := policy
	nextPolicy.Revision++
	check("reopen_no_receipt_denial", errors.Is(reopened.publishJournaledOwnership(ctx, nextPolicy, 2, f.read, nil), ErrDenied) && !reopened.native.journal.acknowledged)
	check("reopen_exact_history_preserved", reflect.DeepEqual(saved, restoreJournalSnapshot(t, filepath.Join(f.journal, "positive"))))
	check("reopen_close", reopened.Close() == nil)
	check("closed_owner_denial", errors.Is(p.publishJournaledOwnership(ctx, nextPolicy, 2, f.read, nil), ErrDenied))
	// Every failure leaves both live stores retired, with their actual visible
	// old/new bytes intact; even a clean namespace never permits an automatic retry.
	for _, stage := range []string{"pending", "ledger", "confirmation"} {
		for _, point := range []string{"write", "syncFile", "closeFile", "verify", "rename", "syncDirectory", "after-sync-drift"} {
			name := stage + "-" + point
			pair := fixture(name)
			failure := func(phase string, tx ownershipPublication) ownershipPublication {
				if phase != stage {
					return tx
				}
				bad := func() error { return syscall.EIO }
				switch point {
				case "write":
					tx.write = bad
				case "syncFile":
					tx.syncFile = bad
				case "closeFile":
					old := tx.closeFile
					tx.closeFile = func() error { _ = old(); return syscall.EIO }
				case "verify":
					tx.verify = bad
				case "rename":
					tx.rename = bad
				case "syncDirectory":
					tx.syncDirectory = bad
				case "after-sync-drift":
					old := tx.syncDirectory
					tx.syncDirectory = func() error {
						if e := old(); e != nil {
							return e
						}
						path := filepath.Join(f.journal, name, executionJournalName)
						if stage == "ledger" {
							path = filepath.Join(f.ledger, name, "ownership-100.json")
						}
						return os.WriteFile(path, []byte("damaged-owned-fixture"), 0600)
					}
				}
				return tx
			}
			e := pair.publishJournaledOwnership(ctx, journaledPolicyFixture(t), 0, f.read, failure)
			check(name+"_denial", e != nil)
			check(name+"_retirement", pair.blocked && pair.native.journal.failed && pair.native.ledger.publicationBlocked)
			beforeJ := restoreJournalSnapshot(t, filepath.Join(f.journal, name))
			beforeL := restoreJournalSnapshot(t, filepath.Join(f.ledger, name))
			retry := pair.publishJournaledOwnership(ctx, journaledPolicyFixture(t), 0, f.read, nil)
			check(name+"_retry_preserves_bytes", errors.Is(retry, ErrStateUncertain) && reflect.DeepEqual(beforeJ, restoreJournalSnapshot(t, filepath.Join(f.journal, name))) && reflect.DeepEqual(beforeL, restoreJournalSnapshot(t, filepath.Join(f.ledger, name))))
			if pair.Close() != nil {
				t.Fatal("owned failed pair close")
			}
		}
	}
	for _, mode := range []string{"rollback", "journal-inode", "pending-cancel", "disarmed", "overflow", "orphan"} {
		pair := fixture(mode)
		next := journaledPolicyFixture(t)
		var seam func(string, ownershipPublication) ownershipPublication
		callCtx := ctx
		switch mode {
		case "rollback":
			if pair.publishJournaledOwnership(ctx, next, 0, f.read, nil) != nil || os.Remove(filepath.Join(f.ledger, mode, "ownership-100.json")) != nil {
				t.Fatal("owned rollback fixture")
			}
		case "journal-inode":
			path := filepath.Join(f.journal, mode, executionJournalName)
			data, e := os.ReadFile(path)
			if e != nil || os.WriteFile(path+".owned", data, 0600) != nil || os.Rename(path+".owned", path) != nil {
				t.Fatal("owned journal replacement")
			}
		case "pending-cancel":
			var cancel context.CancelFunc
			callCtx, cancel = context.WithCancel(ctx)
			seam = func(stage string, tx ownershipPublication) ownershipPublication {
				if stage == "pending" {
					old := tx.syncDirectory
					tx.syncDirectory = func() error { e := old(); cancel(); return e }
				}
				return tx
			}
		case "disarmed":
			j := pair.native.journal.ack
			j.Generation++
			j.Disarmed = true
			if pair.native.journal.publish(j, nil) != nil {
				t.Fatal("owned disarmed fixture")
			}
		case "overflow":
			// Accepted codec metadata and exact original receipt, fixture-only high epoch.
			j := pair.native.journal.ack
			j.Generation = ^uint64(0) - 1
			data, e := encodeExecutionJournal(j)
			if e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(f.journal, mode, executionJournalName)
			if os.WriteFile(path, data, 0600) != nil {
				t.Fatal("owned epoch fixture")
			}
			snapshot, e := pair.native.journal.read(nil)
			if e != nil {
				t.Fatal(e)
			}
			pair.native.journal.lease.mu.Lock()
			actual, e := pair.native.journal.inspectLocked("", nil)
			pair.native.journal.lease.mu.Unlock()
			if e != nil {
				t.Fatal(e)
			}
			pair.native.journal.ack = snapshot
			pair.native.journal.ackBytes = actual.data
			pair.native.journal.ackMetadata = actual.metadata
		case "orphan":
			if os.WriteFile(filepath.Join(f.ledger, mode, ".owned-orphan"), []byte("preserve"), 0600) != nil {
				t.Fatal("owned orphan")
			}
		}
		check(mode+"_denial", pair.publishJournaledOwnership(callCtx, next, 0, f.read, seam) != nil)
		check(mode+"_retirement", pair.blocked && pair.native.journal.failed && pair.native.ledger.publicationBlocked)
		if mode == "pending-cancel" {
			_, e := os.Lstat(filepath.Join(f.ledger, mode, "ownership-100.json"))
			check("pending_cancel_no_ledger_effect", os.IsNotExist(e) && pair.native.journal.ack.Metadata.Pending)
		}
		if pair.Close() != nil {
			t.Fatal("owned negative pair close")
		}
	}
	t.Log("JOURNALED_PUBLICATION_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
}
func journaledPolicyFixture(t *testing.T) Policy {
	t.Helper()
	p, e := DecodeOwnershipRecord(ownershipGolden())
	if e != nil {
		t.Fatal(e)
	}
	return p
}

func TestJournaledPublicationCutProcess(t *testing.T) {
	phase := os.Getenv("HORVIUM_JOURNALED_CUT")
	if phase == "" {
		t.Skip("owned reexec only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("disposable root required")
	}
	read := func(context.Context, int) (ExecutionBackingObservation, error) {
		return ExecutionBackingObservation{Kind: "ufs", UFS: [2]uint32{11, 22}}, nil
	}
	p := journaledFixture(t, os.Getenv("HORVIUM_JOURNALED_JOURNAL"), os.Getenv("HORVIUM_JOURNALED_LEDGER"), read)
	policy := journaledPolicyFixture(t)
	if phase == "uid" {
		if unix.Setresuid(1, 1, 1) != nil {
			t.Fatal("owned credential drop")
		}
		if !errors.Is(p.publishJournaledOwnership(context.Background(), policy, 0, read, nil), ErrDenied) {
			t.Fatal("kernel UID accepted")
		}
		fmt.Println("JOURNALED_UID_RESULT denied=pass")
		return
	}
	e := p.publishJournaledOwnership(context.Background(), policy, 0, read, func(stage string, tx ownershipPublication) ownershipPublication {
		if stage == phase {
			old := tx.syncDirectory
			tx.syncDirectory = func() error {
				if e := old(); e != nil {
					return e
				}
				fmt.Println("JOURNALED_CUT_REACHED " + phase)
				_, _ = io.Copy(io.Discard, os.Stdin)
				return ErrDenied
			}
		}
		return tx
	})
	t.Fatal("unreached journaled cut", e)
}
func TestNativeJournaledPublicationCuts(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable root state VM required")
	}
	anchor, e := nativeUFSStateAnchor(t, "horvium-journaled-cuts-", false)
	if e != nil {
		t.Fatal(e)
	}
	executable, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	read := func(context.Context, int) (ExecutionBackingObservation, error) {
		return ExecutionBackingObservation{Kind: "ufs", UFS: [2]uint32{11, 22}}, nil
	}
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("journaled cut assertion", name)
		}
		markers = append(markers, name+"=pass")
	}
	for _, phase := range []string{"pending", "ledger", "confirmation", "uid"} {
		journal, ledger := filepath.Join(anchor, phase+"-journal"), filepath.Join(anchor, phase+"-ledger")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		child := exec.CommandContext(ctx, executable, "-test.v", "-test.run=^TestJournaledPublicationCutProcess$")
		child.Env = append(os.Environ(), "HORVIUM_JOURNALED_CUT="+phase, "HORVIUM_JOURNALED_JOURNAL="+journal, "HORVIUM_JOURNALED_LEDGER="+ledger)
		input, e := child.StdinPipe()
		if e != nil {
			t.Fatal(e)
		}
		output, e := child.StdoutPipe()
		if e != nil {
			t.Fatal(e)
		}
		var diagnostic bytes.Buffer
		child.Stderr = &diagnostic
		if child.Start() != nil {
			t.Fatal("owned journaled child start")
		}
		waited := false
		t.Cleanup(func() {
			_ = input.Close()
			if !waited {
				_ = child.Process.Kill()
				_ = child.Wait()
			}
			cancel()
		})
		scan := bufio.NewScanner(output)
		reached := false
		expected := "JOURNALED_CUT_REACHED " + phase
		if phase == "uid" {
			expected = "JOURNALED_UID_RESULT denied=pass"
		}
		for scan.Scan() {
			if scan.Text() == expected {
				reached = true
				break
			}
		}
		check(phase+"_barrier", reached && ctx.Err() == nil)
		var killErr error
		if phase != "uid" {
			killErr = child.Process.Kill()
		}
		waitErr := child.Wait()
		waited = true
		_ = input.Close()
		cancel()
		if phase == "uid" {
			check("uid_child_reaped", waitErr == nil)
		} else {
			exit, ok := waitErr.(*exec.ExitError)
			check(phase+"_killed_reaped", killErr == nil && ok && exit.Sys().(syscall.WaitStatus).Signal() == syscall.SIGKILL)
		}
		p, e := OpenExecutionDomainLeases(context.Background(), journal, ledger)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { _ = p.Close() })
		got, e := p.Observe(context.Background(), strings.Repeat("1", 32), read)
		check(phase+"_reopen_observation", e == nil && got.journalPresent && got.journal.Owner.State == "active")
		shouldPending := phase == "pending" || phase == "ledger"
		check(phase+"_pending_preserved", got.journal.Metadata.Pending == shouldPending)
		_, recordErr := os.Lstat(filepath.Join(ledger, "ownership-100.json"))
		if phase == "pending" || phase == "uid" {
			check(phase+"_ledger_visibility", os.IsNotExist(recordErr) && got.census == got.journal.Census)
		} else {
			check(phase+"_ledger_visibility", recordErr == nil && ((phase == "ledger" && got.census == got.journal.Metadata.After) || (phase == "confirmation" && got.census == got.journal.Census)))
		}
		beforeJ := restoreJournalSnapshot(t, journal)
		beforeL := restoreJournalSnapshot(t, ledger)
		check(phase+"_no_reconstructed_writer", errors.Is(p.publishJournaledOwnership(context.Background(), journaledPolicyFixture(t), 0, read, nil), ErrDenied) && !p.native.journal.acknowledged)
		check(phase+"_history_preserved", reflect.DeepEqual(beforeJ, restoreJournalSnapshot(t, journal)) && reflect.DeepEqual(beforeL, restoreJournalSnapshot(t, ledger)))
		check(phase+"_close", p.Close() == nil)
	}
	t.Log("JOURNALED_PUBLICATION_CUTS_RESULT " + strings.Join(markers, " "))
}
