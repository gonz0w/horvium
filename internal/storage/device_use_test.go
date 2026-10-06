// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

func deviceUseFixture() DeviceUseCapture {
	return DeviceUseCapture{GEOM: []byte(signatureGEOM), Mounts: []byte(ufsMounts), Swap: []byte(emptySwap), DiskInfo: []byte(signatureInfo), DiskSerial: []byte(signatureSerial), Filesystem: &FilesystemProbeCapture{Device: "/dev/ada1", Output: []byte("ufs\n")}}
}

func TestDeviceUseReviewCombinesSelectedPositiveFacts(t *testing.T) {
	c := deviceUseFixture()
	r, err := ReviewDeviceUse(c, inventory.DefaultLimits())
	u := r.Observation()
	if err != nil || u.Disk.Provider.Name != "ada1" || len(u.Mounts) != 1 || u.Mounts[0].Target != "/fixture" || len(u.Swap) != 0 || len(u.Pools) != 0 || !u.SignatureAvailable || u.Signature != "ufs" {
		t.Fatal("selected mount/signature facts were not retained", err)
	}
	c.Mounts = []byte("tank/ROOT/default / zfs rw 0 0\ntank/data /data zfs rw 0 0\n")
	c.Pools, c.PoolStatus, c.Filesystem = []byte(importedList), []PoolUseCapture{systemCapture()}, nil
	r, err = ReviewDeviceUse(c, inventory.DefaultLimits())
	u = r.Observation()
	if err != nil || len(u.Mounts) != 2 || u.Mounts[0].Target != "/" || len(u.Pools) != 1 || u.Pools[0].GUID != "100" || len(u.Pools[0].Disks) != 2 || u.SignatureAvailable || u.Signature != "" {
		t.Fatal("whole-pool/root backing or unavailable signature lost", err)
	}
	c = deviceUseFixture()
	c.Swap = []byte(emptySwap + "/dev/ada1 65532 0 65532 0%\n")
	r, err = ReviewDeviceUse(c, inventory.DefaultLimits())
	if err != nil || len(r.Observation().Swap) != 1 {
		t.Fatal("selected active swap not retained", err)
	}
}

func TestDeviceUseNoFactsNeverMeansCleared(t *testing.T) {
	c := deviceUseFixture()
	c.Mounts, c.Filesystem = []byte(strings.Replace(ufsMounts, "/dev/ada1 /fixture ufs ro 2 2\n", "", 1)), nil
	r, err := ReviewDeviceUse(c, inventory.DefaultLimits())
	u := r.Observation()
	if err != nil || u.Disk.Provider.Name == "" || len(u.Mounts)+len(u.Swap)+len(u.Pools) != 0 || u.SignatureAvailable || u.Signature != "" {
		t.Fatal("empty positive set manufactured signature clearance", err)
	}
	if err := (DeviceUseReview{}).Validate(c, inventory.DefaultLimits()); !errors.Is(err, ErrUnavailable) {
		t.Fatal("zero baseline validated", err)
	}
}

