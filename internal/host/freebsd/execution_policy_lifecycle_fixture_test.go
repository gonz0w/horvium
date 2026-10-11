//go:build freebsd && horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
)

var multiplePolicyLifecycleControls = strings.Fields("cpu_affinity registered active_successor active_private_record current_boot leases_held nonjournal_unchanged original_observation observation_preserves_history copied_denied decoded_denied cancelled_no_effect closed_successor ledger_unchanged close_retains_leases retired_observation retired_finish release_once locks_released registration_consumed budget_profile")

// Query this disposable process, never modify a system-wide or product CPU set.
func multiplePolicyLifecycleAffinity(t *testing.T) bool {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cpus, e := execute(c, "/sbin/sysctl", []string{"-n", "hw.ncpu"}, 4096)
	if e != nil || strings.TrimSpace(string(cpus.data)) != "1" {
		return false
	}
	out, e := execute(c, "/usr/bin/cpuset", []string{"-g", "-p", fmt.Sprint(os.Getpid())}, 4096)
	if e != nil {
		return false
	}
	prefix := fmt.Sprintf("pid %d mask:", os.Getpid())
	found, exact := 0, false
	for _, line := range strings.Split(string(out.data), "\n") {
		if strings.HasPrefix(line, prefix) {
			found++
			exact = strings.TrimSpace(strings.TrimPrefix(line, prefix)) == "0"
		}
	}
	return found == 1 && exact
}

func TestMultiplePolicyRootLifecycleChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_POLICY_ROOT_LIFECYCLE_CHILD")
	if mode == "" {
		t.Skip("owned lifecycle child only")
	}
	requirePolicyFactsVM(t)
	if os.Getenv("HORVIUM_POLICY_ROOT_LIFECYCLE_VM") != "1" || os.Getenv("HORVIUM_CLOSED_STARTUP_OWNER_VM") != "1" || (mode != "first" && mode != "restart") || runtime.GOMAXPROCS(0) != 1 {
		t.Fatal("bounded original root lifecycle child required")
	}
	j, l := os.Getenv("HORVIUM_RESTART_JOURNAL"), os.Getenv("HORVIUM_RESTART_LEDGER")
	before, old := readActivationFixture(t, j)
	journalBefore, ledgerBefore := multiplePolicyReviewFiles(t, j), multiplePolicyReviewFiles(t, l)
	if len(journalBefore) != 2 || len(ledgerBefore) != 3 || old.Owner.State != "closed" {
		t.Fatal("original two-policy closed namespace")
	}
	for name, data := range ledgerBefore {
		if name == "writer.lock" {
			continue
		}
		p, e := storage.DecodeOwnershipRecord(data)
		if e != nil || name != "ownership-"+p.PoolGUID+".json" || len(p.Members) != 2 {
			t.Fatal("complete original native policy baseline", e)
		}
	}
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("root lifecycle", mode, name)
		}
		for _, old := range seen {
			if name == old {
				t.Fatal("duplicate lifecycle control")
			}
		}
		seen = append(seen, name)
	}
	check("cpu_affinity", multiplePolicyLifecycleAffinity(t))
	deadline, bounded := t.Deadline()
	timeoutFlag := flag.Lookup("test.timeout")
	check("budget_profile", bounded && timeoutFlag != nil && policyFixtureChildBudgetValid(timeoutFlag.Value.String(), deadline, time.Now()))
	call := func(stage string, budget time.Duration, run func(context.Context) error) error {
		c, cancel := context.WithTimeout(context.Background(), budget)
		defer cancel()
		start := time.Now()
		e := run(c)
		t.Logf("root lifecycle stage=%s elapsed=%.2fs error=%v", stage, time.Since(start).Seconds(), e)
		return e
	}
	var owner *commissionedExecutionOwner
	err := call("startup", 2*time.Minute, func(c context.Context) error {
		var e error
		// Use the actual existing factory, without substitute/wrapped producers.
		owner, e = activateNativeClosedExecutionOwner(c, j, l)
		return e
	})
	check("registered", err == nil && owner != nil)
	defer owner.Close()
	activeBytes, active := readActivationFixture(t, j)
	raw, e := readNativeBoot()
	check("current_boot", e == nil && active.Owner.Boot == hex.EncodeToString(raw[:]))
	identity, e := hex.DecodeString(active.Owner.ID)
	want := old
	want.Generation++
	want.Owner.ID, want.Owner.Boot, want.Owner.State = active.Owner.ID, active.Owner.Boot, "active"
	check("active_successor", e == nil && len(identity) == 16 && active.Owner.ID != old.Owner.ID && active == want && !bytes.Equal(before, activeBytes))
	var st unix.Stat_t
	check("active_private_record", unix.Lstat(filepath.Join(j, "execution.json"), &st) == nil && st.Uid == 0 && st.Mode&07777 == 0600 && st.Mode&unix.S_IFMT == unix.S_IFREG && st.Nlink == 1)
	check("leases_held", nativeActivationLockHeld(t, j) && nativeActivationLockHeld(t, l))
	journalActive := multiplePolicyReviewFiles(t, j)
	expectedFiles := make(map[string][]byte, len(journalBefore))
	for name, data := range journalBefore {
		expectedFiles[name] = data
	}
	expectedFiles["execution.json"] = activeBytes
	check("nonjournal_unchanged", reflect.DeepEqual(expectedFiles, journalActive) && reflect.DeepEqual(ledgerBefore, multiplePolicyReviewFiles(t, l)))
	var observation storage.ExecutionDomainsObservation
	err = call("observe", 90*time.Second, func(c context.Context) error {
		var e error
		observation, e = owner.Observe(c)
		return e
	})
	check("original_observation", err == nil && observation != (storage.ExecutionDomainsObservation{}))
	check("observation_preserves_history", reflect.DeepEqual(journalActive, multiplePolicyReviewFiles(t, j)) && reflect.DeepEqual(ledgerBefore, multiplePolicyReviewFiles(t, l)))
	copied := new(commissionedExecutionOwner)
	reflect.ValueOf(copied).Elem().Set(reflect.ValueOf(owner).Elem())
	var denied storage.ExecutionDomainsObservation
	err = call("copied", 90*time.Second, func(c context.Context) error {
		var e error
		denied, e = copied.Observe(c)
		return e
	})
	check("copied_denied", err != nil && denied == (storage.ExecutionDomainsObservation{}) && call("copied-finish", 90*time.Second, copied.FinishIdle) != nil && copied.Close() != nil)
	wire, e := json.Marshal(owner)
	var decoded commissionedExecutionOwner
	check("decoded_denied", e == nil && string(wire) == "{}" && json.Unmarshal(wire, &decoded) == nil && call("decoded", 90*time.Second, func(c context.Context) error {
		var e error
		denied, e = decoded.Observe(c)
		return e
	}) != nil && denied == (storage.ExecutionDomainsObservation{}) && call("decoded-finish", 90*time.Second, decoded.FinishIdle) != nil && decoded.Close() != nil)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	untouched := true
	for _, c := range []context.Context{nil, cancelled} {
		got, e := owner.Observe(c)
		untouched = untouched && e != nil && got == (storage.ExecutionDomainsObservation{}) && owner.FinishIdle(c) != nil
	}
	check("cancelled_no_effect", untouched && reflect.DeepEqual(journalActive, multiplePolicyReviewFiles(t, j)) && reflect.DeepEqual(ledgerBefore, multiplePolicyReviewFiles(t, l)))
	err = call("finish", 90*time.Second, owner.FinishIdle)
	_, closed := readActivationFixture(t, j)
	want = active
	want.Generation++
	want.Owner.State = "closed"
	check("closed_successor", err == nil && closed == want)
	check("ledger_unchanged", reflect.DeepEqual(ledgerBefore, multiplePolicyReviewFiles(t, l)))
	check("close_retains_leases", nativeActivationLockHeld(t, j) && nativeActivationLockHeld(t, l))
	journalClosed := multiplePolicyReviewFiles(t, j)
	err = call("retired-observe", 90*time.Second, func(c context.Context) error {
		var e error
		denied, e = owner.Observe(c)
		return e
	})
	check("retired_observation", err != nil && denied == (storage.ExecutionDomainsObservation{}))
	check("retired_finish", call("retired-finish", 90*time.Second, owner.FinishIdle) != nil && reflect.DeepEqual(journalClosed, multiplePolicyReviewFiles(t, j)) && reflect.DeepEqual(ledgerBefore, multiplePolicyReviewFiles(t, l)))
	check("release_once", owner.Close() == nil && owner.Close() == nil)
	check("locks_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
	var other *commissionedExecutionOwner
	err = call("registration-consumed", 90*time.Second, func(c context.Context) error {
		var e error
		other, e = activateNativeClosedExecutionOwner(c, j+"-other", l+"-other")
		return e
	})
	check("registration_consumed", err != nil && other == nil && !activationPathExists(j+"-other") && !activationPathExists(l+"-other"))
	if len(seen) != len(multiplePolicyLifecycleControls) {
		t.Fatal("incomplete child lifecycle inventory")
	}
	fmt.Println("POLICY_ROOT_LIFECYCLE_CHILD_RESULT " + mode + " " + strings.Join(seen, " "))
}

