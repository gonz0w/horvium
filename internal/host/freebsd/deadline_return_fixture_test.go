//go:build horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

func TestDeadlineReturnOriginalCollectorContexts(t *testing.T) {
	for _, name := range []string{"backing-collector", "pool-collector", "system-collector", "provider-collector"} {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			caller, cancel := context.WithTimeout(storage.EnableDeadlineProvenance(context.Background(), start), 180*time.Second)
			defer cancel()
			callerDeadline, _ := caller.Deadline()
			original := &deadlineReaderError{context.DeadlineExceeded}
			calls := 0
			var actualChild context.Context
			var actualDeadline time.Time
			cases := deadlineCollectorCases(t, func(c context.Context) error {
				calls++
				actualChild = c
				actualDeadline, _ = c.Deadline()
				return original
			})
			for _, tc := range cases {
				if tc.name == name {
					before := time.Since(start).Nanoseconds()
					got := tc.collect(caller)
					after := time.Since(start).Nanoseconds()
					trace := storage.ExtractDeadlineProvenance(got)
					if trace.Status != "complete" || len(trace.Frames) != 2 {
						t.Fatalf("missing return-site provenance: %+v", trace)
					}
					command, collector := trace.Frames[0], trace.Frames[1]
					if command.Site != tc.command || command.Phase != "capture" || collector.Site != tc.name || collector.Phase != "collection" || command.AllowanceNS != int64(10*time.Second) || collector.AllowanceNS != int64(45*time.Second) {
						t.Fatalf("wrong original scope: %+v", trace)
					}
					if command.ChildDeadlineNS != actualDeadline.Sub(start).Nanoseconds() || command.ParentDeadlineNS != collector.ChildDeadlineNS || collector.ParentDeadlineNS != callerDeadline.Sub(start).Nanoseconds() {
						t.Fatal("replacement or resampled context/deadline")
					}
					if command.ChildDeadlineNS < before+int64(10*time.Second) || command.ChildDeadlineNS > after+int64(10*time.Second) || collector.ChildDeadlineNS < before+int64(45*time.Second) || collector.ChildDeadlineNS > after+int64(45*time.Second) {
						t.Fatal("original native allowance changed")
					}
					if command.ChildBefore != "none" || command.ChildAfter != "none" || command.ParentAfter != "none" || collector.ChildAfter != "none" || collector.ParentAfter != "none" || command.Constraint != "unresolved" || actualChild.Err() != context.Canceled || caller.Err() != nil || calls != 1 {
						t.Fatal("capture occurred after deferred cancellation or added a read")
					}
					var typed *deadlineReaderError
					if !errors.Is(got, context.DeadlineExceeded) || !errors.As(got, &typed) || typed != original || got.Error() != original.Error() {
						t.Fatal("original error semantics lost")
					}
					encoded := storage.EncodeDeadlineProvenance(got)
					if len(encoded) > 8192 || bytes.Contains(encoded, []byte("private")) {
						t.Fatal("unbounded or raw return evidence")
					}
					cancel()
					if after := storage.ExtractDeadlineProvenance(got); !reflect.DeepEqual(trace, after) {
						t.Fatal("caller cancellation rewrote immutable frames")
					}
				}
			}
		})
	}
}

func TestDeadlineReturnExpiredCollectorGuard(t *testing.T) {
	for _, name := range []string{"backing-collector", "pool-collector", "system-collector", "provider-collector"} {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			caller, cancel := context.WithDeadline(storage.EnableDeadlineProvenance(context.Background(), start), start.Add(-time.Second))
			defer cancel()
			calls := 0
			for _, tc := range deadlineCollectorCases(t, func(context.Context) error { calls++; return nil }) {
				if tc.name == name {
					got := tc.collect(caller)
					trace := storage.ExtractDeadlineProvenance(got)
					if trace.Status != "complete" || len(trace.Frames) != 2 {
						t.Fatalf("missing guard/collector path: %+v", trace)
					}
					guard, collector := trace.Frames[0], trace.Frames[1]
					if guard.Site != tc.command || guard.Phase != "guard" || guard.ChildPresent || guard.ChildDeadline || guard.AllowanceNS != 0 || guard.ParentAfter != "deadline" || guard.Constraint != "parent-only" || collector.Site != tc.name || collector.Constraint != "both-expired" || calls != 0 || !errors.Is(got, context.DeadlineExceeded) {
						t.Fatalf("invented child or changed expired read behavior: %+v", trace)
					}
				}
			}
		})
	}
}
