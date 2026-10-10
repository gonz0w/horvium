// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

var retirementFaults = []string{"boot-before", "system-before", "ledger-before", "journal-before", "ledger-entry", "write", "syncFile", "closeFile", "verify", "rename", "syncDirectory", "after-sync-file", "boot-after", "system-after", "journal-after", "ledger-after", "cancel-after", "lost-ack", "after-sync-directory", "after-sync-ledger", "after-sync-lock", "metadata-pending", "native-pending", "disarmed", "no-ack"}

func TestNativeClosedStartupRetirementUFS(t *testing.T) { nativeClosedStartupRetirement(t, "ufs") }
func TestNativeClosedStartupRetirementZFS(t *testing.T) { nativeClosedStartupRetirement(t, "zfs") }
func nativeClosedStartupRetirement(t *testing.T, kind string) {
	restartStorageGuard(t)
	f := newRestoreFixture(t, kind)
	ctx := context.Background()
	use := activationSystemFixture(t, f.journal, f.ledger, f.read)
	sys := func(context.Context) (PoolSystemUse, error) { return use, nil }
	original := func(name string) (*ExecutionStartupReview, *executionStartupNative, *ExecutionRestartActivation, string, string) {
		j, l, _ := restartSeed(t, f, "retirement-"+name)
		r, n, e := reviewExecutionStartup(ctx, j, l, f.read, sys, activationBoot)
		if e != nil {
			t.Fatal(e)
		}
		defer func() {
			if r != nil {
				_ = r.Close()
			}
		}()
		a, e := r.Activate(ctx)
		if e != nil {
			t.Fatal("original active", e)
		}
		t.Cleanup(func() { _ = a.Close() })
		return r, n, a, j, l
	}
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("restart retirement", name)
		}
		for _, s := range seen {
			if s == name+"=pass" {
				t.Fatal("duplicate control")
			}
		}
		seen = append(seen, name+"=pass")
	}
	r, n, a, j, l := original("positive")
	defer a.Close()
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	for _, x := range []struct {
		name string
		c    context.Context
	}{{"nil", nil}, {"cancelled", cancelled}} {
		check(x.name+"_denied", a.FinishIdle(x.c) != nil)
		_, e := a.Observe(ctx)
		check(x.name+"_original_survives", e == nil)
	}
	copy := new(ExecutionRestartActivation)
	reflect.ValueOf(copy).Elem().Set(reflect.ValueOf(a).Elem())
	check("copied_denied", copy.FinishIdle(ctx) != nil)
	wire, e := json.Marshal(a)
	var decoded ExecutionRestartActivation
	check("decoded_denied", e == nil && string(wire) == "{}" && json.Unmarshal(wire, &decoded) == nil && decoded.FinishIdle(ctx) != nil)
	before := n.snapshot.record
	ledgerBefore := freshNamespaceSnapshot(t, l)
	var phases []string
	a.finish = func(c context.Context) error {
		return n.finishIdle(c, func(tx ownershipPublication) ownershipPublication {
			sf, sd := tx.syncFile, tx.syncDirectory
			tx.syncFile = func() error {
				e := sf()
				if e == nil {
					phases = append(phases, "file")
				}
				return e
			}
			tx.syncDirectory = func() error {
				e := sd()
				if e == nil {
					phases = append(phases, "directory")
				}
				return e
			}
			return tx
		})
	}
	check("original_finish", a.FinishIdle(ctx) == nil)
	check("real_sync_order", reflect.DeepEqual(phases, []string{"file", "directory"}))
	expected := before
	expected.Generation++
	expected.Owner.State = "closed"
	actual, e := n.pair.native.journal.read(nil)
	check("exact_closed_record", e == nil && actual == expected && n.pair.native.journal.ack == expected)
	check("exact_ledger", reflect.DeepEqual(ledgerBefore, freshNamespaceSnapshot(t, l)))
	check("stores_retired", n.pair.blocked && n.pair.native.journal.failed && n.pair.native.journal.lease.publicationBlocked && n.pair.native.ledger.publicationBlocked)
	check("both_leases_held", restartLocksHeld(t, j, l))
	check("old_review_cannot_release", r.Close() != nil && restartLocksHeld(t, j, l))
	bj, bl := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
	check("repeat_denied", a.FinishIdle(ctx) != nil)
	_, e = a.Observe(ctx)
	check("observation_denied", e != nil)
	check("repeat_preserves", reflect.DeepEqual(bj, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, l)))
	check("release_once", a.Close() == nil && a.Close() == nil)
	check("released_leases", restartLocksReleased(t, j, l))
	q, nn, e := reviewExecutionStartup(ctx, j, l, f.read, sys, activationBoot)
	check("reopen_metadata_only", e == nil && q != nil && !nn.pair.native.journal.acknowledged && !nn.pair.native.journal.initializable && q.Close() == nil)
	fresh, e := activateFreshExecutionJournal(ctx, j, l, f.read, sys, activationBoot)
	check("no_recommission", e != nil && fresh == nil)
	for _, mode := range retirementFaults {
		_, n, a, j, l := original("fault-" + mode)
		c, stop := context.WithCancel(ctx)
		n.boot = func(c context.Context) ([16]byte, error) {
			v, e := activationBoot(c)
			if mode == "boot-before" || mode == "boot-after" && n.pair.native.journal.ack.Owner.State == "closed" {
				v[0] ^= 1
			}
			return v, e
		}
		n.system = func(c context.Context) (PoolSystemUse, error) {
			closed := n.pair.native.journal.ack.Owner.State == "closed"
			if mode == "system-before" || mode == "system-after" && closed {
				return PoolSystemUse{}, ErrUnavailable
			}
			if closed {
				switch mode {
				case "journal-after":
					_ = os.Chmod(j, 0700)
				case "ledger-after":
					_ = os.Chmod(l, 0700)
				case "cancel-after":
					stop()
				}
			}
			return use, nil
		}
		switch mode {
		case "journal-before":
			if os.Chmod(filepath.Join(j, executionJournalName), 0600) != nil {
				t.Fatal("journal touch")
			}
		case "ledger-before":
			if os.Chmod(l, 0700) != nil {
				t.Fatal("ledger touch")
			}
		case "ledger-entry":
			if os.WriteFile(filepath.Join(l, "unexpected"), []byte("fixture"), 0600) != nil {
				t.Fatal("ledger entry")
			}
		case "no-ack":
			n.pair.native.journal.acknowledged = false
		case "metadata-pending", "native-pending", "disarmed":
			record := n.snapshot.record
			switch mode {
			case "metadata-pending":
				record.Metadata = journalMetadataFixture(record)
			case "native-pending":
				record.Native = journalNativeFixture(record)
			case "disarmed":
				record.Disarmed = true
			}
			data, e := encodeExecutionJournal(record)
			if e != nil || os.WriteFile(filepath.Join(j, executionJournalName), data, 0600) != nil {
				t.Fatal("controlled unsafe record", e)
			}
		}
		a.finish = func(c context.Context) error {
			return n.finishIdle(c, func(tx ownershipPublication) ownershipPublication {
				switch mode {
				case "write":
					tx.write = func() error { return ErrDenied }
				case "syncFile":
					tx.syncFile = func() error { return ErrDenied }
				case "closeFile":
					old := tx.closeFile
					tx.closeFile = func() error { _ = old(); return ErrDenied }
				case "verify":
					tx.verify = func() error { return ErrDenied }
				case "rename":
					tx.rename = func() error { return ErrDenied }
				case "syncDirectory":
					tx.syncDirectory = func() error { return ErrDenied }
				case "lost-ack":
					tx.syncDirectory = func() error { return nil }
				case "after-sync-file", "after-sync-directory", "after-sync-ledger", "after-sync-lock":
					old := tx.syncDirectory
					tx.syncDirectory = func() error {
						if e := old(); e != nil {
							return e
						}
						path, modebits := filepath.Join(j, executionJournalName), os.FileMode(0600)
						switch mode {
						case "after-sync-directory":
							path, modebits = j, 0700
						case "after-sync-ledger":
							path, modebits = l, 0700
						case "after-sync-lock":
							path = filepath.Join(j, stateWriterName)
						}
						return os.Chmod(path, modebits)
					}
				}
				return tx
			})
		}
		check(mode+"_denied", a.FinishIdle(c) != nil)
		check(mode+"_retired", a.blocked && n.pair.blocked && n.pair.native.journal.failed && n.pair.native.journal.lease.publicationBlocked && n.pair.native.ledger.publicationBlocked)
		bj, bl := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
		check(mode+"_retry_preserves", a.FinishIdle(ctx) != nil && reflect.DeepEqual(bj, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, l)))
		check(mode+"_release", a.Close() == nil && restartLocksReleased(t, j, l))
		stop()
	}
	// Controlled historical fixture only; new original active/closed receipts are real.
	j, l, b := restartSeed(t, f, "retirement-capacity")
	record, e := decodeExecutionJournal(b)
	if e != nil {
		t.Fatal(e)
	}
	record.Generation = math.MaxUint64 - 2
	b, e = encodeExecutionJournal(record)
	if e != nil || os.WriteFile(filepath.Join(j, executionJournalName), b, 0600) != nil {
		t.Fatal("controlled capacity", e)
	}
	r, n, e = reviewExecutionStartup(ctx, j, l, f.read, sys, activationBoot)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	a, e = r.Activate(ctx)
	if a != nil {
		defer a.Close()
	}
	check("capacity_original_active", e == nil && a != nil && n.snapshot.record.Generation == math.MaxUint64-1)
	check("capacity_finish", a.FinishIdle(ctx) == nil)
	actual, e = n.pair.native.journal.read(nil)
	check("capacity_exact_closed", e == nil && actual.Generation == math.MaxUint64 && actual.Owner.State == "closed")
	check("capacity_retired", n.pair.blocked && n.pair.native.journal.failed && n.pair.native.ledger.publicationBlocked)
	check("capacity_release", a.Close() == nil && restartLocksReleased(t, j, l))
	q, _, e = reviewExecutionStartup(ctx, j, l, f.read, sys, activationBoot)
	if q != nil {
		defer q.Close()
	}
	check("capacity_no_restart", e != nil && q == nil)
	t.Log("CLOSED_STARTUP_RETIREMENT_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(seen, " "))
}

