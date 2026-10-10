// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestExecutionActivationReceiptLifetime(t *testing.T) {
	ctx := context.Background()
	observed, released := 0, 0
	value := ExecutionDomainsObservation{census: "private-test-metadata"}
	a := &ExecutionJournalActivation{}
	a.original = a
	a.observe = func(c context.Context) (ExecutionDomainsObservation, error) {
		if c != ctx {
			t.Error("context lost")
		}
		observed++
		return value, nil
	}
	a.release = func() error { released++; return nil }
	for i := 0; i < 2; i++ {
		got, e := a.Observe(ctx)
		if e != nil || got != value {
			t.Fatal("original observation", e)
		}
	}
	if observed != 2 {
		t.Fatal("observation not repeated")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	for _, c := range []context.Context{nil, cancelled} {
		got, e := a.Observe(c)
		if e == nil || got != (ExecutionDomainsObservation{}) || observed != 2 {
			t.Fatal("invalid context reached producer")
		}
	}
	copied := &ExecutionJournalActivation{original: a, observe: a.observe, release: a.release}
	got, e := copied.Observe(ctx)
	if e == nil || got != (ExecutionDomainsObservation{}) || copied.Close() == nil || observed != 2 || released != 0 {
		t.Fatal("copied lifetime reached original")
	}
	wire, e := json.Marshal(a)
	if e != nil || string(wire) != "{}" {
		t.Fatal("authority serialized")
	}
	var decoded ExecutionJournalActivation
	if json.Unmarshal(wire, &decoded) != nil {
		t.Fatal("metadata decode")
	}
	got, e = decoded.Observe(ctx)
	if e == nil || got != (ExecutionDomainsObservation{}) || decoded.Close() == nil {
		t.Fatal("serialized lifetime reconstructed")
	}
	if a.Close() != nil || a.Close() != nil || released != 1 {
		t.Fatal("close not idempotent")
	}
	got, e = a.Observe(ctx)
	if e == nil || got != (ExecutionDomainsObservation{}) || observed != 2 {
		t.Fatal("closed lifetime reached producer")
	}
	got, e = (*ExecutionJournalActivation)(nil).Observe(ctx)
	if e == nil || got != (ExecutionDomainsObservation{}) || (*ExecutionJournalActivation)(nil).Close() == nil {
		t.Fatal("nil lifetime accepted")
	}
}
func TestExecutionActivationReceiptFailure(t *testing.T) {
	called, released := 0, 0
	a := &ExecutionJournalActivation{}
	a.original = a
	a.observe = func(context.Context) (ExecutionDomainsObservation, error) {
		called++
		return ExecutionDomainsObservation{journalPresent: true}, ErrUnavailable
	}
	a.release = func() error { released++; return ErrUnavailable }
	got, e := a.Observe(context.Background())
	if !errors.Is(e, ErrUnavailable) || got != (ExecutionDomainsObservation{}) {
		t.Fatal("partial observation escaped")
	}
	got, e = a.Observe(context.Background())
	if !errors.Is(e, ErrStateUncertain) || got != (ExecutionDomainsObservation{}) || called != 1 {
		t.Fatal("failed lifetime retried producer")
	}
	if !errors.Is(a.Close(), ErrUnavailable) || !errors.Is(a.Close(), ErrUnavailable) || released != 1 {
		t.Fatal("release error lost or repeated")
	}
}
func TestExecutionActivationReceiptCloseExclusion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	entered, release := make(chan struct{}), make(chan struct{})
	a := &ExecutionJournalActivation{}
	a.original = a
	a.observe = func(context.Context) (ExecutionDomainsObservation, error) {
		close(entered)
		select {
		case <-release:
		case <-ctx.Done():
			return ExecutionDomainsObservation{}, ctx.Err()
		}
		return ExecutionDomainsObservation{}, nil
	}
	released := false
	a.release = func() error { released = true; return nil }
	done := make(chan error, 1)
	go func() { _, e := a.Observe(ctx); done <- e }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("observation did not enter")
	}
	started, closed := make(chan struct{}), make(chan error, 1)
	go func() { close(started); closed <- a.Close() }()
	<-started
	early := false
	select {
	case <-closed:
		early = true
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	if early {
		t.Fatal("release overlapped observation")
	}
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	if e := <-closed; e != nil || !released {
		t.Fatal("original close did not finish", e)
	}
}
