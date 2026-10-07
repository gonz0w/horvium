// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestEnrollmentPublicationKernelUIDProcess(t *testing.T) {
	if os.Getenv("HORVIUM_ENROLLMENT_UID_CHILD") != "1" {
		t.Skip("owned credential child only")
	}
	if os.Geteuid() == 0 {
		t.Fatal("actual nonroot credential required")
	}
	p := PropertyEnrollmentState{lease: &StateDirectory{}}
	if e := p.publishEnrollmentMetadata(context.Background(), "root", Observation{}, nil, PoolSystemUse{}, time.Now, defaultStateCatalogLimits, nil); !errors.Is(e, ErrStateDirectory) {
		t.Fatal("kernel uid guard", e)
	}
}

// Actual metadata filesystem/transactions; typed native inputs are synthetic
// fixture facts. The pool-property suite separately proves actual native inputs.
func TestNativeEnrollmentPublication(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable root state VM required")
	}
	anchor, e := nativeStateAnchor(t, "horvium-enrollment-publish-")
	if e != nil {
		t.Fatal(e)
	}
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("enrollment publication", name)
		}
		results = append(results, name+"=pass")
	}
	p, o, d, u := enrollmentFixture(t)
	var fs unix.Statfs_t
	if e = unix.Statfs(anchor, &fs); e != nil {
		t.Fatal(e)
	}
	// Bind the actual owned metadata mount to explicitly synthetic disjoint system
	// providers. These fixture views qualify metadata, not runtime device safety.
	u.system.Mounts[0].Source = unix.ByteSliceToString(fs.Mntfromname[:])
	u.system.Mounts[0].Target = unix.ByteSliceToString(fs.Mntonname[:])
	u.system.Mounts[0].Type = unix.ByteSliceToString(fs.Fstypename[:])
	at := time.Now()
	newLease := func(name string) (*StateDirectory, string) {
		t.Helper()
		path := filepath.Join(anchor, name)
		if os.Mkdir(path, 0700) != nil {
			t.Fatal("owned directory")
		}
		s, e := OpenStateDirectory(path)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { _ = s.Close() })
		return s, path
	}
	write := func(path, name string, p Policy) []byte {
		t.Helper()
		b, e := EncodeOwnershipRecord(p)
		if e != nil || os.WriteFile(filepath.Join(path, name), b, 0600) != nil {
			t.Fatal("owned policy fixture", e)
		}
		return b
	}
	bind := func(s *StateDirectory, next, previous Policy) PropertyEnrollmentState {
		t.Helper()
		plan, e := BuildPropertyEnrollment(at, "root", next, o, d, u)
		if e != nil {
			t.Fatal(e)
		}
		bound, e := s.BindPropertyEnrollment(at, "root", plan, previous, o, d, u)
		if e != nil {
			t.Fatal("binding", e)
		}
		return bound
	}
	publish := func(bound PropertyEnrollmentState, ctx context.Context, clock func() time.Time, limits stateCatalogLimits, wrap func(ownershipPublication) ownershipPublication) error {
		return bound.publish(ctx, "root", o, d, u, clock, func() error { return bound.publishEnrollmentMetadata(ctx, "root", o, d, u, clock, limits, wrap) })
	}
	clock := func() time.Time { return at }
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
	s, path := newLease("first")
	first := bind(s, p, Policy{})
	rebound, e := s.BindPropertyEnrollment(at, "root", first.plan, Policy{}, o, d, u)
	if e != nil {
		t.Fatal(e)
	}
	check("first", first.Publish(context.Background(), "root", o, d, u) == nil)
	b, e := os.ReadFile(filepath.Join(path, "ownership-100.json"))
	expected, _ := EncodeOwnershipRecord(p)
	check("rebound_review_denied", errors.Is(rebound.Publish(context.Background(), "root", o, d, u), ErrDenied))
	check("exact_bytes", e == nil && bytes.Equal(b, expected))
	info, e := os.Lstat(filepath.Join(path, "ownership-100.json"))
	if e != nil {
		t.Fatal(e)
	}
	st := info.Sys().(*syscall.Stat_t)
	check("private_file", info.Mode().IsRegular() && st.Uid == 0 && st.Mode&07777 == 0600 && st.Nlink == 1)
	lock, e := os.Lstat(filepath.Join(path, stateWriterName))
	if e != nil {
		t.Fatal(e)
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if first.Publish(context.Background(), "root", o, d, u) == nil {
				successes.Add(1)
			}
		})
	}
	wg.Wait()
	check("copies_denied", successes.Load() == 0)
	old, e := os.Open(filepath.Join(path, "ownership-100.json"))
	if e != nil {
		t.Fatal(e)
	}
	next := cloneEnrollmentPolicy(p)
	next.Revision++
	successor := bind(s, next, p)
	check("successor", successor.Publish(context.Background(), "root", o, d, u) == nil)
	oldBytes := make([]byte, len(expected))
	_, readErr := old.Read(oldBytes)
	closeErr := old.Close()
	check("old_inode", readErr == nil && closeErr == nil && bytes.Equal(oldBytes, expected))
	currentLock, e := os.Lstat(filepath.Join(path, stateWriterName))
	check("stable_lock", e == nil && currentLock.Sys().(*syscall.Stat_t).Ino == lock.Sys().(*syscall.Stat_t).Ino)
	check("no_temporary", noTemps(path))
	legacyLease, legacyPath := newLease("legacy")
	legacy := cloneEnrollmentPolicy(p)
	legacy.Members = nil
	write(legacyPath, "ownership-100.json", legacy)
	check("legacy", bind(legacyLease, next, legacy).Publish(context.Background(), "root", o, d, u) == nil)
	serialLease, _ := newLease("serialized")
	left, right := bind(serialLease, p, Policy{}), bind(serialLease, p, Policy{})
	var serialSuccess atomic.Int32
	for _, candidate := range []PropertyEnrollmentState{left, right} {
		wg.Go(func() {
			e := candidate.Publish(context.Background(), "root", o, d, u)
			if e == nil {
				serialSuccess.Add(1)
			} else if !errors.Is(e, ErrDrift) {
				t.Error(e)
			}
		})
	}
	wg.Wait()
	check("serialized_reviews", serialSuccess.Load() == 1)
	_ = serialLease.Close()
	modes := []string{"record_drift", "unrelated_drift", "unknown", "orphan", "queued_hold", "terminal_hold", "late_policy", "late_policy_before_rename", "late_unrelated", "late_job", "late_unknown", "late_orphan", "file_sync", "rename", "cleanup", "directory_sync", "capacity", "expired_after_sync", "cancelled_after_sync", "backwards_before_rename", "expired_before_rename", "late_success", "closed"}
	for _, mode := range modes {
		lease, path := newLease(mode)
		original := write(path, "ownership-100.json", p)
		foreign := Policy{Revision: 1, PoolGUID: "999", PoolName: "foreign_fixture"}
		foreignBytes := write(path, "ownership-999.json", foreign)
		bound := bind(lease, next, p)
		ctx, cancel := context.WithCancel(context.Background())
		now := at
		testClock := func() time.Time { return now }
		limits := defaultStateCatalogLimits
		changed := cloneEnrollmentPolicy(p)
		changed.Scopes[0].Properties = []Property{Compression}
		mutate := func(which string) {
			switch which {
			case "policy":
				write(path, "ownership-100.json", changed)
			case "unrelated":
				foreign.Revision++
				write(path, "ownership-999.json", foreign)
			case "unknown":
				if os.WriteFile(filepath.Join(path, "foreign"), []byte("marker"), 0600) != nil {
					t.Fatal("marker")
				}
			case "orphan":
				if os.WriteFile(filepath.Join(path, ".ownership-"+strings.Repeat("d", 32)+".tmp"), []byte("marker"), 0600) != nil {
					t.Fatal("orphan")
				}
			case "job", "terminal":
				j := jobFixture()
				if which == "terminal" {
					j = restartJobFixture(t, j, JobCancelled, "", "")
				}
				data, e := EncodeJobRecord(j)
				if e != nil || os.WriteFile(filepath.Join(path, "job-"+j.ID+".json"), data, 0600) != nil {
					t.Fatal("job", e)
				}
			}
		}
		switch mode {
		case "record_drift":
			mutate("policy")
		case "unrelated_drift":
			mutate("unrelated")
		case "unknown", "orphan":
			mutate(mode)
		case "queued_hold":
			mutate("job")
		case "terminal_hold":
			mutate("terminal")
		case "capacity":
			limits.bytes = uint64(len(original) + len(foreignBytes))
		case "closed":
			_ = lease.Close()
		}
		wrap := func(tx ownershipPublication) ownershipPublication {
			switch mode {
			case "file_sync":
				tx.syncFile = func() error { return errors.New("owned file sync failure") }
			case "rename":
				tx.rename = func() error { return errors.New("owned rename failure") }
			case "cleanup":
				tx.write = func() error { return errors.New("owned write failure") }
				tx.cleanup = func() error { return errors.New("owned cleanup failure") }
			case "directory_sync":
				tx.syncDirectory = func() error { return errors.New("owned directory sync failure") }
			case "late_success":
				original := tx.syncDirectory
				tx.syncDirectory = func() error { e := original(); cancel(); return e }
			case "late_policy_before_rename":
				original := tx.rename
				tx.rename = func() error { mutate("policy"); return original() }
			case "backwards_before_rename", "expired_before_rename":
				original := tx.rename
				tx.rename = func() error {
					if mode == "backwards_before_rename" {
						now = at.Add(-time.Second)
					} else {
						now = at.Add(5 * time.Minute)
					}
					return original()
				}
			default:
				original := tx.syncFile
				tx.syncFile = func() error {
					e := original()
					if e != nil {
						return e
					}
					switch mode {
					case "late_policy":
						mutate("policy")
					case "late_unrelated":
						mutate("unrelated")
					case "late_job":
						mutate("job")
					case "late_unknown":
						mutate("unknown")
					case "late_orphan":
						mutate("orphan")
					case "expired_after_sync":
						now = at.Add(5 * time.Minute)
					case "cancelled_after_sync":
						cancel()
					}
					return nil
				}
			}
			return tx
		}
		err := publish(bound, ctx, testClock, limits, wrap)
		expectedReason := ErrStateDirectory
		switch mode {
		case "record_drift", "unrelated_drift", "late_policy", "late_policy_before_rename", "late_unrelated":
			expectedReason = ErrDrift
		case "unknown", "late_unknown":
			expectedReason = ErrStateRecord
		case "orphan", "late_orphan", "cleanup", "directory_sync":
			expectedReason = ErrStateUncertain
		case "queued_hold", "terminal_hold", "late_job":
			expectedReason = ErrDenied
		case "capacity":
			expectedReason = ErrLimit
		case "expired_after_sync", "backwards_before_rename", "expired_before_rename":
			expectedReason = ErrExpired
		case "cancelled_after_sync":
			expectedReason = context.Canceled
		}
		ok := errors.Is(err, expectedReason)
		if mode == "late_success" {
			ok = err == nil
		}
		check(mode, ok)
		checkName := ""
		switch mode {
		case "orphan":
			_ = os.Remove(filepath.Join(path, ".ownership-"+strings.Repeat("d", 32)+".tmp"))
			checkName = "orphan_latched"
		case "cleanup":
			checkName = "cleanup_blocks"
		case "directory_sync":
			checkName = "uncertain_blocks"
		}
		if checkName != "" {
			check(checkName, errors.Is(lease.PublishOwnership(next, p.Revision), ErrStateUncertain))
		}
		got, e := os.ReadFile(filepath.Join(path, "ownership-100.json"))
		newBytes, _ := EncodeOwnershipRecord(next)
		if mode == "directory_sync" {
			check("uncertain_visible", e == nil && bytes.Equal(got, newBytes))
		} else if mode == "late_success" {
			if e != nil || !bytes.Equal(got, newBytes) {
				t.Fatal("late acknowledgement record")
			}
		} else if mode != "record_drift" && mode != "late_policy" && mode != "late_policy_before_rename" {
			if e != nil || !bytes.Equal(got, original) {
				t.Fatal("denial changed predecessor", mode, e)
			}
		}
		foreignGot, e := os.ReadFile(filepath.Join(path, "ownership-999.json"))
		if mode != "unrelated_drift" && mode != "late_unrelated" && (e != nil || !bytes.Equal(foreignGot, foreignBytes)) {
			t.Fatal("foreign changed", mode)
		}
		if mode != "cleanup" && mode != "orphan" && mode != "late_orphan" && !noTemps(path) {
			t.Fatal("temporary leaked", mode)
		}
		if e := publish(bound, context.Background(), clock, defaultStateCatalogLimits, nil); !errors.Is(e, ErrDenied) {
			t.Fatal("attempt replayed", mode, e)
		}
		cancel()
		_ = lease.Close()
	}
	childDirectory, e := os.MkdirTemp("/tmp", "horvium-enrollment-uid-")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if os.RemoveAll(childDirectory) != nil {
			t.Error("uid child fixture teardown")
		}
	})
	if os.Chmod(childDirectory, 0755) != nil {
		t.Fatal("owned executable directory")
	}
	executable, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	src, e := os.Open(executable)
	if e != nil {
		t.Fatal(e)
	}
	dst, e := os.OpenFile(filepath.Join(childDirectory, "owned.test"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if e != nil {
		t.Fatal(e)
	}
	_, copyErr := io.Copy(dst, src)
	sourceClose := src.Close()
	targetClose := dst.Close()
	if copyErr != nil || sourceClose != nil || targetClose != nil {
		t.Fatal("owned executable copy")
	}
	childContext, childCancel := context.WithTimeout(context.Background(), 30*time.Second)
	cmd := exec.CommandContext(childContext, filepath.Join(childDirectory, "owned.test"), "-test.run=^TestEnrollmentPublicationKernelUIDProcess$")
	cmd.Dir = "/"
	cmd.Env = append(os.Environ(), "HORVIUM_ENROLLMENT_UID_CHILD=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534, Groups: []uint32{}}}
	childErr := cmd.Run()
	childCancel()
	check("kernel_uid_denied", childErr == nil && os.RemoveAll(childDirectory) == nil)
	check("zero_denied", errors.Is((PropertyEnrollmentState{}).Publish(context.Background(), "root", o, d, u), ErrInvalid))
	check("fixture_teardown", s.Close() == nil && legacyLease.Close() == nil && os.RemoveAll(anchor) == nil)
	t.Log("ENROLLMENT_PUBLICATION_RESULT " + strings.Join(results, " "))
}
