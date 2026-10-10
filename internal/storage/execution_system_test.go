// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"github.com/gonz0w/horvium/internal/inventory"
	"testing"
)

func executionSystemFixture(t *testing.T) (propertyStateBacking, executionBackingIdentity, StateCatalog, PoolSystemUse) {
	t.Helper()
	u := propertySystemFixture(t, "/dev/gpt/rootfs / ufs rw 1 1\ntank/data /srv/data zfs rw 0 0\n", emptySwap)
	b := propertyStateBacking{kind: "ufs", source: "/dev/gpt/rootfs", target: "/", fsid: [2]int32{11, 22}}
	id := executionBackingIdentity{kind: "ufs", ufs: [2]uint32{11, 22}}
	return b, id, StateCatalog{Ownership: []OwnershipSummary{{PoolGUID: "100"}}}, u
}

func TestExecutionSystemDomainsRequireBothCompleteIndependentBackings(t *testing.T) {
	b, id, c, u := executionSystemFixture(t)
	if e := checkExecutionSystemDomains(b, b, id, id, c, u); e != nil {
		t.Fatal("complete UFS pair", e)
	}
	// Distinct native ZFS domains in one unowned system pool remain observations.
	z := u
	z.system.Mounts = append([]MountUse(nil), u.system.Mounts...)
	z.system.Mounts = append(z.system.Mounts, MountUse{Source: "boot/journal", Target: "/var/horvium/journal", Type: "zfs", PoolGUID: "200", Disks: u.system.Mounts[0].Disks}, MountUse{Source: "boot/ledger", Target: "/var/horvium/ledger", Type: "zfs", PoolGUID: "200", Disks: u.system.Mounts[0].Disks})
	z.pools = append(append([]ImportedPoolUse(nil), u.pools...), ImportedPoolUse{GUID: "200", Name: "boot", Disks: u.system.Mounts[0].Disks})
	j, l := propertyStateBacking{kind: "zfs", source: "boot/journal", target: "/var/horvium/journal"}, propertyStateBacking{kind: "zfs", source: "boot/ledger", target: "/var/horvium/ledger"}
	ji, li := executionBackingIdentity{kind: "zfs", pool: 200, dataset: 201}, executionBackingIdentity{kind: "zfs", pool: 200, dataset: 202}
	if e := checkExecutionSystemDomains(j, l, ji, li, c, z); e != nil {
		t.Fatal("complete ZFS pair", e)
	}
	if e := checkExecutionSystemDomains(b, l, id, li, c, z); e != nil {
		t.Fatal("mixed pair", e)
	}
	for _, role := range []string{"journal", "ledger"} {
		for _, mode := range []string{"readonly", "softupdates", "unknown-kind", "missing-mount", "foreign-source", "foreign-target", "data-mount", "no-disks", "owned-pool", "physical-overlap", "zero-use", "unresolved-owner", "duplicate-owner", "invalid-owner", "kind-drift", "ufs-id-drift", "invalid-id"} {
			t.Run(role+"/"+mode, func(t *testing.T) {
				bb, ii, cc, uu := b, id, c, u
				cc.Ownership = append([]OwnershipSummary(nil), c.Ownership...)
				uu.system.Mounts = append([]MountUse(nil), u.system.Mounts...)
				switch mode {
				case "readonly":
					bb.readonly = true
				case "softupdates":
					bb.softupdates = true
				case "unknown-kind":
					bb.kind = "tmpfs"
				case "missing-mount":
					uu.system.Mounts = uu.system.Mounts[1:]
				case "foreign-source":
					bb.source = "/dev/other"
				case "foreign-target":
					bb.target = "/var"
				case "data-mount":
					bb = propertyStateBacking{kind: "zfs", source: "tank/data", target: "/srv/data"}
					ii = executionBackingIdentity{kind: "zfs", pool: 100, dataset: 101}
				case "no-disks":
					uu.system.Mounts[0].Disks = nil
				case "owned-pool":
					uu.system.Mounts[0].PoolGUID = "100"
				case "physical-overlap":
					uu.system.Mounts[0].Disks = []inventory.Provider{u.pools[0].Disks[0]}
				case "zero-use":
					uu = PoolSystemUse{}
				case "unresolved-owner":
					cc.Ownership = append(cc.Ownership, OwnershipSummary{PoolGUID: "999"})
				case "duplicate-owner":
					cc.Ownership = append(cc.Ownership, cc.Ownership[0])
				case "invalid-owner":
					cc.Ownership[0].PoolGUID = "0100"
				case "kind-drift":
					ii = executionBackingIdentity{kind: "zfs", pool: 200, dataset: 201}
				case "ufs-id-drift":
					ii.ufs[0]++
				case "invalid-id":
					ii = executionBackingIdentity{}
				}
				jb, lb, jid, lid := b, b, id, id
				if role == "journal" {
					jb, jid = bb, ii
				} else {
					lb, lid = bb, ii
				}
				if e := checkExecutionSystemDomains(jb, lb, jid, lid, cc, uu); e == nil {
					t.Fatal("unsafe pair accepted")
				}
			})
		}
	}
	for _, mode := range []string{"foreign-guid", "foreign-name", "missing-pool", "invalid-dataset"} {
		t.Run("zfs/"+mode, func(t *testing.T) {
			zz, jid := z, ji
			zz.pools = append([]ImportedPoolUse(nil), z.pools...)
			zz.system.Mounts = append([]MountUse(nil), z.system.Mounts...)
			switch mode {
			case "foreign-guid":
				zz.system.Mounts[2].PoolGUID = "300"
			case "foreign-name":
				zz.pools[1].Name = "foreign"
			case "missing-pool":
				zz.pools = zz.pools[:1]
			case "invalid-dataset":
				jid.dataset = 0
			}
			if e := checkExecutionSystemDomains(j, l, jid, li, c, zz); e == nil {
				t.Fatal("foreign ZFS correspondence")
			}
		})
	}
}
