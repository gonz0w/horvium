// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math"
)

type dashboardModeControlReceipt struct {
	Fixture              dashboardFixtureReceipt `json:"fixture"`
	PredicateAssertions  int                     `json:"predicate_assertions"`
	Replacements         int                     `json:"replacements"`
	NativeGUIDAssertions int                     `json:"native_guid_assertions"`
}
type dashboardBoundaryReceipt struct {
	AcceptedModelRecords int    `json:"accepted_model_records"`
	RecordExhaustions    int    `json:"record_exhaustions"`
	ByteExhaustions      int    `json:"byte_exhaustions"`
	Recoveries           int    `json:"recoveries"`
	SampledRSS           uint64 `json:"sampled_rss_bytes"`
	HighWater            uint64 `json:"high_water_sum_bytes"`
	Samples              int    `json:"samples"`
	MaxGapNS             int64  `json:"max_sample_gap_ns"`
	OutsideNormalBudget  bool   `json:"outside_normal_budget"`
}
type dashboardNativeControlReceipt struct {
	CapturedCommands int    `json:"captured_commands"`
	SubtreeExits     int    `json:"subtree_exits"`
	StaleRetentions  int    `json:"stale_retentions"`
	Recoveries       int    `json:"recoveries"`
	BootState        string `json:"boot_state"`
}
type dashboardParentControlReceipt struct {
	Parent              string                                 `json:"parent"`
	Modes               map[string]dashboardModeControlReceipt `json:"modes"`
	PredicateAssertions int                                    `json:"predicate_assertions"`
	Boundary            *dashboardBoundaryReceipt              `json:"boundary,omitempty"`
	Native              *dashboardNativeControlReceipt         `json:"native,omitempty"`
}
type dashboardControlLedger struct {
	parent           string
	declarations     map[string]dashboardModeControlReceipt
	modes            map[string]dashboardModeControlReceipt
	boundaryObserved *dashboardBoundaryReceipt
	nativeObserved   *dashboardNativeControlReceipt
	failed           bool
}