func TestNativeMultiplePolicyLifecycleUFS(t *testing.T) {
	nativeMultiplePolicyLifecycleParent(t, "ufs")
}
func TestNativeMultiplePolicyLifecycleZFS(t *testing.T) {
	nativeMultiplePolicyLifecycleParent(t, "zfs")
}
func nativeMultiplePolicyLifecycleParent(t *testing.T, kind string) {
	t.Helper()
	if os.Getenv("HORVIUM_POLICY_ROOT_LIFECYCLE_VM") != "1" || os.Getenv("HORVIUM_POLICY_ROOT_REVIEW_VM") != "1" || os.Getenv("HORVIUM_CLOSED_STARTUP_OWNER_VM") != "1" {
		t.Skip("owned original lifecycle fixture only")
	}
	if !multiplePolicyLifecycleAffinity(t) {
		t.Fatal("owned lifecycle parent requires actual CPU 0 affinity")
	}
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("root lifecycle parent", kind, name)
		}
		markers = append(markers, name+"=pass")
	}
	budgetOK := nativeMultiplePolicyFactsFixtureWithProfile(t, policyFixtureLifecycleWork, func(ctx context.Context, policies []storage.Policy, disk inventory.Provider) {
		domains, mounts := nativeMultiplePolicyMetadataFixture(t, ctx, kind, disk, func(jp, lp string) {
			j, l := filepath.Join(jp, "lifecycle"), filepath.Join(lp, "lifecycle")
			system, e := NewSystemUseSource()
			if e != nil {
				t.Fatal(e)
			}
			if e = storage.SeedExecutionPolicyFixture(ctx, j, l, readNativeExecutionBacking, system.Collect, readNativeActivationBoot, policies); e != nil {
				t.Fatal("original two-policy lifecycle seed", e)
			}
			_, seed := readActivationFixture(t, j)
			ledgerBefore := multiplePolicyReviewFiles(t, l)
			check("seed_closed", seed.Owner.State == "closed" && !seed.Metadata.Pending && !seed.Native.Pending && !seed.Disarmed && len(ledgerBefore) == 3 && !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
			for _, mode := range []string{"first", "restart"} {
				_, before := readActivationFixture(t, j)
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestMultiplePolicyRootLifecycleChild$", "-test.timeout=330s", "-test.count=1")
				cmd.Env = append(os.Environ(), "GOMAXPROCS=1", "HORVIUM_POLICY_ROOT_LIFECYCLE_CHILD="+mode, "HORVIUM_RESTART_JOURNAL="+j, "HORVIUM_RESTART_LEDGER="+l)
				data, err := cmd.CombinedOutput()
				if err != nil {
					t.Logf("bounded root lifecycle child: %.16384s", data)
				}
				check(mode+"_child_waited", err == nil && cmd.ProcessState != nil && cmd.ProcessState.Success() && len(data) <= 16<<10)
				prefix := "POLICY_ROOT_LIFECYCLE_CHILD_RESULT " + mode + " "
				var rows []string
				for _, line := range strings.Split(string(data), "\n") {
					if strings.HasPrefix(line, prefix) {
						rows = append(rows, strings.TrimPrefix(line, prefix))
					}
				}
				good := len(rows) == 1
				if good {
					names := strings.Fields(rows[0])
					sort.Strings(names)
					want := append([]string(nil), multiplePolicyLifecycleControls...)
					sort.Strings(want)
					good = reflect.DeepEqual(names, want)
				}
				check(mode+"_child_inventory", good)
				for _, name := range multiplePolicyLifecycleControls {
					check(mode+"_"+name, true)
				}
				_, after := readActivationFixture(t, j)
				raw, e := readNativeBoot()
				want := before
				want.Generation += 2
				want.Owner.ID, want.Owner.Boot = after.Owner.ID, hex.EncodeToString(raw[:])
				check(mode+"_exact_closed", e == nil && after == want && after.Owner.ID != before.Owner.ID)
				check(mode+"_ledger_preserved", reflect.DeepEqual(ledgerBefore, multiplePolicyReviewFiles(t, l)))
				check(mode+"_leases_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
			}
		})
		check("domain_teardown", domains)
		check("mounts_restored", mounts)
		t.Log("POLICY_ROOT_LIFECYCLE_DOMAIN_TEARDOWN " + kind + " filesystems=pass directories=pass")
	})
	check("parent_budget", budgetOK)
	t.Log("POLICY_ROOT_LIFECYCLE_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
}
