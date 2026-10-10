// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestClosedExecutionJournalSuccessor(t *testing.T) {
	before := executionJournalFixture()
	before.Generation = 2
	before.Owner.State = "closed"
	owner := strings.Repeat("b", 32)
	boot := [16]byte{0, 1, 2, 3}
	next, e := closedExecutionJournalSuccessor(before, owner, boot)
	expected := before
	expected.Generation++
	expected.Owner = executionJournalOwner{ID: owner, Boot: hex.EncodeToString(boot[:]), State: "active"}
	if e != nil || next != expected {
		t.Fatal("exact fresh active candidate", e)
	}
	if checkExecutionJournalSuccessor(before, next) == nil {
		t.Fatal("ordinary reopen granted restart")
	}
	if e = checkExecutionJournalSuccessor(next, func() executionJournal { v := next; v.Generation++; v.Owner.State = "closed"; return v }()); e != nil {
		t.Fatal("ordinary active successor changed", e)
	}
	for _, mode := range []string{"active", "overflow", "no-close-capacity", "bad-record", "old-owner", "installation-owner", "invalid-owner", "short-owner", "disarmed", "native-pending", "metadata-pending"} {
		t.Run(mode, func(t *testing.T) {
			b, id := before, owner
			switch mode {
			case "active":
				b.Owner.State = "active"
			case "overflow":
				b.Generation = math.MaxUint64
			case "no-close-capacity":
				b.Generation = math.MaxUint64 - 1
			case "bad-record":
				b.Version++
			case "old-owner":
				id = b.Owner.ID
			case "installation-owner":
				id = b.Binding.Installation
			case "invalid-owner":
				id = strings.Repeat("X", 32)
			case "short-owner":
				id = "a"
			case "disarmed":
				b.Disarmed = true
			case "native-pending":
				b.Native = journalNativeFixture(b)
			case "metadata-pending":
				b.Metadata = journalMetadataFixture(b)
			}
			old := b
			got, e := closedExecutionJournalSuccessor(b, id, boot)
			if e == nil || got != (executionJournal{}) || b != old {
				t.Fatal("unsafe candidate accepted")
			}
		})
	}
	before.Generation = math.MaxUint64 - 2
	if v, e := closedExecutionJournalSuccessor(before, owner, boot); e != nil || v.Generation != math.MaxUint64-1 {
		t.Fatal("bounded final restart", e)
	}
}
func restartActivationFixture(read func(context.Context) (ExecutionDomainsObservation, error), release func() error) *ExecutionRestartActivation {
	a := &ExecutionRestartActivation{observe: read, release: release}
	a.original = a
	return a
}
func TestExecutionRestartTransfer(t *testing.T) {
	ctx := context.Background()
	attempts, releases := 0, 0
	a := restartActivationFixture(func(context.Context) (ExecutionDomainsObservation, error) { return ExecutionDomainsObservation{}, nil }, func() error { releases++; return nil })
	r := startupReviewFixture(func(context.Context) (ExecutionDomainsObservation, error) { return ExecutionDomainsObservation{}, nil }, func() error { t.Fatal("transferred review released original leases"); return nil })
	r.activate = func(context.Context) (*ExecutionRestartActivation, error) { attempts++; return a, nil }
	copied := new(ExecutionStartupReview)
	reflect.ValueOf(copied).Elem().Set(reflect.ValueOf(r).Elem())
	data, e := json.Marshal(r)
	var decoded ExecutionStartupReview
	if e != nil || string(data) != "{}" || json.Unmarshal(data, &decoded) != nil {
		t.Fatal("encoded authority")
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	for _, q := range []*ExecutionStartupReview{nil, {}, copied, &decoded} {
		if got, e := q.Activate(ctx); e == nil || got != nil {
			t.Fatal("foreign review")
		}
	}
	for _, c := range []context.Context{nil, cancelled} {
		if got, e := r.Activate(c); e == nil || got != nil {
			t.Fatal("invalid context")
		}
	}
	if attempts != 0 {
		t.Fatal("pre-entry consumed")
	}
	got, e := r.Activate(ctx)
	if e != nil || got != a || attempts != 1 {
		t.Fatal("original transfer", e)
	}
	if _, e = r.Activate(ctx); e == nil {
		t.Fatal("repeated transfer")
	}
	if _, e = r.Observe(ctx); e == nil {
		t.Fatal("old review remained live")
	}
	if r.Close() == nil || releases != 0 {
		t.Fatal("old review revoked new activation")
	}
	if _, e = a.Observe(ctx); e != nil {
		t.Fatal("original activation", e)
	}
	if a.Close() != nil || a.Close() != nil || releases != 1 {
		t.Fatal("release once")
	}
	if _, e = a.Observe(ctx); e == nil {
		t.Fatal("closed activation")
	}
}
func TestExecutionRestartFailures(t *testing.T) {
	for _, mode := range []string{"factory", "partial", "late-cancel", "nil-success", "unissued-success", "read", "release"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			attempts, releases, oldReleases := 0, 0, 0
			a := restartActivationFixture(func(context.Context) (ExecutionDomainsObservation, error) {
				if mode == "read" {
					return ExecutionDomainsObservation{}, ErrDrift
				}
				return ExecutionDomainsObservation{}, nil
			}, func() error {
				releases++
				if mode == "release" {
					return ErrDenied
				}
				return nil
			})
			r := startupReviewFixture(func(context.Context) (ExecutionDomainsObservation, error) { return ExecutionDomainsObservation{}, nil }, func() error { oldReleases++; return nil })
			r.activate = func(context.Context) (*ExecutionRestartActivation, error) {
				attempts++
				switch mode {
				case "factory":
					return nil, ErrDenied
				case "partial":
					return a, ErrDrift
				case "late-cancel":
					cancel()
					return a, nil
				case "nil-success":
					return nil, nil
				case "unissued-success":
					return &ExecutionRestartActivation{}, nil
				}
				return a, nil
			}
			got, e := r.Activate(ctx)
			if mode == "read" || mode == "release" {
				if e != nil || got != a {
					t.Fatal("original", e)
				}
				if mode == "read" {
					if _, e = a.Observe(context.Background()); e == nil {
						t.Fatal("read error")
					}
					if _, e = a.Observe(context.Background()); e == nil {
						t.Fatal("retired read")
					}
				}
				e = a.Close()
				if (mode == "release") != errors.Is(e, ErrDenied) || releases != 1 || a.Close() != e {
					t.Fatal("release retained")
				}
			} else {
				if e == nil || got != nil {
					t.Fatal("uncertain transfer")
				}
				if _, e = r.Activate(context.Background()); e == nil || attempts != 1 {
					t.Fatal("retry")
				}
				if r.Close() != nil || oldReleases != 1 {
					t.Fatal("failed original release")
				}
				if (mode == "partial" || mode == "late-cancel") && releases != 1 {
					t.Fatal("partial leaked")
				}
			}
		})
	}
}
func TestExecutionRestartOriginalLifecycle(t *testing.T) {
	entered, releaseRead := make(chan struct{}), make(chan struct{})
	releases := 0
	a := restartActivationFixture(func(context.Context) (ExecutionDomainsObservation, error) {
		close(entered)
		<-releaseRead
		return ExecutionDomainsObservation{}, nil
	}, func() error { releases++; return nil })
	copied := new(ExecutionRestartActivation)
	reflect.ValueOf(copied).Elem().Set(reflect.ValueOf(a).Elem())
	data, e := json.Marshal(a)
	var decoded ExecutionRestartActivation
	if e != nil || string(data) != "{}" || json.Unmarshal(data, &decoded) != nil {
		t.Fatal("wire")
	}
	for _, q := range []*ExecutionRestartActivation{nil, {}, copied, &decoded} {
		if _, e = q.Observe(context.Background()); e == nil || q.Close() == nil {
			t.Fatal("foreign activation")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, e := a.Observe(ctx); done <- e }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("no observation")
	}
	closed := make(chan error, 1)
	go func() { closed <- a.Close() }()
	select {
	case <-closed:
		t.Fatal("release overlapped read")
	default:
	}
	close(releaseRead)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	if e := <-closed; e != nil || releases != 1 {
		t.Fatal("release")
	}
}
