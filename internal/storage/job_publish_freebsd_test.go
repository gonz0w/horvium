// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func TestNativeJobPublication(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("owned root required")
	}
	anchor, err := nativeStateAnchor(t, "horvium-job-publish-")
	if err != nil {
		t.Fatal(err)
	}
	var leases []*StateDirectory
	defer func() {
		for _, s := range leases {
			_ = s.Close()
		}
		_ = os.RemoveAll(anchor)
	}()
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("job publication assertion", name)
		}
		results = append(results, name+"=pass")
	}
	newLease := func(name string) (*StateDirectory, string) {
		t.Helper()
		p := filepath.Join(anchor, name)
		if os.Mkdir(p, 0700) != nil {
			t.Fatal("owned directory")
		}
		s, e := OpenStateDirectory(p)
		if e != nil {
			t.Fatal(e)
		}
		leases = append(leases, s)
		return s, p
	}
	write := func(p string, data []byte) {
		t.Helper()
		if os.WriteFile(p, data, 0600) != nil {
			t.Fatal("owned write")
		}
	}
	noTemps := func(p string) bool {
		entries, e := os.ReadDir(p)
		if e != nil {
			return false
		}
		for _, v := range entries {
			if strings.HasPrefix(v.Name(), ".job-") || strings.HasPrefix(v.Name(), ".ownership-") {
				return false
			}
		}
		return true
	}
	j := catalogJob(1, false)
	data, e := EncodeJobRecord(j)
	if e != nil {
		t.Fatal(e)
	}
	next, e := TransitionJob(j, JobApplying, 1001, "launch-intent", "")
	if e != nil {
		t.Fatal(e)
	}
	nextBytes, e := EncodeJobRecord(next)
	if e != nil {
		t.Fatal(e)
	}
	p, e := DecodeOwnershipRecord(ownershipGolden())
	if e != nil {
		t.Fatal(e)
	}
	s, path := newLease("normal")
	target := filepath.Join(path, "job-"+j.ID+".json")
	check("create", s.PublishJob(j, 0) == nil)
	b, e := os.ReadFile(target)
	check("canonical_bytes", e == nil && bytes.Equal(b, data))
	old, e := os.Stat(target)
	if e != nil {
		t.Fatal(e)
	}
	st := old.Sys().(*syscall.Stat_t)
	check("private_file", st.Uid == 0 && st.Mode&07777 == 0600 && st.Nlink == 1 && old.Mode().IsRegular())
	lockBefore, e := os.Stat(filepath.Join(path, stateWriterName))
	if e != nil {
		t.Fatal(e)
	}
	check("duplicate_denial", errors.Is(s.PublishJob(j, 0), ErrDrift))
	check("revision_denial", errors.Is(s.PublishJob(next, 0), ErrDrift) && errors.Is(s.PublishJob(next, 2), ErrDrift) && errors.Is(s.PublishJob(next, ^uint64(0)), ErrDrift))
	bad := j
	bad.ID = "../writer.lock"
	check("invalid_denial", errors.Is(s.PublishJob(bad, 0), ErrStateRecord))
	held, e := os.Open(target)
	if e != nil {
		t.Fatal(e)
	}
	check("replace", s.PublishJob(next, 1) == nil)
	oldData, e := io.ReadAll(held)
	ce := held.Close()
	check("old_descriptor_preserved", e == nil && ce == nil && bytes.Equal(oldData, data))
	current, e := os.Stat(target)
	b, be := os.ReadFile(target)
	check("new_inode", e == nil && be == nil && !os.SameFile(old, current) && bytes.Equal(b, nextBytes))
	got, e := s.ReadJob(j.ID)
	check("readback", e == nil && got.History[len(got.History)-1].State == JobApplying && got.Intent == j.Intent)
	lockAfter, e := os.Stat(filepath.Join(path, stateWriterName))
	check("stable_lock", e == nil && os.SameFile(lockBefore, lockAfter))
	check("stale_denial", errors.Is(s.PublishJob(next, 1), ErrDrift))
	rewritten, e := TransitionJob(next, JobObserving, 1002, "native-observation", strings.Repeat("d", 64))
	if e != nil {
		t.Fatal(e)
	}
	rewriteIntent := rewritten
	rewriteIntent.Intent.Change.Value = "off"
	check("intent_rewrite_denial", errors.Is(s.PublishJob(rewriteIntent, 2), ErrDrift))
	rewriteHistory := rewritten
	rewriteHistory.History = append([]JobTransition(nil), rewritten.History...)
	rewriteHistory.History[0].At = 999
	check("history_rewrite_denial", errors.Is(s.PublishJob(rewriteHistory, 2), ErrDrift))
	check("observing_publication", s.PublishJob(rewritten, 2) == nil)

	finished, e := TransitionJob(rewritten, JobSucceeded, 1003, "verified", strings.Repeat("d", 64))
	if e != nil {
		t.Fatal(e)
	}
	check("terminal_publication", s.PublishJob(finished, 3) == nil)
	check("terminal_replay_denial", errors.Is(s.PublishJob(finished, 4), ErrDrift))
	check("mixed_ownership_create", s.PublishOwnership(p, 0) == nil)
	pnext := p
	pnext.Revision++
	check("mixed_ownership_replace", s.PublishOwnership(pnext, 1) == nil)
	catalog, e := s.InspectState()
	ownerBytes, _ := EncodeOwnershipRecord(pnext)
	finishedBytes, _ := EncodeJobRecord(finished)
	check("mixed_accounting", e == nil && len(catalog.Ownership) == 1 && len(catalog.Jobs) == 1 && catalog.Nonterminal == 0 && catalog.Terminal == 1 && catalog.Bytes == uint64(len(ownerBytes)+len(finishedBytes)))
	check("no_temporary", noTemps(path))
	// Fresh reduced bounds exercise each budget without evicting existing records.
	bounded, folder := newLease("bounded")
	invoked := false
	wrap := func(tx ownershipPublication) ownershipPublication { invoked = true; return tx }
	limits := stateCatalogLimits{1, 1, 1, uint64(len(data))}
	check("create_byte_denial", errors.Is(bounded.publishJobBounded(j, 0, stateCatalogLimits{1, 1, 1, limits.bytes - 1}, wrap), ErrLimit) && !invoked && noTemps(folder))
	check("exact_create", bounded.publishJobBounded(j, 0, limits, nil) == nil)
	record := filepath.Join(folder, "job-"+j.ID+".json")
	before, _ := os.Stat(record)
	reserve := uint64(len(data) + len(nextBytes))
	limits.bytes = reserve - 1
	check("replacement_reservation", errors.Is(bounded.publishJobBounded(next, 1, limits, wrap), ErrLimit) && !invoked)
	b, e = os.ReadFile(record)
	after, se := os.Stat(record)
	check("denial_preservation", e == nil && se == nil && bytes.Equal(b, data) && os.SameFile(before, after) && noTemps(folder))
	limits.bytes = reserve
	check("exact_replace", bounded.publishJobBounded(next, 1, limits, nil) == nil)
	second := catalogJob(2, false)
	limits.bytes = 64 << 20
	check("nonterminal_limit", errors.Is(bounded.publishJobBounded(second, 0, limits, wrap), ErrLimit) && !invoked)
	collision := second
	collision.Intent.SubmissionID = j.Intent.SubmissionID
	check("submission_collision", errors.Is(bounded.PublishJob(collision, 0), ErrDrift))
	terminal := catalogJob(2, true)
	terminalData, _ := EncodeJobRecord(terminal)
	write(filepath.Join(folder, "job-"+terminal.ID+".json"), terminalData)
	// Applying->observing->succeeded needs a free terminal slot; none exists here.
	observing, _ := TransitionJob(next, JobObserving, 1002, "native-observation", strings.Repeat("d", 64))
	if bounded.publishJobBounded(observing, 2, limits, nil) != nil {
		t.Fatal("owned observing")
	}
	success, _ := TransitionJob(observing, JobSucceeded, 1003, "verified", strings.Repeat("d", 64))
	check("terminal_limit", errors.Is(bounded.publishJobBounded(success, 3, limits, wrap), ErrLimit) && !invoked)
	check("invalid_limits", errors.Is(bounded.publishJobBounded(second, 0, stateCatalogLimits{}, wrap), ErrLimit) && !invoked)
	mixed, _ := newLease("mixed-capacity")
	if mixed.PublishJob(j, 0) != nil {
		t.Fatal("mixed job")
	}
	ownerSize := uint64(len(ownershipGolden()))
	check("owner_reserves_job_bytes", errors.Is(mixed.publishOwnershipBounded(p, 0, ownershipPublicationLimits{1, uint64(len(data)) + ownerSize - 1}, wrap), ErrLimit) && !invoked)
	check("owner_exact_mixed_bytes", mixed.publishOwnershipBounded(p, 0, ownershipPublicationLimits{1, uint64(len(data)) + ownerSize}, nil) == nil)
	total := uint64(len(data)) + ownerSize
	jobLimits := defaultStateCatalogLimits
	jobLimits.bytes = total + uint64(len(nextBytes)) - 1
	check("job_reserves_owner_bytes", errors.Is(mixed.publishJobBounded(next, 1, jobLimits, wrap), ErrLimit) && !invoked)
	jobLimits.bytes++
	check("job_exact_mixed_bytes", mixed.publishJobBounded(next, 1, jobLimits, nil) == nil)
	moved := filepath.Join(anchor, "moved")
	if os.Rename(path, moved) != nil {
		t.Fatal("owned rename")
	}
	path = moved
	another := catalogJob(5, false)
	check("directory_anchor", s.PublishJob(another, 0) == nil)
	foreign := filepath.Join(anchor, "foreign")
	write(foreign, data)
	for _, kind := range []string{"mode", "owner", "hardlink", "symlink", "fifo", "directory", "damage", "newer", "wrong_id", "wrong_kind"} {
		fixture, dir := newLease("unsafe-" + kind)
		name := filepath.Join(dir, "job-"+j.ID+".json")
		control := data
		switch kind {
		case "symlink":
			e = os.Symlink(foreign, name)
		case "hardlink":
			e = os.Link(foreign, name)
		case "fifo":
			e = unix.Mkfifo(name, 0600)
		case "directory":
			e = os.Mkdir(name, 0700)
		default:
			switch kind {
			case "damage":
				control = []byte("damaged")
			case "newer":
				control = bytes.Replace(data, []byte(`"version":1`), []byte(`"version":2`), 1)
			case "wrong_id":
				control = terminalData
			case "wrong_kind":
				control = ownershipGolden()
			}
			e = os.WriteFile(name, control, 0600)
			if e == nil && kind == "mode" {
				e = os.Chmod(name, 0644)
			}
			if e == nil && kind == "owner" {
				e = os.Chown(name, 65534, 65534)
			}
		}
		if e != nil {
			t.Fatal("owned control", kind, e)
		}
		before, e := os.Lstat(name)
		if e != nil {
			t.Fatal(e)
		}
		invoked = false
		err = fixture.publishJobBounded(next, 1, defaultStateCatalogLimits, wrap)
		after, e := os.Lstat(name)
		check("unsafe_"+kind+"_denial", err != nil && !invoked && e == nil && os.SameFile(before, after) && before.Mode() == after.Mode() && noTemps(dir))
		if before.Mode().IsRegular() {
			b, e = os.ReadFile(name)
			if e != nil || !bytes.Equal(b, control) {
				t.Fatal("unsafe bytes changed")
			}
		}
	}
	for _, kind := range []string{"unknown", "ownership", "job-orphan", "ownership-orphan"} {
		fixture, dir := newLease("preflight-" + kind)
		control := filepath.Join(dir, "unknown")
		value := []byte("owned unresolved intent")
		switch kind {
		case "ownership":
			control = filepath.Join(dir, "ownership-100.json")
		case "job-orphan":
			control = filepath.Join(dir, ".job-"+strings.Repeat("a", 32)+".tmp")
		case "ownership-orphan":
			control = filepath.Join(dir, ".ownership-"+strings.Repeat("a", 32)+".tmp")
		}
		write(control, value)
		invoked = false
		err = fixture.publishJobBounded(j, 0, defaultStateCatalogLimits, wrap)
		want := ErrStateRecord
		if strings.HasSuffix(kind, "orphan") {
			want = ErrStateUncertain
		}
		b, e = os.ReadFile(control)
		check("preflight_"+strings.ReplaceAll(kind, "-", "_")+"_denial", errors.Is(err, want) && !invoked && e == nil && bytes.Equal(b, value))
		if strings.HasSuffix(kind, "orphan") {
			if os.Remove(control) != nil {
				t.Fatal("owned orphan removal")
			}
			if !errors.Is(fixture.PublishJob(j, 0), ErrStateUncertain) || !errors.Is(fixture.PublishOwnership(p, 0), ErrStateUncertain) {
				t.Fatal("orphan latch bypass")
			}
		}
	}
	check("orphan_latch", true)
	for _, phase := range []string{"write", "file_sync", "close", "verify", "rename"} {
		fixture, dir := newLease("phase-" + phase)
		if fixture.PublishJob(j, 0) != nil {
			t.Fatal("phase baseline")
		}
		err = fixture.publishJobBounded(next, 1, defaultStateCatalogLimits, func(tx ownershipPublication) ownershipPublication {
			fail := func() error { return errors.New("owned phase failure") }
			switch phase {
			case "write":
				tx.write = fail
			case "file_sync":
				tx.syncFile = fail
			case "close":
				tx.closeFile = fail
			case "verify":
				tx.verify = fail
			case "rename":
				tx.rename = fail
			}
			return tx
		})
		b, e = os.ReadFile(filepath.Join(dir, "job-"+j.ID+".json"))
		check(phase+"_failure", errors.Is(err, ErrStateDirectory) && e == nil && bytes.Equal(b, data) && noTemps(dir))
		if fixture.PublishJob(next, 1) != nil {
			t.Fatal("known cleaned failure blocked safe fixture successor")
		}
	}
	// A ctime-only change after the final snapshot still refuses publication.
	ctimeLease, ctimeDir := newLease("ctime-drift")
	if ctimeLease.PublishJob(j, 0) != nil {
		t.Fatal("ctime baseline")
	}
	ctimeTarget := filepath.Join(ctimeDir, "job-"+j.ID+".json")
	ctimeBefore, e := os.Stat(ctimeTarget)
	if e != nil {
		t.Fatal(e)
	}
	err = ctimeLease.publishJobBounded(next, 1, defaultStateCatalogLimits, func(tx ownershipPublication) ownershipPublication {
		verify := tx.verify
		tx.verify = func() error {
			var before, after unix.Stat_t
			if unix.Fstatat(int(ctimeLease.directory.Fd()), "job-"+j.ID+".json", &before, unix.AT_SYMLINK_NOFOLLOW) != nil {
				t.Fatal("owned ctime observation")
			}
			if unix.UtimesNanoAt(int(ctimeLease.directory.Fd()), "job-"+j.ID+".json", []unix.Timespec{before.Atim, before.Mtim}, unix.AT_SYMLINK_NOFOLLOW) != nil || unix.Fstatat(int(ctimeLease.directory.Fd()), "job-"+j.ID+".json", &after, unix.AT_SYMLINK_NOFOLLOW) != nil || before.Ctim == after.Ctim || before.Mtim != after.Mtim {
				t.Fatal("owned ctime-only change unavailable")
			}
			return verify()
		}
		return tx
	})
	check("destination_ctime_denial", errors.Is(err, ErrDrift))
	ctimeAfter, se := os.Stat(ctimeTarget)
	b, e = os.ReadFile(ctimeTarget)
	check("ctime_denial_preservation", e == nil && se == nil && bytes.Equal(b, data) && os.SameFile(ctimeBefore, ctimeAfter) && ctimeBefore.ModTime() == ctimeAfter.ModTime() && noTemps(ctimeDir))
	uncertain, dir := newLease("sync-uncertain")
	if uncertain.PublishJob(j, 0) != nil {
		t.Fatal("uncertain baseline")
	}
	err = uncertain.publishJobBounded(next, 1, defaultStateCatalogLimits, func(tx ownershipPublication) ownershipPublication {
		tx.syncDirectory = func() error { return errors.New("owned sync failure") }
		return tx
	})
	check("directory_sync_uncertain", errors.Is(err, ErrStateUncertain))
	got, e = uncertain.ReadJob(j.ID)
	check("uncertain_readable", e == nil && got.History[len(got.History)-1].Revision == 2)
	check("uncertain_job_blocked", errors.Is(uncertain.PublishJob(rewritten, 2), ErrStateUncertain))
	check("uncertain_owner_blocked", errors.Is(uncertain.PublishOwnership(p, 0), ErrStateUncertain))
	cleanup, _ := newLease("cleanup-uncertain")
	err = cleanup.publishJobBounded(j, 0, defaultStateCatalogLimits, func(tx ownershipPublication) ownershipPublication {
		tx.write = func() error { return errors.New("owned write failure") }
		real := tx.cleanup
		tx.cleanup = func() error {
			if e := real(); e != nil {
				return e
			}
			return errors.New("owned cleanup acknowledgement failure")
		}
		return tx
	})
	check("cleanup_uncertain", errors.Is(err, ErrStateUncertain) && errors.Is(cleanup.PublishJob(j, 0), ErrStateUncertain) && errors.Is(cleanup.PublishOwnership(p, 0), ErrStateUncertain))
	// Add actual records after file sync to prove the final census cannot be skipped.
	for _, kind := range []string{"unknown", "orphan", "bytes", "count", "submission"} {
		fixture, dir := newLease("late-" + kind)
		other := catalogJob(2, false)
		otherBytes, _ := EncodeJobRecord(other)
		control := filepath.Join(dir, "job-"+other.ID+".json")
		value := otherBytes
		lim := defaultStateCatalogLimits
		want := ErrLimit
		switch kind {
		case "unknown":
			control = filepath.Join(dir, "unknown")
			want = ErrStateRecord
		case "orphan":
			control = filepath.Join(dir, ".job-"+strings.Repeat("b", 32)+".tmp")
			want = ErrStateUncertain
		case "bytes":
			lim.bytes = uint64(len(data))
		case "count":
			lim.nonterminal = 1
		case "submission":
			other.Intent.SubmissionID = j.Intent.SubmissionID
			value, _ = EncodeJobRecord(other)
			want = ErrDrift
		}
		err = fixture.publishJobBounded(j, 0, lim, func(tx ownershipPublication) ownershipPublication {
			verify := tx.verify
			tx.verify = func() error { write(control, value); return verify() }
			return tx
		})
		b, e = os.ReadFile(control)
		_, missing := os.Stat(filepath.Join(dir, "job-"+j.ID+".json"))
		entries, se := os.ReadDir(dir)
		check("late_"+kind+"_denial", errors.Is(err, want) && e == nil && bytes.Equal(b, value) && os.IsNotExist(missing) && se == nil && len(entries) == 2)
	}
	race, racePath := newLease("race")
	start := make(chan struct{})
	out := make(chan error, 2)
	for _, job := range []Job{j, second} {
		go func() { <-start; out <- race.publishJobBounded(job, 0, stateCatalogLimits{1, 1, 1, 64 << 20}, nil) }()
	}
	close(start)
	a, bErr := <-out, <-out
	check("serialized_last_slot", a == nil && errors.Is(bErr, ErrLimit) || bErr == nil && errors.Is(a, ErrLimit))
	catalog, e = race.InspectState()
	check("serialized_catalog", e == nil && len(catalog.Jobs) == 1 && noTemps(racePath))
	duplicate, _ := newLease("duplicate-race")
	start = make(chan struct{})
	for range 2 {
		go func() { <-start; out <- duplicate.PublishJob(j, 0) }()
	}
	close(start)
	a, bErr = <-out, <-out
	check("serialized_duplicate", a == nil && errors.Is(bErr, ErrDrift) || bErr == nil && errors.Is(a, ErrDrift))
	drift, dir := newLease("directory-drift")
	if os.Chmod(dir, 0755) != nil {
		t.Fatal("owned chmod")
	}
	check("directory_denial", errors.Is(drift.PublishJob(j, 0), ErrStateDirectory))
	drift, dir = newLease("lock-drift")
	lock := filepath.Join(dir, stateWriterName)
	if os.Rename(lock, lock+"-old") != nil || os.WriteFile(lock, nil, 0600) != nil {
		t.Fatal("owned lock replacement")
	}
	check("lock_denial", errors.Is(drift.PublishJob(j, 0), ErrStateDirectory))
	check("lease_close", s.Close() == nil)
	check("closed_denial", errors.Is(s.PublishJob(j, 0), ErrStateDirectory))
	check("inert_denial", errors.Is((*StateDirectory)(nil).PublishJob(j, 0), ErrStateDirectory) && errors.Is((&StateDirectory{}).PublishJob(j, 0), ErrStateDirectory))
	b, e = os.ReadFile(foreign)
	check("foreign_preservation", e == nil && bytes.Equal(b, data))
	for _, lease := range leases {
		if lease.Close() != nil {
			t.Fatal("owned close")
		}
	}
	check("fixture_teardown", os.RemoveAll(anchor) == nil)
	t.Log("JOB_PUBLICATION_RESULT " + strings.Join(results, " "))
}
