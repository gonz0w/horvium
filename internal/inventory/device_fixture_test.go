// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"errors"
	"testing"
)

const deviceFixtureBytes = 64 << 20

// This is a test-only recognizer for declared blank virtio disks, not a product
// eligibility check. DEV/LABEL consumers exist without partition/data use.
func fixtureDiskCorrespondence(graph GEOM, p Provider) error {
	deny := errors.New("unqualified fixture backing correspondence")
	classes := make(map[string]string)
	for _, c := range graph.Classes {
		classes[c.ID] = c.Name
	}
	foundDisk, foundDev := false, false
	serial := ""
	for _, id := range p.Identifiers {
		if id.Kind == "serial" {
			serial = id.Value
		}
	}
	for _, g := range graph.Geoms {
		if g.ID == p.GeomID {
			if classes[g.ClassID] != "DISK" || p.Class != "DISK" || p.Kind != DiskCandidate || len(g.Consumers) != 0 {
				return deny
			}
			foundDisk = true
		}
		incoming := false
		for _, c := range g.Consumers {
			if c.ProviderID == p.ID {
				incoming = true
			}
		}
		if !incoming {
			continue
		}
		if len(g.Consumers) != 1 {
			return deny
		}
		var outputs []Provider
		for _, q := range graph.Providers {
			if q.GeomID == g.ID {
				outputs = append(outputs, q)
			}
		}
		switch classes[g.ClassID] {
		case "DEV":
			if g.Name != p.Name || len(outputs) != 0 {
				return deny
			}
			foundDev = true
		case "LABEL":
			if serial == "" || len(outputs) != 1 {
				return deny
			}
			q := outputs[0]
			if q.Name != "diskid/DISK-"+serial || q.Class != "LABEL" || q.Kind != Logical || q.MediaSize != p.MediaSize || q.SectorSize != p.SectorSize {
				return deny
			}
		default:
			return deny
		}
	}
	if !foundDisk || !foundDev {
		return deny
	}
	return nil
}

func TestDeviceFixtureAllowsDevAndVerifiedDiskIdentAliases(t *testing.T) {
	p := Provider{ID: "p", GeomID: "disk", Name: "vtbd1", Class: "DISK", Kind: DiskCandidate, MediaSize: deviceFixtureBytes, SectorSize: 512, Identifiers: []Identifier{{Kind: "serial", Value: "HVM-UNIQUE-512", Source: "geom.config.ident"}}}
	g := GEOM{Classes: []Class{{ID: "c1", Name: "DISK"}, {ID: "c2", Name: "DEV"}, {ID: "c3", Name: "LABEL"}}, Geoms: []Geom{{ID: "disk", ClassID: "c1", Name: p.Name}, {ID: "dev", ClassID: "c2", Name: p.Name, Consumers: []Consumer{{ProviderID: p.ID}}}}, Providers: []Provider{p}}
	if err := fixtureDiskCorrespondence(g, p); err != nil {
		t.Fatal("normal DEV attachment rejected", err)
	}
	alias := Provider{ID: "alias", GeomID: "label", Name: "diskid/DISK-HVM-UNIQUE-512", Class: "LABEL", Kind: Logical, MediaSize: p.MediaSize, SectorSize: p.SectorSize}
	g.Geoms = append(g.Geoms, Geom{ID: "label", ClassID: "c3", Consumers: []Consumer{{ProviderID: p.ID}}})
	g.Providers = append(g.Providers, alias)
	if err := fixtureDiskCorrespondence(g, p); err != nil {
		t.Fatal("verified disk-ident alias rejected", err)
	}
	for _, change := range []func(*GEOM){
		func(x *GEOM) { x.Geoms[0].Consumers = []Consumer{{ProviderID: "other"}} },
		func(x *GEOM) { x.Classes[1].Name = "PART" },
		func(x *GEOM) { x.Providers[1].Name = "diskid/DISK-other" },
		func(x *GEOM) { x.Providers[1].SectorSize = 4096 },
		func(x *GEOM) { x.Geoms[2].Consumers = append(x.Geoms[2].Consumers, Consumer{ProviderID: "other"}) },
		func(x *GEOM) { x.Geoms = x.Geoms[:1] },
	} {
		x := GEOM{Classes: append([]Class(nil), g.Classes...), Geoms: append([]Geom(nil), g.Geoms...), Providers: append([]Provider(nil), g.Providers...)}
		change(&x)
		if err := fixtureDiskCorrespondence(x, p); err == nil {
			t.Fatal("unqualified fixture relationship accepted")
		}
	}
}
