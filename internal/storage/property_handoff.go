// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"errors"
	"time"
)

// ClaimPropertyExecution transfers one fresh, exactly acknowledged admission to
// one worker through this ORIGINAL coordinator/backend. It performs no native I/O,
// accepts identifiers only and cannot reconstruct a claim from stored bytes.
// Matching Submit replies remain admission acknowledgements, not status/launch.
// The session argument still requires independent application authentication.
//
// The original trusted backend must implement admission, execution and authority
// binding. It must hold the same original root lease and prove native eligibility,
// recovery, current durable records and global/pool launch serialization. This
// portable handoff proves NONE of those native obligations. No runtime adapter
// implements that combined boundary yet. Calls are synchronous, bounded by the
// accepted coordinator/sequence contracts; no worker or service is started here.
func (c *PropertyCoordinator) ClaimPropertyExecution(ctx context.Context, session, id string) (*PropertyExecution, error) {
	if err := coordinatorContext(ctx); err != nil {
		return nil, err
	}
	if !token(session) || !validJobDigest(id) {
		return nil, ErrInvalid
	}
	if c == nil {
		return nil, ErrUnavailable
	}
	a, backend, err := c.reservePropertyExecutionAuthority(ctx, session, id)
	if err != nil {
		return nil, err
	}
	granted := false
	defer func() {
		if !granted {
			a.retire()
		}
	}()
	// Consume the original claim before calling the binder, outside owner.mu.
	// The backend may inspect the owned admission but cannot use pending authority.
	if err = backend.BindPropertyExecutionAuthority(a); err != nil {
		return nil, err
	}
	if err = a.activate(ctx); err != nil {
		return nil, err
	}
	bound := &propertyHandoffBackend{PropertyExecutionBackend: backend, owner: c, id: id, admitted: a.identity.admitted, authority: a}
	e := NewPropertyExecution(bound)
	e.finish = a.retire
	// Keep the original civil-clock lineage even after launch retirement: an
	// actual verified late outcome still has durable publication obligations.
	e.clock = func() time.Time {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.clock == nil {
			return time.Time{}
		}
		at, err := c.now()
		if err != nil {
			return time.Time{}
		}
		return at
	}
	granted = true
	return e, nil
}

func (c *PropertyCoordinator) reservePropertyExecutionAuthority(ctx context.Context, session, id string) (*PropertyExecutionAuthority, propertyAuthorityBackend, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := coordinatorContext(ctx); err != nil {
		return nil, nil, err
	}
	if c.backend == nil || c.clock == nil || c.reviews == nil {
		return nil, nil, ErrUnavailable
	}
	if c.publicationBlocked {
		return nil, nil, ErrStateUncertain
	}
	at, err := c.now()
	if err != nil {
		return nil, nil, err
	}
	// The existing 64-review bound is unchanged; no decoded ledger can discover
	// or reconstruct one of these volatile original identities.
	for planID, entry := range c.reviews {
		if entry.ack.job.ID != id {
			continue
		}
		if entry.plan.session != session || entry.submission == "" || entry.executionClaimed || entry.duplicateAdmission || len(entry.ack.job.History) != 1 {
			return nil, nil, ErrDenied
		}
		if !at.Before(entry.plan.review.ExpiresAt) {
			return nil, nil, ErrExpired
		}
		encoded, err := EncodePropertyJob(entry.ack)
		if err != nil {
			return nil, nil, err
		}
		backend, ok := c.backend.(propertyAuthorityBackend)
		if !ok {
			return nil, nil, ErrUnavailable
		}
		final, err := c.now()
		if err != nil {
			return nil, nil, err
		}
		if !final.Before(entry.plan.review.ExpiresAt) {
			return nil, nil, ErrExpired
		}
		if err = coordinatorContext(ctx); err != nil {
			return nil, nil, err
		}
		x := &propertyExecutionAuthorityIdentity{owner: c, plan: Plan{review: entry.plan.Review(), session: entry.plan.session, policyHash: entry.plan.policyHash}, admitted: encoded, submission: entry.submission}
		entry.executionClaimed = true
		entry.executionAuthority = x
		c.reviews[planID] = entry
		return &PropertyExecutionAuthority{identity: x}, backend, nil
	}
	return nil, nil, ErrDenied
}

