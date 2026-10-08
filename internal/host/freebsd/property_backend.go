// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// Private integration; no runtime consumer. Clearance is a trusted helper-side
// obligation (kernel peer/session, recovery and global launch), never client data.
// Only disposable fixtures supply it until those gates are implemented/qualified.
// One instance owns one admission attempt/control and borrows the original lease.
type propertyBackend struct {
	mu        sync.Mutex
	guid      string
	root      func() bool
	clock     func() time.Time
	clearance func(context.Context) error
	initial   func(context.Context, string) (storage.Policy, storage.Observation, error)
	admit     func(storage.PropertyJob) (storage.PropertyJob, bool, error)
	claim     func(string) (*propertyBackendClaim, error)

	last                                   time.Time
	blocked, clockBlocked                  bool
	admissionAttempted, bindingAttempted   bool
	controlAttempted, observationAttempted bool
	current                                storage.PropertyJob
	source                                 *propertyBackendClaim
	authority                              *storage.PropertyExecutionAuthority
	control                                *propertyControl
	controlErr                             error
	quiet, matched                         bool
	digest                                 string
}

// Private seams are portable tests only. The native factory binds every method
// to its same original opaque state claim, independent source and fixed control.
type propertyBackendClaim struct {
	collect   func(context.Context, string) (storage.PropertyJob, storage.Policy, storage.Observation, error)
	verify    func(storage.PropertyJob) error
	applying  func(storage.PropertyJob) (storage.PropertyJob, error)
	successor func(storage.PropertyJob) (storage.PropertyJob, error)
	control   func(storage.PropertyJob, func(context.Context) error) (*propertyControl, error)
}

func newPropertyBackend(state *storage.StateDirectory, guid string, clearance func(context.Context) error) (*propertyBackend, error) {
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	if os.Geteuid() != 0 {
		return nil, ErrPermission
	}
	if state == nil || clearance == nil {
		return nil, storage.ErrDenied
	}
	if err := storage.ValidatePoolSelection(guid, "selection"); err != nil {
		return nil, err
	}
	initial, err := NewPropertyStateSource(state)
	if err != nil {
		return nil, err
	}
	b := &propertyBackend{guid: guid, root: func() bool { return os.Geteuid() == 0 }, clock: time.Now, clearance: clearance, initial: initial.Collect, admit: state.PublishPropertySubmission}
	b.claim = func(id string) (*propertyBackendClaim, error) {
		claim, err := state.ClaimPropertyState(id)
		if err != nil {
			return nil, err
		}
		source, err := NewPropertyClaimSource(claim)
		if err != nil {
			return nil, err
		}
		return &propertyBackendClaim{collect: source.Collect, verify: claim.VerifyPropertyApplyingAcknowledgement, applying: claim.PublishApplyingPropertyJob, successor: claim.PublishPropertyJobSuccessor, control: func(j storage.PropertyJob, live func(context.Context) error) (*propertyControl, error) {
			return newPropertyClaimControl(claim, j, live)
		}}, nil
	}
	return b, nil
}

