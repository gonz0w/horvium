// SPDX-License-Identifier: BSD-3-Clause
package storage

import "strings"

// Private kernel projection; never a request or persisted configuration schema.
// Capacity/free-block counters are not stable mount identity or durability proof.
type propertyStateBacking struct {
	kind, source, target  string
	fsid                  [2]int32
	flags                 uint64
	readonly, softupdates bool
}

func checkPropertyStateBacking(b propertyStateBacking, u PoolSystemUse, owners map[string]bool) error {
	if !u.initialized {
		return ErrUnavailable
	}
	if b.readonly || b.kind != "ufs" && b.kind != "zfs" || b.kind == "ufs" && b.softupdates {
		return ErrStateDirectory
	}
	if len(b.source) >= 1023 || len(b.target) >= 1023 || b.source == "" || !systemPath(b.target) {
		return ErrStateDirectory
	}
	system := false
	for _, target := range []string{"/", "/boot", "/etc", "/usr", "/var", "/root"} {
		if b.target == target || target != "/" && strings.HasPrefix(b.target, target+"/") {
			system = true
		}
	}
	if !system {
		return ErrDenied
	}
	for _, m := range u.system.Mounts {
		if m.Source == b.source && m.Target == b.target && m.Type == b.kind {
			if len(m.Disks) == 0 || m.PoolGUID != "" && owners[m.PoolGUID] {
				return ErrDenied
			}
			// An owned imported pool must not share physical backing with state,
			// regardless of its mount spelling or whether it is the selected pool.
			stateIDs := make(map[string]bool, len(m.Disks))
			for _, d := range m.Disks {
				stateIDs[d.ID] = true
			}
			for _, p := range u.pools {
				if !owners[p.GUID] {
					continue
				}
				for _, d := range p.Disks {
					if stateIDs[d.ID] {
						return ErrDenied
					}
				}
			}
			return nil
		}
	}
	return ErrDrift
}
func checkPropertyStateJobs(v JobReservations, guid string) error {
	if !v.ready || !validGUID(guid) {
		return ErrInvalid
	}
	// Pending work anywhere still needs independent executor/recovery checks.
	if v.nonterminal != 0 {
		return ErrDenied
	}
	for _, pool := range v.pools {
		if pool.PoolGUID == guid && len(pool.Jobs) > 0 {
			return ErrDenied
		}
	}
	return nil
}
