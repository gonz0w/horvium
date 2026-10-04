// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"errors"
	"testing"
	"time"
)

func TestBoundedCloneOwnsNestedData(t *testing.T) {
	at := time.Now().UTC()
	memory := uint64(100)
	s := Snapshot{StartedAt: at, FinishedAt: at, Host: Section[Host]{Status: Partial, StartedAt: at, FinishedAt: at, Data: Host{PhysicalMemory: &memory}, Problems: []Problem{{Source: "architecture", Code: "failed"}}}, GEOM: Section[GEOM]{Status: OK, StartedAt: at, FinishedAt: at, Data: GEOM{Classes: []Class{{ID: "c", Name: "MD"}}, Geoms: []Geom{{ID: "g", ClassID: "c", Name: "md", Consumers: []Consumer{{ID: "x", ProviderID: "p"}}}}, Providers: []Provider{{ID: "p", GeomID: "g", Name: "md", Class: "MD", Kind: Memory, SectorSize: 512, Identity: IdentityNotApplicable, Identifiers: []Identifier{{Kind: "serial", Value: "fixture", Source: "geom.config.ident"}}}}}}, Pools: Section[[]Pool]{Status: OK, StartedAt: at, FinishedAt: at}, Datasets: Section[[]Dataset]{Status: OK, StartedAt: at, FinishedAt: at, Data: []Dataset{{Name: "fixture/data", GUID: "1"}}}}
	got, err := CloneBounded(s, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	memory = 200
	s.Host.Problems[0].Code = "changed"
	s.GEOM.Data.Geoms[0].Consumers[0].ID = "changed"
	s.GEOM.Data.Providers[0].Identifiers[0].Value = "changed"
	s.Datasets.Data[0].Name = "changed"
	if *got.Host.Data.PhysicalMemory != 100 || got.Host.Problems[0].Code != "failed" || got.GEOM.Data.Geoms[0].Consumers[0].ID != "x" || got.GEOM.Data.Providers[0].Identifiers[0].Value != "fixture" || got.Datasets.Data[0].Name != "fixture/data" {
		t.Fatal("source retained ownership")
	}
	for _, change := range []func(*Limits){func(l *Limits) { l.MaxBytes = 1 }, func(l *Limits) { l.MaxRecords = 1 }, func(l *Limits) { l.MaxFieldBytes = 1 }} {
		l := DefaultLimits()
		change(&l)
		if _, err := CloneBounded(got, l); !errors.Is(err, ErrLimit) {
			t.Fatalf("unbounded clone: %v", err)
		}
	}
	got.Datasets.Data[0].GUID = "0"
	if _, err := CloneBounded(got, DefaultLimits()); !errors.Is(err, ErrMalformed) {
		t.Fatal("malformed generation cloned")
	}
}
