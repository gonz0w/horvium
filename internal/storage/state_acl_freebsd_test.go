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
	"reflect"
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
					nonownerWrite = nonownerWrite || kind == 4 && entry.Tag != 1 && entry.Type == 0x100 && entry.Perm&uint32(0x10|0x20|0x80|0x100|0x400|0x800|0x2000|0x4000) != 0
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
	property := nativePropertyJobFixture(t)
	if p, duplicate, err := s.PublishPropertySubmission(property); err != ErrStateDirectory || duplicate || !reflect.DeepEqual(p, PropertyJob{}) {
		t.Fatal("submission publication bypassed kernel UID")
	}
	if p, err := s.ReadPropertyJob(property.job.ID); err != ErrStateDirectory || !reflect.DeepEqual(p, PropertyJob{}) {
		t.Fatal("property read bypassed kernel UID")
	}
	if err := s.PublishPropertyJob(property, 0); err != ErrStateDirectory {
		t.Fatal("property publication bypassed kernel UID")
	}
	s.propertyOrigin = s
	if c, digest, e := s.inspectExecutionCensus(executionJournalFixture().Binding, defaultStateCatalogLimits, nil); e != ErrStateDirectory || digest != "" || !reflect.DeepEqual(c, StateCatalog{}) {
		t.Fatal("execution census bypassed kernel UID")
	}
	job := jobFixture()
	if j, err := s.ReadJob(job.ID); err != ErrStateDirectory || j.ID != "" {
		t.Fatal("ACL grant bypassed job reader kernel UID guard")
	}
	if v, err := s.InspectJobReservations(); err != ErrStateDirectory || !reflect.DeepEqual(v, JobReservations{}) {
		t.Fatal("reservation inspection bypassed kernel UID")
	}
	if c, err := s.InspectState(); err != ErrStateDirectory || !reflect.DeepEqual(c, StateCatalog{}) {
		t.Fatal("ACL grant bypassed catalog kernel UID guard")
	}
	if err := s.PublishJob(job, 0); err != ErrStateDirectory {
		t.Fatal("job publication bypassed kernel UID guard")
	}
	if err := s.PublishOwnership(Policy{}, 0); err != ErrStateDirectory {
		t.Fatal("ownership publication bypassed kernel UID guard")
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
	jobFD, err := unix.Openat(3, "job-"+job.ID+".json", unix.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal("native job grant was not effective")
	}
	jobFile := os.NewFile(uintptr(jobFD), "acl-job-record")
	defer jobFile.Close()
	if unix.Fstat(jobFD, &st) != nil || st.Uid != 0 || st.Mode&07777 != 0600 {
		t.Fatal("private job mode/UID not retained")
	}
	jobBytes, err := EncodeJobRecord(job)
	data, readErr := io.ReadAll(io.LimitReader(jobFile, 65537))
	if err != nil || readErr != nil || !bytes.Equal(data, jobBytes) {
		t.Fatal("kernel job grant did not read fixture bytes")
	}
	propertyFD, err := unix.Openat(3, "job-"+property.job.ID+".json", unix.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal("native property grant unavailable")
	}
	propertyFile := os.NewFile(uintptr(propertyFD), "acl-property-record")
	defer propertyFile.Close()
	if unix.Fstat(propertyFD, &st) != nil || st.Uid != 0 || st.Mode&07777 != 0600 {
		t.Fatal("property private mode/UID not retained")
	}
	want, err := EncodePropertyJob(property)
	data, readErr = io.ReadAll(io.LimitReader(propertyFile, 65537))
	if err != nil || readErr != nil || !bytes.Equal(data, want) {
		t.Fatal("kernel property grant did not read fixture bytes")
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
			run("/sbin/newfs", "-u", "/dev/"+device)
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
			job := jobFixture()
			jobBytes, err := EncodeJobRecord(job)
			jobPath := filepath.Join(statePath, "job-"+job.ID+".json")
			if err != nil || os.WriteFile(jobPath, jobBytes, 0600) != nil {
				t.Fatal("owned job record unavailable")
			}
			jobRecord, err := os.OpenFile(jobPath, os.O_RDWR, 0)
			if err != nil {
				t.Fatal("owned job descriptor unavailable")
			}
			defer jobRecord.Close()

			property := nativePropertyJobFixture(t)
			propertyBytes, err := EncodePropertyJob(property)
			propertyPath := filepath.Join(statePath, "job-"+property.job.ID+".json")
			if err != nil || os.WriteFile(propertyPath, propertyBytes, 0600) != nil {
				t.Fatal("owned property record unavailable")
			}
			propertyRecord, err := os.OpenFile(propertyPath, os.O_RDWR, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer propertyRecord.Close()
			propertyFD := int(propertyRecord.Fd())
			jobFD := int(jobRecord.Fd())
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
			jobACL := original(jobFD)
			propertyACL := original(propertyFD)
			propertyNext, err := TransitionPropertyJob(property, JobApplying, property.job.History[0].At+1, "launch-intent", "")
			if err != nil {
				t.Fatal(err)
			}
			validProperty := func() bool {
				p, e := s.ReadPropertyJob(property.job.ID)
				return e == nil && reflect.DeepEqual(p, property)
			}
			deniedPropertyRead := func() bool {
				p, e := s.ReadPropertyJob(property.job.ID)
				return e == ErrStateDirectory && reflect.DeepEqual(p, PropertyJob{})
			}
			deniedPropertyPublish := func() bool { return s.PublishPropertyJob(propertyNext, 1) == ErrStateDirectory }
			check(prefix+"property_baseline", validProperty())
			validJob := func() bool { j, e := s.ReadJob(job.ID); return e == nil && j.ID == job.ID }
			if !validJob() {
				t.Fatal("baseline job ACL read refused")
			}
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
			deniedJob := func() bool { j, e := s.ReadJob(job.ID); return e == ErrStateDirectory && j.ID == "" }
			nextJob, err := TransitionJob(job, JobApplying, 1001, "launch-intent", "")
			if err != nil {
				t.Fatal(err)
			}
			deniedPublication := func() bool { return s.PublishJob(nextJob, 1) == ErrStateDirectory }
			validReservations := func() bool {
				v, e := s.InspectJobReservations()
				return e == nil && len(v.Pools()) == 1 && len(v.Pools()[0].Jobs) == 2
			}
			deniedReservations := func() bool {
				v, e := s.InspectJobReservations()
				return e == ErrStateDirectory && reflect.DeepEqual(v, JobReservations{})
			}
			check(prefix+"reservation_baseline", validReservations())
			validSubmission := func() bool {
				p, duplicate, err := s.PublishPropertySubmission(property)
				return err == nil && duplicate && reflect.DeepEqual(p, property)
			}
			deniedSubmission := func() bool {
				p, duplicate, err := s.PublishPropertySubmission(property)
				return err == ErrStateDirectory && !duplicate && reflect.DeepEqual(p, PropertyJob{})
			}
			check(prefix+"submission_baseline", validSubmission())
			deniedCatalog := func() bool {
				c, e := s.InspectState()
				return e == ErrStateDirectory && reflect.DeepEqual(c, StateCatalog{})
			}
			deniedExecutionCensus := func() bool {
				c, digest, e := s.inspectExecutionCensus(executionJournalFixture().Binding, defaultStateCatalogLimits, nil)
				return e == ErrStateDirectory && digest == "" && reflect.DeepEqual(c, StateCatalog{})
			}
			preserved := func(fd int, mode uint16, want stateACL) bool {
				var st, named unix.Stat_t
				a, e := readStateACL(fd, model.kind)
				path := map[int]string{dirFD: statePath, lockFD: filepath.Join(statePath, stateWriterName), recordFD: recordPath, jobFD: jobPath, propertyFD: propertyPath}[fd]
				return unix.Fstat(fd, &st) == nil && unix.Lstat(path, &named) == nil && named.Dev == st.Dev && named.Ino == st.Ino && st.Uid == 0 && st.Mode&07777 == mode && e == nil && a == want
			}
			bad := namedFixtureACL(dirACL, model.kind, 0)
			setFixtureACL(t, dirFD, model.kind, bad)
			check(prefix+"directory_denial", deniedOpen(statePath))
			check(prefix+"directory_read_denial", deniedRead())
			check(prefix+"job_directory_denial", deniedJob())
			check(prefix+"catalog_directory_denial", deniedCatalog())
			check(prefix+"execution_census_directory_denial", deniedExecutionCensus())
			check(prefix+"reservation_directory_denial", deniedReservations())
			check(prefix+"submission_directory_denial", deniedSubmission())
			check(prefix+"job_publish_directory_denial", deniedPublication())
			check(prefix+"property_read_directory_denial", deniedPropertyRead())
			check(prefix+"property_publish_directory_denial", deniedPropertyPublish())
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
			check(prefix+"job_lock_denial", deniedJob())
			check(prefix+"catalog_lock_denial", deniedCatalog())
			check(prefix+"execution_census_lock_denial", deniedExecutionCensus())
			check(prefix+"reservation_lock_denial", deniedReservations())
			check(prefix+"submission_lock_denial", deniedSubmission())
			check(prefix+"job_publish_lock_denial", deniedPublication())
			check(prefix+"property_read_lock_denial", deniedPropertyRead())
			check(prefix+"property_publish_lock_denial", deniedPropertyPublish())
			check(prefix+"lock_no_repair", preserved(lockFD, 0600, bad))
			setFixtureACL(t, lockFD, model.kind, lockACL)
			bad = namedFixtureACL(recordACL, model.kind, 0)
			setFixtureACL(t, recordFD, model.kind, bad)
			check(prefix+"record_denial", deniedRead())
			check(prefix+"catalog_ownership_denial", deniedCatalog())
			check(prefix+"execution_census_ownership_denial", deniedExecutionCensus())
			check(prefix+"reservation_ownership_denial", deniedReservations())
			check(prefix+"submission_ownership_denial", deniedSubmission())
			check(prefix+"job_publish_ownership_denial", deniedPublication())
			check(prefix+"property_publish_ownership_denial", deniedPropertyPublish())
			data, e := os.ReadFile(recordPath)
			check(prefix+"record_no_repair", preserved(recordFD, 0600, bad) && e == nil && bytes.Equal(data, ownershipGolden()))
			setFixtureACL(t, recordFD, model.kind, recordACL)
			bad = namedFixtureACL(jobACL, model.kind, 0)
			setFixtureACL(t, jobFD, model.kind, bad)
			check(prefix+"job_record_denial", deniedJob())
			check(prefix+"catalog_job_denial", deniedCatalog())
			check(prefix+"execution_census_job_denial", deniedExecutionCensus())
			check(prefix+"reservation_job_denial", deniedReservations())
			check(prefix+"submission_job_denial", deniedSubmission())
			check(prefix+"job_publish_job_denial", deniedPublication())
			check(prefix+"property_publish_job_denial", deniedPropertyPublish())
			data, e = os.ReadFile(jobPath)
			check(prefix+"job_record_no_repair", preserved(jobFD, 0600, bad) && e == nil && bytes.Equal(data, jobBytes))
			setFixtureACL(t, jobFD, model.kind, jobACL)
			bad = namedFixtureACL(propertyACL, model.kind, 0)
			setFixtureACL(t, propertyFD, model.kind, bad)
			check(prefix+"property_read_record_denial", deniedPropertyRead())
			check(prefix+"execution_census_property_denial", deniedExecutionCensus())
			check(prefix+"reservation_property_denial", deniedReservations())
			check(prefix+"submission_property_denial", deniedSubmission())
			check(prefix+"property_publish_record_denial", deniedPropertyPublish())
			data, e = os.ReadFile(propertyPath)
			check(prefix+"property_record_preserved", preserved(propertyFD, 0600, bad) && e == nil && bytes.Equal(data, propertyBytes))
			setFixtureACL(t, propertyFD, model.kind, propertyACL)

			if model.kind == 2 {
				setFixtureACL(t, dirFD, 3, testModeACL(2, 0700))
			} else {
				bad = namedFixtureACL(dirACL, 4, 0)
				bad.Entries[0].Flags = 3
				setFixtureACL(t, dirFD, 4, bad)
			}
			check(prefix+"inheritance_denial", deniedOpen(statePath))
			check(prefix+"inheritance_read_denial", deniedRead())
			check(prefix+"catalog_inheritance_denial", deniedCatalog())
			check(prefix+"execution_census_inheritance_denial", deniedExecutionCensus())
			check(prefix+"reservation_inheritance_denial", deniedReservations())
			check(prefix+"submission_inheritance_denial", deniedSubmission())
			check(prefix+"job_publish_inheritance_denial", deniedPublication())
			check(prefix+"property_read_inheritance_denial", deniedPropertyRead())
			check(prefix+"property_publish_inheritance_denial", deniedPropertyPublish())
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
				setFixtureACL(t, jobFD, 4, namedFixtureACL(jobACL, 4, 8))
				setFixtureACL(t, propertyFD, 4, namedFixtureACL(propertyACL, 4, 8))
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
				check("job_kernel_uid_denial", childErr == nil)
				check("catalog_kernel_uid_denial", childErr == nil)
				check("execution_census_kernel_uid_denial", childErr == nil)
				check("reservation_kernel_uid_denial", childErr == nil)
				check("submission_kernel_uid_denial", childErr == nil)
				check("job_publish_kernel_uid_denial", childErr == nil)
				check("property_kernel_grant", childErr == nil)
				check("property_read_kernel_uid_denial", childErr == nil)
				check("property_publish_kernel_uid_denial", childErr == nil)
				if os.RemoveAll(processPath) != nil {
					t.Fatal("process fixture removal failed")
				}
				_, e = os.Lstat(processPath)
				check("process_teardown", os.IsNotExist(e))
				setFixtureACL(t, dirFD, 4, dirACL)
				setFixtureACL(t, recordFD, 4, recordACL)
				setFixtureACL(t, jobFD, 4, jobACL)
				setFixtureACL(t, propertyFD, 4, propertyACL)
			}
			check(prefix+"restored_read", valid())
			check(prefix+"reservation_restored", validReservations())
			check(prefix+"submission_restored", validSubmission())
			check(prefix+"job_restored_read", validJob())
			catalog, e := s.InspectState()
			check(prefix+"catalog_restored", e == nil && len(catalog.Ownership) == 1 && len(catalog.Jobs) == 2 && catalog.Bytes == uint64(len(ownershipGolden())+len(jobBytes)+len(propertyBytes)))
			census, digest, e := s.inspectExecutionCensus(executionJournalFixture().Binding, defaultStateCatalogLimits, nil)
			check(prefix+"execution_census_restored", e == nil && validJobDigest(digest) && reflect.DeepEqual(census, catalog))
			check(prefix+"job_publish_restored", s.PublishJob(nextJob, 1) == nil)
			check(prefix+"property_read_restored", validProperty())
			check(prefix+"property_publish_restored", s.PublishPropertyJob(propertyNext, 1) == nil)
			check(prefix+"lease_close", s.Close() == nil)
			if record.Close() != nil || jobRecord.Close() != nil || propertyRecord.Close() != nil || pf.Close() != nil {
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
