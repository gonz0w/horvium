// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"os"
)

// InspectPropertyJob reads only this claim's original queued/applying job from
// a strict complete census under the original root lease. The digest binds all
// record names and presentation bytes. No record/handle substitution, restored
// authority, effect authorization or reservation release follows from success.
func (c *PropertyStateClaim) InspectPropertyJob(id string) (PropertyJob, [32]byte, error) {
	if os.Geteuid() != 0 {
		return PropertyJob{}, [32]byte{}, ErrStateDirectory
	}
	if err := ValidatePropertyJobSelection(id); err != nil {
		return PropertyJob{}, [32]byte{}, err
	}
	if c == nil || c.state == nil || c.identity == nil || id != c.identity.id {
		return PropertyJob{}, [32]byte{}, ErrDenied
	}
	s := c.state
	s.mu.Lock()
	defer s.mu.Unlock()
	return c.inspectPropertyJobLocked()
}

// Caller holds the original state's mutex and has checked actual kernel UID.
func (c *PropertyStateClaim) inspectPropertyJobLocked() (PropertyJob, [32]byte, error) {
	s := c.state
	if s.propertyOrigin != s || s.propertyAdmissions.active != c.identity {
		return PropertyJob{}, [32]byte{}, ErrDenied
	}
	if s.publicationBlocked {
		return PropertyJob{}, [32]byte{}, ErrStateUncertain
	}
	job, digest, err := s.inspectPropertyJobLocked(c.identity.id, func(v JobReservations, current PropertyJob) error {
		return s.propertyAdmissions.check(c.identity, v, current, current.job.Intent.PoolGUID)
	})
	if errors.Is(err, ErrStateUncertain) {
		s.publicationBlocked = true
	}
	return job, digest, err
}

// ReadOwnership selects only the original job's pool through the same claim.
// Complete claim/census reads bracket the existing descriptor-anchored reader
// while holding the root mutex. Policy and whole-ledger drift discard output.
// This supplies root metadata provenance, not native eligibility or recovery.
func (c *PropertyStateClaim) ReadOwnership(guid string) (Policy, error) {
	if os.Geteuid() != 0 {
		return Policy{}, ErrStateDirectory
	}
	if len(guid) > 20 || !validGUID(guid) {
		return Policy{}, ErrInvalid
	}
	if c == nil || c.state == nil || c.identity == nil {
		return Policy{}, ErrDenied
	}
	s := c.state
	s.mu.Lock()
	defer s.mu.Unlock()
	job, before, err := c.inspectPropertyJobLocked()
	if err != nil {
		return Policy{}, err
	}
	if job.job.Intent.PoolGUID != guid {
		return Policy{}, ErrDenied
	}
	p, err := s.readOwnershipLocked(guid)
	if err != nil {
		return Policy{}, err
	}
	_, after, err := c.inspectPropertyJobLocked()
	if err != nil {
		return Policy{}, err
	}
	if before != after {
		return Policy{}, ErrDrift
	}
	return p, nil
}
