// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// PropertyJobSource joins complete retained metadata to independently collected
// policy/native facts through one borrowed root state lease. It owns no lease,
// admits no work and changes no history, hold or native object. A match supplies
// no execution attribution, quiescence, recovered-state clearance or success.
type PropertyJobSource struct {
	policy  *PropertyPolicySource
	inspect func(string) (storage.PropertyJob, [32]byte, error)
}

func NewPropertyJobSource(state *storage.StateDirectory) (*PropertyJobSource, error) {
	policy, err := NewPropertyPolicySource(state)
	if err != nil {
		return nil, err
	}
	return &PropertyJobSource{policy: policy, inspect: state.InspectPropertyJob}, nil
}

// Assess selects a retained opaque job ID only. The original lease supplies the
// complete job/ledger and policy; that policy supplies the native pool address.
// Independently collected observations cannot be replaced by request inputs.
// Complete ledger reads bracket collection and exact selected bytes are also
// checked. Ordinary drift/cancellation/unsafe state discards the whole result;
// no data label or digest establishes an atomic fence against another root actor.
func (s *PropertyJobSource) Assess(ctx context.Context, id string) (storage.PropertyJobAssessment, error) {
	zero := storage.PropertyJobAssessment{}
	if ctx == nil {
		return zero, storage.ErrInvalid
	}
	if err := storage.ValidatePropertyJobSelection(id); err != nil {
		return zero, err
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if s == nil || s.policy == nil || s.inspect == nil {
		return zero, ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 135*time.Second)
	defer cancel()
	job, digest, err := s.inspect(id)
	if err != nil {
		return zero, err
	}
	if job.Job().ID != id || digest == ([32]byte{}) {
		return zero, storage.ErrDrift
	}
	before, err := storage.EncodePropertyJob(job)
	if err != nil {
		return zero, err
	}
	if err = ctx.Err(); err != nil {
		return zero, err
	}
	p, o, err := s.policy.Collect(ctx, job.Job().Intent.PoolGUID)
	if err != nil {
		return zero, err
	}
	result, err := job.AssessObservation(p, o)
	if err != nil {
		return zero, err
	}
	if err = ctx.Err(); err != nil {
		return zero, err
	}
	next, finalDigest, err := s.inspect(id)
	if err != nil {
		return zero, err
	}
	after, err := storage.EncodePropertyJob(next)
	if err != nil {
		return zero, err
	}
	if digest != finalDigest || !bytes.Equal(before, after) {
		return zero, storage.ErrDrift
	}
	if err = ctx.Err(); err != nil {
		return zero, err
	}
	return result, nil
}
