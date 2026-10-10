// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var restartOwnerCases = map[string][]string{
	"clean":       {"nil_no_creation", "cancelled_no_creation", "original_registered", "both_locks_held", "fresh_active_identity", "binding_census_preserved", "original_observation", "copied_denied", "decoded_denied", "cancelled_no_effect", "other_startup_denied", "other_commission_denied", "finish_denied", "finish_retires", "release_once", "active_preserved", "registration_not_reset"},
	"touch":       {"original_registered", "directory_drift_denied", "observation_retired", "release_once", "active_preserved", "registration_not_reset"},
	"uid":         {"original_registered", "actual_nonroot", "inherited_root_descriptors", "inherited_operations_denied", "active_preserved", "nonroot_release_denied"},
	"active":      {"active_denied", "active_preserved", "both_locks_released", "registration_not_reset"},
	"missing":     {"missing_denied", "no_creation", "registration_not_reset", "retry_denied"},
	"boot":        {"fault_denied", "closed_preserved", "both_locks_released", "registration_not_reset"},
	"system":      {"fault_denied", "closed_preserved", "both_locks_released", "registration_not_reset"},
	"post-system": {"fault_denied", "fresh_active_preserved", "both_locks_released", "registration_not_reset"},
	"post-touch":  {"fault_denied", "fresh_active_preserved", "both_locks_released", "registration_not_reset"},
}

func TestClosedStartupOwnerChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_RESTART_OWNER_CHILD")
	if mode == "" {
		t.Skip("owned restart child only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_CLOSED_STARTUP_OWNER_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("owned root child required")
	}
	names, ok := restartOwnerCases[mode]
	if !ok {
		t.Fatal("unknown mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	j, l := os.Getenv("HORVIUM_RESTART_JOURNAL"), os.Getenv("HORVIUM_RESTART_LEDGER")
	var seen []string
	check := func(n string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("restart owner", n)
		}
		seen = append(seen, n)
	}
	alternative := func() bool {
		q, e := commissionNativeExecutionOwner(ctx, j+"-other", l+"-other")
		return q == nil && e != nil && !activationPathExists(j+"-other") && !activationPathExists(l+"-other")
	}
	if mode == "missing" {
		q, e := activateNativeClosedExecutionOwner(ctx, j, l)
		check("missing_denied", q == nil && e != nil)
		check("no_creation", !activationPathExists(j) && !activationPathExists(l))
		check("registration_not_reset", alternative())
		q, e = activateNativeClosedExecutionOwner(ctx, j, l)
		check("retry_denied", q == nil && e != nil)
	} else {
		before, old := readActivationFixture(t, j)
		if mode == "active" {
			q, e := activateNativeClosedExecutionOwner(ctx, j, l)
			check("active_denied", q == nil && e != nil)
			after, _ := readActivationFixture(t, j)
			check("active_preserved", bytes.Equal(before, after))
			check("both_locks_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
			check("registration_not_reset", alternative())
		} else if mode == "boot" || mode == "system" || strings.HasPrefix(mode, "post-") {
			source, e := NewSystemUseSource()
			if e != nil {
				t.Fatal(e)
			}
			bootCount := 0
			boot := func(c context.Context) ([16]byte, error) {
				v, e := readNativeActivationBoot(c)
				bootCount++
				if mode == "boot" && bootCount == 3 && e == nil {
					v[0] ^= 1
				}
				return v, e
			}
			system := func(c context.Context) (storage.PoolSystemUse, error) {
				v, e := source.Collect(c)
				_, r := readActivationFixture(t, j)
				if e == nil {
					if mode == "system" && bootCount >= 3 {
						return storage.PoolSystemUse{}, storage.ErrUnavailable
					}
					if r.Owner.State == "active" {
						switch mode {
						case "post-system":
							return storage.PoolSystemUse{}, storage.ErrUnavailable
						case "post-touch":
							e = os.Chmod(l, 0700)
						}
					}
				}
				return v, e
			}
			q, e := commissionExecutionOwner(ctx, nativeExecutionOwners, func() bool { return os.Geteuid() == 0 }, func(c context.Context) (executionOwnerSource, error) {
				r, e := storage.ReviewExecutionStartup(c, j, l, readNativeExecutionBacking, system, boot)
				if e != nil {
					return executionOwnerSource{}, e
				}
				a, e := r.Activate(c)
				if e != nil {
					return executionOwnerSource{}, errors.Join(e, r.Close())
				}
				return executionOwnerSource{observe: a.Observe, finish: func(context.Context) error { return storage.ErrDenied }, release: a.Close}, nil
			})
			check("fault_denied", q == nil && e != nil)
			after, r := readActivationFixture(t, j)
			if strings.HasPrefix(mode, "post-") {
				check("fresh_active_preserved", r.Owner.State == "active" && r.Generation == old.Generation+1 && r.Owner.ID != old.Owner.ID && r.Binding == old.Binding && r.Census == old.Census && !bytes.Equal(before, after))
			} else {
				check("closed_preserved", bytes.Equal(before, after))
			}
			check("both_locks_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
			check("registration_not_reset", alternative())
		} else {
			if mode == "clean" {
				cancelled, stop := context.WithCancel(ctx)
				stop()
				for _, x := range []struct {
					name string
					c    context.Context
				}{{"nil", nil}, {"cancelled", cancelled}} {
					q, e := activateNativeClosedExecutionOwner(x.c, j, l)
					check(x.name+"_no_creation", q == nil && e != nil)
				}
			}
			o, e := activateNativeClosedExecutionOwner(ctx, j, l)
			check("original_registered", e == nil && o != nil)
			defer o.Close()
			active, r := readActivationFixture(t, j)
			switch mode {
			case "clean":
				check("both_locks_held", nativeActivationLockHeld(t, j) && nativeActivationLockHeld(t, l))
				raw, e := readNativeBoot()
				check("fresh_active_identity", e == nil && r.Generation == old.Generation+1 && r.Owner.State == "active" && r.Owner.ID != old.Owner.ID && r.Owner.ID != old.Binding.Installation && r.Owner.Boot == hex.EncodeToString(raw[:]))
				check("binding_census_preserved", r.Binding == old.Binding && r.Census == old.Census && !r.Metadata.Pending && !r.Native.Pending && !r.Disarmed)
				_, e = o.Observe(ctx)
				check("original_observation", e == nil)
				copy := new(commissionedExecutionOwner)
				reflect.ValueOf(copy).Elem().Set(reflect.ValueOf(o).Elem())
				_, e = copy.Observe(ctx)
				check("copied_denied", e != nil && copy.Close() != nil)
				wire, e := json.Marshal(o)
				var decoded commissionedExecutionOwner
				de := json.Unmarshal(wire, &decoded)
				_, ce := decoded.Observe(ctx)
				check("decoded_denied", e == nil && de == nil && string(wire) == "{}" && ce != nil && decoded.Close() != nil)
				cancelled, stop := context.WithCancel(ctx)
				stop()
				_, e = o.Observe(cancelled)
				_, oe := o.Observe(ctx)
				check("cancelled_no_effect", e != nil && oe == nil)
				q, e := reviewNativeExecutionStartupOwner(ctx, j+"-other", l+"-other")
				check("other_startup_denied", q == nil && e != nil && !activationPathExists(j+"-other") && !activationPathExists(l+"-other"))
				check("other_commission_denied", alternative())
				check("finish_denied", o.FinishIdle(ctx) != nil)
				_, e = o.Observe(ctx)
				check("finish_retires", e != nil)
				check("release_once", o.Close() == nil && o.Close() == nil)
				after, _ := readActivationFixture(t, j)
				check("active_preserved", bytes.Equal(after, active))
				check("registration_not_reset", alternative())
			case "touch":
				if os.Chmod(l, 0700) != nil {
					t.Fatal("touch")
				}
				_, e = o.Observe(ctx)
				check("directory_drift_denied", e != nil)
				_, e = o.Observe(ctx)
				check("observation_retired", e != nil)
				check("release_once", o.Close() == nil && o.Close() == nil)
				after, _ := readActivationFixture(t, j)
				check("active_preserved", bytes.Equal(after, active))
				check("registration_not_reset", alternative())
			case "uid":
				held, e := os.Open(filepath.Join(j, "execution.json"))
				if e != nil {
					t.Fatal(e)
				}
				defer held.Close()
				if unix.Setgroups([]int{}) != nil || unix.Setgid(65534) != nil || unix.Setuid(65534) != nil {
					t.Fatal("uid")
				}
				check("actual_nonroot", os.Geteuid() == 65534)
				var st unix.Stat_t
				check("inherited_root_descriptors", unix.Fstat(int(held.Fd()), &st) == nil && st.Uid == 0)
				_, e = o.Observe(ctx)
				check("inherited_operations_denied", errors.Is(e, ErrPermission) && o.FinishIdle(ctx) != nil)
				after, e := io.ReadAll(held)
				check("active_preserved", e == nil && bytes.Equal(after, active))
				check("nonroot_release_denied", errors.Is(o.Close(), ErrPermission) && errors.Is(o.Close(), ErrPermission))
			}
		}
	}
	if !reflect.DeepEqual(seen, names) {
		t.Fatal("incomplete controls", seen)
	}
	fmt.Println("CLOSED_STARTUP_OWNER_CHILD_RESULT " + mode + " " + strings.Join(seen, " "))
}
func TestNativeClosedStartupOwnerZFS(t *testing.T) { nativeClosedStartupOwnerParent(t, "zfs") }
func TestNativeClosedStartupOwnerUFS(t *testing.T) { nativeClosedStartupOwnerParent(t, "ufs") }
func nativeClosedStartupOwnerParent(t *testing.T, kind string) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PROPERTY_VM") != "1" || os.Getenv("HORVIUM_CLOSED_STARTUP_OWNER_VM") != "1" {
		t.Skip("owned restart owner fixture required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	cleaned := false
	nativePropertySystemFixtureWithClaim(t, ctx, func(name string, ok bool) {
		if name != "claim_fixture_teardown" || !ok || cleaned {
			t.Fatal("owned data fixture teardown", name)
		}
		cleaned = true
	}, func(policy storage.Policy, ordinary string, disk inventory.Provider) {
		nativeClosedStartupOwnerFixture(t, ctx, kind, policy, ordinary, disk)
	})
	if !cleaned {
		t.Fatal("data fixture remains")
	}
	t.Log("CLOSED_STARTUP_OWNER_DATA_TEARDOWN " + kind + " restored=pass")
}
func nativeClosedStartupOwnerFixture(t *testing.T, ctx context.Context, kind string, policy storage.Policy, ordinary string, disk inventory.Provider) {
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
			t.Fatal("restart owner fixture", kind, name)
		}
		for _, old := range markers {
			if old == name+"=pass" {
				t.Fatal("duplicate startup control")
			}
		}
		markers = append(markers, name+"=pass")
	}
	j, l := filepath.Join(jp, "closed"), filepath.Join(lp, "closed")
	a, e := activateNativeFreshExecutionJournal(ctx, j, l)
	if e != nil {
		t.Fatal("closed seed", e)
	}
	if a.FinishIdle(ctx) != nil || a.Close() != nil {
		t.Fatal("original closed seed")
	}
	closed, cr := readActivationFixture(t, j)
	raw, e := readNativeBoot()
	check("seed_closed_receipt", e == nil && cr.Generation == 2 && cr.Owner.State == "closed" && cr.Owner.Boot == hex.EncodeToString(raw[:]) && !cr.Metadata.Pending && !cr.Native.Pending && !cr.Disarmed)
	ja, la := filepath.Join(jp, "active"), filepath.Join(lp, "active")
	a, e = activateNativeFreshExecutionJournal(ctx, ja, la)
	if e != nil {
		t.Fatal("active seed", e)
	}
	if a.Close() != nil {
		t.Fatal("active release")
	}
	active, ar := readActivationFixture(t, ja)
	check("seed_active_receipt", ar.Generation == 1 && ar.Owner.State == "active")
	var jf, lf unix.Statfs_t
	if unix.Statfs(j, &jf) != nil || unix.Statfs(l, &lf) != nil {
		t.Fatal("independent metadata mounts")
	}
	jm, lm := persistentMount(jf), persistentMount(lf)
	check("journal_ledger_independent", kind == "ufs" && jm.source == lm.source && j != l || kind == "zfs" && jm.source != lm.source && !strings.HasPrefix(jm.source, lm.source+"/") && !strings.HasPrefix(lm.source, jm.source+"/"))
	source, e := NewSystemUseSource()
	if e != nil {
		t.Fatal(e)
	}
	use, e := source.Collect(ctx)
	if e != nil {
		t.Fatal(e)
	}
	disjoint := false
	for _, m := range use.System().Mounts {
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
	check("declared_disk_disjoint", disjoint)
	pair, e = openNativeExecutionDomains(ctx, j, l)
	if e != nil {
		t.Fatal(e)
	}
	got, e := observeNativeExecutionSystemDomains(ctx, pair, cr.Binding.Installation)
	check("closed_observation_metadata_only", e == nil && got != (storage.ExecutionDomainsObservation{}))
	if pair.Close() != nil {
		t.Fatal("ordinary metadata close")
	}
	pair = nil

	for _, mode := range []string{"clean", "touch", "uid", "active", "missing", "boot", "system", "post-system", "post-touch"} {
		cj, cl := filepath.Join(jp, "restart-"+mode), filepath.Join(lp, "restart-"+mode)
		var before []byte
		if mode != "missing" {
			seed, e := activateNativeFreshExecutionJournal(ctx, cj, cl)
			if e != nil {
				t.Fatal(e)
			}
			if mode != "active" && seed.FinishIdle(ctx) != nil {
				t.Fatal("closed child seed")
			}
			if seed.Close() != nil {
				t.Fatal("seed release")
			}
			before, _ = readActivationFixture(t, cj)
		}
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestClosedStartupOwnerChild$", "-test.timeout=150s")
		cmd.Env = append(os.Environ(), "HORVIUM_RESTART_OWNER_CHILD="+mode, "HORVIUM_RESTART_JOURNAL="+cj, "HORVIUM_RESTART_LEDGER="+cl)
		data, e := cmd.CombinedOutput()
		if e != nil {
			limit := len(data)
			if limit > 16<<10 {
				limit = 16 << 10
			}
			t.Logf("owned restart child %s: %s", mode, data[:limit])
		}
		check(mode+"_child_waited", e == nil && cmd.ProcessState != nil && cmd.ProcessState.Success() && len(data) <= 16<<10)
		prefix := "CLOSED_STARTUP_OWNER_CHILD_RESULT " + mode + " "
		found := false
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, prefix) {
				if found || strings.TrimPrefix(line, prefix) != strings.Join(restartOwnerCases[mode], " ") {
					t.Fatal("child inventory")
				}
				found = true
			}
		}
		if !found {
			t.Fatal("missing child inventory")
		}
		for _, name := range restartOwnerCases[mode] {
			check(mode+"_"+name, true)
		}
		if mode != "missing" {
			check(mode+"_parent_locks_released", !nativeActivationLockHeld(t, cj) && !nativeActivationLockHeld(t, cl))
			history, r := readActivationFixture(t, cj)
			var old nativeActivationRecord
			if json.Unmarshal(before, &old) != nil {
				t.Fatal("old history")
			}
			changed := mode == "clean" || mode == "touch" || mode == "uid" || strings.HasPrefix(mode, "post-")
			if changed {
				check(mode+"_parent_history", r.Owner.State == "active" && r.Generation == old.Generation+1 && r.Owner.ID != old.Owner.ID && r.Owner.Boot == old.Owner.Boot && r.Binding == old.Binding && r.Census == old.Census && !r.Disarmed && !r.Metadata.Pending && !r.Native.Pending)
			} else {
				check(mode+"_parent_history", bytes.Equal(history, before))
			}
		}
	}

	after, _ := readActivationFixture(t, j)
	check("closed_history_final", bytes.Equal(after, closed))
	after, _ = readActivationFixture(t, ja)
	check("active_history_final", bytes.Equal(after, active))
	afterSentinel, e := os.ReadFile(filepath.Join(ordinary, "sentinel"))
	check("data_sentinel_preserved", e == nil && bytes.Equal(afterSentinel, sentinel))
	cleaned = clean()
	check("domain_teardown", cleaned)
	check("mounts_restored", must("/sbin/mount", "-p") == mountsBefore)
	t.Log("CLOSED_STARTUP_OWNER_DOMAIN_TEARDOWN " + kind + " filesystems=pass directories=pass")
	t.Log("CLOSED_STARTUP_OWNER_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
}
