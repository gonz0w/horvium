// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"math"
	"os"
	"time"
)

// Private metadata transaction, not commissioning/admission/execution authority.
// The public opener cannot obtain the original journal receipt required here.
// Future runtime callers must use the qualified registered startup/global owner.
type executionPublicationRecord struct {
	name, prefix string
	data         []byte
	expected     uint64
	policy       *Policy
	job          *PropertyJob
}
type executionPublicationPlan struct {
	before, after, previous string
}

func (p *ExecutionDomainLeases) publishJournaledOwnership(ctx context.Context, next Policy, expected uint64, read func(context.Context, int) (ExecutionBackingObservation, error), wrap func(string, ownershipPublication) ownershipPublication) error {
	data, e := EncodeOwnershipRecord(next)
	if e != nil {
		return e
	}
	if !validOwnershipPublicationRevision(next.Revision, expected) {
		return ErrDrift
	}
	owned, e := DecodeOwnershipRecord(data)
	if e != nil {
		return e
	}
	return p.publishJournaledRecord(ctx, executionPublicationRecord{"ownership-" + owned.PoolGUID + ".json", ".ownership-", data, expected, &owned, nil}, read, wrap)
}
func (p *ExecutionDomainLeases) publishJournaledPropertyJob(ctx context.Context, next PropertyJob, expected uint64, read func(context.Context, int) (ExecutionBackingObservation, error), wrap func(string, ownershipPublication) ownershipPublication) error {
	data, e := EncodePropertyJob(next)
	if e != nil {
		return e
	}
	if !validOwnershipPublicationRevision(uint64(len(next.job.History)), expected) {
		return ErrDrift
	}
	owned, e := DecodePropertyJob(data)
	if e != nil {
		return e
	}
	return p.publishJournaledRecord(ctx, executionPublicationRecord{"job-" + owned.job.ID + ".json", ".job-", data, expected, nil, &owned}, read, wrap)
}
func (r executionPublicationRecord) validate(c StateCatalog) error {
	if r.policy != nil {
		return allowMixedOwnershipPublication(c, *r.policy, r.expected, uint64(len(r.data)), defaultStateCatalogLimits)
	}
	if r.job != nil {
		return allowJobPublication(c, r.job.job, r.expected, uint64(len(r.data)), defaultStateCatalogLimits)
	}
	return ErrDenied
}
func (r executionPublicationRecord) predecessor(data []byte) error {
	if r.expected == 0 {
		if len(data) != 0 {
			return ErrDrift
		}
		return nil
	}
	if r.policy != nil {
		old, e := DecodeOwnershipRecord(data)
		if e != nil {
			return e
		}
		if old.PoolGUID != r.policy.PoolGUID || old.Revision != r.expected {
			return ErrDrift
		}
		return nil
	}
	if r.job != nil {
		old, e := DecodePropertyJob(data)
		if e != nil {
			return e
		}
		if uint64(len(old.job.History)) != r.expected {
			return ErrDrift
		}
		return ValidatePropertyJobSuccessor(old, *r.job)
	}
	return ErrDenied
}

