// SPDX-License-Identifier: BSD-3-Clause
package storage

import "crypto/sha256"

// Fixed volatile bound, independent of durable store capacity. Exhaustion loses
// launch-readiness provenance, never a successful durable submission reply.
const maxPropertyAdmissionReceipts = 64

type propertyStateClaimIdentity struct {
	id                string
	receipt           [32]byte
	applyingAttempted bool
}

// Access only under StateDirectory.mu. No disk decoder populates this ledger.
type propertyAdmissionLedger struct {
	receipts map[string][32]byte
	active   *propertyStateClaimIdentity
}

// Called only after exact new initial submission publication and final sync.
// Canonical data has already passed the whole-record codec. Never replace an
// earlier receipt or evict it to admit a second lineage for the same identifier.
func (a *propertyAdmissionLedger) remember(id string, data []byte) {
	if len(a.receipts) >= maxPropertyAdmissionReceipts {
		return
	}
	if a.receipts == nil {
		a.receipts = make(map[string][32]byte)
	}
	if _, exists := a.receipts[id]; !exists {
		a.receipts[id] = sha256.Sum256(data)
	}
}

func (a *propertyAdmissionLedger) claim(v JobReservations, id string) (*propertyStateClaimIdentity, error) {
	if a.active != nil {
		return nil, ErrDenied
	}
	digest, ok := a.receipts[id]
	if !ok || !v.ready {
		return nil, ErrDenied
	}
	r, ok := v.records[id]
	if !ok || len(r.property.job.History) != 1 {
		return nil, ErrDenied
	}
	c := &propertyStateClaimIdentity{id: id, receipt: digest}
	if err := checkLivePropertyStateJobs(v, r.property, r.job.Intent.PoolGUID, c); err != nil {
		return nil, err
	}
	a.active = c
	return c, nil
}

func (a *propertyAdmissionLedger) check(c *propertyStateClaimIdentity, v JobReservations, current PropertyJob, guid string) error {
	if c == nil || a.active != c {
		return ErrDenied
	}
	return checkLivePropertyStateJobs(v, current, guid, c)
}

func checkLivePropertyStateJobs(v JobReservations, current PropertyJob, guid string, c *propertyStateClaimIdentity) error {
	if !v.ready || c == nil || !validGUID(guid) || current.job.ID != c.id || current.job.Intent.PoolGUID != guid {
		return ErrDenied
	}
	data, err := EncodePropertyJob(current)
	if err != nil {
		return err
	}
	if len(current.job.History) != 1 && (len(current.job.History) != 2 || current.job.History[1].State != JobApplying) {
		return ErrDenied
	}
	queued := PropertyJob{job: current.Job(), intent: current.Intent()}
	queued.job.History = queued.job.History[:1]
	initial, err := EncodePropertyJob(queued)
	if err != nil || sha256.Sum256(initial) != c.receipt {
		return ErrDrift
	}
	r, exists := v.records[c.id]
	if !exists {
		return ErrDenied
	}
	actual, err := EncodePropertyJob(r.property)
	if err != nil || sha256.Sum256(actual) != sha256.Sum256(data) {
		return ErrDrift
	}
	for id, other := range v.records {
		if id != c.id && (!other.job.Terminal() || other.job.Intent.PoolGUID == guid) {
			return ErrDenied
		}
	}
	return nil
}

// PropertyStateClaim is a state prerequisite tied to one freshly
// published initial job and the original open root lease. It has no constructor,
// persisted form, release or native execution method. Copies share the same
// identity and one exact applying-metadata publication attempt.
// It supplies no session, clock, expiry, native eligibility, executor quiescence
// or restored-state clearance. The original coordinator/backend must supply all
// those independently before any native control. Only FreeBSD can issue it.
type PropertyStateClaim struct {
	state    *StateDirectory
	identity *propertyStateClaimIdentity
}
