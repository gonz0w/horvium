//go:build horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

type afterFirstDeadlineContext struct {
	context.Context
	deadline time.Time
	reads    int
}

func (c *afterFirstDeadlineContext) Deadline() (time.Time, bool) { return c.deadline, true }
func (c *afterFirstDeadlineContext) Err() error {
	c.reads++
	if c.reads == 1 {
		return nil
	}
	return context.DeadlineExceeded
}

func TestDeadlineReturnActivationParentOnlyGuards(t *testing.T) {
	for _, locked := range []bool{false, true} {
		t.Run(map[bool]string{false: "entry", true: "locked"}[locked], func(t *testing.T) {
			start := time.Now()
			base := EnableDeadlineProvenance(context.Background(), start)
			caller, cancel := context.WithDeadline(base, start.Add(-time.Second))
			defer cancel()
			var ctx context.Context = caller
			if locked {
				ctx = &afterFirstDeadlineContext{Context: base, deadline: start.Add(-time.Second)}
			}
			calls := 0
			r := startupReviewFixture(func(context.Context) (ExecutionDomainsObservation, error) {
				t.Fatal("guard observed domains")
				return ExecutionDomainsObservation{}, nil
			}, func() error { t.Fatal("guard released leases"); return nil })
			r.activate = func(context.Context) (*ExecutionRestartActivation, error) { calls++; return nil, ErrDenied }
			owner, err := r.Activate(ctx)
			trace := ExtractDeadlineProvenance(err)
			if owner != nil || calls != 0 || r.blocked || r.transferred || !errors.Is(err, context.DeadlineExceeded) || trace.Status != "complete" || len(trace.Frames) != 1 {
				t.Fatalf("activation guard changed behavior or lost provenance: %+v", trace)
			}
			f := trace.Frames[0]
			if f.Site != "activation" || f.Phase != "guard" || f.ChildPresent || f.AllowanceNS != 0 || f.ParentAfter != "deadline" {
				t.Fatalf("invented activation child: %+v", f)
			}
		})
	}
}
