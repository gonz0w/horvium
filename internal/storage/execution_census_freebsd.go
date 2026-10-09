// SPDX-License-Identifier: BSD-3-Clause
package storage

// Private native observation; a supplied binding is metadata, never provenance.
// The original journal owner must establish its objects/backing independently.
func (s *StateDirectory) inspectExecutionCensus(binding executionJournalBinding, limits stateCatalogLimits, afterRead func()) (StateCatalog, string, error) {
	return StateCatalog{}, "", ErrDenied
}
