//go:build freebsd && horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

// This is fact qualification only: these codec inputs are outside any product
// ledger/journal and cannot grant startup, enrollment or publication authority.
func TestMultiplePolicyFactsChild(t *testing.T) {
	dir := os.Getenv("HORVIUM_POLICY_FACTS_CHILD")
	if dir == "" {
		t.Skip("owned facts child only")
	}
	requirePolicyFactsVM(t)
	if runtime.GOMAXPROCS(0) != 1 {
		t.Fatal("child execution thread bound")
	}
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal(name)
		}
		seen = append(seen, name+"=pass")
	}
	var policies [2]storage.Policy
	var before [2][]byte
	for i := range policies {
		b, e := os.ReadFile(filepath.Join(dir, fmt.Sprintf("baseline-%d", i)))
		if e != nil || len(b) > 64<<10 {
			t.Fatal("bounded baseline", e)
		}
		policies[i], e = storage.DecodeOwnershipRecord(b)
		if e != nil || len(policies[i].Members) != 1 || len(policies[i].Scopes) != 1 {
			t.Fatal("baseline codec", e)
		}
		before[i] = b
	}
	check("native_baselines", policies[0].Revision == 1 && policies[1].Revision == 1)
	check("distinct_pools", policies[0].PoolGUID != policies[1].PoolGUID && policies[0].PoolName != policies[1].PoolName && policies[0].Scopes[0].GUID != policies[1].Scopes[0].GUID)
	check("distinct_members", policies[0].Members[0].Serial != policies[1].Members[0].Serial && policies[0].Members[0].VdevGUID != policies[1].Members[0].VdevGUID)
	source, e := newExecutionStartupPolicySource()
	if e != nil {
		t.Fatal(e)
	}
	ctx, stop := context.WithTimeout(context.Background(), 90*time.Second)
	defer stop()
	deadline, bounded := ctx.Deadline()
	var observations [2]storage.Observation
	var disks [2][]inventory.DiskObservation
	var uses [2]storage.PoolSystemUse
	validate := func(p storage.Policy, i int) error {
		return errors.Join(storage.ValidatePropertyMembers(p, observations[i], disks[i]), storage.CheckPropertySystemExclusions(observations[i], uses[i]), storage.CheckPropertySystemProviders(p.PoolGUID, uses[i], disks[i]))
	}
	for i, p := range policies {
		start := time.Now()
		observations[i], disks[i], uses[i], e = source(ctx, p)
		t.Logf("facts child pool=%d elapsed=%.2fs error=%v", i, time.Since(start).Seconds(), e)
		if e != nil {
			t.Fatal("actual fixed facts", e)
		}
		check([]string{"first_pass", "second_pass"}[i], validate(p, i) == nil)
	}
	last := policies[1]
	last.Members = append([]storage.PoolMember(nil), last.Members...)
	last.Members[0].Serial = "HVM-DRIFT-DENIAL"
	check("last_member_denied", validate(last, 1) != nil && validate(policies[0], 0) == nil)
	last = policies[1]
	last.Scopes = append([]storage.Scope(nil), last.Scopes...)
	last.Scopes[0].GUID = policies[0].Scopes[0].GUID
	check("last_scope_denied", validate(last, 1) != nil && validate(policies[0], 0) == nil)
	check("zero_eligibility", !observations[0].Eligible && !observations[1].Eligible)
	unchanged, preserved := true, true
	for i, p := range policies {
		encoded, err := storage.EncodeOwnershipRecord(p)
		unchanged = unchanged && err == nil && bytes.Equal(encoded, before[i])
		current, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("baseline-%d", i)))
		preserved = preserved && err == nil && bytes.Equal(current, before[i])
	}
	check("no_policy_mutation", unchanged)
	check("records_preserved", preserved)
	after, stillBounded := ctx.Deadline()
	check("bounded_context", bounded && stillBounded && deadline == after && ctx.Err() == nil)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	o, d, u, err := source(cancelled, policies[1])
	check("cancelled_zero", err != nil && reflect.DeepEqual(o, storage.Observation{}) && d == nil && reflect.DeepEqual(u, storage.PoolSystemUse{}))
	t.Log("POLICY_FACTS_CHILD_RESULT " + strings.Join(seen, " "))
}

func requirePolicyFactsVM(t *testing.T) {
	t.Helper()
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_POLICY_FACTS_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("owned disposable root fixture required")
	}
}

