// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestUnusedRestartClosedSuccessor(t *testing.T) {
	before := executionJournalFixture()
	before.Generation = 3
	expected := before
	expected.Generation++
	expected.Owner.State = "closed"
	if next, e := unusedRestartClosedSuccessor(before); e != nil || next != expected {
		t.Fatal("exact closed successor", e)
	}
	for _, mode := range []string{"closed", "overflow", "version", "owner", "disarmed", "metadata", "native"} {
		t.Run(mode, func(t *testing.T) {
			v := before
			switch mode {
			case "closed":
				v.Owner.State = "closed"
			case "overflow":
				v.Generation = math.MaxUint64
			case "version":
				v.Version++
			case "owner":
				v.Owner.ID = "invalid"
			case "disarmed":
				v.Disarmed = true
			case "metadata":
				v.Metadata = journalMetadataFixture(v)
			case "native":
				v.Native = journalNativeFixture(v)
			}
			old := v
			if got, e := unusedRestartClosedSuccessor(v); e == nil || got != (executionJournal{}) || v != old {
				t.Fatal("unsafe close accepted")
			}
		})
	}
	before.Generation = math.MaxUint64 - 1
	if v, e := unusedRestartClosedSuccessor(before); e != nil || v.Generation != math.MaxUint64 {
		t.Fatal("reserved final close", e)
	}
}

func TestRestartRetirementOriginalLifecycle(t *testing.T) {
	ctx := context.Background()
	finishes, releases := 0, 0
	a := restartActivationFixture(func(context.Context) (ExecutionDomainsObservation, error) { return ExecutionDomainsObservation{}, nil }, func() error { releases++; return nil })
	a.finish = func(context.Context) error { finishes++; return nil }
	cancelled, stop := context.WithCancel(ctx)
	stop()
	copy := new(ExecutionRestartActivation)
	reflect.ValueOf(copy).Elem().Set(reflect.ValueOf(a).Elem())
	wire, e := json.Marshal(a)
	var decoded ExecutionRestartActivation
	if e != nil || json.Unmarshal(wire, &decoded) != nil || string(wire) != "{}" {
		t.Fatal("serialized authority")
	}
	for _, v := range []struct {
		a *ExecutionRestartActivation
		c context.Context
	}{{nil, ctx}, {a, nil}, {a, cancelled}, {copy, ctx}, {&decoded, ctx}} {
		if v.a.FinishIdle(v.c) == nil || finishes != 0 {
			t.Fatal("unissued/context finish")
		}
	}
	if _, e := a.Observe(ctx); e != nil {
		t.Fatal("pre-entry denial retired original", e)
	}
	if e := a.FinishIdle(ctx); e != nil || finishes != 1 {
		t.Fatal("original finish", e)
	}
	if a.FinishIdle(ctx) == nil || finishes != 1 {
		t.Fatal("finish retried")
	}
	if _, e := a.Observe(ctx); e == nil {
		t.Fatal("finished observation")
	}
	if a.Close() != nil || a.Close() != nil || releases != 1 {
		t.Fatal("original release")
	}
	if a.FinishIdle(ctx) == nil || finishes != 1 {
		t.Fatal("closed finish")
	}
}

func TestRestartRetirementFailuresAreTerminal(t *testing.T) {
	for _, mode := range []string{"error", "cancel", "blocked", "missing-consumer", "released"} {
		t.Run(mode, func(t *testing.T) {
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			calls, releases := 0, 0
			a := restartActivationFixture(func(context.Context) (ExecutionDomainsObservation, error) { return ExecutionDomainsObservation{}, nil }, func() error { releases++; return nil })
			a.finish = func(context.Context) error {
				calls++
				if mode == "cancel" {
					stop()
					return nil
				}
				return ErrUnavailable
			}
			switch mode {
			case "blocked":
				a.blocked = true
			case "missing-consumer":
				a.finish = nil
			case "released":
				if a.Close() != nil {
					t.Fatal("release")
				}
			}
			e := a.FinishIdle(ctx)
			if e == nil {
				t.Fatal("failure accepted")
			}
			if mode == "cancel" && !errors.Is(e, context.Canceled) {
				t.Fatal("lost cancellation", e)
			}
			want := 0
			if mode == "error" || mode == "cancel" {
				want = 1
			}
			if calls != want {
				t.Fatal("consumer entered incorrectly")
			}
			if a.FinishIdle(context.Background()) == nil || calls != want {
				t.Fatal("failure retried")
			}
			if mode != "missing-consumer" {
				if _, e := a.Observe(context.Background()); e == nil {
					t.Fatal("failed lifetime observed")
				}
			}
			if a.Close() != nil || a.Close() != nil || releases != 1 {
				t.Fatal("failure release")
			}
		})
	}
}

func TestRestartRetirementExcludesObservationAndRelease(t *testing.T) {
	ctx := context.Background()
	entered, continueFinish := make(chan struct{}), make(chan struct{})
	released := make(chan struct{}, 1)
	a := restartActivationFixture(func(context.Context) (ExecutionDomainsObservation, error) {
		t.Error("observation entered after finish")
		return ExecutionDomainsObservation{}, nil
	}, func() error { released <- struct{}{}; return nil })
	a.finish = func(context.Context) error { close(entered); <-continueFinish; return nil }
	finished := make(chan error, 1)
	go func() { finished <- a.FinishIdle(ctx) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("finish not entered")
	}
	observed := make(chan error, 1)
	go func() { _, e := a.Observe(ctx); observed <- e }()
	closed := make(chan error, 1)
	go func() { closed <- a.Close() }()
	select {
	case <-released:
		t.Fatal("leases released during finish")
	case <-observed:
		t.Fatal("observation overlapped finish")
	case <-time.After(20 * time.Millisecond):
	}
	close(continueFinish)
	select {
	case e := <-finished:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("finish stuck")
	}
	select {
	case e := <-observed:
		if e == nil {
			t.Fatal("observation resumed")
		}
	case <-time.After(time.Second):
		t.Fatal("observe stuck")
	}
	select {
	case e := <-closed:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("close stuck")
	}
}
