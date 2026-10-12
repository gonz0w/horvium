//go:build horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gonz0w/horvium/internal/storage"
)

var policyProvenanceChildControls = []string{"cpu_affinity", "child_budget", "caller_deadline", "original_call", "sampler_joined", "samples_bounded", "owner_release", "deadline_unchanged", "trace_enabled", "error_preserved", "snapshots_before_cancel", "trace_inventory", "original_contexts"}

func policyProvenanceChild(data []byte) ([]byte, []byte, bool) {
	if len(data) > 16<<10 {
		return nil, nil, false
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 6 || lines[0] != "=== RUN   TestMultiplePolicyStartupProvenanceChild" || lines[5] != "PASS" {
		return nil, nil, false
	}
	match := regexp.MustCompile(`^--- PASS: TestMultiplePolicyStartupProvenanceChild \(([0-9]+(?:\.[0-9]+)?)s\)$`).FindStringSubmatch(lines[4])
	if len(match) != 2 {
		return nil, nil, false
	}
	elapsed, e := strconv.ParseFloat(match[1], 64)
	if e != nil || elapsed < 0 || elapsed > 210 {
		return nil, nil, false
	}
	trace, ok := strings.CutPrefix(lines[1], "POLICY_STARTUP_PROVENANCE_TRACE ")
	if !ok {
		return nil, nil, false
	}
	if _, ok = policyProvenanceTrace([]byte(trace)); !ok {
		return nil, nil, false
	}
	summary, ok := strings.CutPrefix(lines[2], "POLICY_STARTUP_PROVENANCE_SUMMARY ")
	if !ok || len(summary) > 8192 || !json.Valid([]byte(summary)) {
		return nil, nil, false
	}
	controls, ok := strings.CutPrefix(lines[3], "POLICY_STARTUP_PROVENANCE_CHILD_RESULT ")
	actual := strings.Fields(controls)
	want := append([]string{}, policyProvenanceChildControls...)
	slices.Sort(actual)
	slices.Sort(want)
	if !ok || !slices.Equal(actual, want) {
		return nil, nil, false
	}
	return []byte(summary), []byte(trace), true
}

func policyProvenanceTrace(data []byte) (storage.DeadlineProvenance, bool) {
	var trace storage.DeadlineProvenance
	if len(data) > 8192 || json.Unmarshal(data, &trace) != nil || trace.Frames == nil || len(trace.Frames) > 12 {
		return trace, false
	}
	canonical, e := json.Marshal(trace)
	if e != nil || !bytes.Equal(canonical, data) || !slices.Contains([]string{"success", "not-deadline", "unattributed", "complete", "ambiguous", "invalid", "overflow"}, trace.Status) {
		return trace, false
	}
	for _, f := range trace.Frames {
		if !slices.Contains([]string{"review", "activation", "backing-collector", "backing-command", "pool-collector", "pool-command", "provider-collector", "system-collector", "system-command"}, f.Site) || !slices.Contains([]string{"guard", "inspect", "first-inspection", "second-inspection", "publication", "final-inspection", "collection", "capture"}, f.Phase) || f.Operation != "deadline" || !regexp.MustCompile(`^0(?:\.(?:[0-9]|1[01])){0,47}$`).MatchString(f.Branch) || !slices.Contains([]string{"local-expiry", "inherited", "parent-expired", "both-expired", "parent-only", "canceled", "overlap", "unresolved"}, f.Constraint) {
			return trace, false
		}
		for _, class := range []string{f.ChildBefore, f.ChildAfter, f.ParentBefore, f.ParentAfter} {
			if !slices.Contains([]string{"none", "deadline", "canceled", "other", "absent"}, class) {
				return trace, false
			}
		}
	}
	return trace, true
}

func policyProvenanceOriginalContexts(trace storage.DeadlineProvenance, returned int64) bool {
	if returned < 0 || returned > 210000000000 {
		return false
	}
	for _, f := range trace.Frames {
		if f.BeginNS < 0 || f.EndNS < f.BeginNS || f.EndNS > returned || !f.ParentDeadline || f.ParentDeadlineNS < 0 || f.ParentDeadlineNS > 180000000000 || (f.ChildPresent && (!f.ChildDeadline || f.ChildDeadlineNS < 0 || f.ChildDeadlineNS > f.ParentDeadlineNS)) {
			return false
		}
	}
	return true
}
