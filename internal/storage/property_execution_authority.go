// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"reflect"
)

// PropertyExecutionAuthorityBinder retains the original coordinator's opaque
// prelaunch authority on the SAME backend that admitted the job. The synchronous
// binder must be bounded and perform no effect or publication; the handle is
// inactive until it returns successfully and final original checks pass.
type PropertyExecutionAuthorityBinder interface {
	BindPropertyExecutionAuthority(*PropertyExecutionAuthority) error
}

type propertyAuthorityBackend interface {
	PropertyExecutionBackend
	PropertyExecutionAuthorityBinder
}

// PropertyExecutionAuthority is an opaque original-coordinator capability, never
// wire data. There is no constructor or reset. Copies share one private identity.
// It proves retained plan/session CONTEXT and clock/publication lineage, not
// authentication, root/native eligibility, recovery or global launch fencing.
// Those remain the original trusted backend's independent obligations.
type PropertyExecutionAuthority struct {
	identity *propertyExecutionAuthorityIdentity
}

type propertyExecutionAuthorityIdentity struct {
	owner      *PropertyCoordinator
	plan       Plan
	admitted   []byte
	submission string
	// Protected by owner.mu. Retire only revokes launch authority; it releases
	// no durable reservation and must not discard post-effect outcome duties.
	active bool
}

func (a *PropertyExecutionAuthority) AdmittedJob() (PropertyJob, error) {
	if a == nil || a.identity == nil || a.identity.owner == nil {
		return PropertyJob{}, ErrDenied
	}
	// Immutable owned bytes remain readable during binding and after retirement.
	// They are admission evidence, never current status or a fresh durability ack.
	return DecodePropertyJob(a.identity.admitted)
}

// CheckLive is suitable for the backend's final prelaunch callback. It accepts
// no asserted actor, clock, record, observations or replacement coordinator.
// Context belongs to the independent worker, not the original HTTP request.
func (a *PropertyExecutionAuthority) CheckLive(ctx context.Context) error {
	if err := coordinatorContext(ctx); err != nil {
		return err
	}
	if a == nil || a.identity == nil || a.identity.owner == nil {
		return ErrDenied
	}
	c := a.identity.owner
	c.mu.Lock()
	defer c.mu.Unlock()
	return a.checkLocked(ctx)
}

func (a *PropertyExecutionAuthority) checkLocked(ctx context.Context) error {
	x := a.identity
	c := x.owner
	if c.publicationBlocked {
		return ErrStateUncertain
	}
	if !x.active {
		return ErrDenied
	}
	entry, ok := c.reviews[x.plan.review.ID]
	if !ok || entry.executionAuthority != x || !entry.executionClaimed || entry.duplicateAdmission || entry.submission != x.submission || !reflect.DeepEqual(entry.plan, x.plan) {
		return ErrDenied
	}
	if c.clock == nil {
		return ErrUnavailable
	}
	at, err := c.now()
	if err != nil {
		return err
	}
	if at.Before(x.plan.review.CreatedAt) || !at.Before(x.plan.review.ExpiresAt) {
		return ErrExpired
	}
	actual, err := EncodePropertyJob(entry.ack)
	if err != nil || !bytes.Equal(actual, x.admitted) {
		return ErrDrift
	}
	// Encoding is bounded but can take time. Retain the original civil-clock
	// lineage and recheck expiry/context before reporting live authority.
	at, err = c.now()
	if err != nil {
		return err
	}
	if !at.Before(x.plan.review.ExpiresAt) {
		return ErrExpired
	}
	return coordinatorContext(ctx)
}

func (a *PropertyExecutionAuthority) activate(ctx context.Context) error {
	c := a.identity.owner
	c.mu.Lock()
	defer c.mu.Unlock()
	a.identity.active = true
	if err := a.checkLocked(ctx); err != nil {
		a.identity.active = false
		return err
	}
	return nil
}

func (a *PropertyExecutionAuthority) retire() {
	c := a.identity.owner
	c.mu.Lock()
	a.identity.active = false
	c.mu.Unlock()
}
