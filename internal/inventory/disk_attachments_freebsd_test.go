// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Only this opt-in disposable fixture test creates a partition table. Product
// code remains pure/read-only. No operator-supplied path or live device is used.
func TestNativeDiskAttachments(t *testing.T) {
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
		cmd.Stdout, cmd.Stderr = &out, io.Discard
		if err := cmd.Run(); err != nil {
			t.Fatalf("native attachment fixture failed via %s (arguments/values withheld): %v", path, err)
		}
		return out.Bytes()
	}
	collect := func() (GEOM, []byte) {
		data := run("/sbin/sysctl", "-b", "kern.geom.confxml")
		g, err := ParseGEOM(data, DefaultLimits())
		if err != nil {
			t.Fatal("native attachment graph unavailable", err)
		}
		return g, data
	}
	graph, raw := collect()
	var disks []Provider
	var target Provider
	seen := make(map[string]int)
	for _, p := range graph.Providers {
		if p.Kind != DiskCandidate || p.MediaSize != deviceFixtureBytes {
			continue
		}
		if p.Class != "DISK" || !strings.HasPrefix(p.Name, "vtbd") || !diskAddress(p.Name) {
			t.Fatal("unqualified fixture address/class")
		}
		serial := ""
		for _, id := range p.Identifiers {
			if id.Kind != "serial" || (id.Source != "geom.config.ident" && id.Source != "provider.config.ident") || (serial != "" && serial != id.Value) {
				t.Fatal("unqualified fixture identifier")
			}
			serial = id.Value
		}
		seen[serial]++
		sector, identity := uint64(512), IdentityUnique
		switch serial {
		case "HVM-UNIQUE-512":
			target = p
		case "HVM-UNIQUE-4K":
			sector = 4096
		case "HVM-DUPLICATE":
			identity = IdentityAmbiguous
		case "":
			identity = IdentityMissing
		default:
			t.Fatal("foreign provider in fixture set")
		}
		if p.SectorSize != sector || p.Identity != identity || fixtureDiskCorrespondence(graph, p) != nil {
			t.Fatal("unqualified fixture geometry/identity/backing")
		}
		disks = append(disks, p)
	}
	if len(disks) != 5 || seen["HVM-UNIQUE-512"] != 1 || seen["HVM-UNIQUE-4K"] != 1 || seen["HVM-DUPLICATE"] != 2 || seen[""] != 1 {
		t.Fatal("declared fixture set incomplete")
	}
	path := "/dev/" + target.Name
	observation, err := ParseDiskObservation(raw, run("/usr/sbin/diskinfo", path), run("/usr/sbin/diskinfo", "-s", path), DefaultLimits())
	if err != nil || !reflect.DeepEqual(observation.Provider, target) {
		t.Fatal("independent fixture identity/geometry unavailable")
	}
	attachments, err := ParseDiskAttachments(raw, target.Name, DefaultLimits())
	if err != nil || !reflect.DeepEqual(attachments.Disk, target) || len(attachments.Aliases) != 1 || attachments.Aliases[0].Name != "diskid/DISK-HVM-UNIQUE-512" {
		t.Fatal("actual native disk-ident alias not recognized")
	}
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
			t.Fatal("root pool identity unavailable")
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
	before := make(map[string]string)
	for _, p := range disks {
		if roots[p.Name] {
			t.Fatal("boot backing selected as fixture")
		}
		if p.Name != target.Name {
			hash := strings.TrimSpace(string(run("/sbin/sha256", "-q", "/dev/"+p.Name)))
			if len(hash) != 64 {
				t.Fatal("foreign fixture hash unavailable")
			}
			before[p.Name] = hash
		}
	}
	// Registered before mutation; cleanup uses only the exact verified fixture.
	// Never force-destroy a table or delete an unowned partition on failure.
	tableCreated, partitionCreated := false, false
	cleanup := func() bool {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanCancel()
		step := func(args ...string) bool {
			cmd := exec.CommandContext(cleanCtx, "/sbin/gpart", args...)
			cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
			cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
			return cmd.Run() == nil
		}
		if partitionCreated {
			if !step("delete", "-i", "1", target.Name) {
				return false
			}
			partitionCreated = false
		}
		if tableCreated {
			if !step("destroy", target.Name) {
				return false
			}
			tableCreated = false
		}
		return true
	}
	t.Cleanup(func() {
		if !cleanup() {
			t.Error("owned partition fixture cleanup failed; guest must be discarded")
		}
	})
	assertPartitionDenial := func(expectedOutputs int) {
		g, data := collect()
		found := 0
		classes := make(map[string]string)
		for _, c := range g.Classes {
			classes[c.ID] = c.Name
		}
		for _, geom := range g.Geoms {
			for _, consumer := range geom.Consumers {
				if consumer.ProviderID != target.ID || classes[geom.ClassID] != "PART" {
					continue
				}
				found++
				outputs := 0
				for _, p := range g.Providers {
					if p.GeomID == geom.ID {
						outputs++
						if p.Kind != Logical || p.Class != "PART" || p.MediaSize != 16<<20 || p.SectorSize != target.SectorSize {
							t.Fatal("actual partition observation mismatch")
						}
					}
				}
				if outputs != expectedOutputs {
					t.Fatal("actual partition table shape mismatch")
				}
			}
		}
		if found != 1 {
			t.Fatal("actual PART backing relation missing")
		}
		result, err := ParseDiskAttachments(data, target.Name, DefaultLimits())
		if !errors.Is(err, ErrDiskAttachments) || !reflect.DeepEqual(result, DiskAttachments{}) {
			t.Fatal("partition attachment accepted or partial projection exposed")
		}
	}
	run("/sbin/gpart", "create", "-s", "gpt", target.Name)
	tableCreated = true
	assertPartitionDenial(0)
	run("/sbin/gpart", "add", "-i", "1", "-t", "freebsd-zfs", "-a", "4k", "-s", "16m", target.Name)
	partitionCreated = true
	assertPartitionDenial(1)
	if !cleanup() {
		t.Fatal("owned partition fixture cleanup failed")
	}
	after, data := collect()
	result, err := ParseDiskAttachments(data, target.Name, DefaultLimits())
	if err != nil || !reflect.DeepEqual(result.Disk, target) || len(result.Aliases) != 1 {
		t.Fatal("fixture attachment graph not restored")
	}
	for _, p := range disks {
		found := false
		for _, q := range after.Providers {
			if q.ID == p.ID && reflect.DeepEqual(p, q) {
				found = true
			}
		}
		if !found {
			t.Fatal("fixture identity/geometry changed")
		}
		if p.Name != target.Name && strings.TrimSpace(string(run("/sbin/sha256", "-q", "/dev/"+p.Name))) != before[p.Name] {
			t.Fatal("foreign fixture content changed")
		}
	}
	afterRoots, err := fixtureBootDisks(after, leaves)
	if err != nil || !reflect.DeepEqual(roots, afterRoots) || mountedRoot() != rootMount {
		t.Fatal("guest root mount/backing changed")
	}
	if rootPool != "" && strings.TrimSpace(string(run("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", rootPool))) != bootGUID {
		t.Fatal("guest root pool identity changed")
	}
	t.Log("DEVICE_ATTACHMENT_RESULT native_alias=pass partition_denial=pass blank_table_denial=pass fixture_teardown=pass foreign_preservation=pass")
}
