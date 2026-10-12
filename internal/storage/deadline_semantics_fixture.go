//go:build horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
)

// DeadlineProvenanceErrorPreserved checks only a disposable diagnostic error.
func DeadlineProvenanceErrorPreserved(err error) bool {
	active := make(map[error]bool)
	var annotations []*deadlineAnnotation
	nodes := 0
	var visit func(error) bool
	visit = func(e error) bool {
		if e == nil {
			return true
		}
		nodes++
		v := reflect.ValueOf(e)
		if nodes > deadlineTraversalLimit || !v.Comparable() || active[e] || (v.Kind() == reflect.Pointer && v.IsNil()) {
			return false
		}
		active[e] = true
		defer delete(active, e)
		// No unknown classifier hooks are invoked by this diagnostic check.
		if _, ok := e.(interface{ Is(error) bool }); ok {
			return false
		}
		if _, ok := e.(interface{ As(any) bool }); ok {
			return false
		}
		if a, ok := e.(*deadlineAnnotation); ok {
			if a.original == nil {
				return false
			}
			annotations = append(annotations, a)
		}
		if joined, ok := e.(interface{ Unwrap() []error }); ok {
			children := joined.Unwrap()
			if len(children) == 0 || len(children) > deadlineFrameLimit {
				return false
			}
			for _, child := range children {
				if child == nil || !visit(child) {
					return false
				}
			}
		} else if wrapped, ok := e.(interface{ Unwrap() error }); ok {
			child := wrapped.Unwrap()
			if child == nil || !visit(child) {
				return false
			}
		}
		return true
	}
	if !visit(err) {
		return false
	}
	// Validate the graph before calling Error/Is/As. Private annotation types
	// themselves are excluded from original typed-error targets.
	for _, annotation := range annotations {
		original := annotation.original
		for {
			inner, ok := original.(*deadlineAnnotation)
			if !ok {
				break
			}
			original = inner.original
		}
		target := reflect.New(reflect.TypeOf(original))
		if annotation.Error() != annotation.original.Error() || !errors.Is(annotation, original) || !errors.As(annotation, target.Interface()) || target.Elem().Interface() != original {
			return false
		}
	}
	return true
}