// The exact child pauses only after the actual declared sync/acknowledgement.
func TestClosedStartupRetirementCutChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_RESTART_RETIRE_CUT")
	if mode == "" {
		t.Skip("owned cut child only")
	}
	restartStorageGuard(t)
	j, l := os.Getenv("HORVIUM_RESTART_JOURNAL"), os.Getenv("HORVIUM_RESTART_LEDGER")
	read := func(c context.Context, fd int) (ExecutionBackingObservation, error) {
		var fs unix.Statfs_t
		if c.Err() != nil || unix.Fstatfs(fd, &fs) != nil || unix.ByteSliceToString(fs.Fstypename[:]) != "ufs" {
			return ExecutionBackingObservation{}, ErrDenied
		}
		return ExecutionBackingObservation{Kind: "ufs", UFS: [2]uint32{uint32(fs.Fsid.Val[0]), uint32(fs.Fsid.Val[1])}}, nil
	}
	use := activationSystemFixture(t, filepath.Dir(j), filepath.Dir(l), read)
	sys := func(context.Context) (PoolSystemUse, error) { return use, nil }
	r, n, e := reviewExecutionStartup(context.Background(), j, l, read, sys, activationBoot)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	a, e := r.Activate(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	barrier := func() {
		fmt.Println("CLOSED_STARTUP_RETIRE_CUT_REACHED " + mode)
		_, _ = io.Copy(io.Discard, os.Stdin)
		t.Fatal("cut continued")
	}
	if mode == "uid" {
		if unix.Setgroups([]int{}) != nil || unix.Setgid(65534) != nil || unix.Setuid(65534) != nil || os.Geteuid() != 65534 {
			t.Fatal("drop")
		}
		for _, s := range []*StateDirectory{n.pair.native.journal.lease, n.pair.native.ledger} {
			var st unix.Stat_t
			if unix.Fstat(int(s.directory.Fd()), &st) != nil || st.Uid != 0 {
				t.Fatal("inherited root")
			}
		}
		if e := a.FinishIdle(context.Background()); !errors.Is(e, ErrDenied) {
			t.Fatal("nonroot retirement")
		}
		barrier()
	}
	wrap := func(tx ownershipPublication) ownershipPublication {
		if mode == "file-sync" {
			old := tx.syncFile
			tx.syncFile = func() error {
				if e := old(); e != nil {
					return e
				}
				barrier()
				return ErrDenied
			}
		}
		if mode == "directory-sync" {
			old := tx.syncDirectory
			tx.syncDirectory = func() error {
				if e := old(); e != nil {
					return e
				}
				barrier()
				return ErrDenied
			}
		}
		return tx
	}
	a.finish = func(c context.Context) error { return n.finishIdle(c, wrap) }
	e = a.FinishIdle(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if mode == "acknowledged" {
		barrier()
	}
	t.Fatal("unknown mode")
}
func TestNativeClosedStartupRetirementCuts(t *testing.T) {
	restartStorageGuard(t)
	f := newRestoreFixture(t, "ufs")
	var seen []string
	check := func(n string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("restart cut", n)
		}
		seen = append(seen, n+"=pass")
	}
	for _, mode := range []string{"file-sync", "directory-sync", "acknowledged", "uid"} {
		j, l, b := restartSeed(t, f, "cut-"+mode)
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestClosedStartupRetirementCutChild$", "-test.timeout=45s")
		cmd.Env = append(os.Environ(), "HORVIUM_RESTART_RETIRE_CUT="+mode, "HORVIUM_RESTART_JOURNAL="+j, "HORVIUM_RESTART_LEDGER="+l)
		in, e := cmd.StdinPipe()
		if e != nil {
			t.Fatal(e)
		}
		out, e := cmd.StdoutPipe()
		if e != nil {
			t.Fatal(e)
		}
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if cmd.Start() != nil {
			t.Fatal("start")
		}
		func() {
			defer cancel()
			defer in.Close()
			defer func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			scan := bufio.NewScanner(out)
			reached := false
			for scan.Scan() {
				if scan.Text() == "CLOSED_STARTUP_RETIRE_CUT_REACHED "+mode {
					reached = true
					break
				}
			}
			if !reached {
				t.Log("child stderr", stderr.String())
			}
			check(mode+"_barrier", reached)
			_ = cmd.Process.Kill()
			e = cmd.Wait()
			var ee *exec.ExitError
			check(mode+"_killed_reaped", errors.As(e, &ee) && ee.ProcessState.Sys().(syscall.WaitStatus).Signal() == syscall.SIGKILL)
			data, e := os.ReadFile(filepath.Join(j, executionJournalName))
			record, de := decodeExecutionJournal(data)
			old, _ := decodeExecutionJournal(b)
			expected := "closed"
			gen := old.Generation + 2
			if mode == "file-sync" || mode == "uid" {
				expected = "active"
				gen = old.Generation + 1
			}
			check(mode+"_visible_history", e == nil && de == nil && record.Owner.State == expected && record.Generation == gen && record.Census == old.Census && record.Binding == old.Binding && record.Owner.ID != old.Owner.ID && record.Owner.ID != old.Binding.Installation && record.Owner.Boot == old.Owner.Boot)
			bj, bl := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
			use := activationSystemFixture(t, f.journal, f.ledger, f.read)
			q, nn, er := reviewExecutionStartup(context.Background(), j, l, f.read, func(context.Context) (PoolSystemUse, error) { return use, nil }, activationBoot)
			if mode == "directory-sync" || mode == "acknowledged" {
				check(mode+"_metadata_only", er == nil && q != nil && !nn.pair.native.journal.acknowledged && q.Close() == nil)
			} else {
				check(mode+"_no_startup_takeover", er != nil && q == nil)
			}
			check(mode+"_history_preserved", reflect.DeepEqual(bj, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, l)))
			sj, ej := openStateDirectory(j, false)
			sl, el := openStateDirectory(l, false)
			check(mode+"_leases_released", ej == nil && el == nil && sj.Close() == nil && sl.Close() == nil)
		}()
	}
	t.Log("CLOSED_STARTUP_RETIRE_CUT_RESULT " + strings.Join(seen, " "))
}
