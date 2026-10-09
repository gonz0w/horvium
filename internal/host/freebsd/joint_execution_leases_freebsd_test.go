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

func TestNativeJointExecutionLeasesUFS(t *testing.T) { jointExecutionFixture(t, "ufs") }
func TestNativeJointExecutionLeasesZFS(t *testing.T) { jointExecutionFixture(t, "zfs") }

func jointExecutionFixture(t *testing.T, kind string) {
	if os.Getenv("HORVIUM_JOINT_CHILD") == "1" {
		p, e := storage.OpenExecutionDomainLeases(context.Background(), os.Getenv("HORVIUM_JOINT_JOURNAL"), os.Getenv("HORVIUM_JOINT_LEDGER"))
		if e != nil {
			t.Fatal(e)
		}
		defer p.Close()
		if unix.Setgroups([]int{}) != nil || unix.Setgid(65534) != nil || unix.Setuid(65534) != nil || os.Geteuid() != 65534 {
			t.Fatal("kernel credential drop")
		}
		o, e := observeNativeExecutionDomains(context.Background(), p, strings.Repeat("1", 32))
		if e == nil || o != (storage.ExecutionDomainsObservation{}) {
			t.Fatal("nonroot joint owner accepted", e)
		}
		t.Log("joint-kernel-uid-denied")
		return
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_BACKING_VM") != "1" {
		t.Skip("disposable joint lease VM required")
	}
	path, run := persistentNativeFixture(t, kind)
	journal, ledger := filepath.Join(path, "journal"), filepath.Join(path, "ledger")
	pool := ""
	if kind == "zfs" {
		var fs unix.Statfs_t
		if unix.Statfs(path, &fs) != nil {
			t.Fatal("owned ZFS mount")
		}
		pool = strings.Split(unix.ByteSliceToString(fs.Mntfromname[:]), "/")[0]
		run("/sbin/zfs", "create", pool+"/journal")
		journal = run("/sbin/zfs", "get", "-H", "-o", "value", "mountpoint", pool+"/journal")
		ledger = path
		if os.Chmod(journal, 0700) != nil {
			t.Fatal("owned journal mode")
		}
	} else if os.Mkdir(journal, 0700) != nil || os.Mkdir(ledger, 0700) != nil {
		t.Fatal("owned pair directories")
	}
	for _, p := range []string{journal, ledger} {
		if os.WriteFile(filepath.Join(p, "writer.lock"), nil, 0600) != nil {
			t.Fatal("owned pre-existing lock")
		}
	}
	ctx := context.Background()
	installation := strings.Repeat("1", 32)
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("joint assertion", kind, name)
		}
		markers = append(markers, name+"=pass")
	}
	open := func() *storage.ExecutionDomainLeases {
		t.Helper()
		p, e := openNativeExecutionDomains(ctx, journal, ledger)
		if e != nil || p == nil {
			t.Fatal("actual joint open", e)
		}
		return p
	}
	p, e := openNativeExecutionDomains(ctx, journal, ledger)
	check("pair_open", e == nil && p != nil)
	t.Cleanup(func() { _ = p.Close() })
	first, e := observeNativeExecutionDomains(ctx, p, installation)
	check("native_pair", e == nil && first != (storage.ExecutionDomainsObservation{}))
	again, e := observeNativeExecutionDomains(ctx, p, installation)
	check("repeat", e == nil && again == first)
	order := []string{}
	ids := map[string]storage.ExecutionBackingObservation{}
	bounded := true
	read := func(c context.Context, fd int) (storage.ExecutionBackingObservation, error) {
		var st, jst, lst unix.Stat_t
		if unix.Fstat(fd, &st) != nil || unix.Stat(journal, &jst) != nil || unix.Stat(ledger, &lst) != nil {
			return storage.ExecutionBackingObservation{}, storage.ErrUnavailable
		}
		label := ""
		if st.Dev == jst.Dev && st.Ino == jst.Ino {
			label = "journal"
		}
		if st.Dev == lst.Dev && st.Ino == lst.Ino {
			label = "ledger"
		}
		if label == "" {
			return storage.ExecutionBackingObservation{}, storage.ErrDrift
		}
		order = append(order, label)
		deadline, ok := c.Deadline()
		bounded = bounded && ok && time.Until(deadline) > 0 && time.Until(deadline) <= 90*time.Second
		b, e := readNativeExecutionBacking(c, fd)
		ids[label] = b
		return b, e
	}
	again, e = p.Observe(ctx, installation, read)
	check("journal_first_probes", e == nil && again == first && strings.Join(order, ",") == "journal,ledger,journal,ledger")
	check("bounded_pair_context", bounded)
	j, l := ids["journal"], ids["ledger"]
	good := j.Kind == kind && l.Kind == kind
	if kind == "ufs" {
		good = good && j.UFS == l.UFS && j.UFS != ([2]uint32{})
	} else {
		good = good && j.Pool == l.Pool && j.Pool != 0 && j.Dataset != l.Dataset
	}
	check("actual_backing_pair", good)
	for _, name := range []string{journal, ledger} {
		s, e := storage.OpenStateDirectory(name)
		if s != nil {
			_ = s.Close()
		}
		check(map[bool]string{true: "journal_busy", false: "ledger_busy"}[name == journal], s == nil && errors.Is(e, storage.ErrStateLocked))
	}
	deny := func(q *storage.ExecutionDomainLeases, c context.Context, inst string, r func(context.Context, int) (storage.ExecutionBackingObservation, error)) error {
		t.Helper()
		o, e := q.Observe(c, inst, r)
		if e == nil || o != (storage.ExecutionDomainsObservation{}) {
			t.Fatal("partial or unsafe joint observation", e)
		}
		return e
	}
	called := false
	never := func(context.Context, int) (storage.ExecutionBackingObservation, error) { called = true; return j, nil }
	check("nil_owner", deny(nil, ctx, installation, never) != nil && !called)
	check("inert_owner", deny(&storage.ExecutionDomainLeases{}, ctx, installation, never) != nil && !called)
	check("nil_context", deny(p, nil, installation, never) != nil && !called)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	check("cancelled", errors.Is(deny(p, cancelled, installation, never), context.Canceled) && !called)
	check("invalid_installation", deny(p, ctx, "", never) != nil && !called)
	check("nil_reader", deny(p, ctx, installation, nil) != nil)
	copied := new(storage.ExecutionDomainLeases)
	reflect.ValueOf(copied).Elem().Set(reflect.ValueOf(p).Elem())
	check("copied_owner", errors.Is(deny(copied, ctx, installation, never), storage.ErrDenied) && !called)
	check("copied_close", errors.Is(copied.Close(), storage.ErrDenied))
	again, e = observeNativeExecutionDomains(ctx, p, installation)
	check("original_survives_copy", e == nil && again == first)
	check("first_close", p.Close() == nil)
	held, e := storage.OpenStateDirectory(journal)
	if e != nil {
		t.Fatal(e)
	}
	q, e := openNativeExecutionDomains(ctx, journal, filepath.Join(path, "missing-ledger"))
	check("journal_first_acquisition", q == nil && errors.Is(e, storage.ErrStateLocked))
	if held.Close() != nil {
		t.Fatal("journal contention cleanup")
	}
	held, e = storage.OpenStateDirectory(ledger)
	if e != nil {
		t.Fatal(e)
	}
	q, e = openNativeExecutionDomains(ctx, journal, ledger)
	check("ledger_contention", q == nil && errors.Is(e, storage.ErrStateLocked))
	free, e := storage.OpenStateDirectory(journal)
	check("partial_open_cleanup", e == nil && free != nil)
	if free.Close() != nil || held.Close() != nil {
		t.Fatal("partial open handles cleanup")
	}
	for _, side := range []string{"journal", "ledger"} {
		name := journal
		if side == "ledger" {
			name = ledger
		}
		lock := filepath.Join(name, "writer.lock")
		if os.Remove(lock) != nil {
			t.Fatal("owned missing lock fixture")
		}
		q, e = openNativeExecutionDomains(ctx, journal, ledger)
		_, statErr := os.Lstat(lock)
		check(side+"_lock_not_created", q == nil && e != nil && os.IsNotExist(statErr))
		if os.WriteFile(lock, nil, 0600) != nil {
			t.Fatal("explicit fixture lock restoration")
		}
	}
	q, e = openNativeExecutionDomains(ctx, journal, journal)
	check("aliased_domains_denied", q == nil && e != nil)
	// Every unsafe observation retires its original pair; restoration cannot
	// reconstruct authority. Subsequent cases use freshly opened observation owners.
	p = open()
	step := 0
	e = deny(p, ctx, installation, func(c context.Context, fd int) (storage.ExecutionBackingObservation, error) {
		step++
		b, e := readNativeExecutionBacking(c, fd)
		if step == 2 {
			name := filepath.Join(journal, "writer.lock")
			if os.Chmod(name, 0640) != nil || os.Chmod(name, 0600) != nil {
				t.Fatal("owned cross-domain drift")
			}
		}
		return b, e
	})
	check("cross_domain_drift", errors.Is(e, storage.ErrDrift))
	check("drift_retirement", errors.Is(deny(p, ctx, installation, never), storage.ErrStateUncertain) && !called)
	if p.Close() != nil {
		t.Fatal("drift owner cleanup")
	}
	policy := storage.Policy{Revision: 1, PoolGUID: "100", PoolName: "tank", Scopes: []storage.Scope{{GUID: "101", Name: "tank", Properties: []storage.Property{storage.Quota}}}}
	data, e := storage.EncodeOwnershipRecord(policy)
	if e != nil {
		t.Fatal(e)
	}
	record := filepath.Join(ledger, "ownership-100.json")
	if os.WriteFile(record, data, 0600) != nil {
		t.Fatal("owned ledger record")
	}
	p = open()
	step = 0
	e = deny(p, ctx, installation, func(c context.Context, fd int) (storage.ExecutionBackingObservation, error) {
		step++
		b, e := readNativeExecutionBacking(c, fd)
		if step == 3 && os.WriteFile(record, append(append([]byte(nil), data...), '\n'), 0600) != nil {
			t.Fatal("owned exact ledger bytes drift")
		}
		return b, e
	})
	check("exact_ledger_census_drift", errors.Is(e, storage.ErrDrift))
	if p.Close() != nil || os.Remove(record) != nil {
		t.Fatal("ledger drift cleanup")
	}
	p = open()
	step = 0
	unknown := filepath.Join(journal, "unknown")
	e = deny(p, ctx, installation, func(c context.Context, fd int) (storage.ExecutionBackingObservation, error) {
		step++
		b, e := readNativeExecutionBacking(c, fd)
		if step == 3 && os.WriteFile(unknown, nil, 0600) != nil {
			t.Fatal("owned journal drift")
		}
		return b, e
	})
	check("journal_namespace_drift", e != nil)
	if p.Close() != nil || os.Remove(unknown) != nil {
		t.Fatal("journal drift cleanup")
	}
	p = open()
	e = deny(p, ctx, installation, func(context.Context, int) (storage.ExecutionBackingObservation, error) {
		return j, storage.ErrUnavailable
	})
	check("reader_error", e != nil)
	check("reader_error_retirement", errors.Is(deny(p, ctx, installation, never), storage.ErrStateUncertain) && !called)
	if p.Close() != nil {
		t.Fatal("reader failure cleanup")
	}
	p = open()
	live, stop := context.WithCancel(ctx)
	e = deny(p, live, installation, func(c context.Context, fd int) (storage.ExecutionBackingObservation, error) {
		b, e := readNativeExecutionBacking(c, fd)
		stop()
		return b, e
	})
	check("in_call_cancel", errors.Is(e, context.Canceled))
	if p.Close() != nil {
		t.Fatal("cancellation cleanup")
	}
	// Open the child's own pair while root, then drop real kernel credentials.
	cj, cl := filepath.Join(journal, "child-journal"), filepath.Join(ledger, "child-ledger")
	for _, name := range []string{cj, cl} {
		if os.Mkdir(name, 0700) != nil || os.WriteFile(filepath.Join(name, "writer.lock"), nil, 0600) != nil {
			t.Fatal("owned child pair")
		}
	}
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	childCtx, end := context.WithTimeout(ctx, 15*time.Second)
	cmd := exec.CommandContext(childCtx, exe, "-test.v", "-test.run=^"+t.Name()+"$")
	cmd.Env = append(os.Environ(), "HORVIUM_JOINT_CHILD=1", "HORVIUM_JOINT_JOURNAL="+cj, "HORVIUM_JOINT_LEDGER="+cl)
	out, e := cmd.CombinedOutput()
	end()
	check("kernel_uid_denial", e == nil && strings.Contains(string(out), "joint-kernel-uid-denied"))
	check("child_reaped", cmd.ProcessState != nil && cmd.ProcessState.Success())
	if os.RemoveAll(cj) != nil || os.RemoveAll(cl) != nil {
		t.Fatal("owned child pair removal")
	}
	p = open()
	again, e = observeNativeExecutionDomains(ctx, p, installation)
	check("reopen_new_origin", e == nil && again != (storage.ExecutionDomainsObservation{}) && again != first)
	entered, release := make(chan struct{}), make(chan struct{})
	result := make(chan error, 1)
	go func() {
		once := true
		_, e := p.Observe(ctx, installation, func(c context.Context, fd int) (storage.ExecutionBackingObservation, error) {
			if once {
				once = false
				close(entered)
				<-release
			}
			return readNativeExecutionBacking(c, fd)
		})
		result <- e
	}()
	<-entered
	started, closed := make(chan struct{}), make(chan error, 1)
	go func() { close(started); closed <- p.Close() }()
	<-started
	blocked := false
	select {
	case <-closed:
	case <-time.After(30 * time.Millisecond):
		blocked = true
	}
	close(release)
	check("close_excluded", blocked)
	check("observation_finishes", <-result == nil)
	check("close_finishes", <-closed == nil)
	check("closed_denial", deny(p, ctx, installation, never) != nil && !called)
	check("close_idempotent", p.Close() == nil)
	for _, name := range []string{journal, ledger} {
		free, e := storage.OpenStateDirectory(name)
		if e != nil {
			t.Fatal("closed pair retained lock", e)
		}
		if free.Close() != nil {
			t.Fatal("released fixture close")
		}
	}
	check("both_locks_released", true)
	if kind == "zfs" {
		same := filepath.Join(journal, "same")
		sj, sl := filepath.Join(same, "journal"), filepath.Join(same, "ledger")
		if os.Mkdir(same, 0700) != nil {
			t.Fatal("same dataset root")
		}
		for _, name := range []string{sj, sl} {
			if os.Mkdir(name, 0700) != nil || os.WriteFile(filepath.Join(name, "writer.lock"), nil, 0600) != nil {
				t.Fatal("same dataset pair")
			}
		}
		q, e = openNativeExecutionDomains(ctx, sj, sl)
		if e != nil {
			t.Fatal("owned same-dataset open", e)
		}
		o, e := observeNativeExecutionDomains(ctx, q, installation)
		check("same_dataset_denial", e != nil && o == (storage.ExecutionDomainsObservation{}))
		if q.Close() != nil || os.RemoveAll(same) != nil {
			t.Fatal("same dataset cleanup")
		}
	}
	t.Log("JOINT_EXECUTION_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
}
