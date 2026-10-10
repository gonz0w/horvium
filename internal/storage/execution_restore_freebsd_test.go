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
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// This exact-binary fixture interrupts a file-level restore after deleting the
// newer pending job. It is not the runtime helper or an operator restore command.
func TestExecutionRestoreCutProcess(t *testing.T) {
	path := os.Getenv("HORVIUM_RESTORE_CUT_DIRECTORY")
	if path == "" {
		t.Skip("owned restore reexec only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("disposable root state VM required")
	}
	job := filepath.Join(path, "job-"+strings.Repeat("e", 64)+".json")
	if os.Remove(job) != nil {
		t.Fatal("owned pending job removal")
	}
	dir, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if dir.Sync() != nil || dir.Close() != nil {
		t.Fatal("owned interrupted ledger sync")
	}
	fmt.Fprintln(os.Stdout, "RESTORE_CUT_REACHED job_removed")
	_, _ = io.Copy(io.Discard, os.Stdin)
	t.Fatal("restore cut unexpectedly resumed")
}

type restoreFixture struct {
	journal, ledger                         string
	read                                    func(context.Context, int) (ExecutionBackingObservation, error)
	snapshot, restoreLedger, restoreJournal func()
}

func restoreNativeCommand(t *testing.T, program string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, e := exec.CommandContext(ctx, program, args...).CombinedOutput()
	if e != nil || len(out) > 4096 {
		t.Fatal("owned restore native command", program, args, e, string(out))
	}
	return strings.TrimSpace(string(out))
}

// The UFS procedure replaces only a backed-up record, never its directory/lock.
// The ZFS procedure rolls back one owned dataset, never recursively its parent.
func newRestoreFixture(t *testing.T, kind string) restoreFixture {
	t.Helper()
	var anchor string
	var e error
	if kind == "ufs" {
		anchor, e = nativeUFSStateAnchor(t, "horvium-restore-", false)
		if e != nil {
			t.Fatal(e)
		}
	} else {
		anchor, e = os.MkdirTemp("/root", "horvium-restore-zfs-")
		if e != nil {
			t.Fatal(e)
		}
	}
	f := restoreFixture{journal: filepath.Join(anchor, "journal"), ledger: filepath.Join(anchor, "ledger")}
	if kind == "zfs" {
		pool := filepath.Base(anchor)
		created := false
		t.Cleanup(func() {
			if created {
				restoreNativeCommand(t, "/sbin/zpool", "destroy", pool)
			}
			if got := restoreNativeCommand(t, "/sbin/zpool", "list", "-H", "-o", "name"); strings.Contains("\n"+got+"\n", "\n"+pool+"\n") {
				t.Fatal("owned restore pool remains")
			}
			if os.RemoveAll(anchor) != nil {
				t.Fatal("owned restore backing removal")
			}
			if _, e := os.Lstat(anchor); !os.IsNotExist(e) {
				t.Fatal("owned restore backing remains")
			}
			t.Log("STATE_BACKING_RESULT " + t.Name() + " filesystem=zfs softupdates=off teardown=pass")
		})
		disk := filepath.Join(anchor, "owned-pool.img")
		file, e := os.OpenFile(disk, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			t.Fatal(e)
		}
		truncateErr := file.Truncate(256 << 20)
		closeErr := file.Close()
		if truncateErr != nil || closeErr != nil {
			t.Fatal("owned ZFS backing")
		}
		restoreNativeCommand(t, "/sbin/zpool", "create", "-o", "cachefile=none", "-O", "mountpoint=none", pool, disk)
		created = true
		for _, role := range []string{"journal", "ledger"} {
			restoreNativeCommand(t, "/sbin/zfs", "create", "-o", "mountpoint="+filepath.Join(anchor, role), pool+"/"+role)
			if os.Chmod(filepath.Join(anchor, role), 0700) != nil {
				t.Fatal("owned restore dataset mode")
			}
		}
		number := func(program string, args ...string) uint64 {
			value, e := strconv.ParseUint(restoreNativeCommand(t, program, args...), 10, 64)
			if e != nil || value == 0 {
				t.Fatal("owned native GUID")
			}
			return value
		}
		f.read = func(ctx context.Context, fd int) (ExecutionBackingObservation, error) {
			if ctx.Err() != nil {
				return ExecutionBackingObservation{}, ctx.Err()
			}
			var fs unix.Statfs_t
			if unix.Fstatfs(fd, &fs) != nil || unix.ByteSliceToString(fs.Fstypename[:]) != "zfs" {
				return ExecutionBackingObservation{}, ErrDrift
			}
			source := unix.ByteSliceToString(fs.Mntfromname[:])
			if source != pool+"/ledger" && source != pool+"/journal" {
				return ExecutionBackingObservation{}, ErrDrift
			}
			return ExecutionBackingObservation{Kind: "zfs", Pool: number("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool), Dataset: number("/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", source)}, nil
		}
		f.snapshot = func() {
			restoreNativeCommand(t, "/sbin/zfs", "snapshot", pool+"/ledger@before", pool+"/journal@before")
		}
		f.restoreLedger = func() { restoreNativeCommand(t, "/sbin/zfs", "rollback", pool+"/ledger@before") }
		f.restoreJournal = func() { restoreNativeCommand(t, "/sbin/zfs", "rollback", pool+"/journal@before") }
	} else {
		for _, path := range []string{f.journal, f.ledger} {
			if os.Mkdir(path, 0700) != nil {
				t.Fatal("owned restore domain")
			}
		}
		// This fixture observes the accepted native mount fs_id projection. Guarded
		// on-disk identity production/system eligibility have separate host gates.
		f.read = func(ctx context.Context, fd int) (ExecutionBackingObservation, error) {
			if ctx.Err() != nil {
				return ExecutionBackingObservation{}, ctx.Err()
			}
			var fs unix.Statfs_t
			if unix.Fstatfs(fd, &fs) != nil || unix.ByteSliceToString(fs.Fstypename[:]) != "ufs" || fs.Flags&unix.MNT_SOFTDEP != 0 {
				return ExecutionBackingObservation{}, ErrDrift
			}
			return ExecutionBackingObservation{Kind: "ufs", UFS: [2]uint32{uint32(fs.Fsid.Val[0]), uint32(fs.Fsid.Val[1])}}, nil
		}
		backup := filepath.Join(anchor, "ledger-export")
		oldJournal := filepath.Join(anchor, "joint-restore-control")
		f.snapshot = func() {
			b, e := os.ReadFile(filepath.Join(f.ledger, "ownership-100.json"))
			if e != nil || os.WriteFile(backup, b, 0600) != nil {
				t.Fatal("owned file-level ledger export")
			}
			b, e = os.ReadFile(filepath.Join(f.journal, executionJournalName))
			if e != nil || os.WriteFile(oldJournal, b, 0600) != nil {
				t.Fatal("owned joint-restore control")
			}
		}
		f.restoreLedger = func() { restoreFixtureFile(t, backup, filepath.Join(f.ledger, "ownership-100.json")) }
		f.restoreJournal = func() { restoreFixtureFile(t, oldJournal, filepath.Join(f.journal, executionJournalName)) }
	}
	for _, path := range []string{f.journal, f.ledger} {
		if os.WriteFile(filepath.Join(path, stateWriterName), nil, 0600) != nil {
			t.Fatal("owned pre-existing restore lock")
		}
	}
	return f
}

func restoreFixtureFile(t *testing.T, source, target string) {
	t.Helper()
	data, e := os.ReadFile(source)
	if e != nil {
		t.Fatal(e)
	}
	stage, e := os.CreateTemp(filepath.Dir(target), ".owned-restore-")
	if e != nil {
		t.Fatal(e)
	}
	defer stage.Close()
	if _, e = stage.Write(data); e != nil {
		t.Fatal(e)
	}
	if stage.Sync() != nil || stage.Close() != nil || os.Rename(stage.Name(), target) != nil {
		t.Fatal("owned exact file-level restore")
	}
	dir, e := os.Open(filepath.Dir(target))
	if e != nil {
		t.Fatal(e)
	}
	if dir.Sync() != nil || dir.Close() != nil {
		t.Fatal("owned restore directory sync")
	}
}

type restoreObjectSnapshot struct {
	stat syscall.Stat_t
	data string
}

func restoreJournalSnapshot(t *testing.T, path string) map[string]restoreObjectSnapshot {
	t.Helper()
	entries, e := os.ReadDir(path)
	if e != nil {
		t.Fatal(e)
	}
	names := []string{"."}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	result := map[string]restoreObjectSnapshot{}
	for _, name := range names {
		info, e := os.Lstat(filepath.Join(path, name))
		if e != nil {
			t.Fatal(e)
		}
		data := ""
		if info.Mode().IsRegular() {
			b, e := os.ReadFile(filepath.Join(path, name))
			if e != nil {
				t.Fatal(e)
			}
			data = string(b)
		}
		result[name] = restoreObjectSnapshot{publicationCutSnapshotStat(*info.Sys().(*syscall.Stat_t)), data}
	}
	return result
}

func TestNativeIndependentRestoreUFS(t *testing.T) { nativeIndependentRestore(t, "ufs") }
func TestNativeIndependentRestoreZFS(t *testing.T) { nativeIndependentRestore(t, "zfs") }

func nativeIndependentRestore(t *testing.T, kind string) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable root restore VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("actual root required")
	}
	f := newRestoreFixture(t, kind)
	ctx := context.Background()
	installation := strings.Repeat("1", 32)
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("independent restore assertion", name)
		}
		markers = append(markers, name+"=pass")
	}
	p, e := OpenExecutionDomainLeases(ctx, f.journal, f.ledger)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = p.Close() })
	first, e := p.Observe(ctx, installation, f.read)
	check("original_binding", e == nil && first.origin == p && !first.journalPresent)
	var jst, lst unix.Stat_t
	var jfs, lfs unix.Statfs_t
	check("separate_restore_domains", unix.Fstat(int(p.native.journal.lease.directory.Fd()), &jst) == nil && unix.Fstat(int(p.native.ledger.directory.Fd()), &lst) == nil && unix.Fstatfs(int(p.native.journal.lease.directory.Fd()), &jfs) == nil && unix.Fstatfs(int(p.native.ledger.directory.Fd()), &lfs) == nil && ((kind == "ufs" && jfs.Fsid == lfs.Fsid && jst.Ino != lst.Ino) || (kind == "zfs" && jfs.Fsid != lfs.Fsid && first.binding.JournalBacking != first.binding.LedgerBacking)))
	check("initial_pair_closed", p.Close() == nil)
	js, e := openExecutionJournalStore(f.journal)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = js.close() })
	ls, e := OpenStateDirectory(f.ledger)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = ls.Close() })
	policy, e := DecodeOwnershipRecord(ownershipGolden())
	if e != nil {
		t.Fatal(e)
	}
	check("baseline_ledger_ack", ls.PublishOwnership(policy, 0) == nil)
	oldOwner, e := os.ReadFile(filepath.Join(f.ledger, "ownership-100.json"))
	if e != nil {
		t.Fatal(e)
	}
	j := executionJournalFixture()
	j.Binding = first.binding
	_, j.Census, e = ls.inspectExecutionCensus(j.Binding, defaultStateCatalogLimits, nil)
	if e != nil {
		t.Fatal(e)
	}
	check("initial_journal_ack", js.initialize(j, nil) == nil)
	old := j
	f.snapshot()
	check("export_before_pending_job", !old.Native.Pending && !old.Disarmed)
	records := map[string][]byte{"ownership-100.json": oldOwner}
	publish := func(name string, data []byte, effect func() error, prefix string) {
		t.Helper()
		pending := j
		pending.Generation++
		predecessor := ""
		if before, ok := records[name]; ok {
			predecessor = journalContentDigest(before)
		}
		records[name] = data
		names := []string{"ownership-100.json"}
		if len(records) == 2 {
			names = append(names, "job-"+strings.Repeat("e", 64)+".json")
		}
		sort.Strings(names)
		pending.Metadata = executionJournalMetadata{Pending: true, Before: j.Census, After: expectedExecutionCensus(j.Binding, names, records), Name: name, Predecessor: predecessor, Successor: journalContentDigest(data)}
		check(prefix+"_pending_ack", js.publish(pending, nil) == nil)
		check(prefix+"_ledger_ack", effect() == nil)
		_, census, e := ls.inspectExecutionCensus(j.Binding, defaultStateCatalogLimits, nil)
		check(prefix+"_actual_after", e == nil && census == pending.Metadata.After)
		j = pending
		j.Generation++
		j.Metadata = executionJournalMetadata{}
		j.Census = census
		check(prefix+"_confirmation_ack", js.publish(j, nil) == nil)
	}
	policy.Revision++
	nextOwner, e := EncodeOwnershipRecord(policy)
	if e != nil {
		t.Fatal(e)
	}
	publish("ownership-100.json", nextOwner, func() error { return ls.PublishOwnership(policy, 1) }, "owner")
	job := nativePropertyJobFixture(t)
	pending := j
	pending.Generation++
	pending.Native = journalNativeFixture(j)
	pending.Native.Job = job.job.ID
	pending.Native.Submission = job.job.Intent.SubmissionID
	pending.Native.PoolGUID = job.job.Intent.PoolGUID
	pending.Native.FilesystemGUID = job.intent.review.Intent.FilesystemGUID
	intentBytes, e := EncodePropertyIntent(job.intent)
	if e != nil {
		t.Fatal(e)
	}
	pending.Native.Intent = journalContentDigest(intentBytes)
	check("native_pending_ack", js.publish(pending, nil) == nil)
	j = pending
	data, e := EncodePropertyJob(job)
	if e != nil {
		t.Fatal(e)
	}
	jobName := "job-" + job.job.ID + ".json"
	publish(jobName, data, func() error { return ls.PublishPropertyJob(job, 0) }, "job")
	current := j
	check("current_differs_from_export", current.Census != old.Census && current.Native.Pending && current.Native.Job == job.job.ID)
	// Fixtures contain no actual executor/admission. Observe real file/directory
	// sync ordering while the ledger's complete exact bytes remain unchanged.
	barrier := j
	barrier.Generation++
	barrier.Disarmed = true
	fileSynced, dirSynced := false, false
	ledgerSame := func() bool {
		b, e := os.ReadFile(filepath.Join(f.ledger, "ownership-100.json"))
		q, qe := os.ReadFile(filepath.Join(f.ledger, jobName))
		return e == nil && qe == nil && bytes.Equal(b, nextOwner) && bytes.Equal(q, data)
	}
	e = js.publish(barrier, func(tx ownershipPublication) ownershipPublication {
		sf, sd := tx.syncFile, tx.syncDirectory
		tx.syncFile = func() error {
			if !ledgerSame() {
				return ErrDrift
			}
			if e := sf(); e != nil {
				return e
			}
			fileSynced = true
			return nil
		}
		tx.syncDirectory = func() error {
			if !fileSynced || !ledgerSame() {
				return ErrDrift
			}
			if e := sd(); e != nil {
				return e
			}
			dirSynced = true
			return nil
		}
		return tx
	})
	check("barrier_file_sync_before_restore", fileSynced)
	check("barrier_directory_sync_before_restore", dirSynced)
	check("barrier_ack_before_restore", e == nil && js.ack == barrier && ledgerSame())
	rearm := barrier
	rearm.Generation++
	rearm.Disarmed = false
	check("ordinary_rearm_denied", errors.Is(js.publish(rearm, nil), ErrDenied))
	check("original_leases_closed", ls.Close() == nil && js.close() == nil)
	journalBefore := restoreJournalSnapshot(t, f.journal)
	observe := func() (ExecutionDomainsObservation, *ExecutionDomainLeases) {
		t.Helper()
		pair, e := OpenExecutionDomainLeases(ctx, f.journal, f.ledger)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { _ = pair.Close() })
		got, e := pair.Observe(ctx, installation, f.read)
		if e != nil {
			t.Fatal("restored observation", e)
		}
		return got, pair
	}
	// Kill and actually wait this binary at the interrupted file-level boundary.
	executable, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	cutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	child := exec.CommandContext(cutCtx, executable, "-test.run=^TestExecutionRestoreCutProcess$", "-test.v")
	child.Env = append(os.Environ(), "HORVIUM_RESTORE_CUT_DIRECTORY="+f.ledger)
	stdin, e := child.StdinPipe()
	if e != nil {
		t.Fatal(e)
	}
	stdout, e := child.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	var diagnostic bytes.Buffer
	child.Stderr = &diagnostic
	if child.Start() != nil {
		t.Fatal("owned restore child start")
	}
	waited := false
	defer func() {
		_ = stdin.Close()
		if !waited {
			_ = child.Process.Kill()
			_ = child.Wait()
		}
	}()
	scanner := bufio.NewScanner(stdout)
	reached := false
	for scanner.Scan() {
		if scanner.Text() == "RESTORE_CUT_REACHED job_removed" {
			reached = true
			break
		}
	}
	check("interruption_child_ready", reached)
	killErr := child.Process.Kill()
	waitErr := child.Wait()
	waited = true
	_ = stdin.Close()
	exit, exitOK := waitErr.(*exec.ExitError)
	killed := exitOK && exit.ProcessState.Sys().(syscall.WaitStatus).Signaled() && exit.ProcessState.Sys().(syscall.WaitStatus).Signal() == syscall.SIGKILL
	check("interruption_child_killed_reaped", killErr == nil && killed && child.ProcessState != nil)
	got, pair := observe()
	check("interrupted_job_absent", lenMustRestoreCatalog(t, pair.native.ledger) == 0)
	check("interrupted_census_mismatch", got.census != barrier.Census && got.census != old.Census)
	check("interrupted_pending_retained", got.journal == barrier && got.journal.Native == current.Native)
	check("interrupted_journal_preserved", reflect.DeepEqual(journalBefore, restoreJournalSnapshot(t, f.journal)))
	check("interrupted_binding_preserved", got.binding == first.binding)
	check("interrupted_no_reconstructed_writer", !pair.native.journal.acknowledged && !pair.native.journal.initializable && errors.Is(pair.native.journal.publish(rearm, nil), ErrDenied))
	check("interrupted_pair_closed", pair.Close() == nil)
	f.restoreLedger()
	got, pair = observe()
	check("ledger_only_old_census", got.census == old.Census && got.census != barrier.Census)
	restored, e := os.ReadFile(filepath.Join(f.ledger, "ownership-100.json"))
	check("ledger_only_exact_old_bytes", e == nil && bytes.Equal(restored, oldOwner))
	check("ledger_only_job_absent", lenMustRestoreCatalog(t, pair.native.ledger) == 0)
	check("ledger_only_journal_preserved", reflect.DeepEqual(journalBefore, restoreJournalSnapshot(t, f.journal)))
	check("ledger_only_pending_retained", got.journal == barrier && got.journal.Native == current.Native && got.journal.Disarmed)
	check("ledger_only_binding_preserved", got.binding == first.binding)
	check("ledger_only_no_reconstructed_writer", !pair.native.journal.acknowledged && !pair.native.journal.initializable && errors.Is(pair.native.journal.publish(rearm, nil), ErrDenied))
	repeated, e := pair.Observe(ctx, installation, f.read)
	check("ledger_only_repeat_observation", e == nil && repeated == got)
	check("restored_pair_closed", pair.Close() == nil)
	// Deliberately restore the old journal too. Equality can conceal lost pending
	// state; these local bytes are not an anti-rollback proof or restart credential.
	f.restoreJournal()
	got, pair = observe()
	check("joint_restore_equal_census", got.census == old.Census && got.journal.Census == old.Census)
	check("joint_restore_lost_pending_fact", got.journal == old && !got.journal.Native.Pending && !got.journal.Disarmed)
	next := old
	next.Generation++
	next.Owner.State = "closed"
	check("joint_restore_no_reconstructed_writer", !pair.native.journal.acknowledged && !pair.native.journal.initializable && errors.Is(pair.native.journal.initialize(old, nil), ErrDenied) && errors.Is(pair.native.journal.publish(next, nil), ErrDenied))
	check("joint_restore_pair_closed", pair.Close() == nil)
	// Missing half cannot commission or repair a domain during observation.
	if os.Remove(filepath.Join(f.journal, executionJournalName)) != nil {
		t.Fatal("owned journal removal")
	}
	got, pair = observe()
	check("missing_journal_observation_only", !got.journalPresent && !pair.native.journal.initializable && errors.Is(pair.native.journal.initialize(old, nil), ErrDenied))
	check("missing_journal_pair_closed", pair.Close() == nil)
	missing, e := OpenExecutionDomainLeases(ctx, f.journal, filepath.Join(filepath.Dir(f.ledger), "absent-ledger"))
	check("missing_ledger_denial", e != nil && missing == nil)
	// No repair, replay, new job, cleared hold or reconstructed authority occurs.
	check("missing_ledger_preserves_journal", len(restoreJournalSnapshot(t, f.journal)) == 2)
	t.Log("EXECUTION_RESTORE_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
}

func lenMustRestoreCatalog(t *testing.T, s *StateDirectory) int {
	t.Helper()
	c, e := s.InspectState()
	if e != nil {
		t.Fatal(e)
	}
	return len(c.Jobs)
}
