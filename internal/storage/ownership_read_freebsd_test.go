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
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestNativeOwnershipReadProcess(t *testing.T) {
	if os.Getenv("HORVIUM_READ_PROCESS") != "deny" {
		t.Skip("requires owned inherited-descriptor fixture")
	}
	if os.Geteuid() != 65534 {
		t.Fatal("kernel UID not dropped")
	}
	groups, err := os.Getgroups()
	if err != nil || os.Getegid() != 65534 {
		t.Fatal("kernel group identity unavailable")
	}
	// FreeBSD getgroups includes the primary group; no other group is allowed.
	for _, gid := range groups {
		if gid != 65534 {
			t.Fatal("supplementary group retained")
		}
	}
	dir, writer := os.NewFile(3, "owned-directory"), os.NewFile(4, "owned-writer")
	defer dir.Close()
	defer writer.Close()
	var dst, wst unix.Stat_t
	if unix.Fstat(3, &dst) != nil || unix.Fstat(4, &wst) != nil || dst.Uid != 0 || dst.Mode&unix.S_IFMT != unix.S_IFDIR || !validStateLock(wst) {
		t.Fatal("root descriptors not inherited")
	}
	s := &StateDirectory{directory: dir, writer: writer}
	if p, err := s.ReadOwnership("100"); !errors.Is(err, ErrStateDirectory) || !reflect.DeepEqual(p, Policy{}) {
		t.Fatal("non-root reader returned policy", err)
	}
}

