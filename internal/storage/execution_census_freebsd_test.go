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

func TestNativeExecutionCensus(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("owned root required")
	}
	anchor, err := nativeStateAnchor(t, "horvium-execution-census-")
	if err != nil {
		t.Fatal(err)
	}
	var results []string
	var leases []*StateDirectory
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("execution census assertion", name)
		}
		results = append(results, name+"=pass")
	}
	newLease := func(name string) (*StateDirectory, string) {
		t.Helper()
		path := filepath.Join(anchor, name)
		if os.Mkdir(path, 0700) != nil {
			t.Fatal("owned directory creation")
		}
		s, e := OpenStateDirectory(path)
		if e != nil {
			t.Fatal(e)
		}
		leases = append(leases, s)
		t.Cleanup(func() { _ = s.Close() })
		return s, path
	}
	// Synthetic binding tests bytes only; it cannot establish persistent native
	// installation/object/backing identities or independent restore provenance.
	binding := executionJournalFixture().Binding
	limits := defaultStateCatalogLimits
	denied := func(s *StateDirectory, b executionJournalBinding, l stateCatalogLimits, drift func()) error {
		t.Helper()
		c, digest, e := s.inspectExecutionCensus(b, l, drift)
		if e == nil || digest != "" || !reflect.DeepEqual(c, StateCatalog{}) {
			t.Fatal("partial native census escaped", e)
		}
		return e
	}
	s, path := newLease("normal")
	c, empty, e := s.inspectExecutionCensus(binding, limits, nil)
	check("empty_lock", e == nil && c.Bytes == 0 && empty == expectedExecutionCensus(binding, nil, nil))
	policy, e := DecodeOwnershipRecord(ownershipGolden())
	if e != nil || s.PublishOwnership(policy, 0) != nil {
		t.Fatal("owned policy publication", e)
	}
	j := catalogJob(1, false)
	if s.PublishJob(j, 0) != nil {
		t.Fatal("owned job publication")
	}
	jobName := "job-" + j.ID + ".json"
	jobBytes, _ := EncodeJobRecord(j)
	names := []string{jobName, "ownership-100.json"}
	records := map[string][]byte{jobName: jobBytes, "ownership-100.json": ownershipGolden()}
	c, first, e := s.inspectExecutionCensus(binding, limits, nil)
	check("published_mixed_bytes", e == nil && first != empty && first == expectedExecutionCensus(binding, names, records))
	plain, e := s.InspectState()
	check("same_complete_catalog", e == nil && reflect.DeepEqual(c, plain))
	prior, e := os.Stat(filepath.Join(path, jobName))
	if e != nil {
		t.Fatal(e)
	}
	_, repeat, e := s.inspectExecutionCensus(binding, limits, nil)
	check("repeat", e == nil && repeat == first)
	after, e := os.Stat(filepath.Join(path, jobName))
	check("inode_preservation", e == nil && os.SameFile(prior, after))
	_, digest, e := s.inspectExecutionCensus(binding, stateCatalogLimits{1, 1, 1, c.Bytes}, nil)
	check("exact_limits", e == nil && digest == first)
	check("byte_limit", errors.Is(denied(s, binding, stateCatalogLimits{1, 1, 1, c.Bytes - 1}, nil), ErrLimit))
	check("invalid_limits", errors.Is(denied(s, binding, stateCatalogLimits{}, nil), ErrLimit))
	bad := binding
	bad.Lock = ""
	check("invalid_binding", errors.Is(denied(s, bad, limits, nil), ErrStateRecord))
	bad = binding
	bad.Installation = strings.Repeat("9", 32)
	_, digest, e = s.inspectExecutionCensus(bad, limits, nil)
	check("binding_bytes", e == nil && digest != first && digest == expectedExecutionCensus(bad, names, records))
	alias := &StateDirectory{directory: s.directory, writer: s.writer, propertyOrigin: s}
	check("copied_handle", errors.Is(denied(alias, binding, limits, nil), ErrDenied))
	_, digest, e = s.inspectExecutionCensus(binding, limits, nil)
	check("original_survives_copy", e == nil && digest == first)
	// Same semantic record with different presentation must change the census.
	ownerPath := filepath.Join(path, "ownership-100.json")
	presented := append(append([]byte(nil), ownershipGolden()...), '\n')
	if os.WriteFile(ownerPath, presented, 0600) != nil {
		t.Fatal("owned presentation")
	}
	records["ownership-100.json"] = presented
	_, digest, e = s.inspectExecutionCensus(binding, limits, nil)
	check("exact_presentation", e == nil && digest != first && digest == expectedExecutionCensus(binding, names, records))
	policy.Revision++
	if s.PublishOwnership(policy, 1) != nil {
		t.Fatal("owned successor publication")
	}
	records["ownership-100.json"], e = EncodeOwnershipRecord(policy)
	if e != nil {
		t.Fatal(e)
	}
	_, successor, e := s.inspectExecutionCensus(binding, limits, nil)
	check("published_successor", e == nil && successor != digest && successor == expectedExecutionCensus(binding, names, records))
	journal := executionJournalFixture()
	journal.Census = successor
	wire, e := encodeExecutionJournal(journal)
	if e != nil {
		t.Fatal(e)
	}
	decoded, e := decodeExecutionJournal(wire)
	check("journal_consumes_census", e == nil && decoded == journal)
	for _, name := range names {
		target := filepath.Join(path, name)
		marker := "job_drift"
		if name == "ownership-100.json" {
			marker = "ownership_drift"
		}
		check(marker, errors.Is(denied(s, binding, limits, func() {
			if os.WriteFile(target, append(append([]byte(nil), records[name]...), ' '), 0600) != nil {
				t.Fatal("owned drift")
			}
		}), ErrDrift))
		if os.WriteFile(target, records[name], 0600) != nil {
			t.Fatal("owned restore")
		}
	}
	unknown := filepath.Join(path, "unknown")
	check("namespace_drift", errors.Is(denied(s, binding, limits, func() {
		if os.WriteFile(unknown, nil, 0600) != nil {
			t.Fatal("owned namespace drift")
		}
	}), ErrDrift))
	check("unknown_name", errors.Is(denied(s, binding, limits, nil), ErrStateRecord))
	if os.Remove(unknown) != nil {
		t.Fatal("owned unknown cleanup")
	}
	orphan := filepath.Join(path, ".job-"+strings.Repeat("a", 32)+".tmp")
	if os.WriteFile(orphan, []byte("owned pending bytes"), 0600) != nil {
		t.Fatal("owned orphan")
	}
	check("orphan", errors.Is(denied(s, binding, limits, nil), ErrStateUncertain))
	data, e := os.ReadFile(orphan)
	check("orphan_preserved", e == nil && string(data) == "owned pending bytes")
	if os.Remove(orphan) != nil {
		t.Fatal("owned orphan cleanup")
	}
	duplicate := catalogJob(2, false)
	duplicate.Intent.SubmissionID = j.Intent.SubmissionID
	dupBytes, e := EncodeJobRecord(duplicate)
	dupPath := filepath.Join(path, "job-"+duplicate.ID+".json")
	if e != nil || os.WriteFile(dupPath, dupBytes, 0600) != nil {
		t.Fatal("owned duplicate")
	}
	check("submission_collision", errors.Is(denied(s, binding, limits, nil), ErrDrift))
	if os.Remove(dupPath) != nil {
		t.Fatal("owned duplicate cleanup")
	}
	if os.Remove(filepath.Join(path, jobName)) != nil {
		t.Fatal("owned omission")
	}
	_, digest, e = s.inspectExecutionCensus(binding, limits, nil)
	check("omission_changes_digest", e == nil && digest != successor && digest == expectedExecutionCensus(binding, []string{"ownership-100.json"}, records))
	if os.WriteFile(filepath.Join(path, jobName), jobBytes, 0600) != nil {
		t.Fatal("owned job restore")
	}
	foreign := filepath.Join(anchor, "foreign")
	if os.WriteFile(foreign, jobBytes, 0600) != nil {
		t.Fatal("owned foreign")
	}
	for _, kind := range []string{"symlink", "hardlink", "fifo", "mode", "damage", "oversize"} {
		fixture, folder := newLease(kind)
		target := filepath.Join(folder, jobName)
		switch kind {
		case "symlink":
			e = os.Symlink(foreign, target)
		case "hardlink":
			e = os.Link(foreign, target)
		case "fifo":
			e = unix.Mkfifo(target, 0600)
		default:
			data = jobBytes
			if kind == "damage" {
				data = []byte("{}")
			}
			if kind == "oversize" {
				data = bytes.Repeat([]byte("x"), maxStateRecordBytes+1)
			}
			e = os.WriteFile(target, data, 0600)
			if e == nil && kind == "mode" {
				e = os.Chmod(target, 0644)
			}
		}
		if e != nil {
			t.Fatal("owned unsafe object", e)
		}
		denied(fixture, binding, limits, nil)
		check(kind+"_denial", true)
	}
	moved := filepath.Join(anchor, "moved")
	if os.Rename(path, moved) != nil {
		t.Fatal("owned directory rename")
	}
	path = moved
	_, digest, e = s.inspectExecutionCensus(binding, limits, nil)
	check("descriptor_anchor", e == nil && digest == successor)
	lock := filepath.Join(path, stateWriterName)
	check("late_lock_replacement", errors.Is(denied(s, binding, limits, func() {
		if os.Rename(lock, filepath.Join(anchor, "original-lock")) != nil || os.WriteFile(lock, nil, 0600) != nil {
			t.Fatal("owned lock replacement")
		}
	}), ErrStateDirectory))
	blocked, folder := newLease("uncertain")
	initial, e := DecodeOwnershipRecord(ownershipGolden())
	if e != nil {
		t.Fatal(e)
	}
	e = blocked.publishOwnership(initial, 0, func(tx ownershipPublication) ownershipPublication {
		tx.syncDirectory = func() error { return errors.New("owned sync failure") }
		return tx
	})
	if !errors.Is(e, ErrStateUncertain) {
		t.Fatal("owned uncertainty", e)
	}
	c, digest, e = blocked.inspectExecutionCensus(binding, limits, nil)
	check("uncertainty_observed_only", e == nil && validJobDigest(digest) && c.Ownership[0].Revision == initial.Revision && errors.Is(blocked.PublishOwnership(initial, 0), ErrStateUncertain))
	data, e = os.ReadFile(filepath.Join(folder, "ownership-100.json"))
	check("uncertain_bytes_preserved", e == nil && bytes.Equal(data, ownershipGolden()))
	check("lease_close", blocked.Close() == nil)
	check("closed_denial", errors.Is(denied(blocked, binding, limits, nil), ErrDenied))
	check("inert_denial", errors.Is(denied(nil, binding, limits, nil), ErrStateDirectory) && errors.Is(denied(&StateDirectory{}, binding, limits, nil), ErrDenied))
	data, e = os.ReadFile(foreign)
	check("foreign_preserved", e == nil && bytes.Equal(data, jobBytes))
	for _, lease := range leases {
		if lease.Close() != nil {
			t.Fatal("owned lease close")
		}
	}
	if os.RemoveAll(anchor) != nil {
		t.Fatal("owned fixture teardown")
	}
	_, e = os.Stat(anchor)
	check("fixture_teardown", os.IsNotExist(e))
	t.Log("EXECUTION_CENSUS_RESULT " + strings.Join(results, " "))
}
