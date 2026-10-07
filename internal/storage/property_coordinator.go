// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const maxPropertyReviews = 64

// PropertyBackend is a trusted helper-side boundary, never a request adapter.
// ObserveProperty must independently read current root-owned policy and collect
// complete fresh native eligibility/impact. PublishPropertySubmission must use
// the held root StateDirectory lease. No product adapter implements this full
// boundary yet; installing a mock does not establish native/root authority.
type PropertyBackend interface {
	ObserveProperty(context.Context, Intent) (Policy, Observation, error)
	PublishPropertySubmission(PropertyJob) (PropertyJob, bool, error)
}

type propertyReviewEntry struct {
	plan       Plan
	ack        PropertyJob
	submission string
	// An existing-store retry is not this coordinator's fresh admission. A
	// successful new admission can hand off its original live claim only once.
	duplicateAdmission, executionClaimed bool
}

// PropertyCoordinator connects volatile review to metadata submission. It owns
// at most 64 reviews/acknowledgements, serializes confirms and starts no worker.
// It executes no storage command. It must stay inside a kernel-authorized helper;
// callers must independently authenticate the current application session.
// Do not copy it. Restart discards every plan; retained durable jobs require the
// separate recovery/lookup gates and cannot restore execution authority here.
type PropertyCoordinator struct {
	mu                               sync.Mutex
	backend                          PropertyBackend
	clock                            func() time.Time
	last                             time.Time
	clockBlocked, publicationBlocked bool
	reviews                          map[string]propertyReviewEntry
}

func NewPropertyCoordinator(backend PropertyBackend) *PropertyCoordinator {
	return &PropertyCoordinator{backend: backend, clock: time.Now, reviews: make(map[string]propertyReviewEntry)}
}

func (c *PropertyCoordinator) now() (time.Time, error) {
	if c.clockBlocked {
		return time.Time{}, ErrExpired
	}
	// Compare civil time explicitly: Time.Before would otherwise prefer the
	// monotonic component and hide a backwards wall-clock jump from deadlines.
	at := c.clock().Round(0).UTC()
	if !canonicalIntentTime(at) || at.Before(c.last) {
		c.clockBlocked = true
		return time.Time{}, ErrExpired
	}
	c.last = at
	return at, nil
}

func coordinatorContext(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalid
	}
	return ctx.Err()
}

