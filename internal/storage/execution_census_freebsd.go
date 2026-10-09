// SPDX-License-Identifier: BSD-3-Clause
package storage

import "os"

// Private native observation; a supplied binding is metadata, never provenance.
// The original journal owner must establish its objects/backing independently.
func (s *StateDirectory) inspectExecutionCensus(binding executionJournalBinding, limits stateCatalogLimits, afterRead func()) (StateCatalog, string, error) {
	if s == nil || os.Geteuid() != 0 {
		return StateCatalog{}, "", ErrStateDirectory
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inspectExecutionCensusLocked(binding, limits, afterRead)
}

// Caller holds mu. Finish only after the native reader's final whole-namespace
// checks, never from a partial stream or solely from valid record checksums.
// Observation cannot clear previously latched publication uncertainty.
func (s *StateDirectory) inspectExecutionCensusLocked(binding executionJournalBinding, limits stateCatalogLimits, afterRead func()) (StateCatalog, string, error) {
	if s == nil || os.Geteuid() != 0 {
		return StateCatalog{}, "", ErrStateDirectory
	}
	if s.propertyOrigin != s {
		return StateCatalog{}, "", ErrDenied
	}
	if !limits.valid() {
		return StateCatalog{}, "", ErrLimit
	}
	if !validExecutionJournalBinding(binding) {
		return StateCatalog{}, "", ErrStateRecord
	}
	var census *executionCensus
	catalog, err := s.inspectStateStreamLocked(limits, afterRead, "", func(names []string) error {
		var e error
		census, e = newExecutionCensus(binding, names, limits)
		return e
	}, func(entry stateRecordName, data []byte) error {
		return census.add(entry.name, data)
	})
	if err != nil {
		return StateCatalog{}, "", err
	}
	digest, err := census.finish()
	if err != nil {
		return StateCatalog{}, "", err
	}
	return catalog, digest, nil
}
