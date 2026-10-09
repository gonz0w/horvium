// SPDX-License-Identifier: BSD-3-Clause
package storage

import "sync"

// ExecutionDomainLeases owns pre-existing journal then ledger leases. It has no
// commissioning, publisher, startup, restore or global launch authority.
// Do not copy it. Close releases the owned leases, never their persistent files.
type ExecutionDomainLeases struct {
	mu              sync.Mutex
	origin          *ExecutionDomainLeases
	native          *executionDomainLeaseNative
	closed, blocked bool
	binding         executionJournalBinding
	bound           bool
}

// ExecutionDomainsObservation is immutable private metadata under both original
// leases. It cannot serialize authority or reconstruct a publication receipt.
type ExecutionDomainsObservation struct {
	origin                *ExecutionDomainLeases
	binding               executionJournalBinding
	census, journalDigest string
	journalPresent        bool
	journal               executionJournal
}

func checkExecutionDomainJournal(binding executionJournalBinding, present bool, journal executionJournal) error {
	if !validExecutionJournalBinding(binding) {
		return ErrStateRecord
	}
	if !present {
		if journal != (executionJournal{}) {
			return ErrDrift
		}
		return nil
	}
	if err := validateExecutionJournal(journal); err != nil {
		return err
	}
	if journal.Binding != binding {
		return ErrDrift
	}
	return nil
}
