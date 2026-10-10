// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"sync"
)

// An original read-only closed-incarnation review. It has no publication,
// incarnation, restore, restart-clearance or native launch method. Do not copy.
type ExecutionStartupReview struct {
	mu              sync.Mutex
	original        *ExecutionStartupReview
	observe         func(context.Context) (ExecutionDomainsObservation, error)
	release         func() error
	closed, blocked bool
	closeErr        error
}

func checkClosedExecutionStartup(j executionJournal, b executionJournalBinding, census string) error {
	return ErrDenied
}
func (r *ExecutionStartupReview) Observe(ctx context.Context) (ExecutionDomainsObservation, error) {
	return ExecutionDomainsObservation{}, ErrDenied
}
func (r *ExecutionStartupReview) Close() error { return ErrDenied }
