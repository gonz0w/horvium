// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func resourceFixture(t *testing.T, mode string) *dashboardPreparedFixture {
	t.Helper()
	original := performanceSnapshot(100, 0)
	switch mode {
	case "synthetic-10000":
		original = performanceSnapshot(10000, 0)
	case "native-small":
		original = adapterNativeInventory()
	case "projection-10000":
		original = adapterProjection()
	}
	var prepared *dashboardPreparedFixture
	var err error
	if mode == "native-small" {
		prepared, err = newDashboardPreparedFixture(mode, original, adapterWitness(original))
	} else {
		prepared, err = newDashboardPreparedFixture(mode, original, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}
func resourceWorkload(t *testing.T) dashboardMeasuredWorkload {
	t.Helper()
	a, b := adapterSamples(), adapterSamples()
	initial, err := dashboardBuildPhaseReceipt(a, "initial", false)
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := dashboardBuildPhaseReceipt(b, "replacement", false)
	if err != nil {
		t.Fatal(err)
	}
	return dashboardMeasuredWorkload{Initial: initial, Replacement: replacement, InitialRaw: a, ReplacementRaw: b, Replacements: 3}
}
func resourceMeasurements(small bool) dashboardResourceMeasurements {
	m := dashboardResourceMeasurements{Warmup: 60 * time.Second, SampledRSS: 32 << 20, ManagementHigh: 16 << 20, HelperHigh: 24 << 20, Samples: 3000, MaxGap: 123456789}
	if small {
		m.Idle = dashboardIdleObservation{User: 100 * time.Millisecond, System: 200 * time.Millisecond, Wall: 300 * time.Second, Percent: 0.1}
	}
	return m
}
func TestDashboardCPUCounterProjectionPreservesSeparateRoundedTimes(t *testing.T) {
	result, err := dashboardParseCPU([]byte("41 0:01.25 2:02.50\n42 4:03.00 0:02.75\n"), [2]int{41, 42})
	if err != nil || result.User != 244250*time.Millisecond || result.System != 125250*time.Millisecond {
		t.Fatal("separate CPU projection", result, err)
	}
}
func TestDashboardCPUCounterProjectionRejectsIncompleteMalformedOrOverflow(t *testing.T) {
	for _, text := range []string{"", "41 0:01.00 0:02.00\n", "41 0:01.00 0:02.00\n41 0:01.00 0:02.00\n", "41 0:01.00 0:02.00\n43 0:01.00 0:02.00\n", "41 0:61.00 0:02.00\n42 0:01.00 0:02.00\n", "41 0:01.0 0:02.00\n42 0:01.00 0:02.00\n", "41 0:01.000000001 0:02.00\n42 0:01.00 0:02.00\n", "41 999999999999999999999:01.00 0:02.00\n42 0:01.00 0:02.00\n", strings.Repeat("x", 65537)} {
		if _, err := dashboardParseCPU([]byte(text), [2]int{41, 42}); err == nil {
			t.Fatal("accepted invalid CPU projection")
		}
	}
	for _, pids := range [][2]int{{41, 41}, {0, 42}, {-1, 42}} {
		if _, err := dashboardParseCPU([]byte("41 0:00.00 0:00.00\n42 0:00.00 0:00.00\n"), pids); err == nil {
			t.Fatal("invalid owned identities")
		}
	}
}
func TestDashboardIdleDeltaUsesActualWallAndRejectsRegression(t *testing.T) {
	before := dashboardCPUCounters{User: time.Second, System: 2 * time.Second}
	after := dashboardCPUCounters{User: 2 * time.Second, System: 4 * time.Second}
	observation, err := dashboardObserveIdle(before, after, 301*time.Second)
	if err != nil || observation.User != time.Second || observation.System != 2*time.Second || observation.Wall != 301*time.Second || observation.Percent != 300.0/301 {
		t.Fatal("idle deltas or actual wall", observation, err)
	}
	for _, v := range []struct {
		before, after dashboardCPUCounters
		wall          time.Duration
	}{{before, dashboardCPUCounters{User: 0, System: 4 * time.Second}, 300 * time.Second}, {dashboardCPUCounters{User: -1}, after, 300 * time.Second}, {before, after, 300*time.Second - 1}, {dashboardCPUCounters{}, dashboardCPUCounters{User: math.MaxInt64, System: 1}, 300 * time.Second}} {
		if _, err := dashboardObserveIdle(v.before, v.after, v.wall); err == nil {
			t.Fatal("invalid idle observation")
		}
	}
}
func TestDashboardResourceRecordsOwnPhasesAndConservativePeaks(t *testing.T) {
	for _, mode := range []string{"synthetic-100", "synthetic-10000", "native-small", "projection-10000"} {
		t.Run(mode, func(t *testing.T) {
			prepared := resourceFixture(t, mode)
			work := resourceWorkload(t)
			measure := resourceMeasurements(!prepared.large)
			row, err := dashboardBuildResource(prepared, work, measure)
			if err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(row)
			work.Initial.Samples[0].DurationNS++
			work.InitialRaw[0].Duration++
			prepared.fixture.InventorySHA256 = "modified"
			again, _ := json.Marshal(row)
			if string(encoded) != string(again) {
				t.Fatal("mutable published record")
			}
			if row.Resources.HighWater != 40<<20 || row.Resources.BudgetPeak != 40<<20 || row.Procedure.WarmupWallNS != 60000000000 {
				t.Fatal("actual measurements lost")
			}
		})
	}
}
func TestDashboardResourceRecordsRejectMissingInvalidOrFailedMeasurements(t *testing.T) {
	prepared := resourceFixture(t, "synthetic-100")
	changes := []func(*dashboardResourceMeasurements){func(m *dashboardResourceMeasurements) { m.Warmup-- }, func(m *dashboardResourceMeasurements) { m.SampledRSS = 0 }, func(m *dashboardResourceMeasurements) { m.ManagementHigh = 0 }, func(m *dashboardResourceMeasurements) { m.HelperHigh = 0 }, func(m *dashboardResourceMeasurements) { m.Samples = 0 }, func(m *dashboardResourceMeasurements) { m.MaxGap = 0 }, func(m *dashboardResourceMeasurements) { m.SampledRSS = 128<<20 + 1 }, func(m *dashboardResourceMeasurements) { m.HelperHigh = math.MaxUint64 }, func(m *dashboardResourceMeasurements) { m.Idle.Percent = math.NaN() }, func(m *dashboardResourceMeasurements) { m.Idle.Percent = 0 }, func(m *dashboardResourceMeasurements) { m.Idle.Wall-- }, func(m *dashboardResourceMeasurements) { m.Idle.User = -1 }, func(m *dashboardResourceMeasurements) {
		m.Idle.User = 3 * time.Second
		m.Idle.System = 0
		m.Idle.Percent = 1
	}}
	for _, change := range changes {
		m := resourceMeasurements(true)
		change(&m)
		if row, err := dashboardBuildResource(prepared, resourceWorkload(t), m); err == nil || row != nil {
			t.Fatal("invalid measurement accepted")
		}
	}
	if _, err := dashboardBuildResource(nil, resourceWorkload(t), resourceMeasurements(true)); err == nil {
		t.Fatal("nil prepared input")
	}
	large := resourceFixture(t, "synthetic-10000")
	if _, err := dashboardBuildResource(large, resourceWorkload(t), resourceMeasurements(true)); err == nil {
		t.Fatal("invented large idle")
	}
	m := resourceMeasurements(false)
	m.HelperHigh = 256 << 20
	if _, err := dashboardBuildResource(large, resourceWorkload(t), m); err == nil {
		t.Fatal("high-water sum exceeds large budget")
	}
}
func TestDashboardResourceRecordsRevalidateRawPhasesAndReplacements(t *testing.T) {
	prepared := resourceFixture(t, "synthetic-10000")
	for _, change := range []func(*dashboardMeasuredWorkload){func(w *dashboardMeasuredWorkload) { w.Replacements = 2 }, func(w *dashboardMeasuredWorkload) { w.InitialRaw = w.InitialRaw[:999] }, func(w *dashboardMeasuredWorkload) { w.Initial.Samples[0].DurationNS++ }, func(w *dashboardMeasuredWorkload) { w.ReplacementRaw[0].Predicate = "failed" }, func(w *dashboardMeasuredWorkload) {
		for i := range w.InitialRaw {
			if i%5 == 4 {
				w.InitialRaw[i].Duration = 250 * time.Millisecond
			}
		}
	}} {
		w := resourceWorkload(t)
		change(&w)
		if _, err := dashboardBuildResource(prepared, w, resourceMeasurements(false)); err == nil {
			t.Fatal("malformed, stale or slow phase accepted")
		}
	}
}
func TestDashboardResourceGoJSONPassesIndependentFourModeContract(t *testing.T) {
	python, _ := filepath.Abs("../../.tools/venv/bin/python3")
	if _, err := os.Stat(python); err != nil {
		t.Skip("pinned development Python unavailable; native qualification not established")
	}
	module, _ := filepath.Abs("../../tools/dashboard-resource-evidence.py")
	script := `import importlib.util,json,sys
spec=importlib.util.spec_from_file_location('validator',sys.argv[1]);module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
module.validate_mode(json.load(sys.stdin))
`
	for _, mode := range []string{"synthetic-100", "synthetic-10000", "native-small", "projection-10000"} {
		prepared := resourceFixture(t, mode)
		work := resourceWorkload(t)
		work.ReplacementRaw[0].Outcome = "expiry"
		work.ReplacementRaw[0].Status = 409
		work.ReplacementRaw[0].Predicate = "safe-expiry"
		work.Replacement, _ = dashboardBuildPhaseReceipt(work.ReplacementRaw, "replacement", prepared.large)
		row, err := dashboardBuildResource(prepared, work, resourceMeasurements(!prepared.large))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(row)
		cmd := exec.Command(python, "-c", script, module)
		cmd.Stdin = strings.NewReader(string(b))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatal("independent resource contract", err, string(output))
		}
	}
}
func TestDashboardLegacyResourceFieldsMatchPublishedObservations(t *testing.T) {
	prepared := resourceFixture(t, "native-small")
	m := resourceMeasurements(true)
	m.MaxGap = 1234567
	row, err := dashboardBuildResource(prepared, resourceWorkload(t), m)
	if err != nil {
		t.Fatal(err)
	}
	legacy := dashboardLegacyResource(row, true)
	if legacy["conservative_combined_high_water_rss_bytes"] != row.Resources.HighWater || legacy["idle_cpu_percent_one_core"] != row.Resources.IdlePercent || legacy["max_sample_gap_ms"] != 1.234 || legacy["owned_datasets"] != 100 {
		t.Fatal("legacy observations disagree", legacy)
	}
	q := row.Phases.Initial.Summary["all"].SuccessLatency
	if !reflect.DeepEqual(legacy["p95_ms"], float64(q.P95NS/1000)/1000) {
		t.Fatal("legacy latency rounding")
	}
}

func TestDashboardResourceBudgetEdgesAndSampledPeakRemainObservable(t *testing.T) {
	for _, mode := range []string{"synthetic-100", "synthetic-10000"} {
		p := resourceFixture(t, mode)
		m := resourceMeasurements(!p.large)
		budget := uint64(128 << 20)
		if p.large {
			budget = 256 << 20
		}
		m.SampledRSS = budget
		record, err := dashboardBuildResource(p, resourceWorkload(t), m)
		if err != nil || record.Resources.BudgetPeak != budget || record.Resources.HighWater != 40<<20 {
			t.Fatal("exact boundary or sampled max lost", err)
		}
		m.SampledRSS++
		if _, err := dashboardBuildResource(p, resourceWorkload(t), m); err == nil {
			t.Fatal("sampled peak beyond budget accepted")
		}
	}
}
