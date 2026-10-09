// SPDX-License-Identifier: BSD-3-Clause
package storage

import "golang.org/x/sys/unix"

// Private file durability component, never startup/global/execution authority.
// Original native provisioning/backing and dual-lease owner are still required.
type executionJournalStore struct {
	original             *executionJournalStore
	lease                *StateDirectory
	acknowledged, failed bool
	ack                  executionJournal
	ackBytes             []byte
	ackMetadata          unix.Stat_t
}

func openExecutionJournalStore(path string) (*executionJournalStore, error) {
	return nil, ErrDenied
}
func (s *executionJournalStore) close() error { return ErrDenied }
func (s *executionJournalStore) read(afterRead func()) (executionJournal, error) {
	return executionJournal{}, ErrDenied
}
func (s *executionJournalStore) initialize(j executionJournal, wrap func(ownershipPublication) ownershipPublication) error {
	return ErrDenied
}
func (s *executionJournalStore) publish(j executionJournal, wrap func(ownershipPublication) ownershipPublication) error {
	return ErrDenied
}

func openStateDirectory(path string, create bool) (*StateDirectory, error) { return nil, ErrDenied }
