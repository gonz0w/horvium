// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

// Called only by the explicitly opted-in disposable native collector test.
// Every mutated device belongs to the five declared snapshot-backed lab disks;
// selection verifies geometry/serial and excludes independently resolved root.
func nativePropertySystemFixture(t *testing.T, ctx context.Context, check func(string, bool)) {
	t.Helper()
	command := func(ctx context.Context, path string, args ...string) (string, error) {
		out, err := execute(ctx, path, args, 1<<20)
		return string(out.data), err
	}
	must := func(path string, args ...string) string {
		out, err := command(ctx, path, args...)
		if err != nil {
			t.Fatal("owned system fixture command (values withheld)", path, err)
		}
		return out
	}
	graph, err := inventory.ParseGEOM([]byte(must("/sbin/sysctl", "-b", "kern.geom.confxml")), inventory.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	mountsBefore := must("/sbin/mount", "-p")
	swapBefore := must("/usr/sbin/swapinfo", "-k")
	// Independently resolve all actually mounted ZFS pools; the two pinned guests
	// have different system backing. No boot-pool mutation is needed.
	names := map[string]bool{}
	for _, line := range strings.Split(mountsBefore, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 6 && fields[2] == "zfs" {
			names[strings.Split(fields[0], "/")[0]] = true
		}
	}
	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)
	var rootCaptures []storage.PoolUseCapture
	for _, name := range ordered {
		if storage.ValidatePoolSelection("1", name) != nil {
			t.Fatal("unsafe fixture root pool")
		}
		rootCaptures = append(rootCaptures, storage.PoolUseCapture{Name: name, GUID: []byte(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", name)), StatusGUIDs: []byte(must("/sbin/zpool", "status", "-g", "-p", name)), StatusPaths: []byte(must("/sbin/zpool", "status", "-L", "-P", "-p", name))})
	}
	baseline, err := storage.ParseSystemDeviceUse([]byte(must("/sbin/sysctl", "-b", "kern.geom.confxml")), []byte(mountsBefore), []byte(swapBefore), rootCaptures, inventory.DefaultLimits())
	if err != nil {
		t.Fatal("fixture root use", err)
	}
	rootIDs := map[string]bool{}
	for _, m := range baseline.Mounts {
		if m.Target == "/" {
			for _, d := range m.Disks {
				rootIDs[d.ID] = true
			}
		}
	}
	var first, second, missing inventory.Provider
	var extras []inventory.Provider
	count := 0
	for _, p := range graph.Providers {
		if p.Kind != inventory.DiskCandidate || p.MediaSize != 64<<20 {
			continue
		}
		count++
		if p.Class != "DISK" || !strings.HasPrefix(p.Name, "vtbd") || strings.ContainsAny(p.Name, "/ \\\t\r\n") || rootIDs[p.ID] {
			t.Fatal("unqualified/boot fixture disk")
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
			if first.ID != "" || p.SectorSize != 512 || p.Identity != inventory.IdentityUnique {
				t.Fatal("512 fixture")
			}
			first = p
		case "HVM-UNIQUE-4K":
			if second.ID != "" || p.SectorSize != 4096 || p.Identity != inventory.IdentityUnique {
				t.Fatal("4k fixture")
			}
			second = p
		case "HVM-DUPLICATE":
			if p.Identity != inventory.IdentityAmbiguous {
				t.Fatal("duplicate fixture")
			}
			extras = append(extras, p)
		case "":
			if p.Identity != inventory.IdentityMissing || missing.ID != "" {
				t.Fatal("missing fixture")
			}
			missing = p
		default:
			t.Fatal("foreign fixture serial")
		}
	}
	if count != 5 || len(rootIDs) == 0 || first.ID == "" || second.ID == "" || missing.ID == "" || len(extras) != 2 {
		t.Fatal("declared snapshot fixture incomplete")
	}
	sort.Slice(extras, func(i, j int) bool { return extras[i].Name < extras[j].Name })
	swapDisk := extras[0]
	for _, d := range []inventory.Provider{first, second, extras[0], extras[1], missing} {
		for _, m := range baseline.Mounts {
			for _, used := range m.Disks {
				if used.ID == d.ID {
					t.Fatal("fixture already mounted")
				}
			}
		}
		for _, s := range baseline.Swap {
			for _, used := range s.Disks {
				if used.ID == d.ID {
					t.Fatal("fixture already swapping")
				}
			}
		}
		infoRows := must("/usr/sbin/diskinfo", "/dev/"+d.Name)
		serialRows := must("/usr/sbin/diskinfo", "-s", "/dev/"+d.Name)
		if d.Identity != inventory.IdentityUnique {
			// This explicitly declared swap fixture has a duplicate serial. That
			// is positive system-use evidence, never runtime creation identity.
			fields := strings.Fields(infoRows)
			expectedSerial := "HVM-DUPLICATE"
			if d.ID == missing.ID {
				expectedSerial = ""
			}
			if len(fields) < 3 || strings.TrimPrefix(fields[0], "/dev/") != d.Name || fields[1] != "512" || fields[2] != "67108864" || strings.TrimSpace(serialRows) != expectedSerial {
				t.Fatal("swap fixture geometry/address")
			}
		} else {
			info, err := inventory.ParseDiskObservation([]byte(must("/sbin/sysctl", "-b", "kern.geom.confxml")), []byte(infoRows), []byte(serialRows), inventory.DefaultLimits())
			if err != nil || info.Provider.Name != d.Name || info.Provider.MediaSize != 64<<20 {
				t.Fatal("fixture address/geometry", err)
			}
		}
	}
	private, err := os.MkdirTemp("/root", "horvium-property-system-")
	if err != nil {
		t.Fatal(err)
	}
	ordinary := t.TempDir()
	systemMount := filepath.Join(private, "mounted")
	if err = os.Mkdir(systemMount, 0700); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(private, "state")
	if err = os.Mkdir(statePath, 0700); err != nil {
		t.Fatal(err)
	}
	pool := "horvium_system_" + filepath.Base(private)[len("horvium-property-system-"):]
	guid := ""
	created, swapping := false, false
	var state *storage.StateDirectory
	cleanup := func() bool {
		clean, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if swapping {
			current, e := command(clean, "/usr/sbin/swapinfo", "-k")
			if e != nil || !strings.Contains(current, "/dev/"+swapDisk.Name+" ") && !strings.Contains(current, "/dev/"+swapDisk.Name+"\t") {
				t.Error("owned swap identity unavailable")
				return false
			}
			if _, e = command(clean, "/sbin/swapoff", "/dev/"+swapDisk.Name); e != nil {
				t.Error("owned swap cleanup", e)
				return false
			}
			swapping = false
		}
		if created {
			actual, e := command(clean, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)
			if e != nil || guid == "" || strings.TrimSpace(actual) != guid {
				t.Error("owned direct pool identity changed")
				return false
			}
			if _, e = command(clean, "/sbin/zpool", "destroy", pool); e != nil {
				t.Error("owned direct pool cleanup", e)
				return false
			}
			created = false
			if _, e = command(clean, "/sbin/zpool", "list", "-H", pool); e == nil {
				t.Error("owned direct pool remains")
				return false
			}
		}
		if state != nil {
			if state.Close() != nil {
				t.Error("owned safety lease cleanup")
				return false
			}
			state = nil
		}
		if os.RemoveAll(private) != nil {
			t.Error("owned private fixture cleanup")
			return false
		}
		if _, e := os.Lstat(private); !errors.Is(e, os.ErrNotExist) {
			t.Error("owned private fixture remains")
			return false
		}
		return true
	}
	t.Cleanup(func() { cleanup() })
	if _, e := command(ctx, "/sbin/zpool", "list", "-H", pool); e == nil {
		t.Fatal("fixture pool exists")
	}
	must("/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", "-o", "ashift=12", pool, "mirror", "/dev/"+first.Name, "/dev/"+second.Name)
	created = true
	guid = strings.TrimSpace(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool))
	rootGUID := strings.TrimSpace(must("/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", pool))
	state, err = storage.OpenStateDirectory(statePath)
	if err != nil {
		t.Fatal(err)
	}
	policy := storage.Policy{Revision: 1, PoolGUID: guid, PoolName: pool, Scopes: []storage.Scope{{GUID: rootGUID, Name: pool, Properties: []storage.Property{storage.Quota}}}}
	data, err := storage.EncodeOwnershipRecord(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(statePath, "ownership-"+guid+".json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	source, err := NewSystemUseSource()
	if err != nil {
		t.Fatal(err)
	}
	u, err := source.Collect(ctx)
	if err != nil {
		t.Fatal("actual system collector", err)
	}
	rootOK := false
	for _, m := range u.System().Mounts {
		if m.Target == "/" {
			rootOK = len(m.Disks) > 0
			for _, d := range m.Disks {
				rootOK = rootOK && rootIDs[d.ID]
			}
		}
	}
	check("system_root_backing", rootOK)
	poolOK := false
	for _, p := range u.Pools() {
		if p.GUID == guid && p.Name == pool {
			names := map[string]bool{}
			for _, d := range p.Disks {
				names[d.Name] = true
			}
			poolOK = len(names) == 2 && names[first.Name] && names[second.Name]
		}
	}
	check("system_pool_backing", poolOK)
	safety, err := NewPropertySafetySource(state)
	if err != nil {
		t.Fatal(err)
	}
	must("/sbin/zfs", "set", "mountpoint="+ordinary, pool)
	sentinel := []byte("independent data sentinel")
	if err = os.WriteFile(filepath.Join(ordinary, "sentinel"), sentinel, 0600); err != nil {
		t.Fatal(err)
	}
	p, o, e := safety.Collect(ctx, guid)
	_, planErr := storage.Build(time.Now(), "fixture-session", p, o, storage.Intent{FilesystemGUID: rootGUID, Property: storage.Quota, Value: "1"})
	check("safety_bound", e == nil && o.Complete && !o.Eligible && p.PoolGUID == guid && errors.Is(planErr, storage.ErrUnavailable))
	providers, err := NewPoolProviderSource()
	if err != nil {
		t.Fatal(err)
	}
	disks, err := providers.Collect(ctx, o)
	check("providers_correlated", err == nil && len(disks) == 2 && !o.Eligible)
	geometryOK := len(disks) == 2
	for _, disk := range disks {
		geometryOK = geometryOK && disk.Provider.MediaSize == 64<<20 && disk.SectorCount == disk.Provider.MediaSize/disk.Provider.SectorSize && (disk.Provider.Name == first.Name && disk.Provider.SectorSize == 512 || disk.Provider.Name == second.Name && disk.Provider.SectorSize == 4096)
	}
	check("provider_geometry", geometryOK)
	nativePropertyProviderDenials(t, ctx, pool, extras, missing, check)
	must("/sbin/zfs", "set", "mountpoint="+systemMount, pool)
	p, o, e = safety.Collect(ctx, guid)
	check("safety_system_mount_denied", errors.Is(e, storage.ErrDenied) && reflect.DeepEqual(p, storage.Policy{}) && reflect.DeepEqual(o, storage.Observation{}))
	must("/sbin/zfs", "set", "mountpoint="+ordinary, pool)
	p, o, e = safety.Collect(ctx, guid)
	check("safety_fresh_restored", e == nil && !o.Eligible && p.PoolGUID == guid)
	must("/sbin/swapon", "/dev/"+swapDisk.Name)
	swapping = true
	u, e = source.Collect(ctx)
	swapOK := false
	for _, s := range u.System().Swap {
		if s.Source == "/dev/"+swapDisk.Name && len(s.Disks) == 1 && s.Disks[0].Name == swapDisk.Name {
			swapOK = true
		}
	}
	check("system_swap_observed", e == nil && swapOK)
	p, o, e = safety.Collect(ctx, guid)
	check("safety_swap_disjoint", e == nil && !o.Eligible && p.PoolGUID == guid)
	retained, e := os.ReadFile(filepath.Join(ordinary, "sentinel"))
	check("system_mount_preserved", e == nil && reflect.DeepEqual(retained, sentinel))
	if !cleanup() {
		t.Fatal("direct fixture teardown failed")
	}
	check("system_fixture_teardown", must("/sbin/mount", "-p") == mountsBefore && must("/usr/sbin/swapinfo", "-k") == swapBefore)
}
