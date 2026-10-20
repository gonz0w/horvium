// SPDX-License-Identifier: BSD-3-Clause
package storage

import "sync"

// ExecutionDomainLeases owns pre-existing journal then ledger leases. It has no
// commissioning, startup, restore or global launch authority. The public opener
// is observation-only; private publication additionally needs the original
// journal acknowledgement, which cannot be recovered from disk.
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
	systemObserved        bool // Observation only; no commissioning or global permit.
	jobsPresent           bool // Complete census, including terminal/other-pool jobs.
}

func (o ExecutionDomainsObservation) controlInspectionState() (string, error) {
	if o.origin == nil || !o.systemObserved || !o.journalPresent {
		return "", ErrUnavailable
	}
	if o.jobsPresent || checkClosedExecutionStartup(o.journal, o.binding, o.census) != nil {
		return "recovery-required", nil
	}
	return "closed", nil
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
