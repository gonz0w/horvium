// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

// Provision only a disposable private domain's empty lock, not installation,
// persistent backing, registered ledger, boot or restoration provenance.
func nativeJournalFixture(t *testing.T, anchor, name string) (*executionJournalStore, string) {
	t.Helper()
	path := filepath.Join(anchor, name)
	if os.Mkdir(path, 0700) != nil || os.WriteFile(filepath.Join(path, stateWriterName), nil, 0600) != nil {
		t.Fatal("owned journal domain unavailable")
	}
	s, e := openExecutionJournalStore(path)
	if e != nil {
		t.Fatal("owned journal lease", e)
	}
	t.Cleanup(func() { _ = s.close() })
	return s, path
}

func journalContentDigest(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestNativeExecutionJournalStore(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	anchor, e := nativeStateAnchor(t, "horvium-journal-store-")
	if e != nil {
		t.Fatal(e)
	}
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("journal store assertion", name)
		}
		results = append(results, name+"=pass")
	}
	s, path := nativeJournalFixture(t, anchor, "journal")
	defer s.close()
	entries, e := os.ReadDir(path)
	check("empty_existing_lock", e == nil && len(entries) == 1 && entries[0].Name() == stateWriterName)
	second, e := openExecutionJournalStore(path)
	check("second_writer_denial", second == nil && errors.Is(e, ErrStateLocked))
	missing := filepath.Join(anchor, "missing-lock")
	if os.Mkdir(missing, 0700) != nil {
		t.Fatal("owned missing domain")
	}
	second, e = openExecutionJournalStore(missing)
	_, absent := os.Lstat(filepath.Join(missing, stateWriterName))
	check("no_lock_initialization", second == nil && e != nil && os.IsNotExist(absent))
	j, e := s.read(nil)
	check("missing_record_denial", e != nil && j == (executionJournal{}))
	copied := *s
	check("copied_close_denial", errors.Is(copied.close(), ErrDenied))
	j, e = copied.read(nil)
	check("copied_read_denial", errors.Is(e, ErrDenied) && j == (executionJournal{}))
	initial := executionJournalFixture()
	check("copied_publish_denial", errors.Is(copied.initialize(initial, nil), ErrDenied))
	// Original journal lease is held first; the ledger is a separate owned domain.
	ledgerPath := filepath.Join(anchor, "ledger")
	if os.Mkdir(ledgerPath, 0700) != nil {
		t.Fatal("owned ledger")
	}
	ledger, e := OpenStateDirectory(ledgerPath)
	if e != nil {
		t.Fatal(e)
	}
	defer ledger.Close()
	_, initial.Census, e = ledger.inspectExecutionCensus(initial.Binding, defaultStateCatalogLimits, nil)
	if e != nil {
		t.Fatal(e)
	}
	bad := initial
	bad.Generation = 2
	check("noninitial_denial", errors.Is(s.initialize(bad, nil), ErrDenied))
	check("initial_publication", s.initialize(initial, nil) == nil)
	wire, _ := encodeExecutionJournal(initial)
	record := filepath.Join(path, executionJournalName)
	data, e := os.ReadFile(record)
	check("canonical_bytes", e == nil && bytes.Equal(data, wire))
	info, e := os.Lstat(record)
	if e != nil {
		t.Fatal(e)
	}
	st := info.Sys().(*syscall.Stat_t)
	check("private_file", info.Mode().IsRegular() && info.Mode().Perm() == 0600 && st.Uid == 0 && st.Nlink == 1)
	lock, e := os.Stat(filepath.Join(path, stateWriterName))
	if e != nil {
		t.Fatal(e)
	}
	j, e = s.read(nil)
	check("independent_read", e == nil && j == initial)
	check("no_reinitialize", errors.Is(s.initialize(initial, nil), ErrDenied))
	// Loss of a journal cannot turn an observation handle into a fresh initializer.
	missingStore, missingPath := nativeJournalFixture(t, anchor, "missing-reopened")
	if missingStore.initialize(initial, nil) != nil || missingStore.close() != nil {
		t.Fatal("owned missing reopen setup")
	}
	missingStore, e = openExecutionJournalStore(missingPath)
	if e != nil || os.Remove(filepath.Join(missingPath, executionJournalName)) != nil {
		t.Fatal("owned missing reopen")
	}
	missingErr := missingStore.initialize(initial, nil)
	_, missingErrStat := os.Stat(filepath.Join(missingPath, executionJournalName))
	check("missing_reopen_no_initialization", errors.Is(missingErr, ErrDenied) && os.IsNotExist(missingErrStat))
	if missingStore.close() != nil {
		t.Fatal("owned missing reopen close")
	}
	missingStore, missingPath = nativeJournalFixture(t, anchor, "missing-original")
	if missingStore.initialize(initial, nil) != nil || os.Remove(filepath.Join(missingPath, executionJournalName)) != nil {
		t.Fatal("owned missing original")
	}
	missingNext := initial
	missingNext.Generation++
	missingNext.Owner.State = "closed"
	j, e = missingStore.read(nil)
	check("missing_original_retirement", errors.Is(e, ErrStateDirectory) && j == (executionJournal{}) && errors.Is(missingStore.publish(missingNext, nil), ErrStateUncertain) && errors.Is(missingStore.initialize(initial, nil), ErrStateUncertain))
	if missingStore.close() != nil {
		t.Fatal("owned missing original close")
	}
	_, e = s.lease.InspectState()
	check("separate_namespace", errors.Is(e, ErrStateRecord))
	owner := ownershipGolden()
	policy, e := DecodeOwnershipRecord(owner)
	if e != nil {
		t.Fatal(e)
	}
	pending := initial
	pending.Generation++
	pending.Metadata = executionJournalMetadata{Pending: true, Before: initial.Census,
		After: expectedExecutionCensus(initial.Binding, []string{"ownership-100.json"}, map[string][]byte{"ownership-100.json": owner}),
		Name:  "ownership-100.json", Successor: journalContentDigest(owner)}
	gap := pending
	gap.Generation++
	check("generation_gap_denial", errors.Is(s.publish(gap, nil), ErrDenied))
	check("pending_publication", s.publish(pending, nil) == nil)
	j, e = s.read(nil)
	check("pending_readback", e == nil && j == pending)
	check("real_ledger_publication", ledger.PublishOwnership(policy, 0) == nil)
	_, observed, e := ledger.inspectExecutionCensus(initial.Binding, defaultStateCatalogLimits, nil)
	check("actual_after_census", e == nil && observed == pending.Metadata.After)
	confirm := pending
	confirm.Generation++
	confirm.Metadata = executionJournalMetadata{}
	confirm.Census = observed
	check("confirmed_publication", s.publish(confirm, nil) == nil)
	old, e := os.Open(record)
	if e != nil {
		t.Fatal(e)
	}
	oldInfo, e := old.Stat()
	if e != nil {
		t.Fatal(e)
	}
	// Persisting native-pending is metadata only: no executor is launched here.
	native := confirm
	native.Generation++
	native.Native = journalNativeFixture(native)
	check("native_pending_publication", s.publish(native, nil) == nil)
	now, e := os.Stat(record)
	check("replacement_inode", e == nil && !os.SameFile(oldInfo, now))
	oldBytes := make([]byte, len(wire)*2)
	n, e := old.Read(oldBytes)
	expectedOld, _ := encodeExecutionJournal(confirm)
	check("old_descriptor_preserved", e == nil && bytes.Equal(oldBytes[:n], expectedOld))
	if old.Close() != nil {
		t.Fatal("owned old descriptor close")
	}
	policy.Revision++
	nextOwner, _ := EncodeOwnershipRecord(policy)
	next := native
	next.Generation++
	next.Metadata = executionJournalMetadata{Pending: true, Before: native.Census,
		After: expectedExecutionCensus(initial.Binding, []string{"ownership-100.json"}, map[string][]byte{"ownership-100.json": nextOwner}),
		Name:  "ownership-100.json", Predecessor: journalContentDigest(owner), Successor: journalContentDigest(nextOwner)}
	check("distinct_metadata_pending", s.publish(next, nil) == nil)
	check("real_ledger_successor", ledger.PublishOwnership(policy, 1) == nil)
	_, observed, e = ledger.inspectExecutionCensus(initial.Binding, defaultStateCatalogLimits, nil)
	if e != nil {
		t.Fatal(e)
	}
	confirmation := next
	confirmation.Generation++
	confirmation.Metadata = executionJournalMetadata{}
	confirmation.Census = observed
	check("metadata_preserves_native", s.publish(confirmation, nil) == nil && confirmation.Native == native.Native)
	j, e = s.read(nil)
	check("both_domains_readback", e == nil && j == confirmation && j.Native.Pending)
	barrier := confirmation
	barrier.Generation++
	barrier.Disarmed = true
	check("restore_barrier_publication", s.publish(barrier, nil) == nil)
	cleared := barrier
	cleared.Generation++
	cleared.Disarmed = false
	check("no_ordinary_rearm", errors.Is(s.publish(cleared, nil), ErrDenied))
	afterLock, e := os.Stat(filepath.Join(path, stateWriterName))
	check("stable_lock", e == nil && os.SameFile(lock, afterLock))
	entries, e = os.ReadDir(path)
	check("no_temporary", e == nil && len(entries) == 2)
	check("close", s.close() == nil && s.close() == nil)
	j, e = s.read(nil)
	check("closed_denial", e != nil && j == (executionJournal{}))
	reopened, e := openExecutionJournalStore(path)
	if e != nil {
		t.Fatal(e)
	}
	j, e = reopened.read(nil)
	check("reopen_observation", e == nil && j == barrier)
	check("reopen_no_authority", errors.Is(reopened.initialize(initial, nil), ErrDenied) && errors.Is(reopened.publish(cleared, nil), ErrDenied))
	if reopened.close() != nil {
		t.Fatal("owned reopen close")
	}
	// A clean active record also cannot reconstruct its original receipt on reopen.
	active, activePath := nativeJournalFixture(t, anchor, "active")
	if active.initialize(initial, nil) != nil || active.close() != nil {
		t.Fatal("owned active fixture")
	}
	active, e = openExecutionJournalStore(activePath)
	if e != nil {
		t.Fatal(e)
	}
	check("active_reopen_no_authority", errors.Is(active.publish(pending, nil), ErrDenied))
	if active.close() != nil {
		t.Fatal("owned active close")
	}
	// Competing calls serialize one actual generation, without a global-launch claim.
	serial, _ := nativeJournalFixture(t, anchor, "serialized")
	if serial.initialize(initial, nil) != nil {
		t.Fatal("owned serialized fixture")
	}
	start := make(chan struct{})
	answers := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() { defer workers.Done(); <-start; answers <- serial.publish(pending, nil) }()
	}
	close(start)
	workers.Wait()
	close(answers)
	wins, denials := 0, 0
	for e := range answers {
		if e == nil {
			wins++
		} else {
			denials++
		}
	}
	check("serialized_generation", wins == 1 && denials == 1)
	if serial.close() != nil {
		t.Fatal("owned serialized close")
	}
	foreign := filepath.Join(anchor, "foreign")
	if os.WriteFile(foreign, wire, 0600) != nil {
		t.Fatal("owned sentinel")
	}
	for _, kind := range []string{"symlink", "hardlink", "fifo", "mode", "owner", "damage", "oversize", "unknown", "orphan"} {
		fixture, folder := nativeJournalFixture(t, anchor, "unsafe-"+kind)
		target := filepath.Join(folder, executionJournalName)
		switch kind {
		case "symlink":
			e = os.Symlink(foreign, target)
		case "hardlink":
			e = os.Link(foreign, target)
		case "fifo":
			e = unix.Mkfifo(target, 0600)
		default:
			content := wire
			if kind == "damage" {
				content = []byte("{}")
			}
			if kind == "oversize" {
				content = bytes.Repeat([]byte("x"), maxStateRecordBytes+1)
			}
			if kind == "unknown" {
				target = filepath.Join(folder, "unknown")
			}
			if kind == "orphan" {
				target = filepath.Join(folder, executionJournalTemporaryPrefix+strings.Repeat("a", 32)+".tmp")
			}
			e = os.WriteFile(target, content, 0600)
			if e == nil && kind == "mode" {
				e = os.Chmod(target, 0644)
			}
			if e == nil && kind == "owner" {
				e = os.Chown(target, 65534, 65534)
			}
		}
		if e != nil {
			t.Fatal("owned unsafe fixture", e)
		}
		before, e := os.Lstat(target)
		if e != nil {
			t.Fatal(e)
		}
		j, readErr := fixture.read(nil)
		publishErr := fixture.initialize(initial, nil)
		after, e := os.Lstat(target)
		check(kind+"_denial", readErr != nil && publishErr != nil && j == (executionJournal{}) && e == nil && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size())
		if fixture.close() != nil {
			t.Fatal("owned unsafe close")
		}
		badStore, e := openExecutionJournalStore(folder)
		if e == nil || badStore != nil {
			t.Fatal("unsafe reopen accepted", kind)
		}
	}
	for _, phase := range []string{"write", "syncFile", "closeFile", "verify", "rename", "syncDirectory", "cleanup", "lost-ack", "after-ack-drift", "late-record", "late-lock"} {
		fixture, folder := nativeJournalFixture(t, anchor, "failure-"+phase)
		if fixture.initialize(initial, nil) != nil {
			t.Fatal("owned failure setup")
		}
		err := fixture.publish(pending, func(tx ownershipPublication) ownershipPublication {
			fail := func() error { return errors.New("owned transaction failure") }
			switch phase {
			case "write":
				tx.write = fail
			case "syncFile":
				tx.syncFile = fail
			case "closeFile":
				tx.closeFile = fail
			case "verify":
				tx.verify = fail
			case "rename":
				tx.rename = fail
			case "syncDirectory":
				tx.syncDirectory = fail
			case "cleanup":
				// Fail orphan cleanup after closing the actual staging descriptor;
				// a retained name must not make the disposable UFS unmount busy.
				closeFile := tx.closeFile
				tx.write = fail
				tx.cleanup = func() error {
					if e := closeFile(); e != nil {
						return e
					}
					return fail()
				}
			case "lost-ack", "after-ack-drift":
				original := tx.syncDirectory
				tx.syncDirectory = func() error {
					if e := original(); e != nil {
						return e
					}
					if phase == "lost-ack" {
						return fail()
					}
					b, _ := encodeExecutionJournal(pending)
					if os.WriteFile(filepath.Join(folder, executionJournalName), append(b, ' '), 0600) != nil {
						t.Fatal("owned acknowledgement drift")
					}
					return nil
				}
			case "late-record", "late-lock":
				original := tx.syncFile
				tx.syncFile = func() error {
					if e := original(); e != nil {
						return e
					}
					if phase == "late-record" {
						if os.WriteFile(filepath.Join(folder, executionJournalName), append(append([]byte(nil), wire...), ' '), 0600) != nil {
							t.Fatal("owned predecessor drift")
						}
					} else {
						lock := filepath.Join(folder, stateWriterName)
						if os.Rename(lock, filepath.Join(anchor, "retained-lock")) != nil || os.WriteFile(lock, nil, 0600) != nil {
							t.Fatal("owned lock drift")
						}
					}
					return nil
				}
			}
			return tx
		})
		check(phase+"_failure", err != nil && errors.Is(fixture.publish(pending, nil), ErrStateUncertain))
		if phase == "syncDirectory" || phase == "lost-ack" || phase == "after-ack-drift" {
			j, e = fixture.read(nil)
			if e != nil || j != pending {
				t.Fatal("visible uncertain metadata missing", phase, e)
			}
			check(phase+"_visible", true)
		}
		if fixture.close() != nil {
			t.Fatal("owned failure close")
		}
	}
	// Detected identical-byte replacement cannot acquire the old live receipt.
	replaced, folder := nativeJournalFixture(t, anchor, "identical-replacement")
	if replaced.initialize(initial, nil) != nil {
		t.Fatal("owned replacement setup")
	}
	if os.Rename(filepath.Join(folder, executionJournalName), filepath.Join(anchor, "old-journal")) != nil || os.WriteFile(filepath.Join(folder, executionJournalName), wire, 0600) != nil {
		t.Fatal("owned replacement")
	}
	check("identical_inode_replacement", errors.Is(replaced.publish(pending, nil), ErrDrift))
	if replaced.close() != nil {
		t.Fatal("owned replaced close")
	}
	// Drift after the read discards the entire observation.
	drift, folder := nativeJournalFixture(t, anchor, "read-drift")
	if drift.initialize(initial, nil) != nil {
		t.Fatal("owned drift setup")
	}
	j, e = drift.read(func() {
		if os.WriteFile(filepath.Join(folder, "late"), nil, 0600) != nil {
			t.Fatal("owned namespace drift")
		}
	})
	check("read_namespace_drift", errors.Is(e, ErrDrift) && j == (executionJournal{}))
	if drift.close() != nil {
		t.Fatal("owned drift close")
	}
	var inert *executionJournalStore
	j, e = inert.read(nil)
	check("inert_denial", e != nil && j == (executionJournal{}) && (&executionJournalStore{}).initialize(initial, nil) != nil && inert.close() == nil)
	data, e = os.ReadFile(foreign)
	check("foreign_preservation", e == nil && bytes.Equal(data, wire))
	if ledger.Close() != nil || os.RemoveAll(anchor) != nil {
		t.Fatal("owned journal teardown")
	}
	_, e = os.Stat(anchor)
	check("fixture_teardown", os.IsNotExist(e))
	t.Log("EXECUTION_JOURNAL_STORE_RESULT " + strings.Join(results, " "))
}

