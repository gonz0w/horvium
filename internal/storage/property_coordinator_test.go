// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// Only a fixture supplies eligibility here. No product native adapter exists.
type coordinatorFixture struct {
	policy                     Policy
	observation                Observation
	observations, publications int
	observe                    func()
	observeErr, publishErr     error
	returned                   *PropertyJob
	duplicate                  bool
	publish                    func(PropertyJob) (PropertyJob, bool, error)
}

func (b *coordinatorFixture) ObserveProperty(ctx context.Context, _ Intent) (Policy, Observation, error) {
	b.observations++
	if b.observe != nil {
		b.observe()
	}
	return b.policy, b.observation, b.observeErr
}
func (b *coordinatorFixture) PublishPropertySubmission(p PropertyJob) (PropertyJob, bool, error) {
	b.publications++
	if b.publish != nil {
		return b.publish(p)
	}
	if b.returned != nil {
		return *b.returned, b.duplicate, b.publishErr
	}
	return p, false, b.publishErr
}
func coordinatorSetup(t testing.TB) (*PropertyCoordinator, *coordinatorFixture, *time.Time, Intent) {
	t.Helper()
	p, o, i := fixture()
	b := &coordinatorFixture{policy: p, observation: o}
	c := NewPropertyCoordinator(b)
	now := instant.Add(123456789 * time.Nanosecond)
	c.clock = func() time.Time { return now }
	return c, b, &now, i
}

func TestPropertyCoordinatorReviewToSubmission(t *testing.T) {
	c, b, now, intent := coordinatorSetup(t)
	r, err := c.Review(context.Background(), "session-a", intent)
	if err != nil {
		t.Fatal(err)
	}
	if !r.CreatedAt.Equal(now.Truncate(time.Millisecond)) || len(r.Affected) != 2 {
		t.Fatal("lost bounded review")
	}
	r.Context[0].Name = "caller-edited"
	*now = now.Add(time.Second)
	job, duplicate, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("a", 64))
	if err != nil || duplicate || b.publications != 1 || b.observations != 2 {
		t.Fatal("review did not refresh/admit once", err)
	}
	if job.Job().Intent.Change != intent || job.Intent().Review().Context[0].Name != "tank" || job.Job().History[0].At != uint64(now.UnixMilli()) {
		t.Fatal("changed reviewed selection/time")
	}
	job.job.History[0].Reason = "caller-edited"
	job.intent.review.Context[0].Name = "caller-edited"
	again, duplicate, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("a", 64))
	if err != nil || !duplicate || b.publications != 1 || b.observations != 2 || again.Job().History[0].Reason != "admitted" || again.Intent().Review().Context[0].Name != "tank" {
		t.Fatal("retry changed/republished acknowledgement", err)
	}
	for _, input := range []struct{ session, submission string }{{"other", strings.Repeat("a", 64)}, {"session-a", strings.Repeat("b", 64)}} {
		if result, dup, err := c.Submit(context.Background(), input.session, r.ID, input.submission); !errors.Is(err, ErrDenied) || dup || !reflect.DeepEqual(result, PropertyJob{}) {
			t.Fatal("contradictory retry escaped", err)
		}
	}
}

func TestPropertyCoordinatorConcurrentConfirm(t *testing.T) {
	c, b, _, i := coordinatorSetup(t)
	r, err := c.Review(context.Background(), "session-a", i)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var first string
	newCount := 0
	for range 32 {
		wg.Go(func() {
			p, dup, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("a", 64))
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				t.Error(err)
				return
			}
			if first == "" {
				first = p.Job().ID
			}
			if first != p.Job().ID {
				t.Error("multiple jobs")
			}
			if !dup {
				newCount++
			}
		})
	}
	wg.Wait()
	if newCount != 1 || b.publications != 1 || b.observations != 2 {
		t.Fatal("duplicate durable attempt", newCount, b.publications)
	}
}

func TestPropertyCoordinatorFreshDenials(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*coordinatorFixture, *time.Time)
		want   error
	}{
		{"policy", func(b *coordinatorFixture, _ *time.Time) { b.policy.Revision++ }, ErrDrift},
		{"property", func(b *coordinatorFixture, _ *time.Time) {
			b.observation.Filesystems[0].Properties.Quota = Value{"1", Local, "101", ""}
		}, ErrDrift},
		{"incomplete", func(b *coordinatorFixture, _ *time.Time) { b.observation.Complete = false }, ErrDrift},
		{"ineligible", func(b *coordinatorFixture, _ *time.Time) { b.observation.Eligible = false }, ErrDrift},
		{"expired during observation", func(b *coordinatorFixture, now *time.Time) { b.observe = func() { *now = now.Add(5 * time.Minute) } }, ErrExpired},
		{"cancel during observation", func(b *coordinatorFixture, _ *time.Time) { b.observeErr = context.Canceled }, context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, b, now, i := coordinatorSetup(t)
			r, err := c.Review(context.Background(), "session-a", i)
			if err != nil {
				t.Fatal(err)
			}
			tc.change(b, now)
			p, dup, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("a", 64))
			if !errors.Is(err, tc.want) || dup || !reflect.DeepEqual(p, PropertyJob{}) || b.publications != 0 {
				t.Fatal("denial published/exposed job", err)
			}
		})
	}
}

