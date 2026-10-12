//go:build horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func provenanceChildFixture() []byte {
	return []byte("=== RUN   TestMultiplePolicyStartupProvenanceChild\nPOLICY_STARTUP_PROVENANCE_TRACE {\"status\":\"success\",\"frames\":[]}\nPOLICY_STARTUP_PROVENANCE_SUMMARY {}\nPOLICY_STARTUP_PROVENANCE_CHILD_RESULT " + strings.Join(policyProvenanceChildControls, " ") + "\n--- PASS: TestMultiplePolicyStartupProvenanceChild (1.0s)\nPASS\n")
}

func TestPolicyProvenanceChildTranscript(t *testing.T) {
	data := provenanceChildFixture()
	summary, trace, ok := policyProvenanceChild(data)
	if !ok || !bytes.Equal(summary, []byte("{}")) || !bytes.Equal(trace, []byte(`{"status":"success","frames":[]}`)) || len(policyProvenanceChildControls) != 13 {
		t.Fatal("complete exact child evidence lost")
	}
	for _, bad := range [][]byte{nil, append(append([]byte{}, data...), data...), append(append([]byte{}, data...), []byte("raw inventory\n")...), bytes.ReplaceAll(data, []byte("--- PASS:"), []byte("--- FAIL:")), bytes.ReplaceAll(data, []byte("(1.0s)"), []byte("(211.0s)")), bytes.ReplaceAll(data, []byte("frames\":[]"), []byte("frames\":[],\"raw\":1")), bytes.Repeat([]byte("x"), 16<<10+1)} {
		if _, _, ok := policyProvenanceChild(bad); ok {
			t.Fatal("unknown, failed or unbounded child accepted")
		}
	}
	for _, name := range policyProvenanceChildControls {
		for _, replacement := range []string{"", name + " " + name, "unknown"} {
			bad := strings.Replace(string(data), name, replacement, 1)
			if _, _, ok := policyProvenanceChild([]byte(bad)); ok {
				t.Fatal("inexact child control accepted")
			}
		}
	}
}

func TestPolicyProvenanceTraceSanitizationAndOriginalContexts(t *testing.T) {
	data := []byte(`{"status":"success","frames":[]}`)
	trace, ok := policyProvenanceTrace(data)
	if !ok || !policyProvenanceOriginalContexts(trace, 1) {
		t.Fatal("explicit empty success lost")
	}
	for _, bad := range []string{`{}`, `{"status":"success","frames":null}`, `{"status":"success","status":"success","frames":[]}`, `{"status":"unknown","frames":[]}`, `{"status":"success","frames":[],"raw":"private"}`} {
		if _, ok := policyProvenanceTrace([]byte(bad)); ok {
			t.Fatal("inexact scalar shape accepted")
		}
	}
	// Scalar fixture only: a return observation on the original 180s caller.
	row := `{"status":"complete","frames":[{"site":"activation","operation":"deadline","phase":"final-inspection","branch":"0","allowance_ns":90000000000,"child_present":true,"child_deadline":true,"child_deadline_ns":90000000000,"parent_deadline":true,"parent_deadline_ns":180000000000,"child_before":"deadline","child_after":"deadline","parent_before":"none","parent_after":"none","begin_ns":121000000000,"end_ns":121000000001,"constraint":"local-expiry"}]}`
	trace, ok = policyProvenanceTrace([]byte(row))
	if !ok || !policyProvenanceOriginalContexts(trace, 122000000000) {
		t.Fatal("original bounded return snapshot lost")
	}
	for _, bad := range []string{strings.Replace(row, "activation", "/private/path", 1), strings.Replace(row, `"branch":"0"`, `"branch":"raw"`, 1), strings.Replace(row, `"parent_after":"none"`, `"parent_after":"raw"`, 1), strings.Replace(row, `"constraint":"local-expiry"`, `"constraint":"raw"`, 1)} {
		if _, ok := policyProvenanceTrace([]byte(bad)); ok {
			t.Fatal("raw scalar identifier accepted")
		}
	}
	for _, change := range []func(){func() { trace.Frames[0].ParentDeadlineNS++ }, func() { trace.Frames[0].EndNS = 123000000000 }, func() { trace.Frames[0].ParentDeadline = false }} {
		if err := json.Unmarshal([]byte(row), &trace); err != nil {
			t.Fatal(err)
		}
		change()
		if policyProvenanceOriginalContexts(trace, 122000000000) {
			t.Fatal("renewed ancestry or post-return snapshot accepted")
		}
	}
}
