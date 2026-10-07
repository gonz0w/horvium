// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"os"
)

// Private reduced-bound/after-read seams are only for disposable tests. The
// public binder always uses validated defaults and the actual held root lease.
func (s *StateDirectory) propertyEnrollmentState(next, previous Policy, u PoolSystemUse, limits stateCatalogLimits, afterRead func()) ([32]byte, error) {
	var zero [32]byte
	if s == nil || os.Geteuid() != 0 {
		return zero, ErrStateDirectory
	}
	data, err := EncodeOwnershipRecord(next)
	if err != nil {
		return zero, err
	}
	if len(next.Members) == 0 || len(next.Scopes) == 0 {
		return zero, ErrDenied
	}
	canonical, err := enrollmentPredecessor(next, previous)
	if err != nil {
		return zero, err
	}
	var expected []byte
	if canonical.Revision != 0 {
		expected, err = EncodeOwnershipRecord(canonical)
		if err != nil {
			return zero, err
		}
	}
	if !u.initialized {
		return zero, ErrUnavailable
	}
	b, err := newReservationBuilder(limits)
	if err != nil {
		return zero, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.publicationBlocked {
		return zero, ErrStateUncertain
	}
	dir, err := s.ownershipDirectoryLocked()
	if err != nil {
		return zero, err
	}
	before, err := propertyStateFilesystem(dir)
	if err != nil {
		return zero, err
	}
	h := sha256.New()
	_, _ = h.Write([]byte("horvium-property-enrollment-state-v1\x00"))
	frame := func(data []byte) {
		_, _ = h.Write(binary.BigEndian.AppendUint64(nil, uint64(len(data))))
		_, _ = h.Write(data)
	}
	found := false
	catalog, err := s.inspectStateRecordsLocked(limits, afterRead, "", func(entry stateRecordName, current []byte) error {
		// Every byte has already passed strict decoding and descriptor/ACL checks.
		// Exact complete records, including unrelated policies/jobs, bind review.
		frame([]byte(entry.name))
		frame(current)
		if entry.jobID != "" {
			return b.add(current)
		}
		if entry.poolGUID != next.PoolGUID {
			return nil
		}
		if canonical.Revision == 0 {
			return ErrDrift
		}
		got, e := DecodeOwnershipRecord(current)
		if e != nil {
			return e
		}
		encoded, e := EncodeOwnershipRecord(got)
		if e != nil {
			return e
		}
		if !bytes.Equal(encoded, expected) {
			return ErrDrift
		}
		found = true
		return nil
	})
	if errors.Is(err, ErrStateUncertain) {
		s.publicationBlocked = true
	}
	if err != nil {
		return zero, err
	}
	if canonical.Revision != 0 && !found {
		return zero, ErrDrift
	}
	if err = checkEnrollmentStateCatalog(catalog, next, canonical, uint64(len(data)), limits, b.finish(), before, u); err != nil {
		return zero, err
	}
	if _, err = s.ownershipDirectoryLocked(); err != nil {
		return zero, err
	}
	after, err := propertyStateFilesystem(dir)
	if err != nil {
		return zero, err
	}
	if before != after {
		return zero, ErrDrift
	}
	// Internal hash framing, not a durable schema or an exported host inventory.
	frame([]byte(before.kind))
	frame([]byte(before.source))
	frame([]byte(before.target))
	_ = binary.Write(h, binary.BigEndian, before.fsid)
	_ = binary.Write(h, binary.BigEndian, before.flags)
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return digest, nil
}
