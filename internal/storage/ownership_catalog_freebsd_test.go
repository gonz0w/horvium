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

func TestNativeOwnershipCatalog(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("owned root required")
	}
	anchor, err := os.MkdirTemp("/root", "horvium-catalog-")
	if err != nil {
		t.Fatal("owned anchor unavailable", err)
	}
	defer os.RemoveAll(anchor)
	var results []string
	var leases []*StateDirectory
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("catalog assertion", name)
		}
		results = append(results, name+"=pass")
	}
	newLease := func(name string) (*StateDirectory, string) {
		t.Helper()
		path := filepath.Join(anchor, name)
		if os.Mkdir(path, 0700) != nil {
			t.Fatal("owned directory unavailable")
		}
		s, e := OpenStateDirectory(path)
		if e != nil {
			t.Fatal("owned lease unavailable", e)
		}
		leases = append(leases, s)
		t.Cleanup(func() { _ = s.Close() })
		return s, path
	}
	denied := func(s *StateDirectory) error {
		t.Helper()
		got, e := s.InspectOwnership()
		if e == nil || !reflect.DeepEqual(got, OwnershipCatalog{}) {
			t.Fatal("partial/unsafe catalog", e)
		}
		return e
	}
	good := ownershipGolden()
	foreign := filepath.Join(anchor, "foreign")
	if os.WriteFile(foreign, good, 0600) != nil {
		t.Fatal("foreign fixture unavailable")
	}
	p, e := DecodeOwnershipRecord(good)
	if e != nil {
		t.Fatal("policy unavailable", e)
	}
	s, path := newLease("normal")
	got, e := s.InspectOwnership()
	check("empty_catalog", e == nil && len(got.Records) == 0 && got.Bytes == 0)
	if s.PublishOwnership(p, 0) != nil {
		t.Fatal("owned publication unavailable")
	}
	q := p
	q.PoolGUID = "200"
	qbytes, e := EncodeOwnershipRecord(q)
	if e != nil || s.PublishOwnership(q, 0) != nil {
		t.Fatal("second publication unavailable", e)
	}
	got, e = s.InspectOwnership()
	check("complete_catalog", e == nil && len(got.Records) == 2)
	check("sorted_records", got.Records[0].PoolGUID == "100" && got.Records[1].PoolGUID == "200" && got.Records[0].Revision == p.Revision && got.Records[1].Revision == q.Revision)
	total := uint64(len(good) + len(qbytes))
	check("byte_accounting", got.Bytes == total && got.Records[0].Bytes == uint64(len(good)))
	got.Records[0].PoolGUID = "changed"
	again, e := s.InspectOwnership()
	check("independent_results", e == nil && again.Records[0].PoolGUID == "100")
	again, e = s.InspectOwnership()
	check("repeated_catalog", e == nil && again.Bytes == total && len(again.Records) == 2)
	exact, e := s.inspectOwnership(2, total, nil)
	check("exact_limits", e == nil && reflect.DeepEqual(exact, again))
	limited, e := s.inspectOwnership(1, total, nil)
	check("count_limit", errors.Is(e, ErrLimit) && reflect.DeepEqual(limited, OwnershipCatalog{}))
	limited, e = s.inspectOwnership(2, total-1, nil)
	check("byte_limit", errors.Is(e, ErrLimit) && reflect.DeepEqual(limited, OwnershipCatalog{}))
	for _, bounds := range []struct {
		count int
		size  uint64
	}{{0, total}, {maxRecords + 1, total}, {2, 0}, {2, maxOwnershipCatalogBytes + 1}} {
		limited, e = s.inspectOwnership(bounds.count, bounds.size, nil)
		if !errors.Is(e, ErrLimit) || !reflect.DeepEqual(limited, OwnershipCatalog{}) {
			t.Fatal("invalid fixture bounds accepted")
		}
	}
	check("invalid_limits", true)
	before, e := os.Stat(filepath.Join(path, "ownership-100.json"))
	if e != nil {
		t.Fatal("record metadata unavailable")
	}
	b, e := os.ReadFile(filepath.Join(path, "ownership-100.json"))
	check("record_preservation", e == nil && bytes.Equal(b, good))
	after, e := os.Stat(filepath.Join(path, "ownership-100.json"))
	check("inode_preservation", e == nil && os.SameFile(before, after))
	moved := filepath.Join(anchor, "moved")
	if os.Rename(path, moved) != nil {
		t.Fatal("owned rename failed")
	}
	path = moved
	got, e = s.InspectOwnership()
	check("directory_anchor", e == nil && got.Bytes == total)
	orphan := filepath.Join(path, ".ownership-"+strings.Repeat("a", 32)+".tmp")
	if os.WriteFile(orphan, []byte("owned incomplete intent"), 0600) != nil {
		t.Fatal("orphan fixture unavailable")
	}
	check("orphan_denial", errors.Is(denied(s), ErrStateUncertain))
	b, e = os.ReadFile(orphan)
	check("orphan_preservation", e == nil && string(b) == "owned incomplete intent")
	if os.Remove(orphan) != nil {
		t.Fatal("owned orphan cleanup failed")
	}
	for _, control := range []struct{ name, marker string }{{"job-1.json", "unknown_denial"}, {"ownership-0100.json", "malformed_name_denial"}} {
		file := filepath.Join(path, control.name)
		if os.WriteFile(file, good, 0600) != nil {
			t.Fatal("owned control unavailable")
		}
		check(control.marker, errors.Is(denied(s), ErrStateRecord))
		b, e = os.ReadFile(file)
		if e != nil || !bytes.Equal(b, good) || os.Remove(file) != nil {
			t.Fatal("unknown object changed or cleanup failed")
		}
	}
	// Every refusal control is newly created in this owned guest directory only.
	for _, kind := range []string{"symlink", "hardlink", "fifo", "directory", "mode", "owner", "empty", "oversize", "damage", "newer", "wrong_pool"} {
		fixture, folder := newLease("deny-" + kind)
		target := filepath.Join(folder, "ownership-100.json")
		data := good
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
				data = append(append([]byte(nil), good...), 'x')
			case "newer":
				data = bytes.Replace(good, []byte(`"version":1`), []byte(`"version":2`), 1)
			case "wrong_pool":
				data = qbytes
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
			t.Fatal("owned denial fixture unavailable", kind, e)
		}
		original, e := os.Lstat(target)
		if e != nil {
			t.Fatal("control stat unavailable")
		}
		denied(fixture)
		current, e := os.Lstat(target)
		check(kind+"_denial", e == nil && os.SameFile(original, current) && original.Mode() == current.Mode() && original.Size() == current.Size())
		if current.Mode().IsRegular() {
			b, e = os.ReadFile(target)
			if e != nil || !bytes.Equal(b, data) {
				t.Fatal("denied bytes changed", kind)
			}
		}
	}
	record := filepath.Join(path, "ownership-100.json")
	got, e = s.inspectOwnership(maxRecords, maxOwnershipCatalogBytes, func() {
		if os.WriteFile(record, append(append([]byte(nil), good...), 'x'), 0600) != nil {
			t.Fatal("owned drift unavailable")
		}
	})
	check("record_drift_denial", errors.Is(e, ErrDrift) && reflect.DeepEqual(got, OwnershipCatalog{}))
	if os.WriteFile(record, good, 0600) != nil {
		t.Fatal("owned restore unavailable")
	}
	extra := filepath.Join(path, "unknown-after-read")
	got, e = s.inspectOwnership(maxRecords, maxOwnershipCatalogBytes, func() {
		if os.WriteFile(extra, good, 0600) != nil {
			t.Fatal("owned namespace drift unavailable")
		}
	})
	check("namespace_drift_denial", errors.Is(e, ErrDrift) && reflect.DeepEqual(got, OwnershipCatalog{}))
	if os.Remove(extra) != nil {
		t.Fatal("owned namespace cleanup failed")
	}
	drift, folder := newLease("directory-drift")
	if os.Chmod(folder, 0755) != nil {
		t.Fatal("owned chmod unavailable")
	}
	check("directory_drift_denial", errors.Is(denied(drift), ErrStateDirectory))
	drift, folder = newLease("lock-drift")
	lock := filepath.Join(folder, stateWriterName)
	if os.Rename(lock, lock+"-old") != nil || os.WriteFile(lock, nil, 0600) != nil {
		t.Fatal("owned lock drift unavailable")
	}
	check("lock_drift_denial", errors.Is(denied(drift), ErrStateDirectory))
	uncertain, _ := newLease("uncertain")
	e = uncertain.publishOwnership(p, 0, func(tx ownershipPublication) ownershipPublication {
		tx.syncDirectory = func() error { return errors.New("owned directory sync seam") }
		return tx
	})
	if !errors.Is(e, ErrStateUncertain) {
		t.Fatal("owned uncertainty unavailable")
	}
	got, e = uncertain.InspectOwnership()
	check("uncertainty_preserved", e == nil && len(got.Records) == 1 && errors.Is(uncertain.PublishOwnership(q, 0), ErrStateUncertain))
	check("lease_close", s.Close() == nil)
	check("closed_denial", errors.Is(denied(s), ErrStateDirectory))
	check("inert_denial", errors.Is(denied(nil), ErrStateDirectory) && errors.Is(denied(&StateDirectory{}), ErrStateDirectory))
	b, e = os.ReadFile(foreign)
	check("foreign_preservation", e == nil && bytes.Equal(b, good))
	// Close all independently held fixture leases before removing their directories.
	for _, lease := range leases {
		if lease.Close() != nil {
			t.Fatal("owned lease cleanup failed")
		}
	}
	if os.RemoveAll(anchor) != nil {
		t.Fatal("owned cleanup failed")
	}
	_, e = os.Stat(anchor)
	check("fixture_teardown", os.IsNotExist(e))
	t.Log("OWNERSHIP_CATALOG_RESULT " + strings.Join(results, " "))
}
