//go:build horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestDeadlineProvenanceErrorPreserved(t *testing.T) {
	_, _, parent, child := deadlineContexts()
	original := &originalDeadlineError{fmt.Errorf("private original: %w", context.DeadlineExceeded)}
	inner := annotateDeadline(child, parent, original)
	outer := annotateDeadline(child, parent, fmt.Errorf("outer: %w", inner))
	before := EncodeDeadlineProvenance(outer)
	reads := child.reads + parent.reads
	for _, err := range []error{nil, original, inner, outer, context.Canceled, errors.New("private denial"), errors.Join(inner, errors.New("other branch"))} {
		if !DeadlineProvenanceErrorPreserved(err) {
			t.Fatal("actual original Error/Is/As semantics lost")
		}
	}
	var typed *originalDeadlineError
	if !errors.Is(outer, context.DeadlineExceeded) || !errors.As(outer, &typed) || typed != original || !bytes.Equal(before, EncodeDeadlineProvenance(outer)) || child.reads+parent.reads != reads {
		t.Fatal("error verification changed trace or read original contexts")
	}
}

func TestDeadlineProvenanceErrorPreservedRejectsUnsafeGraphs(t *testing.T) {
	var deep error = context.DeadlineExceeded
	for range 49 {
		deep = fmt.Errorf("bounded fixture wrapper: %w", deep)
	}
	for _, err := range []error{&cyclicDeadlineError{}, &deadlineAnnotation{}, deep, errors.Join(make([]error, 13)...)} {
		// errors.Join of nil entries returns nil; test a nonempty oversized join below.
		if err != nil && DeadlineProvenanceErrorPreserved(err) {
			t.Fatal("unsafe graph accepted")
		}
	}
	children := make([]error, 13)
	for i := range children {
		children[i] = context.DeadlineExceeded
	}
	if DeadlineProvenanceErrorPreserved(errors.Join(children...)) {
		t.Fatal("oversized join accepted")
	}
}
