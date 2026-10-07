// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

func memberDisks() []inventory.DiskObservation {
	p := enrolledPolicy()
	result := make([]inventory.DiskObservation, len(p.Members))
	for i, m := range p.Members {
		name := []string{"ada0", "ada1"}[i]
		result[i] = inventory.DiskObservation{Provider: inventory.Provider{ID: name, GeomID: "g" + name, Name: name, Class: "DISK", Kind: inventory.DiskCandidate, Identity: inventory.IdentityUnique, MediaSize: m.MediaSize, SectorSize: m.SectorSize, Identifiers: []inventory.Identifier{{Kind: "serial", Value: m.Serial, Source: m.SerialSource}}}, SectorCount: m.MediaSize / m.SectorSize, StripeSize: m.StripeSize, StripeOffset: m.StripeOffset}
	}
	return result
}
func TestPropertyMembersRequireCompleteFreshEnrolledBaseline(t *testing.T) {
	p, disks := enrolledPolicy(), memberDisks()
	_, o, _ := fixture()
	o.Eligible = false
	if err := ValidatePropertyMembers(p, o, disks); err != nil || o.Eligible {
		t.Fatal("complete baseline or authority", err)
	}
	for _, mode := range []string{"legacy", "partial", "extra", "guid", "serial", "provenance", "geometry", "stripe", "reused-path", "logical", "duplicate-disk", "incomplete"} {
		t.Run(mode, func(t *testing.T) {
			p, d := enrolledPolicy(), memberDisks()
			_, o, _ := fixture()
			o.Eligible = false
			switch mode {
			case "legacy":
				p.Members = nil
			case "partial":
				p.Members = p.Members[:1]
			case "extra":
				p.Members = append(p.Members, PoolMember{VdevGUID: "905", Serial: "OTHER", SerialSource: "geom.config.ident", MediaSize: 67108864, SectorSize: 512})
			case "guid":
				p.Members[0].VdevGUID = "905"
			case "serial":
				p.Members[0].Serial = "OTHER"
			case "provenance":
				p.Members[0].SerialSource = "provider.config.ident"
			case "geometry":
				p.Members[0].MediaSize *= 2
			case "stripe":
				p.Members[0].StripeOffset = 512
			case "reused-path":
				d[0].Provider.Identifiers[0].Value = "REPLACEMENT"
			case "logical":
				d[0].Provider.Kind = inventory.Partition
			case "duplicate-disk":
				d[1] = d[0]
			case "incomplete":
				o.Complete = false
			}
			if err := ValidatePropertyMembers(p, o, d); err == nil {
				t.Fatal("uncertain enrollment accepted")
			}
			if o.Eligible {
				t.Fatal("eligibility changed")
			}
		})
	}
	p.Members = nil
	if err := ValidatePropertyMembers(p, o, disks); !errors.Is(err, ErrDenied) {
		t.Fatal("legacy authority", err)
	}
}
