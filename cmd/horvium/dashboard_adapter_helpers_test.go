// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/gonz0w/horvium/internal/inventory"
)

// Private controller preparation. No reader, HTTP/API lookup or native operation
// is performed here. The caller supplies independently observed original inputs.
type dashboardFixtureReceipt struct {
	Datasets        int    `json:"datasets"`
	OwnedDatasets   int    `json:"owned_datasets"`
	InventorySHA256 string `json:"inventory_sha256"`
	Origin          string `json:"origin"`
}
type dashboardPreparedFixture struct {
	mode      string
	count     int
	large     bool
	fixture   dashboardFixtureReceipt
	plan      []dashboardRequest
	predicate dashboardPredicate
}

func dashboardCanonicalOriginal(s *inventory.Snapshot) {
	slices.SortFunc(s.Pools.Data, func(a, b inventory.Pool) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return dashboardGUIDCompare(a.GUID, b.GUID)
	})
	slices.SortFunc(s.Datasets.Data, func(a, b inventory.Dataset) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return dashboardGUIDCompare(a.GUID, b.GUID)
	})
	slices.SortFunc(s.GEOM.Data.Classes, func(a, b inventory.Class) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(s.GEOM.Data.Geoms, func(a, b inventory.Geom) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(s.GEOM.Data.Providers, func(a, b inventory.Provider) int { return cmp.Compare(a.ID, b.ID) })
	for i := range s.GEOM.Data.Geoms {
		slices.SortFunc(s.GEOM.Data.Geoms[i].Consumers, func(a, b inventory.Consumer) int { return cmp.Compare(a.ID, b.ID) })
	}
	for i := range s.GEOM.Data.Providers {
		slices.SortFunc(s.GEOM.Data.Providers[i].Identifiers, func(a, b inventory.Identifier) int {
			aa, bb := [3]string{a.Kind, a.Value, a.Source}, [3]string{b.Kind, b.Value, b.Source}
			for i := range aa {
				if aa[i] < bb[i] {
					return -1
				}
				if aa[i] > bb[i] {
					return 1
				}
			}
			return 0
		})
	}
}
func dashboardOriginalBytes(s inventory.Snapshot) ([]byte, error) {
	// Observation clocks/section problems are absent: complete sections are required
	// and their values are the immutable oracle. No raw inventory enters receipts.
	return json.Marshal(struct {
		Host     inventory.Host      `json:"host"`
		GEOM     inventory.GEOM      `json:"geom"`
		Pools    []inventory.Pool    `json:"pools"`
		Datasets []inventory.Dataset `json:"datasets"`
	}{s.Host.Data, s.GEOM.Data, s.Pools.Data, s.Datasets.Data})
}
func newDashboardPreparedFixture(mode string, original inventory.Snapshot, owned []inventory.Dataset) (*dashboardPreparedFixture, error) {
	invalid := func() (*dashboardPreparedFixture, error) {
		return nil, errors.New("invalid original dashboard fixture or mode")
	}
	s, err := inventory.CloneBounded(original, inventory.DefaultLimits())
	if err != nil || !s.Complete() || len(s.Datasets.Data) == 0 {
		return invalid()
	}
	dashboardCanonicalOriginal(&s)
	native := mode == "native-small"
	count := len(s.Datasets.Data)
	large := mode == "synthetic-10000" || mode == "projection-10000"
	if native {
		if count < 100 || len(owned) != 100 {
			return invalid()
		}
	} else if mode == "synthetic-100" || mode == "synthetic-10000" || mode == "projection-10000" {
		expectedCount := 100
		if large {
			expectedCount = 10000
		}
		if count != expectedCount || len(owned) != 0 {
			return invalid()
		}
	} else {
		return invalid()
	}
	if native {
		names, guids := map[string]bool{}, map[string]bool{}
		datasets := make(map[string]string, count)
		for _, row := range s.Datasets.Data {
			datasets[row.Name] = row.GUID
		}
		for _, witness := range owned {
			if names[witness.Name] || guids[witness.GUID] || datasets[witness.Name] != witness.GUID {
				return invalid()
			}
			names[witness.Name] = true
			guids[witness.GUID] = true
		}
	} else if mode == "projection-10000" {
		for i, row := range s.Datasets.Data {
			expected := inventory.Dataset{Name: fmt.Sprintf("fixture/dataset-%06d", i), GUID: strconv.Itoa(i + 100), Used: uint64(i), Available: 1 << 40, Referenced: uint64(i)}
			if row != expected {
				return invalid()
			}
		}
	} else {
		declared := performanceSnapshot(count, 0)
		dashboardCanonicalOriginal(&declared)
		a, e1 := dashboardOriginalBytes(s)
		b, e2 := dashboardOriginalBytes(declared)
		if e1 != nil || e2 != nil || string(a) != string(b) {
			return invalid()
		}
	}
	plan, err := dashboardPlan(count)
	if err != nil {
		return invalid()
	}
	predicate, err := newDashboardHTMLPredicate(s, plan)
	if err != nil {
		return invalid()
	}
	encoded, err := dashboardOriginalBytes(s)
	if err != nil {
		return invalid()
	}
	hash := sha256.Sum256(encoded)
	origin := "generated"
	if native {
		origin = "native-original"
	}
	return &dashboardPreparedFixture{mode: mode, count: count, large: large,
		fixture: dashboardFixtureReceipt{count, len(owned), hex.EncodeToString(hash[:]), origin}, plan: plan, predicate: predicate}, nil
}

type dashboardWireQuantiles struct {
	P50NS int64 `json:"p50_ns"`
	P95NS int64 `json:"p95_ns"`
	P99NS int64 `json:"p99_ns"`
}
type dashboardWireSummary struct {
	Scheduled      int                     `json:"scheduled"`
	Success        int                     `json:"success"`
	Expiry         int                     `json:"expiry"`
	SuccessLatency *dashboardWireQuantiles `json:"success_latency"`
	ExpiryLatency  *dashboardWireQuantiles `json:"expiry_latency"`
}
type dashboardWireSample struct {
	Index      int    `json:"index"`
	Case       string `json:"case"`
	Outcome    string `json:"outcome"`
	Status     int    `json:"status"`
	DurationNS int64  `json:"duration_ns"`
	BodyBytes  int    `json:"body_bytes"`
	Predicate  string `json:"predicate"`
}
type dashboardPhaseReceipt struct {
	Samples []dashboardWireSample           `json:"samples"`
	Summary map[string]dashboardWireSummary `json:"summary"`
}

func dashboardWireLatency(count int, q dashboardQuantiles) *dashboardWireQuantiles {
	if count == 0 {
		return nil
	}
	return &dashboardWireQuantiles{int64(q.P50), int64(q.P95), int64(q.P99)}
}
func dashboardBuildPhaseReceipt(samples []dashboardSample, phase string, large bool) (*dashboardPhaseReceipt, error) {
	summary, err := dashboardSummarize(samples, phase)
	if err != nil {
		return nil, err
	}
	if phase == "initial" {
		if err := dashboardLatencyBudget(summary, large); err != nil {
			return nil, err
		}
	}
	out := &dashboardPhaseReceipt{Samples: make([]dashboardWireSample, len(samples)), Summary: make(map[string]dashboardWireSummary)}
	for i, sample := range samples {
		predicate := "original-inventory"
		if sample.Outcome == "expiry" {
			predicate = "safe-expiry"
		}
		if sample.BodyBytes < 1 || sample.BodyBytes > dashboardBodyLimit || sample.Predicate != predicate {
			return nil, errors.New("complete verified dashboard body receipt required")
		}
		out.Samples[i] = dashboardWireSample{sample.Index, sample.Case, sample.Outcome, sample.Status, int64(sample.Duration), sample.BodyBytes, sample.Predicate}
	}
	wire := func(s dashboardCaseSummary) dashboardWireSummary {
		return dashboardWireSummary{s.Scheduled, s.Success, s.Expiry, dashboardWireLatency(s.Success, s.Latency), dashboardWireLatency(s.Expiry, s.ExpiryLatency)}
	}
	out.Summary["all"] = wire(dashboardCaseSummary{summary.Scheduled, summary.Success, summary.Expiry, summary.Latency, summary.ExpiryLatency})
	for _, name := range dashboardCases {
		out.Summary[name] = wire(summary.Cases[name])
	}
	return out, nil
}
func (p *dashboardPreparedFixture) load(client *http.Client, base, phase string) (*dashboardPhaseReceipt, []dashboardSample, error) {
	if p == nil {
		return nil, nil, errors.New("missing prepared dashboard fixture")
	}
	samples, err := dashboardLoad(client, base, p.count, p.plan, phase, p.predicate)
	if err != nil {
		return nil, samples, err
	}
	receipt, err := dashboardBuildPhaseReceipt(samples, phase, p.large)
	return receipt, samples, err
}
