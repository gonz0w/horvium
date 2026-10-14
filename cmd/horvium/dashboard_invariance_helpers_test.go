// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"errors"
	"github.com/gonz0w/horvium/internal/inventory"
)

// Counts only, with ownership from the independent fixture witness. Diagnostic
// current/cache observations never become expectations for the frozen oracle.
func dashboardValueChanges(original, current inventory.Snapshot, ownedPool string, owned []inventory.Dataset) (map[string]map[string]int, error) {
	invalid := func() (map[string]map[string]int, error) {
		return nil, errors.New("uncertain original native identities or ownership")
	}
	a, err := inventory.CloneBounded(original, inventory.DefaultLimits())
	if err != nil || !a.Complete() {
		return invalid()
	}
	b, err := inventory.CloneBounded(current, inventory.DefaultLimits())
	if err != nil || !b.Complete() {
		return invalid()
	}
	if len(a.Pools.Data) > 500 || len(a.Datasets.Data) > 500 || len(a.Pools.Data) != len(b.Pools.Data) || len(a.Datasets.Data) != len(b.Datasets.Data) || len(owned) != 100 {
		return invalid()
	}
	pools := map[string]inventory.Pool{}
	datasets := map[string]inventory.Dataset{}
	for _, p := range b.Pools.Data {
		if _, ok := pools[p.GUID]; ok {
			return invalid()
		}
		pools[p.GUID] = p
	}
	for _, d := range b.Datasets.Data {
		if _, ok := datasets[d.GUID]; ok {
			return invalid()
		}
		datasets[d.GUID] = d
	}
	originalPools := map[string]bool{}
	originalDatasets := map[string]inventory.Dataset{}
	for _, p := range a.Pools.Data {
		if originalPools[p.GUID] {
			return invalid()
		}
		originalPools[p.GUID] = true
	}
	for _, d := range a.Datasets.Data {
		if _, ok := originalDatasets[d.GUID]; ok {
			return invalid()
		}
		originalDatasets[d.GUID] = d
	}
	if !originalPools[ownedPool] {
		return invalid()
	}
	witness := map[string]bool{}
	for _, d := range owned {
		value, ok := originalDatasets[d.GUID]
		if !ok || value.Name != d.Name || witness[d.GUID] {
			return invalid()
		}
		witness[d.GUID] = true
	}
	out := map[string]map[string]int{"owned-pools": {}, "other-pools": {}, "owned-datasets": {}, "other-datasets": {}}
	for _, p := range a.Pools.Data {
		other, ok := pools[p.GUID]
		if !ok {
			return invalid()
		}
		role := "other-pools"
		if p.GUID == ownedPool {
			role = "owned-pools"
		}
		counts := out[role]
		for key, different := range map[string]bool{"name": p.Name != other.Name, "size": p.Size != other.Size, "allocated": p.Allocated != other.Allocated, "free": p.Free != other.Free, "health": p.Health != other.Health} {
			if different {
				counts[key]++
			}
		}
	}
	for _, d := range a.Datasets.Data {
		other, ok := datasets[d.GUID]
		if !ok {
			return invalid()
		}
		role := "other-datasets"
		if witness[d.GUID] {
			role = "owned-datasets"
		}
		counts := out[role]
		for key, different := range map[string]bool{"name": d.Name != other.Name, "used": d.Used != other.Used, "available": d.Available != other.Available, "referenced": d.Referenced != other.Referenced} {
			if different {
				counts[key]++
			}
		}
	}
	return out, nil
}
