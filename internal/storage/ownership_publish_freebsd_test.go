// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestNativeOwnershipPublish(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("owned root fixture required")
	}
	anchor, err := os.MkdirTemp("/root", "horvium-publish-")
	if err != nil {
		t.Fatal("owned anchor unavailable")
	}
	defer os.RemoveAll(anchor)
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("publication assertion", name)
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
		t.Cleanup(func() { _ = s.Close() })
		return s, path
	}
	p, e := DecodeOwnershipRecord(ownershipGolden())
	if e != nil {
		t.Fatal("golden policy unavailable")
	}
	next := p
	next.Revision = 2
	nextBytes, e := EncodeOwnershipRecord(next)
	if e != nil {
		t.Fatal("next policy unavailable")
	}
	s, path := newLease("normal")
	target := filepath.Join(path, "ownership-100.json")
	check("create", s.PublishOwnership(p, 0) == nil)
	b, e := os.ReadFile(target)
	check("canonical_bytes", e == nil && bytes.Equal(b, ownershipGolden()))
	old, e := os.Lstat(target)
	if e != nil {
		t.Fatal("published file unavailable")
	}
	st := old.Sys().(*syscall.Stat_t)
	check("private_file", st.Uid == 0 && st.Mode&07777 == 0600 && st.Nlink == 1 && old.Mode().IsRegular())
	oldLock, e := os.Lstat(filepath.Join(path, stateWriterName))
	if e != nil {
		t.Fatal("lock unavailable")
	}
	check("duplicate_denial", errors.Is(s.PublishOwnership(p, 0), ErrDrift))
	bad := next
	bad.Revision = 3
	check("revision_denial", errors.Is(s.PublishOwnership(bad, 1), ErrDrift) && errors.Is(s.PublishOwnership(next, 9), ErrDrift))
	invalid := p
	invalid.PoolGUID = "../writer.lock"
	check("invalid_denial", s.PublishOwnership(invalid, 0) != nil)
	oversized := p
	oversized.Scopes = make([]Scope, 10001)
	check("capacity_denial", errors.Is(s.PublishOwnership(oversized, 0), ErrLimit))
	held, e := os.Open(target)
	if e != nil {
		t.Fatal("old opened record unavailable")
	}
	check("replace", s.PublishOwnership(next, 1) == nil)
	oldBytes, oldErr := io.ReadAll(held)
	closeErr := held.Close()
	check("old_descriptor_preserved", oldErr == nil && closeErr == nil && bytes.Equal(oldBytes, ownershipGolden()))
	b, e = os.ReadFile(target)
	now, se := os.Lstat(target)
	check("new_inode", e == nil && se == nil && bytes.Equal(b, nextBytes) && now.Sys().(*syscall.Stat_t).Ino != st.Ino)
	got, e := s.ReadOwnership("100")
	check("readback", e == nil && got.Revision == 2 && got.PoolGUID == "100")
	lock, e := os.Lstat(filepath.Join(path, stateWriterName))
	check("stable_lock", e == nil && lock.Sys().(*syscall.Stat_t).Ino == oldLock.Sys().(*syscall.Stat_t).Ino)
	noTemps := func(path string) bool {
		entries, e := os.ReadDir(path)
		if e != nil {
			return false
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".ownership-") {
				return false
			}
		}
		return true
	}
	check("no_temporary", noTemps(path))
	// Unsafe/corrupt existing objects must never be repaired or replaced.
	foreign := filepath.Join(anchor, "foreign")
	if os.WriteFile(foreign, ownershipGolden(), 0600) != nil {
		t.Fatal("foreign fixture unavailable")
	}
	for _, kind := range []string{"mode", "owner", "hardlink", "symlink", "fifo", "directory", "damage", "newer", "wrong_pool"} {
		lease, d := newLease("control-" + kind)
		name := filepath.Join(d, "ownership-100.json")
		switch kind {
		case "symlink":
			err = os.Symlink(foreign, name)
		case "fifo":
			err = unix.Mkfifo(name, 0600)
		case "directory":
			err = os.Mkdir(name, 0700)
		default:
			data := ownershipGolden()
			switch kind {
			case "damage":
				data = []byte("damaged")
			case "newer":
				data = uncheckedOwnership(strings.Replace(ownershipGoldenContent, `"version":1`, `"version":2`, 1))
			case "wrong_pool":
				data = uncheckedOwnership(strings.Replace(ownershipGoldenContent, `"pool_guid":"100"`, `"pool_guid":"999"`, 1))
			}
			err = os.WriteFile(name, data, 0600)
			if err == nil {
				switch kind {
				case "mode":
					err = os.Chmod(name, 0644)
				case "owner":
					err = os.Chown(name, 65534, 65534)
				case "hardlink":
					err = os.Link(name, filepath.Join(d, "linked"))
				}
			}
		}
		if err != nil {
			t.Fatal("owned control unavailable", kind)
		}
		before, e := os.Lstat(name)
		if e != nil {
			t.Fatal("owned metadata unavailable")
		}
		var data []byte
		if before.Mode().IsRegular() {
			data, e = os.ReadFile(name)
			if e != nil {
				t.Fatal("owned bytes unavailable")
			}
		}
		denied := lease.PublishOwnership(next, 1) != nil
		after, e := os.Lstat(name)
		preserved := e == nil && before.Mode() == after.Mode() && before.Size() == after.Size() && before.Sys().(*syscall.Stat_t).Ino == after.Sys().(*syscall.Stat_t).Ino
		if before.Mode().IsRegular() {
			out, e := os.ReadFile(name)
			preserved = preserved && e == nil && bytes.Equal(data, out)
		}
		check(kind+"_denial", denied && preserved && noTemps(d))
		if lease.Close() != nil {
			t.Fatal("control lease close failed")
		}
	}
	// Failures are injected around real native operations only to qualify error
	// accounting/order, not kernel EIO, device honesty or crash durability.
	for _, phase := range []string{"write", "file_sync", "close", "verify", "rename", "directory_sync"} {
		lease, d := newLease("failure-" + phase)
		if lease.PublishOwnership(p, 0) != nil {
			t.Fatal("failure baseline unavailable")
		}
		err = lease.publishOwnership(next, 1, func(tx ownershipPublication) ownershipPublication {
			fail := func() error { return unix.EIO }
			switch phase {
			case "write":
				tx.write = fail
			case "file_sync":
				tx.syncFile = fail
			case "close":
				close := tx.closeFile
				tx.closeFile = func() error {
					if e := close(); e != nil {
						return e
					}
					return unix.EIO
				}
			case "verify":
				tx.verify = fail
			case "rename":
				tx.rename = fail
			case "directory_sync":
				tx.syncDirectory = fail
			}
			return tx
		})
		b, e := os.ReadFile(filepath.Join(d, "ownership-100.json"))
		if phase == "directory_sync" {
			check(phase+"_uncertain", errors.Is(err, ErrStateUncertain) && e == nil && bytes.Equal(b, nextBytes) && noTemps(d))
			third := next
			third.Revision = 3
			check("uncertain_write_blocked", errors.Is(lease.PublishOwnership(third, 2), ErrStateUncertain))
			got, e = lease.ReadOwnership("100")
			check("uncertain_readable", e == nil && got.Revision == 2)
		} else {
			check(phase+"_failure", errors.Is(err, ErrStateDirectory) && e == nil && bytes.Equal(b, ownershipGolden()) && noTemps(d))
		}
		if lease.Close() != nil {
			t.Fatal("failure lease close failed")
		}
	}
	uncertain, ud := newLease("cleanup-failure")
	if uncertain.PublishOwnership(p, 0) != nil {
		t.Fatal("cleanup baseline unavailable")
	}
	err = uncertain.publishOwnership(next, 1, func(tx ownershipPublication) ownershipPublication {
		tx.write = func() error { return unix.EIO }
		cleanup := tx.cleanup
		tx.cleanup = func() error {
			if e := cleanup(); e != nil {
				return e
			}
			return unix.EIO
		}
		return tx
	})
	b, e = os.ReadFile(filepath.Join(ud, "ownership-100.json"))
	check("cleanup_uncertain", errors.Is(err, ErrStateUncertain) && e == nil && bytes.Equal(b, ownershipGolden()) && noTemps(ud))
	check("cleanup_write_blocked", errors.Is(uncertain.PublishOwnership(next, 1), ErrStateUncertain))
	if uncertain.Close() != nil {
		t.Fatal("cleanup fixture close failed")
	}
	// Two cooperating calls using the same expected revision cannot both commit.
	concurrent, cd := newLease("concurrent")
	if concurrent.PublishOwnership(p, 0) != nil {
		t.Fatal("concurrent baseline unavailable")
	}
	start := make(chan struct{})
	outcomes := make(chan error, 2)
	for range 2 {
		go func() { <-start; outcomes <- concurrent.PublishOwnership(next, 1) }()
	}
	close(start)
	first, second := <-outcomes, <-outcomes
	check("serialized_revision", (first == nil && errors.Is(second, ErrDrift) || second == nil && errors.Is(first, ErrDrift)) && noTemps(cd))
	if concurrent.Close() != nil {
		t.Fatal("concurrent lease close failed")
	}
	drift, d := newLease("drift")
	if drift.PublishOwnership(p, 0) != nil {
		t.Fatal("drift baseline unavailable")
	}
	err = drift.publishOwnership(next, 1, func(tx ownershipPublication) ownershipPublication {
		verify := tx.verify
		tx.verify = func() error {
			if os.WriteFile(filepath.Join(d, "ownership-100.json"), []byte("external root change"), 0600) != nil {
				return ErrStateDirectory
			}
			return verify()
		}
		return tx
	})
	b, e = os.ReadFile(filepath.Join(d, "ownership-100.json"))
	check("destination_drift", errors.Is(err, ErrDrift) && e == nil && string(b) == "external root change" && noTemps(d))
	if drift.Close() != nil {
		t.Fatal("drift close failed")
	}
	// Staging must not rebind a changed predecessor, even at the same revision.
	for _, kind := range []string{"inode", "bytes"} {
		lease, folder := newLease("staged-" + kind)
		if lease.PublishOwnership(p, 0) != nil {
			t.Fatal("staged predecessor baseline")
		}
		name := filepath.Join(folder, "ownership-100.json")
		control := ownershipGolden()
		if kind == "bytes" {
			changed, e := DecodeOwnershipRecord(control)
			if e != nil {
				t.Fatal(e)
			}
			changed.Scopes[0].Properties = []Property{Compression}
			control, e = EncodeOwnershipRecord(changed)
			if e != nil {
				t.Fatal(e)
			}
		}
		err = lease.publishOwnership(next, 1, func(tx ownershipPublication) ownershipPublication {
			sync := tx.syncFile
			tx.syncFile = func() error {
				if e := sync(); e != nil {
					return e
				}
				if kind == "inode" {
					if os.Rename(name, name+"-retained") != nil {
						t.Fatal("owned predecessor move")
					}
				}
				if os.WriteFile(name, control, 0600) != nil {
					t.Fatal("owned predecessor change")
				}
				return nil
			}
			return tx
		})
		b, e = os.ReadFile(name)
		check("staged_"+kind+"_denial", errors.Is(err, ErrDrift) && e == nil && bytes.Equal(b, control) && noTemps(folder))
		if kind == "inode" {
			b, e = os.ReadFile(name + "-retained")
			if e != nil || !bytes.Equal(b, ownershipGolden()) {
				t.Fatal("retained original damaged")
			}
		}
		if lease.Close() != nil {
			t.Fatal("staged lease close")
		}
	}
	// Renaming the directory never redirects publication to a replacement path.
	moved := filepath.Join(anchor, "moved")
	if os.Rename(path, moved) != nil || os.Mkdir(path, 0700) != nil {
		t.Fatal("anchor fixture unavailable")
	}
	third := next
	third.Revision = 3
	check("directory_anchor", s.PublishOwnership(third, 2) == nil)
	_, e = os.Lstat(filepath.Join(path, "ownership-100.json"))
	check("replacement_untouched", os.IsNotExist(e))
	if os.Chmod(moved, 0755) != nil {
		t.Fatal("mode control failed")
	}
	third.Revision = 4
	check("directory_drift", s.PublishOwnership(third, 3) != nil && noTemps(moved))
	if os.Chmod(moved, 0700) != nil {
		t.Fatal("mode restore failed")
	}
	lockName := filepath.Join(moved, stateWriterName)
	if os.Chmod(lockName, 0644) != nil {
		t.Fatal("lock control failed")
	}
	check("lock_drift", s.PublishOwnership(third, 3) != nil && noTemps(moved))
	if os.Chmod(lockName, 0600) != nil {
		t.Fatal("lock restore failed")
	}
	check("lease_close", s.Close() == nil)
	check("closed_denial", s.PublishOwnership(third, 3) != nil)
	check("inert_denial", (*StateDirectory)(nil).PublishOwnership(p, 0) != nil && (&StateDirectory{}).PublishOwnership(p, 0) != nil)
	b, e = os.ReadFile(foreign)
	check("foreign_preservation", e == nil && bytes.Equal(b, ownershipGolden()))
	check("fixture_teardown", os.RemoveAll(anchor) == nil)
	_, e = os.Lstat(anchor)
	if !os.IsNotExist(e) {
		t.Fatal("owned fixture remains")
	}
	t.Log("OWNERSHIP_PUBLISH_RESULT " + strings.Join(results, " "))
}

