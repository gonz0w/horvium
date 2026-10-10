// SPDX-License-Identifier: BSD-3-Clause
package storage

import "context"

// FinishIdle durably closes only the original, unused initial activation. It is
// not worker draining, restart clearance or a runtime shutdown operation.
func (a *ExecutionJournalActivation) FinishIdle(context.Context) error { return ErrDenied }
