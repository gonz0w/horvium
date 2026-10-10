// SPDX-License-Identifier: BSD-3-Clause
package storage

import "context"

func (n *executionStartupNative) finishIdle(ctx context.Context, wrap func(ownershipPublication) ownershipPublication) error {
	return ErrDenied
}
