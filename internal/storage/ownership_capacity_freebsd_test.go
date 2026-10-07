// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestNativeOwnershipCapacity(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("owned root required")
	}
	anchor, err := nativeStateAnchor(t, "horvium-capacity-")
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
			t.Fatal("capacity assertion", name)
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
			t.Fatal(e)
		}
		leases = append(leases, s)
		return s, path
	}
	write := func(path string, b []byte) {
		t.Helper()
		if os.WriteFile(path, b, 0600) != nil {
			t.Fatal("owned fixture write failed")
		}
	}
	good := ownershipGolden()
	p, err := DecodeOwnershipRecord(good)
	if err != nil {
		t.Fatal(err)
	}
	q := p
	q.PoolGUID = "200"
	qbytes, err := EncodeOwnershipRecord(q)
	if err != nil {
		t.Fatal(err)
	}
	next := p
	next.Revision++
	nextBytes, err := EncodeOwnershipRecord(next)
	if err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(anchor, "foreign")
	write(foreign, good)
	foreignInfo, _ := os.Stat(foreign)
	noTemps := func(path string) bool {
		names, e := os.ReadDir(path)
		if e != nil {
			return false
		}
		for _, entry := range names {
			if strings.HasPrefix(entry.Name(), ".ownership-") {
				return false
			}
		}
		return true
	}
	s, path := newLease("bounded")
	lockInfo, _ := os.Stat(filepath.Join(path, stateWriterName))
	invoked := false
	wrap := func(tx ownershipPublication) ownershipPublication { invoked = true; return tx }
	size := uint64(len(good))
	err = s.publishOwnershipBounded(p, 0, ownershipPublicationLimits{1, size - 1}, wrap)
	check("create_byte_denial", errors.Is(err, ErrLimit) && !invoked && noTemps(path))
	check("exact_create", s.publishOwnershipBounded(p, 0, ownershipPublicationLimits{1, size}, nil) == nil)
	record := filepath.Join(path, "ownership-100.json")
	oldInfo, _ := os.Stat(record)
	check("count_denial", errors.Is(s.publishOwnershipBounded(q, 0, ownershipPublicationLimits{1, maxOwnershipCatalogBytes}, wrap), ErrLimit) && !invoked)
	reserve := size + uint64(len(nextBytes))
	check("replacement_reservation", errors.Is(s.publishOwnershipBounded(next, 1, ownershipPublicationLimits{1, reserve - 1}, wrap), ErrLimit) && !invoked)
	b, e := os.ReadFile(record)
	info, statErr := os.Stat(record)
	check("denial_preservation", e == nil && statErr == nil && bytes.Equal(b, good) && os.SameFile(oldInfo, info) && noTemps(path))
	check("exact_replace", s.publishOwnershipBounded(next, 1, ownershipPublicationLimits{1, reserve}, nil) == nil)
	read, e := s.ReadOwnership("100")
	check("replacement_readback", e == nil && read.Revision == next.Revision)
	check("stale_denial", errors.Is(s.publishOwnershipBounded(next, 1, ownershipPublicationLimits{1, reserve}, nil), ErrDrift))
	check("invalid_limits", errors.Is(s.publishOwnershipBounded(q, 0, ownershipPublicationLimits{0, reserve}, wrap), ErrLimit) && !invoked)
	currentLock, e := os.Stat(filepath.Join(path, stateWriterName))
	check("stable_lock", e == nil && os.SameFile(lockInfo, currentLock))

	// Unsafe unrelated state must deny publication before even creating our temp.
	for _, kind := range []string{"unknown", "damage", "newer", "wrong_pool", "mode", "symlink", "hardlink", "fifo", "directory"} {
		fixture, folder := newLease("preflight-" + kind)
		if fixture.PublishOwnership(p, 0) != nil {
			t.Fatal("baseline unavailable")
		}
		control := filepath.Join(folder, "ownership-200.json")
		data := qbytes
		switch kind {
		case "unknown":
			control = filepath.Join(folder, "job-1.json")
			write(control, data)
		case "symlink":
			if os.Symlink(foreign, control) != nil {
				t.Fatal("symlink unavailable")
			}
		case "hardlink":
			data = good
			if os.Link(foreign, control) != nil {
				t.Fatal("hardlink unavailable")
			}
		case "fifo":
			if unix.Mkfifo(control, 0600) != nil {
				t.Fatal("FIFO unavailable")
			}
		case "directory":
			if os.Mkdir(control, 0700) != nil {
				t.Fatal("directory unavailable")
			}
		default:
			if kind == "damage" {
				data = append(append([]byte(nil), data...), 'x')
			}
			if kind == "newer" {
				data = bytes.Replace(data, []byte(`"version":1`), []byte(`"version":2`), 1)
			}
			if kind == "wrong_pool" {
				data = good
			}
			write(control, data)
			if kind == "mode" && os.Chmod(control, 0644) != nil {
				t.Fatal("mode unavailable")
			}
		}
		before, e := os.Lstat(control)
		if e != nil {
			t.Fatal(e)
		}
		invoked = false
		err = fixture.publishOwnership(next, 1, wrap)
		after, e := os.Lstat(control)
		unchanged := e == nil && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size()
		if before.Mode().IsRegular() {
			b, e = os.ReadFile(control)
			unchanged = unchanged && e == nil && bytes.Equal(b, data)
		}
		b, e = os.ReadFile(filepath.Join(folder, "ownership-100.json"))
		check(kind+"_denial", err != nil && !invoked && unchanged && e == nil && bytes.Equal(b, good) && noTemps(folder))
	}
	orphanLease, folder := newLease("orphan")
	orphan := filepath.Join(folder, ".ownership-"+strings.Repeat("a", 32)+".tmp")
	write(orphan, []byte("unresolved owned intent"))
	check("orphan_denial", errors.Is(orphanLease.PublishOwnership(p, 0), ErrStateUncertain))
	b, e = os.ReadFile(orphan)
	check("orphan_preservation", e == nil && string(b) == "unresolved owned intent")
	if os.Remove(orphan) != nil {
		t.Fatal("owned orphan cleanup failed")
	}
	check("orphan_latch", errors.Is(orphanLease.PublishOwnership(p, 0), ErrStateUncertain))

	// Add real namespace entries after file sync, before final verification. Only
	// the publisher's own temporary can be excluded and removed on denial.
	for _, kind := range []string{"unknown", "orphan", "count", "bytes"} {
		fixture, folder := newLease("late-" + kind)
		if fixture.PublishOwnership(p, 0) != nil {
			t.Fatal("baseline unavailable")
		}
		limits := ownershipPublicationLimits{maxRecords, maxOwnershipCatalogBytes}
		control := filepath.Join(folder, "ownership-300.json")
		r := p
		r.PoolGUID = "300"
		data, e := EncodeOwnershipRecord(r)
		if e != nil {
			t.Fatal(e)
		}
		want := ErrLimit
		switch kind {
		case "unknown":
			control = filepath.Join(folder, "unknown")
			want = ErrStateRecord
		case "orphan":
			control = filepath.Join(folder, ".ownership-"+strings.Repeat("b", 32)+".tmp")
			want = ErrStateUncertain
		case "count":
			limits.records = 2
		case "bytes":
			limits.bytes = size + uint64(len(qbytes))
		}
		err = fixture.publishOwnershipBounded(q, 0, limits, func(tx ownershipPublication) ownershipPublication {
			verify := tx.verify
			tx.verify = func() error { write(control, data); return verify() }
			return tx
		})
		b, e = os.ReadFile(control)
		_, missing := os.Stat(filepath.Join(folder, "ownership-200.json"))
		entries, scanErr := os.ReadDir(folder)
		check("late_"+kind+"_denial", errors.Is(err, want) && e == nil && bytes.Equal(b, data) && os.IsNotExist(missing) && scanErr == nil && len(entries) == 3)
		read, e = fixture.ReadOwnership("100")
		if e != nil || read.Revision != 1 {
			t.Fatal("prior record unavailable")
		}
		if kind == "orphan" {
			if os.Remove(control) != nil {
				t.Fatal("owned control cleanup failed")
			}
			check("late_orphan_latch", errors.Is(fixture.PublishOwnership(q, 0), ErrStateUncertain))
		}
	}
	race, racePath := newLease("concurrent")
	start := make(chan struct{})
	errorsOut := make(chan error, 2)
	for _, policy := range []Policy{p, q} {
		go func() {
			<-start
			errorsOut <- race.publishOwnershipBounded(policy, 0, ownershipPublicationLimits{1, maxOwnershipCatalogBytes}, nil)
		}()
	}
	close(start)
	a, bErr := <-errorsOut, <-errorsOut
	check("serialized_capacity", a == nil && errors.Is(bErr, ErrLimit) || bErr == nil && errors.Is(a, ErrLimit))
	catalog, e := race.InspectOwnership()
	check("serialized_catalog", e == nil && len(catalog.Records) == 1 && noTemps(racePath))
	b, e = os.ReadFile(foreign)
	info, statErr = os.Stat(foreign)
	check("foreign_preservation", e == nil && statErr == nil && os.SameFile(foreignInfo, info) && bytes.Equal(b, good))
	for _, lease := range leases {
		if lease.Close() != nil {
			t.Fatal("close failed")
		}
	}
	check("lease_close", true)
	check("fixture_teardown", os.RemoveAll(anchor) == nil)
	t.Log("OWNERSHIP_CAPACITY_RESULT " + strings.Join(results, " "))
}
