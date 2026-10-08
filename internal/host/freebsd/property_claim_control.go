// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"os"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// Private binding prerequisite, without a runtime consumer. The complete original
// backend must supply live clearance for its plan/session/civil clock, recovery
// and global/pool reservations, and own exactly one control object. This factory
// cannot reconstruct that authority, promote eligibility or prevent two separately
// constructed controls. Only disposable fixtures install clearance today.
func newPropertyClaimControl(claim *storage.PropertyStateClaim, applying storage.PropertyJob, live func(context.Context) error) (*propertyControl, error) {
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	if os.Geteuid() != 0 {
		return nil, ErrPermission
	}
	if claim == nil || live == nil {
		return nil, storage.ErrDenied
	}
	data, err := storage.EncodePropertyJob(applying)
	if err != nil {
		return nil, err
	}
	owned, err := storage.DecodePropertyJob(data)
	if err != nil {
		return nil, err
	}
	if err = claim.VerifyPropertyApplyingAcknowledgement(owned); err != nil {
		return nil, err
	}
	source, err := NewPropertyClaimSource(claim)
	if err != nil {
		return nil, err
	}
	a := &propertyClaimControlAuthority{expected: owned, live: live, verify: claim.VerifyPropertyApplyingAcknowledgement, collect: source.Collect}
	return &propertyControl{authorize: a.authorize, run: propertySetNative, root: func() bool { return os.Geteuid() == 0 }, clock: time.Now}, nil
}

// Function seams are private portable test fixtures; the factory installs only
// methods from the same original opaque claim and fixed independent native source.
type propertyClaimControlAuthority struct {
	expected storage.PropertyJob
	live     func(context.Context) error
	verify   func(storage.PropertyJob) error
	collect  func(context.Context, string) (storage.PropertyJob, storage.Policy, storage.Observation, error)
}

func (a *propertyClaimControlAuthority) authorize(ctx context.Context, intent storage.PropertyIntent) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
	zero := func(err error) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
		return storage.PropertyJob{}, storage.Policy{}, storage.Observation{}, err
	}
	if ctx == nil {
		return zero(storage.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return zero(err)
	}
	if a == nil || a.live == nil || a.verify == nil || a.collect == nil {
		return zero(ErrUnavailable)
	}
	actual, err := storage.EncodePropertyIntent(intent)
	if err != nil {
		return zero(err)
	}
	expected, err := storage.EncodePropertyIntent(a.expected.Intent())
	if err != nil {
		return zero(err)
	}
	if !bytes.Equal(actual, expected) {
		return zero(storage.ErrDrift)
	}
	data, err := storage.EncodePropertyJob(a.expected)
	if err != nil {
		return zero(err)
	}
	history := a.expected.Job().History
	if len(history) != 2 || history[1].State != storage.JobApplying {
		return zero(storage.ErrDenied)
	}
	if err = a.live(ctx); err != nil {
		return zero(err)
	}
	if err = ctx.Err(); err != nil {
		return zero(err)
	}
	if err = a.verify(a.expected); err != nil {
		return zero(err)
	}
	if err = a.live(ctx); err != nil {
		return zero(err)
	}
	if err = ctx.Err(); err != nil {
		return zero(err)
	}
	job, p, o, err := a.collect(ctx, a.expected.Job().ID)
	if err != nil {
		return zero(err)
	}
	if err = ctx.Err(); err != nil {
		return zero(err)
	}
	got, err := storage.EncodePropertyJob(job)
	if err != nil {
		return zero(err)
	}
	if !bytes.Equal(data, got) || o.Eligible {
		return zero(storage.ErrDrift)
	}
	assessment, err := job.AssessObservation(p, o)
	if err != nil {
		return zero(err)
	}
	if !assessment.BaselineMatches {
		return zero(storage.ErrDrift)
	}
	// Own the reply before the final metadata/live/context checks.
	job, err = storage.DecodePropertyJob(got)
	if err != nil {
		return zero(err)
	}
	if err = a.verify(a.expected); err != nil {
		return zero(err)
	}
	if err = a.live(ctx); err != nil {
		return zero(err)
	}
	if err = ctx.Err(); err != nil {
		return zero(err)
	}
	return job, p, o, nil
}
