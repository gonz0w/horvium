//go:build horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"time"
)

type deadlineFrame struct {
	Site             string `json:"site"`
	Phase            string `json:"phase"`
	Branch           string `json:"branch"`
	AllowanceNS      int64  `json:"allowance_ns"`
	ChildPresent     bool   `json:"child_present"`
	ChildDeadline    bool   `json:"child_deadline"`
	ChildDeadlineNS  int64  `json:"child_deadline_ns"`
	ParentDeadline   bool   `json:"parent_deadline"`
	ParentDeadlineNS int64  `json:"parent_deadline_ns"`
	ChildBefore      string `json:"child_before"`
	ChildAfter       string `json:"child_after"`
	ParentBefore     string `json:"parent_before"`
	ParentAfter      string `json:"parent_after"`
	BeginNS          int64  `json:"begin_ns"`
	EndNS            int64  `json:"end_ns"`
	Constraint       string `json:"constraint"`
}

type DeadlineProvenance struct {
	Status string          `json:"status"`
	Frames []deadlineFrame `json:"frames"`
}

func EnableDeadlineProvenance(ctx context.Context, _ time.Time) context.Context { return ctx }
func AnnotateDeadlineReturn(_ context.Context, _ context.Context, err error, _ DeadlineSite, _ DeadlinePhase, _ time.Duration) error {
	return err
}
func ExtractDeadlineProvenance(_ error) DeadlineProvenance {
	return DeadlineProvenance{Status: "unattributed", Frames: []deadlineFrame{}}
}
func EncodeDeadlineProvenance(_ error) []byte { return []byte(`{"status":"unattributed","frames":[]}`) }
