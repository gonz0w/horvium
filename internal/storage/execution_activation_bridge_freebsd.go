// SPDX-License-Identifier: BSD-3-Clause
package storage

import "context"

// ActivateFreshExecutionJournal is an internal trusted OS-root composition.
// Producers are fixed native facts, never operator input. No runtime consumer
// is registered. The returned lifetime can finish only its unused initial
// activation; it offers no general publisher or native launch method.
func ActivateFreshExecutionJournal(ctx context.Context, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error)) (*ExecutionJournalActivation, error) {
	original, e := activateFreshExecutionJournal(ctx, journal, ledger, read, system, boot)
	if e != nil {
		return nil, e
	}
	a := &ExecutionJournalActivation{observe: original.observe, release: original.close, finish: func(c context.Context) error { return original.finishIdle(c, nil) }}
	a.original = a
	return a, nil
}
