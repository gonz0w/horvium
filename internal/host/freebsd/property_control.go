// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// Private control prerequisite, deliberately without a public constructor or
// runtime consumer. A future root backend must install authorize through its
// ORIGINAL live plan/session/reservation/lease, proving full native eligibility,
// recovery and exact current durable applying state. Stored bytes or a successful
// observation cannot install that authority. Only disposable tests bind it now.
// The callback returns independently owned current records and native facts.
type propertyControl struct {
	mu        sync.Mutex
	authorize func(context.Context, storage.PropertyIntent) (storage.PropertyJob, storage.Policy, storage.Observation, error)
	run       func(context.Context, storage.PropertyIntent) error
	root      func() bool
	clock     func() time.Time
	attempted bool
}

func (c *propertyControl) apply(ctx context.Context, i storage.PropertyIntent) (bool, error) {
	if ctx == nil {
		return false, storage.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if c == nil {
		return false, ErrUnavailable
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.authorize == nil || c.run == nil || c.root == nil || c.clock == nil {
		return false, ErrUnavailable
	}
	if !c.root() {
		return false, ErrPermission
	}
	if c.attempted {
		return false, storage.ErrDenied
	}
	c.attempted = true
	encoded, err := storage.EncodePropertyIntent(i)
	if err != nil {
		return false, err
	}
	before := c.clock().Round(0).UTC()
	if before.IsZero() || !before.Before(i.Review().ExpiresAt) || before.Before(i.Review().CreatedAt) {
		return false, storage.ErrExpired
	}
	ctx, stop := context.WithTimeout(ctx, 145*time.Second)
	defer stop()
	job, policy, observation, err := c.authorize(ctx, i)
	if err != nil {
		return false, err
	}
	if _, err = storage.EncodePropertyJob(job); err != nil {
		return false, err
	}
	history := job.Job().History
	if len(history) != 2 || history[1].State != storage.JobApplying {
		return false, storage.ErrDenied
	}
	bound, err := storage.EncodePropertyIntent(job.Intent())
	if err != nil || !bytes.Equal(encoded, bound) {
		return false, storage.ErrDrift
	}
	assessment, err := job.AssessObservation(policy, observation)
	if err != nil {
		return false, err
	}
	if !assessment.BaselineMatches {
		return false, storage.ErrDrift
	}
	// AssessObservation accepts read-only facts without changing eligibility.
	// Write authority comes ONLY from the original private backend claim above.
	if _, err = propertySetArguments(i); err != nil {
		return false, err
	}
	after := c.clock().Round(0).UTC()
	if after.Before(before) || !after.Before(i.Review().ExpiresAt) || after.UnixMilli() < int64(history[1].At) {
		return false, storage.ErrExpired
	}
	if err = ctx.Err(); err != nil {
		return false, err
	}
	if err = c.run(ctx, i); err != nil {
		return false, err
	}
	// Only the fixed native runner's normal successful guard acknowledgement can
	// establish this narrow quiescence result. Any error remains unverified.
	return true, nil
}

// Validated immutable selection only: exactly one property on one captured GUID
// address, without flags, recursion, arbitrary executable/environment or shell.
// The private authorization callback must freshly match the ENTIRE baseline.
func propertySetArguments(i storage.PropertyIntent) ([]string, error) {
	if _, err := storage.EncodePropertyIntent(i); err != nil {
		return nil, err
	}
	r := i.Review()
	switch r.Intent.Property {
	case storage.Quota, storage.RefQuota, storage.Compression:
	default:
		return nil, storage.ErrDenied
	}
	name := ""
	for _, f := range r.Context {
		if f.GUID == r.Intent.FilesystemGUID {
			name = f.Name
		}
	}
	if name == "" {
		return nil, storage.ErrDenied
	}
	return []string{"set", string(r.Intent.Property) + "=" + r.Intent.Value, name}, nil
}
