// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Only disposable native tests call this. ZFS uses an owned root directory;
// UFS uses a separately owned no-softupdates memory filesystem. Kernel deferred
// removal changes on a softupdates root must not be mistaken for record drift
// exemptions or hidden by a lucky passing run.
func nativeStateAnchor(t *testing.T, prefix string) (string, error) {
	t.Helper()
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("disposable root state VM required")
	}
	var fs unix.Statfs_t
	if unix.Statfs("/root", &fs) != nil {
		t.Fatal("state fixture filesystem")
	}
	if unix.ByteSliceToString(fs.Fstypename[:]) == "ufs" {
		return nativeUFSStateAnchor(t, prefix, false)
	}
	if unix.ByteSliceToString(fs.Fstypename[:]) != "zfs" {
		t.Fatal("unsupported native fixture root")
	}
	anchor, err := os.MkdirTemp("/root", prefix)
	if err != nil {
		return "", err
	}
	t.Cleanup(func() {
		if os.RemoveAll(anchor) != nil {
			t.Fatal("owned state fixture removal")
		}
		if _, err := os.Lstat(anchor); !os.IsNotExist(err) {
			t.Fatal("owned state fixture remains")
		}
		t.Log("STATE_BACKING_RESULT " + t.Name() + " filesystem=zfs softupdates=off teardown=pass")
	})
	return anchor, nil
}

func nativeUFSStateAnchor(t *testing.T, prefix string, soft bool) (string, error) {
	t.Helper()
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("disposable root state VM required")
	}
	container, err := os.MkdirTemp("/root", prefix+"backing-")
	if err != nil {
		return "", err
	}
	device := ""
	mounted := false
	capabilityVerified := false
	run := func(path string, args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return exec.CommandContext(ctx, path, args...).CombinedOutput()
	}
	t.Cleanup(func() {
		if mounted {
			if _, err := run("/sbin/umount", container); err != nil {
				t.Fatal("owned state backing unmount")
			}
		}
		if device != "" {
			if _, err := run("/sbin/mdconfig", "-d", "-u", strings.TrimPrefix(device, "md")); err != nil {
				t.Fatal("owned state backing detach")
			}
		}
		if os.RemoveAll(container) != nil {
			t.Fatal("owned state backing removal")
		}
		if _, err := os.Lstat(container); !os.IsNotExist(err) {
			t.Fatal("owned state backing remains")
		}
		if !capabilityVerified {
			t.Log("STATE_BACKING_UNQUALIFIED teardown=pass")
			return
		}
		value := "off"
		if soft {
			value = "on"
		}
		t.Log("STATE_BACKING_RESULT " + t.Name() + " filesystem=ufs softupdates=" + value + " teardown=pass")
	})
	output, err := run("/sbin/mdconfig", "-a", "-t", "swap", "-s", "128m")
	if err != nil {
		t.Fatal("owned state backing allocation")
	}
	candidate := strings.TrimSpace(string(output))
	if !regexp.MustCompile(`^md[0-9]+$`).MatchString(candidate) {
		t.Fatal("owned state backing identity")
	}
	device = candidate
	args := []string{"-O", "2", "-m", "0"}
	if soft {
		args = append(args, "-U")
	} else {
		args = append(args, "-u")
	}
	args = append(args, "/dev/"+device)
	if _, err := run("/sbin/newfs", args...); err != nil {
		t.Fatal("owned state backing format")
	}
	if _, err := run("/sbin/mount", "-t", "ufs", "/dev/"+device, container); err != nil {
		t.Fatal("owned state backing mount")
	}
	mounted = true
	if os.Chmod(container, 0700) != nil {
		t.Fatal("owned state backing permissions")
	}
	var fs unix.Statfs_t
	if unix.Statfs(container, &fs) != nil || unix.ByteSliceToString(fs.Fstypename[:]) != "ufs" || (fs.Flags&unix.MNT_SOFTDEP != 0) != soft {
		t.Fatal("owned state backing capability mismatch")
	}
	capabilityVerified = true
	return os.MkdirTemp(container, prefix)
}

func TestNativeStateSoftUpdates(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	anchor, err := nativeUFSStateAnchor(t, "horvium-softupdates-", true)
	if err != nil {
		t.Fatal(err)
	}
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("softupdates assertion", name)
		}
		results = append(results, name+"=pass")
	}
	var fs unix.Statfs_t
	check("enabled", unix.Statfs(anchor, &fs) == nil && fs.Flags&unix.MNT_SOFTDEP != 0)
	journalStore, journalFolder := nativeJournalFixture(t, anchor, "journal")
	check("journal_denial", errors.Is(journalStore.initialize(executionJournalFixture(), nil), ErrStateDirectory))
	if journalStore.close() != nil || os.RemoveAll(journalFolder) != nil {
		t.Fatal("owned softupdates journal close")
	}

	s, err := OpenStateDirectory(anchor)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	policy, err := DecodeOwnershipRecord(ownershipGolden())
	if err != nil {
		t.Fatal(err)
	}
	p := nativePropertyJobFixture(t)
	check("ownership_denial", errors.Is(s.PublishOwnership(policy, 0), ErrStateDirectory))
	check("job_denial", errors.Is(s.PublishJob(reservationJob(1, false), 0), ErrStateDirectory))
	check("property_denial", errors.Is(s.PublishPropertyJob(p, 0), ErrStateDirectory))
	got, duplicate, err := s.PublishPropertySubmission(p)
	check("submission_denial", errors.Is(err, ErrStateDirectory) && !duplicate && reflect.DeepEqual(got, PropertyJob{}))
	data := reservationBytes(t, p)
	target := filepath.Join(anchor, "job-"+p.job.ID+".json")
	if os.WriteFile(target, data, 0600) != nil {
		t.Fatal("owned softupdates read fixture")
	}
	got, duplicate, err = s.PublishPropertySubmission(p)
	check("retry_denial", errors.Is(err, ErrStateDirectory) && !duplicate && reflect.DeepEqual(got, PropertyJob{}))
	read, err := s.ReadPropertyJob(p.job.ID)
	check("read_available", err == nil && reflect.DeepEqual(read, p))
	current, err := os.ReadFile(target)
	check("bytes_preserved", err == nil && string(current) == string(data))
	entries, err := os.ReadDir(anchor)
	check("no_temporary", err == nil && len(entries) == 2)
	check("records_teardown", s.Close() == nil && os.RemoveAll(anchor) == nil)
	t.Log("STATE_SOFTUPDATES_RESULT " + strings.Join(results, " "))
}
