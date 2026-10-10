// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

func activateNativeClosedExecutionOwner(ctx context.Context, journal, ledger string) (*commissionedExecutionOwner, error) {
	return nil, storage.ErrDenied
}
