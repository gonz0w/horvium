// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
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

func closedExecutionJournalSuccessor(before executionJournal, owner string, boot [16]byte) (executionJournal, error) {
	return executionJournal{}, ErrDenied
}

// Activate consumes only an original native closed-state review. No caller
// supplies the new identity, boot, durable receipt or replacement state.
func (r *ExecutionStartupReview) Activate(ctx context.Context) (*ExecutionRestartActivation, error) {
	return nil, ErrDenied
}
func (a *ExecutionRestartActivation) Observe(ctx context.Context) (ExecutionDomainsObservation, error) {
	return ExecutionDomainsObservation{}, ErrDenied
}
func (a *ExecutionRestartActivation) Close() error { return ErrDenied }