func TestNativeOwnershipRead(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("requires disposable state QEMU opt-ins")
	}
	if os.Geteuid() != 0 {
		t.Fatal("requires owned guest root")
	}
	anchor, err := os.MkdirTemp("/root", "horvium-read-")
	if err != nil {
		t.Fatal("owned read fixture unavailable")
	}
	defer os.RemoveAll(anchor)
	path := filepath.Join(anchor, "state")
	if os.Mkdir(path, 0700) != nil {
		t.Fatal("owned directory unavailable")
	}
	s, err := OpenStateDirectory(path)
	if err != nil {
		t.Fatal("owned lease unavailable", err)
	}
	defer s.Close()
	results := map[string]bool{}
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("ownership read assertion:", name)
		}
		results[name] = true
	}
	denied := func(guid string) {
		t.Helper()
		p, err := s.ReadOwnership(guid)
		if err == nil || !reflect.DeepEqual(p, Policy{}) {
			t.Fatal("unsafe record returned policy", err)
		}
	}
	name := filepath.Join(path, "ownership-100.json")
	good := ownershipGolden()
	foreign := filepath.Join(anchor, "foreign")
	if os.WriteFile(foreign, good, 0600) != nil {
		t.Fatal("foreign sentinel unavailable")
	}
	denied("100")
	_, err = os.Lstat(name)
	check("missing_denial", os.IsNotExist(err))
	for _, guid := range []string{"", "../writer.lock", "0100", "0", strings.Repeat("9", 30), "100.json"} {
		denied(guid)
	}
	check("selector_denial", true)
	if os.WriteFile(name, good, 0600) != nil {
		t.Fatal("owned record unavailable")
	}
	p, err := s.ReadOwnership("100")
	check("valid_read", err == nil && p.PoolGUID == "100" && len(p.Scopes) == 2)
	b, err := os.ReadFile(name)
	check("bytes_preserved", err == nil && bytes.Equal(b, good))
	old, err := os.Lstat(name)
	if err != nil {
		t.Fatal("record metadata unavailable")
	}
	p.Scopes[0].Properties[0] = Quota
	p, err = s.ReadOwnership("100")
	check("independent_policy", err == nil && p.Scopes[0].Properties[0] == Compression)
	now, err := os.Lstat(name)
	check("inode_preserved", err == nil && old.Sys().(*syscall.Stat_t).Ino == now.Sys().(*syscall.Stat_t).Ino)
	// A real kernel-dropped UID receives live root descriptors; capability
	// possession does not bypass the per-call root check.
	processDir, err := os.MkdirTemp("/tmp", "horvium-read-process-")
	if err != nil {
		t.Fatal("owned process fixture unavailable")
	}
	defer os.RemoveAll(processDir)
	if os.Chmod(processDir, 0755) != nil {
		t.Fatal("process fixture mode unavailable")
	}
	binary := filepath.Join(processDir, "test-bin")
	src, err := os.Open(os.Args[0])
	if err != nil {
		t.Fatal("owned executable source unavailable")
	}
	dest, err := os.OpenFile(binary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		src.Close()
		t.Fatal("owned executable destination unavailable")
	}
	_, copyErr := io.Copy(dest, src)
	e1, e2 := src.Close(), dest.Close()
	if copyErr != nil || e1 != nil || e2 != nil || os.Chmod(binary, 0755) != nil {
		t.Fatal("owned executable copy failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "-test.run=^TestNativeOwnershipReadProcess$")
	cmd.Env = append(os.Environ(), "HORVIUM_READ_PROCESS=deny")
	cmd.ExtraFiles = []*os.File{s.directory, s.writer}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534, Groups: []uint32{}}}
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	check("kernel_uid_denial", cmd.Run() == nil)
	// The copied fixture is not a write path in the reader. Refuse invalid records
	// without repair/truncation, including a valid checksum for another pool GUID.
	for _, kind := range []string{"mode", "owner", "hardlink", "symlink", "fifo", "directory", "empty", "oversize", "damage", "newer", "wrong_pool"} {
		if os.Remove(name) != nil {
			t.Fatal("owned control cleanup failed")
		}
		switch kind {
		case "symlink":
			err = os.Symlink(foreign, name)
		case "fifo":
			err = unix.Mkfifo(name, 0600)
		case "directory":
			err = os.Mkdir(name, 0700)
		default:
			data := good
			switch kind {
			case "empty":
				data = nil
			case "oversize":
				data = bytes.Repeat([]byte(" "), (64<<10)+1)
			case "damage":
				data = []byte(strings.Replace(string(good), `"revision":1`, `"revision":2`, 1))
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
					err = os.Link(name, filepath.Join(path, "linked"))
				}
			}
		}
		if err != nil {
			t.Fatal("owned refusal fixture unavailable", kind)
		}
		before, _ := os.Lstat(name)
		var controlBytes []byte
		if before != nil && before.Mode().IsRegular() {
			controlBytes, err = os.ReadFile(name)
			if err != nil {
				t.Fatal("owned control bytes unavailable")
			}
		}
		denied("100")
		after, _ := os.Lstat(name)
		check(kind+"_denial", before != nil && after != nil && before.Mode() == after.Mode() && before.Size() == after.Size() && before.Sys().(*syscall.Stat_t).Ino == after.Sys().(*syscall.Stat_t).Ino)
		if before.Mode().IsRegular() {
			afterBytes, err := os.ReadFile(name)
			if err != nil || !bytes.Equal(controlBytes, afterBytes) {
				t.Fatal("refusal changed owned control bytes")
			}
		}
		if kind == "hardlink" && os.Remove(filepath.Join(path, "linked")) != nil {
			t.Fatal("owned link cleanup failed")
		}
	}
	check("controls_preserved", true)
	if os.Remove(name) != nil || os.WriteFile(name, good, 0600) != nil {
		t.Fatal("owned record restoration failed")
	}
	// Held descriptors continue to address the original owned directory after a
	// root rename. No path supplied to the reader can select the replacement.
	moved := filepath.Join(anchor, "moved")
	if os.Rename(path, moved) != nil || os.Mkdir(path, 0700) != nil {
		t.Fatal("owned rename control failed")
	}
	if os.WriteFile(filepath.Join(path, "ownership-100.json"), []byte("foreign replacement"), 0600) != nil {
		t.Fatal("replacement fixture unavailable")
	}
	p, err = s.ReadOwnership("100")
	check("directory_anchor", err == nil && p.PoolGUID == "100")
	if os.Chmod(moved, 0755) != nil {
		t.Fatal("directory mode control failed")
	}
	denied("100")
	check("directory_drift_denial", true)
	if os.Chmod(moved, 0700) != nil {
		t.Fatal("directory mode reset failed")
	}
	lock := filepath.Join(moved, "writer.lock")
	if os.Chmod(lock, 0644) != nil {
		t.Fatal("lock mode control failed")
	}
	denied("100")
	check("lock_drift_denial", true)
	if os.Chmod(lock, 0600) != nil {
		t.Fatal("lock mode reset failed")
	}
	if os.Remove(lock) != nil || os.WriteFile(lock, nil, 0600) != nil {
		t.Fatal("lock replacement control failed")
	}
	denied("100")
	check("lock_replacement_denial", true)
	check("lease_close", s.Close() == nil)
	denied("100")
	check("closed_denial", true)
	if p, err := (*StateDirectory)(nil).ReadOwnership("100"); err == nil || !reflect.DeepEqual(p, Policy{}) {
		t.Fatal("nil lease accepted")
	}
	if p, err := (&StateDirectory{}).ReadOwnership("100"); err == nil || !reflect.DeepEqual(p, Policy{}) {
		t.Fatal("zero lease accepted")
	}
	check("inert_denial", true)
	b, err = os.ReadFile(foreign)
	check("foreign_preservation", err == nil && bytes.Equal(b, good))
	if os.RemoveAll(processDir) != nil {
		t.Fatal("owned executable fixture remains")
	}
	_, err = os.Lstat(processDir)
	if !os.IsNotExist(err) {
		t.Fatal("owned process directory remains")
	}
	check("fixture_teardown", os.RemoveAll(anchor) == nil)
	_, err = os.Lstat(anchor)
	if !os.IsNotExist(err) {
		t.Fatal("owned read fixture remains")
	}
	order := []string{"missing_denial", "selector_denial", "valid_read", "bytes_preserved", "independent_policy", "inode_preserved", "kernel_uid_denial", "mode_denial", "owner_denial", "hardlink_denial", "symlink_denial", "fifo_denial", "directory_denial", "empty_denial", "oversize_denial", "damage_denial", "newer_denial", "wrong_pool_denial", "controls_preserved", "directory_anchor", "directory_drift_denial", "lock_drift_denial", "lock_replacement_denial", "lease_close", "closed_denial", "inert_denial", "foreign_preservation", "fixture_teardown"}
	fields := make([]string, 0, len(order))
	for _, name := range order {
		if !results[name] {
			t.Fatal("missing read assertion")
		}
		fields = append(fields, name+"=pass")
	}
	t.Log("OWNERSHIP_READ_RESULT " + strings.Join(fields, " "))
}