func TestNativeOwnershipPublishFilesystem(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("owned root fixture required")
	}
	anchor, err := os.MkdirTemp("/root", "horvium-publish-fs-")
	if err != nil {
		t.Fatal("owned anchor unavailable")
	}
	mountpoint := filepath.Join(anchor, "mount")
	if os.Mkdir(mountpoint, 0700) != nil {
		t.Fatal("owned mountpoint unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	run := func(command string, args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Stderr = io.Discard
		out, e := cmd.Output()
		if e != nil {
			t.Fatal("owned filesystem command failed", command)
		}
		return out
	}
	device := strings.TrimSpace(string(run("/sbin/mdconfig", "-a", "-t", "malloc", "-s", "32m")))
	if !regexp.MustCompile(`^md[0-9]{1,6}$`).MatchString(device) {
		t.Fatal("owned device identity unavailable")
	}
	attached, mounted := true, false
	cleanup := func() bool {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		if mounted {
			cmd := exec.CommandContext(cctx, "/sbin/umount", mountpoint)
			cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
			if cmd.Run() != nil {
				return false
			}
			mounted = false
		}
		if attached {
			cmd := exec.CommandContext(cctx, "/sbin/mdconfig", "-d", "-u", strings.TrimPrefix(device, "md"))
			cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
			if cmd.Run() != nil {
				return false
			}
			attached = false
		}
		return os.RemoveAll(anchor) == nil
	}
	defer func() {
		if !cleanup() {
			t.Error("owned filesystem cleanup failed")
		}
	}()
	run("/sbin/newfs", "-U", "/dev/"+device)
	run("/sbin/mount", "-t", "ufs", "-o", "acls", "/dev/"+device, mountpoint)
	mounted = true
	if os.Chmod(mountpoint, 0700) != nil {
		t.Fatal("owned mount mode unavailable")
	}
	path := filepath.Join(mountpoint, "state")
	if os.Mkdir(path, 0700) != nil {
		t.Fatal("owned state unavailable")
	}
	lease, e := OpenStateDirectory(path)
	if e != nil {
		t.Fatal("owned UFS lease unavailable", e)
	}
	defer lease.Close()
	p, e := DecodeOwnershipRecord(ownershipGolden())
	if e != nil {
		t.Fatal("policy unavailable")
	}
	next := p
	next.Revision = 2
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("filesystem assertion", name)
		}
		results = append(results, name+"=pass")
	}
	check("ufs_publication", lease.PublishOwnership(p, 0) == nil)
	// Actual unsafe POSIX access ACL preserves mode bits but must block replacement.
	record := filepath.Join(path, "ownership-100.json")
	f, e := os.Open(record)
	if e != nil {
		t.Fatal("owned record unavailable")
	}
	defer f.Close()
	original, e := readStateACL(int(f.Fd()), 2)
	if e != nil {
		t.Fatal("owned POSIX ACL unavailable")
	}
	setFixtureACL(t, int(f.Fd()), 2, namedFixtureACL(original, 2, 4))
	check("acl_denial", lease.PublishOwnership(next, 1) != nil)
	catalog, ce := lease.InspectOwnership()
	check("catalog_acl_denial", ce != nil && catalog.Records == nil && catalog.Bytes == 0)
	current, e := readStateACL(int(f.Fd()), 2)
	b, be := os.ReadFile(record)
	check("acl_preservation", e == nil && current == namedFixtureACL(original, 2, 4) && be == nil && bytes.Equal(b, ownershipGolden()))
	check("catalog_acl_preservation", e == nil && current == namedFixtureACL(original, 2, 4) && be == nil && bytes.Equal(b, ownershipGolden()))
	setFixtureACL(t, int(f.Fd()), 2, original)
	catalog, ce = lease.InspectOwnership()
	check("catalog_acl_restored", ce == nil && len(catalog.Records) == 1)
	if f.Close() != nil {
		t.Fatal("owned ACL restore failed")
	}
	job := catalogJob(1, false)
	jobBytes, _ := EncodeJobRecord(job)
	jobNext, _ := TransitionJob(job, JobApplying, 1001, "launch-intent", "")
	jobPath := filepath.Join(path, "job-"+job.ID+".json")
	check("ufs_job_publication", lease.PublishJob(job, 0) == nil)
	// Fill only this new 32 MiB memory filesystem, observing real ENOSPC.
	filler := filepath.Join(mountpoint, "filler")
	out, e := os.OpenFile(filler, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal("filler unavailable")
	}
	block := make([]byte, 65536)
	full := false
	for n := 0; n < 1024; n++ {
		_, e = out.Write(block)
		if e != nil {
			full = errors.Is(e, syscall.ENOSPC)
			break
		}
	}
	_ = out.Close()
	// Large allocations can return ENOSPC while UFS retains fragments sufficient
	// for a small ownership record. Exhaust those through new tiny owned files,
	// then independently require zero free space before calling the publisher.
	var fragments []string
	for n := 0; n < 256; n++ {
		name := filepath.Join(mountpoint, "fragment-"+strconv.Itoa(n))
		f, openErr := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if openErr != nil {
			if !errors.Is(openErr, syscall.ENOSPC) {
				t.Fatal("fragment fixture unavailable")
			}
			break
		}
		fragments = append(fragments, name)
		_, writeErr := f.Write(block[:512])
		_ = f.Close()
		if writeErr != nil {
			if !errors.Is(writeErr, syscall.ENOSPC) {
				t.Fatal("fragment allocation failed")
			}
			break
		}
	}
	var fullFS unix.Statfs_t
	check("native_enospc", full && unix.Fstatfs(int(lease.directory.Fd()), &fullFS) == nil && fullFS.Bfree == 0)
	jobErr := lease.PublishJob(jobNext, 1)
	jb, je := os.ReadFile(jobPath)
	check("job_full_denial", jobErr != nil && je == nil && bytes.Equal(jb, jobBytes))
	err = lease.PublishOwnership(next, 1)
	b, e = os.ReadFile(record)
	check("full_denial", err != nil && e == nil && bytes.Equal(b, ownershipGolden()))
	entries, e := os.ReadDir(path)
	clean := e == nil
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".ownership-") || strings.HasPrefix(entry.Name(), ".job-") {
			clean = false
		}
	}
	check("full_cleanup", clean)
	for _, name := range fragments {
		if os.Remove(name) != nil {
			t.Fatal("owned fragment cleanup failed")
		}
	}
	if os.Remove(filler) != nil || lease.Close() != nil {
		t.Fatal("full fixture reset failed")
	}
	run("/sbin/mount", "-u", "-o", "ro", mountpoint)
	// Construct only a test handle from real read-only descriptors: the normal
	// opener itself refuses this filesystem, while per-call publication must too.
	dir, e := os.Open(path)
	if e != nil {
		t.Fatal("read-only directory unavailable")
	}
	writer, e := os.Open(filepath.Join(path, stateWriterName))
	if e != nil {
		dir.Close()
		t.Fatal("read-only writer unavailable")
	}
	ro := &StateDirectory{directory: dir, writer: writer}
	defer ro.Close()
	if unix.Flock(int(writer.Fd()), unix.LOCK_EX|unix.LOCK_NB) != nil {
		t.Fatal("read-only test lock unavailable")
	}
	var fs unix.Statfs_t
	check("native_readonly", unix.Fstatfs(int(dir.Fd()), &fs) == nil && fs.Flags&unix.MNT_RDONLY != 0)
	check("readonly_denial", errors.Is(ro.PublishOwnership(next, 1), ErrStateDirectory))
	check("job_readonly_denial", errors.Is(ro.PublishJob(jobNext, 1), ErrStateDirectory))
	jb, je = os.ReadFile(jobPath)
	check("job_readonly_preservation", je == nil && bytes.Equal(jb, jobBytes))
	b, e = os.ReadFile(record)
	check("readonly_preservation", e == nil && bytes.Equal(b, ownershipGolden()))
	if ro.Close() != nil {
		t.Fatal("read-only fixture close failed")
	}
	check("filesystem_teardown", cleanup())
	_, e = os.Lstat(anchor)
	if !os.IsNotExist(e) {
		t.Fatal("owned filesystem remains")
	}
	t.Log("OWNERSHIP_PUBLISH_FS_RESULT " + strings.Join(results, " "))
}