func newDashboardControlLedger(parent string) (*dashboardControlLedger, error) {
	if parent != "TestNativePerformanceBudgets" && parent != "TestNativeReaderResourceBudgets" {
		return nil, errors.New("unknown dashboard control parent")
	}
	return &dashboardControlLedger{parent: parent, declarations: map[string]dashboardModeControlReceipt{}, modes: map[string]dashboardModeControlReceipt{}}, nil
}
func (l *dashboardControlLedger) reject() error {
	if l != nil {
		l.failed = true
	}
	return errors.New("incomplete or changed original dashboard controls")
}
func (l *dashboardControlLedger) allowed(mode string) bool {
	if l == nil {
		return false
	}
	if l.parent == "TestNativePerformanceBudgets" {
		return mode == "synthetic-100" || mode == "synthetic-10000"
	}
	return l.parent == "TestNativeReaderResourceBudgets" && (mode == "native-small" || mode == "projection-10000")
}
func (l *dashboardControlLedger) declare(p *dashboardPreparedFixture) error {
	if l == nil || l.failed || p == nil || !l.allowed(p.mode) {
		return l.reject()
	}
	if _, exists := l.declarations[p.mode]; exists {
		return l.reject()
	}
	digest, e := hex.DecodeString(p.fixture.InventorySHA256)
	if e != nil || len(digest) != 32 || hex.EncodeToString(digest) != p.fixture.InventorySHA256 || p.fixture.Datasets != p.count || p.predicate == nil {
		return l.reject()
	}
	native := p.mode == "native-small"
	small := p.mode == "synthetic-100" || native
	if p.large == small || native && (p.count < 100 || p.fixture.OwnedDatasets != 100 || p.nativeGUIDAssertions != 100 || p.fixture.Origin != "native-original") || !native && (p.fixture.OwnedDatasets != 0 || p.nativeGUIDAssertions != 0 || p.fixture.Origin != "generated" || small && p.count != 100 || !small && p.count != 10000) {
		return l.reject()
	}
	l.declarations[p.mode] = dashboardModeControlReceipt{Fixture: p.fixture, NativeGUIDAssertions: p.nativeGUIDAssertions}
	return nil
}
func (l *dashboardControlLedger) completeMode(p *dashboardPreparedFixture, w dashboardMeasuredWorkload, m dashboardResourceMeasurements) error {
	if l == nil || l.failed || p == nil {
		return l.reject()
	}
	declared, exists := l.declarations[p.mode]
	_, duplicate := l.modes[p.mode]
	if !exists || duplicate || declared.Fixture != p.fixture || declared.NativeGUIDAssertions != p.nativeGUIDAssertions {
		return l.reject()
	}
	r, e := dashboardBuildResource(p, w, m)
	if e != nil {
		return l.reject()
	}
	declared.PredicateAssertions = len(r.Phases.Initial.Samples) + len(r.Phases.Replacement.Samples)
	declared.Replacements = r.Procedure.Replacements
	l.modes[p.mode] = declared
	return nil
}
func validDashboardBoundary(r *dashboardBoundaryReceipt) bool {
	return r != nil && r.AcceptedModelRecords == 100000 && r.RecordExhaustions == 1 && r.ByteExhaustions == 1 && r.Recoveries == 1 && r.SampledRSS > 0 && r.HighWater > 0 && r.Samples > 0 && r.MaxGapNS > 0 && r.OutsideNormalBudget
}
func validDashboardNative(r *dashboardNativeControlReceipt) bool {
	return r != nil && r.CapturedCommands == 1 && r.SubtreeExits == 1 && r.StaleRetentions == 1 && r.Recoveries == 1 && r.BootState == "unchanged"
}
func (l *dashboardControlLedger) boundary(r *dashboardBoundaryReceipt) error {
	if l == nil || l.failed || l.parent != "TestNativePerformanceBudgets" || l.boundaryObserved != nil || !validDashboardBoundary(r) {
		return l.reject()
	}
	copy := *r
	l.boundaryObserved = &copy
	return nil
}
func (l *dashboardControlLedger) native(r *dashboardNativeControlReceipt) error {
	if l == nil || l.failed || l.parent != "TestNativeReaderResourceBudgets" || l.nativeObserved != nil || !validDashboardNative(r) {
		return l.reject()
	}
	copy := *r
	l.nativeObserved = &copy
	return nil
}
func (l *dashboardControlLedger) finish() (*dashboardParentControlReceipt, error) {
	invalid := func() (*dashboardParentControlReceipt, error) {
		return nil, errors.New("dashboard parent controls not complete")
	}
	if l == nil || l.failed || len(l.declarations) != 2 || len(l.modes) != 2 {
		return invalid()
	}
	r := &dashboardParentControlReceipt{Parent: l.parent, Modes: map[string]dashboardModeControlReceipt{}}
	for mode, declared := range l.declarations {
		completed, exists := l.modes[mode]
		if !exists || !l.allowed(mode) || completed.Fixture != declared.Fixture || completed.NativeGUIDAssertions != declared.NativeGUIDAssertions || completed.PredicateAssertions != 2000 || completed.Replacements != 3 {
			return invalid()
		}
		r.Modes[mode] = completed
		r.PredicateAssertions += completed.PredicateAssertions
	}
	if r.PredicateAssertions != 4000 {
		return invalid()
	}
	if l.parent == "TestNativePerformanceBudgets" {
		if !validDashboardBoundary(l.boundaryObserved) || l.nativeObserved != nil {
			return invalid()
		}
		copy := *l.boundaryObserved
		r.Boundary = &copy
	} else {
		if !validDashboardNative(l.nativeObserved) || l.boundaryObserved != nil {
			return invalid()
		}
		copy := *l.nativeObserved
		r.Native = &copy
	}
	return r, nil
}

// Call each transition only after its existing native assertion succeeds. Portable
// events test sequencing; they do not independently establish command execution.
type dashboardBoundaryControl struct {
	stage    int
	failed   bool
	observed dashboardBoundaryReceipt
}

