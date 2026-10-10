//go:build !freebsd

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "context"

func activateNativeClosedExecutionOwner(ctx context.Context, journal, ledger string) (*commissionedExecutionOwner, error) {
	return nil, ErrUnsupported
}
