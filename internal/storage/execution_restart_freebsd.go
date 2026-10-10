// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"io"
)

func (n *executionStartupNative) activate(ctx context.Context, entropy io.Reader, wrap func(ownershipPublication) ownershipPublication) (*ExecutionRestartActivation, error) {
	return nil, ErrDenied
}
