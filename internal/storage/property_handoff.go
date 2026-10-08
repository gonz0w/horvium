// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"time"
)

// ClaimPropertyExecution transfers one fresh, exactly acknowledged admission to
// one worker through this ORIGINAL coordinator/backend. It performs no I/O,
// accepts identifiers only and cannot reconstruct a claim from stored bytes.
// Matching Submit replies remain admission acknowledgements, not status/launch.
// The session argument still requires independent application authentication.
//
// The original trusted backend must implement both admission and execution. It
// must hold the same original root lease and prove complete native eligibility,
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
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := coordinatorContext(ctx); err != nil {
		return nil, err
	}
	if c.backend == nil || c.clock == nil || c.reviews == nil {
		return nil, ErrUnavailable
	}
	if c.publicationBlocked {
		return nil, ErrStateUncertain
	}
	at, err := c.now()
	if err != nil {
		return nil, err
	}
	// The retained set has the existing 64-review ceiling. No unbounded job index
	// or decoded metadata is used to discover an execution claim.
	for planID, entry := range c.reviews {
		if entry.ack.job.ID != id {
			continue
		}
		if entry.plan.session != session || entry.submission == "" || entry.executionClaimed || entry.duplicateAdmission || len(entry.ack.job.History) != 1 {
			return nil, ErrDenied
		}
		if !at.Before(entry.plan.review.ExpiresAt) {
			return nil, ErrExpired
		}
		encoded, err := EncodePropertyJob(entry.ack)
		if err != nil {
			return nil, err
		}
		backend, ok := c.backend.(PropertyExecutionBackend)
		if !ok {
			return nil, ErrUnavailable
		}
		final, err := c.now()
		if err != nil {
			return nil, err
		}
		if !final.Before(entry.plan.review.ExpiresAt) {
			return nil, ErrExpired
		}
		if err = coordinatorContext(ctx); err != nil {
			return nil, err
		}
		// Claim under the original mutex before returning ANY executable object.
		// Reconstruction, a second handoff and copies of the returned pointer
		// cannot create a second attempt. Expiry reclamation releases no hold.
		entry.executionClaimed = true
		c.reviews[planID] = entry
		bound := &propertyHandoffBackend{PropertyExecutionBackend: backend, owner: c, id: id, admitted: encoded, expires: entry.plan.review.ExpiresAt}
		e := NewPropertyExecution(bound)
		// Keep the original civil-clock lineage. A worker rollback also blocks
		// the coordinator; outcome publication after forward expiry remains legal.
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
		return e, nil
	}
	return nil, ErrDenied
}

// Only the coordinator can create this envelope. Delegated publication/observation
// retains the original backend and existing exact-acknowledgement sequence. Native
// lease/current-record/reservation proof remains that backend's responsibility.
type propertyHandoffBackend struct {
	PropertyExecutionBackend
	owner    *PropertyCoordinator
	id       string
	admitted []byte
	expires  time.Time
}

func (b *propertyHandoffBackend) live(ctx context.Context) error {
	if err := coordinatorContext(ctx); err != nil {
		return err
	}
	c := b.owner
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.publicationBlocked {
		return ErrStateUncertain
	}
	if c.clock == nil {
		return ErrUnavailable
	}
	at, err := c.now()
	if err != nil {
		return err
	}
	if !at.Before(b.expires) {
		return ErrExpired
	}
	return coordinatorContext(ctx)
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
	if err := b.live(ctx); err != nil {
		return false, err
	}
	// Do not discard an actual late acknowledgement by rechecking after effect.
	// Observe/durable outcome remain the existing sequence's independent gates.
	return b.PropertyExecutionBackend.ApplyProperty(ctx, i)
}
