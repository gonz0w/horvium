// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

// PropertyStateSource adds complete lease-bound metadata/backing/hold checks to
// native property safety. It does not grant eligibility or recovered authority,
// and has no runtime/backend/transport consumer. It borrows the caller's lease.
type PropertyStateSource struct {
	safety *PropertySafetySource
	check  func(storage.Policy, storage.PoolSystemUse) error
}

func NewPropertyStateSource(state *storage.StateDirectory) (*PropertyStateSource, error) {
	safety, err := NewPropertySafetySource(state)
	if err != nil {
		return nil, err
	}
	return &PropertyStateSource{safety: safety, check: state.CheckPropertyState}, nil
}
func (s *PropertyStateSource) Collect(ctx context.Context, guid string) (storage.Policy, storage.Observation, error) {
	if ctx == nil {
		return storage.Policy{}, storage.Observation{}, storage.ErrInvalid
	}
	if s == nil || s.safety == nil || s.check == nil {
		return storage.Policy{}, storage.Observation{}, ErrUnavailable
	}
	return s.safety.collect(ctx, guid, s.check)
}
