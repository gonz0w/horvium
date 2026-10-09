// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
)

func TestNativeOriginalBackingUFS(t *testing.T) { originalBackingFixture(t, "ufs") }
func TestNativeOriginalBackingZFS(t *testing.T) { originalBackingFixture(t, "zfs") }

func originalBackingFixture(t *testing.T, kind string) {
	if os.Getenv("HORVIUM_ORIGINAL_CHILD") == "1" {
		s, e := storage.OpenStateDirectory(os.Getenv("HORVIUM_ORIGINAL_CHILD_PATH"))
		if e != nil {
			t.Fatal(e)
		}
		defer s.Close()
		if unix.Setgroups([]int{}) != nil || unix.Setgid(65534) != nil || unix.Setuid(65534) != nil || os.Geteuid() != 65534 {
			t.Fatal("actual credential drop")
		}
		got, e := collectOriginalExecutionDomain(context.Background(), s)
		if e == nil || got != (storage.ExecutionDomainObservation{}) {
			t.Fatal("original nonroot lease accepted", e)
		}
		t.Log("original-nonroot-denied")
		return
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_BACKING_VM") != "1" {
		t.Skip("disposable original backing VM required")
	}
	path, _ := persistentNativeFixture(t, kind)
	s, e := storage.OpenStateDirectory(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("original backing assertion", kind, name)
		}
		markers = append(markers, name+"=pass")
	}
	ctx := context.Background()
	first, e := collectOriginalExecutionDomain(ctx, s)
	check("native_domain", e == nil && first != (storage.ExecutionDomainObservation{}))
	again, e := collectOriginalExecutionDomain(ctx, s)
	check("repeat", e == nil && again == first)
	var native storage.ExecutionBackingObservation
	exact := false
	oracle := func(ctx context.Context, fd int) (storage.ExecutionBackingObservation, error) {
		var st, named unix.Stat_t
		exact = unix.Fstat(fd, &st) == nil && unix.Stat(path, &named) == nil && st.Dev == named.Dev && st.Ino == named.Ino
		b, e := collectNativePersistentBacking(ctx, fd)
		native = storage.ExecutionBackingObservation{Kind: b.kind, UFS: b.ufs, Pool: b.pool, Dataset: b.dataset}
		return native, e
	}
	again, e = s.ObserveExecutionDomain(ctx, oracle)
	check("original_descriptor", e == nil && exact && again == first && native.Kind == kind)
	called := false
	counted := func(context.Context, int) (storage.ExecutionBackingObservation, error) {
		called = true
		return native, nil
	}
	deny := func(lease *storage.StateDirectory, c context.Context, read func(context.Context, int) (storage.ExecutionBackingObservation, error)) error {
		t.Helper()
		got, e := lease.ObserveExecutionDomain(c, read)
		if e == nil || got != (storage.ExecutionDomainObservation{}) {
			t.Fatal("partial or unsafe domain", e)
		}
		return e
	}
	check("nil_context", deny(s, nil, counted) != nil && !called)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	check("cancelled", errors.Is(deny(s, cancelled, counted), context.Canceled) && !called)
	check("nil_lease", deny(nil, ctx, counted) != nil && !called)
	check("inert_lease", deny(&storage.StateDirectory{}, ctx, counted) != nil && !called)
	// A real structural copy retains the foreign original pointer. Do not Close
	// it: its descriptors are borrowed from the owned original and must survive.
	copyLease := new(storage.StateDirectory)
	reflect.ValueOf(copyLease).Elem().Set(reflect.ValueOf(s).Elem())
	check("copied_lease", errors.Is(deny(copyLease, ctx, counted), storage.ErrDenied) && !called)
	check("nil_reader", deny(s, ctx, nil) != nil)
	check("reader_error", deny(s, ctx, func(context.Context, int) (storage.ExecutionBackingObservation, error) {
		return native, storage.ErrUnavailable
	}) != nil)
	check("invalid_projection", deny(s, ctx, func(context.Context, int) (storage.ExecutionBackingObservation, error) {
		return storage.ExecutionBackingObservation{}, nil
	}) != nil)
	other := storage.ExecutionBackingObservation{Kind: "ufs", UFS: [2]uint32{1, 2}}
	if kind == "ufs" {
		other = storage.ExecutionBackingObservation{Kind: "zfs", Pool: 1, Dataset: 2}
	}
	check("filesystem_mismatch", errors.Is(deny(s, ctx, func(context.Context, int) (storage.ExecutionBackingObservation, error) { return other, nil }), storage.ErrDrift))
	live, stop := context.WithCancel(ctx)
	check("in_call_cancel", errors.Is(deny(s, live, func(context.Context, int) (storage.ExecutionBackingObservation, error) { stop(); return native, nil }), context.Canceled))
	bounded := false
	again, e = s.ObserveExecutionDomain(ctx, func(c context.Context, _ int) (storage.ExecutionBackingObservation, error) {
		deadline, ok := c.Deadline()
		bounded = ok && time.Until(deadline) > 0 && time.Until(deadline) <= 45*time.Second
		return native, nil
	})
	check("bounded_context", e == nil && again == first && bounded)
	lock := filepath.Join(path, "writer.lock")
	check("directory_mode_denial", os.Chmod(path, 0750) == nil && deny(s, ctx, counted) != nil && !called)
	if os.Chmod(path, 0700) != nil {
		t.Fatal("restore directory mode")
	}
	check("lock_mode_denial", os.Chmod(lock, 0640) == nil && deny(s, ctx, counted) != nil && !called)
	if os.Chmod(lock, 0600) != nil {
		t.Fatal("restore lock mode")
	}
	check("lock_nonempty_denial", os.WriteFile(lock, []byte("x"), 0600) == nil && deny(s, ctx, counted) != nil && !called)
	if os.Truncate(lock, 0) != nil {
		t.Fatal("restore empty lock")
	}
	check("lock_hardlink_denial", os.Link(lock, filepath.Join(path, "alias")) == nil && deny(s, ctx, counted) != nil && !called)
	if os.Remove(filepath.Join(path, "alias")) != nil {
		t.Fatal("remove owned alias")
	}
	check("same_inode_drift", errors.Is(deny(s, ctx, func(context.Context, int) (storage.ExecutionBackingObservation, error) {
		if os.Chmod(lock, 0640) != nil || os.Chmod(lock, 0600) != nil {
			t.Fatal("owned lock drift")
		}
		return native, nil
	}), storage.ErrDrift))
	check("namespace_drift", errors.Is(deny(s, ctx, func(context.Context, int) (storage.ExecutionBackingObservation, error) {
		if os.WriteFile(filepath.Join(path, "changed"), nil, 0600) != nil {
			t.Fatal("owned directory drift")
		}
		return native, nil
	}), storage.ErrDrift))
	if os.Remove(filepath.Join(path, "changed")) != nil {
		t.Fatal("remove owned drift")
	}
	old := filepath.Join(path, "old-lock")
	check("lock_replacement_denial", deny(s, ctx, func(context.Context, int) (storage.ExecutionBackingObservation, error) {
		if os.Rename(lock, old) != nil || os.WriteFile(lock, nil, 0600) != nil {
			t.Fatal("owned lock replacement")
		}
		return native, nil
	}) != nil)
	if os.Remove(lock) != nil || os.Rename(old, lock) != nil {
		t.Fatal("restore original lock")
	}
	again, e = collectOriginalExecutionDomain(ctx, s)
	check("restored_identity", e == nil && again == first)
	childPath := filepath.Join(path, "child")
	if os.Mkdir(childPath, 0700) != nil {
		t.Fatal("owned child lease")
	}
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	childCtx, end := context.WithTimeout(ctx, 15*time.Second)
	cmd := exec.CommandContext(childCtx, exe, "-test.v", "-test.run=^"+t.Name()+"$")
	cmd.Env = append(os.Environ(), "HORVIUM_ORIGINAL_CHILD=1", "HORVIUM_ORIGINAL_CHILD_PATH="+childPath)
	out, e := cmd.CombinedOutput()
	end()
	check("kernel_uid_denial", e == nil && strings.Contains(string(out), "original-nonroot-denied"))
	check("child_reaped", cmd.ProcessState != nil && cmd.ProcessState.Success())
	if os.RemoveAll(childPath) != nil {
		t.Fatal("owned child cleanup")
	}
	// Trigger real publication uncertainty on a separate original lease through an
	// owned orphan, then prove observation cannot clear that latch after removal.
	uncertainPath := filepath.Join(path, "uncertain")
	if os.Mkdir(uncertainPath, 0700) != nil {
		t.Fatal("owned uncertain lease")
	}
	u, e := storage.OpenStateDirectory(uncertainPath)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = u.Close() })
	orphan := filepath.Join(uncertainPath, ".ownership-"+strings.Repeat("a", 32)+".tmp")
	if os.WriteFile(orphan, []byte("owned incomplete publication"), 0600) != nil {
		t.Fatal("owned orphan")
	}
	policy := storage.Policy{Revision: 1, PoolGUID: "100", PoolName: "tank", Scopes: []storage.Scope{{GUID: "101", Name: "tank", Properties: []storage.Property{storage.Quota}}}}
	check("uncertainty_latched", errors.Is(u.PublishOwnership(policy, 0), storage.ErrStateUncertain))
	if os.Remove(orphan) != nil {
		t.Fatal("owned orphan removal")
	}
	check("uncertainty_denial", errors.Is(deny(u, ctx, counted), storage.ErrStateUncertain) && !called)
	check("uncertainty_preserved", errors.Is(u.PublishOwnership(policy, 0), storage.ErrStateUncertain))
	if u.Close() != nil || os.RemoveAll(uncertainPath) != nil {
		t.Fatal("uncertain fixture cleanup")
	}
	// Close must wait for the same mutex for the entire descriptor borrowing.
	entered, release := make(chan struct{}), make(chan struct{})
	observed := make(chan error, 1)
	go func() {
		_, e := s.ObserveExecutionDomain(ctx, func(context.Context, int) (storage.ExecutionBackingObservation, error) {
			close(entered)
			<-release
			return native, nil
		})
		observed <- e
	}()
	<-entered
	closeStarted, closed := make(chan struct{}), make(chan error, 1)
	go func() { close(closeStarted); closed <- s.Close() }()
	<-closeStarted
	blocked := false
	select {
	case <-closed:
	case <-time.After(30 * time.Millisecond):
		blocked = true
	}
	close(release)
	check("close_excluded", blocked)
	check("observation_finishes", <-observed == nil)
	check("close_finishes", <-closed == nil)
	check("closed_denial", deny(s, ctx, counted) != nil && !called)
	s, e = storage.OpenStateDirectory(path)
	if e != nil {
		t.Fatal(e)
	}
	reopened, e := collectOriginalExecutionDomain(ctx, s)
	check("reopen_new_origin", e == nil && reopened != (storage.ExecutionDomainObservation{}) && reopened != first)
	check("final_close", s.Close() == nil)
	t.Log("ORIGINAL_BACKING_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
}
