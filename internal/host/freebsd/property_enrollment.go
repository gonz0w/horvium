// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"reflect"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

// PropertyEnrollmentSource collects fresh native evidence for an explicit root
// proposal, then binds a volatile review to a borrowed root state lease. It does
// not infer selections, publish records, grant eligibility or own/close the lease.
type PropertyEnrollmentSource struct {
	pool      *PoolPropertySource
	providers *PoolProviderSource
	system    *SystemUseSource
	bind      func(time.Time, string, storage.PropertyEnrollmentPlan, storage.Policy, storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse) (storage.PropertyEnrollmentState, error)
	now       func() time.Time
}

func NewPropertyEnrollmentSource(state *storage.StateDirectory) (*PropertyEnrollmentSource, error) {
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	if state == nil {
		return nil, storage.ErrStateDirectory
	}
	pool, err := NewPoolPropertySource()
	if err != nil {
		return nil, err
	}
	providers, err := NewPoolProviderSource()
	if err != nil {
		return nil, err
	}
	system, err := NewSystemUseSource()
	if err != nil {
		return nil, err
	}
	return &PropertyEnrollmentSource{pool: pool, providers: providers, system: system, bind: state.BindPropertyEnrollment, now: time.Now}, nil
}

// Review takes a fully explicit proposal and predecessor from a future trusted
// OS-root workflow. An entirely zero predecessor requests first enrollment; a
// failed read cannot supply that decision. Every denial returns zero state.
func (s *PropertyEnrollmentSource) Review(ctx context.Context, rootContext string, proposed, previous storage.Policy) (storage.PropertyEnrollmentState, error) {
	zero := storage.PropertyEnrollmentState{}
	if err := s.preflight(ctx, rootContext, proposed, previous); err != nil {
		return zero, err
	}
	if s.bind == nil {
		return zero, ErrUnavailable
	}
	// Canonical codec copies own both selections before any native I/O.
	owned, err := copyEnrollmentPolicy(proposed)
	if err != nil {
		return zero, err
	}
	if !reflect.DeepEqual(previous, storage.Policy{}) {
		previous, err = copyEnrollmentPolicy(previous)
		if err != nil {
			return zero, err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 135*time.Second)
	defer cancel()
	o, disks, u, err := s.collect(ctx, owned)
	if err != nil {
		return zero, err
	}
	plan, err := storage.BuildPropertyEnrollment(s.now(), rootContext, owned, o, disks, u)
	if err != nil {
		return zero, err
	}
	bound, err := s.bind(s.now(), rootContext, plan, previous, o, disks, u)
	if err != nil {
		return zero, err
	}
	if err = ctx.Err(); err != nil {
		return zero, err
	}
	if err = plan.Validate(s.now(), rootContext, owned, o, disks, u); err != nil {
		return zero, err
	}
	if err = bound.Validate(s.now(), rootContext, o, disks, u); err != nil {
		return zero, err
	}
	if err = ctx.Err(); err != nil {
		return zero, err
	}
	// Metadata inspection must not let expiry or a backwards clock pass unchecked.
	if err = plan.Validate(s.now(), rootContext, owned, o, disks, u); err != nil {
		return zero, err
	}
	if err = ctx.Err(); err != nil {
		return zero, err
	}
	return bound, nil
}

// Validate recollects independent native evidence and uses the binding's original
// root lease. Review copies cannot hydrate a binding or move it to another lease.
func (s *PropertyEnrollmentSource) Validate(ctx context.Context, rootContext string, bound storage.PropertyEnrollmentState) error {
	r := bound.Review()
	if r.ID == "" {
		return storage.ErrInvalid
	}
	if err := s.preflight(ctx, rootContext, r.Policy, bound.Predecessor()); err != nil {
		return err
	}
	if err := enrollmentTime(s.now(), r); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 135*time.Second)
	defer cancel()
	o, disks, u, err := s.collect(ctx, r.Policy)
	if err != nil {
		return err
	}
	if err = bound.Validate(s.now(), rootContext, o, disks, u); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = enrollmentTime(s.now(), r); err != nil {
		return err
	}
	return ctx.Err()
}

func (s *PropertyEnrollmentSource) preflight(ctx context.Context, rootContext string, p, previous storage.Policy) error {
	if ctx == nil {
		return storage.ErrInvalid
	}
	if err := storage.ValidatePropertyEnrollmentSelection(rootContext, p, previous); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.pool == nil || s.providers == nil || s.system == nil || s.now == nil {
		return ErrUnavailable
	}
	return nil
}

func (s *PropertyEnrollmentSource) collect(ctx context.Context, p storage.Policy) (storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse, error) {
	o, err := s.pool.Collect(ctx, p.PoolGUID, p.PoolName)
	if err != nil {
		return storage.Observation{}, nil, storage.PoolSystemUse{}, err
	}
	d, err := s.providers.Collect(ctx, o)
	if err != nil {
		return storage.Observation{}, nil, storage.PoolSystemUse{}, err
	}
	u, err := s.system.Collect(ctx)
	if err != nil {
		return storage.Observation{}, nil, storage.PoolSystemUse{}, err
	}
	if err = ctx.Err(); err != nil {
		return storage.Observation{}, nil, storage.PoolSystemUse{}, err
	}
	return o, d, u, nil
}

func copyEnrollmentPolicy(p storage.Policy) (storage.Policy, error) {
	data, err := storage.EncodeOwnershipRecord(p)
	if err != nil {
		return storage.Policy{}, err
	}
	return storage.DecodeOwnershipRecord(data)
}

func enrollmentTime(at time.Time, r storage.PropertyEnrollmentReview) error {
	if at.Before(r.CreatedAt) || !at.Before(r.ExpiresAt) {
		return storage.ErrExpired
	}
	return nil
}
