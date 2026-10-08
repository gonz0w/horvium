// SPDX-License-Identifier: BSD-3-Clause
package storage

// A read-only prerequisite for the original backend's control binding. A raw
// applying record or interrupted successor attempt cannot establish this exact
// successful applying acknowledgement. This consumes no native/control attempt.
func (a *propertyAdmissionLedger) checkApplyingAcknowledgement(c *propertyStateClaimIdentity, v JobReservations, current PropertyJob) error {
	if c == nil || !c.applyingAttempted || c.successorAttempted || c.acknowledgedRevision != 2 || len(current.job.History) != 2 || current.job.History[1].State != JobApplying {
		return ErrDenied
	}
	return a.checkAcknowledged(c, v, current, c.acknowledgedReceipt, c.acknowledgedRevision)
}
