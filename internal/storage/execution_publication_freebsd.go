// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
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
	return ErrDenied
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
