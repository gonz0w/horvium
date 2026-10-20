// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"

	"github.com/gonz0w/horvium/internal/storage"
)

// Fixed native producers only, through the original one-shot fresh receipt.
// No registered startup/global owner or runtime write eligibility.
func activateNativeFreshExecutionJournal(ctx context.Context, journal, ledger string) (*storage.ExecutionJournalActivation, error) {
	if ctx == nil || !executionRoot() {
		return nil, storage.ErrDenied
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	source, e := NewSystemUseSource()
	if e != nil {
		return nil, e
	}
	return storage.ActivateFreshExecutionJournal(ctx, journal, ledger, readNativeExecutionBacking, source.Collect, readNativeActivationBoot)
}

// Transfer the exact bounded kernel bytes, without UUID/text normalization.
func readNativeActivationBoot(ctx context.Context) ([16]byte, error) {
	if ctx == nil {
		return [16]byte{}, storage.ErrDenied
	}
	if ctx.Err() != nil {
		return [16]byte{}, ctx.Err()
	}
	id, e := readNativeBoot()
	if e != nil {
		return [16]byte{}, e
	}
	if ctx.Err() != nil {
		return [16]byte{}, ctx.Err()
	}
	return [16]byte(id), nil
}
