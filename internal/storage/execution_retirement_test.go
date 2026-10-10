// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecutionActivationFinishIdle(t *testing.T) {
	ctx := context.Background()
	for _, outcome := range []error{nil, ErrUnavailable} {
		finished, released, observed := 0, 0, 0
		a := &ExecutionJournalActivation{}
		a.original = a
		a.observe = func(context.Context) (ExecutionDomainsObservation, error) {
			observed++
			return ExecutionDomainsObservation{}, nil
		}
		a.release = func() error { released++; return nil }
		a.finish = func(context.Context) error { finished++; return outcome }
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		for _, c := range []context.Context{nil, cancelled} {
			if a.FinishIdle(c) == nil || finished != 0 {
				t.Fatal("invalid context reached finish")
			}
		}
		copy := &ExecutionJournalActivation{original: a, finish: a.finish}
		for _, q := range []*ExecutionJournalActivation{nil, {}, copy} {
			if q.FinishIdle(ctx) == nil || finished != 0 {
				t.Fatal("foreign finish accepted")
			}
		}
		if _, e := a.Observe(ctx); e != nil {
			t.Fatal("pre-entry cancellation retired original", e)
		}
		if e := a.FinishIdle(ctx); !errors.Is(e, outcome) || finished != 1 {
			t.Fatal("original finish", e)
		}
		if !errors.Is(a.FinishIdle(ctx), ErrStateUncertain) || finished != 1 {
			t.Fatal("finish repeated")
		}
		if _, e := a.Observe(ctx); !errors.Is(e, ErrStateUncertain) || observed != 1 {
			t.Fatal("finished original observed")
		}
		if a.Close() != nil || a.Close() != nil || released != 1 {
			t.Fatal("finish prevented release")
		}
		if !errors.Is(a.FinishIdle(ctx), ErrDenied) {
			t.Fatal("closed finish accepted")
		}
	}
}
func TestExecutionActivationFinishExclusion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	entered, resume := make(chan struct{}), make(chan struct{})
	a := &ExecutionJournalActivation{}
	a.original = a
	a.observe = func(context.Context) (ExecutionDomainsObservation, error) {
		t.Error("retired observation reached producer")
		return ExecutionDomainsObservation{}, nil
	}
	a.release = func() error { return nil }
	a.finish = func(context.Context) error {
		close(entered)
		select {
		case <-resume:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	finished := make(chan error, 1)
	go func() { finished <- a.FinishIdle(ctx) }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("finish not entered")
	}
	closed := make(chan error, 1)
	go func() { closed <- a.Close() }()
	observation := make(chan error, 1)
	go func() { _, e := a.Observe(ctx); observation <- e }()
	early := false
	select {
	case <-closed:
		early = true
	case <-time.After(30 * time.Millisecond):
	}
	close(resume)
	if early {
		t.Fatal("release overlapped finish")
	}
	if e := <-finished; e != nil {
		t.Fatal(e)
	}
	if e := <-closed; e != nil {
		t.Fatal(e)
	}
	if e := <-observation; e == nil {
		t.Fatal("retired observation accepted")
	}
}
