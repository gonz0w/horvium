// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

// Fixed native producers only, through the original one-shot fresh receipt.
// No registered startup/global owner or runtime write eligibility.
func activateNativeFreshExecutionJournal(context.Context, string, string) (*storage.ExecutionJournalActivation, error) {
	return nil, storage.ErrDenied
}

func readNativeActivationBoot(context.Context) ([16]byte, error) {
	return [16]byte{}, storage.ErrDenied
}
