// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"errors"
	"path"
	"strings"
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

func TestDeviceFixtureRootMountSupportsUFSAndZFS(t *testing.T) {
	for _, text := range []string{"/dev/gpt/rootfs / ufs rw 1 1\n", "zroot/ROOT/default / zfs rw 0 0\n"} {
		root, err := fixtureMountedRoot([]byte(text))
		if err != nil || root.Source == "" {
			t.Fatal("valid mounted fixture root rejected", err)
		}
	}
	for _, text := range []string{"", "host:/root / nfs rw 0 0\n", "/dev/../foreign / ufs rw 1 1\n", "relative / ufs rw 1 1\n", "-pool / zfs rw 0 0\n", "zroot / zfs rw 0 0\nzroot / zfs rw 0 0\n"} {
		if _, err := fixtureMountedRoot([]byte(text)); err == nil {
			t.Fatal("missing/unsupported/ambiguous root accepted")
		}
	}
}

func TestDeviceFixtureBootBackingFollowsAliasesAndRejectsBrokenGraphs(t *testing.T) {
	g := GEOM{Providers: []Provider{
		{ID: "disk", GeomID: "d", Name: "vtbd0", Kind: DiskCandidate},
		{ID: "part", GeomID: "p", Name: "vtbd0p2", Kind: Logical},
		{ID: "label", GeomID: "l", Name: "gpt/rootfs", Kind: Logical},
	}, Geoms: []Geom{
		{ID: "d"}, {ID: "p", Consumers: []Consumer{{ProviderID: "disk"}}},
		{ID: "l", Consumers: []Consumer{{ProviderID: "part"}}},
	}}
	roots, err := fixtureBootDisks(g, []string{"gpt/rootfs"})
	if err != nil || len(roots) != 1 || !roots["vtbd0"] {
		t.Fatal("boot backing alias not resolved", err)
	}
	for _, change := range []func(*GEOM){
		func(x *GEOM) { x.Geoms[1].Consumers = []Consumer{{ProviderID: "label"}} },
		func(x *GEOM) { x.Geoms[1].Consumers = nil },
		func(x *GEOM) { x.Geoms[1].Consumers = []Consumer{{ProviderID: "missing"}} },
		func(x *GEOM) { x.Providers = append(x.Providers, Provider{ID: "other", Name: "gpt/rootfs"}) },
	} {
		x := GEOM{Providers: append([]Provider(nil), g.Providers...), Geoms: append([]Geom(nil), g.Geoms...)}
		change(&x)
		if _, err := fixtureBootDisks(x, []string{"gpt/rootfs"}); err == nil {
			t.Fatal("broken/cyclic/ambiguous boot backing accepted")
		}
	}
	for _, leaves := range [][]string{nil, {"unknown"}} {
		if _, err := fixtureBootDisks(g, leaves); err == nil {
			t.Fatal("unavailable boot control accepted")
		}
	}
}

type fixtureRootMount struct{ Source, Type string }

func fixtureMountedRoot(data []byte) (fixtureRootMount, error) {
	deny := errors.New("guest root control unavailable")
	var root fixtureRootMount
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != "/" {
			continue
		}
		if len(fields) != 6 || root.Source != "" {
			return fixtureRootMount{}, deny
		}
		root = fixtureRootMount{Source: fields[0], Type: fields[2]}
		if path.Clean(root.Source) != root.Source || strings.HasPrefix(root.Source, "-") {
			return fixtureRootMount{}, deny
		}
		switch root.Type {
		case "ufs":
			if !strings.HasPrefix(root.Source, "/dev/") || root.Source == "/dev/" {
				return fixtureRootMount{}, deny
			}
		case "zfs":
			if strings.HasPrefix(root.Source, "/") || root.Source == "." {
				return fixtureRootMount{}, deny
			}
		default:
			return fixtureRootMount{}, deny
		}
	}
	if root.Source == "" {
		return fixtureRootMount{}, deny
	}
	return root, nil
}

func fixtureBootDisks(graph GEOM, leaves []string) (map[string]bool, error) {
	deny := errors.New("root backing correspondence unavailable")
	providers := make(map[string]Provider)
	geoms := make(map[string]Geom)
	names := make(map[string][]string)
	for _, p := range graph.Providers {
		providers[p.ID] = p
		names[p.Name] = append(names[p.Name], p.ID)
	}
	for _, g := range graph.Geoms {
		geoms[g.ID] = g
	}
	roots := make(map[string]bool)
	var follow func(string, map[string]bool) error
	follow = func(id string, visiting map[string]bool) error {
		p, ok := providers[id]
		if !ok || visiting[id] || len(visiting) > 32 {
			return deny
		}
		if p.Kind == DiskCandidate {
			roots[p.Name] = true
			return nil
		}
		visiting[id] = true
		defer delete(visiting, id)
		consumers := geoms[p.GeomID].Consumers
		if len(consumers) == 0 {
			return deny
		}
		for _, c := range consumers {
			if err := follow(c.ProviderID, visiting); err != nil {
				return err
			}
		}
		return nil
	}
	for _, leaf := range leaves {
		if len(names[leaf]) != 1 {
			return nil, deny
		}
		if err := follow(names[leaf][0], make(map[string]bool)); err != nil {
			return nil, err
		}
	}
	if len(roots) == 0 {
		return nil, deny
	}
	return roots, nil
}
