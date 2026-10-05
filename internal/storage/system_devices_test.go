// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

const systemGEOM = `<mesh><class id="dc"><name>DISK</name>
<geom id="d0"><name>ada0</name><provider id="p0"><name>ada0</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider></geom>
<geom id="d1"><name>ada1</name><provider id="p1"><name>ada1</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider></geom>
<geom id="d2"><name>ada2</name><provider id="p2"><name>ada2</name><mediasize>67108864</mediasize><sectorsize>4096</sectorsize></provider></geom></class>
<class id="pc"><name>PART</name><geom id="part"><name>ada0</name><consumer id="c0"><provider ref="p0"/></consumer><provider id="pp"><name>ada0p2</name><mediasize>33554432</mediasize><sectorsize>512</sectorsize></provider></geom></class>
<class id="lc"><name>LABEL</name><geom id="label"><name>ada0p2</name><consumer id="cl"><provider ref="pp"/></consumer><provider id="pl"><name>gpt/rootfs</name><mediasize>33554432</mediasize><sectorsize>512</sectorsize></provider></geom></class></mesh>`
const ufsMounts = "/dev/gpt/rootfs / ufs rw 1 1\ndevfs /dev devfs rw 0 0\n/dev/ada1 /fixture ufs ro 2 2\n"
const emptySwap = "Device 1K-blocks Used Avail Capacity\n"
const oneSwap = emptySwap + "/dev/ada2 65532 0 65532 0%\n"

