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

var startupNativeCases = map[string][]string{
	"touch":   {"closed_registered", "ledger_touch_denied", "drift_retired", "finish_denied", "release", "closed_history_preserved", "both_locks_released", "no_commission_after_failure"},
	"clean":   {"nil_no_creation", "cancelled_no_creation", "closed_registered", "both_locks_held", "original_observation", "repeated_observation", "copied_denied", "decoded_denied", "cancelled_observe_no_effect", "other_startup_no_creation", "other_commission_no_creation", "finish_denied", "finished_observe_denied", "closed_both_locks_released", "closed_registration_not_reset", "closed_history_preserved"},
	"active":  {"active_denied", "active_history_preserved", "both_locks_released", "no_commission_after_failure"},
	"missing": {"missing_denied", "no_creation", "no_commission_after_failure", "no_retry_creation"},
	"drift":   {"closed_registered", "ledger_drift_denied", "drift_retired", "finish_denied", "release", "closed_history_preserved", "both_locks_released", "no_commission_after_failure"},
	"uid":     {"closed_registered", "actual_nonroot", "inherited_root_descriptors", "inherited_operations_denied", "closed_history_preserved", "nonroot_release_denied"},
	"boot":    {"injected_error_denied", "closed_history_preserved", "both_locks_released", "no_commission_after_failure"},
	"system":  {"injected_error_denied", "closed_history_preserved", "both_locks_released", "no_commission_after_failure"},
	"late":    {"injected_error_denied", "closed_history_preserved", "both_locks_released", "no_commission_after_failure"},
}

