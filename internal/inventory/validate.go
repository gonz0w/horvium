// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"strconv"
	"time"
)

func validTimes(start, finish, outerStart, outerFinish time.Time) bool {
	return !start.IsZero() && !finish.IsZero() && !finish.Before(start) && !start.Before(outerStart) && !finish.After(outerFinish)
}

// ValidateSnapshot checks the typed observation contract, independent of transport.
// Callers must bound its records/strings before building validation maps.
func ValidateSnapshot(s Snapshot) error {
	if s.StartedAt.IsZero() || s.FinishedAt.Before(s.StartedAt) {
		return ErrMalformed
	}
	for _, m := range []snapshotMeta{snapshotMetadata(s.Host), snapshotMetadata(s.GEOM), snapshotMetadata(s.Pools), snapshotMetadata(s.Datasets)} {
		if !validTimes(m.StartedAt, m.FinishedAt, s.StartedAt, s.FinishedAt) {
			return ErrMalformed
		}
		if m.Status != OK && m.Status != Partial && m.Status != Unavailable {
			return ErrMalformed
		}
		if (m.Status == OK) != (len(m.Problems) == 0) {
			return ErrMalformed
		}
		seen := map[string]bool{}
		for _, p := range m.Problems {
			if seen[p.Source] {
				return ErrMalformed
			}
			seen[p.Source] = true
			switch p.Source {
			case "freebsd-version", "architecture", "physical-memory", "zfs-version", "geom", "pools", "datasets":
			default:
				return ErrMalformed
			}
			switch p.Code {
			case "limit", "malformed", "timeout", "cancelled", "permission", "unavailable", "failed":
			default:
				return ErrMalformed
			}
		}
	}
	if s.Host.Status == OK {
		h := s.Host.Data
		if h.InstalledKernel == "" || h.RunningKernel == "" || h.Userland == "" || h.Architecture == "" || h.PhysicalMemory == nil || *h.PhysicalMemory == 0 || h.ZFSUserland == "" || h.ZFSKernel == "" {
			return ErrMalformed
		}
	}
	if s.Host.Status == Unavailable && s.Host.Data != (Host{}) {
		return ErrMalformed
	}
	if s.GEOM.Status == Partial || s.Pools.Status == Partial || s.Datasets.Status == Partial {
		return ErrMalformed
	}
	if s.GEOM.Status == Unavailable && (len(s.GEOM.Data.Classes)+len(s.GEOM.Data.Geoms)+len(s.GEOM.Data.Providers)) != 0 {
		return ErrMalformed
	}
	if s.Pools.Status == Unavailable && len(s.Pools.Data) != 0 || s.Datasets.Status == Unavailable && len(s.Datasets.Data) != 0 {
		return ErrMalformed
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
			return ErrMalformed
		}
		classes[v.ID] = v.Name
	}
	for _, v := range s.GEOM.Data.Geoms {
		if !unique(v.ID) || v.Name == "" || classes[v.ClassID] == "" {
			return ErrMalformed
		}
		geoms[v.ID] = classes[v.ClassID]
		for _, c := range v.Consumers {
			if !unique(c.ID) {
				return ErrMalformed
			}
		}
	}
	for _, v := range s.GEOM.Data.Providers {
		if !unique(v.ID) || v.Name == "" || geoms[v.GeomID] != v.Class || v.Class == "" || v.SectorSize == 0 {
			return ErrMalformed
		}
		providers[v.ID] = true
		expected := Logical
		switch v.Class {
		case "DISK":
			expected = DiskCandidate
		case "PART":
			expected = Partition
		case "MD":
			expected = Memory
		}
		if v.Kind != expected {
			return ErrMalformed
		}
		if v.Kind != DiskCandidate {
			if v.Identity != IdentityNotApplicable {
				return ErrMalformed
			}
		} else {
			switch v.Identity {
			case IdentityUnique, IdentityMissing, IdentityAmbiguous:
			default:
				return ErrMalformed
			}
			if (len(v.Identifiers) == 0) != (v.Identity == IdentityMissing) {
				return ErrMalformed
			}
		}
		for _, i := range v.Identifiers {
			key := "ident"
			if i.Kind == "lunid" {
				key = "lunid"
			}
			if (i.Kind != "serial" && i.Kind != "lunid") || i.Value == "" || (i.Source != "geom.config."+key && i.Source != "provider.config."+key) {
				return ErrMalformed
			}
		}
	}
	for _, v := range s.GEOM.Data.Geoms {
		for _, c := range v.Consumers {
			if c.ProviderID != "" && !providers[c.ProviderID] {
				return ErrMalformed
			}
		}
	}
	poolNames := map[string]bool{}
	poolGUIDs := map[string]bool{}
	for _, v := range s.Pools.Data {
		if v.Name == "" || !validGUID(v.GUID) || poolNames[v.Name] || poolGUIDs[v.GUID] || v.Allocated > v.Size || v.Free > v.Size {
			return ErrMalformed
		}
		poolNames[v.Name] = true
		poolGUIDs[v.GUID] = true
		switch v.Health {
		case "ONLINE", "DEGRADED", "FAULTED", "OFFLINE", "UNAVAIL", "REMOVED", "SUSPENDED":
		default:
			return ErrMalformed
		}
	}
	names := map[string]bool{}
	guids := map[string]bool{}
	for _, v := range s.Datasets.Data {
		if v.Name == "" || !validGUID(v.GUID) || names[v.Name] || guids[v.GUID] {
			return ErrMalformed
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

type snapshotMeta struct {
	Status                Status
	StartedAt, FinishedAt time.Time
	Problems              []Problem
}

func snapshotMetadata[T any](s Section[T]) snapshotMeta {
	return snapshotMeta{s.Status, s.StartedAt, s.FinishedAt, s.Problems}
}
