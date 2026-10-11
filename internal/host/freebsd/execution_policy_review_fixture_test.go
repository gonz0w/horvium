//go:build freebsd && horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

var multiplePolicyReviewClean = strings.Fields("registered original_observation both_policies both_leases shared_deadline zero_eligibility copied_denied decoded_denied finish_denied finish_terminal unchanged_journal unchanged_ledger release_once locks_released registration_consumed")
var multiplePolicyReviewDenial = strings.Fields("denied last_policy both_leases shared_deadline zero_eligibility unchanged_journal unchanged_ledger locks_released registration_consumed")

// Read all fixture files/names, including locks, without transferring authority.
func multiplePolicyReviewFiles(t *testing.T, path string) map[string][]byte {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) > 4 {
		t.Fatal("bounded fixture directory", err)
	}
	out := map[string][]byte{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > 64<<10 {
			t.Fatal("bounded regular fixture file", err)
		}
		data, err := os.ReadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[entry.Name()] = data
	}
	return out
}

func TestMultiplePolicyRootReviewChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_POLICY_ROOT_REVIEW_CHILD")
	if mode == "" {
		t.Skip("owned root review child only")
	}
	requirePolicyFactsVM(t)
	if os.Getenv("HORVIUM_POLICY_ROOT_REVIEW_VM") != "1" || (mode != "clean" && mode != "last-member") || runtime.GOMAXPROCS(0) != 1 {
		t.Fatal("bounded root review child required")
	}
	j, l := os.Getenv("HORVIUM_RESTART_JOURNAL"), os.Getenv("HORVIUM_RESTART_LEDGER")
	journalBefore, ledgerBefore := multiplePolicyReviewFiles(t, j), multiplePolicyReviewFiles(t, l)
	var policies []storage.Policy
	for name, data := range ledgerBefore {
		if name == "writer.lock" {
			continue
		}
		p, err := storage.DecodeOwnershipRecord(data)
		if err != nil || name != "ownership-"+p.PoolGUID+".json" || len(p.Members) != 2 {
			t.Fatal("original bounded policy fixture", err)
		}
		policies = append(policies, p)
	}
	if len(policies) != 2 {
		t.Fatal("complete two-policy namespace")
	}
	sort.Slice(policies, func(i, j int) bool {
		return "ownership-"+policies[i].PoolGUID+".json" < "ownership-"+policies[j].PoolGUID+".json"
	})
	source, err := newExecutionStartupPolicySource()
	if err != nil {
		t.Fatal(err)
	}
	system, err := NewSystemUseSource()
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("root review", mode, name)
		}
		for _, old := range seen {
			if old == name {
				t.Fatal("duplicate control")
			}
		}
		seen = append(seen, name)
	}
	var visits []string
	locks, deadlines, ineligible := true, true, true
	var callerDeadline time.Time
	traced := func(c context.Context, p storage.Policy) (storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse, error) {
		deadline, bounded := c.Deadline()
		deadlines = deadlines && bounded && deadline.Equal(callerDeadline) && c.Err() == nil
		locks = locks && nativeActivationLockHeld(t, j) && nativeActivationLockHeld(t, l)
		visits = append(visits, p.PoolGUID)
		o, d, u, e := source(c, p)
		ineligible = ineligible && !o.Eligible
		return o, d, u, e
	}
	call := func(stage string, run func(context.Context) error) error {
		c, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		callerDeadline, _ = c.Deadline()
		start := time.Now()
		e := run(c)
		t.Logf("root review stage=%s elapsed=%.2fs error=%v", stage, time.Since(start).Seconds(), e)
		return e
	}
	var owner *commissionedExecutionOwner
	err = call("review", func(c context.Context) error {
		var e error
		owner, e = commissionExecutionOwner(c, nativeExecutionOwners, func() bool { return os.Geteuid() == 0 }, func(ctx context.Context) (executionOwnerSource, error) {
			r, e := storage.ReviewExecutionPolicyStartup(ctx, j, l, readNativeExecutionBacking, system.Collect, readNativeActivationBoot, traced)
			if r == nil {
				return executionOwnerSource{}, e
			}
			// Private read-only fixture composition: do not expose Activate.
			return executionOwnerSource{observe: r.Observe, finish: func(context.Context) error { return storage.ErrDenied }, release: r.Close}, e
		})
		return e
	})
	both := func() bool {
		first, last := false, false
		for _, guid := range visits {
			first = first || guid == policies[0].PoolGUID
			last = last || guid == policies[1].PoolGUID
		}
		return first && last
	}
	if mode == "clean" {
		check("registered", err == nil && owner != nil)
		defer owner.Close()
		check("both_policies", both())
		visits = nil
		var observation storage.ExecutionDomainsObservation
		err = call("observe", func(c context.Context) error { var e error; observation, e = owner.Observe(c); return e })
		check("original_observation", err == nil && observation != (storage.ExecutionDomainsObservation{}) && both())
		copied := new(commissionedExecutionOwner)
		reflect.ValueOf(copied).Elem().Set(reflect.ValueOf(owner).Elem())
		err = call("copied", func(c context.Context) error { _, e := copied.Observe(c); return e })
		check("copied_denied", err != nil && copied.Close() != nil)
		wire, e := json.Marshal(owner)
		var decoded commissionedExecutionOwner
		check("decoded_denied", e == nil && string(wire) == "{}" && json.Unmarshal(wire, &decoded) == nil && call("decoded", func(c context.Context) error { _, e := decoded.Observe(c); return e }) != nil)
		err = call("finish-denied", owner.FinishIdle)
		check("finish_denied", err != nil)
		err = call("terminal", func(c context.Context) error { var e error; observation, e = owner.Observe(c); return e })
		check("finish_terminal", err != nil && observation == (storage.ExecutionDomainsObservation{}))
		check("release_once", owner.Close() == nil && owner.Close() == nil)
	} else {
		check("denied", err != nil && owner == nil)
		check("last_policy", len(visits) == 2 && visits[0] == policies[0].PoolGUID && visits[1] == policies[1].PoolGUID)
	}
	check("both_leases", locks && len(visits) > 0)
	check("shared_deadline", deadlines)
	check("zero_eligibility", ineligible)
	check("unchanged_journal", reflect.DeepEqual(journalBefore, multiplePolicyReviewFiles(t, j)))
	check("unchanged_ledger", reflect.DeepEqual(ledgerBefore, multiplePolicyReviewFiles(t, l)))
	check("locks_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
	var other *commissionedExecutionOwner
	err = call("registration-consumed", func(c context.Context) error {
		var e error
		other, e = activateNativeClosedExecutionOwner(c, j+"-other", l+"-other")
		return e
	})
	check("registration_consumed", err != nil && other == nil && !activationPathExists(j+"-other") && !activationPathExists(l+"-other"))
	expected := multiplePolicyReviewClean
	if mode == "last-member" {
		expected = multiplePolicyReviewDenial
	}
	if len(seen) != len(expected) {
		t.Fatal("child control count")
	}
	// Emit an exact marker without testing.T's file/line prefix.
	fmt.Println("POLICY_ROOT_REVIEW_CHILD_RESULT " + mode + " " + strings.Join(seen, " "))
}

