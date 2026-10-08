// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// PropertyExecutionBackend is a trusted root-worker contract, not a transport or
// request adapter. Private native composition has no runtime consumer; full
// authentication/recovery/global qualification remains open. Prepare must
// independently read the current durably admitted job, claim its ORIGINAL live
// plan/session, establish reservations, root/store/recovery/executor safety and
// collect fresh policy/eligible native facts. A decoded job cannot supply these.
// Refresh must repeat those checks through the same claim/lease and verify the
// exact durable applying predecessor. It must never restore a claim from disk.
// Publish must durably publish one exact successor under the original root lease.
// Apply accepts one typed immutable property intent, resolves/rechecks the actual
// GUID/address, uses fixed allowlisted arguments and reports quiescence only after
// its sole native child/guard is verified gone. A timeout is not quiescence.
// Observe independently collects fresh complete policy/native facts after Apply.
// Calls must be bounded; the backend owns global launch/pool serialization.
type PropertyExecutionBackend interface {
	PreparePropertyExecution(context.Context, string) (PropertyJob, Policy, Observation, error)
	RefreshPropertyExecution(context.Context, PropertyJob) (Policy, Observation, error)
	PublishPropertyExecution(PropertyJob, uint64) (PropertyJob, error)
	ApplyProperty(context.Context, PropertyIntent) (bool, error)
	ObservePropertyExecution(context.Context, PropertyIntent) (Policy, Observation, error)
}

// PropertyExecution owns exactly one worker attempt, not a scheduler or native
// capability. It starts no goroutine/process and owns no file/lease. Do not copy.
// Reconstruction cannot restore execution authority: the backend still needs
// the original live claim and current durable predecessor. Every valid attempt,
// including a preflight denial, consumes this object; Run never automatically
// retries. Its mutex excludes concurrent calls only on this object.
type PropertyExecution struct {
	mu        sync.Mutex
	backend   PropertyExecutionBackend
	clock     func() time.Time
	last      time.Time
	attempted bool
	// Only an original coordinator handoff installs lifecycle revocation.
	finish func()
}

func NewPropertyExecution(backend PropertyExecutionBackend) *PropertyExecution {
	return &PropertyExecution{backend: backend, clock: time.Now}
}

func (e *PropertyExecution) now(previous PropertyJob) (time.Time, error) {
	at := e.clock().Round(0).UTC()
	if !canonicalIntentTime(at) || at.Before(e.last) {
		return time.Time{}, ErrExpired
	}
	e.last = at
	if len(previous.job.History) > 0 && at.UnixMilli() < int64(previous.job.History[len(previous.job.History)-1].At) {
		return time.Time{}, ErrExpired
	}
	return at, nil
}

