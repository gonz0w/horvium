// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"strconv"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

func validTimes(start, finish, outerStart, outerFinish time.Time) bool {
	return !start.IsZero() && !finish.IsZero() && !finish.Before(start) && !start.Before(outerStart) && !finish.After(outerFinish)
}
func validateSnapshot(s inventory.Snapshot) error {
	if s.StartedAt.IsZero() || s.FinishedAt.Before(s.StartedAt) {
		return ErrProtocol
	}
	for _, m := range []sectionMeta{metadata(s.Host), metadata(s.GEOM), metadata(s.Pools), metadata(s.Datasets)} {
		if !validTimes(m.StartedAt, m.FinishedAt, s.StartedAt, s.FinishedAt) {
			return ErrProtocol
		}
		if m.Status != inventory.OK && m.Status != inventory.Partial && m.Status != inventory.Unavailable {
			return ErrProtocol
		}
		if (m.Status == inventory.OK) != (len(m.Problems) == 0) {
			return ErrProtocol
		}
		seen := map[string]bool{}
		for _, p := range m.Problems {
			if seen[p.Source] {
				return ErrProtocol
			}
			seen[p.Source] = true
			switch p.Source {
			case "freebsd-version", "architecture", "physical-memory", "zfs-version", "geom", "pools", "datasets":
			default:
				return ErrProtocol
			}
			switch p.Code {
			case "limit", "malformed", "timeout", "cancelled", "permission", "unavailable", "failed":
			default:
				return ErrProtocol
			}
		}
	}
	if s.Host.Status == inventory.OK {
		h := s.Host.Data
		if h.InstalledKernel == "" || h.RunningKernel == "" || h.Userland == "" || h.Architecture == "" || h.PhysicalMemory == nil || *h.PhysicalMemory == 0 || h.ZFSUserland == "" || h.ZFSKernel == "" {
			return ErrProtocol
		}
	}
	if s.Host.Status == inventory.Unavailable && s.Host.Data != (inventory.Host{}) {
		return ErrProtocol
	}
	if s.GEOM.Status == inventory.Partial || s.Pools.Status == inventory.Partial || s.Datasets.Status == inventory.Partial {
		return ErrProtocol
	}
	if s.GEOM.Status == inventory.Unavailable && (len(s.GEOM.Data.Classes)+len(s.GEOM.Data.Geoms)+len(s.GEOM.Data.Providers)) != 0 {
		return ErrProtocol
	}
	if s.Pools.Status == inventory.Unavailable && len(s.Pools.Data) != 0 || s.Datasets.Status == inventory.Unavailable && len(s.Datasets.Data) != 0 {
		return ErrProtocol
	}
	classes := map[string]string{}
	geoms := map[string]string{}
	providers := map[string]bool{}
	ids := map[string]bool{}
	unique := func(id string) bool {
		if id == "" || ids[id] {
			return false
		}
		ids[id] = true
		return true
	}
	for _, v := range s.GEOM.Data.Classes {
		if !unique(v.ID) || v.Name == "" {
			return ErrProtocol
		}
		classes[v.ID] = v.Name
	}
	for _, v := range s.GEOM.Data.Geoms {
		if !unique(v.ID) || v.Name == "" || classes[v.ClassID] == "" {
			return ErrProtocol
		}
		geoms[v.ID] = classes[v.ClassID]
		for _, c := range v.Consumers {
			if !unique(c.ID) {
				return ErrProtocol
			}
		}
	}
	for _, v := range s.GEOM.Data.Providers {
		if !unique(v.ID) || v.Name == "" || geoms[v.GeomID] != v.Class || v.Class == "" || v.SectorSize == 0 {
			return ErrProtocol
		}
		providers[v.ID] = true
		expected := inventory.Logical
		switch v.Class {
		case "DISK":
			expected = inventory.DiskCandidate
		case "PART":
			expected = inventory.Partition
		case "MD":
			expected = inventory.Memory
		}
		if v.Kind != expected {
			return ErrProtocol
		}
		if v.Kind != inventory.DiskCandidate {
			if v.Identity != inventory.IdentityNotApplicable {
				return ErrProtocol
			}
		} else {
			switch v.Identity {
			case inventory.IdentityUnique, inventory.IdentityMissing, inventory.IdentityAmbiguous:
			default:
				return ErrProtocol
			}
			if (len(v.Identifiers) == 0) != (v.Identity == inventory.IdentityMissing) {
				return ErrProtocol
			}
		}
		for _, i := range v.Identifiers {
			key := "ident"
			if i.Kind == "lunid" {
				key = "lunid"
			}
			if (i.Kind != "serial" && i.Kind != "lunid") || i.Value == "" || (i.Source != "geom.config."+key && i.Source != "provider.config."+key) {
				return ErrProtocol
			}
		}
	}
	for _, v := range s.GEOM.Data.Geoms {
		for _, c := range v.Consumers {
			if c.ProviderID != "" && !providers[c.ProviderID] {
				return ErrProtocol
			}
		}
	}
	poolNames := map[string]bool{}
	poolGUIDs := map[string]bool{}
	for _, v := range s.Pools.Data {
		if v.Name == "" || !validGUID(v.GUID) || poolNames[v.Name] || poolGUIDs[v.GUID] || v.Allocated > v.Size || v.Free > v.Size {
			return ErrProtocol
		}
		poolNames[v.Name] = true
		poolGUIDs[v.GUID] = true
		switch v.Health {
		case "ONLINE", "DEGRADED", "FAULTED", "OFFLINE", "UNAVAIL", "REMOVED", "SUSPENDED":
		default:
			return ErrProtocol
		}
	}
	names := map[string]bool{}
	guids := map[string]bool{}
	for _, v := range s.Datasets.Data {
		if v.Name == "" || !validGUID(v.GUID) || names[v.Name] || guids[v.GUID] {
			return ErrProtocol
		}
		names[v.Name] = true
		guids[v.GUID] = true
	}
	return nil
}
func validGUID(s string) bool {
	n, err := strconv.ParseUint(s, 10, 64)
	return err == nil && n > 0 && strconv.FormatUint(n, 10) == s
}