func TestDeviceUseRequiresFreshReviewForSelectedDrift(t *testing.T) {
	for _, change := range []func(*DeviceUseCapture){
		func(c *DeviceUseCapture) { c.Mounts = []byte(strings.Replace(ufsMounts, "/fixture", "/changed", 1)) },
		func(c *DeviceUseCapture) { c.Swap = []byte(emptySwap + "/dev/ada1 65532 0 65532 0%\n") },
		func(c *DeviceUseCapture) {
			c.Pools, c.PoolStatus = []byte(importedList), []PoolUseCapture{systemCapture()}
		},
		func(c *DeviceUseCapture) { c.Filesystem = nil },
		func(c *DeviceUseCapture) { c.Filesystem.Output = []byte("zfs\n") },
		func(c *DeviceUseCapture) {
			c.GEOM = []byte(strings.Replace(signatureGEOM, "sig-unique", "new-serial", 1))
			c.DiskSerial = []byte("new-serial\n")
		},
		func(c *DeviceUseCapture) {
			c.GEOM = []byte(strings.ReplaceAll(signatureGEOM, "ada1", "ada3"))
			c.DiskInfo = []byte(strings.Replace(signatureInfo, "ada1", "ada3", 1))
			c.Mounts = []byte(strings.ReplaceAll(ufsMounts, "ada1", "ada3"))
			c.Filesystem.Device = "/dev/ada3"
		},
		func(c *DeviceUseCapture) {
			c.GEOM = []byte(strings.ReplaceAll(signatureGEOM, "67108864", "134217728"))
			c.DiskInfo = []byte(strings.ReplaceAll(strings.ReplaceAll(signatureInfo, "67108864", "134217728"), "131072", "262144"))
		},
	} {
		baseline, err := ReviewDeviceUse(deviceUseFixture(), inventory.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		fresh := deviceUseFixture()
		change(&fresh)
		if err := baseline.Validate(fresh, inventory.DefaultLimits()); !errors.Is(err, ErrDrift) {
			t.Fatal("selected device/use drift ignored", err)
		}
	}
}

func TestDeviceUseNormalizesOrderingAndTransientIDs(t *testing.T) {
	c := deviceUseFixture()
	r, err := ReviewDeviceUse(c, inventory.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	c.GEOM = []byte(strings.ReplaceAll(strings.ReplaceAll(signatureGEOM, `id="d1"`, `id="newgeom"`), `id="p1"`, `id="newprovider"`))
	c.Mounts = []byte("/dev/ada1 /fixture ufs ro 2 2\ndevfs /dev devfs rw 0 0\n/dev/gpt/rootfs / ufs rw 1 1\n")
	c.Swap = []byte(oneSwap) // Unrelated disk use does not change selected facts.
	if err := r.Validate(c, inventory.DefaultLimits()); err != nil {
		t.Fatal("ordering, transient IDs or unrelated swap caused selected drift", err)
	}
	if r.Observation().Disk.Provider.ID != "p1" {
		t.Fatal("retained source graph ID was discarded")
	}
}

func TestDeviceUseRetainsPoolGUIDAndWholeMembershipDrift(t *testing.T) {
	for _, change := range []func(*DeviceUseCapture){
		func(c *DeviceUseCapture) {
			c.Pools = []byte(strings.Replace(importedList, "\t100\t", "\t200\t", 1))
			c.PoolStatus[0].GUID = []byte("200\n")
		},
		func(c *DeviceUseCapture) {
			c.PoolStatus[0].StatusPaths = []byte(strings.Replace(topologyPathRows, "/dev/ada0", "/dev/ada2", 1))
		},
	} {
		r, err := ReviewDeviceUse(deviceUseFixtureWithPool(), inventory.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		fresh := deviceUseFixtureWithPool()
		change(&fresh)
		if err := r.Validate(fresh, inventory.DefaultLimits()); !errors.Is(err, ErrDrift) {
			t.Fatal("pool identity or another backing member changed unnoticed", err)
		}
	}
}

func TestDeviceUseRefusesPartialMismatchedAndOversizedCaptures(t *testing.T) {
	for _, change := range []func(*DeviceUseCapture){
		func(c *DeviceUseCapture) { c.Mounts = nil }, func(c *DeviceUseCapture) { c.Swap = nil },
		func(c *DeviceUseCapture) { c.Pools = []byte(importedList) },
		func(c *DeviceUseCapture) { c.PoolStatus = []PoolUseCapture{systemCapture()} },
		func(c *DeviceUseCapture) { c.Mounts = []byte("tank/ROOT/default / zfs rw 0 0\n") },
		func(c *DeviceUseCapture) { c.DiskSerial = []byte("other\n") },
		func(c *DeviceUseCapture) { c.Filesystem.Device = "/dev/ada2" },
		func(c *DeviceUseCapture) { c.Filesystem.Output = []byte("unknown\n") },
	} {
		c := deviceUseFixture()
		change(&c)
		if r, err := ReviewDeviceUse(c, inventory.DefaultLimits()); err == nil || !reflect.DeepEqual(r.Observation(), DeviceUse{}) {
			t.Fatal("partial combined observation escaped", err)
		}
	}
	c := deviceUseFixture()
	l := inventory.DefaultLimits()
	l.MaxBytes = len(c.GEOM) + len(c.Mounts) + len(c.Swap) + len(c.DiskInfo) + len(c.DiskSerial) + len(c.Filesystem.Device) + len(c.Filesystem.Output) - 1
	if _, err := ReviewDeviceUse(c, l); !errors.Is(err, ErrLimit) {
		t.Fatal("combined byte limit ignored", err)
	}
	l = inventory.DefaultLimits()
	l.MaxRecords = 30
	if _, err := ReviewDeviceUse(c, l); !errors.Is(err, ErrLimit) {
		t.Fatal("independent parsers reused full record budget", err)
	}
	if _, err := ReviewDeviceUse(c, inventory.Limits{}); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid limits accepted", err)
	}
}

func TestDeviceUseReviewOwnsNestedFactsAndInput(t *testing.T) {
	c := deviceUseFixture()
	c.Pools, c.PoolStatus = []byte(importedList), []PoolUseCapture{systemCapture()}
	r, err := ReviewDeviceUse(c, inventory.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	want := r.Observation()
	copy := r.Observation()
	copy.Disk.Provider.Identifiers[0].Value = "changed"
	copy.Mounts[0].Disks[0].Identifiers[0].Value = "changed"
	copy.Pools[0].Disks[1].Identifiers[0].Value = "changed"
	c.GEOM[0], c.PoolStatus[0].GUID[0], c.Filesystem.Output[0] = 'x', '9', 'x'
	if !reflect.DeepEqual(r.Observation(), want) {
		t.Fatal("caller changed retained review")
	}
	if err := r.Validate(deviceUseFixtureWithPool(), inventory.DefaultLimits()); err != nil {
		t.Fatal("review was corrupted by caller", err)
	}
}

func deviceUseFixtureWithPool() DeviceUseCapture {
	c := deviceUseFixture()
	c.Pools, c.PoolStatus = []byte(importedList), []PoolUseCapture{systemCapture()}
	return c
}

func FuzzDeviceUseReview(f *testing.F) {
	f.Add(signatureGEOM, ufsMounts, emptySwap, "", signatureInfo, signatureSerial, "/dev/ada1", "ufs\n", "", "", "", "", true)
	f.Fuzz(func(t *testing.T, geom, mount, swap, list, info, serial, address, output, pool, guid, ids, paths string, probe bool) {
		c := DeviceUseCapture{GEOM: []byte(geom), Mounts: []byte(mount), Swap: []byte(swap), Pools: []byte(list), DiskInfo: []byte(info), DiskSerial: []byte(serial)}
		if pool != "" {
			c.PoolStatus = []PoolUseCapture{{Name: pool, GUID: []byte(guid), StatusGUIDs: []byte(ids), StatusPaths: []byte(paths)}}
		}
		if probe {
			c.Filesystem = &FilesystemProbeCapture{Device: address, Output: []byte(output)}
		}
		l := inventory.DefaultLimits()
		l.MaxBytes = 64 << 10
		l.MaxRecords = 1024
		l.MaxFieldBytes = 1024
		r, err := ReviewDeviceUse(c, l)
		if err != nil {
			if !reflect.DeepEqual(r.Observation(), DeviceUse{}) {
				t.Fatal("partial review")
			}
			return
		}
		if r.Observation().Disk.Provider.Identity != inventory.IdentityUnique || r.Validate(c, l) != nil {
			t.Fatal("unbound or unstable accepted review")
		}
	})
}
