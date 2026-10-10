// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"strconv"
	"strings"
)

// Both domains must be matched to complete fresh system facts while the owner
// holds both original leases. This is observation, not restore/launch clearance.
func checkExecutionSystemDomains(journal, ledger propertyStateBacking, journalID, ledgerID executionBackingIdentity, catalog StateCatalog, use PoolSystemUse) error {
	if !use.initialized {
		return ErrUnavailable
	}
	owners := make(map[string]bool, len(catalog.Ownership))
	if len(catalog.Ownership) > maxRecords {
		return ErrLimit
	}
	for _, o := range catalog.Ownership {
		if !validGUID(o.PoolGUID) || owners[o.PoolGUID] {
			return ErrStateRecord
		}
		owners[o.PoolGUID] = true
		found := false
		for _, p := range use.pools {
			if p.GUID == o.PoolGUID && len(p.Disks) > 0 {
				found = true
			}
		}
		// An exported/offline owner has unresolved physical backing. Never treat it
		// as disjoint merely because native imported-pool enumeration omitted it.
		if !found {
			return ErrUnavailable
		}
	}
	for _, domain := range []struct {
		mount propertyStateBacking
		id    executionBackingIdentity
	}{{journal, journalID}, {ledger, ledgerID}} {
		b, id := domain.mount, domain.id
		if !id.valid() || b.kind != id.kind {
			return ErrDrift
		}
		if id.kind == "ufs" && id.ufs != ([2]uint32{uint32(b.fsid[0]), uint32(b.fsid[1])}) {
			return ErrDrift
		}
		if e := checkPropertyStateBacking(b, use, owners); e != nil {
			return e
		}
		if id.kind == "zfs" {
			guid := strconv.FormatUint(id.pool, 10)
			matched := false
			for _, m := range use.system.Mounts {
				if m.Source == b.source && m.Target == b.target && m.Type == b.kind && m.PoolGUID == guid {
					matched = true
				}
			}
			if !matched {
				return ErrDrift
			}
			matched = false
			for _, p := range use.pools {
				if p.GUID == guid && p.Name == strings.Split(b.source, "/")[0] && len(p.Disks) > 0 {
					matched = true
				}
			}
			if !matched {
				return ErrDrift
			}
		}
	}
	return nil
}
