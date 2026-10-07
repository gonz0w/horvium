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

func TestNativeJobRead(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("owned root required")
	}
	anchor, err := nativeStateAnchor(t, "horvium-job-read-")
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
			t.Fatal("job read assertion", name)
		}
		results = append(results, name+"=pass")
	}
	newLease := func(label string) (*StateDirectory, string) {
		t.Helper()
		path := filepath.Join(anchor, label)
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
			t.Fatal("owned write failed")
		}
	}
	denied := func(s *StateDirectory, id string) error {
		t.Helper()
		got, e := s.ReadJob(id)
		if e == nil || !reflect.DeepEqual(got, Job{}) {
			t.Fatal("unsafe/partial job returned", e)
		}
		return e
	}
	j := jobFixture()
	good, e := EncodeJobRecord(j)
	if e != nil {
		t.Fatal(e)
	}
	foreign := filepath.Join(anchor, "foreign")
	write(foreign, good)
	foreignInfo, _ := os.Stat(foreign)
	s, path := newLease("normal")
	name := filepath.Join(path, "job-"+j.ID+".json")
	denied(s, j.ID)
	_, e = os.Stat(name)
	check("missing_denial", os.IsNotExist(e))
	for _, id := range []string{"", "../writer.lock", strings.Repeat("a", 63), strings.Repeat("A", 64), strings.Repeat("x", 64), j.ID + ".json"} {
		if !errors.Is(denied(s, id), ErrInvalid) {
			t.Fatal("bad selector category")
		}
	}
	check("selector_denial", true)
	write(name, good)
	info, _ := os.Stat(name)
	got, e := s.ReadJob(j.ID)
	check("valid_read", e == nil && reflect.DeepEqual(got, j))
	b, e := os.ReadFile(name)
	check("bytes_preserved", e == nil && bytes.Equal(b, good))
	current, e := os.Stat(name)
	check("inode_preserved", e == nil && os.SameFile(info, current))
	got.History[0].Reason = "changed"
	again, e := s.ReadJob(j.ID)
	check("independent_job", e == nil && again.History[0].Reason == "admitted")
	exact := append(append([]byte(nil), good...), bytes.Repeat([]byte(" "), maxStateRecordBytes-len(good))...)
	write(name, exact)
	got, e = s.ReadJob(j.ID)
	check("exact_size", e == nil && reflect.DeepEqual(got, j))
	write(name, good)
	for _, kind := range []string{"mode", "owner", "symlink", "hardlink", "fifo", "directory", "empty", "oversize", "damage", "newer", "wrong_id", "wrong_kind", "invalid_history"} {
		fixture, folder := newLease("deny-" + kind)
		target := filepath.Join(folder, "job-"+j.ID+".json")
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
			case "wrong_id":
				other := j
				other.ID = strings.Repeat("e", 64)
				data, e = EncodeJobRecord(other)
				if e != nil {
					t.Fatal(e)
				}
			case "wrong_kind":
				data = ownershipGolden()
			case "invalid_history":
				data = uncheckedJob(strings.Replace(jobGoldenContent, `"state":"queued"`, `"state":"succeeded"`, 1))
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
			t.Fatal("control unavailable", kind, e)
		}
		before, e := os.Lstat(target)
		if e != nil {
			t.Fatal(e)
		}
		readErr := denied(fixture, j.ID)
		if kind == "wrong_id" && !errors.Is(readErr, ErrDrift) {
			t.Fatal("ID binding category")
		}
		after, e := os.Lstat(target)
		check(kind+"_denial", e == nil && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size())
		if before.Mode().IsRegular() {
			b, e = os.ReadFile(target)
			if e != nil || !bytes.Equal(b, data) {
				t.Fatal("control bytes changed", kind)
			}
		}
	}
	check("controls_preserved", true)
	moved := path + "-moved"
	if os.Rename(path, moved) != nil || os.Mkdir(path, 0700) != nil {
		t.Fatal("anchor fixture failed")
	}
	write(filepath.Join(path, "job-"+j.ID+".json"), []byte("replacement directory"))
	got, e = s.ReadJob(j.ID)
	check("directory_anchor", e == nil && reflect.DeepEqual(got, j))
	if os.Chmod(moved, 0755) != nil {
		t.Fatal("owned chmod failed")
	}
	check("directory_drift_denial", errors.Is(denied(s, j.ID), ErrStateDirectory))
	if os.Chmod(moved, 0700) != nil {
		t.Fatal("owned restore failed")
	}
	lock := filepath.Join(moved, stateWriterName)
	if os.Chmod(lock, 0644) != nil {
		t.Fatal("lock chmod failed")
	}
	check("lock_drift_denial", errors.Is(denied(s, j.ID), ErrStateDirectory))
	if os.Chmod(lock, 0600) != nil || os.Rename(lock, lock+"-old") != nil {
		t.Fatal("lock fixture failed")
	}
	write(lock, nil)
	check("lock_replacement_denial", errors.Is(denied(s, j.ID), ErrStateDirectory))
	for _, lease := range leases {
		if lease.Close() != nil {
			t.Fatal("lease close failed")
		}
	}
	check("lease_close", true)
	check("closed_denial", errors.Is(denied(s, j.ID), ErrStateDirectory))
	check("inert_denial", errors.Is(denied(&StateDirectory{}, j.ID), ErrStateDirectory) && errors.Is(denied(nil, j.ID), ErrStateDirectory))
	b, e = os.ReadFile(foreign)
	current, err = os.Stat(foreign)
	check("foreign_preservation", e == nil && err == nil && bytes.Equal(b, good) && os.SameFile(foreignInfo, current))
	check("fixture_teardown", os.RemoveAll(anchor) == nil)
	t.Log("JOB_READ_RESULT " + strings.Join(results, " "))
}
