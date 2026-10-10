//go:build !freebsd

// SPDX-License-Identifier: BSD-3-Clause
package storage

import "context"

func ReviewExecutionPolicyStartup(context.Context, string, string, func(context.Context, int) (ExecutionBackingObservation, error), func(context.Context) (PoolSystemUse, error), func(context.Context) ([16]byte, error), ExecutionStartupPolicySource) (*ExecutionStartupReview, error) {
	return nil, ErrUnavailable
}
