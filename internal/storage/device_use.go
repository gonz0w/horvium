// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"sort"
	"strings"

	"github.com/gonz0w/horvium/internal/inventory"
)

// DeviceUseCapture supplies one complete fresh native capture set. PoolStatus
// covers exactly the all-imported-pool catalogue; mounted pools are selected from
// it internally. A nil Filesystem means unavailable, never absent signatures.
// The caller must establish successful execution, completeness, target and
// provenance; these separate reads are not atomic. Inputs are exclusive during
// the call. This is an internal model, not a transport or persisted schema.
type DeviceUseCapture struct {
	GEOM, Mounts, Swap, Pools, DiskInfo, DiskSerial []byte
	PoolStatus                                      []PoolUseCapture
	Filesystem                                      *FilesystemProbeCapture
	PoolLabels                                      *PoolLabelCapture
}

// DeviceUse retains positive selected-disk facts, including the full backing
// of a matching mount/swap/pool. Empty sets are not complete device clearance.
// No eligible/unused/enrolled state or write authority is produced.
type DeviceUse struct {
	Disk                inventory.DiskObservation
	Access              inventory.AccessMode
	Mounts              []MountUse
	Swap                []SwapUse
	Pools               []ImportedPoolUse
	Signature           string
	SignatureAvailable  bool
	PoolLabels          []PoolLabelConfiguration
	PoolLabelsAvailable bool
}

// DeviceUseReview owns an immutable observation baseline. It is not an operation
// plan: no session, expiry, policy, reservation, persistence or submission exists.
type DeviceUseReview struct {
	use         DeviceUse
	initialized bool
}

// ReviewDeviceUse binds selected positive use facts to independent diskinfo and
// serial/GEOM correspondence. It performs no I/O and cannot supply eligibility.
func ReviewDeviceUse(c DeviceUseCapture, limits inventory.Limits) (DeviceUseReview, error) {
	zero := DeviceUseReview{}
	if limits.Validate() != nil {
		return zero, ErrInvalid
	}
	remaining := limits.MaxBytes
	take := func(n int) bool {
		if n > remaining {
			return false
		}
		remaining -= n
		return true
	}
	for _, data := range [][]byte{c.GEOM, c.Mounts, c.Swap, c.Pools, c.DiskInfo, c.DiskSerial} {
		if !take(len(data)) {
			return zero, ErrLimit
		}
	}
	if len(c.PoolStatus) > limits.MaxRecords {
		return zero, ErrLimit
	}
	for _, p := range c.PoolStatus {
		for _, n := range []int{len(p.Name), len(p.GUID), len(p.StatusGUIDs), len(p.StatusPaths)} {
			if !take(n) {
				return zero, ErrLimit
			}
		}
	}
	stages := 5 // mount preparse, disk correspondence/access, system use, imported use
	if c.Filesystem != nil {
		stages++
		if !take(len(c.Filesystem.Device)) || !take(len(c.Filesystem.Output)) {
			return zero, ErrLimit
		}
	}
	if c.PoolLabels != nil {
		stages++
		if !take(len(c.PoolLabels.Device)) || !take(len(c.PoolLabels.Output)) {
			return zero, ErrLimit
		}
	}
	// Each existing bounded parser gets a disjoint conservative quota. Account
	// for repeated graph/row parsing instead of giving every stage the full cap.
	// Refuse rather than borrow unused quota or silently truncate projections.
	perStage := limits
	perStage.MaxRecords /= stages
	perStage.MaxXMLElements /= stages // Access reads XML twice; mount preparse does not.
	if perStage.MaxRecords <= 2 || perStage.MaxXMLElements == 0 {
		return zero, ErrLimit
	}
	diskLimits := perStage
	diskLimits.MaxRecords -= 2 // independent diskinfo and serial scalar records
	disk, err := inventory.ParseDiskObservation(c.GEOM, c.DiskInfo, c.DiskSerial, diskLimits)
	if err != nil {
		if errors.Is(err, inventory.ErrLimit) {
			return zero, ErrLimit
		}
		if errors.Is(err, inventory.ErrDiskCorrelation) {
			return zero, ErrDrift
		}
		return zero, ErrInvalid
	}
	imported, err := ParseImportedPoolUse(c.GEOM, c.Pools, c.PoolStatus, perStage)
	if err != nil {
		return zero, err
	}
	b := systemBudget{limits: perStage, remaining: perStage.MaxRecords}
	mounts, err := mountTable(c.Mounts, &b)
	if err != nil {
		return zero, err
	}
	required := make(map[string]bool)
	for _, m := range mounts {
		if m.Type == "zfs" {
			required[strings.Split(m.Source, "/")[0]] = true
		}
	}
	var mountedPools []PoolUseCapture
	for _, p := range c.PoolStatus {
		if required[p.Name] {
			mountedPools = append(mountedPools, p)
		}
	}
	system, err := ParseSystemDeviceUse(c.GEOM, c.Mounts, c.Swap, mountedPools, perStage)
	if err != nil {
		return zero, err
	}
	access, err := inventory.ParseDiskAccess(c.GEOM, c.DiskInfo, c.DiskSerial, perStage)
	if err != nil {
		if errors.Is(err, inventory.ErrLimit) {
			return zero, ErrLimit
		}
		return zero, ErrUnavailable
	}
	if !reflect.DeepEqual(access.Disk, disk) {
		return zero, ErrDrift
	}
	use := DeviceUse{Disk: disk, Access: access.Mode}
	selected := func(disks []inventory.Provider) bool {
		for _, d := range disks {
			if d.ID == disk.Provider.ID {
				return true
			}
		}
		return false
	}
	for _, m := range system.Mounts {
		if selected(m.Disks) {
			use.Mounts = append(use.Mounts, m)
		}
	}
	for _, s := range system.Swap {
		if selected(s.Disks) {
			use.Swap = append(use.Swap, s)
		}
	}
	for _, p := range imported {
		if selected(p.Disks) {
			use.Pools = append(use.Pools, p)
		}
	}
	if c.Filesystem != nil {
		sig, err := ParseFilesystemSignature(c.GEOM, c.DiskInfo, c.DiskSerial, *c.Filesystem, perStage)
		if err != nil {
			return zero, err
		}
		if !reflect.DeepEqual(sig.Disk, disk) {
			return zero, ErrDrift
		}
		use.Signature, use.SignatureAvailable = sig.Type, true
	}
	if c.PoolLabels != nil {
		labels, err := ParsePoolLabels(c.GEOM, c.DiskInfo, c.DiskSerial, *c.PoolLabels, perStage)
		if err != nil {
			return zero, err
		}
		if !reflect.DeepEqual(labels.Disk, disk) {
			return zero, ErrDrift
		}
		use.PoolLabels, use.PoolLabelsAvailable = labels.Configurations, true
	}
	return DeviceUseReview{use: use, initialized: true}, nil
}