func TestPropertyCoordinatorUncertainPublication(t *testing.T) {
	for _, kind := range []string{"failure", "foreign acknowledgement", "invalid acknowledgement", "false duplicate"} {
		t.Run(kind, func(t *testing.T) {
			c, b, _, i := coordinatorSetup(t)
			r, err := c.Review(context.Background(), "session-a", i)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "failure":
				b.publishErr = ErrStateUncertain
			case "foreign acknowledgement":
				p := propertyJobFixture(t)
				b.returned = &p
			case "invalid acknowledgement":
				p := PropertyJob{}
				b.returned = &p
			case "false duplicate":
				b.duplicate = true
				p := PropertyJob{}
				b.returned = &p
			}
			p, dup, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("a", 64))
			if err == nil || dup || !reflect.DeepEqual(p, PropertyJob{}) || b.publications != 1 {
				t.Fatal("uncertain acknowledgement accepted", err)
			}
			if _, _, err = c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("a", 64)); !errors.Is(err, ErrStateUncertain) || b.publications != 1 {
				t.Fatal("uncertain attempt replayed", err)
			}
			r2, err := c.Review(context.Background(), "session-a", i)
			if err != nil {
				t.Fatal("read-only review disabled", err)
			}
			if _, _, err = c.Submit(context.Background(), "session-a", r2.ID, strings.Repeat("b", 64)); !errors.Is(err, ErrStateUncertain) || b.publications != 1 {
				t.Fatal("another plan bypassed uncertainty", err)
			}
		})
	}
}

func TestPropertyCoordinatorBoundsExpiryAndRestart(t *testing.T) {
	c, b, now, i := coordinatorSetup(t)
	var first Review
	for n := 0; n < 64; n++ {
		r, err := c.Review(context.Background(), "session-a", i)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			first = r
		}
	}
	if _, err := c.Review(context.Background(), "session-a", i); !errors.Is(err, ErrLimit) || b.observations != 64 {
		t.Fatal("unbounded plans/native collection", err)
	}
	*now = first.ExpiresAt
	if _, _, err := c.Submit(context.Background(), "session-a", first.ID, strings.Repeat("a", 64)); !errors.Is(err, ErrExpired) || b.publications != 0 {
		t.Fatal("expired plan published", err)
	}
	r, err := c.Review(context.Background(), "session-a", i)
	if err != nil {
		t.Fatal("expired capacity not reclaimed", err)
	}
	fresh := NewPropertyCoordinator(b)
	fresh.clock = c.clock
	if _, _, err := fresh.Submit(context.Background(), "session-a", r.ID, strings.Repeat("a", 64)); !errors.Is(err, ErrDenied) {
		t.Fatal("restart restored volatile review", err)
	}
	*now = now.Add(-time.Second)
	if _, _, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("a", 64)); !errors.Is(err, ErrExpired) {
		t.Fatal("clock rollback accepted", err)
	}
	*now = now.Add(time.Second)
	if _, err := c.Review(context.Background(), "session-a", i); !errors.Is(err, ErrExpired) {
		t.Fatal("rollback silently recovered", err)
	}
}

