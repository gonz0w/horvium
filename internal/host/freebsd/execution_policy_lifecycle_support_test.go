// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// Test-only typed scalar records; normal product builds exclude this file.
type policyLifecycleStage struct {
	Mode              string `json:"mode"`
	Stage             string `json:"stage"`
	ElapsedNS         int64  `json:"elapsed_ns"`
	BudgetNS          int64  `json:"budget_ns"`
	DeadlineNS        int64  `json:"deadline_ns"`
	DeadlineUnchanged bool   `json:"deadline_unchanged"`
	OperationError    string `json:"operation_error"`
	CallerError       string `json:"caller_error"`
	OwnerPresent      *bool  `json:"owner_present,omitempty"`
}

// Explicit implementation-absent fixture stubs for compiled behavioral red.
func newPolicyLifecycleCaller(stage string, start time.Time) (context.Context, context.CancelFunc, error) {
	return nil, nil, errors.New("fixture caller not implemented")
}
func newPolicyLifecycleStage(mode, stage string, start time.Time, c context.Context, original time.Time, operation error, owner *bool) (policyLifecycleStage, error) {
	return policyLifecycleStage{}, errors.New("fixture scalar not implemented")
}
func policyLifecycleStageRows(output []byte, mode string) ([]string, error) {
	return nil, errors.New("fixture scalar extraction not implemented")
}

func TestPolicyLifecycleOriginalCaller(t *testing.T) {
	for _, stage := range []string{"startup", "observe", "copied", "copied-finish", "decoded", "decoded-finish", "finish", "retired-observe", "retired-finish", "registration-consumed"} {
		start := time.Now()
		c, stop, err := newPolicyLifecycleCaller(stage, start)
		if err != nil || c == nil || stop == nil {
			t.Fatal("original fixed caller", stage, err)
		}
		budget := 90 * time.Second
		if stage == "startup" {
			budget = 180 * time.Second
		}
		deadline, ok := c.Deadline()
		if !ok || !deadline.Equal(start.Add(budget)) || c.Err() != nil {
			t.Fatal("actual absolute stage deadline")
		}
		stop()
		again, _ := c.Deadline()
		if !again.Equal(deadline) || !errors.Is(c.Err(), context.Canceled) {
			t.Fatal("cancellation must not renew deadline")
		}
	}
	for _, stage := range []string{"", "unknown", "startup-renewed"} {
		if c, stop, err := newPolicyLifecycleCaller(stage, time.Now()); err == nil || c != nil || stop != nil {
			t.Fatal("unknown stage grants no context")
		}
	}
	if c, stop, err := newPolicyLifecycleCaller("startup", time.Time{}); err == nil || c != nil || stop != nil {
		t.Fatal("missing original start denied")
	}
}

func TestPolicyLifecycleScalarBeforeCancellation(t *testing.T) {
	start := time.Now()
	c, stop, err := newPolicyLifecycleCaller("startup", start)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	deadline, _ := c.Deadline()
	owner := true
	row, err := newPolicyLifecycleStage("first", "startup", start, c, deadline, nil, &owner)
	if err != nil || row.BudgetNS != int64(180*time.Second) || row.DeadlineNS != row.BudgetNS || !row.DeadlineUnchanged || row.CallerError != "none" || row.OperationError != "none" || row.OwnerPresent == nil || !*row.OwnerPresent || row.ElapsedNS <= 0 {
		t.Fatal("original pre-cancel scalar", row, err)
	}
	stop()
	if row.CallerError != "none" {
		t.Fatal("later cancellation rewrote observation")
	}
	owner = false
	row, err = newPolicyLifecycleStage("restart", "startup", start, c, deadline, context.Canceled, &owner)
	if err != nil || row.CallerError != "cancelled" || row.OperationError != "cancelled" || *row.OwnerPresent {
		t.Fatal("honest operation/caller error", row, err)
	}
	if _, err = newPolicyLifecycleStage("first", "startup", start, c, deadline.Add(time.Second), nil, &owner); err == nil {
		t.Fatal("renewed original deadline accepted")
	}
	old, oldStop := context.WithDeadline(context.Background(), start.Add(120*time.Second))
	defer oldStop()
	if _, err = newPolicyLifecycleStage("first", "startup", start, old, deadline, nil, &owner); err == nil {
		t.Fatal("old caller profile accepted")
	}
}

func TestPolicyLifecycleScalarExtraction(t *testing.T) {
	var output []byte
	for _, stage := range []string{"startup", "observe", "copied", "copied-finish", "decoded", "decoded-finish", "finish", "retired-observe", "retired-finish", "registration-consumed"} {
		budget := int64(90 * time.Second)
		operation := "denied"
		if stage == "startup" || stage == "observe" || stage == "finish" {
			operation = "none"
		}
		row := policyLifecycleStage{Mode: "first", Stage: stage, ElapsedNS: 1, BudgetNS: budget, DeadlineNS: budget, DeadlineUnchanged: true, OperationError: operation, CallerError: "none"}
		if stage == "startup" {
			row.BudgetNS, row.DeadlineNS = int64(180*time.Second), int64(180*time.Second)
			present := true
			row.OwnerPresent = &present
		}
		data, _ := json.Marshal(row)
		output = append(output, []byte("POLICY_ROOT_LIFECYCLE_STAGE ")...)
		output = append(output, data...)
		output = append(output, '\n')
	}
	rows, err := policyLifecycleStageRows(output, "first")
	if err != nil || len(rows) != 10 {
		t.Fatal("exact ordered scalar inventory", err)
	}
	for _, bad := range [][]byte{nil, append(append([]byte(nil), output...), output...), output[:len(output)-10], []byte("POLICY_ROOT_LIFECYCLE_STAGE {\"mode\":\"first\",\"mode\":\"first\"}\n")} {
		if _, err = policyLifecycleStageRows(bad, "first"); err == nil {
			t.Fatal("incomplete/duplicate scalar accepted")
		}
	}
	if _, err = policyLifecycleStageRows(output, "restart"); err == nil {
		t.Fatal("foreign cycle scalars accepted")
	}
}