// Both original lease mutexes are held throughout all three durable publications.
func (p *ExecutionDomainLeases) publishJournaledRecord(ctx context.Context, r executionPublicationRecord, read func(context.Context, int) (ExecutionBackingObservation, error), wrap func(string, ownershipPublication) ownershipPublication) error {
	if p == nil || ctx == nil || read == nil || os.Geteuid() != 0 {
		return ErrDenied
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.origin != p || p.closed || p.native == nil || !p.bound {
		return ErrDenied
	}
	if p.blocked {
		return ErrStateUncertain
	}
	j, l := p.native.journal, p.native.ledger
	if j == nil || l == nil || j.original != j || j.lease == nil || j.lease == l {
		return ErrDenied
	}
	j.lease.mu.Lock()
	defer j.lease.mu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	if !j.acknowledged {
		return ErrDenied
	}
	success := false
	defer func() {
		if !success {
			p.blocked = true
			j.retireLocked()
			l.publicationBlocked = true
		}
	}()
	if j.failed || j.lease.publicationBlocked || l.publicationBlocked {
		return ErrStateUncertain
	}
	if j.ack.Binding != p.binding || j.ack.Owner.State != "active" || j.ack.Disarmed || j.ack.Metadata.Pending {
		return ErrDenied
	}
	if j.ack.Generation > math.MaxUint64-2 {
		return ErrLimit
	}
	observe := func(expected executionJournal, census string) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		got, e := p.observeLocked(ctx, p.binding.Installation, read, nil)
		if e != nil {
			return e
		}
		if got.binding != p.binding || !got.journalPresent || got.journal != expected || got.census != census || expected != j.ack {
			return ErrDrift
		}
		receipt, e := j.inspectLocked("", nil)
		if e != nil {
			return e
		}
		if !receipt.present || receipt.record != j.ack || !bytes.Equal(receipt.data, j.ackBytes) || !sameOwnershipMetadata(receipt.metadata, j.ackMetadata) {
			return ErrDrift
		}
		return nil
	}
	before := j.ack
	if e := observe(before, before.Census); e != nil {
		return e
	}
	plan, e := l.prepareExecutionPublicationLocked(before.Binding, r)
	if e != nil {
		return e
	}
	if plan.before != before.Census {
		return ErrDrift
	}
	pending := before
	pending.Generation++
	pending.Metadata = executionJournalMetadata{true, plan.before, plan.after, r.name, plan.previous, executionRecordDigest(r.data)}
	confirmed := pending
	confirmed.Generation++
	confirmed.Census = plan.after
	confirmed.Metadata = executionJournalMetadata{}
	// Reserve bounded format/generation capacity for both journal transitions before
	// the first effect. Filesystem space/durability can still fail conservatively.
	if _, e = encodeExecutionJournal(pending); e != nil {
		return e
	}
	if _, e = encodeExecutionJournal(confirmed); e != nil {
		return e
	}
	seam := func(phase string) func(ownershipPublication) ownershipPublication {
		if wrap == nil {
			return nil
		}
		return func(tx ownershipPublication) ownershipPublication { return wrap(phase, tx) }
	}
	if e = j.publishSuccessorLocked(pending, seam("pending")); e != nil {
		return e
	}
	if e = observe(pending, plan.before); e != nil {
		return e
	}
	if e = l.publishStateLocked(r.name, r.prefix, r.data, r.expected, defaultStateCatalogLimits, r.validate, func() error {
		old, e := l.readStateBytesLocked(r.name)
		if e != nil {
			return e
		}
		oldDigest := ""
		if len(old) != 0 {
			oldDigest = executionRecordDigest(old)
		}
		if oldDigest != plan.previous {
			return ErrDrift
		}
		return r.predecessor(old)
	}, seam("ledger")); e != nil {
		return e
	}
	if e = observe(pending, plan.after); e != nil {
		return e
	}
	if e = j.publishSuccessorLocked(confirmed, seam("confirmation")); e != nil {
		return e
	}
	if e = observe(confirmed, plan.after); e != nil {
		return e
	}
	success = true
	return nil
}

func (s *StateDirectory) prepareExecutionPublicationLocked(binding executionJournalBinding, r executionPublicationRecord) (executionPublicationPlan, error) {
	var change *executionCensusChange
	var previous []byte
	catalog, e := s.inspectStateStreamLocked(defaultStateCatalogLimits, nil, "", func(names []string) error {
		var e error
		change, e = newExecutionCensusChange(binding, names, r.name, r.data, defaultStateCatalogLimits)
		return e
	}, func(entry stateRecordName, data []byte) error {
		if entry.name == r.name {
			previous = bytes.Clone(data)
		}
		return change.add(entry.name, data)
	})
	if e != nil {
		return executionPublicationPlan{}, e
	}
	if e = r.validate(catalog); e != nil {
		return executionPublicationPlan{}, e
	}
	if e = r.predecessor(previous); e != nil {
		return executionPublicationPlan{}, e
	}
	before, after, predecessor, e := change.finish()
	if e != nil {
		return executionPublicationPlan{}, e
	}
	return executionPublicationPlan{before, after, predecessor}, nil
}
