// SPDX-License-Identifier: BSD-3-Clause
package storage

import "context"

// FinishIdle durably closes only the original, unused initial activation. It is
// not worker draining, restart clearance or a runtime shutdown operation.
func (a *ExecutionJournalActivation) FinishIdle(ctx context.Context) error {
	if a == nil || a.original != a || a.finish == nil || a.release == nil || ctx == nil {
		return ErrDenied
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if a.closed {
		return ErrDenied
	}
	if a.blocked {
		return ErrStateUncertain
	}
	// Retire observation/finish admission before invoking the original consumer.
	// Release still waits for that consumer and remains possible on every outcome.
	a.blocked = true
	return a.finish(ctx)
}
