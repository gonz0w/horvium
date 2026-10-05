// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"bytes"
	"context"
	"errors"
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
			t.Fatalf("native fixture read failed via %s (arguments/values withheld): %v", path, err)
		}
		return out.Bytes()
	}
	collect := func() (GEOM, []byte) {
		data := run("/sbin/sysctl", "-b", "kern.geom.confxml")
		g, err := ParseGEOM(data, DefaultLimits())
		if err != nil {
			t.Fatal("native fixture graph unavailable", err)
		}
		return g, data
	}
	selected := func(g GEOM) []Provider {
		var out []Provider
		for _, p := range g.Providers {
			if p.Kind == DiskCandidate && p.MediaSize == deviceFixtureBytes {
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
	graph, rawGEOM := collect()
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
		serialData := run("/usr/sbin/diskinfo", "-s", path)
		if strings.TrimSpace(string(serialData)) != id {
			t.Fatal("independent serial mismatch")
		}
		geometryData := run("/usr/sbin/diskinfo", path)
		fields := strings.Fields(string(geometryData))
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
		projection, err := ParseDiskObservation(rawGEOM, geometryData, serialData, DefaultLimits())
		if expectedIdentity == IdentityUnique {
			if err != nil || !reflect.DeepEqual(projection.Provider, p) || projection.SectorCount != nums[2] || projection.StripeSize != nums[3] || projection.StripeOffset != nums[4] {
				t.Fatal("product identity/geometry projection did not correlate")
			}
			changed := append([]string(nil), fields...)
			changed[1] = strconv.FormatUint(expectedSector*2, 10)
			if _, err := ParseDiskObservation(rawGEOM, []byte(strings.Join(changed, "\t")+"\n"), serialData, DefaultLimits()); err == nil {
				t.Fatal("product accepted contradictory geometry projection")
			}
		} else if !errors.Is(err, ErrDiskCorrelation) || !reflect.DeepEqual(projection, DiskObservation{}) {
			t.Fatal("product accepted duplicate/missing native serial")
		}
		// DEV and native disk-ident LABEL attachments are correspondence facts,
		// never evidence that a provider is unused or safe for creation.
		if err := fixtureDiskCorrespondence(graph, p); err != nil {
			t.Fatal(err)
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
	// Resolve the image's actual mounted root: supplied images can use UFS or
	// ZFS. Follow its exact leaf/alias backing; never hash boot disk content.
	mountedRoot := func() fixtureRootMount {
		root, err := fixtureMountedRoot(run("/sbin/mount", "-p"))
		if err != nil {
			t.Fatal(err)
		}
		return root
	}
	rootMount := mountedRoot()
	rootPool, bootGUID := "", ""
	leaves := []string{strings.TrimPrefix(rootMount.Source, "/dev/")}
	if rootMount.Type == "zfs" {
		rootPool = strings.Split(rootMount.Source, "/")[0]
		bootGUID = strings.TrimSpace(string(run("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", rootPool)))
		if _, err := strconv.ParseUint(bootGUID, 10, 64); err != nil || bootGUID == "0" {
			t.Fatal("guest root pool identity missing")
		}
		leaves = nil
		for _, line := range strings.Split(string(run("/sbin/zpool", "status", "-L", "-P", rootPool)), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 5 && strings.HasPrefix(fields[0], "/dev/") {
				leaves = append(leaves, strings.TrimPrefix(fields[0], "/dev/"))
			}
		}
	}
	roots, err := fixtureBootDisks(graph, leaves)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range disks {
		if roots[p.Name] {
			t.Fatal("guest boot disk selected as test backing")
		}
	}
	afterGraph, _ := collect()
	after := selected(afterGraph)
	if !reflect.DeepEqual(disks, after) {
		t.Fatal("native fixture identity/geometry drifted")
	}
	for _, p := range after {
		if strings.TrimSpace(string(run("/sbin/sha256", "-q", "/dev/"+p.Name))) != before[p.Name] {
			t.Fatal("read-only fixture changed content")
		}
	}
	afterRoots, err := fixtureBootDisks(afterGraph, leaves)
	if err != nil || !reflect.DeepEqual(roots, afterRoots) || mountedRoot() != rootMount {
		t.Fatal("guest root mount/backing identity changed")
	}
	if rootPool != "" && strings.TrimSpace(string(run("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", rootPool))) != bootGUID {
		t.Fatal("guest root pool identity changed")
	}
	t.Log("DEVICE_OBSERVATION_RESULT distinct_serial=pass duplicate_serial=pass missing_serial=pass sector_variants=pass diskinfo_correlation=pass backing_relationships=pass boot_exclusion=pass read_preservation=pass")
	t.Log("DEVICE_CORRELATION_RESULT unique_serial=pass duplicate_denial=pass missing_denial=pass geometry_binding=pass")
}
