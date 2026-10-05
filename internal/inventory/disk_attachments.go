// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"errors"
	"sort"
)

var ErrDiskAttachments = errors.New("disk has unsupported or contradictory attachments")

// DiskAttachments records a narrow DEV/disk-ident correspondence. It does not
// prove absent signatures, pool/mount/swap/boot use, locality or write eligibility.
// Aliases remain logical providers, never additional independent disks.
type DiskAttachments struct {
	Disk    Provider
	Aliases []Provider
}

// ParseDiskAttachments examines one direct uniquely identified disk and its
// reachable incoming attachments in a complete successful GEOM capture. Only
// terminal DEV nodes and exact whole-disk disk-ident LABEL -> DEV chains are
// recognized. Partition and all other attachments refuse this partial proof.
// No native command, raw probe, runtime wiring or mutation authority is added.
func ParseDiskAttachments(data []byte, name string, limits Limits) (DiskAttachments, error) {
	if err := limits.Validate(); err != nil {
		return DiskAttachments{}, err
	}
	if len(name) > limits.MaxFieldBytes || len(name) > limits.MaxBytes || len(data) > limits.MaxBytes-len(name) {
		return DiskAttachments{}, ErrLimit
	}
	if !diskAddress(name) {
		return DiskAttachments{}, ErrDiskAttachments
	}
	graph, err := ParseGEOM(data, limits)
	if err != nil {
		return DiskAttachments{}, err
	}
	classes := make(map[string]string)
	geoms := make(map[string]Geom)
	outputs := make(map[string][]Provider)
	incoming := make(map[string][]Geom)
	names := make(map[string]int)
	var disk Provider
	for _, c := range graph.Classes {
		classes[c.ID] = c.Name
	}
	for _, g := range graph.Geoms {
		geoms[g.ID] = g
		for _, c := range g.Consumers {
			incoming[c.ProviderID] = append(incoming[c.ProviderID], g)
		}
	}
	for _, p := range graph.Providers {
		names[p.Name]++
		outputs[p.GeomID] = append(outputs[p.GeomID], p)
		if p.Name == name {
			disk = p
		}
	}
	g := geoms[disk.GeomID]
	if names[name] != 1 || disk.Kind != DiskCandidate || disk.Class != "DISK" || disk.Identity != IdentityUnique || g.Name != name || len(g.Consumers) != 0 || len(outputs[g.ID]) != 1 {
		return DiskAttachments{}, ErrDiskAttachments
	}
	serial := ""
	for _, id := range disk.Identifiers {
		if id.Kind == "serial" {
			if serial != "" && serial != id.Value {
				return DiskAttachments{}, ErrDiskAttachments
			}
			serial = id.Value
		}
	}
	if serial == "" {
		return DiskAttachments{}, ErrDiskAttachments
	}
	queue := []Provider{disk}
	devSeen := make(map[string]bool)
	var aliases []Provider
	for i := 0; i < len(queue); i++ {
		parent := queue[i]
		for _, child := range incoming[parent.ID] {
			if len(child.Consumers) != 1 || child.Consumers[0].ProviderID != parent.ID {
				return DiskAttachments{}, ErrDiskAttachments
			}
			out := outputs[child.ID]
			switch classes[child.ClassID] {
			case "DEV":
				if child.Name != parent.Name || len(out) != 0 || devSeen[parent.ID] {
					return DiskAttachments{}, ErrDiskAttachments
				}
				devSeen[parent.ID] = true
			case "LABEL":
				// One exact alias on the selected disk; no chained labels,
				// truncation/escaping guesses or overlap with another provider.
				if parent.ID != disk.ID || len(out) != 1 || len(aliases) != 0 {
					return DiskAttachments{}, ErrDiskAttachments
				}
				p := out[0]
				if p.Name != "diskid/DISK-"+serial || names[p.Name] != 1 || p.Class != "LABEL" || p.Kind != Logical || p.MediaSize != disk.MediaSize || p.SectorSize != disk.SectorSize {
					return DiskAttachments{}, ErrDiskAttachments
				}
				for _, id := range p.Identifiers {
					if id.Kind == "serial" && id.Value != serial {
						return DiskAttachments{}, ErrDiskAttachments
					}
				}
				aliases = append(aliases, p)
				queue = append(queue, p)
			default:
				return DiskAttachments{}, ErrDiskAttachments
			}
		}
		if !devSeen[parent.ID] {
			return DiskAttachments{}, ErrDiskAttachments
		}
	}
	if names["diskid/DISK-"+serial] != 0 && len(aliases) != 1 {
		return DiskAttachments{}, ErrDiskAttachments
	}
	sort.Slice(aliases, func(i, j int) bool { return aliases[i].Name < aliases[j].Name })
	return DiskAttachments{Disk: disk, Aliases: aliases}, nil
}