func TestOwnershipReadMetadataDrift(t *testing.T) {
	st := unix.Stat_t{Dev: 1, Ino: 2, Uid: 0, Gid: 0, Mode: unix.S_IFREG | 0600, Nlink: 1, Size: 10}
	for _, change := range []func(*unix.Stat_t){func(s *unix.Stat_t) { s.Ino++ }, func(s *unix.Stat_t) { s.Dev++ }, func(s *unix.Stat_t) { s.Mode = unix.S_IFREG | 0644 }, func(s *unix.Stat_t) { s.Uid++ }, func(s *unix.Stat_t) { s.Gid++ }, func(s *unix.Stat_t) { s.Nlink++ }, func(s *unix.Stat_t) { s.Size++ }, func(s *unix.Stat_t) { s.Mtim.Sec++ }, func(s *unix.Stat_t) { s.Ctim.Sec++ }, func(s *unix.Stat_t) { s.Btim.Sec++ }, func(s *unix.Stat_t) { s.Flags++ }, func(s *unix.Stat_t) { s.Gen++ }} {
		after := st
		change(&after)
		if sameOwnershipMetadata(st, after) {
			t.Fatal("metadata drift accepted")
		}
	}
	after := st
	after.Atim.Sec++
	if !sameOwnershipMetadata(st, after) {
		t.Fatal("read atime invalidated metadata")
	}
}
