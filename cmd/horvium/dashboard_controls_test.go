// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func controlsLedger(t *testing.T, parent string) *dashboardControlLedger {
	t.Helper()
	l, e := newDashboardControlLedger(parent)
	if e != nil {
		t.Fatal(e)
	}
	return l
}
func controlsMode(t *testing.T, l *dashboardControlLedger, mode string) {
	t.Helper()
	p := resourceFixture(t, mode)
	if e := l.declare(p); e != nil {
		t.Fatal(e)
	}
	if e := l.completeMode(p, resourceWorkload(t), resourceMeasurements(!p.large)); e != nil {
		t.Fatal(e)
	}
}
func controlsBoundary(t *testing.T) *dashboardBoundaryReceipt {
	t.Helper()
	b := new(dashboardBoundaryControl)
	for i, e := range []string{"accepted-model", "record-exhaustion", "byte-exhaustion", "recovery"} {
		if err := b.step(e, []int{100000, 100001, 20010, 110}[i]); err != nil {
			t.Fatal(err)
		}
	}
	r, e := b.finish(resourceMeasurements(false))
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func controlsNative(t *testing.T) *dashboardNativeControlReceipt {
	t.Helper()
	n := new(dashboardNativeControl)
	if e := n.captured(41, 42, 43); e != nil {
		t.Fatal(e)
	}
	if e := n.subtreeExited(); e != nil {
		t.Fatal(e)
	}
	if e := n.retained("original", "original", true); e != nil {
		t.Fatal(e)
	}
	if e := n.recovered("original", "replacement"); e != nil {
		t.Fatal(e)
	}
	r, e := n.unchangedBoot([]byte("config"), []byte("config"), []byte("digest"), []byte("digest"))
	if e != nil {
		t.Fatal(e)
	}
	return r
}

func TestDashboardControlsCompleteOriginalParents(t *testing.T) {
	for _, parent := range []string{"TestNativePerformanceBudgets", "TestNativeReaderResourceBudgets"} {
		l := controlsLedger(t, parent)
		modes := []string{"synthetic-100", "synthetic-10000"}
		if parent == "TestNativeReaderResourceBudgets" {
			modes = []string{"native-small", "projection-10000"}
		}
		for _, mode := range modes {
			controlsMode(t, l, mode)
		}
		if parent == "TestNativePerformanceBudgets" {
			if e := l.boundary(controlsBoundary(t)); e != nil {
				t.Fatal(e)
			}
		} else {
			if e := l.native(controlsNative(t)); e != nil {
				t.Fatal(e)
			}
		}
		r, e := l.finish()
		if e != nil || r.PredicateAssertions != 4000 || len(r.Modes) != 2 {
			t.Fatal("incomplete actual counts", r, e)
		}
		for _, mode := range modes {
			m := r.Modes[mode]
			if m.PredicateAssertions != 2000 || m.Replacements != 3 || m.NativeGUIDAssertions != m.Fixture.OwnedDatasets {
				t.Fatal("wrong original mode controls", m)
			}
		}
		before, _ := json.Marshal(r)
		l.modes[modes[0]] = dashboardModeControlReceipt{}
		again, _ := json.Marshal(r)
		if !bytes.Equal(before, again) {
			t.Fatal("ledger aliases returned observation")
		}
		if _, e := l.finish(); e == nil {
			t.Fatal("damaged completed observations accepted")
		}
	}
}
func TestDashboardControlsRejectMissingDuplicateAndWrongParent(t *testing.T) {
	if _, e := newDashboardControlLedger("other"); e == nil {
		t.Fatal("wrong parent")
	}
	for _, violation := range []func(*dashboardControlLedger) error{
		func(l *dashboardControlLedger) error { return l.declare(resourceFixture(t, "native-small")) },
		func(l *dashboardControlLedger) error {
			return l.completeMode(resourceFixture(t, "synthetic-100"), resourceWorkload(t), resourceMeasurements(true))
		},
		func(l *dashboardControlLedger) error {
			p := resourceFixture(t, "synthetic-100")
			if e := l.declare(p); e != nil {
				t.Fatal(e)
			}
			return l.declare(p)
		},
		func(l *dashboardControlLedger) error {
			controlsMode(t, l, "synthetic-100")
			return l.completeMode(resourceFixture(t, "synthetic-100"), resourceWorkload(t), resourceMeasurements(true))
		},
		func(l *dashboardControlLedger) error { return l.native(controlsNative(t)) },
		func(l *dashboardControlLedger) error {
			if e := l.boundary(controlsBoundary(t)); e != nil {
				t.Fatal(e)
			}
			return l.boundary(controlsBoundary(t))
		},
	} {
		l := controlsLedger(t, "TestNativePerformanceBudgets")
		if e := violation(l); e == nil {
			t.Fatal("invalid control accepted")
		}
		if _, e := l.finish(); e == nil {
			t.Fatal("invalid operation promoted to pass")
		}
		if e := l.declare(resourceFixture(t, "synthetic-10000")); e == nil {
			t.Fatal("failed ledger silently retried")
		}
	}
	l := controlsLedger(t, "TestNativePerformanceBudgets")
	if _, e := l.finish(); e == nil {
		t.Fatal("empty accepted")
	}
	controlsMode(t, l, "synthetic-100")
	if _, e := l.finish(); e == nil {
		t.Fatal("incomplete parent accepted")
	}
}

func TestDashboardControlsRejectChangedOriginalAndFailedRaw(t *testing.T) {
	for _, change := range []func(*dashboardPreparedFixture, *dashboardMeasuredWorkload, *dashboardResourceMeasurements){
		func(p *dashboardPreparedFixture, _ *dashboardMeasuredWorkload, _ *dashboardResourceMeasurements) {
			p.fixture.InventorySHA256 = "changed"
		},
		func(p *dashboardPreparedFixture, _ *dashboardMeasuredWorkload, _ *dashboardResourceMeasurements) {
			p.fixture.Datasets++
		},
		func(p *dashboardPreparedFixture, _ *dashboardMeasuredWorkload, _ *dashboardResourceMeasurements) {
			p.nativeGUIDAssertions++
		},
		func(_ *dashboardPreparedFixture, w *dashboardMeasuredWorkload, _ *dashboardResourceMeasurements) {
			w.InitialRaw[0].Predicate = "wrong"
		},
		func(_ *dashboardPreparedFixture, w *dashboardMeasuredWorkload, _ *dashboardResourceMeasurements) {
			w.Replacements = 2
		},
		func(_ *dashboardPreparedFixture, _ *dashboardMeasuredWorkload, m *dashboardResourceMeasurements) {
			m.Samples = 0
		},
	} {
		l := controlsLedger(t, "TestNativeReaderResourceBudgets")
		p := resourceFixture(t, "native-small")
		w := resourceWorkload(t)
		m := resourceMeasurements(true)
		if e := l.declare(p); e != nil {
			t.Fatal(e)
		}
		change(p, &w, &m)
		if e := l.completeMode(p, w, m); e == nil || len(l.modes) != 0 {
			t.Fatal("changed/failed mode counted")
		}
	}
}
func TestDashboardBoundaryControlsRequireActualOrderedSteps(t *testing.T) {
	for _, first := range []string{"record-exhaustion", "byte-exhaustion", "recovery", "unknown"} {
		b := new(dashboardBoundaryControl)
		if e := b.step(first, 100000); e == nil {
			t.Fatal("reordered step")
		}
		if _, e := b.finish(resourceMeasurements(false)); e == nil {
			t.Fatal("partial boundary")
		}
	}
	for _, count := range []int{0, 99999, 100001} {
		b := new(dashboardBoundaryControl)
		if e := b.step("accepted-model", count); e == nil {
			t.Fatal("wrong accepted boundary")
		}
	}
	b := new(dashboardBoundaryControl)
	if e := b.step("accepted-model", 100000); e != nil {
		t.Fatal(e)
	}
	if e := b.step("accepted-model", 100000); e == nil {
		t.Fatal("duplicate boundary step")
	}
	for _, change := range []func(*dashboardResourceMeasurements){func(m *dashboardResourceMeasurements) { m.SampledRSS = 0 }, func(m *dashboardResourceMeasurements) { m.HelperHigh = 0 }, func(m *dashboardResourceMeasurements) { m.Samples = 0 }, func(m *dashboardResourceMeasurements) { m.MaxGap = 0 }} {
		b := new(dashboardBoundaryControl)
		for i, event := range []string{"accepted-model", "record-exhaustion", "byte-exhaustion", "recovery"} {
			if e := b.step(event, []int{100000, 100001, 20010, 110}[i]); e != nil {
				t.Fatal(e)
			}
		}
		m := resourceMeasurements(false)
		change(&m)
		if _, e := b.finish(m); e == nil {
			t.Fatal("missing real boundary measurement")
		}
	}
}
func TestDashboardNativeControlsRejectIncompleteChangedAndReordered(t *testing.T) {
	for _, ids := range [][3]int{{0, 42, 43}, {41, 41, 43}, {41, 42, 42}} {
		n := new(dashboardNativeControl)
		if e := n.captured(ids[0], ids[1], ids[2]); e == nil {
			t.Fatal("unverified original identities")
		}
	}
	n := new(dashboardNativeControl)
	if e := n.subtreeExited(); e == nil {
		t.Fatal("exit before capture")
	}
	if _, e := n.unchangedBoot([]byte("a"), []byte("a"), []byte("b"), []byte("b")); e == nil {
		t.Fatal("boot without native controls")
	}
	for _, bad := range []int{0, 1, 2, 3} {
		n := new(dashboardNativeControl)
		n.captured(41, 42, 43)
		n.subtreeExited()
		if bad == 0 {
			if e := n.retained("before", "after", true); e == nil {
				t.Fatal("changed stale generation")
			}
			continue
		}
		if bad == 1 {
			if e := n.retained("before", "before", false); e == nil {
				t.Fatal("missing stale")
			}
			continue
		}
		n.retained("before", "before", true)
		if bad == 2 {
			if e := n.recovered("before", "before"); e == nil {
				t.Fatal("no recovery")
			}
			continue
		}
		n.recovered("before", "after")
		if _, e := n.unchangedBoot([]byte("a"), []byte("changed"), []byte("b"), []byte("b")); e == nil {
			t.Fatal("changed boot state")
		}
	}
}

func TestDashboardControlsCountSafeExpiryAndOwnControlCopies(t *testing.T) {
	l := controlsLedger(t, "TestNativePerformanceBudgets")
	p := resourceFixture(t, "synthetic-100")
	w := resourceWorkload(t)
	w.ReplacementRaw[0].Outcome = "expiry"
	w.ReplacementRaw[0].Status = 409
	w.ReplacementRaw[0].Predicate = "safe-expiry"
	var err error
	w.Replacement, err = dashboardBuildPhaseReceipt(w.ReplacementRaw, "replacement", false)
	if err != nil {
		t.Fatal(err)
	}
	if err = l.declare(p); err != nil {
		t.Fatal(err)
	}
	if err = l.completeMode(p, w, resourceMeasurements(true)); err != nil {
		t.Fatal(err)
	}
	controlsMode(t, l, "synthetic-10000")
	b := controlsBoundary(t)
	if err = l.boundary(b); err != nil {
		t.Fatal(err)
	}
	b.Samples = 0
	r, err := l.finish()
	if err != nil || r.PredicateAssertions != 4000 || r.Boundary.Samples == 0 {
		t.Fatal("safe expiry or detached control lost")
	}
	r.Boundary.Samples = 0
	r.Modes["synthetic-100"] = dashboardModeControlReceipt{}
	next, err := l.finish()
	if err != nil || next.Boundary.Samples == 0 || next.Modes["synthetic-100"].PredicateAssertions != 2000 {
		t.Fatal("returned record aliases ledger")
	}
	data, _ := json.Marshal(next)
	if bytes.Contains(data, []byte("vault")) || bytes.Contains(data, []byte("/tmp")) || bytes.Contains(data, []byte("DASHBOARD_CONTROLS")) {
		t.Fatal("private identity or full acceptance marker leaked")
	}
}
func TestDashboardDeclaredModelRecordsMatchOriginalGenerator(t *testing.T) {
	for _, count := range []int{0, 100, 20000, 99990, 99991} {
		s := performanceSnapshot(count, 0)
		actual := len(s.GEOM.Data.Classes) + len(s.GEOM.Data.Geoms) + len(s.GEOM.Data.Providers) + len(s.Pools.Data) + len(s.Datasets.Data)
		if dashboardDeclaredModelRecords(count, 0) != actual || dashboardDeclaredModelRecords(count, 4000) != actual {
			t.Fatal("declared count diverged from generator")
		}
	}
	for _, args := range [][2]int{{-1, 0}, {100001, 0}, {100, -1}, {100, 4001}} {
		if dashboardDeclaredModelRecords(args[0], args[1]) != -1 {
			t.Fatal("invalid control shape")
		}
	}
}
