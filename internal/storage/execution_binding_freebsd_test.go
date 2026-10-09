// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestNativeExecutionBindings(t *testing.T) {
	if os.Getenv("HORVIUM_BINDING_CHILD") == "1" {
		s := &StateDirectory{directory: os.NewFile(3, "inherited-directory"), writer: os.NewFile(4, "inherited-lock")}
		s.propertyOrigin = s
		defer s.Close()
		got, e := s.inspectExecutionBindingObjects(nil)
		if os.Geteuid() == 0 || !errors.Is(e, ErrStateDirectory) || got != (executionBindingObjects{}) {
			t.Fatal("inherited root descriptors bypassed actual UID", e)
		}
		t.Log("inherited-nonroot-denied")
		return
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	anchor, e := nativeStateAnchor(t, "horvium-binding-")
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(anchor, "ledger")
	if os.Mkdir(path, 0700) != nil {
		t.Fatal("owned directory")
	}
	s, e := OpenStateDirectory(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("native binding assertion", name)
		}
		results = append(results, name+"=pass")
	}
	deny := func(lease *StateDirectory, drift func()) error {
		t.Helper()
		got, e := lease.inspectExecutionBindingObjects(drift)
		if e == nil || got != (executionBindingObjects{}) {
			t.Fatal("unsafe or partial object observation", e)
		}
		return e
	}
	first, e := s.inspectExecutionBindingObjects(nil)
	check("native_objects", e == nil && first.directory.valid() && first.lock.valid() && first.directory.inode != first.lock.inode)
	var dst, wst unix.Stat_t
	check("exact_descriptor_identity", unix.Fstat(int(s.directory.Fd()), &dst) == nil && unix.Fstat(int(s.writer.Fd()), &wst) == nil && first.directory == executionStatIdentity(dst) && first.lock == executionStatIdentity(wst))
	check("live_mount_observed", first.mount.kind == "ufs" || first.mount.kind == "zfs")
	again, e := s.inspectExecutionBindingObjects(nil)
	check("repeat", e == nil && again == first)
	policy, e := DecodeOwnershipRecord(ownershipGolden())
	check("actual_publication", e == nil && s.PublishOwnership(policy, 0) == nil)
	again, e = s.inspectExecutionBindingObjects(nil)
	check("publication_identity_stable", e == nil && again == first)
	s.mu.Lock()
	s.publicationBlocked = true
	s.mu.Unlock()
	again, e = s.inspectExecutionBindingObjects(nil)
	check("uncertain_observation", e == nil && again == first && s.publicationBlocked)
	check("uncertainty_not_cleared", errors.Is(s.PublishOwnership(policy, 0), ErrStateUncertain))
	copyLease := &StateDirectory{directory: s.directory, writer: s.writer, propertyOrigin: s}
	check("copied_denial", errors.Is(deny(copyLease, nil), ErrDenied))
	check("nil_denial", errors.Is(deny(nil, nil), ErrStateDirectory))
	check("inert_denial", errors.Is(deny(&StateDirectory{}, nil), ErrDenied))
	lockPath := filepath.Join(path, stateWriterName)
	check("directory_mode_denial", os.Chmod(path, 0750) == nil && errors.Is(deny(s, nil), ErrStateDirectory))
	if os.Chmod(path, 0700) != nil {
		t.Fatal("restore owned directory mode")
	}
	check("lock_mode_denial", os.Chmod(lockPath, 0640) == nil && errors.Is(deny(s, nil), ErrStateDirectory))
	if os.Chmod(lockPath, 0600) != nil {
		t.Fatal("restore owned lock mode")
	}
	check("lock_nonempty_denial", os.WriteFile(lockPath, []byte("x"), 0600) == nil && errors.Is(deny(s, nil), ErrStateDirectory))
	if os.Truncate(lockPath, 0) != nil {
		t.Fatal("restore owned empty lock")
	}
	check("lock_hardlink_denial", os.Link(lockPath, filepath.Join(path, "alias")) == nil && errors.Is(deny(s, nil), ErrStateDirectory))
	if os.Remove(filepath.Join(path, "alias")) != nil {
		t.Fatal("remove owned link")
	}
	check("same_inode_drift", errors.Is(deny(s, func() {
		if os.Chmod(lockPath, 0640) != nil || os.Chmod(lockPath, 0600) != nil {
			t.Fatal("owned lock drift")
		}
	}), ErrDrift))
	check("namespace_drift", errors.Is(deny(s, func() {
		if os.WriteFile(filepath.Join(path, "changed"), nil, 0600) != nil {
			t.Fatal("owned namespace drift")
		}
	}), ErrDrift))
	if os.Remove(filepath.Join(path, "changed")) != nil {
		t.Fatal("owned namespace removal")
	}
	again, e = s.inspectExecutionBindingObjects(nil)
	check("restored_observation", e == nil && again == first && s.publicationBlocked)
	executable, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cmd := exec.CommandContext(ctx, executable, "-test.v", "-test.run=^TestNativeExecutionBindings$")
	cmd.Env = append(os.Environ(), "HORVIUM_BINDING_CHILD=1")
	cmd.ExtraFiles = []*os.File{s.directory, s.writer}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534}}
	output, e := cmd.CombinedOutput()
	cancel()
	check("inherited_kernel_uid_denial", e == nil && strings.Contains(string(output), "inherited-nonroot-denied"))
	check("child_reaped", cmd.ProcessState != nil && cmd.ProcessState.Success())
	if os.Rename(lockPath, filepath.Join(path, "old-lock")) != nil || os.WriteFile(lockPath, nil, 0600) != nil {
		t.Fatal("owned replacement")
	}
	check("lock_replacement_denial", errors.Is(deny(s, nil), ErrStateDirectory))
	check("close", s.Close() == nil)
	check("closed_denial", errors.Is(deny(s, nil), ErrDenied))
	if os.Remove(lockPath) != nil || os.Rename(filepath.Join(path, "old-lock"), lockPath) != nil {
		t.Fatal("restore owned lock name")
	}
	s, e = openStateDirectory(path, false)
	if e != nil {
		t.Fatal(e)
	}
	again, e = s.inspectExecutionBindingObjects(nil)
	check("reopen_identity_stable", e == nil && again == first)
	check("fixture_teardown", s.Close() == nil && os.RemoveAll(path) == nil)
	t.Log("EXECUTION_BINDING_RESULT " + strings.Join(results, " "))
}

