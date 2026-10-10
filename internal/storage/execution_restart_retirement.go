// SPDX-License-Identifier: BSD-3-Clause
package storage

import "context"

// FinishIdle is restricted to the original unused restarted lifetime.
func (a *ExecutionRestartActivation) FinishIdle(ctx context.Context) error { return ErrDenied }

func unusedRestartClosedSuccessor(before executionJournal) (executionJournal, error) {
	return executionJournal{}, ErrDenied
}
