// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Portable test-only support for tagged disposable fixtures; never a product API.
type policyFixtureWorkProfile uint8

const (
	policyFixtureFactsWork policyFixtureWorkProfile = iota
	policyFixtureLifecycleWork
)

func policyFixtureWorkBudget(profile policyFixtureWorkProfile) (time.Duration, error) {
	switch profile {
	case policyFixtureFactsWork:
		return 8 * time.Minute, nil
	case policyFixtureLifecycleWork:
		return 20 * time.Minute, nil
	default:
		return 0, errors.New("unknown fixture work profile")
	}
}
func policyFixtureWorkContext(parent context.Context, profile policyFixtureWorkProfile) (context.Context, context.CancelFunc, error) {
	budget, err := policyFixtureWorkBudget(profile)
	if err != nil {
		return nil, nil, err
	}
	if parent == nil {
		return nil, nil, errors.New("missing fixture parent")
	}
	if err = parent.Err(); err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithTimeout(parent, budget)
	return ctx, cancel, nil
}
func policyFixtureChildBudgetValid(timeoutFlag string, deadline, now time.Time) bool {
	budget, err := time.ParseDuration(timeoutFlag)
	return err == nil && budget == 390*time.Second && !deadline.IsZero() && deadline.After(now) && deadline.Sub(now) <= budget
}

func TestPolicyFixtureWorkBudget(t *testing.T) {
	for _, tc := range []struct {
		profile policyFixtureWorkProfile
		want    time.Duration
	}{{policyFixtureFactsWork, 8 * time.Minute}, {policyFixtureLifecycleWork, 20 * time.Minute}} {
		t.Run(tc.want.String(), func(t *testing.T) {
			parent, cancel := context.WithCancel(context.Background())
			defer cancel()
			start := time.Now()
			ctx, stop, err := policyFixtureWorkContext(parent, tc.profile)
			end := time.Now()
			if err != nil || ctx == nil || stop == nil {
				t.Fatal("fixed live work context", err)
			}
			defer stop()
			deadline, ok := ctx.Deadline()
			if !ok || deadline.Before(start.Add(tc.want)) || deadline.After(end.Add(tc.want)) || ctx.Err() != nil {
				t.Fatal("actual scoped absolute deadline")
			}
			cancel()
			select {
			case <-ctx.Done():
			default:
				t.Fatal("original parent cancellation must propagate")
			}
			again, ok := ctx.Deadline()
			if !ok || !again.Equal(deadline) || !errors.Is(ctx.Err(), context.Canceled) {
				t.Fatal("cancellation cannot renew the original deadline")
			}
		})
	}
	for _, profile := range []policyFixtureWorkProfile{2, 255} {
		if _, err := policyFixtureWorkBudget(profile); err == nil {
			t.Fatal("unknown fixed work profile")
		}
		if ctx, stop, err := policyFixtureWorkContext(context.Background(), profile); err == nil || ctx != nil || stop != nil {
			t.Fatal("unknown profile grants no context")
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, parent := range []context.Context{nil, cancelled} {
		if ctx, stop, err := policyFixtureWorkContext(parent, policyFixtureFactsWork); err == nil || ctx != nil || stop != nil {
			t.Fatal("invalid caller grants no context")
		}
	}
}

func TestPolicyFixtureChildBudget(t *testing.T) {
	now := time.Now()
	for _, flag := range []string{"390s", "6m30s"} {
		for _, remaining := range []time.Duration{time.Second, 390 * time.Second} {
			if !policyFixtureChildBudgetValid(flag, now.Add(remaining), now) {
				t.Fatal("live bounded child", flag, remaining)
			}
		}
	}
	for _, tc := range []struct {
		flag     string
		deadline time.Time
	}{{"150s", now.Add(time.Second)}, {"0", now.Add(time.Second)}, {"bad", now.Add(time.Second)}, {"390s", time.Time{}}, {"390s", now}, {"390s", now.Add(-time.Second)}, {"390s", now.Add(391 * time.Second)}} {
		if policyFixtureChildBudgetValid(tc.flag, tc.deadline, now) {
			t.Fatal("invalid/stale/oversized child cannot pass", tc.flag)
		}
	}
}