func systemCapture() PoolUseCapture {
	return PoolUseCapture{Name: "tank", GUID: []byte("100\n"), StatusGUIDs: []byte(topologyGUIDRows), StatusPaths: []byte(topologyPathRows)}
}
func TestSystemDevicesMapUFSRootMountAndSwap(t *testing.T) {
	u, err := ParseSystemDeviceUse([]byte(systemGEOM), []byte(ufsMounts), []byte(oneSwap), nil, inventory.DefaultLimits())
	if err != nil || len(u.Mounts) != 3 || u.Mounts[0].Target != "/" || len(u.Mounts[0].Disks) != 1 || u.Mounts[0].Disks[0].Name != "ada0" || u.Mounts[0].Disks[0].Identity != inventory.IdentityMissing || len(u.Mounts[1].Disks) != 0 || u.Mounts[2].Disks[0].Name != "ada1" || len(u.Swap) != 1 || u.Swap[0].Disks[0].Name != "ada2" || u.Swap[0].SizeKiB != 65532 {
		t.Fatal("positive system backing lost or identity invented", err)
	}
}
func TestSystemDevicesBindZFSRootToWholePool(t *testing.T) {
	mounts := []byte("tank/ROOT/default / zfs rw 0 0\ntank/data /data zfs rw 0 0\n")
	u, err := ParseSystemDeviceUse([]byte(systemGEOM), mounts, []byte(emptySwap), []PoolUseCapture{systemCapture()}, inventory.DefaultLimits())
	if err != nil || len(u.Mounts) != 2 || u.Mounts[0].PoolGUID != "100" || len(u.Mounts[0].Disks) != 2 || u.Mounts[0].Disks[0].Name != "ada0" || u.Mounts[0].Disks[1].Name != "ada1" {
		t.Fatal("root pool membership not correlated", err)
	}
	for _, change := range []func(*PoolUseCapture){
		func(p *PoolUseCapture) { p.Name = "other" }, func(p *PoolUseCapture) { p.GUID = []byte("0100\n") },
		func(p *PoolUseCapture) {
			p.StatusPaths = []byte(strings.Replace(topologyPathRows, "/dev/ada1", "/dev/missing", 1))
		},
		func(p *PoolUseCapture) {
			p.StatusGUIDs = []byte(strings.Replace(topologyGUIDRows, "902 ONLINE", "902 DEGRADED", 1))
		},
		func(p *PoolUseCapture) {
			p.StatusPaths = []byte(strings.Replace(topologyPathRows, "/dev/ada1", "/tmp/disk", 1))
		},
	} {
		p := systemCapture()
		change(&p)
		if u, err := ParseSystemDeviceUse([]byte(systemGEOM), mounts, []byte(emptySwap), []PoolUseCapture{p}, inventory.DefaultLimits()); err == nil || !reflect.DeepEqual(u, SystemDeviceUse{}) {
			t.Fatal("contradictory/unsupported pool backing accepted")
		}
	}
	for _, pools := range [][]PoolUseCapture{nil, {systemCapture(), systemCapture()}} {
		if _, err := ParseSystemDeviceUse([]byte(systemGEOM), mounts, []byte(emptySwap), pools, inventory.DefaultLimits()); err == nil {
			t.Fatal("missing/duplicate pool projection accepted")
		}
	}
	if _, err := ParseSystemDeviceUse([]byte(systemGEOM), []byte(ufsMounts), []byte(emptySwap), []PoolUseCapture{systemCapture()}, inventory.DefaultLimits()); err == nil {
		t.Fatal("unrelated supplied pool accepted")
	}
}
func TestSystemDevicesRefuseUnavailableMountSwapOrBacking(t *testing.T) {
	for _, mounts := range []string{"", strings.TrimSuffix(ufsMounts, "\n"), ufsMounts + "/dev/ada2 /fixture ufs rw 0 0\n", strings.Replace(ufsMounts, "/ ufs", "/other ufs", 1), strings.Replace(ufsMounts, "/ ufs", "/ nfs", 1), strings.Replace(ufsMounts, "/dev/gpt/rootfs", "/dev/../ada0", 1), strings.Replace(ufsMounts, "/dev/ada1", "/dev/missing", 1), strings.Replace(ufsMounts, "rw 1 1", "rw,ro 1 1", 1), ufsMounts + "/data /alias nullfs rw 0 0\n", ufsMounts + "opaque /unknown unknown rw 0 0\n", strings.Replace(ufsMounts, "/fixture", "/fixture name", 1), strings.Replace(ufsMounts, "1 1", "01 1", 1)} {
		if u, err := ParseSystemDeviceUse([]byte(systemGEOM), []byte(mounts), []byte(oneSwap), nil, inventory.DefaultLimits()); err == nil || !reflect.DeepEqual(u, SystemDeviceUse{}) {
			t.Fatal("partial/ambiguous mount proof", err)
		}
	}
	for _, swap := range []string{"", strings.TrimSuffix(oneSwap, "\n"), strings.Replace(oneSwap, "1K-blocks", "1M-blocks", 1), oneSwap + "/dev/ada2 65532 0 65532 0%\n", strings.Replace(oneSwap, "/dev/ada2", "<NFSfile>", 1), strings.Replace(oneSwap, "/dev/ada2", "/dev/missing", 1), strings.Replace(oneSwap, "65532 0 65532", "65532 1 65532", 1), strings.Replace(oneSwap, "0%", "101%", 1), oneSwap + "Total 65532 0 65532 0%\n"} {
		if u, err := ParseSystemDeviceUse([]byte(systemGEOM), []byte(ufsMounts), []byte(swap), nil, inventory.DefaultLimits()); err == nil || !reflect.DeepEqual(u, SystemDeviceUse{}) {
			t.Fatal("partial/contradictory swap proof", err)
		}
	}
	for _, geom := range []string{strings.Replace(systemGEOM, "ref=\"pp\"", "ref=\"pl\"", 1), strings.Replace(systemGEOM, "ref=\"pp\"", "ref=\"missing\"", 1), strings.Replace(systemGEOM, "<name>DISK</name>", "<name>MD</name>", 1), strings.Replace(systemGEOM, "<name>ada1</name>", "<name>other</name>", 1)} {
		if u, err := ParseSystemDeviceUse([]byte(geom), []byte(ufsMounts), []byte(oneSwap), nil, inventory.DefaultLimits()); err == nil || !reflect.DeepEqual(u, SystemDeviceUse{}) {
			t.Fatal("unresolved/cyclic/logical backing accepted", err)
		}
	}
}
func TestSystemSwapTotalsAndBudgets(t *testing.T) {
	two := oneSwap + "/dev/ada1 65532 4 65528 0%\nTotal 131064 4 131060 0%\n"
	if u, err := ParseSystemDeviceUse([]byte(systemGEOM), []byte(ufsMounts), []byte(two), nil, inventory.DefaultLimits()); err != nil || len(u.Swap) != 2 {
		t.Fatal("complete swap totals refused", err)
	}
	for _, swap := range []string{strings.Replace(two, "Total 131064", "Total 131065", 1), strings.Split(two, "Total")[0], two + "/dev/ada0 4 0 4 0%\n"} {
		if _, err := ParseSystemDeviceUse([]byte(systemGEOM), []byte(ufsMounts), []byte(swap), nil, inventory.DefaultLimits()); err == nil {
			t.Fatal("absent/wrong/nonterminal totals accepted")
		}
	}
	for _, limits := range []inventory.Limits{{MaxBytes: len(systemGEOM), MaxRecords: 100, MaxFieldBytes: 100, MaxXMLDepth: 32, MaxXMLElements: 1000}, {MaxBytes: 1 << 20, MaxRecords: 3, MaxFieldBytes: 100, MaxXMLDepth: 32, MaxXMLElements: 1000}, {MaxBytes: 1 << 20, MaxRecords: 100, MaxFieldBytes: 3, MaxXMLDepth: 32, MaxXMLElements: 1000}} {
		if u, err := ParseSystemDeviceUse([]byte(systemGEOM), []byte(ufsMounts), []byte(oneSwap), nil, limits); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(u, SystemDeviceUse{}) {
			t.Fatal("aggregate/field/record budget ignored", err)
		}
	}
}
func FuzzSystemDeviceUse(f *testing.F) {
	f.Add(systemGEOM, ufsMounts, oneSwap)
	f.Fuzz(func(t *testing.T, geom, mount, swap string) {
		l := inventory.DefaultLimits()
		l.MaxBytes = 64 << 10
		l.MaxRecords = 1024
		l.MaxFieldBytes = 1024
		u, err := ParseSystemDeviceUse([]byte(geom), []byte(mount), []byte(swap), nil, l)
		if err != nil {
			if !reflect.DeepEqual(u, SystemDeviceUse{}) {
				t.Fatal("partial system-use projection")
			}
			return
		}
		roots := 0
		for _, m := range u.Mounts {
			if m.Target == "/" {
				roots++
				if len(m.Disks) == 0 {
					t.Fatal("root backing missing")
				}
			}
			for _, d := range m.Disks {
				if d.Kind != inventory.DiskCandidate {
					t.Fatal("logical provider promoted")
				}
			}
		}
		if roots != 1 {
			t.Fatal("missing/ambiguous root")
		}
	})
}