// Run must receive a root worker lifetime context, not an HTTP request context.
// Before applying acknowledgement, denial returns zero (publication uncertainty
// returns the verified queued predecessor). Afterwards an error returns the last
// EXACT acknowledged record: never fabricate a durable outcome or hide applying.
// A reconcile-required result remains nonterminal and retains its reservation.
// An error/foreign acknowledgement at any publication stops further publication.
// Metadata repair, cancellation, hold release and recovery resolution are absent.
func (e *PropertyExecution) Run(ctx context.Context, id string) (PropertyJob, error) {
	if err := coordinatorContext(ctx); err != nil {
		return PropertyJob{}, err
	}
	if !validJobDigest(id) {
		return PropertyJob{}, ErrInvalid
	}
	if e == nil {
		return PropertyJob{}, ErrUnavailable
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.backend == nil || e.clock == nil {
		return PropertyJob{}, ErrUnavailable
	}
	if e.attempted {
		return PropertyJob{}, ErrDenied
	}
	e.attempted = true
	if e.finish != nil {
		defer e.finish()
	}
	if _, err := e.now(PropertyJob{}); err != nil {
		return PropertyJob{}, err
	}
	job, policy, observation, err := e.backend.PreparePropertyExecution(ctx, id)
	if err != nil {
		return PropertyJob{}, err
	}
	job, err = normalizePropertyJob(job)
	if err != nil {
		return PropertyJob{}, err
	}
	if job.job.ID != id || len(job.job.History) != 1 {
		return PropertyJob{}, ErrDenied
	}
	if err = e.prelaunch(ctx, job, policy, observation); err != nil {
		return PropertyJob{}, err
	}
	latest := clonePropertyJob(job)
	// This closure updates latest ONLY after an exact durable acknowledgement.
	publish := func(state JobState, reason, digest string) error {
		at, err := e.now(latest)
		if err != nil {
			return err
		}
		next, err := TransitionPropertyJob(latest, state, uint64(at.UnixMilli()), reason, digest)
		if err != nil {
			return err
		}
		ack, err := e.backend.PublishPropertyExecution(clonePropertyJob(next), uint64(len(latest.job.History)))
		if err != nil {
			return errors.Join(ErrStateUncertain, err)
		}
		expected, err := EncodePropertyJob(next)
		if err != nil {
			return errors.Join(ErrStateUncertain, err)
		}
		actual, err := EncodePropertyJob(ack)
		if err != nil || !bytes.Equal(expected, actual) {
			return ErrStateUncertain
		}
		latest = clonePropertyJob(next)
		return nil
	}
	if err = publish(JobApplying, "launch-intent", ""); err != nil {
		return latest, err
	}
	reconcile := func(reason string, cause error) (PropertyJob, error) {
		digest := latest.job.History[len(latest.job.History)-1].ObservationDigest
		if err := publish(JobReconcileRequired, reason, digest); err != nil {
			return latest, errors.Join(cause, err)
		}
		return latest, cause
	}
	if err = coordinatorContext(ctx); err != nil {
		return reconcile("executor-unverified", err)
	}
	policy, observation, err = e.backend.RefreshPropertyExecution(ctx, clonePropertyJob(latest))
	if err != nil {
		return reconcile("executor-unverified", err)
	}
	if err = e.prelaunch(ctx, latest, policy, observation); err != nil {
		return reconcile("executor-unverified", err)
	}
	quiet, err := e.backend.ApplyProperty(ctx, latest.Intent())
	if !quiet {
		return reconcile("executor-unverified", errors.Join(ErrUnavailable, err))
	}
	if err != nil {
		return reconcile("ambiguous-effect", err)
	}
	if err = coordinatorContext(ctx); err != nil {
		return reconcile("storage-unavailable", err)
	}
	policy, observation, err = e.backend.ObservePropertyExecution(ctx, latest.Intent())
	if err != nil {
		return reconcile("storage-unavailable", err)
	}
	if err = coordinatorContext(ctx); err != nil {
		return reconcile("storage-unavailable", err)
	}
	digest, err := PropertyExecutionObservationDigest(latest.intent, policy, observation)
	if err != nil {
		return reconcile("storage-unavailable", err)
	}
	matched := latest.intent.ComparePostcondition(policy, observation) == nil
	if err = publish(JobObserving, "native-observation", digest); err != nil {
		return latest, err
	}
	if !matched {
		return reconcile("ambiguous-effect", ErrDrift)
	}
	// Expiry after launch does not erase an actual verified durable result. No
	// new operation is authorized here; worker cancellation during publication
	// also cannot turn a successful durable acknowledgement into a replay cue.
	if err = publish(JobSucceeded, "verified", digest); err != nil {
		return latest, err
	}
	return latest, nil
}

func (e *PropertyExecution) prelaunch(ctx context.Context, p PropertyJob, policy Policy, o Observation) error {
	if err := coordinatorContext(ctx); err != nil {
		return err
	}
	at, err := e.now(p)
	if err != nil {
		return err
	}
	if !at.Before(p.intent.review.ExpiresAt) {
		return ErrExpired
	}
	if err = p.intent.CompareBaseline(policy, o); err != nil {
		return err
	}
	// Comparison is bounded but can consume time. Recheck the worker clock and
	// cancellation after it, immediately before publication/launch.
	if err = coordinatorContext(ctx); err != nil {
		return err
	}
	at, err = e.now(p)
	if err != nil {
		return err
	}
	if !at.Before(p.intent.review.ExpiresAt) {
		return ErrExpired
	}
	return nil
}

// PropertyExecutionObservationDigest hashes the validated, bounded, sorted
// relevant projection and full policy digest. It does not match an expected
// effect or authenticate facts. No digest supplies provenance or quiescence. Unrelated
// sibling properties remain outside the accepted intent's comparison scope.
func PropertyExecutionObservationDigest(i PropertyIntent, p Policy, o Observation) (string, error) {
	if err := preflight(p, o); err != nil {
		return "", err
	}
	files, err := indexCollectedObservation(o)
	if err != nil {
		return "", err
	}
	r, d, err := prepareIndexed(p, o, i.review.Intent, files)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(struct {
		Policy      [32]byte
		Observation Review
	}{d, r})
	if err != nil {
		return "", err
	}
	if len(data) > maxStateRecordBytes {
		return "", ErrLimit
	}
	digest := sha256.Sum256(append([]byte("horvium:property-execution-observation:v1\n"), data...))
	return hex.EncodeToString(digest[:]), nil
}
