// SPDX-License-Identifier: BSD-3-Clause
package storage

import "context"

// Trusted root composition; production fixes native producers. Inspection only.
func ReviewExecutionStartup(ctx context.Context, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error)) (*ExecutionStartupReview, error) {
	return nil, ErrDenied
}
