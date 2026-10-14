// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"math"
	"testing"
	"time"
)

func witnessAccountingFixture() witnessAugmentedMeasurement {
	return witnessAugmentedMeasurement{Mode: "native-small", PairedSampled: 60 << 20, ManagementHigh: 32 << 20, HelperHigh: 32 << 20, ObserverHigh: 16 << 20, ConnectorHigh: 12 << 20, AugmentedSampled: 95 << 20, Connectors: 3, Reaped: 3, Samples: 3000, ScheduledSamples: 3000, MaxGap: 123 * time.Millisecond, ProductCPU: 100 * time.Millisecond, ObserverCPU: 200 * time.Millisecond, ConnectorCPU: 60 * time.Millisecond, IdleWall: 300 * time.Second}
}
func TestDashboardWitnessAugmentedResourcesCountTransientOwners(t *testing.T) {
	m := witnessAccountingFixture()
	r, e := dashboardWitnessAugmentedResources(m)
	if e != nil || r.PairedPeak != 64<<20 || r.ConservativeHigh != 92<<20 || r.AugmentedPeak != 95<<20 || r.AugmentedIdleCPU != 360*time.Millisecond || r.IdlePercent != 0.12 {
		t.Fatal("paired/observer/largest connector and all connector CPU", r, e)
	}
	m.AugmentedSampled = 80 << 20
	r, e = dashboardWitnessAugmentedResources(m)
	if e != nil || r.AugmentedPeak != 92<<20 {
		t.Fatal("unsampled transient high-water omitted", r, e)
	}
}
func TestDashboardWitnessAugmentedResourcesRejectIncompleteOrAlteredBudgets(t *testing.T) {
	cases := map[string]func(*witnessAugmentedMeasurement){
		"active": func(m *witnessAugmentedMeasurement) { m.Active = 1 }, "unreaped": func(m *witnessAugmentedMeasurement) { m.Reaped-- }, "unknown": func(m *witnessAugmentedMeasurement) { m.UnknownDescendants = 1 }, "missing-observer": func(m *witnessAugmentedMeasurement) { m.ObserverHigh = 0 }, "missing-connector": func(m *witnessAugmentedMeasurement) { m.ConnectorHigh = 0 }, "missing-sample": func(m *witnessAugmentedMeasurement) { m.Samples-- }, "no-gap": func(m *witnessAugmentedMeasurement) { m.MaxGap = 0 }, "overflow": func(m *witnessAugmentedMeasurement) { m.ObserverHigh = math.MaxUint64 }, "negative-cpu": func(m *witnessAugmentedMeasurement) { m.ObserverCPU = -1 }, "cpu-overflow": func(m *witnessAugmentedMeasurement) { m.ObserverCPU = math.MaxInt64 }, "short-idle": func(m *witnessAugmentedMeasurement) { m.IdleWall-- }, "one-percent": func(m *witnessAugmentedMeasurement) { m.ProductCPU = 3 * time.Second }, "rss-budget": func(m *witnessAugmentedMeasurement) { m.ObserverHigh = 128 << 20 }, "understated-pair": func(m *witnessAugmentedMeasurement) { m.AugmentedSampled = 1 }, "wrong-mode": func(m *witnessAugmentedMeasurement) { m.Mode = "synthetic-100" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			m := witnessAccountingFixture()
			change(&m)
			if _, e := dashboardWitnessAugmentedResources(m); e == nil {
				t.Fatal("uncertain/changed measurement accepted")
			}
		})
	}
	m := witnessAccountingFixture()
	m.Mode = "projection-10000"
	m.ProductCPU = 0
	m.ObserverCPU = 0
	m.ConnectorCPU = 0
	m.IdleWall = 0
	if _, e := dashboardWitnessAugmentedResources(m); e != nil {
		t.Fatal("large mode invents idle", e)
	}
}

func TestDashboardWitnessAugmentedResourcesKeepLargerOriginalPairPeak(t *testing.T) {
	m := witnessAccountingFixture()
	m.PairedSampled = 100 << 20
	m.AugmentedSampled = 110 << 20
	r, e := dashboardWitnessAugmentedResources(m)
	if e != nil || r.PairedPeak != 100<<20 || r.ConservativeHigh != 128<<20 || r.AugmentedPeak != 128<<20 {
		t.Fatal("original sampled product peak omitted from conservative instrumentation", r, e)
	}
	m.PairedSampled = 120 << 20
	m.AugmentedSampled = 121 << 20
	if _, e := dashboardWitnessAugmentedResources(m); e == nil {
		t.Fatal("conservative original pair plus instrumentation budget bypass")
	}
}