func TestNativeExecutionBindingRemount(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	anchor, e := nativeUFSStateAnchor(t, "horvium-binding-remount-", false)
	if e != nil {
		t.Fatal(e)
	}
	s, e := OpenStateDirectory(anchor)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	before, e := s.inspectExecutionBindingObjects(nil)
	if e != nil || s.Close() != nil {
		t.Fatal("pre-remount native observation", e)
	}
	run := func(path string, args ...string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if out, e := exec.CommandContext(ctx, path, args...).CombinedOutput(); e != nil {
			t.Fatalf("owned remount: %v %s", e, out)
		}
	}
	run("/sbin/umount", before.mount.target)
	run("/sbin/mount", "-t", "ufs", before.mount.source, before.mount.target)
	s, e = openStateDirectory(anchor, false)
	if e != nil {
		t.Fatal(e)
	}
	after, e := s.inspectExecutionBindingObjects(nil)
	if e != nil || before.directory != after.directory || before.lock != after.lock {
		t.Fatal("object identity changed across actual UFS remount", e)
	}
	if s.Close() != nil {
		t.Fatal("remounted lease close")
	}
	t.Log("EXECUTION_BINDING_REMOUNT_RESULT original_objects=pass closed_before_unmount=pass actual_remount=pass identity_preserved=pass close=pass")
}
