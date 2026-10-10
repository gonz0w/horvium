// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

func TestNativeExecutionSystemUFS(t *testing.T) { nativeExecutionSystemParent(t, "ufs") }
func TestNativeExecutionSystemZFS(t *testing.T) { nativeExecutionSystemParent(t, "zfs") }

func nativeExecutionSystemParent(t *testing.T, kind string) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PROPERTY_VM") != "1" || os.Getenv("HORVIUM_EXECUTION_SYSTEM_VM") != "1" {
		t.Skip("owned execution system fixture required")
	}
	ctx, stop := context.WithTimeout(context.Background(), 8*time.Minute)
	defer stop()
	cleaned := false
	nativePropertySystemFixtureWithClaim(t, ctx, func(name string, ok bool) {
		if name != "claim_fixture_teardown" || !ok || cleaned {
			t.Fatal("owned data fixture teardown", name)
		}
		cleaned = true
	}, func(policy storage.Policy, ordinary string, disk inventory.Provider) {
		nativeExecutionSystemFixture(t, ctx, kind, policy, ordinary, disk)
	})
	if !cleaned {
		t.Fatal("data fixture remains")
	}
	t.Log("EXECUTION_SYSTEM_DATA_TEARDOWN " + kind + " restored=pass")
}

// The existing declared fixture independently excludes root/mount/swap and
// verifies all five snapshot disks before supplying this metadata disk.
func nativeExecutionSystemFixture(t *testing.T, ctx context.Context, kind string, policy storage.Policy, ordinary string, disk inventory.Provider) {
	t.Helper()
	command := func(c context.Context, path string, args ...string) (string, error) {
		out, e := execute(c, path, args, 1<<20)
		return string(out.data), e
	}
	must := func(path string, args ...string) string {
		t.Helper()
		out, e := command(ctx, path, args...)
		if e != nil {
			t.Fatal("owned execution fixture command", path, e)
		}
		return out
	}
	if disk.Class != "DISK" || disk.Kind != inventory.DiskCandidate || disk.MediaSize != 64<<20 || !strings.HasPrefix(disk.Name, "vtbd") || strings.ContainsAny(disk.Name, "/ \\\t\r\n") {
		t.Fatal("undeclared metadata disk")
	}
	mountsBefore := must("/sbin/mount", "-p")
	sentinel, e := os.ReadFile(filepath.Join(ordinary, "sentinel"))
	if e != nil {
		t.Fatal(e)
	}
	anchor, e := os.MkdirTemp("/root", "horvium-execution-system-")
	if e != nil {
		t.Fatal(e)
	}
	pool := "horvium_exec_" + strings.TrimPrefix(filepath.Base(anchor), "horvium-execution-system-")
	created, mounted := false, false
	guid := ""
	var pair *storage.ExecutionDomainLeases
	jp, lp := filepath.Join(anchor, "journal"), filepath.Join(anchor, "ledger")
	clean := func() bool {
		c, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if pair != nil {
			if pair.Close() != nil {
				return false
			}
			pair = nil
		}
		if created {
			actual, e := command(c, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)
			if e != nil || guid == "" || strings.TrimSpace(actual) != guid {
				return false
			}
			if _, e = command(c, "/sbin/zpool", "destroy", pool); e != nil {
				return false
			}
			created = false
			if _, e = command(c, "/sbin/zpool", "list", "-H", pool); e == nil {
				return false
			}
		}
		if mounted {
			current, e := command(c, "/sbin/mount", "-p")
			if e != nil || !strings.Contains(current, "/dev/"+disk.Name+"\t"+anchor+"\t") && !strings.Contains(current, "/dev/"+disk.Name+" "+anchor+" ") {
				return false
			}
			if _, e = command(c, "/sbin/umount", anchor); e != nil {
				return false
			}
			mounted = false
		}
		if os.RemoveAll(anchor) != nil {
			return false
		}
		_, e := os.Lstat(anchor)
		return errors.Is(e, os.ErrNotExist)
	}
	cleaned := false
	defer func() {
		if !cleaned && !clean() {
			t.Error("owned system backing teardown")
		}
	}()
	if kind == "ufs" {
		must("/sbin/newfs", "-O", "2", "-m", "0", "-u", "/dev/"+disk.Name)
		must("/sbin/mount", "-t", "ufs", "/dev/"+disk.Name, anchor)
		mounted = true
		if os.Chmod(anchor, 0700) != nil {
			t.Fatal("owned UFS mode")
		}
	} else {
		if _, e := command(ctx, "/sbin/zpool", "list", "-H", pool); e == nil {
			t.Fatal("fixture pool exists")
		}
		must("/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", "-o", "ashift=12", pool, "/dev/"+disk.Name)
		created = true
		guid = strings.TrimSpace(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool))
	}
	for _, path := range []string{jp, lp} {
		if kind == "zfs" {
			must("/sbin/zfs", "create", "-o", "mountpoint="+path, pool+"/"+filepath.Base(path))
		} else if os.Mkdir(path, 0700) != nil {
			t.Fatal("owned domain directory")
		}
		if os.Chmod(path, 0700) != nil || os.WriteFile(filepath.Join(path, "writer.lock"), nil, 0600) != nil {
			t.Fatal("owned existing domain")
		}
	}
	policyBytes, e := storage.EncodeOwnershipRecord(policy)
	if e != nil {
		t.Fatal(e)
	}
	record := filepath.Join(lp, "ownership-"+policy.PoolGUID+".json")
	if os.WriteFile(record, policyBytes, 0600) != nil {
		t.Fatal("owned policy bytes")
	}
	installation := strings.Repeat("1", 32)
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("execution system assertion", kind, name)
		}
		markers = append(markers, name+"=pass")
	}
	open := func() {
		t.Helper()
		if pair != nil && pair.Close() != nil {
			t.Fatal("pair close")
		}
		pair, e = openNativeExecutionDomains(ctx, jp, lp)
		if e != nil {
			t.Fatal("pair open", e)
		}
	}
	open()
	check("pair_open", pair != nil)
	first, e := observeNativeExecutionSystemDomains(ctx, pair, installation)
	check("actual_system_pair", e == nil && first != (storage.ExecutionDomainsObservation{}))
	again, e := observeNativeExecutionSystemDomains(ctx, pair, installation)
	check("repeat", e == nil && again == first)
	plain, e := observeNativeExecutionDomains(ctx, pair, installation)
	check("ordinary_observation_separate", e == nil && plain != first)
	source, e := NewSystemUseSource()
	if e != nil {
		t.Fatal(e)
	}
	use, e := source.Collect(ctx)
	if e != nil {
		t.Fatal(e)
	}
	systemMounts := use.System().Mounts
	disjoint := false
	for _, m := range systemMounts {
		if m.Target == anchor && kind == "ufs" || (m.Target == jp || m.Target == lp) && kind == "zfs" {
			disjoint = len(m.Disks) == 1 && m.Disks[0].Name == disk.Name
			for _, p := range use.Pools() {
				if p.GUID == policy.PoolGUID {
					for _, d := range p.Disks {
						disjoint = disjoint && d.Name != disk.Name
					}
				}
			}
		}
	}
	check("actual_disjoint_backing", disjoint)
	deny := func(c context.Context, owner *storage.ExecutionDomainLeases, system func(context.Context) (storage.PoolSystemUse, error)) error {
		t.Helper()
		got, e := owner.ObserveSystem(c, installation, readNativeExecutionBacking, system)
		if e == nil || got != (storage.ExecutionDomainsObservation{}) {
			t.Fatal("unsafe system observation", e)
		}
		return e
	}
	called := false
	counted := func(context.Context) (storage.PoolSystemUse, error) { called = true; return use, nil }
	check("nil_system", deny(ctx, pair, nil) != nil)
	check("nil_context", deny(nil, pair, counted) != nil && !called)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	check("cancelled_before", errors.Is(deny(cancelled, pair, counted), context.Canceled) && !called)
	copyPair := new(storage.ExecutionDomainLeases)
	reflect.ValueOf(copyPair).Elem().Set(reflect.ValueOf(pair).Elem())
	check("copied_owner", deny(ctx, copyPair, counted) != nil && !called)
	// A root-created unresolved ownership record must not disappear from exclusion.
	missing := storage.Policy{Revision: 1, PoolGUID: "999", PoolName: "unresolved_fixture"}
	missingBytes, e := storage.EncodeOwnershipRecord(missing)
	if e != nil {
		t.Fatal(e)
	}
	missingPath := filepath.Join(lp, "ownership-999.json")
	if os.WriteFile(missingPath, missingBytes, 0600) != nil {
		t.Fatal("owned unresolved policy")
	}
	got, e := observeNativeExecutionSystemDomains(ctx, pair, installation)
	check("unresolved_owner_denied", e != nil && got == (storage.ExecutionDomainsObservation{}))
	if os.Remove(missingPath) != nil {
		t.Fatal("owned policy removal")
	}
	got, e = observeNativeExecutionSystemDomains(ctx, pair, installation)
	check("unresolved_retired", errors.Is(e, storage.ErrStateUncertain) && got == (storage.ExecutionDomainsObservation{}))
	open()
	got, e = observeNativeExecutionSystemDomains(ctx, pair, installation)
	check("reopen_observation_only", e == nil && got != first)
	check("system_error", deny(ctx, pair, func(context.Context) (storage.PoolSystemUse, error) { return use, storage.ErrUnavailable }) != nil)
	check("error_retired", errors.Is(deny(ctx, pair, counted), storage.ErrStateUncertain))
	open()
	check("zero_system_denied", deny(ctx, pair, func(context.Context) (storage.PoolSystemUse, error) { return storage.PoolSystemUse{}, nil }) != nil)
	open()
	step := 0
	check("final_system_error", deny(ctx, pair, func(c context.Context) (storage.PoolSystemUse, error) {
		step++
		if step == 2 {
			return use, storage.ErrUnavailable
		}
		return source.Collect(c)
	}) != nil && step == 2)
	open()
	step = 0
	revised := policy
	revised.Revision++
	revisedBytes, e := storage.EncodeOwnershipRecord(revised)
	if e != nil {
		t.Fatal(e)
	}
	check("system_phase_ledger_drift", errors.Is(deny(ctx, pair, func(c context.Context) (storage.PoolSystemUse, error) {
		u, e := source.Collect(c)
		step++
		if step == 2 && os.WriteFile(record, revisedBytes, 0600) != nil {
			t.Fatal("owned same-inode record drift")
		}
		return u, e
	}), storage.ErrDrift) && step == 2)
	if os.WriteFile(record, policyBytes, 0600) != nil {
		t.Fatal("owned original record restore")
	}
	open()
	// Full system reads and the final census remain inside both original locks.
	entered, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	step = 0
	go func() {
		_, e := pair.ObserveSystem(ctx, installation, readNativeExecutionBacking, func(c context.Context) (storage.PoolSystemUse, error) {
			step++
			if step == 1 {
				close(entered)
				<-release
			}
			return source.Collect(c)
		})
		done <- e
	}()
	<-entered
	closed := make(chan error, 1)
	go func() { closed <- pair.Close() }()
	blocked := false
	select {
	case <-closed:
	case <-time.After(30 * time.Millisecond):
		blocked = true
	}
	close(release)
	check("close_excluded_system", blocked)
	check("system_observation_finishes", <-done == nil)
	check("close_finishes", <-closed == nil)
	pair = nil
	if kind == "zfs" {
		// Actual imported metadata pool ownership denies even with ordinary ledger facts.
		metadataPolicy := storage.Policy{Revision: 1, PoolGUID: guid, PoolName: pool}
		b, e := storage.EncodeOwnershipRecord(metadataPolicy)
		if e != nil {
			t.Fatal(e)
		}
		f := filepath.Join(lp, "ownership-"+guid+".json")
		if os.WriteFile(f, b, 0600) != nil {
			t.Fatal("owned metadata enrollment")
		}
		open()
		got, e = observeNativeExecutionSystemDomains(ctx, pair, installation)
		check("owned_metadata_pool_denied", e != nil && got == (storage.ExecutionDomainsObservation{}))
		if pair.Close() != nil || os.Remove(f) != nil {
			t.Fatal("owned metadata restore")
		}
		pair = nil
		for _, role := range []string{"journal", "ledger"} {
			outside, e := os.MkdirTemp("", "horvium-execution-data-")
			if e != nil {
				t.Fatal(e)
			}
			// Move only the owned dataset to an ordinary non-system mount; existing
			// protected paths are remounted before acquiring the test pair again.
			path := jp
			if role == "ledger" {
				path = lp
			}
			must("/sbin/zfs", "set", "mountpoint="+outside, pool+"/"+role)
			old := path
			if role == "journal" {
				jp = outside
			} else {
				lp = outside
			}
			open()
			got, e = observeNativeExecutionSystemDomains(ctx, pair, installation)
			check(role+"_data_mount_denied", e != nil && got == (storage.ExecutionDomainsObservation{}))
			if pair.Close() != nil {
				t.Fatal("moved pair close")
			}
			pair = nil
			must("/sbin/zfs", "set", "mountpoint="+old, pool+"/"+role)
			if role == "journal" {
				jp = old
			} else {
				lp = old
			}
			if os.Remove(outside) != nil {
				t.Fatal("owned outside mount cleanup")
			}
		}
	}
	after, e := os.ReadFile(record)
	check("ledger_preserved", e == nil && bytes.Equal(after, policyBytes))
	after, e = os.ReadFile(filepath.Join(ordinary, "sentinel"))
	check("data_sentinel_preserved", e == nil && bytes.Equal(after, sentinel))
	check("domain_teardown", clean())
	cleaned = true
	check("mounts_restored", must("/sbin/mount", "-p") == mountsBefore)
	t.Log("EXECUTION_SYSTEM_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
	t.Log("EXECUTION_SYSTEM_DOMAIN_TEARDOWN " + kind + " filesystems=pass directories=pass")
}