func TestNativeMultiplePolicyReviewUFS(t *testing.T) { nativeMultiplePolicyReviewParent(t, "ufs") }
func TestNativeMultiplePolicyReviewZFS(t *testing.T) { nativeMultiplePolicyReviewParent(t, "zfs") }
func nativeMultiplePolicyReviewParent(t *testing.T, kind string) {
	t.Helper()
	if os.Getenv("HORVIUM_POLICY_ROOT_REVIEW_VM") != "1" || os.Getenv("HORVIUM_CLOSED_STARTUP_OWNER_VM") != "1" {
		t.Skip("owned root review fixture only")
	}
	nativeMultiplePolicyFactsFixture(t, func(ctx context.Context, policies []storage.Policy, spare inventory.Provider) {
		nativeMultiplePolicyReviewFixture(t, ctx, kind, policies, spare)
	})
}

func nativeMultiplePolicyReviewFixture(t *testing.T, ctx context.Context, kind string, policies []storage.Policy, disk inventory.Provider) {
	t.Helper()
	command := func(c context.Context, path string, args ...string) (string, error) {
		out, e := execute(c, path, args, 1<<20)
		return string(out.data), e
	}
	must := func(path string, args ...string) string {
		out, e := command(ctx, path, args...)
		if e != nil {
			t.Fatal("owned review command", path, e)
		}
		return out
	}
	if disk.Class != "DISK" || disk.Kind != inventory.DiskCandidate || disk.MediaSize != 64<<20 || disk.Identity != inventory.IdentityAmbiguous || !strings.HasPrefix(disk.Name, "vtbd") || strings.ContainsAny(disk.Name, "/ \\ \t\r\n") {
		t.Fatal("undeclared metadata disk")
	}
	info := strings.Fields(must("/usr/sbin/diskinfo", "/dev/"+disk.Name))
	serial := strings.TrimSpace(must("/usr/sbin/diskinfo", "-s", "/dev/"+disk.Name))
	if len(info) < 3 || strings.TrimPrefix(info[0], "/dev/") != disk.Name || info[1] != "512" || info[2] != "67108864" || serial != "HVM-DUPLICATE" {
		t.Fatal("actual metadata geometry/serial")
	}
	mountsBefore := must("/sbin/mount", "-p")
	anchor, e := os.MkdirTemp("/root", "horvium-policy-root-review-")
	if e != nil {
		t.Fatal(e)
	}
	pool := "horvium_review_" + strings.TrimPrefix(filepath.Base(anchor), "horvium-policy-root-review-")
	var guid string
	created, mounted := false, false
	cleanup := func() bool {
		c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if created {
			actual, err := command(c, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)
			if err != nil || guid == "" || strings.TrimSpace(actual) != guid {
				return false
			}
			if _, err = command(c, "/sbin/zpool", "destroy", pool); err != nil {
				return false
			}
			created = false
			if _, err = command(c, "/sbin/zpool", "list", "-H", pool); err == nil {
				return false
			}
		}
		if mounted {
			var fs unix.Statfs_t
			if unix.Statfs(anchor, &fs) != nil {
				return false
			}
			m := persistentMount(fs)
			if m.kind != "ufs" || m.target != anchor || m.source != "/dev/"+disk.Name {
				return false
			}
			if _, err := command(c, "/sbin/umount", anchor); err != nil {
				return false
			}
			mounted = false
		}
		if os.RemoveAll(anchor) != nil {
			return false
		}
		_, err := os.Lstat(anchor)
		return errors.Is(err, os.ErrNotExist)
	}
	cleaned := false
	defer func() {
		if !cleaned && !cleanup() {
			t.Error("owned root review metadata teardown")
		}
	}()
	if kind == "ufs" {
		must("/sbin/newfs", "-O", "2", "-m", "0", "-u", "/dev/"+disk.Name)
		must("/sbin/mount", "-t", "ufs", "/dev/"+disk.Name, anchor)
		mounted = true
		if os.Chmod(anchor, 0700) != nil {
			t.Fatal("UFS mode")
		}
	} else {
		if storage.ValidatePoolSelection("1", pool) != nil {
			t.Fatal("fixture pool name")
		}
		if _, err := command(ctx, "/sbin/zpool", "list", "-H", pool); err == nil {
			t.Fatal("fixture pool exists")
		}
		must("/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", "-o", "ashift=12", pool, "/dev/"+disk.Name)
		created = true
		guid = strings.TrimSpace(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool))
	}
	jp, lp := filepath.Join(anchor, "journal"), filepath.Join(anchor, "ledger")
	for _, path := range []string{jp, lp} {
		if kind == "zfs" {
			must("/sbin/zfs", "create", "-o", "mountpoint="+path, pool+"/"+filepath.Base(path))
		} else if os.Mkdir(path, 0700) != nil {
			t.Fatal("metadata parent")
		}
		if os.Chmod(path, 0700) != nil {
			t.Fatal("metadata parent mode")
		}
	}
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("root parent", kind, name)
		}
		markers = append(markers, name+"=pass")
	}
	sort.Slice(policies, func(i, j int) bool {
		return "ownership-"+policies[i].PoolGUID+".json" < "ownership-"+policies[j].PoolGUID+".json"
	})
	system, e := NewSystemUseSource()
	if e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"clean", "last-member"} {
		var seed []storage.Policy
		for _, p := range policies {
			b, e := storage.EncodeOwnershipRecord(p)
			if e != nil {
				t.Fatal(e)
			}
			q, e := storage.DecodeOwnershipRecord(b)
			if e != nil {
				t.Fatal(e)
			}
			seed = append(seed, q)
		}
		if mode == "last-member" {
			seed[1].Members[0].Serial = "HVM-DRIFT-DENIAL"
		}
		j, l := filepath.Join(jp, mode), filepath.Join(lp, mode)
		if e = storage.SeedExecutionPolicyFixture(ctx, j, l, readNativeExecutionBacking, system.Collect, readNativeActivationBoot, seed); e != nil {
			t.Fatal("original two-policy seed", e)
		}
		before, record := readActivationFixture(t, j)
		journalBefore, ledgerBefore := multiplePolicyReviewFiles(t, j), multiplePolicyReviewFiles(t, l)
		check(mode+"_seed_closed", record.Owner.State == "closed" && !record.Metadata.Pending && !record.Native.Pending && !record.Disarmed && len(ledgerBefore) == 3 && !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestMultiplePolicyRootReviewChild$", "-test.timeout=150s", "-test.count=1")
		cmd.Env = append(os.Environ(), "GOMAXPROCS=1", "HORVIUM_POLICY_ROOT_REVIEW_CHILD="+mode, "HORVIUM_RESTART_JOURNAL="+j, "HORVIUM_RESTART_LEDGER="+l)
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("bounded root review child: %.16384s", data)
		}
		check(mode+"_child_waited", err == nil && cmd.ProcessState != nil && cmd.ProcessState.Success() && len(data) <= 16<<10)
		expected := multiplePolicyReviewClean
		if mode == "last-member" {
			expected = multiplePolicyReviewDenial
		}
		prefix := "POLICY_ROOT_REVIEW_CHILD_RESULT " + mode + " "
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
			want := append([]string(nil), expected...)
			sort.Strings(want)
			good = reflect.DeepEqual(names, want)
		}
		check(mode+"_child_inventory", good)
		for _, name := range expected {
			check(mode+"_"+name, true)
		}
		after, last := readActivationFixture(t, j)
		check(mode+"_exact_history", bytes.Equal(before, after) && last == record && reflect.DeepEqual(journalBefore, multiplePolicyReviewFiles(t, j)) && reflect.DeepEqual(ledgerBefore, multiplePolicyReviewFiles(t, l)))
		check(mode+"_leases_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
	}
	cleaned = cleanup()
	check("domain_teardown", cleaned)
	check("mounts_restored", must("/sbin/mount", "-p") == mountsBefore)
	t.Log("POLICY_ROOT_REVIEW_DOMAIN_TEARDOWN " + kind + " filesystems=pass directories=pass")
	t.Log("POLICY_ROOT_REVIEW_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
}