// Only the coordinator can create this envelope. Delegated publication/observation
// retains the original backend and existing exact-acknowledgement sequence. Native
// lease/current-record/reservation proof remains that backend's responsibility.
type propertyHandoffBackend struct {
	PropertyExecutionBackend
	owner     *PropertyCoordinator
	id        string
	admitted  []byte
	authority *PropertyExecutionAuthority
}

// A worker's uncertain publication must suspend the same coordinator that
// admitted it. The execution sequence keeps its last exact acknowledgement;
// returning uncertainty here also denies later admissions and issued workers.
// Backend I/O must stay outside owner.mu: root adapters may consult the original
// coordinator. This is propagation, not atomic native launch serialization.
func (b *propertyHandoffBackend) PublishPropertyExecution(next PropertyJob, expected uint64) (PropertyJob, error) {
	b.owner.mu.Lock()
	blocked := b.owner.publicationBlocked
	b.owner.mu.Unlock()
	if blocked {
		return PropertyJob{}, ErrStateUncertain
	}
	// Own the expected bytes before delegation; an adapter must not be able to
	// rewrite an aliased candidate and thereby redefine its acknowledgement.
	encoded, err := EncodePropertyJob(next)
	if err != nil {
		return PropertyJob{}, err
	}
	ack, err := b.PropertyExecutionBackend.PublishPropertyExecution(clonePropertyJob(next), expected)
	actual, encodingErr := EncodePropertyJob(ack)
	if err != nil || encodingErr != nil || !bytes.Equal(encoded, actual) {
		b.owner.mu.Lock()
		b.owner.publicationBlocked = true
		b.owner.mu.Unlock()
		return PropertyJob{}, errors.Join(ErrStateUncertain, err, encodingErr)
	}
	// No expiry/context check after an exact durable acknowledgement. A late
	// verified outcome still belongs to the existing attempt.
	return clonePropertyJob(ack), nil
}

func (b *propertyHandoffBackend) live(ctx context.Context) error {
	return b.authority.CheckLive(ctx)
}

func (b *propertyHandoffBackend) PreparePropertyExecution(ctx context.Context, id string) (PropertyJob, Policy, Observation, error) {
	if id != b.id {
		return PropertyJob{}, Policy{}, Observation{}, ErrDenied
	}
	if err := b.live(ctx); err != nil {
		return PropertyJob{}, Policy{}, Observation{}, err
	}
	j, p, o, err := b.PropertyExecutionBackend.PreparePropertyExecution(ctx, id)
	if err != nil {
		return PropertyJob{}, Policy{}, Observation{}, err
	}
	actual, err := EncodePropertyJob(j)
	if err != nil || !bytes.Equal(actual, b.admitted) {
		return PropertyJob{}, Policy{}, Observation{}, ErrDrift
	}
	if err = b.live(ctx); err != nil {
		return PropertyJob{}, Policy{}, Observation{}, err
	}
	return clonePropertyJob(j), p, o, nil
}

func (b *propertyHandoffBackend) RefreshPropertyExecution(ctx context.Context, j PropertyJob) (Policy, Observation, error) {
	if err := b.live(ctx); err != nil {
		return Policy{}, Observation{}, err
	}
	p, o, err := b.PropertyExecutionBackend.RefreshPropertyExecution(ctx, j)
	if err != nil {
		return Policy{}, Observation{}, err
	}
	if err = b.live(ctx); err != nil {
		return Policy{}, Observation{}, err
	}
	return p, o, nil
}

func (b *propertyHandoffBackend) ApplyProperty(ctx context.Context, i PropertyIntent) (bool, error) {
	// Native clearance exists only through this control attempt, never during
	// post-effect observation/publication. Run also retires every preflight cut.
	defer b.authority.retire()
	if err := b.live(ctx); err != nil {
		return false, err
	}
	// Do not discard an actual late acknowledgement by rechecking after effect.
	// Observe/durable outcome remain the existing sequence's independent gates.
	return b.PropertyExecutionBackend.ApplyProperty(ctx, i)
}
