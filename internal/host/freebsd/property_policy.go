// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"

	"github.com/gonz0w/horvium/internal/storage"
)

// PropertyPolicySource binds fixed native observations to metadata read through
// an already-held root state lease. It does not own/close that lease, enroll any
// scope, supply system/device clearance or implement the mutation backend.
type PropertyPolicySource struct {
	ownership func(string) (storage.Policy, error)
	pool      *PoolPropertySource
}

func NewPropertyPolicySource(state *storage.StateDirectory) (*PropertyPolicySource, error) {
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
	return &PropertyPolicySource{ownership: state.ReadOwnership, pool: pool}, nil
}

// Collect accepts a pool GUID only; its name and explicit scope come from the
// private ownership reader, never the request. Canonical policy reads bracket
// the complete two-pass native collection. Equality detects ordinary drift,
// not intervening root changes or an atomic lease with eventual publication.
// Every denial returns zero policy and observation. Success remains ineligible.
func (s *PropertyPolicySource) Collect(ctx context.Context, guid string) (storage.Policy, storage.Observation, error) {
	p, o, err := s.collect(ctx, guid)
	if err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	return p, o, nil
}

func (s *PropertyPolicySource) collect(ctx context.Context, guid string) (storage.Policy, storage.Observation, error) {
	if ctx == nil {
		return storage.Policy{}, storage.Observation{}, storage.ErrInvalid
	}
	// Reuse the GUID validator with a fixed valid name, before policy I/O.
	if err := storage.ValidatePoolSelection(guid, "selection"); err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	if err := ctx.Err(); err != nil {
		return storage.Policy{}, storage.Observation{}, err
	}
	if s == nil || s.ownership == nil || s.pool == nil {
		return storage.Policy{}, storage.Observation{}, ErrUnavailable
	}
	p, err := s.ownership(guid)
	if err != nil {
		return p, storage.Observation{}, err
	}
	if p.PoolGUID != guid {
		return p, storage.Observation{}, storage.ErrDenied
	}
	before, err := storage.EncodeOwnershipRecord(p)
	if err != nil {
		return p, storage.Observation{}, err
	}
	o, err := s.pool.Collect(ctx, guid, p.PoolName)
	if err != nil {
		return p, o, err
	}
	if err = ctx.Err(); err != nil {
		return p, o, err
	}
	next, err := s.ownership(guid)
	if err != nil {
		return p, o, err
	}
	after, err := storage.EncodeOwnershipRecord(next)
	if err != nil {
		return p, o, err
	}
	if !bytes.Equal(before, after) {
		return p, o, storage.ErrDrift
	}
	if err = storage.ValidatePropertyPolicyBinding(next, o); err != nil {
		return p, o, err
	}
	if err = ctx.Err(); err != nil {
		return p, o, err
	}
	// Decode our bounded canonical bytes to ensure independent, ordered output.
	p, err = storage.DecodeOwnershipRecord(after)
	o.Eligible = false
	return p, o, err
}
