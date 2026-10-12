//go:build !horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

type forbiddenDeadlineContext struct{}

func (forbiddenDeadlineContext) Deadline() (time.Time, bool) { panic("deadline inspection") }
func (forbiddenDeadlineContext) Done() <-chan struct{}       { panic("done inspection") }
func (forbiddenDeadlineContext) Err() error                  { panic("error inspection") }
func (forbiddenDeadlineContext) Value(any) any               { panic("value inspection") }

func TestDeadlineReturnOrdinaryIdentity(t *testing.T) {
	c := forbiddenDeadlineContext{}
	for _, err := range []error{nil, context.DeadlineExceeded, context.Canceled, errors.New("original"), errors.Join(context.DeadlineExceeded, errors.New("other"))} {
		if got := AnnotateDeadlineReturn(c, c, err, DeadlineReview, DeadlineInspect, 90*time.Second); got != err {
			t.Fatal("ordinary adapter changed error identity")
		}
	}
	if allocations := testing.AllocsPerRun(100, func() {
		_ = AnnotateDeadlineReturn(c, c, context.DeadlineExceeded, DeadlineReview, DeadlineInspect, time.Second)
	}); allocations != 0 {
		t.Fatalf("ordinary adapter allocated: %v", allocations)
	}
}
