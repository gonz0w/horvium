// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
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

// This reexec helper uses the real FreeBSD boundary in a separate built process.
// It is not the product helper, a storage executor or a restart recovery claim.
func TestNativeStateDirectoryProcess(t *testing.T) {
	mode := os.Getenv("HORVIUM_STATE_PROCESS")
	if mode == "" {
		t.Skip("requires owned reexec fixture")
	}
	lease, err := OpenStateDirectory(os.Getenv("HORVIUM_STATE_PATH"))
	if mode == "deny" {
		if os.Geteuid() != 65534 {
			t.Fatal("UID was not dropped")
		}
		if lease != nil || !errors.Is(err, ErrStateDirectory) {
			t.Fatal("non-root state access accepted")
		}
		return
	}
	if err != nil {
		t.Fatal("owned state process unavailable", err)
	}
	defer lease.Close()
	fmt.Fprintln(os.Stdout, "STATE_WRITER_READY")
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestNativeStateDirectory(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("requires disposable state QEMU opt-ins")
	}
	if os.Geteuid() != 0 {
		t.Fatal("requires disposable guest root")
	}
	// /root is the trusted root-owned anchor; world-writable /tmp is deliberately
	// outside this directory contract. No operator path is accepted by the fixture.
	anchor, err := os.MkdirTemp("/root", "horvium-state-")
	if err != nil {
		t.Fatal("owned anchor unavailable")
	}
	defer func() {
		if os.RemoveAll(anchor) != nil {
			t.Error("owned state fixture removal failed")
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	results := map[string]bool{}
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("state assertion failed:", name)
		}
		results[name] = true
	}
	makeDir := func(name string) string {
		p := filepath.Join(anchor, name)
		if os.Mkdir(p, 0700) != nil {
			t.Fatal("owned directory create failed")
		}
		return p
	}
	before := []byte("foreign-state-sentinel\n")
	sentinel := filepath.Join(anchor, "foreign")
	if os.WriteFile(sentinel, before, 0600) != nil {
		t.Fatal("owned sentinel unavailable")
	}
	path := makeDir("valid")
	lease, err := OpenStateDirectory(path)
	check("root_directory", err == nil && lease != nil)
	t.Cleanup(func() {
		if lease != nil {
			_ = lease.Close()
		}
	})
	lockPath := filepath.Join(path, "writer.lock")
	info, err := os.Lstat(lockPath)
	check("private_lock", err == nil && info.Mode().IsRegular() && info.Mode().Perm() == 0600 && info.Sys().(*syscall.Stat_t).Uid == 0 && info.Sys().(*syscall.Stat_t).Nlink == 1 && info.Size() == 0)
	original := info.Sys().(*syscall.Stat_t).Ino
	lockFlags, e1 := unix.FcntlInt(lease.writer.Fd(), unix.F_GETFD, 0)
	dirFlags, e2 := unix.FcntlInt(lease.directory.Fd(), unix.F_GETFD, 0)
	check("cloexec", e1 == nil && e2 == nil && lockFlags&unix.FD_CLOEXEC != 0 && dirFlags&unix.FD_CLOEXEC != 0)
	second, err := OpenStateDirectory(path)
	check("second_writer_denial", second == nil && errors.Is(err, ErrStateLocked))
	check("close_release", lease.Close() == nil && lease.Close() == nil)
	lease = nil
	lease, err = OpenStateDirectory(path)
	check("reopen", err == nil && lease != nil)
	info, err = os.Lstat(lockPath)
	check("stable_inode", err == nil && info.Sys().(*syscall.Stat_t).Ino == original)
	// Competing first initializers retain their successful handles until both
	// attempts finish; at most one process-local caller can hold the lease.
	racePath := makeDir("first-writer-race")
	start := make(chan struct{})
	type result struct {
		s   *StateDirectory
		err error
	}
	answers := make(chan result, 2)
	for range 2 {
		go func() { <-start; s, err := OpenStateDirectory(racePath); answers <- result{s, err} }()
	}
	close(start)
	var winner *StateDirectory
	wins, denials := 0, 0
	for range 2 {
		r := <-answers
		if r.s != nil && r.err == nil {
			winner = r.s
			wins++
		} else if r.s == nil && errors.Is(r.err, ErrStateLocked) {
			denials++
		}
	}
	if winner != nil {
		defer winner.Close()
	}
	check("first_writer_race", wins == 1 && denials == 1 && winner.Close() == nil)
	// Independent native directory fsync on the actual system filesystem.
	dir, err := os.Open(path)
	if err != nil {
		t.Fatal("independent directory open failed")
	}
	check("directory_sync", dir.Sync() == nil && dir.Close() == nil)
	check("lease_close", lease.Close() == nil)
	lease = nil
	denied := func(path string) {
		t.Helper()
		s, err := OpenStateDirectory(path)
		if s != nil {
			_ = s.Close()
		}
		if s != nil || !errors.Is(err, ErrStateDirectory) {
			t.Fatal("unsafe state accepted", err)
		}
	}
	for _, bad := range []string{"", "/", "relative", path + "/", path + "/.", path + "/../valid", strings.Repeat("a", 4097)} {
		denied(bad)
	}
	check("path_denial", true)
	missing := filepath.Join(anchor, "missing")
	denied(missing)
	_, err = os.Lstat(missing)
	check("no_directory_creation", os.IsNotExist(err))
	for _, mode := range []os.FileMode{0755, 0770, 0700 | os.ModeSticky, 0700 | os.ModeSetgid} {
		p := makeDir(fmt.Sprintf("mode-%o", mode))
		if os.Chmod(p, mode) != nil {
			t.Fatal("mode control failed")
		}
		denied(p)
	}
	check("directory_mode_denial", true)
	owner := makeDir("other-owner")
	if os.Chown(owner, 65534, 65534) != nil {
		t.Fatal("owner control failed")
	}
	denied(owner)
	check("directory_owner_denial", true)
	unsafe := makeDir("unsafe-parent")
	if os.Chmod(unsafe, 0777) != nil {
		t.Fatal("parent control failed")
	}
	child := filepath.Join(unsafe, "child")
	if os.Mkdir(child, 0700) != nil {
		t.Fatal("child control failed")
	}
	denied(child)
	check("ancestor_denial", true)
	alias := filepath.Join(anchor, "alias")
	if os.Symlink(path, alias) != nil {
		t.Fatal("symlink control failed")
	}
	denied(alias)
	denied(filepath.Join(alias, "child"))
	check("directory_symlink_denial", true)
	// Invalid existing lock objects are refused without replacing or repairing them.
	for _, kind := range []string{"mode", "owner", "nonempty", "hardlink", "symlink", "fifo", "directory"} {
		p := makeDir("lock-" + kind)
		lp := filepath.Join(p, "writer.lock")
		switch kind {
		case "symlink":
			err = os.Symlink(sentinel, lp)
		case "fifo":
			err = unix.Mkfifo(lp, 0600)
		case "directory":
			err = os.Mkdir(lp, 0700)
		default:
			err = os.WriteFile(lp, nil, 0600)
			if err == nil {
				switch kind {
				case "mode":
					err = os.Chmod(lp, 0644)
				case "owner":
					err = os.Chown(lp, 65534, 65534)
				case "nonempty":
					err = os.WriteFile(lp, []byte("occupied"), 0600)
				case "hardlink":
					err = os.Link(lp, filepath.Join(p, "linked"))
				}
			}
		}
		if err != nil {
			t.Fatal("owned lock control unavailable")
		}
		old, _ := os.Lstat(lp)
		denied(p)
		now, _ := os.Lstat(lp)
		check("lock_"+kind+"_denial", old != nil && now != nil && reflect.DeepEqual(old.Sys(), now.Sys()))
	}
	// Only declared local UFS/ZFS system-storage fixtures qualify this boundary.
	mountpoint := makeDir("unsupported-filesystem")
	mounted := false
	unmount := func() bool {
		if !mounted {
			return true
		}
		cctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(cctx, "/sbin/umount", mountpoint)
		cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
		if cmd.Run() != nil {
			return false
		}
		mounted = false
		return true
	}
	defer func() {
		if !unmount() {
			t.Error("owned filesystem mount cleanup failed")
		}
	}()
	mount := exec.CommandContext(ctx, "/sbin/mount", "-t", "tmpfs", "-o", "mode=0700", "tmpfs", mountpoint)
	mount.Stdout, mount.Stderr = io.Discard, io.Discard
	if mount.Run() != nil {
		t.Fatal("owned temporary filesystem unavailable")
	}
	mounted = true
	var fs unix.Statfs_t
	if unix.Statfs(mountpoint, &fs) != nil || unix.ByteSliceToString(fs.Fstypename[:]) != "tmpfs" {
		t.Fatal("independent temporary filesystem identity unavailable")
	}
	mountInfo, err := os.Stat(mountpoint)
	if err != nil || mountInfo.Mode().Perm() != 0700 || mountInfo.Sys().(*syscall.Stat_t).Uid != 0 {
		t.Fatal("unsupported filesystem fixture is not independently safe/private")
	}
	denied(mountpoint)
	_, err = os.Lstat(filepath.Join(mountpoint, "writer.lock"))
	check("unsupported_fs_denial", os.IsNotExist(err))
	check("mount_teardown", unmount())
	// Real kernel UID denial, with supplementary groups cleared before test entry.
	// Publish only the copied test executable in an owned traversable temporary
	// directory so the dropped UID can actually enter the test, independent of
	// Go's private build directory. It contains no state/credentials.
	processDir, err := os.MkdirTemp("/tmp", "horvium-state-process-")
	if err != nil {
		t.Fatal("process fixture unavailable")
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
		_ = src.Close()
		t.Fatal("owned executable destination unavailable")
	}
	_, copyErr := io.Copy(dest, src)
	srcErr, destErr := src.Close(), dest.Close()
	if copyErr != nil || srcErr != nil || destErr != nil {
		t.Fatal("owned executable copy failed")
	}
	cmd := exec.CommandContext(ctx, binary, "-test.run=^TestNativeStateDirectoryProcess$")
	cmd.Env = append(os.Environ(), "HORVIUM_STATE_PROCESS=deny", "HORVIUM_STATE_PATH="+path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534, Groups: []uint32{}}}
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	check("kernel_uid_denial", cmd.Run() == nil)
	// A separate built process acquires its own lease after the parent closes.
	cmd = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNativeStateDirectoryProcess$")
	cmd.Env = append(os.Environ(), "HORVIUM_STATE_PROCESS=hold", "HORVIUM_STATE_PATH="+path)
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal("owned child input failed")
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("owned child output failed")
	}
	cmd.Stderr = io.Discard
	if cmd.Start() != nil {
		t.Fatal("owned child start failed")
	}
	waited := false
	t.Cleanup(func() {
		_ = input.Close()
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	line, err := bufio.NewReader(io.LimitReader(output, 256)).ReadString('\n')
	check("process_lock", err == nil && line == "STATE_WRITER_READY\n")
	second, err = OpenStateDirectory(path)
	check("process_denial", second == nil && errors.Is(err, ErrStateLocked))
	check("process_kill", cmd.Process.Kill() == nil)
	waitErr := cmd.Wait()
	waited = true
	check("process_reaped", waitErr != nil)
	lease, err = OpenStateDirectory(path)
	check("death_release", err == nil && lease != nil)
	info, err = os.Lstat(lockPath)
	check("death_inode_preservation", err == nil && info.Sys().(*syscall.Stat_t).Ino == original)
	check("final_close", lease.Close() == nil)
	lease = nil
	data, err := os.ReadFile(sentinel)
	check("foreign_preservation", err == nil && bytes.Equal(data, before))
	if os.RemoveAll(processDir) != nil {
		t.Fatal("owned process fixture cleanup failed")
	}
	_, err = os.Lstat(processDir)
	if !os.IsNotExist(err) {
		t.Fatal("owned executable directory remains")
	}
	if os.RemoveAll(anchor) != nil {
		t.Fatal("owned state cleanup failed")
	}
	_, err = os.Lstat(anchor)
	check("fixture_teardown", os.IsNotExist(err))
	// Fixed sanitized cases only; no paths, UIDs, inode numbers or state bytes.
	order := []string{"root_directory", "private_lock", "cloexec", "second_writer_denial", "close_release", "reopen", "stable_inode", "first_writer_race", "directory_sync", "lease_close", "path_denial", "no_directory_creation", "directory_mode_denial", "directory_owner_denial", "ancestor_denial", "directory_symlink_denial", "lock_mode_denial", "lock_owner_denial", "lock_nonempty_denial", "lock_hardlink_denial", "lock_symlink_denial", "lock_fifo_denial", "lock_directory_denial", "unsupported_fs_denial", "mount_teardown", "kernel_uid_denial", "process_lock", "process_denial", "process_kill", "process_reaped", "death_release", "death_inode_preservation", "final_close", "foreign_preservation", "fixture_teardown"}
	fields := make([]string, 0, len(order))
	for _, name := range order {
		if !results[name] {
			t.Fatal("missing state assertion")
		}
		fields = append(fields, name+"=pass")
	}
	t.Log("STATE_DIRECTORY_RESULT " + strings.Join(fields, " "))
}
