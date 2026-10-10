// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

var commissioningNativeCases = map[string][]string{
	"clean":   {"nil_no_creation", "cancelled_no_creation", "native_initial", "exact_boot", "active_receipt", "both_locks_held", "original_observation", "copied_denied", "decoded_denied", "cancelled_finish_preserves", "second_path_no_creation", "original_idle_finish", "exact_closed_receipt", "finished_denied", "closed_both_locks_released", "closed_registration_not_reset", "closed_history_preserved"},
	"drift":   {"native_initial", "ledger_drift_denied", "drift_permanently_retired", "finish_denied", "unclean_release", "active_history_preserved", "released_both_locks", "registration_not_reset"},
	"factory": {"failure_denied", "no_creation", "failure_registration_not_reset", "retry_no_creation"},
	"uid":     {"native_initial", "actual_nonroot", "inherited_root_descriptors", "inherited_operations_denied", "active_history_preserved", "nonroot_release_denied", "registration_not_reset"},
}

// An exact-binary child has its own production process registry. Never reset that
// registry in a fixture. Only opted-in owned paths/credentials are used here.
func TestNativeCommissionExecutionOwnerChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_COMMISSION_OWNER_CHILD")
	if mode == "" {
		t.Skip("owned commissioning child only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_EXECUTION_SYSTEM_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("owned root fixture required")
	}
	wanted, ok := commissioningNativeCases[mode]
	if !ok {
		t.Fatal("unknown owned child")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	j, l := os.Getenv("HORVIUM_COMMISSION_JOURNAL"), os.Getenv("HORVIUM_COMMISSION_LEDGER")
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("commissioning", name)
		}
		for _, old := range seen {
			if old == name {
				t.Fatal("duplicate child assertion", name)
			}
		}
		seen = append(seen, name)
	}
	if mode == "factory" {
		q, e := commissionNativeExecutionOwner(ctx, filepath.Join(j, "absent", "child"), filepath.Join(l, "absent", "child"))
		check("failure_denied", q == nil && e != nil)
		check("no_creation", !activationPathExists(j) && !activationPathExists(l))
		q, e = commissionNativeExecutionOwner(ctx, j, l)
		check("failure_registration_not_reset", q == nil && e != nil)
		check("retry_no_creation", !activationPathExists(j) && !activationPathExists(l))
	} else {
		if mode == "clean" {
			c, stop := context.WithCancel(ctx)
			stop()
			for _, x := range []struct {
				name string
				c    context.Context
			}{{"nil", nil}, {"cancelled", c}} {
				q, e := commissionNativeExecutionOwner(x.c, j, l)
				check(x.name+"_no_creation", q == nil && e != nil && !activationPathExists(j) && !activationPathExists(l))
			}
		}
		o, e := commissionNativeExecutionOwner(ctx, j, l)
		check("native_initial", o != nil && e == nil)
		defer o.Close()
		before, initial := readActivationFixture(t, j)
		switch mode {
		case "clean":
			raw, e := readNativeBoot()
			check("exact_boot", e == nil && initial.Owner.Boot == hex.EncodeToString(raw[:]))
			check("active_receipt", initial.Generation == 1 && initial.Owner.State == "active" && !initial.Native.Pending && !initial.Metadata.Pending && !initial.Disarmed)
			check("both_locks_held", nativeActivationLockHeld(t, j) && nativeActivationLockHeld(t, l))
			got, e := o.Observe(ctx)
			check("original_observation", e == nil && got != (storage.ExecutionDomainsObservation{}))
			copy := new(commissionedExecutionOwner)
			reflect.ValueOf(copy).Elem().Set(reflect.ValueOf(o).Elem())
			got, e = copy.Observe(ctx)
			check("copied_denied", e != nil && got == (storage.ExecutionDomainsObservation{}) && copy.FinishIdle(ctx) != nil && copy.Close() != nil)
			wire, e := json.Marshal(o)
			var decoded commissionedExecutionOwner
			check("decoded_denied", e == nil && string(wire) == "{}" && json.Unmarshal(wire, &decoded) == nil && decoded.FinishIdle(ctx) != nil && decoded.Close() != nil)
			c, stop := context.WithCancel(ctx)
			stop()
			e = o.FinishIdle(c)
			after, _ := readActivationFixture(t, j)
			check("cancelled_finish_preserves", e != nil && bytes.Equal(before, after))
			q, e := commissionNativeExecutionOwner(ctx, j+"-other", l+"-other")
			check("second_path_no_creation", q == nil && e != nil && !activationPathExists(j+"-other") && !activationPathExists(l+"-other"))
			check("original_idle_finish", o.FinishIdle(ctx) == nil)
			closedBytes, closed := readActivationFixture(t, j)
			check("exact_closed_receipt", closed.Generation == 2 && closed.Owner.State == "closed" && closed.Owner.ID == initial.Owner.ID && closed.Owner.Boot == initial.Owner.Boot && closed.Binding == initial.Binding && closed.Census == initial.Census && !closed.Metadata.Pending && !closed.Native.Pending && !closed.Disarmed)
			got, e = o.Observe(ctx)
			check("finished_denied", e != nil && got == (storage.ExecutionDomainsObservation{}) && o.FinishIdle(ctx) != nil)
			check("closed_both_locks_released", o.Close() == nil && o.Close() == nil && !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
			q, e = commissionNativeExecutionOwner(ctx, j+"-closed", l+"-closed")
			check("closed_registration_not_reset", q == nil && e != nil && !activationPathExists(j+"-closed") && !activationPathExists(l+"-closed"))
			after, _ = readActivationFixture(t, j)
			check("closed_history_preserved", bytes.Equal(closedBytes, after))
		case "drift":
			if os.WriteFile(filepath.Join(l, "unknown-owned-entry"), []byte("owned"), 0600) != nil {
				t.Fatal("owned ledger drift")
			}
			got, e := o.Observe(ctx)
			check("ledger_drift_denied", e != nil && got == (storage.ExecutionDomainsObservation{}))
			got, e = o.Observe(ctx)
			check("drift_permanently_retired", e != nil && got == (storage.ExecutionDomainsObservation{}))
			check("finish_denied", o.FinishIdle(ctx) != nil)
			check("unclean_release", o.Close() == nil && o.Close() == nil)
			after, _ := readActivationFixture(t, j)
			check("active_history_preserved", bytes.Equal(before, after))
			check("released_both_locks", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
			q, e := commissionNativeExecutionOwner(ctx, j+"-retry", l+"-retry")
			check("registration_not_reset", q == nil && e != nil && !activationPathExists(j+"-retry") && !activationPathExists(l+"-retry"))
		case "uid":
			jf, e := os.Open(filepath.Join(j, "execution.json"))
			if e != nil {
				t.Fatal(e)
			}
			defer jf.Close()
			var dirs []*os.File
			for _, p := range []string{j, l} {
				f, e := os.Open(p)
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
			check("active_history_preserved", e == nil && bytes.Equal(before, data))
			check("nonroot_release_denied", errors.Is(o.Close(), ErrPermission) && errors.Is(o.Close(), ErrPermission))
			q, e := commissionNativeExecutionOwner(ctx, j+"-uid", l+"-uid")
			check("registration_not_reset", q == nil && e != nil)
		}
	}
	// Exactly one terminal result is emitted after every assertion and original
	// release. Parent independently waits/probes both locks and retained history.
	if !reflect.DeepEqual(seen, wanted) {
		t.Fatal("incomplete child assertion inventory", seen)
	}
	fmt.Println("COMMISSION_OWNER_CHILD_RESULT " + mode + " " + strings.Join(wanted, " "))
}

func nativeCommissionExecutionOwnerFixture(t *testing.T, ctx context.Context, journalParent, ledgerParent string, check func(string, bool)) {
	t.Helper()
	for _, mode := range []string{"clean", "drift", "factory", "uid"} {
		j, l := filepath.Join(journalParent, "commission-"+mode), filepath.Join(ledgerParent, "commission-"+mode)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestNativeCommissionExecutionOwnerChild$", "-test.timeout=150s")
		cmd.Env = append(os.Environ(), "HORVIUM_COMMISSION_OWNER_CHILD="+mode, "HORVIUM_COMMISSION_JOURNAL="+j, "HORVIUM_COMMISSION_LEDGER="+l)
		data, e := cmd.CombinedOutput()
		check("commission_"+mode+"_child_waited", e == nil && cmd.ProcessState != nil && cmd.ProcessState.Success() && len(data) <= 16<<10)
		prefix := "COMMISSION_OWNER_CHILD_RESULT " + mode + " "
		found := false
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, prefix) {
				if found || strings.TrimPrefix(line, prefix) != strings.Join(commissioningNativeCases[mode], " ") {
					t.Fatal("incomplete/duplicate commissioning child result")
				}
				found = true
			}
		}
		if !found {
			t.Fatal("commissioning child result missing")
		}
		for _, name := range commissioningNativeCases[mode] {
			check("commission_"+mode+"_"+name, true)
		}
		if mode != "factory" {
			check("commission_"+mode+"_parent_locks_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
			_, r := readActivationFixture(t, j)
			state, generation := "active", uint64(1)
			if mode == "clean" {
				state, generation = "closed", 2
			}
			check("commission_"+mode+"_parent_history", r.Owner.State == state && r.Generation == generation && !r.Metadata.Pending && !r.Native.Pending && !r.Disarmed)
		}
	}
}
