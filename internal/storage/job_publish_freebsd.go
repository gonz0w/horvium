// SPDX-License-Identifier: BSD-3-Clause
package storage

import "os"

// PublishJob publishes validated lifecycle metadata, not admission or native
// execution authority. Creation requires revision one and absence; updates read
// the exact current durable predecessor and append one immutable-history event.
// A nil result confirms this metadata sync/rename sequence only. Caller owns j
// during the call. No automatic retry, pruning or reconciliation is provided.
func (s *StateDirectory) PublishJob(j Job, expectedRevision uint64) error {
	return s.publishJobBounded(j, expectedRevision, defaultStateCatalogLimits, nil)
}

// Reduced limits and phase wrapper exist solely for deterministic owned fixtures.
func (s *StateDirectory) publishJobBounded(j Job, expected uint64, limits stateCatalogLimits, wrap func(ownershipPublication) ownershipPublication) error {
	if s == nil || os.Geteuid() != 0 {
		return ErrStateDirectory
	}
	data, err := EncodeJobRecord(j)
	if err != nil {
		return err
	}
	if !validOwnershipPublicationRevision(uint64(len(j.History)), expected) {
		return ErrDrift
	}
	if !limits.valid() {
		return ErrLimit
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.publishStateLocked("job-"+j.ID+".json", ".job-", data, expected, limits,
		func(c StateCatalog) error { return allowJobPublication(c, j, expected, uint64(len(data)), limits) },
		func() error {
			previous, e := s.readJobLocked(j.ID)
			if e != nil {
				return e
			}
			if previous.History[len(previous.History)-1].Revision != expected {
				return ErrDrift
			}
			return ValidateJobSuccessor(previous, j)
		}, wrap)
}
