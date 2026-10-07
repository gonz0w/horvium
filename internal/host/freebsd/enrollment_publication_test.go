// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

// These orchestration tests do not forge a root binding or qualify publication.
// The public method's real sealed binding/metadata behavior is tested on FreeBSD.
func enrollmentPublicationReview(t *testing.T, s *PropertyEnrollmentSource) (storage.PropertyEnrollmentPlan, storage.PropertyEnrollmentReview) {
	t.Helper()
	p := safetyPolicy()
	o, d, u, err := s.collect(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := storage.BuildPropertyEnrollment(enrollmentInstant, "root", p, o, d, u)
	if err != nil {
		t.Fatal(err)
	}
	return plan, plan.Review()
}

func TestPropertyEnrollmentSourcePublicationRecollectsAndRetainsAcknowledgement(t *testing.T) {
	s := enrollmentSourceFixture(nil)
	plan, r := enrollmentPublicationReview(t, s)
	reads := 0
	original := s.pool.read
	s.pool.read = func(ctx context.Context, p poolProbe, name string, budget int) (readOutput, error) {
		reads++
		return original(ctx, p, name, budget)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := 0
	err := s.publishReview(ctx, "root", r, storage.Policy{}, func(ctx context.Context, o storage.Observation, d []inventory.DiskObservation, u storage.PoolSystemUse) error {
		called++
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 135*time.Second || reads != 16 || o.Eligible || len(d) != 2 || len(u.Pools()) != 1 {
			t.Fatal("missing fresh bounded native inputs")
		}
		if err := plan.Validate(enrollmentInstant, "root", r.Policy, o, d, u); err != nil {
			t.Fatal(err)
		}
		// Publication's actual acknowledgement must survive later cancellation/expiry.
		cancel()
		s.now = func() time.Time { return enrollmentInstant.Add(5 * time.Minute) }
		return nil
	})
	if err != nil || called != 1 {
		t.Fatal("successful acknowledgement lost", err)
	}
}

func TestPropertyEnrollmentSourcePublicationRefusesBeforeAttempt(t *testing.T) {
	for _, mode := range []string{"zero-review", "nil-context", "cancelled", "invalid-context", "expired", "backwards", "missing-clock", "missing-source", "native-failure", "provider-failure", "system-failure", "expired-after-collect", "backwards-after-collect", "cancelled-after-collect", "nil-publisher"} {
		t.Run(mode, func(t *testing.T) {
			s := enrollmentSourceFixture(nil)
			_, r := enrollmentPublicationReview(t, s)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			root := "root"
			after := false
			original := s.system.read
			s.system.read = func(ctx context.Context, p systemProbe, name string, budget int) (readOutput, error) {
				out, err := original(ctx, p, name, budget)
				after = true
				if mode == "cancelled-after-collect" {
					cancel()
				}
				return out, err
			}
			switch mode {
			case "zero-review":
				r = storage.PropertyEnrollmentReview{}
			case "nil-context":
				ctx = nil
			case "cancelled":
				cancel()
			case "invalid-context":
				root = "invalid context"
			case "expired":
				s.now = func() time.Time { return r.ExpiresAt }
			case "backwards":
				s.now = func() time.Time { return r.CreatedAt.Add(-time.Second) }
			case "missing-clock":
				s.now = nil
			case "missing-source":
				s = nil
			case "native-failure":
				s.pool.read = func(context.Context, poolProbe, string, int) (readOutput, error) { return readOutput{}, ErrUnavailable }
			case "provider-failure":
				s.providers.source.read = func(context.Context, systemProbe, string, int) (readOutput, error) {
					return readOutput{}, ErrUnavailable
				}
			case "system-failure":
				s.system.read = func(context.Context, systemProbe, string, int) (readOutput, error) {
					return readOutput{}, ErrUnavailable
				}
			case "expired-after-collect", "backwards-after-collect":
				s.now = func() time.Time {
					if after {
						if mode == "expired-after-collect" {
							return r.ExpiresAt
						}
						return r.CreatedAt.Add(-time.Second)
					}
					return enrollmentInstant
				}
			}
			publish := func(context.Context, storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse) error {
				t.Fatal("denial reached publication")
				return nil
			}
			if mode == "nil-publisher" {
				publish = nil
			}
			if err := s.publishReview(ctx, root, r, storage.Policy{}, publish); err == nil {
				t.Fatal("invalid publication accepted")
			}
		})
	}
}

func TestPropertyEnrollmentSourcePublicationRetainsFailure(t *testing.T) {
	s := enrollmentSourceFixture(nil)
	_, r := enrollmentPublicationReview(t, s)
	for _, expected := range []error{storage.ErrDrift, storage.ErrDenied, storage.ErrStateUncertain, context.Canceled} {
		called := 0
		err := s.publishReview(context.Background(), "root", r, storage.Policy{}, func(context.Context, storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse) error {
			called++
			return expected
		})
		if !errors.Is(err, expected) || called != 1 {
			t.Fatal("publication outcome changed", err)
		}
	}
}

func TestPropertyEnrollmentSourcePublicationCannotHydrateBinding(t *testing.T) {
	s := enrollmentSourceFixture(nil)
	s.pool.read = func(context.Context, poolProbe, string, int) (readOutput, error) {
		t.Fatal("zero binding reached I/O")
		return readOutput{}, nil
	}
	if err := s.Publish(context.Background(), "root", storage.PropertyEnrollmentState{}); !errors.Is(err, storage.ErrInvalid) {
		t.Fatal(err)
	}
}

// Preserve native source/value consistency so drift reaches the sealed review
// validator rather than merely failing the native projection parser.
func enrollmentPropertyDriftOutput(out readOutput) readOutput {
	return testOutput(strings.ReplaceAll(string(out.data), "\tquota\t0\t-\tdefault", "\tquota\t131072\t-\tlocal"))
}

func TestPropertyEnrollmentSourcePublicationPassesFreshDriftToSealedValidation(t *testing.T) {
	s := enrollmentSourceFixture(nil)
	plan, r := enrollmentPublicationReview(t, s)
	original := s.pool.read
	s.pool.read = func(ctx context.Context, p poolProbe, name string, budget int) (readOutput, error) {
		out, err := original(ctx, p, name, budget)
		if p == poolProperties && err == nil {
			out = enrollmentPropertyDriftOutput(out)
		}
		return out, err
	}
	called := false
	err := s.publishReview(context.Background(), "root", r, storage.Policy{}, func(_ context.Context, o storage.Observation, d []inventory.DiskObservation, u storage.PoolSystemUse) error {
		called = true
		return plan.Validate(enrollmentInstant, "root", r.Policy, o, d, u)
	})
	if !called || !errors.Is(err, storage.ErrDrift) {
		t.Fatal("drift fixture did not reach sealed validation", err)
	}
}
