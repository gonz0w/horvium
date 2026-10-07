// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/inventory"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func enrollmentPublicationFixture(t *testing.T) (PropertyEnrollmentState, Observation, []inventory.DiskObservation, PoolSystemUse) {
	t.Helper()
	p, o, d, u := enrollmentFixture(t)
	plan, e := BuildPropertyEnrollment(instant, "root", p, o, d, u)
	if e != nil {
		t.Fatal(e)
	}
	return PropertyEnrollmentState{lease: &StateDirectory{}, plan: plan, claim: plan.claim}, o, d, u
}

func TestEnrollmentPublicationOneAttemptAcrossCopies(t *testing.T) {
	bound, o, d, u := enrollmentPublicationFixture(t)
	var calls atomic.Int32
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			copy := bound
			err := copy.publish(context.Background(), "root", o, d, u, func() time.Time { return instant }, func() error { calls.Add(1); return nil })
			if err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrDenied) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 || successes.Load() != 1 {
		t.Fatal("replayed one review", calls.Load(), successes.Load())
	}
}

func TestEnrollmentPublicationDenialBeforeClaim(t *testing.T) {
	for _, mode := range []string{"nil-context", "cancelled", "expired", "backwards", "context", "native", "system", "zero", "missing-claim", "missing-publisher", "missing-clock"} {
		t.Run(mode, func(t *testing.T) {
			bound, o, d, u := enrollmentPublicationFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			root := "root"
			clock := func() time.Time { return instant }
			publish := func() error { t.Fatal("denial invoked publisher"); return nil }
			switch mode {
			case "nil-context":
				ctx = nil
			case "cancelled":
				cancel()
			case "expired":
				clock = func() time.Time { return instant.Add(5 * time.Minute) }
			case "backwards":
				clock = func() time.Time { return instant.Add(-time.Second) }
			case "context":
				root = "other"
			case "native":
				o.Eligible = true
			case "system":
				u = PoolSystemUse{}
			case "zero":
				bound = PropertyEnrollmentState{}
			case "missing-claim":
				bound.claim = nil
			case "missing-publisher":
				publish = nil
			case "missing-clock":
				clock = nil
			}
			if err := bound.publish(ctx, root, o, d, u, clock, publish); err == nil {
				t.Fatal("denial returned success")
			}
			if bound.claim != nil && bound.claim.attempted {
				t.Fatal("preflight burned claim")
			}
		})
	}
}

func TestEnrollmentPublicationErrorBurnsClaimAndLateSuccessIsReported(t *testing.T) {
	for _, fail := range []bool{false, true} {
		bound, o, d, u := enrollmentPublicationFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		stop := errors.New("publication failure")
		err := bound.publish(ctx, "root", o, d, u, func() time.Time { return instant }, func() error {
			calls++
			cancel()
			if fail {
				return stop
			}
			return nil
		})
		if fail && !errors.Is(err, stop) || !fail && err != nil {
			t.Fatal("incorrect durable acknowledgement", err)
		}
		if e := bound.publish(context.Background(), "root", o, d, u, func() time.Time { return instant }, func() error { calls++; return nil }); !errors.Is(e, ErrDenied) || calls != 1 {
			t.Fatal("failed/successful attempt replayed", e, calls)
		}
	}
}
