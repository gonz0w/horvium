// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"reflect"
	"time"
)

// PropertyJob retains a property review and lifecycle as one immutable metadata
// unit. It cannot authorize work, restore a plan, or prove native eligibility.
// Native metadata access additionally requires the StateDirectory root lease.
type PropertyJob struct {
	job    Job
	intent PropertyIntent
}

// BindPropertyJob checks consistency, not admission. The initial queued event
// must follow capture and precede review expiry. Later history may outlive the
// review; an expired plan cannot authorize launch merely because metadata binds.
// Inputs must remain owned by the caller throughout the call.
func BindPropertyJob(j Job, i PropertyIntent) (PropertyJob, error) {
	result, err := normalizePropertyJob(PropertyJob{job: j, intent: i})
	if err != nil {
		return PropertyJob{}, err
	}
	if _, err = encodePropertyJob(result); err != nil {
		return PropertyJob{}, err
	}
	return result, nil
}

func (p PropertyJob) Job() Job {
	result := p.job
	result.History = append([]JobTransition(nil), p.job.History...)
	return result
}

func (p PropertyJob) Intent() PropertyIntent {
	result := p.intent
	result.review = p.intent.Review()
	return result
}

func normalizePropertyJob(p PropertyJob) (PropertyJob, error) {
	if err := validateJob(p.job); err != nil {
		return PropertyJob{}, err
	}
	i, err := normalizePropertyIntent(p.intent)
	if err != nil {
		return PropertyJob{}, err
	}
	selection := p.job.Intent
	at := time.UnixMilli(int64(p.job.History[0].At))
	if selection.PlanID != i.review.ID || selection.PolicyRevision != i.review.PolicyRevision || selection.PoolGUID != i.review.PoolGUID || selection.Change != i.review.Intent || at.Before(i.capturedAt) || !at.Before(i.review.ExpiresAt) {
		return PropertyJob{}, ErrStateRecord
	}
	return PropertyJob{job: p.Job(), intent: i}, nil
}

// TransitionPropertyJob appends ordinary validated lifecycle metadata, preserving
// the complete review binding. This is neither publication nor native execution.
func TransitionPropertyJob(p PropertyJob, state JobState, at uint64, reason, observationDigest string) (PropertyJob, error) {
	normalized, err := normalizePropertyJob(p)
	if err != nil {
		return PropertyJob{}, err
	}
	next, err := TransitionJob(normalized.job, state, at, reason, observationDigest)
	if err != nil {
		return PropertyJob{}, err
	}
	return BindPropertyJob(next, normalized.intent)
}

// ValidatePropertyJobSuccessor checks the entire immutable baseline and an exact
// single-event append. A future publisher must independently read its trusted
// current predecessor; caller-supplied records do not establish root provenance.
func ValidatePropertyJobSuccessor(previous, next PropertyJob) error {
	before, err := normalizePropertyJob(previous)
	if err != nil {
		return err
	}
	after, err := normalizePropertyJob(next)
	if err != nil {
		return err
	}
	if err = ValidateJobSuccessor(before.job, after.job); err != nil {
		return err
	}
	if !reflect.DeepEqual(before.intent, after.intent) {
		return ErrDrift
	}
	// Refuse successors that cannot fit their single whole-record budget.
	if _, err = encodePropertyJob(before); err != nil {
		return err
	}
	_, err = encodePropertyJob(after)
	return err
}
