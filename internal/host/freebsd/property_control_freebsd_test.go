// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
)

func TestNativePropertyControlUIDChild(t *testing.T) {
	if os.Getenv("HORVIUM_CONTROL_UID_CHILD") != "1" {
		t.Skip("owned child only")
	}
	if os.Geteuid() == 0 {
		t.Fatal("child retained root")
	}
	if !errors.Is(propertySetNative(context.Background(), storage.PropertyIntent{}), ErrPermission) {
		t.Fatal("native control bypassed kernel UID")
	}
	if c, err := newPropertyClaimControl(nil, storage.PropertyJob{}, nil); c != nil || !errors.Is(err, ErrPermission) {
		t.Fatal("claim control binding bypassed kernel UID", err)
	}
	if b, err := newPropertyBackend(nil, "100", nil); b != nil || !errors.Is(err, ErrPermission) {
		t.Fatal("private backend bypassed kernel UID", err)
	}
}

// Test-only reexecuted fixture: a normally exiting parent leaves a child alive
// behind a file barrier. The base guard must withhold success until it is gone.
func TestNativePropertyControlGuardChild(t *testing.T) {
	args := flag.Args()
	if len(args) != 2 {
		return
	}
	dir, mode := args[0], args[1]
	if mode == "child" {
		if os.WriteFile(filepath.Join(dir, "child"), []byte(strconv.Itoa(os.Getpid())), 0600) != nil {
			os.Exit(2)
		}
		until := time.Now().Add(15 * time.Second)
		for time.Now().Before(until) {
			if _, err := os.Stat(filepath.Join(dir, "release")); err == nil {
				os.Exit(0)
			}
			time.Sleep(5 * time.Millisecond)
		}
		os.Exit(2)
	}
	if os.WriteFile(filepath.Join(dir, "parent"), []byte(strconv.Itoa(os.Getpid())), 0600) != nil {
		os.Exit(2)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestNativePropertyControlGuardChild$", "--", dir, "child")
	if mode == "escaped" {
		child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	if child.Start() != nil {
		os.Exit(2)
	}
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		if _, err := os.Stat(filepath.Join(dir, "child")); err == nil {
			os.Exit(0)
		}
		time.Sleep(5 * time.Millisecond)
	}
	os.Exit(2)
}

func nativePropertyControl(t *testing.T, ctx context.Context, source *PoolPropertySource, guid, pool string) {
	t.Helper()
	if os.Geteuid() != 0 || os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PROPERTY_VM") != "1" {
		t.Fatal("owned native root fixture only")
	}
	must := func(t testing.TB, path string, args ...string) string {
		t.Helper()
		out, err := execute(ctx, path, args, 1<<20)
		if err != nil {
			t.Fatal("owned control fixture command", err)
		}
		return string(out.data)
	}
	raw := func() string {
		return must(t, "/sbin/zfs", "get", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,property,value,received,source", "guid,quota,refquota,compression,mountpoint,canmount,readonly,mounted,encryption,encryptionroot,keystatus", pool)
	}
	original := raw()
	target := pool + "/control_probe"
	targetGUID := ""
	created := false
	path := ""
	anchor := ""
	metadata := pool + "/control_metadata"
	metadataGUID := ""
	metadataCreated := false
	var state *storage.StateDirectory
	cleanup := func() bool {
		c, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if state != nil {
			if state.Close() != nil {
				return false
			}
			state = nil
		}
		if path != "" {
			if os.RemoveAll(path) != nil {
				return false
			}
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				return false
			}
			path = ""
		}
		if created {
			out, err := execute(c, "/sbin/zfs", []string{"get", "-H", "-p", "-o", "value", "guid", target}, 4096)
			if err != nil || targetGUID == "" || strings.TrimSpace(string(out.data)) != targetGUID {
				return false
			}
			if _, err = execute(c, "/sbin/zfs", []string{"destroy", target}, 4096); err != nil {
				return false
			}
			if _, err = execute(c, "/sbin/zfs", []string{"list", "-H", target}, 4096); err == nil {
				return false
			}
			created = false
		}
		if metadataCreated {
			out, err := execute(c, "/sbin/zfs", []string{"get", "-H", "-p", "-o", "value", "guid", metadata}, 4096)
			if err != nil || metadataGUID == "" || strings.TrimSpace(string(out.data)) != metadataGUID {
				return false
			}
			if _, err = execute(c, "/sbin/zfs", []string{"destroy", metadata}, 4096); err != nil {
				return false
			}
			if _, err = execute(c, "/sbin/zfs", []string{"list", "-H", metadata}, 4096); err == nil {
				return false
			}
			metadataCreated = false
		}
		if anchor != "" {
			// Remove only the empty owned mountpoint after ZFS has unmounted it.
			if os.Remove(anchor) != nil {
				return false
			}
			if _, err := os.Lstat(anchor); !errors.Is(err, os.ErrNotExist) {
				return false
			}
			anchor = ""
		}
		return true
	}
	t.Cleanup(func() {
		if !cleanup() {
			t.Error("owned control teardown failed; discard guest")
		}
	})
	collect := func(t testing.TB) storage.Observation {
		t.Helper()
		o, err := source.Collect(ctx, guid, pool)
		if err != nil || o.Eligible {
			t.Fatal("fresh read-only control observation", err)
		}
		return o
	}
	var policy storage.Policy
	if !t.Run("preflight", func(t *testing.T) {
		if strings.TrimSpace(must(t, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)) != guid {
			t.Fatal("owned pool changed")
		}
		must(t, "/sbin/zfs", "create", "-o", "mountpoint=none", "-o", "compression=lz4", "-o", "quota=1048576", target)
		created = true
		targetGUID = strings.TrimSpace(must(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", target))
		// Actual publication requires a qualified backing. Keep the guest's UFS
		// softupdates refusal intact; both guests use this explicitly owned ZFS
		// child, independent of their root filesystem.
		var err error
		anchor, err = os.MkdirTemp("/root", "horvium-control-backing-")
		if err != nil {
			t.Fatal(err)
		}
		must(t, "/sbin/zfs", "create", "-o", "mountpoint="+anchor, metadata)
		metadataCreated = true
		metadataGUID = strings.TrimSpace(must(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", metadata))
		var fs unix.Statfs_t
		if os.Chmod(anchor, 0700) != nil || unix.Statfs(anchor, &fs) != nil || unix.ByteSliceToString(fs.Fstypename[:]) != "zfs" || strings.TrimSpace(must(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value", "mounted", metadata)) != "yes" {
			t.Fatal("owned ZFS metadata backing")
		}
		o := collect(t)
		policy = storage.Policy{Revision: 1, PoolGUID: guid, PoolName: pool}
		for _, f := range o.Filesystems {
			policy.Scopes = append(policy.Scopes, storage.Scope{GUID: f.GUID, Name: f.Name, ParentGUID: f.ParentGUID, Properties: []storage.Property{storage.Quota, storage.RefQuota, storage.Compression}})
		}
		path, err = os.MkdirTemp(anchor, "state-")
		if err != nil {
			t.Fatal(err)
		}
		state, err = storage.OpenStateDirectory(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = state.PublishOwnership(policy, 0); err != nil {
			t.Fatal(err)
		}
	}) {
		return
	}
	prepare := func(t testing.TB, property storage.Property, value string) (*propertyControl, storage.PropertyIntent, storage.PropertyJob) {
		t.Helper()
		o := collect(t)
		o.Eligible = true // Only this disposable capture; never product authority.
		at := time.Now().UTC().Truncate(time.Millisecond)
		plan, err := storage.Build(at, "native-control-owned-fixture", policy, o, storage.Intent{FilesystemGUID: targetGUID, Property: property, Value: value})
		if err != nil {
			t.Fatal(err)
		}
		i, err := plan.CapturePropertyIntent(at, "native-control-owned-fixture", policy, o)
		if err != nil {
			t.Fatal(err)
		}
		r := i.Review()
		var id [32]byte
		if _, err = rand.Read(id[:]); err != nil {
			t.Fatal(err)
		}
		j := storage.Job{ID: hex.EncodeToString(id[:]), Intent: storage.JobIntent{PlanID: r.ID, SubmissionID: hex.EncodeToString(id[:]), PolicyRevision: r.PolicyRevision, PoolGUID: r.PoolGUID, Change: r.Intent}, History: []storage.JobTransition{{Revision: 1, At: uint64(at.UnixMilli()), State: storage.JobQueued, Reason: "admitted"}}}
		job, err := storage.BindPropertyJob(j, i)
		if err != nil {
			t.Fatal(err)
		}
		if err = state.PublishPropertyJob(job, 0); err != nil {
			t.Fatal(err)
		}
		job, err = storage.TransitionPropertyJob(job, storage.JobApplying, uint64(at.UnixMilli()), "launch-intent", "")
		if err != nil {
			t.Fatal(err)
		}
		if err = state.PublishPropertyJob(job, 1); err != nil {
			t.Fatal(err)
		}
		expected, err := storage.EncodePropertyJob(job)
		if err != nil {
			t.Fatal(err)
		}
		// This fixture installs a test-only claim after declaring all storage. It
		// proves real file/native controls, not runtime eligibility or admission.
		c := &propertyControl{root: func() bool { return os.Geteuid() == 0 }, clock: time.Now, run: propertySetNative, authorize: func(cctx context.Context, _ storage.PropertyIntent) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
			current, e := state.ReadPropertyJob(job.Job().ID)
			if e != nil {
				return storage.PropertyJob{}, storage.Policy{}, storage.Observation{}, e
			}
			actual, e := storage.EncodePropertyJob(current)
			if e != nil || !bytes.Equal(actual, expected) {
				return storage.PropertyJob{}, storage.Policy{}, storage.Observation{}, storage.ErrDrift
			}
			p, e := state.ReadOwnership(guid)
			if e != nil {
				return storage.PropertyJob{}, storage.Policy{}, storage.Observation{}, e
			}
			fresh, e := source.Collect(cctx, guid, pool)
			return current, p, fresh, e
		}}
		return c, i, job
	}
	for _, tc := range []struct {
		name     string
		property storage.Property
		value    string
	}{{"quota", storage.Quota, "2097152"}, {"refquota", storage.RefQuota, "1048576"}, {"no-limit", storage.Quota, "none"}, {"compression", storage.Compression, "off"}} {
		if !t.Run(tc.name, func(t *testing.T) {
			c, i, job := prepare(t, tc.property, tc.value)
			quiet, err := c.apply(ctx, i)
			if err != nil || !quiet {
				t.Fatal("native typed property control", err)
			}
			after := collect(t)
			if err = i.ComparePostcondition(policy, after); err != nil {
				t.Fatal("independent property postcondition", err)
			}
			stored, err := state.ReadPropertyJob(job.Job().ID)
			a, _ := storage.EncodePropertyJob(stored)
			b, _ := storage.EncodePropertyJob(job)
			if err != nil || !bytes.Equal(a, b) {
				t.Fatal("control invented durable outcome", err)
			}
			if quiet, err = c.apply(ctx, i); !errors.Is(err, storage.ErrDenied) || quiet {
				t.Fatal("native control replay", err)
			}
			if tc.property == storage.RefQuota {
				must(t, "/sbin/zfs", "inherit", "-S", "refquota", target)
			} else if tc.property == storage.Quota {
				must(t, "/sbin/zfs", "set", "quota=1048576", target)
			} else {
				must(t, "/sbin/zfs", "set", "compression=lz4", target)
			}
		}) {
			return
		}
	}
	for _, mode := range []string{"expired", "ledger-drift", "native-drift"} {
		if !t.Run(mode, func(t *testing.T) {
			c, i, job := prepare(t, storage.Quota, "2097152")
			before := raw()
			if mode == "expired" {
				c.clock = func() time.Time { return i.Review().ExpiresAt }
			} else if mode == "ledger-drift" {
				next, err := storage.TransitionPropertyJob(job, storage.JobReconcileRequired, job.Job().History[1].At, "executor-unverified", "")
				if err != nil || state.PublishPropertyJob(next, 2) != nil {
					t.Fatal("owned ledger drift")
				}
			} else {
				must(t, "/sbin/zfs", "set", "compression=off", target)
				before = raw()
			}
			quiet, err := c.apply(ctx, i)
			if err == nil || quiet || raw() != before {
				t.Fatal("native denial changed storage", err)
			}
			if mode == "native-drift" {
				must(t, "/sbin/zfs", "set", "compression=lz4", target)
			}
		}) {
			return
		}
	}
	if !t.Run("wrong-uid", func(t *testing.T) {
		dir, err := os.MkdirTemp("/tmp", "horvium-control-uid-")
		if err != nil || os.Chmod(dir, 0755) != nil {
			t.Fatal("owned UID directory", err)
		}
		defer func() {
			if os.RemoveAll(dir) != nil {
				t.Error("UID teardown")
			}
			if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
				t.Error("UID fixture remains")
			}
		}()
		data, err := os.ReadFile(os.Args[0])
		binary := filepath.Join(dir, "control.test")
		if err != nil || os.WriteFile(binary, data, 0755) != nil {
			t.Fatal("owned executable copy", err)
		}
		cmd := exec.CommandContext(ctx, binary, "-test.v", "-test.run=^TestNativePropertyControlUIDChild$")
		cmd.Dir = dir
		cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin", "HORVIUM_CONTROL_UID_CHILD=1"}
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534, Groups: []uint32{}}}
		out, err := cmd.CombinedOutput()
		if err != nil || !bytes.Contains(out, []byte("--- PASS: TestNativePropertyControlUIDChild")) {
			t.Fatal("actual kernel UID denial", err)
		}
	}) {
		return
	}
	if !t.Run("guard-quiescence", func(t *testing.T) {
		for _, mode := range []string{"same-group", "escaped"} {
			t.Run(mode, func(t *testing.T) {
				dir := t.TempDir()
				cctx, stop := context.WithTimeout(ctx, 8*time.Second)
				defer stop()
				done := make(chan error, 1)
				waited := false
				t.Cleanup(func() {
					stop()
					if !waited {
						select {
						case <-done:
						case <-time.After(3 * time.Second):
							t.Error("owned guard did not finish cleanup")
						}
					}
				})
				go func() {
					_, err := executeNative(cctx, os.Args[0], []string{"-test.run=^TestNativePropertyControlGuardChild$", "--", dir, mode}, 4096)
					done <- err
				}()
				parent := fixturePID(t, filepath.Join(dir, "parent"))
				child := fixturePID(t, filepath.Join(dir, "child"))
				assertProbeStopped(t, parent)
				select {
				case err := <-done:
					waited = true
					t.Fatal("guard returned before descendant exited", err)
				default:
				}
				if os.WriteFile(filepath.Join(dir, "release"), nil, 0600) != nil {
					t.Fatal("owned release")
				}
				select {
				case err := <-done:
					waited = true
					if err != nil {
						t.Fatal("normal guarded completion", err)
					}
				case <-cctx.Done():
					t.Fatal("owned guard did not reap descendant")
				}
				assertProbeStopped(t, child)
			})
		}
	}) {
		return
	}
	t.Run("preserved", func(t *testing.T) {
		if !cleanup() || raw() != original {
			t.Fatal("owned control fixtures not restored")
		}
		collect(t)
	})
}
