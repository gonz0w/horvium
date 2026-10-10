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
	closed, blocked bool
	closeErr        error
}

func (a *ExecutionJournalActivation) Observe(context.Context) (ExecutionDomainsObservation, error) {
	return ExecutionDomainsObservation{}, ErrDenied
}
func (a *ExecutionJournalActivation) Close() error { return ErrDenied }
