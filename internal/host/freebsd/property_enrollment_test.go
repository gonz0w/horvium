// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

var enrollmentInstant = time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)

func enrollmentSourceFixture(bind func(time.Time, string, storage.PropertyEnrollmentPlan, storage.Policy, storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse) (storage.PropertyEnrollmentState, error)) *PropertyEnrollmentSource {
	rows := safetyCaptures()
	return &PropertyEnrollmentSource{pool: policySource(nil).pool, providers: safetyProvider(), system: useSource(func(_ context.Context, p systemProbe, _ string, _ int) (readOutput, error) {
		return testOutput(rows[p]), nil
	}), bind: bind, now: func() time.Time { return enrollmentInstant }}
}

func TestPropertyEnrollmentSourceOwnsSelectionAndRequiresTrustedBinding(t *testing.T) {
	p := safetyPolicy()
	stop := errors.New("fixture stops at trusted binding")
	called := false
	s := enrollmentSourceFixture(func(at time.Time, root string, plan storage.PropertyEnrollmentPlan, previous storage.Policy, o storage.Observation, d []inventory.DiskObservation, u storage.PoolSystemUse) (storage.PropertyEnrollmentState, error) {
		called = true
		r := plan.Review()
		if at != enrollmentInstant || root != "root" || r.Policy.Members[0].Serial == "changed" || len(d) != 2 || o.Eligible || len(u.Pools()) != 1 || !reflect.DeepEqual(previous, storage.Policy{}) {
			t.Fatal("unbound/authoritative/aliased inputs")
		}
		if err := plan.Validate(at, root, r.Policy, o, d, u); err != nil {
			t.Fatal(err)
		}
		return storage.PropertyEnrollmentState{}, stop
	})
	original := s.pool.read
	s.pool.read = func(ctx context.Context, probe poolProbe, name string, budget int) (readOutput, error) {
		p.Members[0].Serial = "changed"
		return original(ctx, probe, name, budget)
	}
	bound, err := s.Review(context.Background(), "root", p, storage.Policy{})
	if !called || !errors.Is(err, stop) || !reflect.DeepEqual(bound, storage.PropertyEnrollmentState{}) {
		t.Fatal("missing trusted binding or partial output", err)
	}
}

func TestPropertyEnrollmentSourceSelectionDeniesBeforeNativeIO(t *testing.T) {
	for _, mode := range []string{"context", "nil-context", "canceled", "pool", "scope", "members", "predecessor", "partial-zero", "overflow", "zero-source"} {
		t.Run(mode, func(t *testing.T) {
			p, previous := safetyPolicy(), storage.Policy{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			root := "root"
			s := enrollmentSourceFixture(func(time.Time, string, storage.PropertyEnrollmentPlan, storage.Policy, storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse) (storage.PropertyEnrollmentState, error) {
				t.Fatal("unexpected binding")
				return storage.PropertyEnrollmentState{}, nil
			})
			s.pool.read = func(context.Context, poolProbe, string, int) (readOutput, error) {
				t.Fatal("invalid selection reached native I/O")
				return readOutput{}, nil
			}
			switch mode {
			case "context":
				root = "invalid root context"
			case "nil-context":
				ctx = nil
			case "canceled":
				cancel()
			case "pool":
				p.PoolGUID = "-1"
			case "scope":
				p.Scopes = nil
			case "members":
				p.Members = nil
			case "predecessor":
				previous = p
			case "partial-zero":
				previous.PoolGUID = p.PoolGUID
			case "overflow":
				previous = p
				previous.Revision = ^uint64(0)
			case "zero-source":
				s = nil
			}
			bound, err := s.Review(ctx, root, p, previous)
			if err == nil || !reflect.DeepEqual(bound, storage.PropertyEnrollmentState{}) {
				t.Fatal("invalid proposal granted state", err)
			}
		})
	}
}

func TestPropertyEnrollmentSourceOwnsPredecessorBeforeNativeIO(t *testing.T) {
	previous := safetyPolicy()
	proposed, err := copyEnrollmentPolicy(previous)
	if err != nil {
		t.Fatal(err)
	}
	proposed.Revision++
	stop := errors.New("owned predecessor checked")
	called := false
	s := enrollmentSourceFixture(func(_ time.Time, _ string, _ storage.PropertyEnrollmentPlan, owned storage.Policy, _ storage.Observation, _ []inventory.DiskObservation, _ storage.PoolSystemUse) (storage.PropertyEnrollmentState, error) {
		called = true
		if owned.Members[0].Serial != "SERIAL-A" || owned.Scopes[0].Properties[0] != storage.Quota {
			t.Fatal("predecessor aliases native caller input")
		}
		return storage.PropertyEnrollmentState{}, stop
	})
	original := s.pool.read
	s.pool.read = func(ctx context.Context, probe poolProbe, name string, budget int) (readOutput, error) {
		previous.Members[0].Serial = "changed"
		previous.Scopes[0].Properties[0] = storage.RefQuota
		return original(ctx, probe, name, budget)
	}
	bound, err := s.Review(context.Background(), "root", proposed, previous)
	if !called || !errors.Is(err, stop) || !reflect.DeepEqual(bound, storage.PropertyEnrollmentState{}) {
		t.Fatal("missing owned predecessor", err)
	}
}

func TestPropertyEnrollmentSourceRechecksContextAndClockAfterBinding(t *testing.T) {
	for _, mode := range []string{"canceled", "expired", "clock-backwards", "zero-binding", "missing-bind", "missing-clock", "native-failure", "system-provider-drift"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			s := enrollmentSourceFixture(func(time.Time, string, storage.PropertyEnrollmentPlan, storage.Policy, storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse) (storage.PropertyEnrollmentState, error) {
				if mode == "canceled" {
					cancel()
				}
				return storage.PropertyEnrollmentState{}, nil
			})
			s.now = func() time.Time {
				calls++
				if calls >= 3 {
					if mode == "expired" {
						return enrollmentInstant.Add(5 * time.Minute)
					}
					if mode == "clock-backwards" {
						return enrollmentInstant.Add(-time.Second)
					}
				}
				return enrollmentInstant
			}
			switch mode {
			case "missing-bind":
				s.bind = nil
			case "missing-clock":
				s.now = nil
			case "native-failure":
				s.pool.read = func(context.Context, poolProbe, string, int) (readOutput, error) { return readOutput{}, ErrUnavailable }
			case "system-provider-drift":
				s.system = useSource(func(_ context.Context, p systemProbe, _ string, _ int) (readOutput, error) {
					rows := safetyCaptures()
					if p == systemGEOM {
						rows[p] = strings.ReplaceAll(rows[p], "SERIAL-A", "CHANGED")
					}
					return testOutput(rows[p]), nil
				})
			}
			bound, err := s.Review(ctx, "root", safetyPolicy(), storage.Policy{})
			if err == nil || !reflect.DeepEqual(bound, storage.PropertyEnrollmentState{}) {
				t.Fatal("unchecked final result", err)
			}
			if mode == "canceled" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if (mode == "expired" || mode == "clock-backwards") && !errors.Is(err, storage.ErrExpired) {
				t.Fatal(err)
			}
		})
	}
}

func TestPropertyEnrollmentSourceCannotHydrateValidation(t *testing.T) {
	s := enrollmentSourceFixture(nil)
	s.pool.read = func(context.Context, poolProbe, string, int) (readOutput, error) {
		t.Fatal("zero binding reached native I/O")
		return readOutput{}, nil
	}
	if err := s.Validate(context.Background(), "root", storage.PropertyEnrollmentState{}); err == nil {
		t.Fatal("zero binding validated")
	}
}
