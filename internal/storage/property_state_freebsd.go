// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// CheckPropertyState verifies selected policy, complete private records, observed
// system backing capability and retained recovery holds under the root lease.
// Success is a read-only prerequisite, not admission, executor quiescence,
// validation of restored metadata or write eligibility. It repairs/releases
// nothing. Observed orphan uncertainty latches this lease's existing write block.
// Inputs must be trusted root policy and fresh complete native system collection.
func (s *StateDirectory) CheckPropertyState(p Policy, u PoolSystemUse) error {
	return s.checkPropertyState(p, u, defaultStateCatalogLimits, nil)
}

// Private reduced-bound/read seam for disposable fixtures, never a runtime knob.
func (s *StateDirectory) checkPropertyState(p Policy, u PoolSystemUse, limits stateCatalogLimits, afterRead func()) error {
	return s.checkPropertyStateClaim(p, u, limits, afterRead, nil, PropertyJob{})
}

func (s *StateDirectory) checkPropertyStateClaim(p Policy, u PoolSystemUse, limits stateCatalogLimits, afterRead func(), claim *PropertyStateClaim, current PropertyJob) error {
	if s == nil || os.Geteuid() != 0 {
		return ErrStateDirectory
	}
	expected, err := EncodeOwnershipRecord(p)
	if err != nil {
		return err
	}
	if len(p.Members) == 0 {
		return ErrDenied
	}
	if !u.initialized {
		return ErrUnavailable
	}
	b, err := newReservationBuilder(limits)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if claim != nil && s.propertyOrigin != s {
		return ErrStateDirectory
	}
	if s.publicationBlocked {
		return ErrStateUncertain
	}
	dir, err := s.ownershipDirectoryLocked()
	if err != nil {
		return err
	}
	before, err := propertyStateFilesystem(dir)
	if err != nil {
		return err
	}
	owners := map[string]bool{}
	found := false
	_, err = s.inspectStateRecordsLocked(limits, afterRead, "", func(entry stateRecordName, data []byte) error {
		if entry.jobID != "" {
			return b.add(data)
		}
		owners[entry.poolGUID] = true
		if entry.poolGUID == p.PoolGUID {
			got, e := DecodeOwnershipRecord(data)
			if e != nil {
				return e
			}
			canonical, e := EncodeOwnershipRecord(got)
			if e != nil {
				return e
			}
			if !bytes.Equal(expected, canonical) {
				return ErrDrift
			}
			found = true
		}
		return nil
	})
	if errors.Is(err, ErrStateUncertain) {
		s.publicationBlocked = true
	}
	if err != nil {
		return err
	}
	if !found {
		return ErrDenied
	}
	view := b.finish()
	if claim == nil {
		err = checkPropertyStateJobs(view, p.PoolGUID)
	} else if claim.state != s {
		err = ErrDenied
	} else {
		err = s.propertyAdmissions.check(claim.identity, view, current, p.PoolGUID)
	}
	if err != nil {
		return err
	}
	if err = checkPropertyStateBacking(before, u, owners); err != nil {
		return err
	}
	if _, err = s.ownershipDirectoryLocked(); err != nil {
		return err
	}
	after, err := propertyStateFilesystem(dir)
	if err != nil {
		return err
	}
	if before != after {
		return ErrDrift
	}
	return nil
}
func propertyStateFilesystem(fd int) (propertyStateBacking, error) {
	var fs unix.Statfs_t
	if unix.Fstatfs(fd, &fs) != nil {
		return propertyStateBacking{}, ErrStateDirectory
	}
	return propertyStateBacking{kind: unix.ByteSliceToString(fs.Fstypename[:]), source: unix.ByteSliceToString(fs.Mntfromname[:]), target: unix.ByteSliceToString(fs.Mntonname[:]), fsid: fs.Fsid.Val, flags: fs.Flags, readonly: fs.Flags&unix.MNT_RDONLY != 0, softupdates: fs.Flags&unix.MNT_SOFTDEP != 0}, nil
}
