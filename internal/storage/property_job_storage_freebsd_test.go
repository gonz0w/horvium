// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func nativePropertyJobFixture(t testing.TB) PropertyJob {
	t.Helper()
	p := propertyJobFixture(t)
	p.job.ID = strings.Repeat("e", 64)
	p.job.Intent.SubmissionID = strings.Repeat("f", 64)
	return p
}

func TestNativePropertyJobStorage(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("owned root required")
	}
	anchor, e := os.MkdirTemp("/root", "horvium-property-state-")
	if e != nil {
		t.Fatal(e)
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
			t.Fatal("property state assertion", name)
		}
		results = append(results, name+"=pass")
	}
	lease := func(name string) (*StateDirectory, string) {
		t.Helper()
		path := filepath.Join(anchor, name)
		if os.Mkdir(path, 0700) != nil {
			t.Fatal("owned directory")
		}
		s, e := OpenStateDirectory(path)
		if e != nil {
			t.Fatal(e)
		}
		leases = append(leases, s)
		return s, path
	}
	write := func(name string, b []byte) {
		t.Helper()
		if os.WriteFile(name, b, 0600) != nil {
			t.Fatal("owned write")
		}
	}
	noTemps := func(path string) bool {
		entries, e := os.ReadDir(path)
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
	deniedRead := func(s *StateDirectory, id string) error {
		t.Helper()
		p, e := s.ReadPropertyJob(id)
		if e == nil || !reflect.DeepEqual(p, PropertyJob{}) {
			t.Fatal("unsafe partial baseline", e)
		}
		return e
	}
	p := nativePropertyJobFixture(t)
	data, _ := EncodePropertyJob(p)
	at := p.job.History[0].At + 1
	applying, e := TransitionPropertyJob(p, JobApplying, at, "launch-intent", "")
	if e != nil {
		t.Fatal(e)
	}
	nextBytes, _ := EncodePropertyJob(applying)
	s, path := lease("normal")
	target := filepath.Join(path, "job-"+p.job.ID+".json")
	check("missing_denial", deniedRead(s, p.job.ID) != nil)
	for _, id := range []string{"", "../writer.lock", strings.Repeat("E", 64), strings.Repeat("e", 63), p.job.ID + ".json"} {
		if !errors.Is(deniedRead(s, id), ErrInvalid) {
			t.Fatal("selector category")
		}
	}
	check("selector_denial", true)
	check("create", s.PublishPropertyJob(p, 0) == nil)
	b, e := os.ReadFile(target)
	check("canonical_bytes", e == nil && bytes.Equal(b, data))
	old, e := os.Stat(target)
	if e != nil {
		t.Fatal(e)
	}
	st := old.Sys().(*syscall.Stat_t)
	check("private_file", st.Uid == 0 && st.Mode&07777 == 0600 && st.Nlink == 1 && old.Mode().IsRegular())
	got, e := s.ReadPropertyJob(p.job.ID)
	check("complete_read", e == nil && reflect.DeepEqual(got, p))
	view := got.Intent()
	view.review.Context[0].Name = "changed"
	viewJob := got.Job()
	viewJob.History[0].Reason = "changed"
	got, e = s.ReadPropertyJob(p.job.ID)
	check("independent_read", e == nil && reflect.DeepEqual(got, p))
	exact := append(append([]byte(nil), data...), bytes.Repeat([]byte(" "), maxStateRecordBytes-len(data))...)
	write(target, exact)
	got, e = s.ReadPropertyJob(p.job.ID)
	check("exact_size", e == nil && reflect.DeepEqual(got, p))
	write(target, data)
	check("legacy_reader_refuses", func() bool { j, e := s.ReadJob(p.job.ID); return e != nil && reflect.DeepEqual(j, Job{}) }())
	check("legacy_publisher_refuses_conversion", s.PublishJob(applying.Job(), 1) != nil)
	check("duplicate_denial", errors.Is(s.PublishPropertyJob(p, 0), ErrDrift))
	check("revision_denial", errors.Is(s.PublishPropertyJob(applying, 0), ErrDrift) && errors.Is(s.PublishPropertyJob(applying, 2), ErrDrift) && errors.Is(s.PublishPropertyJob(applying, ^uint64(0)), ErrDrift))
	check("invalid_denial", s.PublishPropertyJob(PropertyJob{}, 0) != nil)
	held, e := os.Open(target)
	if e != nil {
		t.Fatal(e)
	}
	lockBefore, _ := os.Stat(filepath.Join(path, stateWriterName))
	check("replace", s.PublishPropertyJob(applying, 1) == nil)
	oldBytes, re := io.ReadAll(held)
	ce := held.Close()
	check("old_descriptor_preserved", re == nil && ce == nil && bytes.Equal(oldBytes, data))
	current, e := os.Stat(target)
	b, be := os.ReadFile(target)
	check("new_inode", e == nil && be == nil && !os.SameFile(old, current) && bytes.Equal(b, nextBytes))
	lockAfter, e := os.Stat(filepath.Join(path, stateWriterName))
	check("stable_lock", e == nil && os.SameFile(lockBefore, lockAfter))
	check("stale_denial", errors.Is(s.PublishPropertyJob(applying, 1), ErrDrift))
	observing, e := TransitionPropertyJob(applying, JobObserving, at+1, "native-observation", strings.Repeat("b", 64))
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name   string
		change func(*PropertyJob)
	}{
		{"baseline_rewrite", func(p *PropertyJob) { p.intent.review.Pool.LoadGUID = "999" }},
		{"session_rewrite", func(p *PropertyJob) { p.intent.sessionDigest[0] ^= 1 }},
		{"policy_rewrite", func(p *PropertyJob) { p.intent.policyDigest[0] ^= 1 }},
		{"capture_rewrite", func(p *PropertyJob) { p.intent.capturedAt = p.intent.capturedAt.Add(-1) }},
		{"history_rewrite", func(p *PropertyJob) { p.job.History[0].At++ }},
	} {
		changed, _ := BindPropertyJob(observing.Job(), observing.Intent())
		tc.change(&changed)
		if _, e := EncodePropertyJob(changed); e != nil {
			t.Fatal("independently valid rewrite", e)
		}
		check(tc.name+"_denial", errors.Is(s.PublishPropertyJob(changed, 2), ErrDrift))
	}
	check("observing", s.PublishPropertyJob(observing, 2) == nil)
	terminal, e := TransitionPropertyJob(observing, JobSucceeded, at+2, "verified", strings.Repeat("c", 64))
	if e != nil {
		t.Fatal(e)
	}
	check("terminal", s.PublishPropertyJob(terminal, 3) == nil)
	check("terminal_replay_denial", errors.Is(s.PublishPropertyJob(terminal, 4), ErrDrift))
	owner, _ := DecodeOwnershipRecord(ownershipGolden())
	check("mixed_owner", s.PublishOwnership(owner, 0) == nil)
	legacy := catalogJob(1, false)
	check("mixed_legacy", s.PublishJob(legacy, 0) == nil)
	terminalBytes, _ := EncodePropertyJob(terminal)
	legacyBytes, _ := EncodeJobRecord(legacy)
	catalog, e := s.InspectState()
	check("mixed_accounting", e == nil && len(catalog.Ownership) == 1 && len(catalog.Jobs) == 2 && catalog.Nonterminal == 1 && catalog.Terminal == 1 && catalog.Bytes == uint64(len(ownershipGolden())+len(legacyBytes)+len(terminalBytes)))
	legacyCollision := catalogJob(2, false)
	legacyCollision.Intent.SubmissionID = p.job.Intent.SubmissionID
	check("legacy_submission_collision", errors.Is(s.PublishJob(legacyCollision, 0), ErrDrift))
	propertyCollision := p
	propertyCollision.job.ID = strings.Repeat("d", 64)
	propertyCollision.job.Intent.SubmissionID = legacy.Intent.SubmissionID
	check("property_submission_collision", errors.Is(s.PublishPropertyJob(propertyCollision, 0), ErrDrift))
	legacyNext, _ := TransitionJob(legacy, JobApplying, 1001, "launch-intent", "")
	conversion := p
	conversion.job.ID = legacy.ID
	conversion.job.Intent.SubmissionID = legacy.Intent.SubmissionID
	conversion, _ = TransitionPropertyJob(conversion, JobApplying, at, "launch-intent", "")
	check("property_publisher_refuses_conversion", s.PublishPropertyJob(conversion, 1) != nil)
	check("no_temporary", noTemps(path))
	bounds, folder := lease("bounds")
	invoked := false
	wrap := func(tx ownershipPublication) ownershipPublication { invoked = true; return tx }
	limits := stateCatalogLimits{1, 1, 1, uint64(len(data)) - 1}
	check("create_byte_denial", errors.Is(bounds.publishPropertyJobBounded(p, 0, limits, wrap), ErrLimit) && !invoked && noTemps(folder))
	limits.bytes++
	check("exact_create", bounds.publishPropertyJobBounded(p, 0, limits, nil) == nil)
	limits.bytes = uint64(len(data)+len(nextBytes)) - 1
	check("replacement_reservation", errors.Is(bounds.publishPropertyJobBounded(applying, 1, limits, wrap), ErrLimit) && !invoked)
	b, e = os.ReadFile(filepath.Join(folder, "job-"+p.job.ID+".json"))
	check("budget_preservation", e == nil && bytes.Equal(b, data) && noTemps(folder))
	limits.bytes++
	check("exact_replace", bounds.publishPropertyJobBounded(applying, 1, limits, nil) == nil)
	limits.bytes = 64 << 20
	second := p
	second.job.ID = strings.Repeat("d", 64)
	second.job.Intent.SubmissionID = strings.Repeat("d", 64)
	check("nonterminal_limit", errors.Is(bounds.publishPropertyJobBounded(second, 0, limits, wrap), ErrLimit) && !invoked)
	check("mixed_owner_byte_denial", errors.Is(bounds.publishOwnershipBounded(owner, 0, ownershipPublicationLimits{1, uint64(len(nextBytes)+len(ownershipGolden())) - 1}, wrap), ErrLimit) && !invoked)
	check("mixed_owner_exact", bounds.publishOwnershipBounded(owner, 0, ownershipPublicationLimits{1, uint64(len(nextBytes) + len(ownershipGolden()))}, nil) == nil)
	cancelled, _ := TransitionPropertyJob(second, JobCancelled, at, "operator-cancelled", "")
	cancelledBytes, _ := EncodePropertyJob(cancelled)
	write(filepath.Join(folder, "job-"+second.job.ID+".json"), cancelledBytes)
	if bounds.PublishPropertyJob(observing, 2) != nil {
		t.Fatal("owned observing")
	}
	check("terminal_limit", errors.Is(bounds.publishPropertyJobBounded(terminal, 3, limits, wrap), ErrLimit) && !invoked)
	check("invalid_limits", errors.Is(bounds.publishPropertyJobBounded(second, 0, stateCatalogLimits{}, wrap), ErrLimit) && !invoked)
	foreign := filepath.Join(anchor, "foreign")
	write(foreign, data)
	foreignStat, _ := os.Stat(foreign)
	for _, kind := range []string{"mode", "owner", "symlink", "hardlink", "fifo", "directory", "empty", "oversize", "damage", "newer", "wrong_id", "legacy", "binding"} {
		unsafe, dir := lease("unsafe-" + kind)
		name := filepath.Join(dir, "job-"+p.job.ID+".json")
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
			case "empty":
				control = nil
			case "oversize":
				control = bytes.Repeat([]byte("x"), maxStateRecordBytes+1)
			case "damage":
				control = []byte("damaged")
			case "newer":
				control = bytes.Replace(data, []byte(`"version":1`), []byte(`"version":2`), 1)
			case "wrong_id":
				control = cancelledBytes
			case "legacy":
				j := legacy
				j.ID = p.job.ID
				control, _ = EncodeJobRecord(j)
			case "binding":
				control = propertyJobResign(bytes.Replace(data, []byte(`"plan_id":"`+p.job.Intent.PlanID+`"`), []byte(`"plan_id":"`+strings.Repeat("b", 64)+`"`), 1))
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
			t.Fatal("unsafe fixture", kind, e)
		}
		check(kind+"_read_denial", deniedRead(unsafe, p.job.ID) != nil)
		check(kind+"_publish_denial", unsafe.PublishPropertyJob(applying, 1) != nil && noTemps(dir))
		if kind != "symlink" && kind != "fifo" && kind != "directory" {
			b, e = os.ReadFile(name)
			if e != nil || !bytes.Equal(b, control) {
				t.Fatal("unsafe bytes changed", kind)
			}
		}
	}
	for _, phase := range []string{"write", "syncFile", "closeFile", "verify", "rename"} {
		failure, dir := lease("phase-" + phase)
		injected := errors.New("owned phase error")
		e = failure.publishPropertyJobBounded(p, 0, defaultStateCatalogLimits, func(tx ownershipPublication) ownershipPublication {
			fail := func() error { return injected }
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
			}
			return tx
		})
		check(phase+"_failure", errors.Is(e, injected) && noTemps(dir))
	}
	uncertain, dir := lease("uncertain")
	e = uncertain.publishPropertyJobBounded(p, 0, defaultStateCatalogLimits, func(tx ownershipPublication) ownershipPublication {
		tx.syncDirectory = func() error { return errors.New("owned sync failure") }
		return tx
	})
	check("directory_sync_uncertain", errors.Is(e, ErrStateUncertain))
	got, e = uncertain.ReadPropertyJob(p.job.ID)
	check("uncertain_readable", e == nil && reflect.DeepEqual(got, p))
	check("uncertain_all_publishers_blocked", errors.Is(uncertain.PublishPropertyJob(applying, 1), ErrStateUncertain) && errors.Is(uncertain.PublishJob(legacy, 0), ErrStateUncertain) && errors.Is(uncertain.PublishOwnership(owner, 0), ErrStateUncertain) && noTemps(dir))
	late, dir := lease("late")
	e = late.publishPropertyJobBounded(p, 0, defaultStateCatalogLimits, func(tx ownershipPublication) ownershipPublication {
		prior := tx.verify
		tx.verify = func() error { write(filepath.Join(dir, "unknown"), []byte("preserve")); return prior() }
		return tx
	})
	check("late_unknown_denial", errors.Is(e, ErrStateRecord) && noTemps(dir))
	orphan, dir := lease("orphan")
	orphanName := filepath.Join(dir, ".job-"+strings.Repeat("a", 32)+".tmp")
	write(orphanName, data)
	check("orphan_latch", errors.Is(orphan.PublishPropertyJob(p, 0), ErrStateUncertain) && orphan.publicationBlocked)
	b, e = os.ReadFile(orphanName)
	check("orphan_preservation", e == nil && bytes.Equal(b, data))
	staged, dir := lease("staged")
	if staged.PublishPropertyJob(p, 0) != nil {
		t.Fatal("owned staged")
	}
	external := p.Intent()
	external.review.Pool.LoadGUID = "999"
	changed, _ := BindPropertyJob(p.Job(), external)
	changedBytes, _ := EncodePropertyJob(changed)
	e = staged.publishPropertyJobBounded(applying, 1, defaultStateCatalogLimits, func(tx ownershipPublication) ownershipPublication {
		prior := tx.syncFile
		tx.syncFile = func() error { write(filepath.Join(dir, "job-"+p.job.ID+".json"), changedBytes); return prior() }
		return tx
	})
	b, be = os.ReadFile(filepath.Join(dir, "job-"+p.job.ID+".json"))
	check("staged_baseline_denial", errors.Is(e, ErrDrift) && be == nil && bytes.Equal(b, changedBytes) && noTemps(dir))
	race, _ := lease("race")
	answers := make(chan error, 2)
	for n := 0; n < 2; n++ {
		go func() { answers <- race.PublishPropertyJob(p, 0) }()
	}
	a, z := <-answers, <-answers
	check("serialized_duplicate", a == nil && errors.Is(z, ErrDrift) || z == nil && errors.Is(a, ErrDrift))
	moved := filepath.Join(anchor, "moved")
	if os.Rename(path, moved) != nil {
		t.Fatal("owned rename")
	}
	got, e = s.ReadPropertyJob(p.job.ID)
	check("directory_anchor", e == nil && reflect.DeepEqual(got, terminal) && s.PublishJob(legacyNext, 1) == nil)
	check("lease_close", s.Close() == nil)
	check("closed_denial", deniedRead(s, p.job.ID) != nil && s.PublishPropertyJob(p, 0) != nil)
	check("inert_denial", deniedRead(&StateDirectory{}, p.job.ID) != nil && (&StateDirectory{}).PublishPropertyJob(p, 0) != nil)
	b, e = os.ReadFile(foreign)
	after, _ := os.Stat(foreign)
	check("foreign_preservation", e == nil && bytes.Equal(b, data) && os.SameFile(foreignStat, after))
	for _, s := range leases {
		if s.Close() != nil {
			t.Fatal("owned close")
		}
	}
	if os.RemoveAll(anchor) != nil {
		t.Fatal("owned cleanup")
	}
	_, e = os.Lstat(anchor)
	check("fixture_teardown", os.IsNotExist(e))
	t.Log("PROPERTY_JOB_STORAGE_RESULT " + strings.Join(results, " "))
}