// Separate exact-binary processes retain the production registry. No fixture
// resets registration. All paths and credential changes belong to this VM.
func TestNativeExecutionStartupChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_STARTUP_CHILD")
	if mode == "" {
		t.Skip("owned startup child only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_EXECUTION_STARTUP_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("owned root startup child required")
	}
	names, ok := startupNativeCases[mode]
	if !ok {
		t.Fatal("unknown owned child")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	j, l := os.Getenv("HORVIUM_STARTUP_JOURNAL"), os.Getenv("HORVIUM_STARTUP_LEDGER")
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("startup", name)
		}
		for _, old := range seen {
			if old == name {
				t.Fatal("duplicate assertion")
			}
		}
		seen = append(seen, name)
	}
	alternative := func() bool {
		q, e := commissionNativeExecutionOwner(ctx, j+"-other", l+"-other")
		return q == nil && e != nil && !activationPathExists(j+"-other") && !activationPathExists(l+"-other")
	}
	if mode == "missing" {
		q, e := reviewNativeExecutionStartupOwner(ctx, j, l)
		check("missing_denied", q == nil && e != nil)
		check("no_creation", !activationPathExists(j) && !activationPathExists(l))
		check("no_commission_after_failure", alternative())
		q, e = reviewNativeExecutionStartupOwner(ctx, j, l)
		check("no_retry_creation", q == nil && e != nil && !activationPathExists(j) && !activationPathExists(l))
	} else {
		before, _ := readActivationFixture(t, j)
		if mode == "active" {
			q, e := reviewNativeExecutionStartupOwner(ctx, j, l)
			check("active_denied", q == nil && e != nil)
			after, _ := readActivationFixture(t, j)
			check("active_history_preserved", bytes.Equal(before, after))
			check("both_locks_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
			check("no_commission_after_failure", alternative())
		} else if mode == "boot" || mode == "system" || mode == "late" {
			source, e := NewSystemUseSource()
			if e != nil {
				t.Fatal(e)
			}
			boots, systems := 0, 0
			boot := func(c context.Context) ([16]byte, error) {
				v, e := readNativeActivationBoot(c)
				boots++
				if mode == "boot" && boots == 2 && e == nil {
					v[0] ^= 1
				}
				return v, e
			}
			system := func(c context.Context) (storage.PoolSystemUse, error) {
				v, e := source.Collect(c)
				systems++
				if mode == "system" && systems == 2 && e == nil {
					return v, storage.ErrUnavailable
				}
				if mode == "late" && systems == 2 && e == nil {
					e = os.Chmod(l, 0700)
				}
				return v, e
			}
			// Faults wrap actual fixed reads, through the same original registry. These
			// are private fixture transfers, not an alternate production constructor.
			q, e := commissionExecutionOwner(ctx, nativeExecutionOwners, func() bool { return os.Geteuid() == 0 }, func(c context.Context) (executionOwnerSource, error) {
				v, e := storage.ReviewExecutionStartup(c, j, l, readNativeExecutionBacking, system, boot)
				if v == nil {
					return executionOwnerSource{}, e
				}
				return executionOwnerSource{observe: v.Observe, finish: func(context.Context) error { return storage.ErrDenied }, release: v.Close}, e
			})
			check("injected_error_denied", q == nil && e != nil)
			after, _ := readActivationFixture(t, j)
			check("closed_history_preserved", bytes.Equal(before, after))
			check("both_locks_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
			check("no_commission_after_failure", alternative())
		} else {
			if mode == "clean" {
				cancelled, stop := context.WithCancel(ctx)
				stop()
				for _, x := range []struct {
					name string
					c    context.Context
				}{{"nil", nil}, {"cancelled", cancelled}} {
					q, e := reviewNativeExecutionStartupOwner(x.c, j+"-"+x.name, l+"-"+x.name)
					check(x.name+"_no_creation", q == nil && e != nil && !activationPathExists(j+"-"+x.name) && !activationPathExists(l+"-"+x.name))
				}
			}
			o, e := reviewNativeExecutionStartupOwner(ctx, j, l)
			check("closed_registered", o != nil && e == nil)
			defer o.Close()
			switch mode {
			case "clean":
				check("both_locks_held", nativeActivationLockHeld(t, j) && nativeActivationLockHeld(t, l))
				first, e := o.Observe(ctx)
				check("original_observation", e == nil && first != (storage.ExecutionDomainsObservation{}))
				again, e := o.Observe(ctx)
				check("repeated_observation", e == nil && again == first)
				copied := new(commissionedExecutionOwner)
				reflect.ValueOf(copied).Elem().Set(reflect.ValueOf(o).Elem())
				got, e := copied.Observe(ctx)
				check("copied_denied", e != nil && got == (storage.ExecutionDomainsObservation{}) && copied.FinishIdle(ctx) != nil && copied.Close() != nil)
				wire, e := json.Marshal(o)
				var decoded commissionedExecutionOwner
				check("decoded_denied", e == nil && string(wire) == "{}" && json.Unmarshal(wire, &decoded) == nil && decoded.FinishIdle(ctx) != nil && decoded.Close() != nil)
				cancelled, stop := context.WithCancel(ctx)
				stop()
				got, e = o.Observe(cancelled)
				again, live := o.Observe(ctx)
				check("cancelled_observe_no_effect", e != nil && got == (storage.ExecutionDomainsObservation{}) && live == nil && again == first)
				q, e := reviewNativeExecutionStartupOwner(ctx, j+"-startup", l+"-startup")
				check("other_startup_no_creation", q == nil && e != nil && !activationPathExists(j+"-startup") && !activationPathExists(l+"-startup"))
				check("other_commission_no_creation", alternative())
				check("finish_denied", o.FinishIdle(ctx) != nil)
				got, e = o.Observe(ctx)
				check("finished_observe_denied", e != nil && got == (storage.ExecutionDomainsObservation{}))
				check("closed_both_locks_released", o.Close() == nil && o.Close() == nil && !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
				q, e = reviewNativeExecutionStartupOwner(ctx, j, l)
				check("closed_registration_not_reset", q == nil && e != nil)
				after, _ := readActivationFixture(t, j)
				check("closed_history_preserved", bytes.Equal(before, after))
			case "drift", "touch":
				if mode == "touch" {
					if os.Chmod(l, 0700) != nil {
						t.Fatal("owned same-inode directory touch")
					}
				} else if os.WriteFile(filepath.Join(l, "unknown-startup-entry"), []byte("owned-startup-drift"), 0600) != nil {
					t.Fatal("owned drift")
				}
				got, e := o.Observe(ctx)
				name := "ledger_drift_denied"
				if mode == "touch" {
					name = "ledger_touch_denied"
				}
				check(name, e != nil && got == (storage.ExecutionDomainsObservation{}))
				got, e = o.Observe(ctx)
				check("drift_retired", e != nil && got == (storage.ExecutionDomainsObservation{}))
				check("finish_denied", o.FinishIdle(ctx) != nil)
				check("release", o.Close() == nil && o.Close() == nil)
				after, _ := readActivationFixture(t, j)
				check("closed_history_preserved", bytes.Equal(before, after))
				check("both_locks_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
				check("no_commission_after_failure", alternative())
			case "uid":
				jf, e := os.Open(filepath.Join(j, "execution.json"))
				if e != nil {
					t.Fatal(e)
				}
				defer jf.Close()
				var dirs []*os.File
				for _, path := range []string{j, l} {
					f, e := os.Open(path)
					if e != nil {
						t.Fatal(e)
					}
					defer f.Close()
					dirs = append(dirs, f)
				}
				check("actual_nonroot", unix.Setgroups([]int{}) == nil && unix.Setgid(65534) == nil && unix.Setuid(65534) == nil && os.Geteuid() == 65534)
				owned := true
				for _, f := range dirs {
					var st unix.Stat_t
					owned = owned && unix.Fstat(int(f.Fd()), &st) == nil && st.Uid == 0
				}
				check("inherited_root_descriptors", owned)
				got, e := o.Observe(ctx)
				check("inherited_operations_denied", e != nil && got == (storage.ExecutionDomainsObservation{}) && o.FinishIdle(ctx) != nil)
				data, e := io.ReadAll(io.LimitReader(jf, 64<<10+1))
				check("closed_history_preserved", e == nil && bytes.Equal(before, data))
				check("nonroot_release_denied", errors.Is(o.Close(), ErrPermission) && errors.Is(o.Close(), ErrPermission))
			}
		}
	}
	if !reflect.DeepEqual(seen, names) {
		t.Fatal("incomplete startup child inventory", seen)
	}
	fmt.Println("EXECUTION_STARTUP_CHILD_RESULT " + mode + " " + strings.Join(names, " "))
}
func TestNativeExecutionStartupZFS(t *testing.T) { nativeExecutionStartupParent(t, "zfs") }
func TestNativeExecutionStartupUFS(t *testing.T) { nativeExecutionStartupParent(t, "ufs") }
func nativeExecutionStartupParent(t *testing.T, kind string) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PROPERTY_VM") != "1" || os.Getenv("HORVIUM_EXECUTION_STARTUP_VM") != "1" {
		t.Skip("owned startup fixture required")
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
		nativeExecutionStartupFixture(t, ctx, kind, policy, ordinary, disk)
	})
	if !cleaned {
		t.Fatal("data fixture remains")
	}
	t.Log("EXECUTION_STARTUP_DATA_TEARDOWN " + kind + " restored=pass")
}
func nativeExecutionStartupFixture(t *testing.T, ctx context.Context, kind string, policy storage.Policy, ordinary string, disk inventory.Provider) {
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
	anchor, e := os.MkdirTemp("/root", "horvium-execution-startup-")
	if e != nil {
		t.Fatal(e)
	}
	pool := "horvium_startup_" + strings.TrimPrefix(filepath.Base(anchor), "horvium-execution-startup-")
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
			t.Fatal("startup fixture", kind, name)
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
	for _, mode := range []string{"clean", "touch", "active", "missing", "drift", "uid", "boot", "system", "late"} {
		cj, cl := j, l
		if mode == "active" {
			cj, cl = ja, la
		}
		if mode == "missing" {
			cj, cl = filepath.Join(jp, "absent"), filepath.Join(lp, "absent")
		}
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestNativeExecutionStartupChild$", "-test.timeout=150s")
		cmd.Env = append(os.Environ(), "HORVIUM_STARTUP_CHILD="+mode, "HORVIUM_STARTUP_JOURNAL="+cj, "HORVIUM_STARTUP_LEDGER="+cl)
		data, e := cmd.CombinedOutput()
		if e != nil {
			limit := len(data)
			if limit > 16<<10 {
				limit = 16 << 10
			}
			t.Logf("owned startup child %s: %s", mode, data[:limit])
		}
		check(mode+"_child_waited", e == nil && cmd.ProcessState != nil && cmd.ProcessState.Success() && len(data) <= 16<<10)
		prefix := "EXECUTION_STARTUP_CHILD_RESULT " + mode + " "
		found := false
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, prefix) {
				if found || strings.TrimPrefix(line, prefix) != strings.Join(startupNativeCases[mode], " ") {
					t.Fatal("incomplete/duplicate startup child")
				}
				found = true
			}
		}
		if !found {
			t.Fatal("missing child inventory")
		}
		for _, name := range startupNativeCases[mode] {
			check(mode+"_"+name, true)
		}
		if mode != "missing" {
			check(mode+"_parent_locks_released", !nativeActivationLockHeld(t, cj) && !nativeActivationLockHeld(t, cl))
			history, r := readActivationFixture(t, cj)
			expected, state := closed, "closed"
			if mode == "active" {
				expected, state = active, "active"
			}
			check(mode+"_parent_history", bytes.Equal(history, expected) && r.Owner.State == state)
		}
		if mode == "drift" {
			name := filepath.Join(l, "unknown-startup-entry")
			data, e := os.ReadFile(name)
			if e != nil || string(data) != "owned-startup-drift" || os.Remove(name) != nil {
				t.Fatal("owned drift fixture restoration")
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
	t.Log("EXECUTION_STARTUP_DOMAIN_TEARDOWN " + kind + " filesystems=pass directories=pass")
	t.Log("EXECUTION_STARTUP_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
}
