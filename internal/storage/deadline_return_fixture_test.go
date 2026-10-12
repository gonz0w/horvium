//go:build horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// Deterministic context observations; no sleeping or replacement timers.
type observedDeadlineContext struct {
	context.Context
	deadline time.Time
	present  bool
	err      error
	next     error
	change   bool
	reads    int
}

func (c *observedDeadlineContext) Deadline() (time.Time, bool) { return c.deadline, c.present }
func (c *observedDeadlineContext) Err() error {
	c.reads++
	if c.change && c.reads > 1 {
		return c.next
	}
	return c.err
}

type originalDeadlineError struct{ error }
type cyclicDeadlineError struct{}

func (e *cyclicDeadlineError) Error() string { return "private raw error" }
func (e *cyclicDeadlineError) Unwrap() error { return e }

func deadlineContexts() (time.Time, context.Context, *observedDeadlineContext, *observedDeadlineContext) {
	start := time.Now().Add(-time.Second)
	base := EnableDeadlineProvenance(context.Background(), start)
	parent := &observedDeadlineContext{Context: base, deadline: start.Add(180 * time.Second), present: true}
	child := &observedDeadlineContext{Context: base, deadline: start.Add(90 * time.Second), present: true, err: context.DeadlineExceeded}
	return start, base, parent, child
}
func annotateDeadline(c, p context.Context, err error) error {
	return AnnotateDeadlineReturn(c, p, err, DeadlineReview, DeadlineInspect, 90*time.Second)
}
func oneDeadlineFrame(t *testing.T, err error) deadlineFrame {
	t.Helper()
	got := ExtractDeadlineProvenance(err)
	if got.Status != "complete" || len(got.Frames) != 1 {
		t.Fatalf("missing complete observation: %+v", got)
	}
	return got.Frames[0]
}
func TestDeadlineProvenanceEnablementAndSemantics(t *testing.T) {
	_, base, parent, child := deadlineContexts()
	original := &originalDeadlineError{fmt.Errorf("private argument: %w", context.DeadlineExceeded)}
	if got := annotateDeadline(context.Background(), context.Background(), original); got != original {
		t.Fatal("disabled identity changed")
	}
	for _, err := range []error{nil, context.Canceled, errors.New("nondeadline")} {
		if got := annotateDeadline(child, parent, err); got != err {
			t.Fatal("nondeadline identity changed")
		}
	}
	got := annotateDeadline(child, parent, original)
	if got == original || got.Error() != original.Error() || !errors.Is(got, context.DeadlineExceeded) {
		t.Fatal("annotation missing or underlying semantics changed")
	}
	var found *originalDeadlineError
	if !errors.As(got, &found) || found != original {
		t.Fatal("original typed error lost")
	}
	if child.Context != base || child.reads != 2 || parent.reads != 2 {
		t.Fatal("original contexts not sampled twice")
	}
	encoded := EncodeDeadlineProvenance(got)
	if bytes.Contains(encoded, []byte("private")) || !json.Valid(encoded) || len(encoded) > 8192 {
		t.Fatalf("unsafe scalar encoding: %s", encoded)
	}
}
func TestDeadlineProvenanceOriginalContextSnapshots(t *testing.T) {
	cases := []struct {
		name, constraint string
		mutate           func(*observedDeadlineContext, *observedDeadlineContext)
	}{
		{"local", "local-expiry", func(p, c *observedDeadlineContext) {}},
		{"inherited", "inherited", func(p, c *observedDeadlineContext) { c.deadline = p.deadline }},
		{"both", "both-expired", func(p, c *observedDeadlineContext) { p.err = context.DeadlineExceeded }},
		{"canceled", "canceled", func(p, c *observedDeadlineContext) { p.err = context.Canceled }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, _, parent, child := deadlineContexts()
			tc.mutate(parent, child)
			frame := oneDeadlineFrame(t, annotateDeadline(child, parent, context.DeadlineExceeded))
			if frame.Constraint != tc.constraint || frame.ChildDeadlineNS != child.deadline.Sub(start).Nanoseconds() || frame.ParentDeadlineNS != parent.deadline.Sub(start).Nanoseconds() || frame.BeginNS > frame.EndNS || frame.AllowanceNS != int64(90*time.Second) {
				t.Fatalf("wrong original observations: %+v", frame)
			}
		})
	}
	t.Run("parent-only", func(t *testing.T) {
		_, base, parent, _ := deadlineContexts()
		parent.err = context.DeadlineExceeded
		got := AnnotateDeadlineReturn(nil, parent, context.DeadlineExceeded, DeadlineReview, DeadlineGuard, 0)
		frame := oneDeadlineFrame(t, got)
		if frame.ChildPresent || frame.ChildDeadline || frame.AllowanceNS != 0 || frame.Constraint != "parent-only" || parent.Context != base {
			t.Fatal("invented guard child")
		}
	})
	t.Run("overlap", func(t *testing.T) {
		_, _, parent, child := deadlineContexts()
		parent.change = true
		parent.next = context.DeadlineExceeded
		got := ExtractDeadlineProvenance(annotateDeadline(child, parent, context.DeadlineExceeded))
		if got.Status != "ambiguous" || len(got.Frames) != 1 || got.Frames[0].Constraint != "overlap" {
			t.Fatalf("overlap falsely atomic: %+v", got)
		}
	})
	t.Run("immutable-inner", func(t *testing.T) {
		_, base, parent, child := deadlineContexts()
		inner := annotateDeadline(child, parent, context.DeadlineExceeded)
		before := oneDeadlineFrame(t, inner)
		child.err = context.Canceled
		parent.err = context.Canceled
		outer := AnnotateDeadlineReturn(base, context.Background(), fmt.Errorf("outer: %w", inner), DeadlineActivation, DeadlineFinalInspection, 90*time.Second)
		got := ExtractDeadlineProvenance(outer)
		if len(got.Frames) != 2 || !reflect.DeepEqual(got.Frames[0], before) || child.reads != 2 || parent.reads != 2 {
			t.Fatalf("inner resampled after cancellation: %+v", got)
		}
	})
}
func TestDeadlineProvenanceBoundsAndBranches(t *testing.T) {
	_, _, parent, child := deadlineContexts()
	t.Run("success", func(t *testing.T) {
		got := ExtractDeadlineProvenance(nil)
		if got.Status != "success" || got.Frames == nil || len(got.Frames) != 0 {
			t.Fatalf("missing empty success: %+v", got)
		}
	})
	t.Run("unattributed", func(t *testing.T) {
		if got := ExtractDeadlineProvenance(context.DeadlineExceeded); got.Status != "unattributed" {
			t.Fatal(got)
		}
	})
	t.Run("frame-cap", func(t *testing.T) {
		var err error = context.DeadlineExceeded
		for range 12 {
			err = annotateDeadline(child, parent, err)
		}
		got := ExtractDeadlineProvenance(err)
		if got.Status != "complete" || len(got.Frames) != 12 || len(EncodeDeadlineProvenance(err)) > 8192 {
			t.Fatalf("wrong boundary: %+v", got)
		}
		err = annotateDeadline(child, parent, err)
		got = ExtractDeadlineProvenance(err)
		if got.Status != "overflow" || len(got.Frames) > 12 || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("cap not retained: %+v", got)
		}
		for range 100 {
			if next := annotateDeadline(child, parent, err); next != err {
				t.Fatal("overflow chain grew")
			}
		}
	})
	t.Run("join", func(t *testing.T) {
		a := annotateDeadline(child, parent, context.DeadlineExceeded)
		b := annotateDeadline(child, parent, fmt.Errorf("other: %w", context.DeadlineExceeded))
		got := ExtractDeadlineProvenance(errors.Join(a, b))
		if got.Status != "ambiguous" || len(got.Frames) != 2 || got.Frames[0].Branch == got.Frames[1].Branch {
			t.Fatalf("competing paths flattened: %+v", got)
		}
	})
	t.Run("unknown-join", func(t *testing.T) {
		got := ExtractDeadlineProvenance(errors.Join(annotateDeadline(child, parent, context.DeadlineExceeded), context.DeadlineExceeded))
		if got.Status != "ambiguous" {
			t.Fatal(got)
		}
	})
	t.Run("cycle", func(t *testing.T) {
		got := ExtractDeadlineProvenance(&cyclicDeadlineError{})
		if got.Status != "invalid" {
			t.Fatal(got)
		}
	})
	t.Run("depth", func(t *testing.T) {
		var err error = context.DeadlineExceeded
		for range 60 {
			err = fmt.Errorf("%w", err)
		}
		got := ExtractDeadlineProvenance(err)
		if got.Status != "invalid" {
			t.Fatal(got)
		}
	})
	t.Run("unknown-site", func(t *testing.T) {
		got := ExtractDeadlineProvenance(AnnotateDeadlineReturn(child, parent, context.DeadlineExceeded, 255, DeadlineInspect, time.Second))
		if got.Status != "invalid" {
			t.Fatal(got)
		}
	})
	t.Run("nondeadline", func(t *testing.T) {
		got := ExtractDeadlineProvenance(context.Canceled)
		if got.Status != "not-deadline" {
			t.Fatal(got)
		}
	})
}
