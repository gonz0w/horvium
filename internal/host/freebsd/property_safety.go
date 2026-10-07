// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"reflect"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
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
	if err = bindSystemProviders(o.PoolGUID, u, providers); err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	if err = ctx.Err(); err != nil {
		return storage.Policy{}, storage.Observation{}, err
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

func bindSystemProviders(guid string, u storage.PoolSystemUse, disks []inventory.DiskObservation) error {
	for _, pool := range u.Pools() {
		if pool.GUID != guid {
			continue
		}
		if len(pool.Disks) != len(disks) {
			return storage.ErrDrift
		}
		seen := map[string]bool{}
		for _, d := range disks {
			matched := false
			for _, p := range pool.Disks {
				if p.Name != d.Provider.Name {
					continue
				}
				// IDs are graph-local; independently compare complete provider
				// identity/provenance/geometry rather than persisting graph IDs.
				expected := d.Provider
				expected.ID = ""
				expected.GeomID = ""
				p.ID = ""
				p.GeomID = ""
				matched = reflect.DeepEqual(expected, p)
			}
			if !matched || seen[d.Provider.Name] {
				return storage.ErrDrift
			}
			seen[d.Provider.Name] = true
		}
		return nil
	}
	return storage.ErrUnavailable
}