// Review obtains trusted fresh inputs, builds an independent immutable review
// and retains the private plan. Millisecond creation is conservative and matches
// the durable lifecycle clock. Expired entries are removed only to bound volatile
// memory; durable records/holds are never deleted or released.
func (c *PropertyCoordinator) Review(ctx context.Context, session string, intent Intent) (Review, error) {
	if err := coordinatorContext(ctx); err != nil {
		return Review{}, err
	}
	if !token(session) || !validIntent(intent) {
		return Review{}, ErrInvalid
	}
	if c == nil {
		return Review{}, ErrUnavailable
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := coordinatorContext(ctx); err != nil {
		return Review{}, err
	}
	if c.backend == nil || c.clock == nil || c.reviews == nil {
		return Review{}, ErrUnavailable
	}
	at, err := c.now()
	if err != nil {
		return Review{}, err
	}
	for id, e := range c.reviews {
		if !at.Before(e.plan.review.ExpiresAt) {
			delete(c.reviews, id)
		}
	}
	if len(c.reviews) >= maxPropertyReviews {
		return Review{}, ErrLimit
	}
	policy, observation, err := c.backend.ObserveProperty(ctx, intent)
	if err != nil {
		return Review{}, err
	}
	if err = coordinatorContext(ctx); err != nil {
		return Review{}, err
	}
	at, err = c.now()
	if err != nil {
		return Review{}, err
	}
	plan, err := Build(at.Truncate(time.Millisecond), session, policy, observation, intent)
	if err != nil {
		return Review{}, err
	}
	if _, exists := c.reviews[plan.review.ID]; exists {
		return Review{}, ErrUnavailable
	}
	c.reviews[plan.review.ID] = propertyReviewEntry{plan: plan}
	return plan.Review(), nil
}

// Submit refreshes trusted policy/observations and checks time/session before
// claiming one plan for one metadata publication attempt. It accepts identifiers
// only, never a client-supplied review, policy, observation, job or clock.
// Matching confirms return owned copies of the retained admission acknowledgement
// while this plan remains live; they are not current status or a fresh sync ack.
// After expiry/restart, use the future durable lookup/recovery path instead.
// Any publication error or invalid acknowledgement blocks all further submissions
// on this coordinator. Review stays available; no automatic publication retry,
// orphan repair, storage launch or reservation release is performed.
func (c *PropertyCoordinator) Submit(ctx context.Context, session, planID, submissionID string) (PropertyJob, bool, error) {
	if err := coordinatorContext(ctx); err != nil {
		return PropertyJob{}, false, err
	}
	if !token(session) || !validJobDigest(planID) || !validJobDigest(submissionID) {
		return PropertyJob{}, false, ErrInvalid
	}
	if c == nil {
		return PropertyJob{}, false, ErrUnavailable
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := coordinatorContext(ctx); err != nil {
		return PropertyJob{}, false, err
	}
	if c.backend == nil || c.clock == nil || c.reviews == nil {
		return PropertyJob{}, false, ErrUnavailable
	}
	at, err := c.now()
	if err != nil {
		return PropertyJob{}, false, err
	}
	entry, exists := c.reviews[planID]
	if !exists || entry.plan.session != session {
		return PropertyJob{}, false, ErrDenied
	}
	if !at.Before(entry.plan.review.ExpiresAt) {
		return PropertyJob{}, false, ErrExpired
	}
	if c.publicationBlocked {
		return PropertyJob{}, false, ErrStateUncertain
	}
	if entry.submission != "" {
		if entry.submission != submissionID {
			return PropertyJob{}, false, ErrDenied
		}
		return clonePropertyJob(entry.ack), true, nil
	}
	policy, observation, err := c.backend.ObserveProperty(ctx, entry.plan.review.Intent)
	if err != nil {
		return PropertyJob{}, false, err
	}
	if err = coordinatorContext(ctx); err != nil {
		return PropertyJob{}, false, err
	}
	at, err = c.now()
	if err != nil {
		return PropertyJob{}, false, err
	}
	if err = entry.plan.Validate(at, session, policy, observation); err != nil {
		return PropertyJob{}, false, err
	}
	intent, err := entry.plan.CapturePropertyIntent(at.Truncate(time.Millisecond), session, policy, observation)
	if err != nil {
		return PropertyJob{}, false, err
	}
	var randomID [32]byte
	if _, err = rand.Read(randomID[:]); err != nil {
		return PropertyJob{}, false, ErrUnavailable
	}
	job := Job{ID: hex.EncodeToString(randomID[:]), Intent: JobIntent{PlanID: planID, SubmissionID: submissionID,
		PolicyRevision: policy.Revision, PoolGUID: policy.PoolGUID, Change: entry.plan.review.Intent},
		History: []JobTransition{{Revision: 1, At: uint64(at.UnixMilli()), State: JobQueued, Reason: "admitted"}}}
	candidate, err := BindPropertyJob(job, intent)
	if err != nil {
		return PropertyJob{}, false, err
	}
	// Capture/encoding can take time. The final trusted clock/context check must
	// still pass before calling the metadata publisher; no HTTP clock is accepted.
	final, err := c.now()
	if err != nil {
		return PropertyJob{}, false, err
	}
	if !final.Before(entry.plan.review.ExpiresAt) {
		return PropertyJob{}, false, ErrExpired
	}
	if err = coordinatorContext(ctx); err != nil {
		return PropertyJob{}, false, err
	}
	entry.submission = submissionID
	c.reviews[planID] = entry
	ack, duplicate, err := c.backend.PublishPropertySubmission(candidate)
	if err != nil {
		c.publicationBlocked = true
		return PropertyJob{}, false, err
	}
	if !validPropertyAcknowledgement(candidate, ack, duplicate) {
		c.publicationBlocked = true
		return PropertyJob{}, false, ErrStateUncertain
	}
	entry.ack = clonePropertyJob(ack)
	entry.duplicateAdmission = duplicate
	c.reviews[planID] = entry
	return clonePropertyJob(ack), duplicate, nil
}

func clonePropertyJob(p PropertyJob) PropertyJob {
	return PropertyJob{job: p.Job(), intent: p.Intent()}
}

func validPropertyAcknowledgement(candidate, ack PropertyJob, duplicate bool) bool {
	data, err := EncodePropertyJob(ack)
	if err != nil {
		return false
	}
	if !duplicate {
		expected, err := EncodePropertyJob(candidate)
		return err == nil && bytes.Equal(data, expected)
	}
	view, err := BuildJobReservations([][]byte{data})
	if err != nil {
		return false
	}
	_, matched, err := view.CheckPropertySubmission(candidate)
	return err == nil && matched
}
