// SPDX-License-Identifier: BSD-3-Clause
package storage

// Metadata validation only. The caller must independently establish the complete
// root/native/session/clock/executor boundary before authorizing any effect.
func (a *propertyAdmissionLedger) validateApplying(c *propertyStateClaimIdentity, v JobReservations, current, next PropertyJob) error {
	if err := a.check(c, v, current, current.job.Intent.PoolGUID); err != nil {
		return err
	}
	if len(current.job.History) != 1 || len(next.job.History) != 2 || next.job.History[1].State != JobApplying {
		return ErrDenied
	}
	return ValidatePropertyJobSuccessor(current, next)
}
