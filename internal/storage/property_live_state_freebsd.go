// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"os"
)

// ClaimPropertyState issues one read-only readiness claim for an exact newly
// published queued submission on this live root lease. Stored metadata, matching
// retries after reopening and raw job publication cannot create its receipt.
// The complete census still refuses other pending work and selected-pool holds.
// A successful claim is retained until Close, even if never used. No native
// executor, eligibility promotion or recovery/reservation release is enabled.
func (s *StateDirectory) ClaimPropertyState(id string) (*PropertyStateClaim, error) {
	if s == nil || os.Geteuid() != 0 {
		return nil, ErrStateDirectory
	}
	if err := ValidatePropertyJobSelection(id); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.publicationBlocked {
		return nil, ErrStateUncertain
	}
	if _, err := s.ownershipDirectoryLocked(); err != nil {
		return nil, err
	}
	v, _, err := s.inspectJobReservationsLocked(defaultStateCatalogLimits, nil, "")
	if errors.Is(err, ErrStateUncertain) {
		s.publicationBlocked = true
	}
	if err != nil {
		return nil, err
	}
	if _, err = s.ownershipDirectoryLocked(); err != nil {
		return nil, err
	}
	c, err := s.propertyAdmissions.claim(v, id)
	if err != nil {
		return nil, err
	}
	return &PropertyStateClaim{state: s, identity: c}, nil
}

// CheckPropertyState retains the complete existing policy/record/backing checks,
// allowing only this claim's exact queued job or single applying successor as
// its own hold. current must be the trusted worker's entire expected record;
// it is compared with independent strict readback under the same root mutex.
// Policy/system inputs remain trusted fresh-native obligations, not evidence
// inferred from caller labels. Success supplies no authorization for effects.
func (c *PropertyStateClaim) CheckPropertyState(p Policy, u PoolSystemUse, current PropertyJob) error {
	if c == nil || c.state == nil {
		return ErrDenied
	}
	return c.state.checkPropertyStateClaim(p, u, defaultStateCatalogLimits, nil, c, current)
}
