// SPDX-License-Identifier: BSD-3-Clause
package storage

import "context"

// ActivateFreshExecutionJournal is an internal trusted OS-root composition.
// Producers are fixed native facts, never operator input. No runtime consumer
// is registered, and the returned lifetime offers no publication/launch methods.
func ActivateFreshExecutionJournal(context.Context, string, string, func(context.Context, int) (ExecutionBackingObservation, error), func(context.Context) (PoolSystemUse, error), func(context.Context) ([16]byte, error)) (*ExecutionJournalActivation, error) {
	return nil, ErrDenied
}
