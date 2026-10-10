//go:build freebsd && horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var policyOwnerModes = []string{"clean", "clean-again", "legacy", "member", "scope", "missing-pool", "late", "uid"}
var policyOwnerSuccess = []string{"registered", "leases-held", "original-observation", "copied-denied", "decoded-denied", "closed", "retired", "released", "registration-consumed"}
var policyOwnerDenial = []string{"denied", "preserved", "released", "registration-consumed"}
var policyOwnerLate = []string{"denied", "active-preserved", "released", "registration-consumed"}
var policyOwnerUID = []string{"registered", "actual-nonroot", "inherited-root", "operations-denied", "active-preserved", "released"}

func policyOwnerInventory(mode string) []string {
	switch mode {
	case "clean", "clean-again":
		return policyOwnerSuccess
	case "uid":
		return policyOwnerUID
	case "late":
		return policyOwnerLate
	default:
		return policyOwnerDenial
	}
}
func TestExecutionPolicyOwnerChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_POLICY_OWNER_CHILD")
	if mode == "" {
		t.Skip("owned policy child only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_CLOSED_STARTUP_OWNER_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("owned root child required")
	}
	known := false
	for _, m := range policyOwnerModes {
		known = known || m == mode
	}
	if !known {
		t.Fatal("unknown mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	j, l := os.Getenv("HORVIUM_RESTART_JOURNAL"), os.Getenv("HORVIUM_RESTART_LEDGER")
	before, old := readActivationFixture(t, j)
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal(mode, name)
		}
		seen = append(seen, name)
	}
	consumed := func() bool {
		q, e := activateNativeClosedExecutionOwner(ctx, j+"-other", l+"-other")
		return e != nil && q == nil && !activationPathExists(j+"-other") && !activationPathExists(l+"-other")
	}
	var o *commissionedExecutionOwner
	var e error
	if mode == "late" {
		system, se := NewSystemUseSource()
		if se != nil {
			t.Fatal(se)
		}
		source, se := newExecutionStartupPolicySource()
		if se != nil {
			t.Fatal(se)
		}
		wrapped := func(c context.Context, p storage.Policy) (storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse, error) {
			obs, d, u, e := source(c, p)
			_, record := readActivationFixture(t, j)
			if e == nil && record.Owner.State == "active" {
				d[0].Provider.SectorSize = 0
			}
			return obs, d, u, e
		}
		o, e = commissionExecutionOwner(ctx, nativeExecutionOwners, func() bool { return os.Geteuid() == 0 }, func(c context.Context) (executionOwnerSource, error) {
			r, e := storage.ReviewExecutionPolicyStartup(c, j, l, readNativeExecutionBacking, system.Collect, readNativeActivationBoot, wrapped)
			if e != nil {
				return executionOwnerSource{}, e
			}
			a, e := r.Activate(c)
			if e != nil {
				return executionOwnerSource{}, errors.Join(e, r.Close())
			}
			return executionOwnerSource{observe: a.Observe, finish: a.FinishIdle, release: a.Close}, nil
		})
	} else {
		o, e = activateNativeClosedExecutionOwner(ctx, j, l)
	}
	if mode == "clean" || mode == "clean-again" {
		check("registered", e == nil && o != nil)
		defer o.Close()
		check("leases-held", nativeActivationLockHeld(t, j) && nativeActivationLockHeld(t, l))
		_, e = o.Observe(ctx)
		check("original-observation", e == nil)
		copied := new(commissionedExecutionOwner)
		reflect.ValueOf(copied).Elem().Set(reflect.ValueOf(o).Elem())
		_, e = copied.Observe(ctx)
		check("copied-denied", e != nil && copied.FinishIdle(ctx) != nil && copied.Close() != nil)
		wire, e := json.Marshal(o)
		var decoded commissionedExecutionOwner
		check("decoded-denied", e == nil && string(wire) == "{}" && json.Unmarshal(wire, &decoded) == nil && decoded.FinishIdle(ctx) != nil)
		e = o.FinishIdle(ctx)
		_, record := readActivationFixture(t, j)
		check("closed", e == nil && record.Owner.State == "closed" && record.Generation == old.Generation+2 && record.Owner.ID != old.Owner.ID && record.Census == old.Census && record.Binding == old.Binding)
		_, e = o.Observe(ctx)
		check("retired", e != nil && o.FinishIdle(ctx) != nil)
		check("released", o.Close() == nil && o.Close() == nil && !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
		check("registration-consumed", consumed())
	} else if mode == "uid" {
		check("registered", e == nil && o != nil)
		defer o.Close()
		if unix.Setgroups([]int{}) != nil || unix.Setgid(65534) != nil || unix.Setuid(65534) != nil {
			t.Fatal("drop uid")
		}
		check("actual-nonroot", os.Geteuid() == 65534)
		inherited := 0
		for fd := 0; fd < 256; fd++ {
			var st unix.Stat_t
			if unix.Fstat(fd, &st) == nil && st.Uid == 0 && st.Mode&unix.S_IFMT == unix.S_IFDIR {
				inherited++
			}
		}
		check("inherited-root", inherited >= 2)
		_, e = o.Observe(ctx)
		check("operations-denied", e != nil && o.FinishIdle(ctx) != nil)
		check("active-preserved", o.blocked)
		check("released", o.Close() != nil && o.closed)
	} else {
		check("denied", e != nil && o == nil)
		after, record := readActivationFixture(t, j)
		if mode == "late" {
			check("active-preserved", record.Owner.State == "active" && record.Generation == old.Generation+1 && record.Census == old.Census && record.Binding == old.Binding)
		} else {
			check("preserved", bytes.Equal(before, after))
		}
		check("released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
		check("registration-consumed", consumed())
	}
	if !reflect.DeepEqual(seen, policyOwnerInventory(mode)) {
		t.Fatal("incomplete inventory", seen)
	}
	fmt.Println("POLICY_STARTUP_OWNER_CHILD_RESULT " + mode + " " + strings.Join(seen, " "))
}
func TestNativePolicyStartupLifecycleOwnerUFS(t *testing.T) {
	nativePolicyStartupParent(t, "ufs", "lifecycle", []string{"clean", "clean-again", "late", "uid"})
}
func TestNativePolicyStartupLifecycleOwnerZFS(t *testing.T) {
	nativePolicyStartupParent(t, "zfs", "lifecycle", []string{"clean", "clean-again", "late", "uid"})
}
func TestNativePolicyStartupDenialOwnerUFS(t *testing.T) {
	nativePolicyStartupParent(t, "ufs", "denial", []string{"legacy", "member", "scope", "missing-pool"})
}
func TestNativePolicyStartupDenialOwnerZFS(t *testing.T) {
	nativePolicyStartupParent(t, "zfs", "denial", []string{"legacy", "member", "scope", "missing-pool"})
}
func nativePolicyStartupParent(t *testing.T, kind, group string, modes []string) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PROPERTY_VM") != "1" || os.Getenv("HORVIUM_CLOSED_STARTUP_OWNER_VM") != "1" {
		t.Skip("owned policy fixture required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	cleaned := false
	nativePropertySystemFixtureWithClaim(t, ctx, func(name string, ok bool) {
		if name != "claim_fixture_teardown" || !ok || cleaned {
			t.Fatal("data fixture teardown")
		}
		cleaned = true
	}, func(p storage.Policy, ordinary string, disk inventory.Provider) {
		nativePolicyStartupFixture(t, ctx, kind, group, modes, p, ordinary, disk)
	})
	if !cleaned {
		t.Fatal("data fixture remains")
	}
	t.Log("POLICY_STARTUP_DATA_TEARDOWN " + group + " " + kind + " restored=pass")
}
func nativePolicyStartupFixture(t *testing.T, ctx context.Context, kind, group string, modes []string, policy storage.Policy, ordinary string, disk inventory.Provider) {
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
	anchor, e := os.MkdirTemp("/root", "horvium-closed-startup-owner-")
	if e != nil {
		t.Fatal(e)
	}
	pool := "horvium_restart_" + strings.TrimPrefix(filepath.Base(anchor), "horvium-closed-startup-owner-")
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
			var fs unix.Statfs_t
			if unix.Statfs(anchor, &fs) != nil {
				return false
			}
			mount := persistentMount(fs)
			if mount.kind != "ufs" || mount.target != anchor || mount.source != "/dev/"+disk.Name {
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
			t.Fatal("owned metadata parent")
		}
		if os.Chmod(path, 0700) != nil {
			t.Fatal("owned metadata parent mode")
		}
	}
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("policy fixture", kind, name)
		}
		for _, m := range markers {
			if m == name+"=pass" {
				t.Fatal("duplicate")
			}
		}
		markers = append(markers, name+"=pass")
	}
	source, e := NewSystemUseSource()
	if e != nil {
		t.Fatal(e)
	}
	j, l := filepath.Join(jp, "clean"), filepath.Join(lp, "clean")
	for _, mode := range modes {
		cj, cl := j, l
		if mode != "clean-again" {
			cj, cl = filepath.Join(jp, mode), filepath.Join(lp, mode)
			if mode == "clean" {
				j, l = cj, cl
			}
			data, e := storage.EncodeOwnershipRecord(policy)
			if e != nil {
				t.Fatal(e)
			}
			p, e := storage.DecodeOwnershipRecord(data)
			if e != nil {
				t.Fatal(e)
			}
			switch mode {
			case "legacy":
				p.Members = nil
			case "member":
				p.Members[0].Serial = "CHANGED-FIXTURE"
			case "scope":
				p.Scopes[0].GUID = "999"
			case "missing-pool":
				p.PoolGUID = "999"
			}
			if e = storage.SeedExecutionPolicyFixture(ctx, cj, cl, readNativeExecutionBacking, source.Collect, readNativeActivationBoot, []storage.Policy{p}); e != nil && mode != "missing-pool" {
				t.Fatal("original fixture publisher", mode, e)
			}
		}
		before, old := readActivationFixture(t, cj)
		ownership, e := os.ReadFile(filepath.Join(cl, "ownership-"+oldPolicyGUID(mode, policy.PoolGUID)+".json"))
		if e != nil {
			t.Fatal(e)
		}
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestExecutionPolicyOwnerChild$", "-test.timeout=150s")
		cmd.Env = append(os.Environ(), "HORVIUM_POLICY_OWNER_CHILD="+mode, "HORVIUM_RESTART_JOURNAL="+cj, "HORVIUM_RESTART_LEDGER="+cl)
		data, e := cmd.CombinedOutput()
		if e != nil {
			t.Logf("policy child %s: %.16000s", mode, data)
		}
		check(mode+"_child_waited", e == nil && cmd.ProcessState != nil && cmd.ProcessState.Success() && len(data) <= 16<<10)
		prefix := "POLICY_STARTUP_OWNER_CHILD_RESULT " + mode + " "
		found := 0
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, prefix) {
				if strings.TrimPrefix(line, prefix) != strings.Join(policyOwnerInventory(mode), " ") {
					t.Fatal("child inventory")
				}
				found++
			}
		}
		check(mode+"_inventory", found == 1)
		for _, name := range policyOwnerInventory(mode) {
			check(mode+"_"+name, true)
		}
		check(mode+"_parent_leases_released", !nativeActivationLockHeld(t, cj) && !nativeActivationLockHeld(t, cl))
		after, record := readActivationFixture(t, cj)
		state, steps := "closed", uint64(0)
		if mode == "clean" || mode == "clean-again" {
			steps = 2
		} else if mode == "late" || mode == "uid" {
			state, steps = "active", 1
		}
		check(mode+"_parent_history", record.Generation == old.Generation+steps && record.Owner.State == state && record.Census == old.Census && record.Binding == old.Binding && (steps != 0 || bytes.Equal(before, after)))
		afterPolicy, e := os.ReadFile(filepath.Join(cl, "ownership-"+oldPolicyGUID(mode, policy.PoolGUID)+".json"))
		check(mode+"_exact_policy", e == nil && bytes.Equal(ownership, afterPolicy))
	}
	check("data_sentinel_preserved", func() bool {
		data, e := os.ReadFile(filepath.Join(ordinary, "sentinel"))
		return e == nil && bytes.Equal(data, sentinel)
	}())
	cleaned = clean()
	check("domain_teardown", cleaned)
	check("mounts_restored", must("/sbin/mount", "-p") == mountsBefore)
	t.Log("POLICY_STARTUP_DOMAIN_TEARDOWN " + group + " " + kind + " filesystems=pass directories=pass")
	t.Log("POLICY_STARTUP_" + strings.ToUpper(group) + "_OWNER_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
}
func oldPolicyGUID(mode, guid string) string {
	if mode == "missing-pool" {
		return "999"
	}
	return guid
}
