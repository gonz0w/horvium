// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// PublishPropertySubmission checks complete retained plan/submission/pool holds
// and publishes one initial queued property record while holding the root lease's
// mutex. New records return only after file/rename/directory sync. A matched retry
// returns owned existing current metadata without rewriting or re-syncing it;
// this is not recovery validation or a fresh durability acknowledgement.
//
// This metadata primitive has no runtime consumer and authorizes no native work.
// Live caller/session/clock/policy/eligibility and recovery gates remain required
// before executable admission. Terminal records retain holds. Any error returns
// zero/false, including a visible record after uncertain final directory sync.
func (s *StateDirectory) PublishPropertySubmission(p PropertyJob) (PropertyJob, bool, error) {
	return s.publishPropertySubmissionBounded(p, defaultStateCatalogLimits, nil)
}

// Reduced limits/phase wrapper are disposable fixture seams, not runtime policy.
func (s *StateDirectory) publishPropertySubmissionBounded(p PropertyJob, limits stateCatalogLimits, wrap func(ownershipPublication) ownershipPublication) (PropertyJob, bool, error) {
	if s == nil || os.Geteuid() != 0 {
		return PropertyJob{}, false, ErrStateDirectory
	}
	bound, err := BindPropertyJob(p.job, p.intent)
	if err != nil {
		return PropertyJob{}, false, err
	}
	if len(bound.job.History) != 1 {
		return PropertyJob{}, false, ErrStateRecord
	}
	data, err := EncodePropertyJob(bound)
	if err != nil {
		return PropertyJob{}, false, err
	}
	if !limits.valid() {
		return PropertyJob{}, false, ErrLimit
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.publicationBlocked {
		return PropertyJob{}, false, ErrStateUncertain
	}
	dir, err := s.ownershipDirectoryLocked()
	if err != nil {
		return PropertyJob{}, false, err
	}
	var fs unix.Statfs_t
	if unix.Fstatfs(dir, &fs) != nil || unix.ByteSliceToString(fs.Fstypename[:]) == "ufs" && fs.Flags&unix.MNT_SOFTDEP != 0 {
		return PropertyJob{}, false, ErrStateDirectory
	}
	view, _, err := s.inspectJobReservationsLocked(limits, nil, "")
	if errors.Is(err, ErrStateUncertain) {
		s.publicationBlocked = true
	}
	if err != nil {
		return PropertyJob{}, false, err
	}
	current, duplicate, err := view.CheckPropertySubmission(bound)
	if err != nil || duplicate {
		return current, duplicate, err
	}
	inspect := func(temporary string) (StateCatalog, error) {
		view, catalog, err := s.inspectJobReservationsLocked(limits, nil, temporary)
		if err != nil {
			return StateCatalog{}, err
		}
		_, duplicate, err := view.CheckPropertySubmission(bound)
		// A newly appearing holder/retry after the first check is drift, never
		// permission to overwrite it or report this unpublished candidate.
		if duplicate || errors.Is(err, ErrDenied) {
			return StateCatalog{}, ErrDrift
		}
		if err != nil {
			return StateCatalog{}, err
		}
		return catalog, nil
	}
	err = s.publishStateInspectedLocked("job-"+bound.job.ID+".json", ".job-", data, 0,
		func(c StateCatalog) error { return allowJobPublication(c, bound.job, 0, uint64(len(data)), limits) },
		func() error { return ErrDrift }, inspect, wrap)
	if err != nil {
		return PropertyJob{}, false, err
	}
	if s.propertyOrigin == s {
		s.propertyAdmissions.remember(bound.job.ID, data)
	}
	return bound, false, nil
}