// Reused by the existing actual POSIX/NFSv4 memory-filesystem parent.
func nativeJournalACLModel(t *testing.T, mountpoint string, kind int, check func(string, bool)) {
	t.Helper()
	for _, object := range []string{"directory", "lock", "record", "inheritance"} {
		fixture, folder := nativeJournalFixture(t, mountpoint, "journal-"+object)
		j := executionJournalFixture()
		if fixture.initialize(j, nil) != nil {
			t.Fatal("owned journal ACL setup")
		}
		fd := int(fixture.lease.directory.Fd())
		var record *os.File
		if object == "lock" {
			fd = int(fixture.lease.writer.Fd())
		}
		if object == "record" {
			var e error
			record, e = os.OpenFile(filepath.Join(folder, executionJournalName), os.O_RDWR, 0)
			if e != nil {
				t.Fatal(e)
			}
			fd = int(record.Fd())
		}
		original, e := readStateACL(fd, kind)
		if e != nil {
			t.Fatal(e)
		}
		if object == "inheritance" && kind == 2 {
			setFixtureACL(t, fd, 3, testModeACL(2, 0700))
		} else {
			bad := namedFixtureACL(original, kind, 0)
			if object == "inheritance" {
				bad.Entries[0].Flags = 3
			}
			setFixtureACL(t, fd, kind, bad)
		}
		next := j
		next.Generation++
		next.Owner.State = "closed"
		publishErr := fixture.publish(next, nil)
		got, readErr := fixture.read(nil)
		check("journal_"+object+"_denial", errors.Is(publishErr, ErrStateDirectory) && errors.Is(readErr, ErrStateDirectory) && got == (executionJournal{}))
		if object == "inheritance" && kind == 2 {
			setFixtureACL(t, fd, 3, stateACL{})
		} else {
			setFixtureACL(t, fd, kind, original)
		}
		got, readErr = fixture.read(nil)
		check("journal_"+object+"_restored_observation", readErr == nil && got == j && errors.Is(fixture.publish(next, nil), ErrStateUncertain))
		if record != nil && record.Close() != nil || fixture.close() != nil {
			t.Fatal("owned journal ACL close")
		}
	}
}
