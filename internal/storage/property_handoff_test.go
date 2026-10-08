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

// This combined synthetic backend owns both admission and execution. It is not
// a root lease, native eligibility source or installed runtime adapter.
type handoffFixture struct {
	*coordinatorFixture
	*executionFixture
}

func handoffSetup(t testing.TB, duplicate bool) (*PropertyCoordinator, *handoffFixture, *time.Time, PropertyJob) {
	t.Helper()
	c, admission, now, intent := coordinatorSetup(t)
	b := &handoffFixture{admission, &executionFixture{policy: admission.policy, baseline: admission.observation, quiet: true}}
	c.backend = b
	admission.publish = func(j PropertyJob) (PropertyJob, bool, error) {
		b.job = clonePropertyJob(j)
		b.effect = b.baseline
		b.effect.Filesystems = append([]Filesystem(nil), b.baseline.Filesystems...)
		b.effect.Filesystems[0].Properties.Quota = Value{j.Intent().Review().Intent.Value, Local, "101", ""}
		b.effect.Eligible = false
		return clonePropertyJob(j), duplicate, nil
	}
	r, err := c.Review(context.Background(), "session-a", intent)
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(time.Second)
	j, dup, err := c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("a", 64))
	if err != nil || dup != duplicate {
		t.Fatal("fixture admission", err)
	}
	return c, b, now, j
}

func TestPropertyExecutionHandoffOriginalAdmission(t *testing.T) {
	c, b, _, j := handoffSetup(t, false)
	id, plan := j.Job().ID, j.Intent().Review().ID
	j.job.History[0].Reason = "caller-edited"
	j.intent.review.Context[0].Name = "caller-edited"
	ack, dup, err := c.Submit(context.Background(), "session-a", plan, strings.Repeat("a", 64))
	if err != nil || !dup || ack.Job().History[0].Reason != "admitted" {
		t.Fatal("confirm lost original acknowledgement", err)
	}
	e, err := c.ClaimPropertyExecution(context.Background(), "session-a", id)
	if err != nil || e == nil || b.observations != 2 || b.publications != 1 || len(b.calls) != 0 {
		t.Fatal("handoff performed I/O or lost original claim", err)
	}
	got, err := e.Run(context.Background(), id)
	if err != nil || executionState(got) != JobSucceeded || !reflect.DeepEqual(b.calls, []string{"prepare", "applying", "refresh", "apply", "observe", "observing", "succeeded"}) {
		t.Fatal("handoff lost durable execution order", err, b.calls)
	}
	if again, err := c.ClaimPropertyExecution(context.Background(), "session-a", id); again != nil || !errors.Is(err, ErrDenied) {
		t.Fatal("second handoff", err)
	}
	if _, err := e.Run(context.Background(), id); !errors.Is(err, ErrDenied) || executionCount(b.executionFixture, "apply") != 1 {
		t.Fatal("handoff execution replay", err)
	}
	retained, dup, err := c.Submit(context.Background(), "session-a", plan, strings.Repeat("a", 64))
	if err != nil || !dup || executionState(retained) != JobQueued || b.publications != 1 {
		t.Fatal("execution rewrote retained admission response", err)
	}
}

