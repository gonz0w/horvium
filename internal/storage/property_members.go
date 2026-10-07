// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"strings"

	"github.com/gonz0w/horvium/internal/inventory"
)

// ValidatePropertyMembers requires an exact explicit root member baseline for
// every supported native leaf. Its inputs must come from trusted root policy and
// independently correlated native reads. It grants no enrollment/write authority
// and never changes Eligible; system/store/recovery/admission gates are separate.
func ValidatePropertyMembers(p Policy, o Observation, disks []inventory.DiskObservation) error {
	if len(disks) > maxVdevs {
		return ErrLimit
	}
	canonical, err := normalizeOwnership(p)
	if err != nil {
		return err
	}
	if len(canonical.Members) == 0 {
		return ErrDenied
	}
	if err = ValidatePropertyPolicyBinding(canonical, o); err != nil {
		return err
	}
	if err = VerifyPoolTopology(o.PoolGUID, o.PoolName, o.Topology); err != nil {
		return err
	}
	leaves := map[string]string{}
	for _, v := range o.Topology.Nodes {
		if v.Kind == "leaf" {
			name, ok := strings.CutPrefix(v.Address, "/dev/")
			if !ok {
				return ErrDrift
			}
			leaves[v.GUID] = name
		}
	}
	if len(leaves) != len(canonical.Members) || len(disks) != len(leaves) {
		return ErrDrift
	}
	byName := map[string]inventory.DiskObservation{}
	for _, d := range disks {
		provider := d.Provider
		if _, ok := byName[provider.Name]; ok {
			return ErrDrift
		}
		if provider.Class != "DISK" || provider.Kind != inventory.DiskCandidate || provider.Identity != inventory.IdentityUnique || provider.SectorSize == 0 || provider.MediaSize == 0 || provider.MediaSize%provider.SectorSize != 0 || d.SectorCount != provider.MediaSize/provider.SectorSize {
			return ErrDrift
		}
		byName[provider.Name] = d
	}
	for _, m := range canonical.Members {
		d, ok := byName[leaves[m.VdevGUID]]
		if !ok || d.Provider.MediaSize != m.MediaSize || d.Provider.SectorSize != m.SectorSize || d.StripeSize != m.StripeSize || d.StripeOffset != m.StripeOffset {
			return ErrDrift
		}
		matched := false
		for _, id := range d.Provider.Identifiers {
			if id.Kind != "serial" {
				continue
			}
			if id.Value != m.Serial || (id.Source != "geom.config.ident" && id.Source != "provider.config.ident") {
				return ErrDrift
			}
			if id.Source == m.SerialSource {
				matched = true
			}
		}
		if !matched {
			return ErrDrift
		}
	}
	return nil
}
