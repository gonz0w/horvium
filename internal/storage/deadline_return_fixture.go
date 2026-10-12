//go:build horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"time"
)

const deadlineFrameLimit = 12
const deadlineByteLimit = 8192
const deadlineTraversalLimit = 48

type deadlineFixtureKey struct{}
type deadlineFixtureStart struct{ start time.Time }

type deadlineFrame struct {
	Site             string `json:"site"`
	Operation        string `json:"operation"`
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

// DeadlineProvenance exists only in disposable tagged builds. Status describes
// diagnostic attribution, never native acceptance or authority to retry.
type DeadlineProvenance struct {
	Status string          `json:"status"`
	Frames []deadlineFrame `json:"frames"`
}

type deadlineAnnotation struct {
	original error
	frame    deadlineFrame
	terminal string
}

func (e *deadlineAnnotation) Error() string { return e.original.Error() }
func (e *deadlineAnnotation) Unwrap() error { return e.original }

// EnableDeadlineProvenance carries only a fixture's original monotonic origin.
// It does not create a deadline, timer, callback, cancellation or collector.
func EnableDeadlineProvenance(ctx context.Context, start time.Time) context.Context {
	if ctx == nil || start.IsZero() || start == start.Round(0) {
		return ctx
	}
	return context.WithValue(ctx, deadlineFixtureKey{}, deadlineFixtureStart{start})
}

// AnnotateDeadlineReturn observes actual original contexts before their scope
// cancels them. No product return sites call this adapter in the portable slice.
func AnnotateDeadlineReturn(child, parent context.Context, err error, site DeadlineSite, phase DeadlinePhase, allowance time.Duration) error {
	if err == nil {
		return err
	}
	enabled := child
	if enabled == nil {
		enabled = parent
	}
	if enabled == nil {
		return err
	}
	origin, ok := enabled.Value(deadlineFixtureKey{}).(deadlineFixtureStart)
	if !ok || origin.start.IsZero() {
		return err
	}
	walk := inspectDeadlineError(err)
	// A terminal annotation propagates identically: never grow a cap/error chain.
	if walk.terminal || walk.deadlines == 0 {
		return err
	}
	if walk.invalid {
		return &deadlineAnnotation{original: err, terminal: "invalid"}
	}
	if len(walk.frames) >= deadlineFrameLimit {
		return &deadlineAnnotation{original: err, terminal: "overflow"}
	}
	siteName, phaseName, valid := deadlineIdentifiers(site, phase, allowance, child != nil)
	if !valid || parent == nil {
		return &deadlineAnnotation{original: err, terminal: "invalid"}
	}
	frame := deadlineFrame{Site: siteName, Operation: "deadline", Phase: phaseName, AllowanceNS: int64(allowance), ChildPresent: child != nil}
	frame.BeginNS = time.Since(origin.start).Nanoseconds()
	if child != nil {
		frame.ChildBefore = deadlineContextClass(child.Err())
	} else {
		frame.ChildBefore = "absent"
	}
	frame.ParentBefore = deadlineContextClass(parent.Err())
	if child != nil {
		deadline, present := child.Deadline()
		frame.ChildDeadline = present
		if present {
			frame.ChildDeadlineNS = deadline.Sub(origin.start).Nanoseconds()
		}
	}
	deadline, present := parent.Deadline()
	frame.ParentDeadline = present
	if present {
		frame.ParentDeadlineNS = deadline.Sub(origin.start).Nanoseconds()
	}
	if child != nil {
		frame.ChildAfter = deadlineContextClass(child.Err())
	} else {
		frame.ChildAfter = "absent"
	}
	frame.ParentAfter = deadlineContextClass(parent.Err())
	frame.EndNS = time.Since(origin.start).Nanoseconds()
	frame.Constraint = deadlineConstraint(frame)
	next := &deadlineAnnotation{original: err, frame: frame}
	// Extraction checks the same frame/node/encoded bounds as construction.
	trace := ExtractDeadlineProvenance(next)
	if trace.Status == "overflow" || trace.Status == "invalid" {
		return &deadlineAnnotation{original: err, terminal: trace.Status}
	}
	return next
}

func deadlineIdentifiers(site DeadlineSite, phase DeadlinePhase, allowance time.Duration, child bool) (string, string, bool) {
	sites := [...]string{"", "review", "activation", "backing-collector", "backing-command", "pool-collector", "pool-command", "provider-collector", "system-collector", "system-command"}
	phases := [...]string{"", "guard", "inspect", "first-inspection", "second-inspection", "publication", "final-inspection", "collection", "capture"}
	if int(site) >= len(sites) || int(phase) >= len(phases) || site == 0 || phase == 0 {
		return "", "", false
	}
	if !child {
		return sites[site], phases[phase], phase == DeadlineGuard && allowance == 0
	}
	expected := 45 * time.Second
	validPhase := phase == DeadlineCollection
	switch site {
	case DeadlineReview:
		expected, validPhase = 90*time.Second, phase == DeadlineInspect
	case DeadlineActivation:
		expected = 90 * time.Second
		validPhase = phase >= DeadlineFirstInspection && phase <= DeadlineFinalInspection
	case DeadlineBackingCommand, DeadlinePoolCommand, DeadlineSystemCommand:
		expected, validPhase = 10*time.Second, phase == DeadlineCapture
	}
	return sites[site], phases[phase], validPhase && allowance == expected
}

func deadlineContextClass(err error) string {
	switch err {
	case nil:
		return "none"
	case context.DeadlineExceeded:
		return "deadline"
	case context.Canceled:
		return "canceled"
	default:
		return "other"
	}
}

func deadlineConstraint(f deadlineFrame) string {
	if f.ChildBefore != f.ChildAfter || f.ParentBefore != f.ParentAfter {
		return "overlap"
	}
	if !f.ChildPresent {
		return "parent-only"
	}
	if f.ChildAfter == "canceled" || f.ParentAfter == "canceled" {
		return "canceled"
	}
	if f.ChildAfter == "deadline" && f.ParentAfter == "deadline" {
		return "both-expired"
	}
	if f.ChildDeadline && f.ParentDeadline && f.ChildDeadlineNS == f.ParentDeadlineNS {
		return "inherited"
	}
	if f.ChildAfter == "deadline" && f.ParentAfter == "none" && f.ChildDeadline && (!f.ParentDeadline || f.ChildDeadlineNS < f.ParentDeadlineNS) {
		return "local-expiry"
	}
	if f.ParentAfter == "deadline" {
		return "parent-expired"
	}
	return "unresolved"
}

type deadlineWalk struct {
	frames                               []deadlineFrame
	nodes, deadlines, otherLeaves        int
	invalid, overflow, terminal, overlap bool
	active                               map[error]bool
	annotations                          map[*deadlineAnnotation]bool
}

func inspectDeadlineError(err error) deadlineWalk {
	w := deadlineWalk{frames: []deadlineFrame{}, active: make(map[error]bool), annotations: make(map[*deadlineAnnotation]bool)}
	w.visit(err, "0")
	return w
}

func (w *deadlineWalk) visit(err error, branch string) {
	if err == nil || w.invalid || w.overflow {
		return
	}
	w.nodes++
	value := reflect.ValueOf(err)
	if w.nodes > deadlineTraversalLimit || !value.Comparable() || w.active[err] || (value.Kind() == reflect.Pointer && value.IsNil()) {
		w.invalid = true
		return
	}
	w.active[err] = true
	defer delete(w.active, err)
	if err == context.DeadlineExceeded {
		w.deadlines++
		return
	}
	if annotation, ok := err.(*deadlineAnnotation); ok {
		if annotation == nil || annotation.original == nil || w.annotations[annotation] {
			w.invalid = true
			return
		}
		w.annotations[annotation] = true
		if annotation.terminal != "" {
			w.terminal = true
			if annotation.terminal == "overflow" {
				w.overflow = true
			} else {
				w.invalid = true
			}
			return
		}
		w.visit(annotation.original, branch)
		if w.invalid || w.overflow {
			return
		}
		if len(w.frames) >= deadlineFrameLimit {
			w.overflow = true
			return
		}
		f := annotation.frame
		if !validDeadlineFrame(f) {
			w.invalid = true
			return
		}
		f.Branch = branch
		w.frames = append(w.frames, f)
		w.overlap = w.overlap || f.Constraint == "overlap"
		return
	}
	if join, ok := err.(interface{ Unwrap() []error }); ok {
		children := join.Unwrap()
		if len(children) == 0 || len(children) > deadlineFrameLimit {
			w.invalid = true
			return
		}
		for i, child := range children {
			if child == nil {
				w.invalid = true
				return
			}
			path := branch
			if len(children) > 1 {
				path += "." + strconv.Itoa(i)
			}
			w.visit(child, path)
		}
		return
	}
	if wrap, ok := err.(interface{ Unwrap() error }); ok {
		child := wrap.Unwrap()
		if child == nil {
			w.invalid = true
			return
		}
		w.visit(child, branch)
		return
	}
	if _, unknown := err.(interface{ Is(error) bool }); unknown {
		w.invalid = true
		return
	}
	w.otherLeaves++
}

func validDeadlineFrame(f deadlineFrame) bool {
	// All strings originate in fixed identifier/class tables. Recheck on
	// extraction rather than trusting malformed in-package fixture records.
	var site DeadlineSite
	for s := DeadlineReview; s <= DeadlineSystemCommand; s++ {
		name, _, _ := deadlineIdentifiers(s, DeadlineGuard, 0, false)
		if name == f.Site {
			site = s
			break
		}
	}
	var phase DeadlinePhase
	for p := DeadlineGuard; p <= DeadlineCapture; p++ {
		_, name, _ := deadlineIdentifiers(DeadlineReview, p, 0, false)
		if name == f.Phase {
			phase = p
			break
		}
	}
	_, _, valid := deadlineIdentifiers(site, phase, time.Duration(f.AllowanceNS), f.ChildPresent)
	if !valid || f.Operation != "deadline" || f.BeginNS < 0 || f.EndNS < f.BeginNS || f.Constraint != deadlineConstraint(f) {
		return false
	}
	class := func(s string) bool { return s == "none" || s == "deadline" || s == "canceled" || s == "other" }
	if !class(f.ParentBefore) || !class(f.ParentAfter) {
		return false
	}
	if f.ChildPresent {
		if !f.ChildDeadline {
			return false
		}
		if (f.ChildBefore == "deadline" || f.ChildAfter == "deadline") && f.ChildDeadlineNS > f.EndNS {
			return false
		}
		if !class(f.ChildBefore) || !class(f.ChildAfter) {
			return false
		}
	} else if f.ChildBefore != "absent" || f.ChildAfter != "absent" || f.ChildDeadline || f.ChildDeadlineNS != 0 {
		return false
	}
	if (f.ParentBefore == "deadline" || f.ParentAfter == "deadline") && (!f.ParentDeadline || f.ParentDeadlineNS > f.EndNS) {
		return false
	}
	return (f.ChildDeadline || f.ChildDeadlineNS == 0) && (f.ParentDeadline || f.ParentDeadlineNS == 0)
}

// ExtractDeadlineProvenance traverses original wrapping with strict node and
// frame caps. It never calls Error or reconstructs canceled inner contexts.
func ExtractDeadlineProvenance(err error) DeadlineProvenance {
	if err == nil {
		return DeadlineProvenance{Status: "success", Frames: []deadlineFrame{}}
	}
	w := inspectDeadlineError(err)
	status := "complete"
	switch {
	case w.invalid:
		status = "invalid"
	case w.overflow:
		status = "overflow"
	case w.deadlines == 0:
		status = "not-deadline"
	case w.deadlines > 1 || w.overlap:
		status = "ambiguous"
	case w.otherLeaves > 0:
		status = "invalid"
	case len(w.frames) == 0:
		status = "unattributed"
	}
	trace := DeadlineProvenance{Status: status, Frames: w.frames}
	encoded, e := json.Marshal(trace)
	if e != nil || len(encoded) > deadlineByteLimit {
		return DeadlineProvenance{Status: "overflow", Frames: []deadlineFrame{}}
	}
	return trace
}

// EncodeDeadlineProvenance emits only bounded fixed scalars. Raw errors,
// context values, command arguments and absolute clock values never enter JSON.
func EncodeDeadlineProvenance(err error) []byte {
	encoded, _ := json.Marshal(ExtractDeadlineProvenance(err))
	return encoded
}
