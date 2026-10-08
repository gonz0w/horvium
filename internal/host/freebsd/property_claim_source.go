// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// PropertyClaimSource joins one original opaque root claim to complete fresh
// policy, provider, system-use and backing observations. It borrows the claim's
// lease; every metadata operation comes from that claim, with no replacement
// state handle or caller policy/facts. It is read-only, remains ineligible and
// does not implement executor/recovery/session/clock clearance or a backend.
type PropertyClaimSource struct {
	safety  *PropertySafetySource
	inspect func(string) (storage.PropertyJob, [32]byte, error)
	check   func(storage.Policy, storage.PoolSystemUse, storage.PropertyJob) error
}

func NewPropertyClaimSource(claim *storage.PropertyStateClaim) (*PropertyClaimSource, error) {
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	if claim == nil {
		return nil, storage.ErrDenied
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
	safety := &PropertySafetySource{policy: &PropertyPolicySource{ownership: claim.ReadOwnership, pool: pool}, providers: providers, system: system}
	return &PropertyClaimSource{safety: safety, inspect: claim.InspectPropertyJob, check: claim.CheckPropertyState}, nil
}

// Collect accepts an opaque job ID only. Complete claim-bound census reads
// bracket guarded native collection and the original claim's backing check in
// the final policy window. Exact job and full ledger presentation drift deny.
// Every failure returns zero projections; success keeps Eligible=false. Facts
// may differ from the intent baseline, including after an effect: this observer
// cannot authorize it or invent attribution, durable outcomes or hold release.
func (s *PropertyClaimSource) Collect(ctx context.Context, id string) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
	zero := func(err error) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
		return storage.PropertyJob{}, storage.Policy{}, storage.Observation{}, err
	}
	if ctx == nil {
		return zero(storage.ErrInvalid)
	}
	if err := storage.ValidatePropertyJobSelection(id); err != nil {
		return zero(err)
	}
	if err := ctx.Err(); err != nil {
		return zero(err)
	}
	if s == nil || s.safety == nil || s.inspect == nil || s.check == nil {
		return zero(ErrUnavailable)
	}
	ctx, cancel := context.WithTimeout(ctx, 135*time.Second)
	defer cancel()
	job, digest, err := s.inspect(id)
	if err != nil {
		return zero(err)
	}
	if job.Job().ID != id || digest == ([32]byte{}) {
		return zero(storage.ErrDrift)
	}
	before, err := storage.EncodePropertyJob(job)
	if err != nil {
		return zero(err)
	}
	history := job.Job().History
	if len(history) != 1 && (len(history) != 2 || history[1].State != storage.JobApplying) {
		return zero(storage.ErrDenied)
	}
	if err = ctx.Err(); err != nil {
		return zero(err)
	}
	p, o, err := s.safety.collect(ctx, job.Job().Intent.PoolGUID, func(p storage.Policy, u storage.PoolSystemUse) error {
		return s.check(p, u, job)
	})
	if err != nil {
		return zero(err)
	}
	if _, err = job.AssessObservation(p, o); err != nil {
		return zero(err)
	}
	if err = ctx.Err(); err != nil {
		return zero(err)
	}
	next, finalDigest, err := s.inspect(id)
	if err != nil {
		return zero(err)
	}
	after, err := storage.EncodePropertyJob(next)
	if err != nil {
		return zero(err)
	}
	if digest != finalDigest || !bytes.Equal(before, after) {
		return zero(storage.ErrDrift)
	}
	if err = ctx.Err(); err != nil {
		return zero(err)
	}
	o.Eligible = false
	return job, p, o, nil
}
