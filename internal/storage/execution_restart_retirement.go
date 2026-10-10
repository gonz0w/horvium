// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
)

// FinishIdle durably closes only this original unused restarted lifetime.
// It does not drain workers or grant general shutdown/recovery authority.
func (a *ExecutionRestartActivation) FinishIdle(ctx context.Context) error {
	if ctx == nil || a == nil || a.original != a || a.finish == nil || a.observe == nil || a.release == nil {
		return ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return e
	}
	if a.closed {
		return ErrDenied
	}
	if a.blocked {
		return ErrStateUncertain
	}
	// Retire authority before entry; explicit release waits for this consumer.
	a.blocked = true
	return errors.Join(a.finish(ctx), ctx.Err())
}

// Framing only; the original native acknowledgement and empty census are required.
func unusedRestartClosedSuccessor(before executionJournal) (executionJournal, error) {
	next := before
	next.Generation++
	next.Owner.State = "closed"
	if e := checkExecutionJournalSuccessor(before, next); e != nil {
		return executionJournal{}, e
	}
	return next, nil
}
