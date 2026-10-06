// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ACL writes exist only in owned native fixtures; the product adapter is read-only.
func setFixtureACL(t *testing.T, fd, kind int, a stateACL) {
	t.Helper()
	if kind == 3 && a.Count == 0 {
		_, _, errno := unix.Syscall(unix.SYS___ACL_DELETE_FD, uintptr(fd), uintptr(kind), 0)
		if errno != 0 {
			t.Fatal("owned default ACL could not be removed", errno)
		}
		return
	}
	_, _, errno := unix.Syscall(unix.SYS___ACL_SET_FD, uintptr(fd), uintptr(kind), uintptr(unsafe.Pointer(&a)))
	runtime.KeepAlive(&a)
	if errno != 0 {
		t.Fatal("owned ACL control could not be installed", errno)
	}
}

func namedFixtureACL(a stateACL, kind int, perm uint32) stateACL {
	if kind == 2 {
		a.Count = 5
		a.Entries[4] = a.Entries[2]
		a.Entries[2] = a.Entries[1]
		a.Entries[1] = stateACLEntry{Tag: 2, ID: 65534, Perm: perm}
		a.Entries[3] = stateACLEntry{Tag: 16, ID: ^uint32(0)} // masked grant keeps mode 0700/0600
	} else {
		copy(a.Entries[1:], a.Entries[:a.Count])
		a.Count++
		a.Entries[0] = stateACLEntry{Tag: 2, ID: 65534, Perm: perm, Type: 0x100}
	}
	return a
}

// Failure diagnostics classify only this disposable VM's declared fixture path.
// No pathname, principal ID, inode, ACL text or operator inventory is emitted.
func diagnoseFixtureACL(t *testing.T, paths []string) {
	t.Helper()
	for i, path := range paths {
		fd, e := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
		if e != nil {
			t.Logf("ACL diagnostic part=%d open_error", i)
			continue
		}
		p, pe := unix.Fpathconf(fd, 59)
		n, ne := unix.Fpathconf(fd, 64)
		for _, kind := range []int{2, 3, 4} {
			a, err := readStateACL(fd, kind)
			named, flags, id, rights, nonownerWrite, types := false, false, false, false, false, false
			if a.Count <= 254 {
				for _, entry := range a.Entries[:a.Count] {
					named = named || entry.Tag != 1 && entry.Tag != 4 && entry.Tag != 32 && entry.Tag != 64
					flags = flags || entry.Flags != 0
					id = id || entry.ID != ^uint32(0)
					rights = rights || kind == 4 && entry.Perm&^uint32(0xfff9) != 0
					types = types || kind == 4 && entry.Type != 0x100 && entry.Type != 0x200
					nonownerWrite = nonownerWrite || kind == 4 && entry.Tag != 1 && entry.Type == 0x100 && entry.Perm&uint32(0x10|0x20|0x100|0x400|0x800|0x2000|0x4000) != 0
				}
			}
			t.Logf("ACL diagnostic part=%d posix=%d/%v nfs4=%d/%v kind=%d retrieval=%v max=%d count=%d spare_zero=%t named=%t flags=%t special_id=%t unknown_rights=%t unknown_type=%t nonowner_write=%t", i, p, pe, n, ne, kind, err, a.Max, a.Count, a.Spare == [4]int32{}, named, flags, id, rights, types, nonownerWrite)
		}
		_ = unix.Close(fd)
	}
}

