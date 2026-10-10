// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package freebsd

import "context"

func reviewNativeExecutionStartupOwner(ctx context.Context, journal, ledger string) (*commissionedExecutionOwner, error) {
	return nil, ErrUnsupported
}
