// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import "context"

func ActivateFreshExecutionJournal(context.Context, string, string, func(context.Context, int) (ExecutionBackingObservation, error), func(context.Context) (PoolSystemUse, error), func(context.Context) ([16]byte, error)) (*ExecutionJournalActivation, error) {
	return nil, ErrUnavailable
}
