// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"sync"
)

// ExecutionJournalActivation retains the original initial journal receipt through
// an internal root boundary. Observation and release only; never startup/global
// registration, clean shutdown or a native launch credential. Do not copy it.
type ExecutionJournalActivation struct {
	mu              sync.Mutex
	original        *ExecutionJournalActivation
	observe         func(context.Context) (ExecutionDomainsObservation, error)
	release         func() error
	finish          func(context.Context) error
	closed, blocked bool
	closeErr        error
}

// Observe repeats the original native facts. Copies are rejected before touching
// their mutex; a failed producer permanently retires this lifetime.
func (a *ExecutionJournalActivation) Observe(ctx context.Context) (ExecutionDomainsObservation, error) {
	zero := ExecutionDomainsObservation{}
	if a == nil || a.original != a || a.observe == nil || a.release == nil || ctx == nil {
		return zero, ErrDenied
	}
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	if a.closed {
		return zero, ErrDenied
	}
	if a.blocked {
		return zero, ErrStateUncertain
	}
	got, e := a.observe(ctx)
	if e != nil {
		a.blocked = true
		return zero, e
	}
	return got, nil
}

// Close waits for observation, releases the original leases once and retains the
// original release result. It never publishes orderly shutdown or repairs state.
func (a *ExecutionJournalActivation) Close() error {
	if a == nil || a.original != a || a.release == nil {
		return ErrDenied
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.closed {
		a.closed = true
		a.closeErr = a.release()
	}
	return a.closeErr
}