// Never wait for backend.mu while a coordinator may hold its own mutex. Native
// callbacks consult that coordinator; busy admission/read calls fail promptly.
func (b *propertyBackend) acquire(ctx context.Context) error {
	if ctx == nil {
		return storage.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if b == nil {
		return ErrUnavailable
	}
	if !b.mu.TryLock() {
		return storage.ErrDenied
	}
	if err := b.check(ctx, false); err != nil {
		b.mu.Unlock()
		return err
	}
	return nil
}
func (b *propertyBackend) check(ctx context.Context, launch bool) error {
	if ctx == nil {
		return storage.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if b.root == nil || b.clock == nil || b.clearance == nil {
		return ErrUnavailable
	}
	if !b.root() {
		return ErrPermission
	}
	if b.blocked {
		return storage.ErrStateUncertain
	}
	if b.clockBlocked {
		return storage.ErrExpired
	}
	if err := b.checkClock(launch); err != nil {
		return err
	}
	if launch {
		if b.authority != nil {
			if err := b.authority.CheckLive(ctx); err != nil {
				return err
			}
		}
		if err := b.clearance(ctx); err != nil {
			return err
		}
		// Clearance can take time or observe new uncertainty. It cannot extend
		// the original coordinator/clock/root window without a final check.
		if !b.root() {
			return ErrPermission
		}
		if b.authority != nil {
			if err := b.authority.CheckLive(ctx); err != nil {
				return err
			}
		}
		if err := b.checkClock(true); err != nil {
			return err
		}
	}
	return ctx.Err()
}
func (b *propertyBackend) checkClock(launch bool) error {
	at := b.clock().Round(0).UTC()
	if at.IsZero() || at.Year() < 1970 || at.Year() > 9999 || at.Before(b.last) {
		b.clockBlocked = true
		return storage.ErrExpired
	}
	b.last = at
	if launch && b.current.Job().ID != "" {
		r := b.current.Intent().Review()
		if at.Before(r.CreatedAt) || !at.Before(r.ExpiresAt) {
			return storage.ErrExpired
		}
	}
	return nil
}

func ownBackendJob(j storage.PropertyJob) (storage.PropertyJob, error) {
	data, err := storage.EncodePropertyJob(j)
	if err != nil {
		return storage.PropertyJob{}, err
	}
	return storage.DecodePropertyJob(data)
}
func sameBackendJob(a, b storage.PropertyJob) bool {
	x, e := storage.EncodePropertyJob(a)
	y, f := storage.EncodePropertyJob(b)
	return e == nil && f == nil && bytes.Equal(x, y)
}
func (b *propertyBackend) sameIntent(i storage.PropertyIntent) bool {
	x, e := storage.EncodePropertyIntent(i)
	y, f := storage.EncodePropertyIntent(b.current.Intent())
	return e == nil && f == nil && bytes.Equal(x, y)
}
func (s *propertyBackendClaim) ready() bool {
	return s != nil && s.collect != nil && s.verify != nil && s.applying != nil && s.successor != nil && s.control != nil
}

func (b *propertyBackend) ObserveProperty(ctx context.Context, i storage.Intent) (storage.Policy, storage.Observation, error) {
	zero := func(err error) (storage.Policy, storage.Observation, error) {
		return storage.Policy{}, storage.Observation{}, err
	}
	if err := b.acquire(ctx); err != nil {
		return zero(err)
	}
	defer b.mu.Unlock()
	if b.admissionAttempted || b.initial == nil {
		return zero(storage.ErrDenied)
	}
	if err := b.check(ctx, true); err != nil {
		return zero(err)
	}
	p, o, err := b.initial(ctx, b.guid)
	if err != nil {
		return zero(err)
	}
	if p.PoolGUID != b.guid || o.PoolGUID != b.guid || o.Eligible {
		return zero(storage.ErrDrift)
	}
	if err = storage.ValidatePropertyPolicyBinding(p, o); err != nil {
		return zero(err)
	}
	if err = b.check(ctx, true); err != nil {
		return zero(err)
	}
	// Private trusted-backend output only, after native/store and clearance gates.
	// Independent public sources keep false eligibility. No runtime uses this path.
	o.Eligible = true
	return p, o, nil
}
func (b *propertyBackend) PublishPropertySubmission(candidate storage.PropertyJob) (storage.PropertyJob, bool, error) {
	ctx := context.Background()
	zero := func(err error) (storage.PropertyJob, bool, error) { return storage.PropertyJob{}, false, err }
	if err := b.acquire(ctx); err != nil {
		return zero(err)
	}
	defer b.mu.Unlock()
	if b.admissionAttempted || b.initial == nil || b.admit == nil || b.claim == nil {
		return zero(storage.ErrDenied)
	}
	candidate, err := ownBackendJob(candidate)
	if err != nil {
		return zero(err)
	}
	if candidate.Job().Intent.PoolGUID != b.guid || len(candidate.Job().History) != 1 {
		return zero(storage.ErrDenied)
	}
	b.admissionAttempted = true
	b.current = candidate
	if err = b.check(ctx, true); err != nil {
		return zero(err)
	}
	p, o, err := b.initial(ctx, b.guid)
	if err != nil {
		return zero(err)
	}
	if o.Eligible {
		return zero(storage.ErrDrift)
	}
	o.Eligible = true
	if err = candidate.Intent().CompareBaseline(p, o); err != nil {
		return zero(err)
	}
	if err = b.check(ctx, true); err != nil {
		return zero(err)
	}
	ack, duplicate, err := b.admit(candidate)
	if err != nil || (!duplicate && !sameBackendJob(candidate, ack)) {
		b.blocked = true
		return zero(errors.Join(storage.ErrStateUncertain, err))
	}
	ack, err = ownBackendJob(ack)
	if err != nil {
		b.blocked = true
		return zero(storage.ErrStateUncertain)
	}
	if duplicate {
		return ack, true, nil
	} // Metadata retry cannot install fresh authority.
	b.current = ack
	b.source, err = b.claim(ack.Job().ID)
	if err != nil || !b.source.ready() {
		b.blocked = true
		return zero(errors.Join(storage.ErrStateUncertain, err))
	}
	return ack, false, nil
}
func (b *propertyBackend) BindPropertyExecutionAuthority(a *storage.PropertyExecutionAuthority) error {
	ctx := context.Background()
	if err := b.acquire(ctx); err != nil {
		return err
	}
	defer b.mu.Unlock()
	if b.bindingAttempted {
		return storage.ErrDenied
	}
	b.bindingAttempted = true
	if a == nil || !b.source.ready() || len(b.current.Job().History) != 1 {
		return storage.ErrDenied
	}
	j, err := a.AdmittedJob()
	if err != nil {
		return err
	}
	if !sameBackendJob(j, b.current) {
		return storage.ErrDrift
	}
	// The coordinator keeps this capability inactive until successful binding and
	// final original checks. Neither binding nor decoded bytes launch anything.
	b.authority = a
	return nil
}

func (b *propertyBackend) collect(ctx context.Context, baseline bool) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
	zero := func(err error) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
		return storage.PropertyJob{}, storage.Policy{}, storage.Observation{}, err
	}
	if !b.source.ready() || b.authority == nil {
		return zero(storage.ErrDenied)
	}
	if err := b.check(ctx, baseline); err != nil {
		return zero(err)
	}
	applying := len(b.current.Job().History) == 2
	if applying {
		if err := b.source.verify(b.current); err != nil {
			return zero(err)
		}
	}
	j, p, o, err := b.source.collect(ctx, b.current.Job().ID)
	if err != nil {
		return zero(err)
	}
	if !sameBackendJob(j, b.current) || o.Eligible {
		return zero(storage.ErrDrift)
	}
	a, err := j.AssessObservation(p, o)
	if err != nil {
		return zero(err)
	}
	if baseline && !a.BaselineMatches {
		return zero(storage.ErrDrift)
	}
	j, err = ownBackendJob(j)
	if err != nil {
		return zero(err)
	}
	if applying {
		if err = b.source.verify(b.current); err != nil {
			return zero(err)
		}
	}
	if err = b.check(ctx, baseline); err != nil {
		return zero(err)
	}
	o.Eligible = baseline
	return j, p, o, nil
}
func (b *propertyBackend) PreparePropertyExecution(ctx context.Context, id string) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
	if err := b.acquire(ctx); err != nil {
		return storage.PropertyJob{}, storage.Policy{}, storage.Observation{}, err
	}
	defer b.mu.Unlock()
	if id != b.current.Job().ID || len(b.current.Job().History) != 1 {
		return storage.PropertyJob{}, storage.Policy{}, storage.Observation{}, storage.ErrDenied
	}
	return b.collect(ctx, true)
}
func (b *propertyBackend) RefreshPropertyExecution(ctx context.Context, j storage.PropertyJob) (storage.Policy, storage.Observation, error) {
	if err := b.acquire(ctx); err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	defer b.mu.Unlock()
	if !sameBackendJob(j, b.current) || len(j.Job().History) != 2 {
		return storage.Policy{}, storage.Observation{}, storage.ErrDrift
	}
	if b.control == nil {
		return storage.Policy{}, storage.Observation{}, errors.Join(storage.ErrDenied, b.controlErr)
	}
	_, p, o, err := b.collect(ctx, true)
	return p, o, err
}
func (b *propertyBackend) PublishPropertyExecution(next storage.PropertyJob, expected uint64) (storage.PropertyJob, error) {
	ctx := context.Background()
	if err := b.acquire(ctx); err != nil {
		return storage.PropertyJob{}, err
	}
	defer b.mu.Unlock()
	if !b.source.ready() || b.authority == nil || expected != uint64(len(b.current.Job().History)) {
		return storage.PropertyJob{}, storage.ErrDenied
	}
	next, err := ownBackendJob(next)
	if err != nil {
		return storage.PropertyJob{}, err
	}
	if err = storage.ValidatePropertyJobSuccessor(b.current, next); err != nil {
		return storage.PropertyJob{}, err
	}
	h := next.Job().History
	state := h[len(h)-1].State
	var publish func(storage.PropertyJob) (storage.PropertyJob, error)
	switch state {
	case storage.JobApplying:
		if b.control != nil || b.controlAttempted {
			return storage.PropertyJob{}, storage.ErrDenied
		}
		if err = b.check(ctx, true); err != nil {
			return storage.PropertyJob{}, err
		}
		publish = b.source.applying
	case storage.JobObserving:
		if !b.quiet || !b.observationAttempted || b.digest == "" || h[len(h)-1].ObservationDigest != b.digest {
			return storage.PropertyJob{}, storage.ErrDenied
		}
		publish = b.source.successor
	case storage.JobSucceeded:
		if !b.quiet || !b.matched || b.digest == "" || h[len(h)-1].ObservationDigest != b.digest {
			return storage.PropertyJob{}, storage.ErrDenied
		}
		publish = b.source.successor
	case storage.JobReconcileRequired:
		publish = b.source.successor
	default:
		return storage.PropertyJob{}, storage.ErrDenied
	}
	ack, err := publish(next)
	if err != nil || !sameBackendJob(next, ack) {
		b.blocked = true
		return storage.PropertyJob{}, errors.Join(storage.ErrStateUncertain, err)
	}
	b.current, err = ownBackendJob(ack)
	if err != nil {
		b.blocked = true
		return storage.PropertyJob{}, storage.ErrStateUncertain
	}
	if state == storage.JobApplying {
		b.control, b.controlErr = b.source.control(b.current, func(ctx context.Context) error { return b.check(ctx, true) })
		// Preserve an ACTUAL durable applying acknowledgement even when control
		// setup fails. Refresh denies launch; the worker can record reconciliation.
	}
	return ownBackendJob(b.current)
}
func (b *propertyBackend) ApplyProperty(ctx context.Context, i storage.PropertyIntent) (bool, error) {
	if err := b.acquire(ctx); err != nil {
		return false, err
	}
	defer b.mu.Unlock()
	if b.controlAttempted || b.control == nil || !b.sameIntent(i) || len(b.current.Job().History) != 2 {
		return false, storage.ErrDenied
	}
	b.controlAttempted = true
	if err := b.check(ctx, true); err != nil {
		return false, err
	}
	b.quiet, b.controlErr = b.control.apply(ctx, i)
	return b.quiet, b.controlErr
}
func (b *propertyBackend) ObservePropertyExecution(ctx context.Context, i storage.PropertyIntent) (storage.Policy, storage.Observation, error) {
	if err := b.acquire(ctx); err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	defer b.mu.Unlock()
	if b.observationAttempted || !b.controlAttempted || !b.quiet || b.controlErr != nil || !b.sameIntent(i) || len(b.current.Job().History) != 2 {
		return storage.Policy{}, storage.Observation{}, storage.ErrDenied
	}
	b.observationAttempted = true
	_, p, o, err := b.collect(ctx, false)
	if err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	digest, err := storage.PropertyExecutionObservationDigest(i, p, o)
	if err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	if err = b.check(ctx, false); err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	b.digest = digest
	b.matched = i.ComparePostcondition(p, o) == nil
	return p, o, nil
}
