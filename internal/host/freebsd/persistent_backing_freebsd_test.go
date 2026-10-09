// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func persistentNativeFixture(t *testing.T, kind string) (string, func(string, ...string) string) {
	t.Helper()
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_BACKING_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("owned disposable root backing fixture required")
	}
	root, e := os.MkdirTemp("/root", "horvium-persistent-")
	if e != nil {
		t.Fatal(e)
	}
	run := func(path string, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		out, e := execute(ctx, path, args, 65536)
		if e != nil {
			t.Fatalf("owned fixture command %s: %v", path, e)
		}
		return strings.TrimSpace(string(out.data))
	}
	device := ""
	pool := ""
	mounted := false
	t.Cleanup(func() {
		if mounted {
			if kind == "ufs" {
				run("/sbin/umount", filepath.Join(root, "mount"))
			} else {
				run("/sbin/zpool", "destroy", pool)
			}
		}
		if device != "" {
			run("/sbin/mdconfig", "-d", "-u", strings.TrimPrefix(device, "md"))
		}
		if os.RemoveAll(root) != nil {
			t.Fatal("owned fixture removal")
		}
		if _, e := os.Lstat(root); !os.IsNotExist(e) {
			t.Fatal("owned fixture remains")
		}
		t.Log("PERSISTENT_BACKING_TEARDOWN " + kind + " filesystem=pass device=pass directory=pass")
	})
	device = run("/sbin/mdconfig", "-a", "-t", "swap", "-s", "256m")
	if !strings.HasPrefix(device, "md") || strings.Trim(strings.TrimPrefix(device, "md"), "0123456789") != "" || len(device) < 3 {
		t.Fatal("owned device identity")
	}
	path := filepath.Join(root, "mount")
	if kind == "ufs" {
		if os.Mkdir(path, 0700) != nil {
			t.Fatal("owned mount directory")
		}
		run("/sbin/newfs", "-O", "2", "-m", "0", "-u", "/dev/"+device)
		run("/sbin/mount", "-t", "ufs", "/dev/"+device, path)
		mounted = true
	} else {
		var nonce [8]byte
		if _, e := rand.Read(nonce[:]); e != nil {
			t.Fatal(e)
		}
		pool = "horvium_persistent_" + hex.EncodeToString(nonce[:])
		run("/sbin/zpool", "create", "-f", "-O", "mountpoint="+path, pool, "/dev/"+device)
		mounted = true
		run("/sbin/zfs", "create", pool+"/private")
		path = filepath.Join(path, "private")
	}
	if os.Chmod(path, 0700) != nil {
		t.Fatal("owned filesystem permissions")
	}
	return path, run
}

func TestNativePersistentBackingUFS(t *testing.T) {
	if os.Getenv("HORVIUM_PERSISTENT_CHILD") == "1" {
		if unix.Setgroups([]int{}) != nil || unix.Setgid(65534) != nil || unix.Setuid(65534) != nil || os.Geteuid() != 65534 {
			t.Fatal("credential drop")
		}
		id, e := collectNativePersistentBacking(context.Background(), 3)
		if e == nil || id != (persistentBackingIdentity{}) {
			t.Fatal("inherited descriptor bypassed UID")
		}
		t.Log("inherited-denied")
		return
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_BACKING_VM") != "1" {
		t.Skip("disposable backing VM required")
	}
	path, run := persistentNativeFixture(t, "ufs")
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("persistent UFS assertion", name)
		}
		results = append(results, name+"=pass")
	}
	f, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = f.Close() })
	first, e := collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("native_id", e == nil && first.kind == "ufs" && first.ufs[0] != 0 && first.ufs[1] != 0 && first.pool == 0 && first.dataset == 0)
	var fs unix.Statfs_t
	check("actual_kernel_correspondence", unix.Fstatfs(int(f.Fd()), &fs) == nil && first.ufs == ([2]uint32{uint32(fs.Fsid.Val[0]), uint32(fs.Fsid.Val[1])}))
	again, e := collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("repeat", e == nil && again == first)
	check("record_change", os.WriteFile(filepath.Join(path, "record"), []byte("fixture"), 0600) == nil)
	again, e = collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("record_identity_stable", e == nil && again == first)
	run("/sbin/mount", "-u", "-o", "ro", path)
	id, e := collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("readonly_denial", e != nil && id == (persistentBackingIdentity{}))
	run("/sbin/mount", "-u", "-o", "rw", path)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	id, e = collectNativePersistentBacking(ctx, int(f.Fd()))
	check("cancelled", e != nil && id == (persistentBackingIdentity{}))
	id, e = collectNativePersistentBacking(nil, int(f.Fd()))
	check("nil_context", e != nil && id == (persistentBackingIdentity{}))
	id, e = collectNativePersistentBacking(context.Background(), -1)
	check("invalid_fd", e != nil && id == (persistentBackingIdentity{}))
	check("unsafe_mode", os.Chmod(path, 0755) == nil)
	id, e = collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("mode_denied", e != nil && id == (persistentBackingIdentity{}))
	if os.Chmod(path, 0700) != nil {
		t.Fatal("restore owned mode")
	}
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	childCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	cmd := exec.CommandContext(childCtx, exe, "-test.v", "-test.run=^TestNativePersistentBackingUFS$")
	cmd.Env = append(os.Environ(), "HORVIUM_PERSISTENT_CHILD=1")
	cmd.ExtraFiles = []*os.File{f}
	out, e := cmd.CombinedOutput()
	stop()
	check("inherited_uid_denial", e == nil && strings.Contains(string(out), "inherited-denied"))
	check("child_reaped", cmd.ProcessState != nil && cmd.ProcessState.Success())
	source := unix.ByteSliceToString(fs.Mntfromname[:])
	check("close_before_remount", f.Close() == nil)
	run("/sbin/umount", path)
	run("/sbin/mount", "-t", "ufs", source, path)
	f, e = os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	again, e = collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("remount_identity_stable", e == nil && again == first)
	check("close", f.Close() == nil)
	id, e = collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("closed_denied", e != nil && id == (persistentBackingIdentity{}))
	t.Log("PERSISTENT_BACKING_UFS_RESULT " + strings.Join(results, " "))
}

