// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"encoding/hex"
	"errors"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type dashboardCPUCounters struct{ User, System time.Duration }
type dashboardIdleObservation struct {
	User    time.Duration `json:"user_ns"`
	System  time.Duration `json:"system_ns"`
	Wall    time.Duration `json:"wall_ns"`
	Percent float64       `json:"percent_one_core"`
}

// /bin/ps usertime and systime are rounded hundredths on both pinned releases.
// Conversion to nanosecond units does not create kernel nanosecond precision.
func dashboardParseCPU(data []byte, pids [2]int) (dashboardCPUCounters, error) {
	var out dashboardCPUCounters
	invalid := func() (dashboardCPUCounters, error) {
		return dashboardCPUCounters{}, errors.New("invalid paired CPU observation")
	}
	if len(data) > 65536 || pids[0] <= 0 || pids[1] <= 0 || pids[0] == pids[1] {
		return invalid()
	}
	seen := map[int]bool{}
	parse := func(value string) (time.Duration, error) {
		p := strings.Split(value, ":")
		if len(p) != 2 || !digits(p[0]) || len(p[1]) != 5 || p[1][2] != '.' || !digits(p[1][:2]) || !digits(p[1][3:]) {
			return 0, errors.New("invalid rounded CPU time")
		}
		return cpuTime(value)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 || !digits(f[0]) {
			return invalid()
		}
		pid, err := strconv.Atoi(f[0])
		if err != nil || seen[pid] || pid != pids[0] && pid != pids[1] {
			return invalid()
		}
		seen[pid] = true
		user, e1 := parse(f[1])
		system, e2 := parse(f[2])
		if e1 != nil || e2 != nil || user > time.Duration(math.MaxInt64)-out.User || system > time.Duration(math.MaxInt64)-out.System {
			return invalid()
		}
		out.User += user
		out.System += system
	}
	if len(seen) != 2 {
		return invalid()
	}
	return out, nil
}
func dashboardObserveIdle(before, after dashboardCPUCounters, wall time.Duration) (dashboardIdleObservation, error) {
	if before.User < 0 || before.System < 0 || after.User < before.User || after.System < before.System || wall < 300*time.Second {
		return dashboardIdleObservation{}, errors.New("incomplete or regressed idle observation")
	}
	user, system := after.User-before.User, after.System-before.System
	if user > time.Duration(math.MaxInt64)-system {
		return dashboardIdleObservation{}, errors.New("idle CPU overflow")
	}
	percent, err := idlePercent(0, user+system, wall)
	return dashboardIdleObservation{user, system, wall, percent}, err
}

type dashboardResourceMeasurements struct {
	Warmup         time.Duration            `json:"warmup_wall_ns"`
	SampledRSS     uint64                   `json:"sampled_combined_peak_rss_bytes"`
	ManagementHigh uint64                   `json:"management_high_water_bytes"`
	HelperHigh     uint64                   `json:"helper_high_water_bytes"`
	Samples        int                      `json:"rss_samples"`
	MaxGap         time.Duration            `json:"max_sample_gap_ns"`
	Idle           dashboardIdleObservation `json:"idle"`
}
type dashboardResourceProcedure struct {
	Clients                        int   `json:"clients"`
	SessionCount                   int   `json:"session_count"`
	Requests                       int   `json:"requests"`
	ReplacementRequests            int   `json:"replacement_requests"`
	Replacements                   int   `json:"replacements"`
	Retries                        int   `json:"retries"`
	HiddenAPIRequests              int   `json:"hidden_api_requests"`
	WarmupSeconds                  int   `json:"warmup_seconds"`
	IdleSeconds                    int   `json:"idle_seconds"`
	ScheduledCollectSeconds        int   `json:"scheduled_collect_seconds"`
	DefaultLimit                   int   `json:"default_limit"`
	MaximumLimit                   int   `json:"maximum_limit"`
	ComparisonRows                 int   `json:"comparison_rows"`
	ResponseCeilingBytes           int   `json:"response_ceiling_bytes"`
	PredicateControllerOutsidePair bool  `json:"predicate_controller_outside_pair"`
	WarmupWallNS                   int64 `json:"warmup_wall_ns"`
}
type dashboardResourceValues struct {
	SampledRSS      uint64  `json:"sampled_combined_peak_rss_bytes"`
	HighWater       uint64  `json:"conservative_combined_high_water_rss_bytes"`
	BudgetPeak      uint64  `json:"budget_peak_rss_bytes"`
	ManagementHigh  uint64  `json:"management_high_water_bytes"`
	HelperHigh      uint64  `json:"helper_high_water_bytes"`
	Samples         int     `json:"rss_samples"`
	MaxGapNS        int64   `json:"max_sample_gap_ns"`
	NominalSampleNS int64   `json:"nominal_sample_ns"`
	IdleUserNS      int64   `json:"idle_user_ns"`
	IdleSystemNS    int64   `json:"idle_system_ns"`
	IdleWallNS      int64   `json:"idle_wall_ns"`
	IdlePercent     float64 `json:"idle_cpu_percent_one_core"`
}
type dashboardResourcePhases struct {
	Initial     *dashboardPhaseReceipt `json:"initial"`
	Replacement *dashboardPhaseReceipt `json:"replacement"`
}
type dashboardResourceReceipt struct {
	Mode string `json:"mode"`
	dashboardFixtureReceipt
	Procedure dashboardResourceProcedure `json:"procedure"`
	Resources dashboardResourceValues    `json:"resources"`
	Phases    dashboardResourcePhases    `json:"phases"`
}

