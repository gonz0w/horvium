// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// Private file durability component, never startup/global/execution authority.
// Original native provisioning/backing and dual-lease owner are still required.
type executionJournalStore struct {
	original             *executionJournalStore
	lease                *StateDirectory
	initializable        bool
	acknowledged, failed bool
	ack                  executionJournal
	ackBytes             []byte
	ackMetadata          unix.Stat_t
}

func openExecutionJournalStore(path string) (*executionJournalStore, error) {
	lease, err := openStateDirectory(path, false)
	if err != nil {
		return nil, err
	}
	s := &executionJournalStore{lease: lease}
	s.original = s
	lease.mu.Lock()
	snapshot, err := s.inspectLocked("", nil)
	s.initializable = !snapshot.present
	lease.mu.Unlock()
	if err != nil {
		_ = lease.Close()
		return nil, err
	}
	// Existing readable metadata does not reconstruct a publication receipt.
	return s, nil
}

// Releasing descriptors does not publish a closed lifecycle or prove quiescence.
func (s *executionJournalStore) close() error {
	if s == nil {
		return nil
	}
	if s.original != s || s.lease == nil {
		return ErrDenied
	}
	return s.lease.Close()
}

func (s *executionJournalStore) guard() error {
	if s == nil || os.Geteuid() != 0 {
		return ErrStateDirectory
	}
	if s.original != s || s.lease == nil {
		return ErrDenied
	}
	return nil
}

func (s *executionJournalStore) read(afterRead func()) (executionJournal, error) {
	if err := s.guard(); err != nil {
		return executionJournal{}, err
	}
	s.lease.mu.Lock()
	defer s.lease.mu.Unlock()
	snapshot, err := s.inspectLocked("", afterRead)
	if err != nil {
		s.retireLocked()
		return executionJournal{}, err
	}
	if !snapshot.present {
		if !s.initializable {
			s.retireLocked()
		}
		return executionJournal{}, ErrStateDirectory
	}
	s.initializable = false
	return snapshot.record, nil
}

func (s *executionJournalStore) initialize(j executionJournal, wrap func(ownershipPublication) ownershipPublication) error {
	if err := s.guard(); err != nil {
		return err
	}
	s.lease.mu.Lock()
	defer s.lease.mu.Unlock()
	if s.failed || s.lease.publicationBlocked {
		return ErrStateUncertain
	}
	if s.acknowledged || !s.initializable {
		return ErrDenied
	}
	if err := checkInitialExecutionJournal(j); err != nil {
		return err
	}
	snapshot, err := s.inspectLocked("", nil)
	if err != nil {
		s.retireLocked()
		return err
	}
	if snapshot.present {
		s.initializable = false
		return ErrDenied
	}
	return s.publishLocked(j, 0, wrap)
}

func (s *executionJournalStore) publish(j executionJournal, wrap func(ownershipPublication) ownershipPublication) error {
	if err := s.guard(); err != nil {
		return err
	}
	s.lease.mu.Lock()
	defer s.lease.mu.Unlock()
	if s.failed || s.lease.publicationBlocked {
		return ErrStateUncertain
	}
	if !s.acknowledged {
		return ErrDenied
	}
	if err := checkExecutionJournalSuccessor(s.ack, j); err != nil {
		return err
	}
	snapshot, err := s.inspectLocked("", nil)
	if err == nil && (!snapshot.present || snapshot.record != s.ack || !bytes.Equal(snapshot.data, s.ackBytes) || !sameOwnershipMetadata(snapshot.metadata, s.ackMetadata)) {
		err = ErrDrift
	}
	if err != nil {
		s.retireLocked()
		return err
	}
	return s.publishLocked(j, s.ack.Generation, wrap)
}

func (s *executionJournalStore) retireLocked() {
	s.failed = true
	s.lease.publicationBlocked = true
}

