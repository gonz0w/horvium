// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

// Only this disposable fixture creates/destroys its randomly named unmounted
// pool. Product code performs no I/O and obtains no pool mutation authority.
func TestNativeImportedPoolUse(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_DEVICE_VM") != "1" {
		t.Skip("requires disposable device QEMU opt-ins")
	}
	if os.Geteuid() != 0 {
		t.Fatal("requires disposable guest root")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	run := func(ctx context.Context, path string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, path, args...)
		cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
		var out systemCaptureBuffer
		cmd.Stdout, cmd.Stderr = &out, io.Discard
		err := cmd.Run()
		return out.Bytes(), err
	}
	must := func(path string, args ...string) []byte {
		out, err := run(ctx, path, args...)
		if err != nil {
			t.Fatalf("imported-pool fixture failed via %s (arguments/values withheld): %v", path, err)
		}
		return out
	}
	type capture struct {
		graph inventory.GEOM
		raw   []byte
		list  []byte
		pools []PoolUseCapture
		use   []ImportedPoolUse
	}
	collect := func(ctx context.Context) (capture, error) {
		var c capture
		var err error
		c.raw, err = run(ctx, "/sbin/sysctl", "-b", "kern.geom.confxml")
		if err != nil {
			return c, ErrUnavailable
		}
		c.graph, err = inventory.ParseGEOM(c.raw, inventory.DefaultLimits())
		if err != nil {
			return c, ErrUnavailable
		}
		c.list, err = run(ctx, "/sbin/zpool", "list", "-H", "-p", "-o", "name,guid,size,alloc,free,health")
		if err != nil {
			return c, ErrUnavailable
		}
		rows, err := inventory.ParsePools(c.list, inventory.DefaultLimits())
		if err != nil {
			return c, ErrUnavailable
		}
		for _, row := range rows {
			if !validName(row.Name) || strings.ContainsAny(row.Name, "/@#") {
				return c, ErrUnavailable
			}
			p := PoolUseCapture{Name: row.Name}
			p.GUID, err = run(ctx, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", row.Name)
			if err != nil {
				return c, ErrUnavailable
			}
			p.StatusGUIDs, err = run(ctx, "/sbin/zpool", "status", "-g", "-p", row.Name)
			if err != nil {
				return c, ErrUnavailable
			}
			p.StatusPaths, err = run(ctx, "/sbin/zpool", "status", "-L", "-P", "-p", row.Name)
			if err != nil {
				return c, ErrUnavailable
			}
			c.pools = append(c.pools, p)
		}
		c.use, err = ParseImportedPoolUse(c.raw, c.list, c.pools, inventory.DefaultLimits())
		return c, err
	}
	baseline, err := collect(ctx)
	if err != nil {
		t.Fatal("initial imported catalogue unavailable", err)
	}
	boot, err := systemFixtureBootDisk(baseline.graph)
	if err != nil {
		t.Fatal("boot image unavailable", err)
	}
	var selected inventory.Provider
	var disks []inventory.Provider
	seen := make(map[string]int)
	for _, p := range baseline.graph.Providers {
		if p.Kind != inventory.DiskCandidate || p.MediaSize != 64<<20 {
			continue
		}
		if p.Class != "DISK" || !strings.HasPrefix(p.Name, "vtbd") || strings.ContainsAny(p.Name, "/ \\\t\r\n") {
			t.Fatal("foreign fixture address")
		}
		serial := ""
		for _, id := range p.Identifiers {
			if id.Kind != "serial" || (id.Source != "geom.config.ident" && id.Source != "provider.config.ident") || (serial != "" && serial != id.Value) {
				t.Fatal("foreign fixture identity")
			}
			serial = id.Value
		}
		seen[serial]++
		sector, identity := uint64(512), inventory.IdentityUnique
		switch serial {
		case "HVM-UNIQUE-512":
		case "HVM-UNIQUE-4K":
			sector = 4096
			selected = p
		case "HVM-DUPLICATE":
			identity = inventory.IdentityAmbiguous
		case "":
			identity = inventory.IdentityMissing
		default:
			t.Fatal("foreign disk in fixture set")
		}
		if p.SectorSize != sector || p.Identity != identity {
			t.Fatal("fixture geometry/identity unavailable")
		}
		disks = append(disks, p)
	}
	if len(disks) != 5 || selected.ID == "" || selected.ID == boot.ID || seen["HVM-UNIQUE-512"] != 1 || seen["HVM-UNIQUE-4K"] != 1 || seen["HVM-DUPLICATE"] != 2 || seen[""] != 1 {
		t.Fatal("declared fixture set incomplete")
	}
	for _, pool := range baseline.use {
		for _, d := range pool.Disks {
			if d.ID == selected.ID {
				t.Fatal("selected fixture already in imported pool")
			}
		}
	}
	systemUse := func(c capture) SystemDeviceUse {
		mounts, swaps := must("/sbin/mount", "-p"), must("/usr/sbin/swapinfo", "-k")
		b := systemBudget{limits: inventory.DefaultLimits(), remaining: 100000}
		rows, err := mountTable(mounts, &b)
		if err != nil {
			t.Fatal("actual mounts unavailable", err)
		}
		mounted := make(map[string]bool)
		for _, m := range rows {
			if m.Type == "zfs" {
				mounted[strings.Split(m.Source, "/")[0]] = true
			}
		}
		var captures []PoolUseCapture
		for _, p := range c.pools {
			if mounted[p.Name] {
				captures = append(captures, p)
			}
		}
		use, err := ParseSystemDeviceUse(c.raw, mounts, swaps, captures, inventory.DefaultLimits())
		if err != nil {
			t.Fatal("actual system backing unavailable", err)
		}
		return use
	}
	initialSystem := systemUse(baseline)
	notSystemUsed := func(use SystemDeviceUse) {
		for _, m := range use.Mounts {
			for _, d := range m.Disks {
				if d.ID == selected.ID {
					t.Fatal("fixture unexpectedly mounted")
				}
			}
		}
		for _, s := range use.Swap {
			for _, d := range s.Disks {
				if d.ID == selected.ID {
					t.Fatal("fixture unexpectedly active swap")
				}
			}
		}
	}
	notSystemUsed(initialSystem)
	path := "/dev/" + selected.Name
	combined := func(c capture, info, serial []byte, probe *FilesystemProbeCapture) (DeviceUseReview, DeviceUseCapture) {
		inputs := DeviceUseCapture{GEOM: c.raw, Mounts: must("/sbin/mount", "-p"), Swap: must("/usr/sbin/swapinfo", "-k"), Pools: c.list, PoolStatus: c.pools, DiskInfo: info, DiskSerial: serial, Filesystem: probe}
		review, err := ReviewDeviceUse(inputs, inventory.DefaultLimits())
		if err != nil {
			t.Fatal("combined native pool-use observation unavailable", err)
		}
		return review, inputs
	}
	idleReview, _ := combined(baseline, must("/usr/sbin/diskinfo", path), must("/usr/sbin/diskinfo", "-s", path), nil)
	idle := idleReview.Observation()
	if len(idle.Mounts)+len(idle.Swap)+len(idle.Pools) != 0 || idle.SignatureAvailable || idle.Signature != "" {
		t.Fatal("no positive facts manufactured clearance")
	}
	observation, err := inventory.ParseDiskObservation(baseline.raw, must("/usr/sbin/diskinfo", path), must("/usr/sbin/diskinfo", "-s", path), inventory.DefaultLimits())
	attachment, attachmentErr := inventory.ParseDiskAttachments(baseline.raw, selected.Name, inventory.DefaultLimits())
	if err != nil || attachmentErr != nil || !reflect.DeepEqual(observation.Provider, selected) || !reflect.DeepEqual(attachment.Disk, selected) {
		t.Fatal("selected fixture correspondence unavailable")
	}
	before := make(map[string]string)
	for _, p := range disks {
		if p.ID != selected.ID {
			hash := strings.TrimSpace(string(must("/sbin/sha256", "-q", "/dev/"+p.Name)))
			if len(hash) != 64 {
				t.Fatal("foreign hash unavailable")
			}
			before[p.Name] = hash
		}
	}
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal("fixture nonce unavailable")
	}
	name := "hvm-imported-" + hex.EncodeToString(nonce[:])
	for _, p := range baseline.use {
		if p.Name == name {
			t.Fatal("fixture namespace occupied")
		}
	}
	pending := false
	createdGUID := ""
	cleanup := func() bool {
		if !pending {
			return true
		}
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		current, err := collect(cctx)
		if err != nil {
			return false
		}
		for _, p := range current.use {
			if p.Name == name {
				if (createdGUID != "" && p.GUID != createdGUID) || len(p.Disks) != 1 || !reflect.DeepEqual(p.Disks[0], selected) {
					return false
				}
				if _, err := run(cctx, "/sbin/zpool", "destroy", name); err != nil {
					return false
				}
				pending = false
				return true
			}
		}
		pending = false
		return true
	}
	t.Cleanup(func() {
		if !cleanup() {
			t.Error("owned imported-pool cleanup unverified")
		}
	})
	pending = true
	must("/sbin/zpool", "create", "-o", "cachefile=none", "-o", "compatibility=legacy", "-O", "mountpoint=none", "-O", "canmount=off", name, path)
	actual, err := collect(ctx)
	if err != nil {
		t.Fatal("actual imported pool did not correlate", err)
	}
	found := 0
	for _, p := range actual.use {
		if p.Name == name {
			found++
			createdGUID = p.GUID
			if len(p.Disks) != 1 || !reflect.DeepEqual(p.Disks[0], selected) {
				t.Fatal("unmounted imported pool did not bind declared disk")
			}
		}
	}
	if found != 1 || createdGUID == "" {
		t.Fatal("owned pool missing from complete catalogue")
	}
	probe := func(raw []byte) (FilesystemSignature, FilesystemProbeCapture, []byte, []byte) {
		info, serial := must("/usr/sbin/diskinfo", path), must("/usr/sbin/diskinfo", "-s", path)
		capture := FilesystemProbeCapture{Device: path, Output: must("/usr/sbin/fstyp", "-u", path)}
		got, err := ParseFilesystemSignature(raw, info, serial, capture, inventory.DefaultLimits())
		if err != nil || got.Type != "zfs" || !reflect.DeepEqual(got.Disk.Provider, selected) {
			t.Fatal("actual ZFS signature did not bind declared disk", err)
		}
		return got, capture, info, serial
	}
	_, activeProbe, activeInfo, activeSerial := probe(actual.raw)
	activeReview, activeInputs := combined(actual, activeInfo, activeSerial, &activeProbe)
	joined := activeReview.Observation()
	if len(joined.Pools) != 1 || joined.Pools[0].Name != name || joined.Pools[0].GUID != createdGUID || !reflect.DeepEqual(joined.Disk.Provider, selected) || len(joined.Mounts)+len(joined.Swap) != 0 || !joined.SignatureAvailable || joined.Signature != "zfs" {
		t.Fatal("active imported pool/signature did not join selected disk")
	}
	if err := idleReview.Validate(activeInputs, inventory.DefaultLimits()); !errors.Is(err, ErrDrift) {
		t.Fatal("new actual pool use did not invalidate baseline", err)
	}
	fresh, err := collect(ctx)
	if err != nil {
		t.Fatal("fresh imported catalogue unavailable", err)
	}
	_, freshProbe, freshInfo, freshSerial := probe(fresh.raw)
	_, freshInputs := combined(fresh, freshInfo, freshSerial, &freshProbe)
	if err := activeReview.Validate(freshInputs, inventory.DefaultLimits()); err != nil {
		t.Fatal("unchanged actual use did not revalidate", err)
	}
	// Independent fixed six-column native row must identify the same pool GUID.
	matches := 0
	for _, line := range strings.Split(string(actual.list), "\n") {
		f := strings.Split(line, "\t")
		if len(f) == 6 && f[0] == name {
			matches++
			if f[1] != createdGUID {
				t.Fatal("independent catalogue GUID mismatch")
			}
		}
	}
	if matches != 1 {
		t.Fatal("independent catalogue row unavailable")
	}
	use := systemUse(actual)
	notSystemUsed(use)
	for _, m := range use.Mounts {
		if m.Source == name || strings.HasPrefix(m.Source, name+"/") {
			t.Fatal("owned pool unexpectedly mounted")
		}
	}
	var missing []PoolUseCapture
	for _, p := range actual.pools {
		if p.Name != name {
			missing = append(missing, p)
		}
	}
	if got, err := ParseImportedPoolUse(actual.raw, actual.list, missing, inventory.DefaultLimits()); err == nil || got != nil {
		t.Fatal("missing actual pool status accepted")
	}
	missingInputs := activeInputs
	missingInputs.PoolStatus = missing
	if got, err := ReviewDeviceUse(missingInputs, inventory.DefaultLimits()); err == nil || !reflect.DeepEqual(got.Observation(), DeviceUse{}) {
		t.Fatal("combined missing actual status escaped")
	}
	if !cleanup() {
		t.Fatal("owned pool teardown failed")
	}
	after, err := collect(ctx)
	if err != nil || !reflect.DeepEqual(after.use, baseline.use) {
		t.Fatal("foreign imported catalogue changed", err)
	}
	// The pool is gone from the actual imported catalogue, but the base probe
	// still recognizes its residual first-label metadata. Never call this blank.
	probeHash := strings.TrimSpace(string(must("/sbin/sha256", "-q", path)))
	if len(probeHash) != 64 {
		t.Fatal("closed fixture probe hash unavailable")
	}
	_, probeCapture, info, serial := probe(after.raw)
	removedReview, removedInputs := combined(after, info, serial, &probeCapture)
	removed := removedReview.Observation()
	if len(removed.Pools)+len(removed.Mounts)+len(removed.Swap) != 0 || !removed.SignatureAvailable || removed.Signature != "zfs" {
		t.Fatal("residual signature disappeared from joined facts")
	}
	if err := activeReview.Validate(removedInputs, inventory.DefaultLimits()); !errors.Is(err, ErrDrift) {
		t.Fatal("actual pool removal did not invalidate baseline", err)
	}
	if strings.TrimSpace(string(must("/sbin/sha256", "-q", path))) != probeHash {
		t.Fatal("filesystem probe changed closed fixture bytes")
	}
	probeCapture.Device = "/dev/" + boot.Name
	if got, err := ParseFilesystemSignature(after.raw, info, serial, probeCapture, inventory.DefaultLimits()); err == nil || !reflect.DeepEqual(got, FilesystemSignature{}) {
		t.Fatal("signature probe from another address accepted")
	}
	afterSystem := systemUse(after)
	if !reflect.DeepEqual(initialSystem, afterSystem) {
		t.Fatal("root/mount/swap backing changed")
	}
	for _, p := range disks {
		found := false
		for _, current := range after.graph.Providers {
			if current.ID == p.ID && reflect.DeepEqual(current, p) {
				found = true
			}
		}
		if !found {
			t.Fatal("fixture identity/geometry drifted")
		}
		if hash, ok := before[p.Name]; ok && strings.TrimSpace(string(must("/sbin/sha256", "-q", "/dev/"+p.Name))) != hash {
			t.Fatal("foreign fixture bytes changed")
		}
	}
	t.Log("DEVICE_SIGNATURE_ZFS_RESULT zfs_signature=pass removed_signature=pass probe_preservation=pass binding_denial=pass")
	t.Log("DEVICE_USE_POOL_RESULT imported_join=pass new_pool_drift=pass unchanged=pass residual_join=pass pool_removal_drift=pass missing_capture_denial=pass no_clearance=pass")
	t.Log("DEVICE_IMPORTED_POOL_RESULT unmounted_pool=pass catalogue_binding=pass missing_denial=pass fixture_teardown=pass foreign_preservation=pass root_preservation=pass")
}
