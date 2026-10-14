// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"math"
	"time"
)

type witnessAugmentedMeasurement struct {
	Mode                                                                                     string
	PairedSampled, ManagementHigh, HelperHigh, ObserverHigh, ConnectorHigh, AugmentedSampled uint64
	Connectors, Reaped, Active, UnknownDescendants, Samples, ScheduledSamples                int
	MaxGap                                                                                   time.Duration
	ProductCPU, ObserverCPU, ConnectorCPU, IdleWall                                          time.Duration
}
type witnessAugmentedResources struct {
	Mode             string        `json:"mode"`
	PairedPeak       uint64        `json:"paired_peak_bytes"`
	AugmentedPeak    uint64        `json:"augmented_peak_bytes"`
	ConservativeHigh uint64        `json:"conservative_high_bytes"`
	ConnectorPeak    uint64        `json:"connector_peak_bytes"`
	Connectors       int           `json:"connectors"`
	Samples          int           `json:"samples"`
	MaxGapNS         int64         `json:"max_gap_ns"`
	AugmentedIdleCPU time.Duration `json:"augmented_idle_cpu_ns"`
	IdlePercent      float64       `json:"idle_percent_one_core"`
}

func dashboardWitnessAugmentedResources(m witnessAugmentedMeasurement) (witnessAugmentedResources, error) {
	bad := func() (witnessAugmentedResources, error) { return witnessAugmentedResources{}, errDashboardWitness }
	if m.Mode != "native-small" && m.Mode != "projection-10000" || m.PairedSampled == 0 || m.ManagementHigh == 0 || m.HelperHigh == 0 || m.ObserverHigh == 0 || m.ConnectorHigh == 0 || m.Connectors <= 0 || m.Reaped != m.Connectors || m.Active != 0 || m.UnknownDescendants != 0 || m.Samples <= 0 || m.Samples != m.ScheduledSamples || m.MaxGap <= 0 || m.AugmentedSampled < m.PairedSampled {
		return bad()
	}
	add := func(values ...uint64) (uint64, bool) {
		var total uint64
		for _, value := range values {
			if value > math.MaxUint64-total {
				return 0, false
			}
			total += value
		}
		return total, true
	}
	pair, ok := add(m.ManagementHigh, m.HelperHigh)
	if !ok {
		return bad()
	}
	pair = max(pair, m.PairedSampled)
	high, ok := add(pair, m.ObserverHigh, m.ConnectorHigh)
	if !ok {
		return bad()
	}
	peak := max(high, m.AugmentedSampled)
	budget := uint64(256 << 20)
	var cpu time.Duration
	var percent float64
	if m.Mode == "native-small" {
		budget = 128 << 20
		if m.IdleWall < 300*time.Second || m.ProductCPU < 0 || m.ObserverCPU < 0 || m.ConnectorCPU < 0 || m.ProductCPU > time.Duration(math.MaxInt64)-m.ObserverCPU || m.ProductCPU+m.ObserverCPU > time.Duration(math.MaxInt64)-m.ConnectorCPU {
			return bad()
		}
		cpu = m.ProductCPU + m.ObserverCPU + m.ConnectorCPU
		percent = 100 * float64(cpu) / float64(m.IdleWall)
		if percent >= 1 || math.IsNaN(percent) || math.IsInf(percent, 0) {
			return bad()
		}
	} else if m.IdleWall != 0 || m.ProductCPU != 0 || m.ObserverCPU != 0 || m.ConnectorCPU != 0 {
		return bad()
	}
	if peak > budget || pair > budget {
		return bad()
	}
	return witnessAugmentedResources{m.Mode, pair, peak, high, m.ConnectorHigh, m.Connectors, m.Samples, int64(m.MaxGap), cpu, percent}, nil
}
