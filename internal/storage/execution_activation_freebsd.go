// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"io"
)

// Unregistered initial journal receipt only; never a runtime launch credential.
type freshExecutionActivation struct {
	original *freshExecutionActivation
	fresh    *freshExecutionDomains
	journal  executionJournal
}

func (f *freshExecutionDomains) activate(ctx context.Context, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error), entropy io.Reader, wrap func(ownershipPublication) ownershipPublication) (*freshExecutionActivation, error) {
	return nil, ErrDenied
}
func (a *freshExecutionActivation) observe(ctx context.Context, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error)) (ExecutionDomainsObservation, error) {
	return ExecutionDomainsObservation{}, ErrDenied
}
func (a *freshExecutionActivation) close() error { return ErrDenied }
