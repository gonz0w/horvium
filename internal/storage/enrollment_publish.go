// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"sync"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

// Copies of one sealed binding share this volatile one-attempt claim. It is not
// a durable retry identifier, restored-state clearance or a cross-process fence.
type enrollmentPublicationClaim struct {
	mu        sync.Mutex
	attempted bool
}

// Publish writes only the reviewed root ownership metadata through the original
// lease. Callers must independently collect fresh native inputs. It enables no
// native executor or management workflow. Any publication attempt consumes this
// review, even on failure; reconcile and explicitly review again, never replay.
func (p PropertyEnrollmentState) Publish(ctx context.Context, rootContext string, o Observation, disks []inventory.DiskObservation, u PoolSystemUse) error {
	return p.publish(ctx, rootContext, o, disks, u, time.Now, func() error {
		return p.publishEnrollmentMetadata(ctx, rootContext, o, disks, u, time.Now, defaultStateCatalogLimits, nil)
	})
}

// Private clock/publisher seams exercise transitions without native filesystem
// work. Public publication always uses the actual clock and original root lease.
func (p PropertyEnrollmentState) publish(ctx context.Context, rootContext string, o Observation, disks []inventory.DiskObservation, u PoolSystemUse, now func() time.Time, publish func() error) error {
	if ctx == nil || p.lease == nil || p.claim == nil || now == nil || publish == nil {
		return ErrInvalid
	}
	p.claim.mu.Lock()
	defer p.claim.mu.Unlock()
	if p.claim.attempted {
		return ErrDenied
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := p.plan.Validate(now(), rootContext, p.plan.review.Policy, o, disks, u); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	p.claim.attempted = true
	// A successful native transaction is its actual acknowledgement, even when
	// cancellation/expiry happens after rename. Do not encourage a second write.
	return publish()
}
