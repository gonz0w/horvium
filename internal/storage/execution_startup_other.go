// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import "context"

func ReviewExecutionStartup(ctx context.Context, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error)) (*ExecutionStartupReview, error) {
	return nil, ErrStateDirectory
}