func TestNativeStateACLProcess(t *testing.T) {
	if os.Getenv("HORVIUM_ACL_PROCESS") != "grant" {
		t.Skip("owned kernel credential fixture required")
	}
	if os.Geteuid() != 65534 || os.Getegid() != 65534 {
		t.Fatal("kernel credentials not dropped")
	}
	groups, err := os.Getgroups()
	if err != nil {
		t.Fatal("group credentials unavailable")
	}
	for _, group := range groups {
		if group != 65534 {
			t.Fatal("privileged supplementary group retained")
		}
	}
	dir := os.NewFile(3, "acl-directory")
	writer := os.NewFile(4, "acl-writer")
	defer dir.Close()
	defer writer.Close()
	s := &StateDirectory{directory: dir, writer: writer}
	if p, err := s.ReadOwnership("100"); err != ErrStateDirectory || p.PoolGUID != "" {
		t.Fatal("ACL grant bypassed product kernel UID guard")
	}
	var st unix.Stat_t
	if unix.Fstat(3, &st) != nil || st.Uid != 0 || st.Mode&07777 != 0700 {
		t.Fatal("private mode/UID not retained")
	}
	fd, err := unix.Openat(3, "ownership-100.json", unix.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal("native named grant was not effective")
	}
	f := os.NewFile(uintptr(fd), "acl-record")
	defer f.Close()
	if unix.Fstat(fd, &st) != nil || st.Uid != 0 || st.Mode&07777 != 0600 {
		t.Fatal("record mode/UID not retained")
	}
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || !bytes.Equal(data, ownershipGolden()) {
		t.Fatal("kernel ACL grant did not read fixture bytes")
	}
}