func TestNativePersistentBackingZFS(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_BACKING_VM") != "1" {
		t.Skip("disposable backing VM required")
	}
	path, run := persistentNativeFixture(t, "zfs")
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("persistent ZFS assertion", name)
		}
		results = append(results, name+"=pass")
	}
	f, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = f.Close() })
	first, e := collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("native_id", e == nil && first.kind == "zfs" && first.pool != 0 && first.dataset != 0 && first.ufs == ([2]uint32{}))
	var fs unix.Statfs_t
	if unix.Fstatfs(int(f.Fd()), &fs) != nil {
		t.Fatal("owned mount")
	}
	source := unix.ByteSliceToString(fs.Mntfromname[:])
	pool := strings.Split(source, "/")[0]
	check("exact_dataset", run("/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", source) == strconv.FormatUint(first.dataset, 10))
	check("exact_pool", run("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool) == strconv.FormatUint(first.pool, 10))
	again, e := collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("repeat", e == nil && again == first)
	check("close_before_rename", f.Close() == nil)
	run("/sbin/zfs", "rename", source, pool+"/renamed")
	source = pool + "/renamed"
	renamedPath := run("/sbin/zfs", "get", "-H", "-o", "value", "mountpoint", source)
	f, e = os.Open(renamedPath)
	if e != nil {
		t.Fatal(e)
	}
	again, e = collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("rename_identity_stable", e == nil && again == first)
	check("close_before_remount", f.Close() == nil)
	run("/sbin/zfs", "unmount", source)
	run("/sbin/zfs", "mount", source)
	newPath := run("/sbin/zfs", "get", "-H", "-o", "value", "mountpoint", source)
	f, e = os.Open(newPath)
	if e != nil {
		t.Fatal(e)
	}
	again, e = collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("remount_identity_stable", e == nil && again == first)
	run("/sbin/zfs", "create", pool+"/other")
	otherPath := run("/sbin/zfs", "get", "-H", "-o", "value", "mountpoint", pool+"/other")
	if os.Chmod(otherPath, 0700) != nil {
		t.Fatal("other owned mount mode")
	}
	other, e := os.Open(otherPath)
	if e != nil {
		t.Fatal(e)
	}
	second, e := collectNativePersistentBacking(context.Background(), int(other.Fd()))
	check("distinct_dataset", e == nil && second.pool == first.pool && second.dataset != first.dataset)
	check("other_close", other.Close() == nil)
	check("close", f.Close() == nil)
	run("/sbin/zfs", "destroy", source)
	run("/sbin/zfs", "create", source)
	newPath = run("/sbin/zfs", "get", "-H", "-o", "value", "mountpoint", source)
	if os.Chmod(newPath, 0700) != nil {
		t.Fatal("replacement mode")
	}
	f, e = os.Open(newPath)
	if e != nil {
		t.Fatal(e)
	}
	replacement, e := collectNativePersistentBacking(context.Background(), int(f.Fd()))
	check("name_reuse_new_identity", e == nil && replacement.pool == first.pool && replacement.dataset != first.dataset)
	check("replacement_close", f.Close() == nil)
	t.Log("PERSISTENT_BACKING_ZFS_RESULT " + strings.Join(results, " "))
}