func TestPropertyCoordinatorInvalidInputs(t *testing.T) {
	c, b, _, i := coordinatorSetup(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, input := range []struct {
		ctx     context.Context
		session string
		intent  Intent
	}{{nil, "session-a", i}, {ctx, "session-a", i}, {context.Background(), "", i}, {context.Background(), "session-a", Intent{}}} {
		if _, err := c.Review(input.ctx, input.session, input.intent); err == nil {
			t.Fatal("invalid review accepted")
		}
	}
	if b.observations != 0 {
		t.Fatal("invalid request collected native inputs")
	}
	var zero PropertyCoordinator
	if _, err := zero.Review(context.Background(), "session-a", i); !errors.Is(err, ErrUnavailable) {
		t.Fatal("zero backend enabled", err)
	}
	r, err := c.Review(context.Background(), "session-a", i)
	if err != nil {
		t.Fatal(err)
	}
	for _, ids := range [][2]string{{"bad", strings.Repeat("a", 64)}, {r.ID, "bad"}, {strings.Repeat("b", 64), strings.Repeat("a", 64)}} {
		if p, dup, err := c.Submit(context.Background(), "session-a", ids[0], ids[1]); err == nil || dup || !reflect.DeepEqual(p, PropertyJob{}) {
			t.Fatal("invalid submission accepted")
		}
	}
	if _, _, err := c.Submit(ctx, "session-a", r.ID, strings.Repeat("a", 64)); !errors.Is(err, context.Canceled) || b.publications != 0 {
		t.Fatal("cancelled submission published", err)
	}
}

func TestPropertyCoordinatorRetainedDuplicateAndLateAcknowledgement(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		t.Run(map[bool]string{false: "new after deadline", true: "existing current record"}[duplicate], func(t *testing.T) {
			c, b, now, i := coordinatorSetup(t)
			r, err := c.Review(context.Background(), "session-a", i)
			if err != nil {
				t.Fatal(err)
			}
			b.publish = func(candidate PropertyJob) (PropertyJob, bool, error) {
				if !duplicate {
					*now = r.ExpiresAt
					return candidate, false, nil
				}
				j := candidate.Job()
				j.ID = strings.Repeat("f", 64)
				current, err := BindPropertyJob(j, candidate.Intent())
				if err != nil {
					t.Fatal(err)
				}
				current, err = TransitionPropertyJob(current, JobCancelled, j.History[0].At, "operator-cancelled", "")
				if err != nil {
					t.Fatal(err)
				}
				return current, true, nil
			}
			p, dup, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("a", 64))
			if err != nil || dup != duplicate || b.publications != 1 {
				t.Fatal("lost durable acknowledgement", err)
			}
			if duplicate && p.Job().History[1].State != JobCancelled {
				t.Fatal("discarded existing history")
			}
		})
	}
}

func TestPropertyCoordinatorCancellationAndClockDuringCollection(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "clock rollback"}[rollback], func(t *testing.T) {
			c, b, now, i := coordinatorSetup(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r, err := c.Review(ctx, "session-a", i)
			if err != nil {
				t.Fatal(err)
			}
			want := error(context.Canceled)
			b.observe = cancel
			if rollback {
				want = ErrExpired
				b.observe = func() { *now = now.Add(-time.Second) }
			}
			p, dup, err := c.Submit(ctx, "session-a", r.ID, strings.Repeat("a", 64))
			if !errors.Is(err, want) || dup || !reflect.DeepEqual(p, PropertyJob{}) || b.publications != 0 {
				t.Fatal("changed context/clock published", err)
			}
		})
	}
}

func TestPropertyCoordinatorCancelledWaiter(t *testing.T) {
	c, b, _, i := coordinatorSetup(t)
	r, err := c.Review(context.Background(), "session-a", i)
	if err != nil {
		t.Fatal(err)
	}
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	b.publish = func(p PropertyJob) (PropertyJob, bool, error) { close(entered); <-release; return p, false, nil }
	go func() {
		defer close(finished)
		if _, _, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("a", 64)); err != nil {
			t.Error(err)
		}
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	waiter := make(chan error, 1)
	go func() { _, _, err := c.Submit(ctx, "session-a", r.ID, strings.Repeat("a", 64)); waiter <- err }()
	cancel()
	close(release)
	<-finished
	if err := <-waiter; !errors.Is(err, context.Canceled) || b.publications != 1 {
		t.Fatal("cancelled waiter acknowledged/published", err)
	}
}

func FuzzPropertyCoordinatorActions(f *testing.F) {
	f.Add([]byte{0, 1, 1, 2, 3, 4, 5, 6})
	f.Add([]byte{0, 3, 1, 0, 1, 7, 1})
	f.Fuzz(func(t *testing.T, actions []byte) {
		if len(actions) > 64 {
			return
		}
		c, b, now, i := coordinatorSetup(t)
		var review Review
		published := map[string]bool{}
		b.publish = func(candidate PropertyJob) (PropertyJob, bool, error) {
			id := candidate.Job().Intent.PlanID
			if published[id] {
				t.Fatal("second publication for one plan")
			}
			published[id] = true
			return candidate, false, nil
		}
		for _, action := range actions {
			switch action % 8 {
			case 0:
				r, err := c.Review(context.Background(), "session-a", i)
				if err == nil {
					review = r
				}
			case 1, 2:
				session := "session-a"
				if action%8 == 2 {
					session = "other"
				}
				p, dup, err := c.Submit(context.Background(), session, review.ID, strings.Repeat("a", 64))
				if err != nil && (dup || !reflect.DeepEqual(p, PropertyJob{})) {
					t.Fatal("denial exposed acknowledgement")
				}
				if err == nil {
					if _, err := EncodePropertyJob(p); err != nil {
						t.Fatal("invalid acknowledgement")
					}
				}
			case 3:
				*now = now.Add(time.Minute)
			case 4:
				b.policy.Revision++
			case 5:
				b.observation.Complete = !b.observation.Complete
			case 6:
				*now = now.Add(-time.Second)
			case 7:
				b.publishErr = ErrStateUncertain
				b.publish = nil
			}
		}
		if len(c.reviews) > maxPropertyReviews {
			t.Fatal("unbounded retained reviews")
		}
	})
}
