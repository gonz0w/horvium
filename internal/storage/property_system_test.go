// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

func propertySystemFixture(t *testing.T, mounts, swap string) PoolSystemUse {
	t.Helper()
	geom := strings.Replace(systemGEOM, "<name>ada0</name>", "<name>boot0</name>", -1)
	// Data pool is disjoint from the UFS system disk, with a separate third leaf.
	geom = strings.Replace(geom, "</mesh>", `<class id="extra"><name>DISK</name><geom id="d3"><name>ada0</name><provider id="p3"><name>ada0</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider></geom></class></mesh>`, 1)
	u, err := ParsePoolSystemUse([]byte(geom), []byte(mounts), []byte(swap), []byte("tank\t100\t67108864\t0\t67108864\tONLINE\n"), []PoolUseCapture{systemCapture()}, inventory.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestPropertySystemExclusionsAllowOwnDataMountWithoutAuthority(t *testing.T) {
	_, o, i := fixture()
	o.Eligible = false
	u := propertySystemFixture(t, "/dev/gpt/rootfs / ufs rw 1 1\ntank/data /srv/data zfs rw 0 0\n", emptySwap)
	if err := CheckPropertySystemExclusions(o, u); err != nil {
		t.Fatal(err)
	}
	p, _, _ := fixture()
	if _, err := Build(instant, "session", p, o, i); !errors.Is(err, ErrUnavailable) {
		t.Fatal("exclusions granted eligibility", err)
	}
	u.System().Mounts[0].Disks[0].Name = "changed"
	u.Pools()[0].Disks[0].Name = "changed"
	if err := CheckPropertySystemExclusions(o, u); err != nil {
		t.Fatal("accessors alias proof", err)
	}
}

func TestPropertySystemExclusionsDenySystemMountsAndDeviceOverlap(t *testing.T) {
	_, o, _ := fixture()
	o.Eligible = false
	for _, target := range []string{"/", "/boot", "/boot/efi", "/etc", "/usr/local", "/var/db", "/root/state"} {
		mounts := "/dev/gpt/rootfs / ufs rw 1 1\ntank/data " + target + " zfs rw 0 0\n"
		if target == "/" {
			mounts = "tank/data / zfs rw 0 0\n"
		}
		u := propertySystemFixture(t, mounts, emptySwap)
		if err := CheckPropertySystemExclusions(o, u); !errors.Is(err, ErrDenied) {
			t.Fatal("system pool allowed", target, err)
		}
	}
	for _, mounts := range []string{"/dev/ada0 / ufs rw 1 1\n", "/dev/gpt/rootfs / ufs rw 1 1\n/dev/ada1 /foreign ufs rw 0 0\n"} {
		u := propertySystemFixture(t, mounts, emptySwap)
		if err := CheckPropertySystemExclusions(o, u); !errors.Is(err, ErrDenied) {
			t.Fatal("disk overlap allowed", err)
		}
	}
	u := propertySystemFixture(t, "/dev/gpt/rootfs / ufs rw 1 1\n", emptySwap+"/dev/ada1 65532 0 65532 0%\n")
	if err := CheckPropertySystemExclusions(o, u); !errors.Is(err, ErrDenied) {
		t.Fatal("swap overlap", err)
	}
	// Pure normalized-model seam: other imported pools retain positive holds.
	u = propertySystemFixture(t, "/dev/gpt/rootfs / ufs rw 1 1\n", emptySwap)
	other := u.pools[0]
	other.GUID = "999"
	other.Name = "foreign"
	u.pools = append(u.pools, other)
	if err := CheckPropertySystemExclusions(o, u); !errors.Is(err, ErrDenied) {
		t.Fatal("foreign pool overlap", err)
	}
}

func TestPropertySystemExclusionsRejectMissingOrDriftingProof(t *testing.T) {
	_, o, _ := fixture()
	o.Eligible = false
	if err := CheckPropertySystemExclusions(o, PoolSystemUse{}); err == nil {
		t.Fatal("zero proof")
	}
	u := propertySystemFixture(t, "/dev/gpt/rootfs / ufs rw 1 1\n", emptySwap)
	o.Topology.Nodes[1].Address = "/dev/other"
	if err := CheckPropertySystemExclusions(o, u); !errors.Is(err, ErrDrift) {
		t.Fatal("topology drift")
	}
}

func TestPoolSystemUseAggregateBudgetsAndMissingCaptures(t *testing.T) {
	for _, mode := range []string{"missing", "bytes", "records"} {
		limits := inventory.DefaultLimits()
		captures := []PoolUseCapture{systemCapture()}
		switch mode {
		case "missing":
			captures = nil
		case "bytes":
			limits.MaxBytes = len(systemGEOM)
		case "records":
			limits.MaxRecords = 8
		}
		u, err := ParsePoolSystemUse([]byte(systemGEOM), []byte(ufsMounts), []byte(emptySwap), []byte("tank\t100\t67108864\t0\t67108864\tONLINE\n"), captures, limits)
		if err == nil || !reflect.DeepEqual(u, PoolSystemUse{}) {
			t.Fatal("partial/budget use", mode, err)
		}
	}
}

func FuzzPoolSystemUse(f *testing.F) {
	f.Add(systemGEOM, ufsMounts, emptySwap)
	f.Fuzz(func(t *testing.T, geom, mount, swap string) {
		if len(geom)+len(mount)+len(swap) > 32<<10 {
			t.Skip()
		}
		limits := inventory.DefaultLimits()
		limits.MaxBytes = 64 << 10
		limits.MaxRecords = 1024
		u, err := ParsePoolSystemUse([]byte(geom), []byte(mount), []byte(swap), nil, nil, limits)
		if err != nil && !reflect.DeepEqual(u, PoolSystemUse{}) {
			t.Fatal("partial parsed use")
		}
	})
}