func TestNativeMultiplePolicyFacts(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_POLICY_FACTS_VM") != "1" {
		t.Skip("owned disposable facts VM only")
	}
	requirePolicyFactsVM(t)
	ctx, stop := context.WithTimeout(context.Background(), 8*time.Minute)
	defer stop()
	command := func(c context.Context, path string, args ...string) (string, error) {
		out, err := execute(c, path, args, 1<<20)
		return string(out.data), err
	}
	must := func(path string, args ...string) string {
		out, err := command(ctx, path, args...)
		if err != nil {
			t.Fatal("owned fixture command", path, err)
		}
		return out
	}
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal(name)
		}
		seen = append(seen, name+"=pass")
	}
	mountsBefore, swapBefore := must("/sbin/mount", "-p"), must("/usr/sbin/swapinfo", "-k")
	system, e := NewSystemUseSource()
	if e != nil {
		t.Fatal(e)
	}
	use, e := system.Collect(ctx)
	if e != nil {
		t.Fatal("initial physical use", e)
	}
	used, root := map[string]bool{}, false
	for _, m := range use.System().Mounts {
		if m.Target == "/" && len(m.Disks) > 0 {
			root = true
		}
		for _, d := range m.Disks {
			used[d.ID] = true
		}
	}
	for _, s := range use.System().Swap {
		for _, d := range s.Disks {
			used[d.ID] = true
		}
	}
	for _, p := range use.Pools() {
		for _, d := range p.Disks {
			used[d.ID] = true
		}
	}
	geom := []byte(must("/sbin/sysctl", "-b", "kern.geom.confxml"))
	graph, e := inventory.ParseGEOM(geom, inventory.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	var selected [2]inventory.Provider
	count, duplicate, missing := 0, 0, 0
	for _, p := range graph.Providers {
		if p.Kind != inventory.DiskCandidate || p.MediaSize != 64<<20 {
			continue
		}
		count++
		if p.Class != "DISK" || !strings.HasPrefix(p.Name, "vtbd") || strings.ContainsAny(p.Name, "/ \\ \t\r\n") || used[p.ID] {
			t.Fatal("unqualified or used fixture disk")
		}
		serial := ""
		for _, id := range p.Identifiers {
			if id.Kind != "serial" || (id.Source != "geom.config.ident" && id.Source != "provider.config.ident") || serial != "" && serial != id.Value {
				t.Fatal("fixture serial provenance")
			}
			serial = id.Value
		}
		switch serial {
		case "HVM-UNIQUE-512":
			if selected[0].ID != "" || p.SectorSize != 512 || p.Identity != inventory.IdentityUnique {
				t.Fatal("512 fixture")
			}
			selected[0] = p
		case "HVM-UNIQUE-4K":
			if selected[1].ID != "" || p.SectorSize != 4096 || p.Identity != inventory.IdentityUnique {
				t.Fatal("4K fixture")
			}
			selected[1] = p
		case "HVM-DUPLICATE":
			if p.SectorSize != 512 || p.Identity != inventory.IdentityAmbiguous {
				t.Fatal("duplicate fixture")
			}
			duplicate++
		case "":
			if p.SectorSize != 512 || p.Identity != inventory.IdentityMissing {
				t.Fatal("missing fixture")
			}
			missing++
		default:
			t.Fatal("foreign fixture disk")
		}
	}
	if !root || count != 5 || duplicate != 2 || missing != 1 || selected[0].ID == "" || selected[1].ID == "" || selected[0].ID == selected[1].ID {
		t.Fatal("complete disposable disk profile required")
	}
	var geometry [2]inventory.DiskObservation
	for i, p := range selected {
		info, serial := must("/usr/sbin/diskinfo", "/dev/"+p.Name), must("/usr/sbin/diskinfo", "-s", "/dev/"+p.Name)
		geometry[i], e = inventory.ParseDiskObservation(geom, []byte(info), []byte(serial), inventory.DefaultLimits())
		if e != nil || geometry[i].Provider.ID != p.ID || geometry[i].Provider.MediaSize != p.MediaSize || geometry[i].Provider.SectorSize != p.SectorSize {
			t.Fatal("independent fixture address/geometry", e)
		}
	}
	private, e := os.MkdirTemp("/root", "horvium-policy-facts-")
	if e != nil {
		t.Fatal(e)
	}
	ordinary := t.TempDir()
	var names, guids, mounts [2]string
	var created [2]bool
	cleanup := func() bool {
		c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ok := true
		for i := range names {
			if !created[i] {
				continue
			}
			actual, err := command(c, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", names[i])
			if err != nil || guids[i] == "" || strings.TrimSpace(actual) != guids[i] {
				t.Error("owned pool identity changed")
				ok = false
				continue
			}
			if _, err = command(c, "/sbin/zpool", "destroy", names[i]); err != nil {
				t.Error("owned pool removal", err)
				ok = false
				continue
			}
			created[i] = false
			if _, err = command(c, "/sbin/zpool", "list", "-H", names[i]); err == nil {
				t.Error("owned pool remains")
				ok = false
			}
		}
		if created[0] || created[1] {
			return false
		}
		if err := os.RemoveAll(private); err != nil {
			t.Error("private inputs removal", err)
			ok = false
		}
		return ok
	}
	t.Cleanup(func() { cleanup() })
	var records [2][]byte
	observer, e := NewPoolPropertySource()
	if e != nil {
		t.Fatal(e)
	}
	for i, p := range selected {
		names[i] = "horvium_facts_" + strings.TrimPrefix(filepath.Base(private), "horvium-policy-facts-") + fmt.Sprintf("_%d", i)
		if storage.ValidatePoolSelection("1", names[i]) != nil {
			t.Fatal("fixture pool name")
		}
		if _, err := command(ctx, "/sbin/zpool", "list", "-H", names[i]); err == nil {
			t.Fatal("fixture pool exists")
		}
		must("/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", "-o", "ashift=12", names[i], "/dev/"+p.Name)
		created[i] = true
		guids[i] = strings.TrimSpace(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", names[i]))
		rootGUID := strings.TrimSpace(must("/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", names[i]))
		obs, err := observer.Collect(ctx, guids[i], names[i])
		if err != nil {
			t.Fatal("native enrollment baseline", err)
		}
		policy := storage.Policy{Revision: 1, PoolGUID: guids[i], PoolName: names[i], Scopes: []storage.Scope{{GUID: rootGUID, Name: names[i], Properties: []storage.Property{storage.Quota}}}}
		for _, v := range obs.Topology.Nodes {
			if v.Kind != "leaf" {
				continue
			}
			if v.Address != "/dev/"+p.Name {
				t.Fatal("fixture leaf address")
			}
			for _, id := range geometry[i].Provider.Identifiers {
				if id.Kind == "serial" {
					policy.Members = append(policy.Members, storage.PoolMember{VdevGUID: v.GUID, Serial: id.Value, SerialSource: id.Source, MediaSize: p.MediaSize, SectorSize: p.SectorSize, StripeSize: geometry[i].StripeSize, StripeOffset: geometry[i].StripeOffset})
				}
			}
		}
		if len(policy.Members) != 1 {
			t.Fatal("single actual member baseline")
		}
		records[i], err = storage.EncodeOwnershipRecord(policy)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(private, fmt.Sprintf("baseline-%d", i)), records[i], 0400); err != nil {
			t.Fatal(err)
		}
		mounts[i] = filepath.Join(ordinary, fmt.Sprintf("data-%d", i))
		if err = os.Mkdir(mounts[i], 0700); err != nil {
			t.Fatal(err)
		}
		must("/sbin/zfs", "set", "mountpoint="+mounts[i], names[i])
		if err = os.WriteFile(filepath.Join(mounts[i], "sentinel"), []byte(fmt.Sprintf("independent pool sentinel %d", i)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestMultiplePolicyFactsChild$", "-test.timeout=150s", "-test.count=1")
	cmd.Env = append(os.Environ(), "GOMAXPROCS=1", "HORVIUM_POLICY_FACTS_CHILD="+private)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("bounded child failure: %.16384s", output)
	}
	check("child_waited", err == nil && cmd.ProcessState != nil && cmd.ProcessState.Success() && len(output) <= 16<<10)
	var rows []string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "POLICY_FACTS_CHILD_RESULT ") {
			rows = append(rows, strings.TrimSpace(line))
		}
	}
	check("child_inventory", len(rows) == 1 && len(strings.Fields(strings.TrimPrefix(rows[0], "POLICY_FACTS_CHILD_RESULT "))) == 12)
	t.Log(rows[0])
	exact := true
	for i := range records {
		current, err := os.ReadFile(filepath.Join(private, fmt.Sprintf("baseline-%d", i)))
		exact = exact && err == nil && bytes.Equal(current, records[i])
		sentinel, err := os.ReadFile(filepath.Join(mounts[i], "sentinel"))
		check([]string{"sentinel_first", "sentinel_second"}[i], err == nil && bytes.Equal(sentinel, []byte(fmt.Sprintf("independent pool sentinel %d", i))))
	}
	check("policy_bytes", exact)
	check("pools_removed", cleanup() && !created[0] && !created[1])
	check("mounts_restored", must("/sbin/mount", "-p") == mountsBefore)
	check("swap_restored", must("/usr/sbin/swapinfo", "-k") == swapBefore)
	_, err = os.Lstat(private)
	check("inputs_removed", errors.Is(err, os.ErrNotExist))
	t.Log("POLICY_FACTS_PARENT_RESULT " + strings.Join(seen, " "))
}
