// SPDX-License-Identifier: BSD-3-Clause
package storage

import "os"

// PublishPropertyJob publishes baseline/lifecycle metadata under the held lease,
// not admission or storage authority. No runtime caller consumes this primitive.
func (s *StateDirectory) PublishPropertyJob(p PropertyJob, expectedRevision uint64) error {
	return s.publishPropertyJobBounded(p, expectedRevision, defaultStateCatalogLimits, nil)
}

// Reduced limits/phase wrapper exist solely for deterministic disposable fixtures.
func (s *StateDirectory) publishPropertyJobBounded(p PropertyJob, expected uint64, limits stateCatalogLimits, wrap func(ownershipPublication) ownershipPublication) error {
	if s == nil || os.Geteuid() != 0 {
		return ErrStateDirectory
	}
	data, err := EncodePropertyJob(p)
	if err != nil {
		return err
	}
	if !validOwnershipPublicationRevision(uint64(len(p.job.History)), expected) {
		return ErrDrift
	}
	if !limits.valid() {
		return ErrLimit
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.publishStateLocked("job-"+p.job.ID+".json", ".job-", data, expected, limits,
		func(c StateCatalog) error { return allowJobPublication(c, p.job, expected, uint64(len(data)), limits) },
		func() error {
			previous, e := s.readPropertyJobLocked(p.job.ID)
			if e != nil {
				return e
			}
			if previous.job.History[len(previous.job.History)-1].Revision != expected {
				return ErrDrift
			}
			return ValidatePropertyJobSuccessor(previous, p)
		}, wrap)
}
