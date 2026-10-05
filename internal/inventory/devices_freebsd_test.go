// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type deviceCapture struct{ bytes.Buffer }

func (b *deviceCapture) Write(p []byte) (int, error) {
	if len(p) > 64<<20-b.Len() {
		return 0, ErrLimit
	}
	return b.Buffer.Write(p)
}

// TestNativeDeviceFixtures reads only the five fixed host-declared snapshot
// disks. It qualifies test infrastructure, not enrollment or write eligibility.
func TestNativeDeviceFixtures(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_DEVICE_VM") != "1" {
		t.Skip("requires disposable device QEMU opt-ins")
	}
	if os.Geteuid() != 0 {
		t.Fatal("requires disposable guest root")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	run := func(path string, args ...string) []byte {
		cmd := exec.CommandContext(ctx, path, args...)
		cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
		var out deviceCapture
		cmd.Stdout = &out
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			t.Fatalf("native fixture read failed (values withheld): %v", err)
		}
		return out.Bytes()
	}
	collect := func() GEOM {
		g, err := ParseGEOM(run("/sbin/sysctl", "-b", "kern.geom.confxml"), DefaultLimits())
		if err != nil {
			t.Fatal("native fixture graph unavailable", err)
		}
		return g
	}
	selected := func(g GEOM) []Provider {
		var out []Provider
		for _, p := range g.Providers {
			if p.Kind == DiskCandidate && p.MediaSize == 512<<20 {
				out = append(out, p)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		if len(out) != 5 {
			t.Fatal("declared virtual disk set incomplete")
		}
		return out
	}
	serial := func(p Provider) string {
		value := ""
		for _, id := range p.Identifiers {
			if id.Kind != "serial" {
				t.Fatal("unexpected fixture identifier provenance")
			}
			if id.Source != "geom.config.ident" && id.Source != "provider.config.ident" {
				t.Fatal("unexpected serial source")
			}
			if value != "" && value != id.Value {
				t.Fatal("conflicting native fixture serial")
			}
			value = id.Value
		}
		return value
	}
	graph := collect()
	disks := selected(graph)
	seen := make(map[string]int)
	before := make(map[string]string)
	for _, p := range disks {
		if !strings.HasPrefix(p.Name, "vtbd") || strings.ContainsAny(p.Name, "/ \t\r\n") || p.Class != "DISK" {
			t.Fatal("unqualified fixture provider class/address")
		}
		id := serial(p)
		seen[id]++
		expectedSector, expectedStripe := uint64(512), uint64(0)
		expectedIdentity := IdentityUnique
		switch id {
		case "HVM-UNIQUE-512":
			expectedStripe = 4096
		case "HVM-UNIQUE-4K":
			expectedSector = 4096
		case "HVM-DUPLICATE":
			expectedIdentity = IdentityAmbiguous
		case "":
			expectedIdentity = IdentityMissing
		default:
			t.Fatal("foreign disk in declared fixture set")
		}
		if p.SectorSize != expectedSector || p.Identity != expectedIdentity {
			t.Fatal("native sector/identity variant mismatch")
		}
		path := "/dev/" + p.Name
		if strings.TrimSpace(string(run("/usr/sbin/diskinfo", "-s", path))) != id {
			t.Fatal("independent serial mismatch")
		}
		fields := strings.Fields(string(run("/usr/sbin/diskinfo", path)))
		if (len(fields) != 6 && len(fields) != 9) || fields[0] != path {
			t.Fatal("independent geometry projection unavailable")
		}
		nums := make([]uint64, len(fields)-1)
		for i, s := range fields[1:] {
			v, err := strconv.ParseUint(s, 10, 64)
			if err != nil || strconv.FormatUint(v, 10) != s {
				t.Fatal("independent geometry encoding invalid")
			}
			nums[i] = v
		}
		if nums[0] != expectedSector || nums[1] != p.MediaSize || nums[2] != p.MediaSize/p.SectorSize || nums[3] != expectedStripe || nums[4] != 0 {
			t.Fatal("independent native geometry mismatch")
		}
		// New disks are not partitioned and have no backing consumer. This is a
		// fixture-shape assertion, never proof of absent labels/use/remote activity.
		for _, g := range graph.Geoms {
			if g.ID == p.GeomID && len(g.Consumers) != 0 {
				t.Fatal("direct fixture disk has backing consumers")
			}
			for _, c := range g.Consumers {
				if c.ProviderID == p.ID {
					t.Fatal("blank fixture acquired GEOM consumer")
				}
			}
		}
		hash := strings.TrimSpace(string(run("/sbin/sha256", "-q", path)))
		if len(hash) != 64 {
			t.Fatal("fixture content hash missing")
		}
		before[p.Name] = hash
	}
	if seen["HVM-UNIQUE-512"] != 1 || seen["HVM-UNIQUE-4K"] != 1 || seen["HVM-DUPLICATE"] != 2 || seen[""] != 1 {
		t.Fatal("fixture serial cases incomplete")
	}
	// Resolve the guest's mounted ZFS root through actual pool paths and GEOM
	// backing edges. Read it as a negative control; never touch its disk content.
	rootPool := ""
	for _, line := range strings.Split(string(run("/sbin/mount", "-p")), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[1] == "/" && fields[2] == "zfs" {
			rootPool = strings.Split(fields[0], "/")[0]
		}
	}
	if rootPool == "" || strings.HasPrefix(rootPool, "-") {
		t.Fatal("guest root control unavailable")
	}
	bootGUID := strings.TrimSpace(string(run("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", rootPool)))
	if _, err := strconv.ParseUint(bootGUID, 10, 64); err != nil || bootGUID == "0" {
		t.Fatal("guest root pool identity missing")
	}
	providers := make(map[string]Provider)
	geoms := make(map[string]Geom)
	names := make(map[string]string)
	for _, p := range graph.Providers {
		providers[p.ID] = p
		names[p.Name] = p.ID
	}
	for _, g := range graph.Geoms {
		geoms[g.ID] = g
	}
	roots := make(map[string]bool)
	var follow func(string, map[string]bool)
	follow = func(id string, visiting map[string]bool) {
		p, ok := providers[id]
		if !ok || visiting[id] || len(visiting) > 32 {
			t.Fatal("root backing correspondence unavailable")
		}
		if p.Kind == DiskCandidate {
			roots[p.Name] = true
			return
		}
		visiting[id] = true
		defer delete(visiting, id)
		consumers := geoms[p.GeomID].Consumers
		if len(consumers) == 0 {
			t.Fatal("root backing chain incomplete")
		}
		for _, c := range consumers {
			if c.ProviderID == "" {
				t.Fatal("root consumer detached")
			}
			follow(c.ProviderID, visiting)
		}
	}
	for _, line := range strings.Split(string(run("/sbin/zpool", "status", "-L", "-P", rootPool)), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 5 && strings.HasPrefix(fields[0], "/dev/") {
			id := names[strings.TrimPrefix(fields[0], "/dev/")]
			if id == "" {
				t.Fatal("root leaf absent from GEOM")
			}
			follow(id, make(map[string]bool))
		}
	}
	if len(roots) == 0 {
		t.Fatal("guest boot-disk control missing")
	}
	for _, p := range disks {
		if roots[p.Name] {
			t.Fatal("guest boot disk selected as test backing")
		}
	}
	after := selected(collect())
	if !reflect.DeepEqual(disks, after) {
		t.Fatal("native fixture identity/geometry drifted")
	}
	for _, p := range after {
		if strings.TrimSpace(string(run("/sbin/sha256", "-q", "/dev/"+p.Name))) != before[p.Name] {
			t.Fatal("read-only fixture changed content")
		}
	}
	if strings.TrimSpace(string(run("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", rootPool))) != bootGUID {
		t.Fatal("guest root pool identity changed")
	}
	t.Log("DEVICE_OBSERVATION_RESULT distinct_serial=pass duplicate_serial=pass missing_serial=pass sector_variants=pass diskinfo_correlation=pass backing_relationships=pass boot_exclusion=pass read_preservation=pass")
}