// Observation returns independent nested arrays. Positive evidence does not
// become ownership or clearance through this accessor.
func (r DeviceUseReview) Observation() DeviceUse {
	if !r.initialized {
		return DeviceUse{}
	}
	return cloneDeviceUse(r.use)
}

// Validate reparses a complete fresh set and compares selected facts. Even an
// unchanged result grants no launch authority, freshness proof or external fence.
// Names/geometry/identifier provenance/use drift require a new baseline; GEOM's
// observation-local IDs and source row ordering alone do not.
func (r DeviceUseReview) Validate(c DeviceUseCapture, limits inventory.Limits) error {
	if !r.initialized {
		return ErrUnavailable
	}
	fresh, err := ReviewDeviceUse(c, limits)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(normalizeDeviceUse(r.use), normalizeDeviceUse(fresh.use)) {
		return ErrDrift
	}
	return nil
}

func cloneDeviceUse(u DeviceUse) DeviceUse {
	cloneDisks := func(disks []inventory.Provider) []inventory.Provider {
		if disks == nil {
			return nil
		}
		out := append([]inventory.Provider(nil), disks...)
		for i := range out {
			out[i].Identifiers = append([]inventory.Identifier(nil), disks[i].Identifiers...)
		}
		return out
	}
	u.Disk.Provider = cloneDisks([]inventory.Provider{u.Disk.Provider})[0]
	u.Mounts = append([]MountUse(nil), u.Mounts...)
	for i := range u.Mounts {
		u.Mounts[i].Disks = cloneDisks(u.Mounts[i].Disks)
	}
	u.Swap = append([]SwapUse(nil), u.Swap...)
	for i := range u.Swap {
		u.Swap[i].Disks = cloneDisks(u.Swap[i].Disks)
	}
	u.Pools = append([]ImportedPoolUse(nil), u.Pools...)
	for i := range u.Pools {
		u.Pools[i].Disks = cloneDisks(u.Pools[i].Disks)
	}
	u.PoolLabels = append([]PoolLabelConfiguration(nil), u.PoolLabels...)
	for i := range u.PoolLabels {
		u.PoolLabels[i].Labels = append([]int(nil), u.PoolLabels[i].Labels...)
	}
	return u
}

func normalizeDeviceUse(u DeviceUse) DeviceUse {
	u = cloneDeviceUse(u)
	normalize := func(p *inventory.Provider) {
		p.ID, p.GeomID = "", ""
		sort.Slice(p.Identifiers, func(i, j int) bool {
			a, b := p.Identifiers[i], p.Identifiers[j]
			if a.Kind != b.Kind {
				return a.Kind < b.Kind
			}
			if a.Source != b.Source {
				return a.Source < b.Source
			}
			return a.Value < b.Value
		})
	}
	normalize(&u.Disk.Provider)
	for i := range u.Mounts {
		for j := range u.Mounts[i].Disks {
			normalize(&u.Mounts[i].Disks[j])
		}
	}
	for i := range u.Swap {
		for j := range u.Swap[i].Disks {
			normalize(&u.Swap[i].Disks[j])
		}
	}
	for i := range u.Pools {
		for j := range u.Pools[i].Disks {
			normalize(&u.Pools[i].Disks[j])
		}
	}
	return u
}
