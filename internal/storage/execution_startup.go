// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"math"
	"sync"
)

// An original read-only closed-incarnation review. It has no publication,
// incarnation, restore, restart-clearance or native launch method. Do not copy.
type ExecutionStartupReview struct {
	mu              sync.Mutex
	original        *ExecutionStartupReview
	observe         func(context.Context) (ExecutionDomainsObservation, error)
	release         func() error
	activate        func(context.Context) (*ExecutionRestartActivation, error)
	transferred     bool
	closed, blocked bool
	closeErr        error
}

// Metadata consistency only. Native original leases, exact file receipt, current
// kernel boot and complete independent system/backing checks remain mandatory.
func checkClosedExecutionStartup(j executionJournal, b executionJournalBinding, census string) error {
	if e := validateExecutionJournal(j); e != nil {
		return e
	}
	if j.Generation == math.MaxUint64 {
		return ErrLimit
	}
	if j.Owner.State != "closed" || j.Metadata.Pending || j.Native.Pending || j.Disarmed {
		return ErrDenied
	}
	if j.Binding != b || j.Census != census {
		return ErrDrift
	}
	return nil
}
func (r *ExecutionStartupReview) Observe(ctx context.Context) (ExecutionDomainsObservation, error) {
	zero := ExecutionDomainsObservation{}
	if ctx == nil || r == nil || r.original != r || r.observe == nil || r.release == nil {
		return zero, ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	if r.closed {
		return zero, ErrDenied
	}
	if r.blocked {
		return zero, ErrStateUncertain
	}
	got, e := r.observe(ctx)
	e = errors.Join(e, ctx.Err())
	if e != nil {
		r.blocked = true
		return zero, e
	}
	return got, nil
}
func (r *ExecutionStartupReview) Close() error {
	if r == nil || r.original != r || r.release == nil {
		return ErrDenied
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.transferred {
		return ErrDenied
	}
	if !r.closed {
		r.closed = true
		r.blocked = true
		r.closeErr = r.release()
	}
	return r.closeErr
}
