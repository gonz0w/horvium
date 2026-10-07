// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestNativeJobReservations(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("owned root required")
	}
	anchor, err := os.MkdirTemp("/root", "horvium-reservations-")
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
			t.Fatal("reservation assertion", name)
		}
		results = append(results, name+"=pass")
	}
	lease := func(name string) (*StateDirectory, string) {
		t.Helper()
		path := filepath.Join(anchor, name)
		if os.Mkdir(path, 0700) != nil {
			t.Fatal("owned directory")
		}
		s, err := OpenStateDirectory(path)
		if err != nil {
			t.Fatal(err)
		}
		leases = append(leases, s)
		return s, path
	}
	write := func(path string, data []byte) {
		t.Helper()
		if os.WriteFile(path, data, 0600) != nil {
			t.Fatal("owned write")
		}
	}
	denied := func(s *StateDirectory) error {
		t.Helper()
		got, err := s.InspectJobReservations()
		if err == nil || !reflect.DeepEqual(got, JobReservations{}) {
			t.Fatal("partial reservation view", err)
		}
		return err
	}
	p := nativePropertyJobFixture(t)
	applying, err := TransitionPropertyJob(p, JobApplying, p.job.History[0].At+1, "launch-intent", "")
	if err != nil {
		t.Fatal(err)
	}
	data := reservationBytes(t, applying)
	owner := ownershipGolden()
	s, path := lease("normal")
	view, err := s.InspectJobReservations()
	check("empty_view", err == nil && view.ready && len(view.Pools()) == 0 && view.bytes == 0)
	propertyPath := filepath.Join(path, "job-"+p.job.ID+".json")
	write(propertyPath, data)
	write(filepath.Join(path, "ownership-100.json"), owner)
	j := reservationJob(1, true)
	j.Intent.PoolGUID = p.job.Intent.PoolGUID
	legacy := reservationBytes(t, j)
	legacyPath := filepath.Join(path, "job-"+j.ID+".json")
	write(legacyPath, legacy)
	uncertain, _ := TransitionJob(reservationJob(2, false), JobReconcileRequired, 1001, "review-expired", "")
	uncertainBytes := reservationBytes(t, uncertain)
	write(filepath.Join(path, "job-"+uncertain.ID+".json"), uncertainBytes)
	view, err = s.InspectJobReservations()
	check("mixed_holders", err == nil && len(view.Pools()) == 2 && len(view.Pools()[0].Jobs) == 2 && view.Pools()[0].Jobs[0].State == JobCancelled && view.Pools()[1].Jobs[0].State == JobReconcileRequired)
	total := uint64(len(data) + len(owner) + len(legacy) + len(uncertainBytes))
	check("mixed_byte_accounting", view.bytes == total && view.nonterminal == 2 && view.Pools()[0].Jobs[1].Bytes == uint64(len(data)))
	duplicate, matched, err := view.CheckPropertySubmission(p)
	check("complete_duplicate", err == nil && matched && reflect.DeepEqual(duplicate, applying))
	duplicate.job.History[0].Reason = "changed"
	duplicate.intent.review.Context[0].Name = "changed"
	pools := view.Pools()
	pools[0].Jobs[0].State = JobSucceeded
	again, err := s.InspectJobReservations()
	got, matched, dupErr := again.CheckPropertySubmission(p)
	check("independent_results", err == nil && dupErr == nil && matched && reflect.DeepEqual(got, applying) && reflect.DeepEqual(view, again))
	check("repeated_view", reflect.DeepEqual(view, again))
	fresh := PropertyJob{job: p.Job(), intent: p.Intent()}
	fresh.job.ID = strings.Repeat("d", 64)
	fresh.job.Intent.PlanID = strings.Repeat("9", 64)
	fresh.intent.review.ID = fresh.job.Intent.PlanID
	fresh.job.Intent.SubmissionID = strings.Repeat("8", 64)
	_, matched, err = view.CheckPropertySubmission(fresh)
	check("pool_hold_denial", errors.Is(err, ErrDenied) && !matched)
	fresh.job.Intent.PoolGUID = "777"
	fresh.intent.review.PoolGUID = "777"
	fresh.intent.review.Pool.GUID = "777"
	fresh.intent.review.Features.PoolGUID = "777"
	fresh.intent.review.Topology.PoolGUID = "777"
	_, matched, err = view.CheckPropertySubmission(fresh)
	check("unrelated_pool_check", err == nil && !matched)
	_, err = os.Lstat(filepath.Join(path, "job-"+fresh.job.ID+".json"))
	check("no_admission", os.IsNotExist(err))
	limits := stateCatalogLimits{1, 2, 1, total}
	exact, err := s.inspectJobReservations(limits, nil)
	check("exact_limits", err == nil && reflect.DeepEqual(exact.Pools(), view.Pools()) && exact.bytes == total)
	for _, tc := range []struct {
		name   string
		limits stateCatalogLimits
	}{
		{"byte_limit", stateCatalogLimits{1, 2, 1, total - 1}},
		{"nonterminal_limit", stateCatalogLimits{1, 1, 1, total}},
		{"invalid_limits", stateCatalogLimits{}},
	} {
		got, err := s.inspectJobReservations(tc.limits, nil)
		check(tc.name, errors.Is(err, ErrLimit) && reflect.DeepEqual(got, JobReservations{}))
	}
	// Candidate budget includes ownership bytes, not just jobs in the projection.
	folderS, folder := lease("owner-budget")
	write(filepath.Join(folder, "ownership-100.json"), owner)
	newSize := uint64(len(reservationBytes(t, fresh)))
	ownerView, err := folderS.inspectJobReservations(stateCatalogLimits{1, 1, 1, newSize}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, matched, err = ownerView.CheckPropertySubmission(fresh)
	check("ownership_budget", errors.Is(err, ErrLimit) && !matched && ownerView.bytes == uint64(len(owner)))
	pretty := append(append([]byte(nil), data...), bytes.Repeat([]byte(" "), maxStateRecordBytes-len(data))...)
	write(propertyPath, pretty)
	view, err = s.InspectJobReservations()
	check("presentation_bytes", err == nil && view.bytes == total+uint64(len(pretty)-len(data)) && view.Pools()[0].Jobs[1].Bytes == maxStateRecordBytes)
	write(propertyPath, data)
	for _, field := range []string{"submission", "plan"} {
		collision := reservationJob(3, false)
		if field == "submission" {
			collision.Intent.SubmissionID = p.job.Intent.SubmissionID
		} else {
			collision.Intent.PlanID = p.job.Intent.PlanID
		}
		target := filepath.Join(path, "job-"+collision.ID+".json")
		write(target, reservationBytes(t, collision))
		check(field+"_collision", errors.Is(denied(s), ErrDrift))
		if os.Remove(target) != nil {
			t.Fatal("collision cleanup")
		}
	}
	foreign := filepath.Join(anchor, "foreign")
	write(foreign, data)
	for _, kind := range []string{"symlink", "hardlink", "fifo", "directory", "mode", "owner", "empty", "oversize", "damage", "newer", "wrong_id", "wrong_kind"} {
		fixture, folder := lease("bad-" + kind)
		target := filepath.Join(folder, "job-"+p.job.ID+".json")
		bad := data
		switch kind {
		case "symlink":
			err = os.Symlink(foreign, target)
		case "hardlink":
			err = os.Link(foreign, target)
		case "fifo":
			err = unix.Mkfifo(target, 0600)
		case "directory":
			err = os.Mkdir(target, 0700)
		default:
			switch kind {
			case "empty":
				bad = nil
			case "oversize":
				bad = bytes.Repeat([]byte("x"), maxStateRecordBytes+1)
			case "damage":
				bad = append(append([]byte(nil), data...), 'x')
			case "newer":
				bad = bytes.Replace(data, []byte(`"version":1`), []byte(`"version":2`), 1)
			case "wrong_id":
				bad = legacy
			case "wrong_kind":
				bad = owner
			}
			err = os.WriteFile(target, bad, 0600)
			if err == nil && kind == "mode" {
				err = os.Chmod(target, 0644)
			}
			if err == nil && kind == "owner" {
				err = os.Chown(target, 65534, 65534)
			}
		}
		if err != nil {
			t.Fatal(kind, err)
		}
		before, err := os.Lstat(target)
		if err != nil {
			t.Fatal(err)
		}
		_ = denied(fixture)
		after, err := os.Lstat(target)
		check("record_"+kind+"_denial", err == nil && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size())
		if after.Mode().IsRegular() {
			current, err := os.ReadFile(target)
			if err != nil || !bytes.Equal(current, bad) {
				t.Fatal("denied bytes changed")
			}
		}
	}
	write(legacyPath, []byte("damaged"))
	check("legacy_damage_denial", errors.Is(denied(s), ErrStateRecord))
	write(legacyPath, legacy)
	ownerPath := filepath.Join(path, "ownership-100.json")
	write(ownerPath, []byte("damaged"))
	check("ownership_damage_denial", errors.Is(denied(s), ErrStateRecord))
	write(ownerPath, owner)
	for _, name := range []string{"unknown", "job-1.json"} {
		target := filepath.Join(path, name)
		write(target, owner)
		if !errors.Is(denied(s), ErrStateRecord) {
			t.Fatal("unknown accepted")
		}
		if os.Remove(target) != nil {
			t.Fatal("unknown cleanup")
		}
	}
	check("unknown_denial", true)
	for _, terminal := range []bool{false, true} {
		name, limit := "nonterminal", 64
		if terminal {
			name, limit = "terminal", 1000
		}
		fixture, folder := lease(name + "-capacity")
		for i := 0; i < limit; i++ {
			j := reservationJob(i, terminal)
			write(filepath.Join(folder, "job-"+j.ID+".json"), reservationBytes(t, j))
		}
		full, err := fixture.InspectJobReservations()
		check(name+"_ceiling", err == nil && len(full.Pools()) == limit)
		j := reservationJob(limit, terminal)
		write(filepath.Join(folder, "job-"+j.ID+".json"), reservationBytes(t, j))
		check(name+"_exhaustion", errors.Is(denied(fixture), ErrLimit))
	}
	orphan := filepath.Join(path, ".job-"+strings.Repeat("a", 32)+".tmp")
	write(orphan, []byte("owned incomplete"))
	check("orphan_denial", errors.Is(denied(s), ErrStateUncertain))
	current, err := os.ReadFile(orphan)
	check("orphan_preservation", err == nil && string(current) == "owned incomplete")
	if os.Remove(orphan) != nil {
		t.Fatal("orphan cleanup")
	}
	for _, tc := range []struct {
		name, path string
		bytes      []byte
	}{
		{"property_drift", propertyPath, data}, {"ownership_drift", ownerPath, owner},
	} {
		got, err := s.inspectJobReservations(defaultStateCatalogLimits, func() { write(tc.path, append(append([]byte(nil), tc.bytes...), 'x')) })
		check(tc.name, errors.Is(err, ErrDrift) && reflect.DeepEqual(got, JobReservations{}))
		write(tc.path, tc.bytes)
	}
	extra := filepath.Join(path, "late-unknown")
	zero, err := s.inspectJobReservations(defaultStateCatalogLimits, func() { write(extra, owner) })
	check("namespace_drift", errors.Is(err, ErrDrift) && reflect.DeepEqual(zero, JobReservations{}))
	if os.Remove(extra) != nil {
		t.Fatal("late cleanup")
	}
	moved := filepath.Join(anchor, "moved")
	if os.Rename(path, moved) != nil {
		t.Fatal("owned rename")
	}
	view, err = s.InspectJobReservations()
	check("directory_anchor", err == nil && view.bytes == total)
	drift, folder := lease("directory-drift")
	if os.Chmod(folder, 0755) != nil {
		t.Fatal("owned chmod")
	}
	check("directory_denial", errors.Is(denied(drift), ErrStateDirectory))
	drift, folder = lease("lock-drift")
	lock := filepath.Join(folder, stateWriterName)
	if os.Rename(lock, lock+"-old") != nil {
		t.Fatal("owned lock rename")
	}
	write(lock, nil)
	check("lock_denial", errors.Is(denied(drift), ErrStateDirectory))
	blocked, _ := lease("uncertain")
	err = blocked.publishPropertyJob(p, 0, func(tx ownershipPublication) ownershipPublication {
		tx.syncDirectory = func() error { return errors.New("owned seam") }
		return tx
	})
	if !errors.Is(err, ErrStateUncertain) {
		t.Fatal("owned uncertainty", err)
	}
	view, err = blocked.InspectJobReservations()
	check("uncertainty_preserved", err == nil && len(view.Pools()) == 1 && errors.Is(blocked.PublishPropertyJob(applying, 1), ErrStateUncertain))
	check("lease_close", s.Close() == nil)
	check("closed_denial", errors.Is(denied(s), ErrStateDirectory))
	check("inert_denial", errors.Is(denied(nil), ErrStateDirectory) && errors.Is(denied(&StateDirectory{}), ErrStateDirectory))
	current, err = os.ReadFile(foreign)
	check("foreign_preservation", err == nil && bytes.Equal(current, data))
	for _, s := range leases {
		if s.Close() != nil {
			t.Fatal("owned close")
		}
	}
	if os.RemoveAll(anchor) != nil {
		t.Fatal("owned teardown")
	}
	_, err = os.Lstat(anchor)
	check("fixture_teardown", os.IsNotExist(err))
	t.Log("JOB_RESERVATIONS_RESULT " + strings.Join(results, " "))
}
