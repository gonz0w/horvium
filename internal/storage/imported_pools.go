// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"sort"
	"strings"

	"github.com/gonz0w/horvium/internal/inventory"
)

// ImportedPoolUse retains positive membership for an imported pool, including
// pools with no mounted filesystems and unidentified disks. It is not ownership,
// complete device-use clearance, or authority to create, import or destroy pools.
type ImportedPoolUse struct {
	Name, GUID, Health string
	Disks              []inventory.Provider
}

// ParseImportedPoolUse correlates complete successful all-pool list output
// (name,guid,size,alloc,free,health) with independent GUID-value/paired status
// captures for exactly that catalogue and a fresh complete GEOM graph. Captures
// are not atomic and their syntax cannot prove that an omitted pool was absent.
// Unsupported/contradictory projections return no rows. No native commands run.
func ParseImportedPoolUse(geom, list []byte, captures []PoolUseCapture, limits inventory.Limits) ([]ImportedPoolUse, error) {
	if limits.Validate() != nil {
		return nil, ErrInvalid
	}
	remaining := limits.MaxBytes
	take := func(n int) bool {
		if n > remaining {
			return false
		}
		remaining -= n
		return true
	}
	if !take(len(geom)) || !take(len(list)) || len(captures) > limits.MaxRecords {
		return nil, ErrLimit
	}
	for _, p := range captures {
		if !take(len(p.Name)) || !take(len(p.GUID)) || !take(len(p.StatusGUIDs)) || !take(len(p.StatusPaths)) {
			return nil, ErrLimit
		}
	}
	b := systemBudget{limits: limits, remaining: limits.MaxRecords}
	r, err := newSystemResolver(geom, &b)
	if err != nil {
		return nil, err
	}
	if err := b.spend(strings.Count(string(list), "\n")); err != nil {
		return nil, err
	}
	catalogue, err := inventory.ParsePools(list, limits)
	if err != nil {
		if errors.Is(err, inventory.ErrLimit) {
			return nil, ErrLimit
		}
		return nil, ErrInvalid
	}
	if len(catalogue) != len(captures) {
		return nil, ErrUnavailable
	}
	byName := make(map[string]inventory.Pool)
	for _, p := range catalogue {
		if !validName(p.Name) || strings.ContainsAny(p.Name, "/@#") {
			return nil, ErrUnavailable
		}
		byName[p.Name] = p
	}
	observed := make(map[string]bool)
	result := make([]ImportedPoolUse, 0, len(catalogue))
	for _, p := range captures {
		if err := b.spend(1); err != nil {
			return nil, err
		}
		if err := systemField(p.Name, limits); err != nil {
			return nil, err
		}
		row, ok := byName[p.Name]
		if !ok || observed[p.Name] {
			return nil, ErrUnavailable
		}
		backing, err := resolvePoolUse(p, r)
		if err != nil {
			return nil, err
		}
		if backing.guid != row.GUID || backing.health != row.Health {
			return nil, ErrDrift
		}
		if err := b.spend(len(backing.disks)); err != nil {
			return nil, err
		}
		result = append(result, ImportedPoolUse{Name: row.Name, GUID: row.GUID, Health: row.Health, Disks: append([]inventory.Provider(nil), backing.disks...)})
		observed[p.Name] = true
	}
	sort.Slice(result, func(i, j int) bool { return result[i].GUID < result[j].GUID })
	return result, nil
}
