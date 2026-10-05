// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

type systemCaptureBuffer struct{ bytes.Buffer }

func (b *systemCaptureBuffer) Write(p []byte) (int, error) {
	if len(p) > 64<<20-b.Len() {
		return 0, ErrLimit
	}
	return b.Buffer.Write(p)
}

// This test modifies only two newly declared 64 MiB snapshot fixtures. Product
// code has no filesystem/mount/swap operations and no write eligibility flag.
func TestNativeSystemDeviceUse(t *testing.T) {
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
		var out systemCaptureBuffer
		cmd.Stdout, cmd.Stderr = &out, io.Discard
		if err := cmd.Run(); err != nil {
			t.Fatalf("native system fixture failed via %s (arguments/values withheld): %v", path, err)
		}
		return out.Bytes()
	}
	collect := func() (inventory.GEOM, []byte, []byte, []byte, []PoolUseCapture, SystemDeviceUse) {
		raw := run("/sbin/sysctl", "-b", "kern.geom.confxml")
		graph, err := inventory.ParseGEOM(raw, inventory.DefaultLimits())
		if err != nil {
			t.Fatal("system fixture graph unavailable", err)
		}
		mounts, swaps := run("/sbin/mount", "-p"), run("/usr/sbin/swapinfo", "-k")
		budget := systemBudget{limits: inventory.DefaultLimits(), remaining: 100000}
		table, err := mountTable(mounts, &budget)
		if err != nil {
			t.Fatal("actual mount projection unavailable", err)
		}
		names := make(map[string]bool)
		for _, m := range table {
			if m.Type == "zfs" {
				name := strings.Split(m.Source, "/")[0]
				if !validName(name) || strings.ContainsAny(name, "/@#") {
					t.Fatal("invalid native pool address")
				}
				names[name] = true
			}
		}
		ordered := make([]string, 0, len(names))
		for name := range names {
			ordered = append(ordered, name)
		}
		sort.Strings(ordered)
		var pools []PoolUseCapture
		for _, name := range ordered {
			pools = append(pools, PoolUseCapture{Name: name, GUID: run("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", name), StatusGUIDs: run("/sbin/zpool", "status", "-g", "-p", name), StatusPaths: run("/sbin/zpool", "status", "-L", "-P", "-p", name)})
		}
		use, err := ParseSystemDeviceUse(raw, mounts, swaps, pools, inventory.DefaultLimits())
		if err != nil {
			t.Fatal("native system use did not correlate", err)
		}
		return graph, raw, mounts, swaps, pools, use
	}
	graph, raw, mounts, _, _, baseline := collect()
	var disks []inventory.Provider
	boot, err := systemFixtureBootDisk(graph)
	if err != nil {
		t.Fatal("declared boot image unavailable or ambiguous", err)
	}
	var mountDisk, swapDisk inventory.Provider
	seen := make(map[string]int)
	for _, p := range graph.Providers {
		if p.Kind != inventory.DiskCandidate {
			continue
		}
		if p.MediaSize != 64<<20 {
			continue
		}
		if p.Class != "DISK" || !strings.HasPrefix(p.Name, "vtbd") || strings.ContainsAny(p.Name, "/ \\\t\r\n") {
			t.Fatal("unqualified fixture address/class")
		}
		serial := ""
		for _, id := range p.Identifiers {
			if id.Kind != "serial" || (id.Source != "geom.config.ident" && id.Source != "provider.config.ident") || (serial != "" && serial != id.Value) {
				t.Fatal("unqualified fixture serial")
			}
			serial = id.Value
		}
		seen[serial]++
		sector, identity := uint64(512), inventory.IdentityUnique
		switch serial {
		case "HVM-UNIQUE-512":
			mountDisk = p
		case "HVM-UNIQUE-4K":
			sector = 4096
			swapDisk = p
		case "HVM-DUPLICATE":
			identity = inventory.IdentityAmbiguous
		case "":
			identity = inventory.IdentityMissing
		default:
			t.Fatal("foreign disk in fixture set")
		}
		if p.SectorSize != sector || p.Identity != identity {
			t.Fatal("unqualified fixture geometry/identity")
		}
		disks = append(disks, p)
	}
	if len(disks) != 5 || boot.ID == "" || seen["HVM-UNIQUE-512"] != 1 || seen["HVM-UNIQUE-4K"] != 1 || seen["HVM-DUPLICATE"] != 2 || seen[""] != 1 {
		t.Fatal("declared fixture set incomplete")
	}
	root := func(use SystemDeviceUse) MountUse {
		for _, m := range use.Mounts {
			if m.Target == "/" {
				return m
			}
		}
		t.Fatal("root mount missing")
		return MountUse{}
	}
	initialRoot := root(baseline)
	// The launch contract declares exactly one boot-image disk and five fixtures.
	// Independently require the actual root observation to select that sole image.
	if len(initialRoot.Disks) != 1 || !reflect.DeepEqual(initialRoot.Disks[0], boot) {
		t.Fatal("actual boot-image backing not recognized")
	}
	rootRows := 0
	for _, line := range strings.Split(string(mounts), "\n") {
		f := strings.Fields(line)
		if len(f) == 6 && f[1] == "/" {
			rootRows++
			if initialRoot.Source != f[0] || initialRoot.Type != f[2] {
				t.Fatal("independent root row mismatch")
			}
		}
	}
	if rootRows != 1 {
		t.Fatal("independent root unavailable")
	}
	for _, m := range baseline.Mounts {
		for _, p := range m.Disks {
			for _, d := range disks {
				if p.ID == d.ID {
					t.Fatal("fixture already mounted/system-used")
				}
			}
		}
	}
	for _, s := range baseline.Swap {
		for _, p := range s.Disks {
			for _, d := range disks {
				if p.ID == d.ID {
					t.Fatal("fixture already active swap")
				}
			}
		}
	}
	for _, p := range []inventory.Provider{mountDisk, swapDisk} {
		path := "/dev/" + p.Name
		observation, err := inventory.ParseDiskObservation(raw, run("/usr/sbin/diskinfo", path), run("/usr/sbin/diskinfo", "-s", path), inventory.DefaultLimits())
		attachments, attachmentErr := inventory.ParseDiskAttachments(raw, p.Name, inventory.DefaultLimits())
		if err != nil || attachmentErr != nil || !reflect.DeepEqual(observation.Provider, p) || !reflect.DeepEqual(attachments.Disk, p) {
			t.Fatal("selected owned fixture identity/geometry/backing unavailable")
		}
	}
	before := make(map[string]string)
	for _, p := range disks {
		if p.ID != mountDisk.ID && p.ID != swapDisk.ID {
			hash := strings.TrimSpace(string(run("/sbin/sha256", "-q", "/dev/"+p.Name)))
			if len(hash) != 64 {
				t.Fatal("foreign fixture hash unavailable")
			}
			before[p.Name] = hash
		}
	}
	mountpoint := t.TempDir()
	mounted, swapping := false, false
	cleanup := func() bool {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		step := func(path string, args ...string) bool {
			cmd := exec.CommandContext(cctx, path, args...)
			cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
			cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
			return cmd.Run() == nil
		}
		if swapping {
			if !step("/sbin/swapoff", "/dev/"+swapDisk.Name) {
				return false
			}
			swapping = false
		}
		if mounted {
			if !step("/sbin/umount", mountpoint) {
				return false
			}
			mounted = false
		}
		return true
	}
	t.Cleanup(func() {
		if !cleanup() {
			t.Error("owned mount/swap cleanup failed; guest must be discarded")
		}
	})
	run("/sbin/newfs", "/dev/"+mountDisk.Name)
	run("/sbin/mount", "-t", "ufs", "-o", "ro", "/dev/"+mountDisk.Name, mountpoint)
	mounted = true
	run("/sbin/swapon", "/dev/"+swapDisk.Name)
	swapping = true
	_, activeGEOM, activeMounts, activeSwap, pools, active := collect()
	foundMount, foundSwap := false, false
	for _, m := range active.Mounts {
		if m.Target == mountpoint {
			foundMount = true
			if m.Source != "/dev/"+mountDisk.Name || m.Type != "ufs" || len(m.Disks) != 1 || !reflect.DeepEqual(m.Disks[0], mountDisk) {
				t.Fatal("actual UFS backing not mapped")
			}
		}
	}
	for _, s := range active.Swap {
		if s.Source == "/dev/"+swapDisk.Name {
			foundSwap = true
			if s.SizeKiB == 0 || s.SizeKiB > 64<<10 || len(s.Disks) != 1 || !reflect.DeepEqual(s.Disks[0], swapDisk) {
				t.Fatal("actual active swap backing not mapped")
			}
		}
	}
	if !foundMount || !foundSwap {
		t.Fatal("actual mount/swap fixture missing")
	}
	result, err := ParseSystemDeviceUse(activeGEOM, activeMounts, []byte(strings.Replace(string(activeSwap), "1K-blocks", "1M-blocks", 1)), pools, inventory.DefaultLimits())
	if err == nil || !reflect.DeepEqual(result, SystemDeviceUse{}) {
		t.Fatal("unsupported native capture accepted or partial result exposed")
	}
	if !cleanup() {
		t.Fatal("owned system-use cleanup failed")
	}
	after, _, _, _, _, final := collect()
	if !reflect.DeepEqual(root(final), initialRoot) {
		t.Fatal("root mount/GUID/backing changed")
	}
	for _, m := range final.Mounts {
		if m.Target == mountpoint {
			t.Fatal("owned mount remains")
		}
	}
	for _, s := range final.Swap {
		if s.Source == "/dev/"+swapDisk.Name {
			t.Fatal("owned swap remains")
		}
	}
	for _, p := range disks {
		found := false
		for _, q := range after.Providers {
			if p.ID == q.ID && reflect.DeepEqual(p, q) {
				found = true
			}
		}
		if !found {
			t.Fatal("fixture identity/geometry changed")
		}
		if hash, ok := before[p.Name]; ok && strings.TrimSpace(string(run("/sbin/sha256", "-q", "/dev/"+p.Name))) != hash {
			t.Fatal("foreign fixture content changed")
		}
	}
	t.Log("DEVICE_SYSTEM_USE_RESULT root_backing=pass mounted_disk=pass active_swap=pass unsupported_denial=pass fixture_teardown=pass foreign_preservation=pass")
}
