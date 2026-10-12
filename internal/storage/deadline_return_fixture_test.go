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

func (e *originalDeadlineError) Unwrap() error { return e.error }

type cyclicDeadlineError struct{}

func (e *cyclicDeadlineError) Error() string { return "private raw error" }
func (e *cyclicDeadlineError) Unwrap() error { return e }

func deadlineContexts() (time.Time, context.Context, *observedDeadlineContext, *observedDeadlineContext) {
	start := time.Now().Add(-100 * time.Second)
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
		{"inherited", "inherited", func(p, c *observedDeadlineContext) { p.deadline = c.deadline }},
		{"both", "both-expired", func(p, c *observedDeadlineContext) { p.err = context.DeadlineExceeded; p.deadline = c.deadline }},
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
		parent.deadline = time.Now().Add(-time.Second)
		got := AnnotateDeadlineReturn(nil, parent, context.DeadlineExceeded, DeadlineReview, DeadlineGuard, 0)
		frame := oneDeadlineFrame(t, got)
		if frame.ChildPresent || frame.ChildDeadline || frame.AllowanceNS != 0 || frame.Constraint != "parent-only" || parent.Context != base {
			t.Fatal("invented guard child")
		}
	})
	t.Run("overlap", func(t *testing.T) {
		_, _, parent, child := deadlineContexts()
		parent.deadline = child.deadline
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
		outerChild := &observedDeadlineContext{Context: base, deadline: child.deadline, present: true, err: context.DeadlineExceeded}
		outer := AnnotateDeadlineReturn(outerChild, parent, fmt.Errorf("outer: %w", inner), DeadlineActivation, DeadlineFinalInspection, 90*time.Second)
		got := ExtractDeadlineProvenance(outer)
		if len(got.Frames) != 2 || !reflect.DeepEqual(got.Frames[0], before) || child.reads != 2 || parent.reads != 4 {
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

// Unknown custom classification must not be treated as a proven nondeadline.
type customDeadlineClassifier struct{}

func (*customDeadlineClassifier) Error() string        { return "private classification" }
func (*customDeadlineClassifier) Is(target error) bool { return target == context.DeadlineExceeded }

type oversizedDeadlineJoin struct{}

func (*oversizedDeadlineJoin) Error() string   { return "private join" }
func (*oversizedDeadlineJoin) Unwrap() []error { return make([]error, 49) }

func TestDeadlineProvenanceMalformedEvidence(t *testing.T) {
	_, _, parent, child := deadlineContexts()
	cases := []struct {
		name string
		err  error
	}{
		{"typed-nil", (*cyclicDeadlineError)(nil)},
		{"custom-classifier", &customDeadlineClassifier{}},
		{"oversized-join", &oversizedDeadlineJoin{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractDeadlineProvenance(tc.err)
			if got.Status != "invalid" || bytes.Contains(EncodeDeadlineProvenance(tc.err), []byte("private")) {
				t.Fatalf("unknown evidence accepted: %+v", got)
			}
		})
	}
	t.Run("repeated-annotation", func(t *testing.T) {
		err := annotateDeadline(child, parent, context.DeadlineExceeded)
		if got := ExtractDeadlineProvenance(errors.Join(err, err)); got.Status != "invalid" {
			t.Fatalf("duplicate annotation accepted: %+v", got)
		}
	})
	for _, tc := range []struct {
		name      string
		site      DeadlineSite
		phase     DeadlinePhase
		allowance time.Duration
	}{
		{"wrong-allowance", DeadlineReview, DeadlineInspect, time.Second},
		{"wrong-phase", DeadlineReview, DeadlineCapture, 90 * time.Second},
		{"child-guard", DeadlineReview, DeadlineGuard, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractDeadlineProvenance(AnnotateDeadlineReturn(child, parent, context.DeadlineExceeded, tc.site, tc.phase, tc.allowance))
			if got.Status != "invalid" {
				t.Fatalf("unreviewed scope accepted: %+v", got)
			}
		})
	}
}

type disabledDeadlineContext struct{ context.Context }

func (disabledDeadlineContext) Deadline() (time.Time, bool) { panic("disabled deadline inspection") }
func (disabledDeadlineContext) Err() error                  { panic("disabled error inspection") }
func TestDeadlineProvenanceDisabledNoObservation(t *testing.T) {
	var c context.Context = disabledDeadlineContext{context.Background()}
	if got := annotateDeadline(c, c, context.DeadlineExceeded); got != context.DeadlineExceeded {
		t.Fatal("disabled identity changed")
	}
	if allocations := testing.AllocsPerRun(100, func() { _ = annotateDeadline(c, c, context.DeadlineExceeded) }); allocations != 0 {
		t.Fatalf("disabled adapter allocated: %v", allocations)
	}
	if EnableDeadlineProvenance(nil, time.Now()) != nil || EnableDeadlineProvenance(c, time.Time{}) != c || EnableDeadlineProvenance(c, time.Now().Round(0)) != c {
		t.Fatal("invalid enablement changed context")
	}
}
func TestDeadlineProvenanceRejectsMalformedFrames(t *testing.T) {
	_, _, parent, child := deadlineContexts()
	err := annotateDeadline(child, parent, context.DeadlineExceeded)
	original := err.(*deadlineAnnotation)
	for _, mutate := range []func(*deadlineFrame){
		func(f *deadlineFrame) { f.Operation = "private raw data" },
		func(f *deadlineFrame) { f.Site = "private raw data" },
		func(f *deadlineFrame) { f.ChildAfter = "private raw data" },
		func(f *deadlineFrame) { f.Constraint = "private raw data" },
		func(f *deadlineFrame) { f.EndNS = f.BeginNS - 1 },
		func(f *deadlineFrame) { f.ChildDeadline = false },
		func(f *deadlineFrame) { f.ChildDeadline = false; f.ChildDeadlineNS = 0; f.Constraint = "unresolved" },
		func(f *deadlineFrame) { f.ChildDeadlineNS = f.EndNS + 1; f.Constraint = "unresolved" },
	} {
		frame := original.frame
		mutate(&frame)
		malformed := &deadlineAnnotation{original: context.DeadlineExceeded, frame: frame}
		got := ExtractDeadlineProvenance(malformed)
		if got.Status != "invalid" || bytes.Contains(EncodeDeadlineProvenance(malformed), []byte("private")) {
			t.Fatalf("malformed frame accepted: %+v", got)
		}
	}
}

func TestDeadlineProvenanceRealCancellationRetention(t *testing.T) {
	start := time.Now()
	base := EnableDeadlineProvenance(context.Background(), start)
	parent, parentCancel := context.WithDeadline(base, start.Add(180*time.Second))
	defer parentCancel()
	child, childCancel := context.WithDeadline(parent, start.Add(-time.Second))
	defer childCancel()
	originalDeadline, _ := child.Deadline()
	if child.Err() != context.DeadlineExceeded || parent.Err() != nil {
		t.Fatal("expired child fixture did not retain live parent")
	}
	err := annotateDeadline(child, parent, child.Err())
	before := oneDeadlineFrame(t, err)
	childCancel()
	parentCancel()
	if after := oneDeadlineFrame(t, err); !reflect.DeepEqual(before, after) || after.ParentAfter != "none" || after.ChildAfter != "deadline" {
		t.Fatalf("deferred cancellation rewrote snapshot: %+v", after)
	}
	if after, _ := child.Deadline(); after != originalDeadline || before.ChildDeadlineNS != originalDeadline.Sub(start).Nanoseconds() {
		t.Fatal("original effective deadline changed")
	}
}
