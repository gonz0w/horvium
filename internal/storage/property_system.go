// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"reflect"
	"sort"
	"strings"

	"github.com/gonz0w/horvium/internal/inventory"
)

// PoolSystemUse retains a complete parsed positive-use view. Construction is
// private; native freshness/completeness still belongs to the trusted collector.
// It never declares an unused/enrolled disk or grants property eligibility.
type PoolSystemUse struct {
	initialized bool
	system      SystemDeviceUse
	pools       []ImportedPoolUse
	topologies  []PoolTopology
}

func (u PoolSystemUse) System() SystemDeviceUse {
	c := cloneDeviceUse(DeviceUse{Mounts: u.system.Mounts, Swap: u.system.Swap})
	return SystemDeviceUse{Mounts: c.Mounts, Swap: c.Swap}
}
func (u PoolSystemUse) Pools() []ImportedPoolUse {
	return cloneDeviceUse(DeviceUse{Pools: u.pools}).Pools
}

// ParsePoolSystemUse joins every imported pool to mount/swap backing. The shared
// byte ceiling is checked first; disjoint record allowances bound all four parse
// stages. Each missing/unsupported/contradictory capture denies the entire view.
func ParsePoolSystemUse(geom, mount, swap, list []byte, captures []PoolUseCapture, limits inventory.Limits) (PoolSystemUse, error) {
	if limits.Validate() != nil {
		return PoolSystemUse{}, ErrInvalid
	}
	if limits.MaxRecords < 4 || len(captures) > limits.MaxRecords/4 {
		return PoolSystemUse{}, ErrLimit
	}
	remaining := limits.MaxBytes
	take := func(n int) bool {
		if n > remaining {
			return false
		}
		remaining -= n
		return true
	}
	for _, data := range [][]byte{geom, mount, swap, list} {
		if !take(len(data)) {
			return PoolSystemUse{}, ErrLimit
		}
	}
	for _, c := range captures {
		if !take(len(c.Name)) || !take(len(c.GUID)) || !take(len(c.StatusGUIDs)) || !take(len(c.StatusPaths)) {
			return PoolSystemUse{}, ErrLimit
		}
	}
	stage := limits
	stage.MaxRecords /= 4
	pools, err := ParseImportedPoolUse(geom, list, captures, stage)
	if err != nil {
		return PoolSystemUse{}, err
	}
	b := systemBudget{limits: stage, remaining: stage.MaxRecords}
	mounts, err := mountTable(mount, &b)
	if err != nil {
		return PoolSystemUse{}, err
	}
	needed := map[string]bool{}
	for _, m := range mounts {
		if m.Type == "zfs" {
			needed[strings.Split(m.Source, "/")[0]] = true
		}
	}
	var mounted []PoolUseCapture
	for _, c := range captures {
		if needed[c.Name] {
			mounted = append(mounted, c)
		}
	}
	system, err := ParseSystemDeviceUse(geom, mount, swap, mounted, stage)
	if err != nil {
		return PoolSystemUse{}, err
	}
	left := stage.MaxRecords
	topologies := make([]PoolTopology, 0, len(captures))
	for _, c := range captures {
		topology, e := ParsePoolTopology(c.StatusGUIDs, c.StatusPaths, c.Name, strings.TrimSuffix(string(c.GUID), "\n"))
		if e != nil {
			return PoolSystemUse{}, e
		}
		if len(topology.Nodes)+1 > left {
			return PoolSystemUse{}, ErrLimit
		}
		left -= len(topology.Nodes) + 1
		topologies = append(topologies, topology)
	}
	sort.Slice(topologies, func(i, j int) bool { return topologies[i].PoolGUID < topologies[j].PoolGUID })
	return PoolSystemUse{initialized: true, system: system, pools: pools, topologies: topologies}, nil
}

// CheckPropertySystemExclusions compares the selected native topology to the
// complete imported/use view and refuses system mounts or physical-disk overlap
// with other mounts, swap and imported pools. Own ordinary data mounts are allowed.
// Passing proves only these observed exclusions, not stable provider identity,
// root enrollment, store/recovery/admission authority or absence of later drift.
func CheckPropertySystemExclusions(o Observation, u PoolSystemUse) error {
	if !u.initialized {
		return ErrUnavailable
	}
	if err := preflight(Policy{}, o); err != nil {
		return err
	}
	if _, err := indexCollectedObservation(o); err != nil {
		return err
	}
	var selected *ImportedPoolUse
	for i := range u.pools {
		p := &u.pools[i]
		if p.GUID == o.PoolGUID {
			if p.Name != o.PoolName || p.Health != o.Topology.Health {
				return ErrDrift
			}
			selected = p
		}
	}
	if selected == nil {
		return ErrUnavailable
	}
	matched := false
	for _, topology := range u.topologies {
		if topology.PoolGUID == o.PoolGUID {
			matched = reflect.DeepEqual(topology, o.Topology)
		}
	}
	if !matched {
		return ErrDrift
	}
	ids := map[string]bool{}
	for _, disk := range selected.Disks {
		ids[disk.ID] = true
	}
	if len(ids) == 0 {
		return ErrUnavailable
	}
	overlap := func(disks []inventory.Provider) bool {
		for _, d := range disks {
			if ids[d.ID] {
				return true
			}
		}
		return false
	}
	for _, m := range u.system.Mounts {
		if m.PoolGUID == o.PoolGUID {
			for _, target := range []string{"/", "/boot", "/etc", "/usr", "/var", "/root"} {
				if m.Target == target || target != "/" && strings.HasPrefix(m.Target, target+"/") {
					return ErrDenied
				}
			}
		} else if overlap(m.Disks) {
			return ErrDenied
		}
	}
	for _, swap := range u.system.Swap {
		if overlap(swap.Disks) {
			return ErrDenied
		}
	}
	for _, pool := range u.pools {
		if pool.GUID != o.PoolGUID && overlap(pool.Disks) {
			return ErrDenied
		}
	}
	return nil
}

// CheckPropertySystemProviders requires correspondence between independently
// collected selected providers and the later complete system-use graph. Graph
// IDs are local to each capture; identity/provenance/geometry must match exactly.
// It grants no eligibility and complements, rather than replaces, exclusions.
func CheckPropertySystemProviders(guid string, u PoolSystemUse, disks []inventory.DiskObservation) error {
	if !u.initialized {
		return ErrUnavailable
	}
	if !validGUID(guid) {
		return ErrInvalid
	}
	if len(disks) > maxVdevs {
		return ErrLimit
	}
	for _, pool := range u.pools {
		if pool.GUID != guid {
			continue
		}
		if len(pool.Disks) != len(disks) || len(disks) == 0 {
			return ErrDrift
		}
		seen := map[string]bool{}
		byName := map[string]inventory.Provider{}
		for _, p := range pool.Disks {
			if _, ok := byName[p.Name]; ok {
				return ErrDrift
			}
			byName[p.Name] = p
		}
		for _, d := range disks {
			p, ok := byName[d.Provider.Name]
			expected := d.Provider
			expected.ID, expected.GeomID, p.ID, p.GeomID = "", "", "", ""
			if !ok || seen[expected.Name] || !reflect.DeepEqual(expected, p) {
				return ErrDrift
			}
			seen[expected.Name] = true
		}
		return nil
	}
	return ErrUnavailable
}
