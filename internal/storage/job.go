// SPDX-License-Identifier: BSD-3-Clause
package storage

import "strings"

type JobState string

const (
	JobQueued            JobState = "queued"
	JobApplying          JobState = "applying"
	JobObserving         JobState = "observing"
	JobSucceeded         JobState = "succeeded"
	JobFailed            JobState = "failed"
	JobCancelled         JobState = "cancelled"
	JobReconcileRequired JobState = "reconcile-required"
	maxJobTransitions             = 8
)

// JobIntent retains the existing property-change selection and opaque review/
// submission identifiers. It is not a complete executable/recoverable plan,
// authorization, reservation, or proof that the selected native resources exist.
type JobIntent struct {
	PlanID, SubmissionID string
	PolicyRevision       uint64
	PoolGUID             string
	Change               Intent
}

// At records UTC Unix milliseconds. ObservationDigest identifies a later
// independently verified observation; its presence is not native proof.
type JobTransition struct {
	Revision, At              uint64
	State                     JobState
	Reason, ObservationDigest string
}

// Job is lifecycle metadata only. No runtime, durable admission or executor
// consumes it yet. History is retained whole and never truncated to fit.
type Job struct {
	ID      string
	Intent  JobIntent
	History []JobTransition
}

// Terminal refuses malformed history. Reconcile-required remains nonterminal;
// this classification alone never authorizes eviction or reservation release.
func (j Job) Terminal() bool {
	if validateJob(j) != nil {
		return false
	}
	state := j.History[len(j.History)-1].State
	return state == JobSucceeded || state == JobFailed || state == JobCancelled
}

// TransitionJob returns independent metadata, not a durable acknowledgement.
// Only pre-launch cancellation is represented. Reconciliation resolution stays
// disabled until its native attribution/quiescence gates are implemented.
func TransitionJob(j Job, state JobState, at uint64, reason, observationDigest string) (Job, error) {
	if err := validateJob(j); err != nil {
		return Job{}, err
	}
	if len(j.History) == maxJobTransitions {
		return Job{}, ErrLimit
	}
	next := j
	next.History = make([]JobTransition, len(j.History), len(j.History)+1)
	copy(next.History, j.History)
	next.History = append(next.History, JobTransition{Revision: j.History[len(j.History)-1].Revision + 1, At: at, State: state, Reason: reason, ObservationDigest: observationDigest})
	if err := validateJob(next); err != nil {
		return Job{}, err
	}
	return next, nil
}

// A restart projection never returns queued/applying/observing for nonterminal
// work. It changes no record, claims no quiescence, frees no reservation and
// does not publish recovery. Retain original observations/history separately.
func jobRestartProjection(j Job) (JobState, string, error) {
	if err := validateJob(j); err != nil {
		return "", "", err
	}
	last := j.History[len(j.History)-1]
	switch last.State {
	case JobQueued:
		return JobReconcileRequired, "review-expired", nil
	case JobApplying, JobObserving:
		return JobReconcileRequired, "executor-unverified", nil
	default:
		return last.State, last.Reason, nil
	}
}

func validJobDigest(s string) bool {
	return len(s) == 64 && strings.IndexFunc(s, func(r rune) bool { return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') }) < 0
}

func validateJob(j Job) error {
	if len(j.History) > maxJobTransitions {
		return ErrLimit
	}
	for _, field := range []string{j.ID, j.Intent.PlanID, j.Intent.SubmissionID, j.Intent.PoolGUID, j.Intent.Change.FilesystemGUID, string(j.Intent.Change.Property), j.Intent.Change.Value} {
		if len(field) > maxFieldBytes {
			return ErrLimit
		}
	}
	if !validJobDigest(j.ID) || !validJobDigest(j.Intent.PlanID) || !validJobDigest(j.Intent.SubmissionID) || j.Intent.PolicyRevision == 0 || !validGUID(j.Intent.PoolGUID) || !validIntent(j.Intent.Change) || len(j.History) == 0 {
		return ErrStateRecord
	}
	for i, event := range j.History {
		if len(event.Reason) > maxFieldBytes || len(event.ObservationDigest) > maxFieldBytes {
			return ErrLimit
		}
		// Restrict timestamps to the civil UTC range through year 9999. They
		// cannot establish freshness or deadlines without a trusted clock.
		if event.Revision != uint64(i+1) || event.At == 0 || event.At > 253402300799999 {
			return ErrStateRecord
		}
		if i == 0 {
			if event.State != JobQueued || event.Reason != "admitted" || event.ObservationDigest != "" {
				return ErrStateRecord
			}
			continue
		}
		previous := j.History[i-1]
		if event.At < previous.At || !validJobEdge(previous.State, event.State) {
			return ErrStateRecord
		}
		switch event.State {
		case JobApplying:
			if event.Reason != "launch-intent" || event.ObservationDigest != "" {
				return ErrStateRecord
			}
		case JobObserving:
			if event.Reason != "native-observation" || !validJobDigest(event.ObservationDigest) {
				return ErrStateRecord
			}
		case JobSucceeded:
			if event.Reason != "verified" || !validJobDigest(event.ObservationDigest) {
				return ErrStateRecord
			}
		case JobFailed:
			if event.Reason != "verified-failure" || !validJobDigest(event.ObservationDigest) {
				return ErrStateRecord
			}
		case JobCancelled:
			if event.Reason != "operator-cancelled" || event.ObservationDigest != "" {
				return ErrStateRecord
			}
		case JobReconcileRequired:
			if previous.State == JobQueued {
				if event.Reason != "review-expired" {
					return ErrStateRecord
				}
			} else {
				switch event.Reason {
				case "ambiguous-effect", "storage-unavailable", "executor-unverified", "state-io-failure":
				default:
					return ErrStateRecord
				}
			}
			if event.ObservationDigest != previous.ObservationDigest {
				return ErrStateRecord
			}
		default:
			return ErrStateRecord
		}
	}
	return nil
}

func validJobEdge(from, to JobState) bool {
	switch from {
	case JobQueued:
		return to == JobApplying || to == JobCancelled || to == JobReconcileRequired
	case JobApplying:
		return to == JobObserving || to == JobReconcileRequired
	case JobObserving:
		return to == JobSucceeded || to == JobFailed || to == JobReconcileRequired
	default:
		return false
	}
}
