// SPDX-License-Identifier: BSD-3-Clause
package storage

import "os"

// InspectJobReservations derives a complete immutable job view from the held
// root lease's strictly validated mixed namespace. It includes ownership bytes
// in candidate capacity checks and retains full property baselines. Every error
// returns zero state. It acquires no reservation and clears no uncertainty;
// ordinary drift checks/advisory locking cannot fence another OS-root actor.
func (s *StateDirectory) InspectJobReservations() (JobReservations, error) {
	return s.inspectJobReservations(defaultStateCatalogLimits, nil)
}

// Reduced bounds/afterRead are package-private disposable fixture seams.
func (s *StateDirectory) inspectJobReservations(limits stateCatalogLimits, afterRead func()) (JobReservations, error) {
	if s == nil || os.Geteuid() != 0 {
		return JobReservations{}, ErrStateDirectory
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	view, _, err := s.inspectJobReservationsLocked(limits, afterRead, "")
	return view, err
}

// Caller holds mu. Full record binding and reservation accounting share one
// census; no partial view/catalog escapes an unsuccessful final check.
func (s *StateDirectory) inspectJobReservationsLocked(limits stateCatalogLimits, afterRead func(), temporary string) (JobReservations, StateCatalog, error) {
	b, err := newReservationBuilder(limits)
	if err != nil {
		return JobReservations{}, StateCatalog{}, err
	}
	catalog, err := s.inspectStateRecordsLocked(limits, afterRead, temporary, func(entry stateRecordName, data []byte) error {
		if entry.jobID != "" {
			return b.add(data)
		}
		return nil
	})
	if err != nil {
		return JobReservations{}, StateCatalog{}, err
	}
	view := b.finish()
	view.bytes = catalog.Bytes
	return view, catalog, nil
}
