// SPDX-License-Identifier: BSD-3-Clause
package storage

import "crypto/sha256"

// Volatile acknowledgement lineage is populated only after the original claim's
// publisher has synced and independently reread exact bytes/census. It proves
// metadata provenance, never control attribution, eligibility or hold release.
func (a *propertyAdmissionLedger) checkAcknowledged(c *propertyStateClaimIdentity, v JobReservations, current PropertyJob, receipt [32]byte, revision uint64) error {
	if c == nil || a.active != c || !v.ready || receipt == ([32]byte{}) || revision < 2 || current.job.ID != c.id || uint64(len(current.job.History)) != revision {
		return ErrDenied
	}
	data, err := EncodePropertyJob(current)
	if err != nil {
		return err
	}
	if sha256.Sum256(data) != receipt {
		return ErrDrift
	}
	r, exists := v.records[c.id]
	if !exists {
		return ErrDenied
	}
	actual, err := EncodePropertyJob(r.property)
	if err != nil || sha256.Sum256(actual) != receipt {
		return ErrDrift
	}
	for id, other := range v.records {
		if id != c.id && (!other.job.Terminal() || other.job.Intent.PoolGUID == current.job.Intent.PoolGUID) {
			return ErrDenied
		}
	}
	return nil
}

func (a *propertyAdmissionLedger) validateSuccessor(c *propertyStateClaimIdentity, v JobReservations, current, next PropertyJob) error {
	if c == nil {
		return ErrDenied
	}
	if err := a.checkAcknowledged(c, v, current, c.acknowledgedReceipt, c.acknowledgedRevision); err != nil {
		return err
	}
	history := current.job.History
	state := next.job.History
	if len(state) != len(history)+1 {
		return ErrDenied
	}
	last := state[len(state)-1].State
	if len(history) == 2 && history[1].State == JobApplying {
		if last != JobObserving && last != JobReconcileRequired {
			return ErrDenied
		}
	} else if len(history) == 3 && history[2].State == JobObserving {
		if last != JobSucceeded && last != JobReconcileRequired {
			return ErrDenied
		}
	} else {
		return ErrDenied
	}
	return ValidatePropertyJobSuccessor(current, next)
}
