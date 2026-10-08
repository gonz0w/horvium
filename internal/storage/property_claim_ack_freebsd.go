// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"os"
)

// VerifyPropertyApplyingAcknowledgement independently verifies an exact current
// applying record against this original claim's private post-sync receipt and
// complete strict census. Raw publication, visible but lost acknowledgements,
// reopened leases, foreign handles and later/attempted successors cannot qualify.
// This proves metadata lineage only: no session, executor, eligibility, effect
// authorization, global serialization, control claim or hold release is supplied.
func (c *PropertyStateClaim) VerifyPropertyApplyingAcknowledgement(current PropertyJob) error {
	if os.Geteuid() != 0 {
		return ErrStateDirectory
	}
	if _, err := EncodePropertyJob(current); err != nil {
		return err
	}
	if c == nil || c.state == nil || c.identity == nil {
		return ErrDenied
	}
	s := c.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.propertyOrigin != s || s.propertyAdmissions.active != c.identity {
		return ErrDenied
	}
	if s.publicationBlocked {
		return ErrStateUncertain
	}
	_, _, err := s.inspectPropertyJobLocked(c.identity.id, func(v JobReservations, _ PropertyJob) error {
		return s.propertyAdmissions.checkApplyingAcknowledgement(c.identity, v, current)
	})
	if errors.Is(err, ErrStateUncertain) {
		s.publicationBlocked = true
	}
	return err
}