func TestPropertyExecutionHandoffConcurrentSingleUse(t *testing.T) {
	c, b, _, j := handoffSetup(t, false)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var granted []*PropertyExecution
	for range 32 {
		wg.Go(func() {
			e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID)
			mu.Lock()
			defer mu.Unlock()
			if err == nil && e != nil {
				granted = append(granted, e)
			} else if !errors.Is(err, ErrDenied) || e != nil {
				t.Error("invalid concurrent handoff", err)
			}
		})
	}
	wg.Wait()
	if len(granted) != 1 {
		t.Fatal("multiple live claims", len(granted))
	}
	success := 0
	for range 32 {
		wg.Go(func() {
			got, err := granted[0].Run(context.Background(), j.Job().ID)
			mu.Lock()
			defer mu.Unlock()
			if err == nil && executionState(got) == JobSucceeded {
				success++
			} else if !errors.Is(err, ErrDenied) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if success != 1 || executionCount(b.executionFixture, "apply") != 1 {
		t.Fatal("aliases replayed control")
	}
}

func TestPropertyExecutionHandoffDenials(t *testing.T) {
	for _, mode := range []string{"wrong-session", "unknown-job", "unsubmitted", "restarted", "duplicate-admission", "missing-adapter", "publication-blocked", "expired", "rollback", "cancelled", "invalid-id", "nil-context", "nil-coordinator"} {
		t.Run(mode, func(t *testing.T) {
			c, b, now, j := handoffSetup(t, mode == "duplicate-admission")
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			session, id := "session-a", j.Job().ID
			switch mode {
			case "wrong-session":
				session = "session-b"
			case "unknown-job":
				id = strings.Repeat("b", 64)
			case "unsubmitted":
				e := c.reviews[j.Intent().Review().ID]
				e.ack = PropertyJob{}
				c.reviews[j.Intent().Review().ID] = e
			case "restarted":
				c = NewPropertyCoordinator(b)
				c.clock = func() time.Time { return *now }
			case "missing-adapter":
				c.backend = b.coordinatorFixture
			case "publication-blocked":
				c.publicationBlocked = true
			case "expired":
				*now = j.Intent().Review().ExpiresAt
			case "rollback":
				*now = now.Add(-time.Second)
			case "cancelled":
				stop()
			case "invalid-id":
				id = "path/to/job"
			case "nil-context":
				ctx = nil
			case "nil-coordinator":
				c = nil
			}
			if e, err := c.ClaimPropertyExecution(ctx, session, id); err == nil || e != nil || len(b.calls) != 0 {
				t.Fatal("denial granted execution or performed I/O", err)
			}
		})
	}
}

func TestPropertyExecutionHandoffFreshRecordBinding(t *testing.T) {
	for _, mode := range []string{"id", "history", "captured-time", "intent", "missing", "prepare-error", "wrong-run-id"} {
		t.Run(mode, func(t *testing.T) {
			c, b, _, j := handoffSetup(t, false)
			e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID)
			if err != nil {
				t.Fatal(err)
			}
			id := j.Job().ID
			switch mode {
			case "id":
				b.job.job.ID = strings.Repeat("b", 64)
			case "history":
				b.job, err = TransitionPropertyJob(b.job, JobApplying, b.job.job.History[0].At, "launch-intent", "")
			case "captured-time":
				b.job.intent.capturedAt = b.job.intent.capturedAt.Add(time.Millisecond)
				b.job.job.History[0].At++
			case "intent":
				b.job.intent.review.Intent.Value = "1"
				b.job.job.Intent.Change.Value = "1"
			case "missing":
				b.job = PropertyJob{}
			case "prepare-error":
				b.cut = "prepare"
			case "wrong-run-id":
				id = strings.Repeat("b", 64)
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := e.Run(context.Background(), id)
			if err == nil || !reflect.DeepEqual(got, PropertyJob{}) || executionCount(b.executionFixture, "applying") != 0 || executionCount(b.executionFixture, "apply") != 0 {
				t.Fatal("foreign fresh record launched", err)
			}
			if mode == "wrong-run-id" && len(b.calls) != 0 {
				t.Fatal("foreign identifier reached backend")
			}
			if _, err = e.Run(context.Background(), j.Job().ID); !errors.Is(err, ErrDenied) {
				t.Fatal("failed handoff retried", err)
			}
		})
	}
}

func TestPropertyExecutionHandoffFinalClock(t *testing.T) {
	for _, mode := range []string{"expired", "rollback", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			c, b, now, j := handoffSetup(t, false)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reads := 0
			c.clock = func() time.Time {
				reads++
				if reads > 1 {
					switch mode {
					case "expired":
						return j.Intent().Review().ExpiresAt
					case "rollback":
						return now.Add(-time.Second)
					case "cancelled":
						cancel()
					}
				}
				return *now
			}
			if e, err := c.ClaimPropertyExecution(ctx, "session-a", j.Job().ID); err == nil || e != nil || len(b.calls) != 0 {
				t.Fatal("handoff skipped final clock/context check", err)
			}
		})
	}
}

func TestPropertyExecutionHandoffAdmissionUncertainty(t *testing.T) {
	c, b, _, j := handoffSetup(t, false)
	e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID)
	if err != nil {
		t.Fatal(err)
	}
	b.hook = func(at string) {
		if at != "applying" {
			return
		}
		r, err := c.Review(context.Background(), "session-a", j.Job().Intent.Change)
		if err != nil {
			t.Fatal(err)
		}
		b.publish = func(PropertyJob) (PropertyJob, bool, error) {
			return PropertyJob{}, false, ErrStateUncertain
		}
		if _, _, err = c.Submit(context.Background(), "session-a", r.ID, strings.Repeat("b", 64)); !errors.Is(err, ErrStateUncertain) {
			t.Fatal("fixture uncertain admission", err)
		}
	}
	if _, err = e.Run(context.Background(), j.Job().ID); !errors.Is(err, ErrStateUncertain) || executionCount(b.executionFixture, "apply") != 0 {
		t.Fatal("original coordinator uncertainty ignored", err)
	}
}

