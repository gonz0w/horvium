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

func TestPropertyExecutionHandoffPublicationUncertainty(t *testing.T) {
	for _, state := range []JobState{JobApplying, JobObserving, JobSucceeded, JobReconcileRequired} {
		for _, mode := range []string{"error", "lost", "invalid", "foreign", "earlier"} {
			t.Run(string(state)+"/"+mode, func(t *testing.T) {
				c, b, now, j := handoffSetup(t, false)
				e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID)
				if err != nil {
					t.Fatal(err)
				}
				// Issue a second worker before uncertainty. The synthetic backend
				// does not supply native serialization; this test proves only the
				// original coordinator's propagation to all issued workers.
				*now = now.Add(time.Millisecond)
				r, err := c.Review(context.Background(), "session-a", j.Job().Intent.Change)
				if err != nil {
					t.Fatal(err)
				}
				other, _, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("b", 64))
				if err != nil {
					t.Fatal(err)
				}
				second, err := c.ClaimPropertyExecution(context.Background(), "session-a", other.Job().ID)
				if err != nil {
					t.Fatal(err)
				}
				b.job = clonePropertyJob(j)
				if state == JobReconcileRequired {
					b.quiet = false
				}
				switch mode {
				case "error":
					b.cut = string(state)
				case "lost":
					b.lostAck = state
				default:
					b.ackChange = func(p PropertyJob) PropertyJob {
						if executionState(p) != state {
							return p
						}
						switch mode {
						case "invalid":
							return PropertyJob{}
						case "foreign":
							p.job.ID = strings.Repeat("f", 64)
						case "earlier":
							return clonePropertyJob(j)
						}
						return p
					}
				}
				got, err := e.Run(context.Background(), j.Job().ID)
				want := JobApplying
				if state == JobApplying {
					want = JobQueued
				} else if state == JobSucceeded {
					want = JobObserving
				}
				if !errors.Is(err, ErrStateUncertain) || executionState(got) != want || !c.publicationBlocked {
					t.Fatal("worker publication uncertainty did not block original coordinator", err, executionState(got), c.publicationBlocked)
				}
				calls, observations, publications := len(b.calls), b.observations, b.publications
				if got, err := second.Run(context.Background(), other.Job().ID); !errors.Is(err, ErrStateUncertain) || !reflect.DeepEqual(got, PropertyJob{}) {
					t.Fatal("already-issued worker ignored uncertainty", err)
				}
				if next, err := c.ClaimPropertyExecution(context.Background(), "session-a", other.Job().ID); !errors.Is(err, ErrStateUncertain) || next != nil {
					t.Fatal("new handoff ignored uncertainty", err)
				}
				if got, dup, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("b", 64)); !errors.Is(err, ErrStateUncertain) || dup || !reflect.DeepEqual(got, PropertyJob{}) {
					t.Fatal("duplicate admission hid uncertainty", err)
				}
				// Reviews remain read-only and available. Neither new nor
				// concurrent confirmations may perform admission I/O afterward.
				*now = now.Add(time.Millisecond)
				r, err = c.Review(context.Background(), "session-a", j.Job().Intent.Change)
				if err != nil || b.observations != observations+1 {
					t.Fatal("uncertainty removed read-only review", err)
				}
				var wg sync.WaitGroup
				for range 16 {
					wg.Go(func() {
						got, dup, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("c", 64))
						if !errors.Is(err, ErrStateUncertain) || dup || !reflect.DeepEqual(got, PropertyJob{}) {
							t.Error("new admission escaped uncertainty", err)
						}
					})
				}
				wg.Wait()
				if len(b.calls) != calls || b.publications != publications || b.observations != observations+1 {
					t.Fatal("blocked work reached backend")
				}
			})
		}
	}
}

func TestPropertyExecutionHandoffVerifiedReconciliation(t *testing.T) {
	c, b, _, j := handoffSetup(t, false)
	e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID)
	if err != nil {
		t.Fatal(err)
	}
	b.quiet = false
	got, err := e.Run(context.Background(), j.Job().ID)
	if err == nil || errors.Is(err, ErrStateUncertain) || executionState(got) != JobReconcileRequired || c.publicationBlocked {
		t.Fatal("verified reconciliation confused with publication uncertainty", err)
	}
}