func (b *dashboardBoundaryControl) step(event string, count int) error {
	expected := []string{"accepted-model", "record-exhaustion", "byte-exhaustion", "recovery"}
	counts := []int{100000, 100001, 20010, 110}
	if b == nil || b.failed || b.stage >= len(expected) || event != expected[b.stage] || count != counts[b.stage] {
		if b != nil {
			b.failed = true
		}
		return errors.New("incomplete or reordered boundary control")
	}
	switch b.stage {
	case 0:
		b.observed.AcceptedModelRecords = count
	case 1:
		b.observed.RecordExhaustions++
	case 2:
		b.observed.ByteExhaustions++
	case 3:
		b.observed.Recoveries++
	}
	b.stage++
	return nil
}
func (b *dashboardBoundaryControl) finish(m dashboardResourceMeasurements) (*dashboardBoundaryReceipt, error) {
	if b == nil || b.failed || b.stage != 4 || m.SampledRSS == 0 || m.ManagementHigh == 0 || m.HelperHigh == 0 || m.ManagementHigh > math.MaxUint64-m.HelperHigh || m.Samples < 1 || m.MaxGap <= 0 {
		return nil, errors.New("boundary measurement unavailable")
	}
	r := b.observed
	r.SampledRSS, r.HighWater, r.Samples, r.MaxGapNS = m.SampledRSS, m.ManagementHigh+m.HelperHigh, m.Samples, int64(m.MaxGap)
	r.OutsideNormalBudget = b.stage == 4
	return &r, nil
}

type dashboardNativeControl struct {
	stage      int
	failed     bool
	generation string
	observed   dashboardNativeControlReceipt
}

func (n *dashboardNativeControl) reject() error {
	if n != nil {
		n.failed = true
	}
	return errors.New("incomplete native recovery control")
}
func (n *dashboardNativeControl) captured(owner, guard, command int) error {
	if n == nil || n.failed || n.stage != 0 || owner <= 1 || guard <= 1 || command <= 1 || owner == guard || owner == command || guard == command {
		return n.reject()
	}
	n.observed.CapturedCommands++
	n.stage++
	return nil
}
func (n *dashboardNativeControl) subtreeExited() error {
	if n == nil || n.failed || n.stage != 1 {
		return n.reject()
	}
	n.observed.SubtreeExits++
	n.stage++
	return nil
}
func (n *dashboardNativeControl) retained(before, after string, stale bool) error {
	if n == nil || n.failed || n.stage != 2 || before == "" || len(before) > 128 || before != after || !stale {
		return n.reject()
	}
	n.generation = before
	n.observed.StaleRetentions++
	n.stage++
	return nil
}
func (n *dashboardNativeControl) recovered(before, after string) error {
	if n == nil || n.failed || n.stage != 3 || before != n.generation || after == "" || len(after) > 128 || before == after {
		return n.reject()
	}
	n.observed.Recoveries++
	n.stage++
	return nil
}
func (n *dashboardNativeControl) unchangedBoot(before, after, beforeDigest, afterDigest []byte) (*dashboardNativeControlReceipt, error) {
	if n == nil || n.failed || n.stage != 4 || len(before) == 0 || len(before) > 65536 || len(beforeDigest) == 0 || len(beforeDigest) > 65536 || !bytes.Equal(before, after) || !bytes.Equal(beforeDigest, afterDigest) {
		return nil, n.reject()
	}
	n.stage++
	n.observed.BootState = "unchanged"
	r := n.observed
	return &r, nil
}

func dashboardDeclaredModelRecords(datasets, padding int) int {
	if datasets < 0 || datasets > 100000 || padding < 0 || padding > 4000 {
		return -1
	}
	// Padding changes encoded bytes, never object count. Inspect the generator's
	// actual fixed sections without allocating another oversized dataset model.
	s := performanceSnapshot(0, 0)
	return len(s.GEOM.Data.Classes) + len(s.GEOM.Data.Geoms) + len(s.GEOM.Data.Providers) + len(s.Pools.Data) + datasets
}
