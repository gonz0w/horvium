// SPDX-License-Identifier: BSD-3-Clause
package storage

import "context"

// No reconstructed journal or caller declaration can finish an initial lifetime.
func (a *freshExecutionActivation) finishIdle(context.Context, func(ownershipPublication) ownershipPublication) error {
	return ErrDenied
}