func TestPropertyExecutionHandoffLiveGates(t *testing.T) {
	for _, stage := range []string{"before-run", "prepare", "applying", "refresh"} {
		for _, mode := range []string{"expired", "rollback", "cancelled", "publication-blocked"} {
			t.Run(stage+"/"+mode, func(t *testing.T) {
				c, b, now, j := handoffSetup(t, false)
				e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				change := func() {
					switch mode {
					case "expired":
						*now = j.Intent().Review().ExpiresAt
					case "rollback":
						*now = now.Add(-time.Second)
					case "cancelled":
						cancel()
					case "publication-blocked":
						c.publicationBlocked = true
					}
				}
				if stage == "before-run" {
					change()
				} else {
					b.hook = func(at string) {
						if at == stage {
							change()
						}
					}
				}
				if _, err = e.Run(ctx, j.Job().ID); err == nil || executionCount(b.executionFixture, "apply") != 0 {
					t.Fatal("invalid live claim launched", err)
				}
				if mode == "rollback" && !c.clockBlocked {
					t.Fatal("worker rollback did not block original coordinator")
				}
			})
		}
	}
}

func TestPropertyExecutionHandoffLateOutcomeAndCuts(t *testing.T) {
	for _, mode := range []string{"late-success", "applying", "observing", "succeeded", "unquiet"} {
		t.Run(mode, func(t *testing.T) {
			c, b, now, j := handoffSetup(t, false)
			e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "late-success" {
				b.hook = func(at string) {
					if at == "apply" {
						*now = now.Add(5 * time.Minute)
						// Expiry reclamation removes the map entry, not a verified
						// effect or its already-issued worker's outcome obligation.
						if _, err := c.Review(context.Background(), "session-b", j.Job().Intent.Change); err != nil {
							t.Fatal(err)
						}
					}
				}
			} else if mode == "unquiet" {
				b.quiet = false
			} else {
				b.lostAck = JobState(mode)
			}
			got, err := e.Run(context.Background(), j.Job().ID)
			if mode == "late-success" {
				if err != nil || executionState(got) != JobSucceeded || executionCount(b.executionFixture, "apply") != 1 {
					t.Fatal("late durable outcome discarded", err)
				}
			} else if err == nil || executionState(got) == JobSucceeded {
				t.Fatal("uncertainty reported success", err)
			}
			calls := len(b.calls)
			if _, err = e.Run(context.Background(), j.Job().ID); !errors.Is(err, ErrDenied) || len(b.calls) != calls {
				t.Fatal("outcome uncertainty replayed", err)
			}
		})
	}
}

func FuzzPropertyExecutionHandoff(f *testing.F) {
	f.Add([]byte{0, 1, 1, 2, 0})
	f.Add([]byte{3, 0, 4, 1})
	f.Add([]byte{0, 5, 1, 0})
	f.Fuzz(func(t *testing.T, actions []byte) {
		if len(actions) > 32 {
			t.Skip()
		}
		c, b, now, j := handoffSetup(t, false)
		var execution *PropertyExecution
		grants := 0
		for _, action := range actions {
			switch action % 6 {
			case 0:
				e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID)
				if err == nil {
					grants++
					execution = e
				} else if e != nil {
					t.Fatal("partial denied grant")
				}
			case 1:
				if execution != nil {
					_, _ = execution.Run(context.Background(), j.Job().ID)
				}
			case 2:
				_, _, _ = c.Submit(context.Background(), "session-a", j.Intent().Review().ID, strings.Repeat("a", 64))
			case 3:
				*now = now.Add(5 * time.Minute)
			case 4:
				b.executionFixture.policy.Revision++
			case 5:
				c.publicationBlocked = true
			}
			if grants > 1 || executionCount(b.executionFixture, "apply") > 1 || b.publications != 1 {
				t.Fatal("handoff replayed admission or control")
			}
		}
	})
}
