// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
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

var policyLifecycleStages = []string{"startup", "observe", "copied", "copied-finish", "decoded", "decoded-finish", "finish", "retired-observe", "retired-finish", "registration-consumed"}

func policyLifecycleStageBudget(stage string) (time.Duration, error) {
	for _, name := range policyLifecycleStages {
		if name == stage {
			if stage == "startup" {
				return 180 * time.Second, nil
			}
			return 90 * time.Second, nil
		}
	}
	return 0, errors.New("unknown fixture stage")
}

// One absolute test caller; the fixed factory receives this same context.
func newPolicyLifecycleCaller(stage string, start time.Time) (context.Context, context.CancelFunc, error) {
	budget, err := policyLifecycleStageBudget(stage)
	if err != nil || start.IsZero() {
		return nil, nil, errors.New("invalid original fixture caller")
	}
	c, stop := context.WithDeadline(context.Background(), start.Add(budget))
	return c, stop, nil
}

func policyLifecycleError(err error) string {
	switch {
	case err == nil:
		return "none"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, storage.ErrDenied), errors.Is(err, storage.ErrStateUncertain), errors.Is(err, ErrPermission):
		return "denied"
	default:
		return "other"
	}
}

// Caller records this before its deferred cancel; no arguments or raw errors.
func newPolicyLifecycleStage(mode, stage string, start time.Time, c context.Context, original time.Time, operation error, owner *bool) (policyLifecycleStage, error) {
	budget, err := policyLifecycleStageBudget(stage)
	if err != nil || start.IsZero() || c == nil || (mode != "first" && mode != "restart") || !original.Equal(start.Add(budget)) {
		return policyLifecycleStage{}, errors.New("invalid original scalar context")
	}
	deadline, ok := c.Deadline()
	if !ok || !deadline.Equal(original) || (stage == "startup") != (owner != nil) {
		return policyLifecycleStage{}, errors.New("renewed or missing scalar deadline/owner")
	}
	row := policyLifecycleStage{Mode: mode, Stage: stage, ElapsedNS: time.Since(start).Nanoseconds(), BudgetNS: int64(budget), DeadlineNS: deadline.Sub(start).Nanoseconds(), DeadlineUnchanged: true, OperationError: policyLifecycleError(operation), CallerError: policyLifecycleError(c.Err())}
	if owner != nil {
		present := *owner
		row.OwnerPresent = &present
		if present != (operation == nil) {
			return policyLifecycleStage{}, errors.New("inconsistent startup result")
		}
	}
	if row.ElapsedNS <= 0 || row.ElapsedNS > int64(390*time.Second) {
		return policyLifecycleStage{}, errors.New("invalid scalar elapsed time")
	}
	return row, nil
}

func policyLifecycleStageRows(output []byte, mode string) ([]string, error) {
	if len(output) > 16<<10 || (mode != "first" && mode != "restart") {
		return nil, errors.New("unbounded or unknown scalar child")
	}
	const prefix = "POLICY_ROOT_LIFECYCLE_STAGE "
	var rows []string
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		if len(rows) >= len(policyLifecycleStages) {
			return nil, errors.New("extra scalar stage")
		}
		data := []byte(strings.TrimPrefix(line, prefix))
		var row policyLifecycleStage
		if err := json.Unmarshal(data, &row); err != nil {
			return nil, err
		}
		canonical, err := json.Marshal(row)
		budget, budgetErr := policyLifecycleStageBudget(row.Stage)
		positive := row.Stage == "startup" || row.Stage == "observe" || row.Stage == "finish"
		operation := "denied"
		if positive {
			operation = "none"
		}
		if err != nil || !bytes.Equal(canonical, data) || budgetErr != nil || row.Mode != mode || row.Stage != policyLifecycleStages[len(rows)] || row.ElapsedNS <= 0 || row.ElapsedNS > int64(budget) || row.BudgetNS != int64(budget) || row.DeadlineNS != int64(budget) || !row.DeadlineUnchanged || row.CallerError != "none" || row.OperationError != operation || (row.Stage == "startup") != (row.OwnerPresent != nil) || (row.OwnerPresent != nil && !*row.OwnerPresent) {
			return nil, errors.New("invalid original scalar stage")
		}
		rows = append(rows, line)
	}
	if len(rows) != len(policyLifecycleStages) {
		return nil, errors.New("incomplete scalar stages")
	}
	return rows, nil
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
	if row.OwnerPresent == nil || !*row.OwnerPresent {
		t.Fatal("later owner variable mutation rewrote observation")
	}
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

func TestPolicyLifecycleFailureClassification(t *testing.T) {
	for _, expired := range []bool{false, true} {
		start := time.Now()
		if expired {
			start = start.Add(-181 * time.Second)
		}
		c, stop, err := newPolicyLifecycleCaller("startup", start)
		if err != nil {
			t.Fatal(err)
		}
		deadline, _ := c.Deadline()
		owner := false
		row, err := newPolicyLifecycleStage("restart", "startup", start, c, deadline, errors.Join(ErrPermission, context.DeadlineExceeded), &owner)
		stop()
		caller := "none"
		if expired {
			caller = "deadline"
		}
		if err != nil || row.OperationError != "deadline" || row.CallerError != caller || row.OwnerPresent == nil || *row.OwnerPresent {
			t.Fatal("nested timeout must not fabricate caller expiry", row, err)
		}
		data, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = policyLifecycleStageRows(append([]byte("POLICY_ROOT_LIFECYCLE_STAGE "), data...), "restart"); err == nil {
			t.Fatal("honest partial failed observation cannot qualify child")
		}
	}
	if got := policyLifecycleError(errors.New("private native argument")); got != "other" {
		t.Fatal("unknown error must not disclose raw text", got)
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