func TestSystemUseRefusesPotentiallyTruncatedNativeNames(t *testing.T) {
	for _, n := range []int{26, 1018} {
		name := strings.Repeat("x", n)
		extra := `<geom id="long"><name>ada2</name><consumer id="longc"><provider ref="p2"/></consumer><provider id="longp"><name>` + name + `</name><mediasize>67108864</mediasize><sectorsize>4096</sectorsize></provider></geom>`
		geom := strings.Replace(systemGEOM, "</class></mesh>", extra+"</class></mesh>", 1)
		mounts, swap := ufsMounts, oneSwap
		if n == 26 {
			swap = strings.Replace(swap, "/dev/ada2", "/dev/"+name, 1)
		} else {
			mounts = strings.Replace(mounts, "/dev/ada1", "/dev/"+name, 1)
		}
		if u, err := ParseSystemDeviceUse([]byte(geom), []byte(mounts), []byte(swap), nil, inventory.DefaultLimits()); err == nil || !reflect.DeepEqual(u, SystemDeviceUse{}) {
			t.Fatal("potentially truncated name matched another provider")
		}
	}
	mounts := strings.Replace(ufsMounts, "/fixture", "/"+strings.Repeat("x", 1022), 1)
	if _, err := ParseSystemDeviceUse([]byte(systemGEOM), []byte(mounts), []byte(oneSwap), nil, inventory.DefaultLimits()); err == nil {
		t.Fatal("potentially truncated mount target accepted")
	}
}

func TestSystemUseBoundsAggregateRecordsAndRowFloods(t *testing.T) {
	l := inventory.DefaultLimits()
	l.MaxRecords = 25
	if u, err := ParseSystemDeviceUse([]byte(systemGEOM), []byte(ufsMounts), []byte(oneSwap), nil, l); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(u, SystemDeviceUse{}) {
		t.Fatal("individually bounded sources bypassed aggregate record budget", err)
	}
	l = inventory.DefaultLimits()
	l.MaxRecords = 100
	if _, err := ParseSystemDeviceUse([]byte(systemGEOM), []byte(ufsMounts+strings.Repeat("devfs /dev devfs rw 0 0\n", 101)), []byte(oneSwap), nil, l); !errors.Is(err, ErrLimit) {
		t.Fatal("row flood not bounded", err)
	}
	if u, err := ParseSystemDeviceUse([]byte(systemGEOM), []byte(ufsMounts+strings.Repeat("x ", 10000)+"\n"), []byte(oneSwap), nil, inventory.DefaultLimits()); err == nil || !reflect.DeepEqual(u, SystemDeviceUse{}) {
		t.Fatal("field flood accepted")
	}
}