// Called only under the original lease mutex after typed initial/successor checks.
// No semantic pending-state clearance is proved by this file component.
func (s *executionJournalStore) publishLocked(j executionJournal, expected uint64, wrap func(ownershipPublication) ownershipPublication) error {
	data, err := encodeExecutionJournal(j)
	if err != nil {
		return err
	}
	predecessor := func() error {
		// Staging already exists at the second predecessor check. Namespace
		// inspection belongs to the common pipeline, which knows its owned temp.
		current, err := s.lease.readStateBytesLocked(executionJournalName)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, s.ackBytes) {
			return ErrDrift
		}
		// The common pipeline binds original inode and full post-staging metadata.
		return nil
	}
	inspect := func(temporary string) (StateCatalog, error) {
		_, err := s.inspectLocked(temporary, nil)
		return StateCatalog{}, err
	}
	var acknowledged executionJournalSnapshot
	acknowledge := func(dir int, name string, data []byte, written unix.Stat_t) error {
		if err := verifyPublishedState(dir, name, data, written); err != nil {
			return err
		}
		var err error
		acknowledged, err = s.inspectLocked("", nil)
		if err != nil {
			return err
		}
		if !acknowledged.present || acknowledged.record != j || !bytes.Equal(acknowledged.data, data) || acknowledged.metadata.Dev != written.Dev || acknowledged.metadata.Ino != written.Ino {
			return ErrDrift
		}
		return nil
	}
	err = s.lease.publishStateAcknowledgedLocked(executionJournalName, executionJournalTemporaryPrefix, data, expected,
		func(StateCatalog) error { return nil }, predecessor, inspect, acknowledge, wrap)
	if err != nil {
		s.retireLocked()
		return err
	}
	// A lost/altered callback cannot advance the original volatile receipt.
	final, err := s.inspectLocked("", nil)
	if err != nil || !acknowledged.present || !final.present || final.record != j || !bytes.Equal(final.data, data) || !sameOwnershipMetadata(final.metadata, acknowledged.metadata) {
		s.retireLocked()
		return ErrStateUncertain
	}
	s.acknowledged = true
	s.initializable = false
	s.ack, s.ackMetadata = j, final.metadata
	s.ackBytes = append([]byte(nil), data...)
	return nil
}

type executionJournalSnapshot struct {
	present  bool
	record   executionJournal
	data     []byte
	metadata unix.Stat_t
}

// Complete bounded independent enumeration and actual file read. A publication
// may exclude only its inode-verified own temporary through the common pipeline.
// Caller holds lease.mu; callbacks are fixture-only and cannot set authority.
func (s *executionJournalStore) inspectLocked(temporary string, afterRead func()) (executionJournalSnapshot, error) {
	zero := executionJournalSnapshot{}
	if os.Geteuid() != 0 {
		return zero, ErrStateDirectory
	}
	if s.lease.propertyOrigin != s.lease {
		return zero, ErrDenied
	}
	dir, err := s.lease.ownershipDirectoryLocked()
	if err != nil {
		return zero, err
	}
	var before, opened, after unix.Stat_t
	if unix.Fstat(dir, &before) != nil {
		return zero, ErrStateDirectory
	}
	fd, err := unix.Openat(dir, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return zero, ErrStateDirectory
	}
	scan := os.NewFile(uintptr(fd), "storage-execution-journal-namespace")
	defer scan.Close()
	if unix.Fstat(fd, &opened) != nil || !sameOwnershipMetadata(before, opened) {
		return zero, ErrDrift
	}
	var names []string
	for {
		batch, e := scan.Readdirnames(4)
		if len(names)+len(batch) > 3 {
			return zero, ErrLimit
		}
		names = append(names, batch...)
		if e == io.EOF {
			break
		}
		if e != nil {
			return zero, ErrStateDirectory
		}
	}
	if scan.Close() != nil {
		return zero, ErrStateDirectory
	}
	present, err := executionJournalNames(names, temporary)
	if err != nil {
		return zero, err
	}
	snapshot := executionJournalSnapshot{present: present}
	if present {
		if unix.Fstatat(dir, executionJournalName, &snapshot.metadata, unix.AT_SYMLINK_NOFOLLOW) != nil {
			return zero, ErrStateDirectory
		}
		snapshot.data, err = s.lease.readStateBytesLocked(executionJournalName)
		if err != nil {
			return zero, err
		}
		snapshot.record, err = decodeExecutionJournal(snapshot.data)
		if err != nil {
			return zero, err
		}
	}
	if afterRead != nil {
		afterRead()
	}
	if present && (unix.Fstatat(dir, executionJournalName, &after, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(snapshot.metadata, after)) {
		return zero, ErrDrift
	}
	if _, err := s.lease.ownershipDirectoryLocked(); err != nil {
		return zero, err
	}
	if unix.Fstat(dir, &after) != nil || !sameOwnershipMetadata(before, after) {
		return zero, ErrDrift
	}
	return snapshot, nil
}