// Measured values must be supplied by the native controller; portable fixtures
// exercise only the contract. No source identity, control completion or cleanup
// acceptance is synthesized by this per-mode record builder.
func dashboardBuildResource(p *dashboardPreparedFixture, w dashboardMeasuredWorkload, m dashboardResourceMeasurements) (*dashboardResourceReceipt, error) {
	invalid := func() (*dashboardResourceReceipt, error) {
		return nil, errors.New("incomplete or failed dashboard resource observation")
	}
	if p == nil || w.Replacements != 3 || m.Warmup < 60*time.Second || m.SampledRSS == 0 || m.ManagementHigh == 0 || m.HelperHigh == 0 || m.ManagementHigh > math.MaxUint64-m.HelperHigh || m.Samples < 1 || m.MaxGap <= 0 {
		return invalid()
	}
	digest, err := hex.DecodeString(p.fixture.InventorySHA256)
	if err != nil || len(digest) != 32 || hex.EncodeToString(digest) != p.fixture.InventorySHA256 {
		return invalid()
	}
	small := p.mode == "synthetic-100" || p.mode == "native-small"
	if p.large == small || p.fixture.Datasets != p.count || p.count < 1 || p.count > 100000 {
		return invalid()
	}
	if p.mode == "native-small" {
		if p.count < 100 || p.fixture.OwnedDatasets != 100 || p.fixture.Origin != "native-original" {
			return invalid()
		}
	} else {
		if p.mode != "synthetic-100" && p.mode != "synthetic-10000" && p.mode != "projection-10000" || p.fixture.OwnedDatasets != 0 || p.fixture.Origin != "generated" || small && p.count != 100 || !small && p.count != 10000 {
			return invalid()
		}
	}
	initial, e1 := dashboardBuildPhaseReceipt(w.InitialRaw, "initial", !small)
	replacement, e2 := dashboardBuildPhaseReceipt(w.ReplacementRaw, "replacement", !small)
	if e1 != nil || e2 != nil || !reflect.DeepEqual(initial, w.Initial) || !reflect.DeepEqual(replacement, w.Replacement) {
		return invalid()
	}
	high := m.ManagementHigh + m.HelperHigh
	peak := max(m.SampledRSS, high)
	budget := uint64(256 << 20)
	idleSeconds := 0
	if small {
		budget = 128 << 20
		idleSeconds = 300
		expected, e := dashboardObserveIdle(dashboardCPUCounters{}, dashboardCPUCounters{m.Idle.User, m.Idle.System}, m.Idle.Wall)
		if e != nil || m.Idle.Percent != expected.Percent || math.IsNaN(m.Idle.Percent) || math.IsInf(m.Idle.Percent, 0) || m.Idle.Percent >= 1 {
			return invalid()
		}
	} else if m.Idle != (dashboardIdleObservation{}) {
		return invalid()
	}
	if peak > budget {
		return invalid()
	}
	return &dashboardResourceReceipt{
		Mode: p.mode, dashboardFixtureReceipt: p.fixture,
		Procedure: dashboardResourceProcedure{Clients: 10, SessionCount: 1, Requests: len(initial.Samples), ReplacementRequests: len(replacement.Samples), Replacements: w.Replacements, Retries: 0, HiddenAPIRequests: 0, WarmupSeconds: 60, IdleSeconds: idleSeconds, ScheduledCollectSeconds: 30, DefaultLimit: 100, MaximumLimit: 500, ComparisonRows: 20, ResponseCeilingBytes: dashboardBodyLimit, PredicateControllerOutsidePair: true, WarmupWallNS: int64(m.Warmup)},
		Resources: dashboardResourceValues{m.SampledRSS, high, peak, m.ManagementHigh, m.HelperHigh, m.Samples, int64(m.MaxGap), 100000000, int64(m.Idle.User), int64(m.Idle.System), int64(m.Idle.Wall), m.Idle.Percent},
		Phases:    dashboardResourcePhases{initial, replacement},
	}, nil
}
func dashboardLegacyResource(r *dashboardResourceReceipt, native bool) map[string]any {
	p, v := r.Procedure, r.Resources
	q := r.Phases.Initial.Summary["all"].SuccessLatency
	out := map[string]any{"datasets": r.Datasets, "clients": p.Clients, "requests": p.Requests, "replacement_requests": p.ReplacementRequests, "replacements": p.Replacements, "warmup_seconds": p.WarmupSeconds, "idle_seconds": p.IdleSeconds, "idle_cpu_percent_one_core": v.IdlePercent, "sampled_combined_peak_rss_bytes": v.SampledRSS, "conservative_combined_high_water_rss_bytes": v.HighWater, "budget_peak_rss_bytes": v.BudgetPeak, "rss_samples": v.Samples, "max_sample_gap_ms": float64(v.MaxGapNS/1000) / 1000, "p50_ms": float64(q.P50NS/1000) / 1000, "p95_ms": float64(q.P95NS/1000) / 1000, "p99_ms": float64(q.P99NS/1000) / 1000}
	if native {
		out["mode"] = r.Mode
		out["owned_datasets"] = r.OwnedDatasets
	} else {
		out["session_count"] = p.SessionCount
	}
	return out
}
