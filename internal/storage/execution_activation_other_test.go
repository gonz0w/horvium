// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"context"
	"errors"
	"testing"
)

func TestExecutionActivationUnsupportedHost(t *testing.T) {
	called := false
	a, e := ActivateFreshExecutionJournal(context.Background(), "unused-journal", "unused-ledger", func(context.Context, int) (ExecutionBackingObservation, error) {
		called = true
		return ExecutionBackingObservation{}, nil
	}, func(context.Context) (PoolSystemUse, error) { called = true; return PoolSystemUse{}, nil }, func(context.Context) ([16]byte, error) { called = true; return [16]byte{}, nil })
	if a != nil || !errors.Is(e, ErrUnavailable) || called {
		t.Fatal("unsupported host touched producers", e)
	}
}
