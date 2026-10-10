// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"sync"
)

// Original newly published active incarnation. Observation and explicit release
// only; no runtime publisher, shutdown, recovery or native launch authority.
type ExecutionRestartActivation struct {
	mu              sync.Mutex
	original        *ExecutionRestartActivation
	observe         func(context.Context) (ExecutionDomainsObservation, error)
	release         func() error
	closed, blocked bool
	closeErr        error
}

// Framing only. Original review and native durable publication remain mandatory.
func closedExecutionJournalSuccessor(before executionJournal, owner string, boot [16]byte) (executionJournal, error) {
	if e := checkClosedExecutionStartup(before, before.Binding, before.Census); e != nil {
		return executionJournal{}, e
	}
	if before.Generation >= math.MaxUint64-1 {
		return executionJournal{}, ErrLimit
	}
	if !validJournalIdentity(owner) || owner == before.Owner.ID || owner == before.Binding.Installation {
		return executionJournal{}, ErrDenied
	}
	next := before
	next.Generation++
	next.Owner = executionJournalOwner{ID: owner, Boot: hex.EncodeToString(boot[:]), State: "active"}
	if e := validateExecutionJournal(next); e != nil {
		return executionJournal{}, e
	}
	return next, nil
}

// Consume the original native review once. No caller supplies new identities,
// boot, durable receipt or replacement state. Failed publication never retries.
func (r *ExecutionStartupReview) Activate(ctx context.Context) (*ExecutionRestartActivation, error) {
	if ctx == nil || r == nil || r.original != r || r.activate == nil || r.observe == nil || r.release == nil {
		return nil, ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if r.closed || r.blocked || r.transferred {
		return nil, ErrDenied
	}
	r.blocked = true
	a, e := r.activate(ctx)
	e = errors.Join(e, ctx.Err())
	if e == nil && (a == nil || a.original != a || a.observe == nil || a.release == nil) {
		e = ErrDenied
	}
	if e != nil {
		if a != nil {
			e = errors.Join(e, a.Close())
		}
		return nil, e
	}
	r.transferred = true
	return a, nil
}
func (a *ExecutionRestartActivation) Observe(ctx context.Context) (ExecutionDomainsObservation, error) {
	zero := ExecutionDomainsObservation{}
	if ctx == nil || a == nil || a.original != a || a.observe == nil || a.release == nil {
		return zero, ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	if a.closed {
		return zero, ErrDenied
	}
	if a.blocked {
		return zero, ErrStateUncertain
	}
	got, e := a.observe(ctx)
	e = errors.Join(e, ctx.Err())
	if e != nil {
		a.blocked = true
		return zero, e
	}
	return got, nil
}
func (a *ExecutionRestartActivation) Close() error {
	if a == nil || a.original != a || a.release == nil {
		return ErrDenied
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.closed {
		a.closed = true
		a.blocked = true
		a.closeErr = a.release()
	}
	return a.closeErr
}
