// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// PropertySafetySource joins private policy/native properties with complete
// provider and system-use collections. It borrows the root lease and remains
// read-only and ineligible: enrollment provisioning and store/admission/recovery
// gates are absent. Explicit root member facts must match fresh native providers.
type PropertySafetySource struct {
	policy    *PropertyPolicySource
	system    *SystemUseSource
	providers *PoolProviderSource
}

func NewPropertySafetySource(state *storage.StateDirectory) (*PropertySafetySource, error) {
	policy, err := NewPropertyPolicySource(state)
	if err != nil {
		return nil, err
	}
	system, err := NewSystemUseSource()
	if err != nil {
		return nil, err
	}
	providers, err := NewPoolProviderSource()
	if err != nil {
		return nil, err
	}
	return &PropertySafetySource{policy: policy, system: system, providers: providers}, nil
}
func (s *PropertySafetySource) Collect(ctx context.Context, guid string) (storage.Policy, storage.Observation, error) {
	return s.collect(ctx, guid, nil)
}

// A trusted store source can add its lease-bound check within the same final
// policy window. The public safety observer keeps its accepted read-only scope.
func (s *PropertySafetySource) collect(ctx context.Context, guid string, check func(storage.Policy, storage.PoolSystemUse) error) (storage.Policy, storage.Observation, error) {
	if ctx == nil {
		return storage.Policy{}, storage.Observation{}, storage.ErrInvalid
	}
	if s == nil || s.policy == nil || s.system == nil || s.providers == nil {
		return storage.Policy{}, storage.Observation{}, ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 135*time.Second)
	defer cancel()
	p, o, err := s.policy.Collect(ctx, guid)
	if err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	if len(p.Members) == 0 {
		return storage.Policy{}, storage.Observation{}, storage.ErrDenied
	}
	providers, err := s.providers.Collect(ctx, o)
	if err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	if err = storage.ValidatePropertyMembers(p, o, providers); err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	u, err := s.system.Collect(ctx)
	if err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	if err = storage.CheckPropertySystemExclusions(o, u); err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	if err = storage.CheckPropertySystemProviders(o.PoolGUID, u, providers); err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	if err = ctx.Err(); err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	if check != nil {
		if err = check(p, u); err != nil {
			return storage.Policy{}, storage.Observation{}, err
		}
		if err = ctx.Err(); err != nil {
			return storage.Policy{}, storage.Observation{}, err
		}
	}
	// The provider/system phases must not extend the earlier policy baseline window
	// without a final independent read through the same held root lease.
	next, err := s.policy.ownership(guid)
	if err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	before, err := storage.EncodeOwnershipRecord(p)
	if err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	after, err := storage.EncodeOwnershipRecord(next)
	if err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	if !bytes.Equal(before, after) {
		return storage.Policy{}, storage.Observation{}, storage.ErrDrift
	}
	if err = ctx.Err(); err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	o.Eligible = false
	return p, o, nil
}
