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

func TestNativeStateCatalog(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("owned root required")
	}
	anchor, err := nativeStateAnchor(t, "horvium-state-catalog-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(anchor)
	var results []string
	var leases []*StateDirectory
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("state catalog assertion", name)
		}
		results = append(results, name+"=pass")
	}
	newLease := func(name string) (*StateDirectory, string) {
		t.Helper()
		p := filepath.Join(anchor, name)
		if os.Mkdir(p, 0700) != nil {
			t.Fatal("owned directory unavailable")
		}
		s, e := OpenStateDirectory(p)
		if e != nil {
			t.Fatal(e)
		}
		leases = append(leases, s)
		t.Cleanup(func() { _ = s.Close() })
		return s, p
	}
	writeJob := func(p string, j Job) []byte {
		t.Helper()
		data, e := EncodeJobRecord(j)
		if e != nil || os.WriteFile(filepath.Join(p, "job-"+j.ID+".json"), data, 0600) != nil {
			t.Fatal("owned job unavailable", e)
		}
		return data
	}
	denied := func(s *StateDirectory) error {
		t.Helper()
		got, e := s.InspectState()
		if e == nil || !reflect.DeepEqual(got, StateCatalog{}) {
			t.Fatal("unsafe partial census", e)
		}
		return e
	}
	s, path := newLease("normal")
	got, e := s.InspectState()
	check("empty_catalog", e == nil && len(got.Ownership) == 0 && len(got.Jobs) == 0 && got.Bytes == 0)
	owner := ownershipGolden()
	p, e := DecodeOwnershipRecord(owner)
	if e != nil || s.PublishOwnership(p, 0) != nil {
		t.Fatal("owned policy unavailable", e)
	}
	j := catalogJob(1, false)
	jobBytes := writeJob(path, j)
	terminal := catalogJob(2, true)
	terminalBytes := writeJob(path, terminal)
	uncertain, e := TransitionJob(catalogJob(3, false), JobReconcileRequired, 1001, "review-expired", "")
	if e != nil {
		t.Fatal(e)
	}
	uncertainBytes := writeJob(path, uncertain)
	total := uint64(len(owner) + len(jobBytes) + len(terminalBytes) + len(uncertainBytes))
	got, e = s.InspectState()
	check("mixed_catalog", e == nil && len(got.Ownership) == 1 && len(got.Jobs) == 3)
	check("sorted_records", got.Ownership[0].PoolGUID == "100" && got.Jobs[0].ID == j.ID && got.Jobs[1].ID == terminal.ID && got.Jobs[2].ID == uncertain.ID)
	check("byte_accounting", got.Bytes == total && got.Ownership[0].Bytes == uint64(len(owner)) && got.Jobs[0].Bytes == uint64(len(jobBytes)))
	check("class_counts", got.Nonterminal == 2 && got.Terminal == 1 && got.Jobs[2].State == JobReconcileRequired && got.Jobs[1].Revision == 2)
	got.Ownership[0].PoolGUID = "changed"
	got.Jobs[0].SubmissionID = "changed"
	got, e = s.InspectState()
	check("independent_results", e == nil && got.Ownership[0].PoolGUID == "100" && got.Jobs[0].SubmissionID == j.Intent.SubmissionID)
	again, e := s.InspectState()
	check("repeated_catalog", e == nil && reflect.DeepEqual(got, again))
	exact := stateCatalogLimits{1, 2, 1, total}
	got, e = s.inspectState(exact, nil)
	check("exact_limits", e == nil && reflect.DeepEqual(got, again))
	for _, tc := range []struct {
		name   string
		limits stateCatalogLimits
	}{{"ownership_limit", stateCatalogLimits{1, 2, 1, total}}, {"nonterminal_limit", stateCatalogLimits{1, 1, 2, total}}, {"terminal_limit", stateCatalogLimits{1, 2, 1, total}}, {"byte_limit", stateCatalogLimits{1, 2, 1, total - 1}}} {
		switch tc.name {
		case "ownership_limit":
			if os.WriteFile(filepath.Join(path, "ownership-200.json"), owner, 0600) != nil {
				t.Fatal("owned capacity control")
			}
		case "terminal_limit":
			writeJob(path, catalogJob(4, true))
		}
		got, e = s.inspectState(tc.limits, nil)
		check(tc.name, errors.Is(e, ErrLimit) && reflect.DeepEqual(got, StateCatalog{}))
		switch tc.name {
		case "ownership_limit":
			if os.Remove(filepath.Join(path, "ownership-200.json")) != nil {
				t.Fatal("owned capacity cleanup")
			}
		case "terminal_limit":
			if os.Remove(filepath.Join(path, "job-"+catalogJob(4, true).ID+".json")) != nil {
				t.Fatal("owned capacity cleanup")
			}
		}
	}
	for _, limits := range []stateCatalogLimits{{}, {maxRecords + 1, 64, 1000, 64 << 20}, {1, 65, 1000, 64 << 20}, {1, 64, 1001, 64 << 20}, {1, 64, 1000, (64 << 20) + 1}} {
		got, e = s.inspectState(limits, nil)
		if !errors.Is(e, ErrLimit) || !reflect.DeepEqual(got, StateCatalog{}) {
			t.Fatal("invalid fixture bounds")
		}
	}
	check("invalid_limits", true)
	duplicate := catalogJob(4, false)
	duplicate.Intent.SubmissionID = j.Intent.SubmissionID
	writeJob(path, duplicate)
	check("submission_collision", errors.Is(denied(s), ErrDrift))
	if os.Remove(filepath.Join(path, "job-"+duplicate.ID+".json")) != nil {
		t.Fatal("duplicate cleanup")
	}
	before, e := os.Stat(filepath.Join(path, "job-"+j.ID+".json"))
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(path, "job-"+j.ID+".json"))
	check("record_preservation", e == nil && bytes.Equal(b, jobBytes))
	after, e := os.Stat(filepath.Join(path, "job-"+j.ID+".json"))
	check("inode_preservation", e == nil && os.SameFile(before, after))
	moved := filepath.Join(anchor, "moved")
	if os.Rename(path, moved) != nil {
		t.Fatal("owned rename")
	}
	path = moved
	got, e = s.InspectState()
	check("directory_anchor", e == nil && got.Bytes == total)
	orphan := filepath.Join(path, ".ownership-"+strings.Repeat("a", 32)+".tmp")
	if os.WriteFile(orphan, []byte("owned incomplete intent"), 0600) != nil {
		t.Fatal("owned orphan")
	}
	check("orphan_denial", errors.Is(denied(s), ErrStateUncertain))
	b, e = os.ReadFile(orphan)
	check("orphan_preservation", e == nil && string(b) == "owned incomplete intent")
	if os.Remove(orphan) != nil {
		t.Fatal("orphan cleanup")
	}
	for _, name := range []string{"unknown", "job-1.json", "ownership-0100.json"} {
		target := filepath.Join(path, name)
		if os.WriteFile(target, owner, 0600) != nil {
			t.Fatal("unknown fixture")
		}
		if !errors.Is(denied(s), ErrStateRecord) {
			t.Fatal("unknown namespace accepted")
		}
		if os.Remove(target) != nil {
			t.Fatal("unknown cleanup")
		}
	}
	check("unknown_denial", true)
	// Actual fixed class ceilings are checked on wholly owned guest records.
	for _, terminal := range []bool{false, true} {
		name, limit := "nonterminal", 64
		if terminal {
			name, limit = "terminal", 1000
		}
		fixture, folder := newLease(name + "-capacity")
		for i := 1; i <= limit; i++ {
			writeJob(folder, catalogJob(i, terminal))
		}
		got, e = fixture.InspectState()
		check(name+"_ceiling", e == nil && len(got.Jobs) == limit)
		writeJob(folder, catalogJob(limit+1, terminal))
		check(name+"_exhaustion", errors.Is(denied(fixture), ErrLimit))
	}
	foreign := filepath.Join(anchor, "foreign")
	if os.WriteFile(foreign, jobBytes, 0600) != nil {
		t.Fatal("foreign fixture")
	}
	for _, kind := range []string{"symlink", "hardlink", "fifo", "directory", "mode", "owner", "empty", "oversize", "damage", "newer", "wrong_id", "wrong_kind"} {
		fixture, folder := newLease("job-" + kind)
		if os.WriteFile(filepath.Join(folder, "ownership-100.json"), owner, 0600) != nil {
			t.Fatal("mixed owner fixture")
		}
		target := filepath.Join(folder, "job-"+j.ID+".json")
		data := jobBytes
		switch kind {
		case "symlink":
			e = os.Symlink(foreign, target)
		case "hardlink":
			e = os.Link(foreign, target)
		case "fifo":
			e = unix.Mkfifo(target, 0600)
		case "directory":
			e = os.Mkdir(target, 0700)
		default:
			switch kind {
			case "empty":
				data = nil
			case "oversize":
				data = bytes.Repeat([]byte("x"), maxStateRecordBytes+1)
			case "damage":
				data = append(append([]byte(nil), jobBytes...), 'x')
			case "newer":
				data = bytes.Replace(jobBytes, []byte(`"version":1`), []byte(`"version":2`), 1)
			case "wrong_id":
				data = terminalBytes
			case "wrong_kind":
				data = owner
			}
			e = os.WriteFile(target, data, 0600)
			if e == nil && kind == "mode" {
				e = os.Chmod(target, 0644)
			}
			if e == nil && kind == "owner" {
				e = os.Chown(target, 65534, 65534)
			}
		}
		if e != nil {
			t.Fatal("owned denial fixture", kind, e)
		}
		original, e := os.Lstat(target)
		if e != nil {
			t.Fatal(e)
		}
		denied(fixture)
		current, e := os.Lstat(target)
		check("job_"+kind+"_denial", e == nil && os.SameFile(original, current) && original.Mode() == current.Mode() && original.Size() == current.Size())
		if current.Mode().IsRegular() {
			b, e = os.ReadFile(target)
			if e != nil || !bytes.Equal(b, data) {
				t.Fatal("denied bytes changed")
			}
		}
	}
	// Invalid ownership also invalidates an otherwise valid job census.
	fixture, folder := newLease("damaged-owner")
	writeJob(folder, j)
	if os.WriteFile(filepath.Join(folder, "ownership-100.json"), []byte("damaged"), 0600) != nil {
		t.Fatal("owned damage")
	}
	check("ownership_damage_denial", errors.Is(denied(fixture), ErrStateRecord))
	for _, name := range []string{"ownership-100.json", "job-" + j.ID + ".json"} {
		target := filepath.Join(path, name)
		data := owner
		marker := "ownership_record_drift"
		if strings.HasPrefix(name, "job-") {
			data = jobBytes
			marker = "job_record_drift"
		}
		got, e = s.inspectState(defaultStateCatalogLimits, func() {
			if os.WriteFile(target, append(append([]byte(nil), data...), 'x'), 0600) != nil {
				t.Fatal("owned drift")
			}
		})
		check(marker, errors.Is(e, ErrDrift) && reflect.DeepEqual(got, StateCatalog{}))
		if os.WriteFile(target, data, 0600) != nil {
			t.Fatal("owned restore")
		}
	}
	extra := filepath.Join(path, "unknown-after-read")
	got, e = s.inspectState(defaultStateCatalogLimits, func() {
		if os.WriteFile(extra, owner, 0600) != nil {
			t.Fatal("owned namespace drift")
		}
	})
	check("namespace_drift_denial", errors.Is(e, ErrDrift) && reflect.DeepEqual(got, StateCatalog{}))
	if os.Remove(extra) != nil {
		t.Fatal("owned cleanup")
	}
	drift, folder := newLease("directory-drift")
	if os.Chmod(folder, 0755) != nil {
		t.Fatal("owned chmod")
	}
	check("directory_drift_denial", errors.Is(denied(drift), ErrStateDirectory))
	drift, folder = newLease("lock-drift")
	lock := filepath.Join(folder, stateWriterName)
	if os.Rename(lock, lock+"-old") != nil || os.WriteFile(lock, nil, 0600) != nil {
		t.Fatal("owned lock drift")
	}
	check("lock_drift_denial", errors.Is(denied(drift), ErrStateDirectory))
	blocked, folder := newLease("uncertain")
	e = blocked.publishOwnership(p, 0, func(tx ownershipPublication) ownershipPublication {
		tx.syncDirectory = func() error { return errors.New("owned sync seam") }
		return tx
	})
	if !errors.Is(e, ErrStateUncertain) {
		t.Fatal("owned uncertainty", e)
	}
	writeJob(folder, j)
	got, e = blocked.InspectState()
	q := p
	q.PoolGUID = "200"
	check("uncertainty_preserved", e == nil && len(got.Jobs) == 1 && errors.Is(blocked.PublishOwnership(q, 0), ErrStateUncertain))
	check("lease_close", s.Close() == nil)
	check("closed_denial", errors.Is(denied(s), ErrStateDirectory))
	check("inert_denial", errors.Is(denied(nil), ErrStateDirectory) && errors.Is(denied(&StateDirectory{}), ErrStateDirectory))
	b, e = os.ReadFile(foreign)
	check("foreign_preservation", e == nil && bytes.Equal(b, jobBytes))
	for _, lease := range leases {
		if lease.Close() != nil {
			t.Fatal("owned lease close")
		}
	}
	if os.RemoveAll(anchor) != nil {
		t.Fatal("owned teardown")
	}
	_, e = os.Stat(anchor)
	check("fixture_teardown", os.IsNotExist(e))
	t.Log("STATE_CATALOG_RESULT " + strings.Join(results, " "))
}