func TestNativeStateACL(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("native fixture requires root")
	}
	anchor, err := os.MkdirTemp("/root", "horvium-acl-")
	if err != nil {
		t.Fatal("owned anchor unavailable")
	}
	active := 0
	defer func() {
		if active == 0 {
			_ = os.RemoveAll(anchor)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	run := func(command string, args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Stderr = io.Discard
		out, err := cmd.Output()
		if err != nil {
			t.Fatal("owned fixture command failed", command)
		}
		return out
	}
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("ACL assertion failed", name)
		}
		results = append(results, name+"=pass")
	}
	check("descriptor_denial", !safeStateACL(-1, 0700, true))
	for _, model := range []struct {
		name, option string
		kind         int
	}{{"posix", "acls", 2}, {"nfs4", "nfsv4acls", 4}} {
		func() {
			prefix := model.name + "_"
			mountpoint := filepath.Join(anchor, model.name)
			if os.Mkdir(mountpoint, 0700) != nil {
				t.Fatal("mount fixture unavailable")
			}
			device := strings.TrimSpace(string(run("/sbin/mdconfig", "-a", "-t", "malloc", "-s", "32m")))
			if !regexp.MustCompile(`^md[0-9]{1,6}$`).MatchString(device) {
				t.Fatal("owned memory device identity unavailable")
			}
			mounted, attached := false, true
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
					active--
				}
				if attached {
					cmd := exec.CommandContext(cctx, "/sbin/mdconfig", "-d", "-u", strings.TrimPrefix(device, "md"))
					cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
					if cmd.Run() != nil {
						return false
					}
					attached = false
				}
				return true
			}
			defer func() {
				if !cleanup() {
					t.Error("owned ACL filesystem teardown failed")
				}
			}()
			run("/sbin/newfs", "-U", "/dev/"+device)
			run("/sbin/mount", "-t", "ufs", "-o", model.option, "/dev/"+device, mountpoint)
			mounted = true
			active++
			if os.Chmod(mountpoint, 0700) != nil {
				t.Fatal("private mount mode unavailable")
			}
			var fs unix.Statfs_t
			check(prefix+"filesystem", unix.Statfs(mountpoint, &fs) == nil && unix.ByteSliceToString(fs.Fstypename[:]) == "ufs")
			statePath := filepath.Join(mountpoint, "state")
			if os.Mkdir(statePath, 0700) != nil {
				t.Fatal("owned directory unavailable")
			}
			s, err := OpenStateDirectory(statePath)
			if err != nil {
				diagnoseFixtureACL(t, []string{"/", "/root", anchor, mountpoint, statePath})
				t.Fatal("baseline ACL lease refused", model.name)
			}
			defer s.Close()
			recordPath := filepath.Join(statePath, "ownership-100.json")
			if os.WriteFile(recordPath, ownershipGolden(), 0600) != nil {
				t.Fatal("owned record unavailable")
			}
			record, err := os.OpenFile(recordPath, os.O_RDWR, 0)
			if err != nil {
				t.Fatal("owned record descriptor unavailable")
			}
			defer record.Close()
			dirFD, lockFD, recordFD := int(s.directory.Fd()), int(s.writer.Fd()), int(record.Fd())
			posix, e1 := unix.Fpathconf(dirFD, 59)
			nfs4, e2 := unix.Fpathconf(dirFD, 64)
			check(prefix+"capabilities", e1 == nil && e2 == nil && (model.kind == 2 && posix == 1 && nfs4 == 0 || model.kind == 4 && posix == 0 && nfs4 == 1))
			original := func(fd int) stateACL {
				a, e := readStateACL(fd, model.kind)
				if e != nil {
					t.Fatal("kernel ACL unavailable")
				}
				return a
			}
			dirACL, lockACL, recordACL := original(dirFD), original(lockFD), original(recordFD)
			valid := func() bool { p, e := s.ReadOwnership("100"); return e == nil && p.PoolGUID == "100" }
			check(prefix+"baseline", valid())
			deniedOpen := func(path string) bool {
				handle, e := OpenStateDirectory(path)
				if handle != nil {
					_ = handle.Close()
				}
				return handle == nil && e == ErrStateDirectory
			}
			deniedRead := func() bool { p, e := s.ReadOwnership("100"); return e == ErrStateDirectory && p.PoolGUID == "" }
			preserved := func(fd int, mode uint16, want stateACL) bool {
				var st, named unix.Stat_t
				a, e := readStateACL(fd, model.kind)
				path := map[int]string{dirFD: statePath, lockFD: filepath.Join(statePath, stateWriterName), recordFD: recordPath}[fd]
				return unix.Fstat(fd, &st) == nil && unix.Lstat(path, &named) == nil && named.Dev == st.Dev && named.Ino == st.Ino && st.Uid == 0 && st.Mode&07777 == mode && e == nil && a == want
			}
			bad := namedFixtureACL(dirACL, model.kind, 0)
			setFixtureACL(t, dirFD, model.kind, bad)
			check(prefix+"directory_denial", deniedOpen(statePath))
			check(prefix+"directory_read_denial", deniedRead())
			check(prefix+"directory_no_repair", preserved(dirFD, 0700, bad))
			setFixtureACL(t, dirFD, model.kind, dirACL)
			parent := filepath.Join(mountpoint, "ancestor")
			child := filepath.Join(parent, "state")
			if os.Mkdir(parent, 0700) != nil || os.Mkdir(child, 0700) != nil {
				t.Fatal("ancestor fixture unavailable")
			}
			pf, e := os.Open(parent)
			if e != nil {
				t.Fatal("ancestor descriptor unavailable")
			}
			defer pf.Close()
			pa := original(int(pf.Fd()))
			setFixtureACL(t, int(pf.Fd()), model.kind, namedFixtureACL(pa, model.kind, 0))
			check(prefix+"ancestor_denial", deniedOpen(child))
			_, e = os.Lstat(filepath.Join(child, stateWriterName))
			check(prefix+"ancestor_no_creation", os.IsNotExist(e))
			setFixtureACL(t, int(pf.Fd()), model.kind, pa)
			bad = namedFixtureACL(lockACL, model.kind, 0)
			setFixtureACL(t, lockFD, model.kind, bad)
			check(prefix+"lock_denial", deniedOpen(statePath))
			check(prefix+"lock_read_denial", deniedRead())
			check(prefix+"lock_no_repair", preserved(lockFD, 0600, bad))
			setFixtureACL(t, lockFD, model.kind, lockACL)
			bad = namedFixtureACL(recordACL, model.kind, 0)
			setFixtureACL(t, recordFD, model.kind, bad)
			check(prefix+"record_denial", deniedRead())
			data, e := os.ReadFile(recordPath)
			check(prefix+"record_no_repair", preserved(recordFD, 0600, bad) && e == nil && bytes.Equal(data, ownershipGolden()))
			setFixtureACL(t, recordFD, model.kind, recordACL)
			if model.kind == 2 {
				setFixtureACL(t, dirFD, 3, testModeACL(2, 0700))
			} else {
				bad = namedFixtureACL(dirACL, 4, 0)
				bad.Entries[0].Flags = 3
				setFixtureACL(t, dirFD, 4, bad)
			}
			check(prefix+"inheritance_denial", deniedOpen(statePath))
			check(prefix+"inheritance_read_denial", deniedRead())
			if model.kind == 2 {
				a, e := readStateACL(dirFD, 3)
				check(prefix+"inheritance_no_repair", e == nil && a.Count == 3)
				setFixtureACL(t, dirFD, 3, stateACL{Max: 254})
			} else {
				check(prefix+"inheritance_no_repair", preserved(dirFD, 0700, bad))
				setFixtureACL(t, dirFD, 4, dirACL)
			}
			if model.kind == 4 {
				setFixtureACL(t, dirFD, 4, namedFixtureACL(dirACL, 4, 1))
				setFixtureACL(t, recordFD, 4, namedFixtureACL(recordACL, 4, 8))
				processPath, e := os.MkdirTemp("/tmp", "horvium-acl-process-")
				if e != nil {
					t.Fatal("process fixture unavailable")
				}
				defer os.RemoveAll(processPath)
				if os.Chmod(processPath, 0755) != nil {
					t.Fatal("process fixture inaccessible")
				}
				exe, e := os.Executable()
				if e != nil {
					t.Fatal("test executable unavailable")
				}
				src, e := os.Open(exe)
				if e != nil {
					t.Fatal("test executable inaccessible")
				}
				binary := filepath.Join(processPath, "acl.test")
				dst, e := os.OpenFile(binary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0755)
				if e != nil {
					_ = src.Close()
					t.Fatal("owned child executable unavailable")
				}
				_, copyErr := io.Copy(dst, src)
				closeErr := dst.Close()
				_ = src.Close()
				if copyErr != nil || closeErr != nil || os.Chmod(binary, 0755) != nil {
					t.Fatal("owned child executable copy failed")
				}
				cmd := exec.CommandContext(ctx, binary, "-test.run=^TestNativeStateACLProcess$")
				cmd.Env = append(os.Environ(), "HORVIUM_ACL_PROCESS=grant")
				cmd.ExtraFiles = []*os.File{s.directory, s.writer}
				cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534, Groups: []uint32{}}}
				cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
				childErr := cmd.Run()
				check("kernel_nfs4_grant", childErr == nil)
				check("kernel_uid_denial", childErr == nil)
				if os.RemoveAll(processPath) != nil {
					t.Fatal("process fixture removal failed")
				}
				_, e = os.Lstat(processPath)
				check("process_teardown", os.IsNotExist(e))
				setFixtureACL(t, dirFD, 4, dirACL)
				setFixtureACL(t, recordFD, 4, recordACL)
			}
			check(prefix+"restored_read", valid())
			check(prefix+"lease_close", s.Close() == nil)
			if record.Close() != nil || pf.Close() != nil {
				t.Fatal("owned fixture descriptor close failed")
			}
			check(prefix+"unmount", cleanup() && !mounted)
			check(prefix+"device_teardown", !attached)
		}()
	}
	if os.RemoveAll(anchor) != nil {
		t.Fatal("owned ACL anchor cleanup failed")
	}
	_, err = os.Lstat(anchor)
	check("fixture_teardown", os.IsNotExist(err))
	fmt.Println("STATE_ACL_RESULT " + strings.Join(results, " "))
}
