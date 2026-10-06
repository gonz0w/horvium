// SPDX-License-Identifier: BSD-3-Clause
package storage

import "os"

// PublishOwnership publishes metadata, never enrollment/storage authority. Zero
// expectedRevision requires absence and revision one; replacement requires the
// exact predecessor and successor. Capacity includes jobs and the entire new
// temporary alongside every old record. No repair/deletion/retry is provided.
func (s *StateDirectory) PublishOwnership(p Policy, expectedRevision uint64) error {
	return s.publishOwnership(p, expectedRevision, nil)
}

// Package-private reduced limits/phase seam, never a runtime tuning API.
func (s *StateDirectory) publishOwnership(p Policy, expected uint64, wrap func(ownershipPublication) ownershipPublication) error {
	return s.publishOwnershipBounded(p, expected, ownershipPublicationLimits{maxRecords, maxOwnershipCatalogBytes}, wrap)
}

func (s *StateDirectory) publishOwnershipBounded(p Policy, expected uint64, bounds ownershipPublicationLimits, wrap func(ownershipPublication) ownershipPublication) error {
	if s == nil || os.Geteuid() != 0 {
		return ErrStateDirectory
	}
	data, err := EncodeOwnershipRecord(p)
	if err != nil {
		return err
	}
	if !validOwnershipPublicationRevision(p.Revision, expected) {
		return ErrDrift
	}
	if !bounds.valid() {
		return ErrLimit
	}
	limits := defaultStateCatalogLimits
	limits.ownership = bounds.records
	limits.bytes = bounds.bytes
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.publishStateLocked("ownership-"+p.PoolGUID+".json", ".ownership-", data, expected, limits,
		func(c StateCatalog) error {
			return allowMixedOwnershipPublication(c, p, expected, uint64(len(data)), limits)
		},
		func() error {
			current, e := s.readOwnershipLocked(p.PoolGUID)
			if e != nil {
				return e
			}
			if expected == 0 || current.Revision != expected {
				return ErrDrift
			}
			return nil
		}, wrap)
}
