// SPDX-License-Identifier: BSD-3-Clause
package storage

import "errors"

// ErrStateUncertain means publication or cleanup may have a visible metadata
// effect without confirmed durability. It is never a successful acknowledgement.
// Reconcile actual state before any retry; the current lease blocks more writes.
var ErrStateUncertain = errors.New("storage state publication requires reconciliation")

// The private transaction seam makes phase ordering/failure handling testable.
// Native implementations still perform real descriptor-relative operations;
// this is not an executor, user failpoint or durability simulation interface.
type ownershipPublication struct {
	write, syncFile, closeFile, verify, rename, syncDirectory, cleanup func() error
}

func commitOwnershipPublication(tx ownershipPublication) error {
	for _, step := range []func() error{tx.write, tx.syncFile, tx.closeFile, tx.verify, tx.rename} {
		if err := step(); err != nil {
			if tx.cleanup() != nil {
				return ErrStateUncertain
			}
			if errors.Is(err, ErrDrift) || errors.Is(err, ErrLimit) || errors.Is(err, ErrStateUncertain) || errors.Is(err, ErrStateRecord) {
				return err
			}
			return ErrStateDirectory
		}
	}
	// Rename has already published. Never unlink/roll back the destination on a
	// failed directory sync: a visible record is not a durable acknowledgement.
	if tx.syncDirectory() != nil {
		return ErrStateUncertain
	}
	return nil
}

func validOwnershipPublicationRevision(next, expected uint64) bool {
	return expected != ^uint64(0) && next == expected+1
}
