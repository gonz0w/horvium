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

// Synthetic authority belongs only to this fixture. No native backend implements
// the live-plan/root/executor contract yet.
type executionFixture struct {
	job              PropertyJob
	policy           Policy
	baseline, effect Observation
	calls            []string
	cut              string
	quiet            bool
	hook             func(string)
	ackChange        func(PropertyJob) PropertyJob
	lostAck          JobState
}

func (b *executionFixture) step(name string) error {
	b.calls = append(b.calls, name)
	if b.hook != nil {
		b.hook(name)
	}
	if b.cut == name {
		return ErrUnavailable
	}
	return nil
}
func (b *executionFixture) PreparePropertyExecution(context.Context, string) (PropertyJob, Policy, Observation, error) {
	e := b.step("prepare")
	return clonePropertyJob(b.job), b.policy, b.baseline, e
}
func (b *executionFixture) RefreshPropertyExecution(context.Context, PropertyJob) (Policy, Observation, error) {
	e := b.step("refresh")
	return b.policy, b.baseline, e
}
func (b *executionFixture) PublishPropertyExecution(p PropertyJob, expected uint64) (PropertyJob, error) {
	state := p.Job().History[len(p.Job().History)-1].State
	e := b.step(string(state))
	if expected != uint64(len(b.job.job.History)) || ValidatePropertyJobSuccessor(b.job, p) != nil {
		return PropertyJob{}, ErrDrift
	}
	if e != nil {
		return PropertyJob{}, e
	}
	b.job = clonePropertyJob(p)
	if b.lostAck == state {
		return PropertyJob{}, ErrUnavailable
	}
	if b.ackChange != nil {
		return b.ackChange(clonePropertyJob(p)), nil
	}
	return clonePropertyJob(p), nil
}
func (b *executionFixture) ApplyProperty(context.Context, PropertyIntent) (bool, error) {
	return b.quiet, b.step("apply")
}
func (b *executionFixture) ObservePropertyExecution(context.Context, PropertyIntent) (Policy, Observation, error) {
	e := b.step("observe")
	return b.policy, b.effect, e
}
func executionSetup(t testing.TB) (*PropertyExecution, *executionFixture, *time.Time) {
	t.Helper()
	i, _, p, o := propertyIntentFixture(t)
	j := propertyJobFixture(t)
	// The fixture's deterministic review ID makes these independently captured
	// records identical; full binding, not just selection, must match.
	if !reflect.DeepEqual(j.Intent(), i) {
		t.Fatal("fixture binding")
	}
	effect := o
	effect.Filesystems = append([]Filesystem(nil), o.Filesystems...)
	effect.Filesystems[0].Properties.Quota = Value{i.review.Intent.Value, Local, "101", ""}
	effect.Eligible = false
	b := &executionFixture{job: j, policy: p, baseline: o, effect: effect, quiet: true}
	e := NewPropertyExecution(b)
	now := instant.Add(2 * time.Second)
	e.clock = func() time.Time { return now }
	return e, b, &now
}
func executionState(p PropertyJob) JobState {
	j := p.Job()
	if len(j.History) == 0 {
		return ""
	}
	return j.History[len(j.History)-1].State
}
func executionCount(b *executionFixture, name string) int {
	n := 0
	for _, c := range b.calls {
		if c == name {
			n++
		}
	}
	return n
}

func TestPropertyExecutionDurableOrdering(t *testing.T) {
	e, b, _ := executionSetup(t)
	original, _ := EncodePropertyJob(b.job)
	b.hook = func(name string) {
		if name == "apply" && executionState(b.job) != JobApplying {
			t.Fatal("effect preceded durable applying intent")
		}
		if name == "observe" && executionCount(b, "apply") != 1 {
			t.Fatal("observation preceded control")
		}
	}
	got, err := e.Run(context.Background(), b.job.Job().ID)
	if err != nil || executionState(got) != JobSucceeded {
		t.Fatal("verified result missing", err)
	}
	want := []string{"prepare", "applying", "refresh", "apply", "observe", "observing", "succeeded"}
	if !reflect.DeepEqual(b.calls, want) {
		t.Fatal("wrong durability order", b.calls)
	}
	if got.Job().History[2].ObservationDigest == "" || got.Job().History[2].ObservationDigest != got.Job().History[3].ObservationDigest {
		t.Fatal("lost independent facts")
	}
	if _, err = e.Run(context.Background(), b.job.Job().ID); !errors.Is(err, ErrDenied) || executionCount(b, "apply") != 1 {
		t.Fatal("attempt replayed", err)
	}
	got.job.History[0].Reason = "caller-edited"
	got.intent.review.Context[0].Name = "caller-edited"
	if b.job.Job().History[0].Reason != "admitted" || b.job.Intent().Review().Context[0].Name != "tank" {
		t.Fatal("result aliases durable record")
	}
	if _, err = DecodePropertyJob(original); err != nil {
		t.Fatal("input corrupted", err)
	}
}

func TestPropertyExecutionPublicationCuts(t *testing.T) {
	for _, cut := range []string{"applying", "observing", "succeeded", "reconcile-required"} {
		t.Run(cut, func(t *testing.T) {
			e, b, _ := executionSetup(t)
			b.cut = cut
			if cut == "reconcile-required" {
				b.quiet = false
			}
			got, err := e.Run(context.Background(), b.job.Job().ID)
			if !errors.Is(err, ErrStateUncertain) || executionState(got) == JobSucceeded {
				t.Fatal("uncertain publication acknowledged", err)
			}
			want := JobQueued
			if cut == "observing" || cut == "reconcile-required" {
				want = JobApplying
			}
			if cut == "succeeded" {
				want = JobObserving
			}
			if executionState(got) != want || executionState(b.job) != want {
				t.Fatal("invented durable successor")
			}
			count := executionCount(b, "apply")
			if cut == "applying" && count != 0 || cut != "applying" && count != 1 {
				t.Fatal("effect at wrong cut")
			}
			if _, err = e.Run(context.Background(), b.job.Job().ID); !errors.Is(err, ErrDenied) || executionCount(b, "apply") != count {
				t.Fatal("cut replayed", err)
			}
		})
	}
}

func TestPropertyExecutionLostDurableAcknowledgement(t *testing.T) {
	for _, state := range []JobState{JobApplying, JobObserving, JobSucceeded} {
		t.Run(string(state), func(t *testing.T) {
			e, b, _ := executionSetup(t)
			b.lostAck = state
			id := b.job.Job().ID
			got, err := e.Run(context.Background(), id)
			if !errors.Is(err, ErrStateUncertain) || executionState(b.job) != state || executionState(got) == state {
				t.Fatal("lost acknowledgement invented knowledge", err)
			}
			if state == JobApplying && executionCount(b, "apply") != 0 {
				t.Fatal("launched without acknowledged intent")
			}
			count := len(b.calls)
			if _, err = e.Run(context.Background(), id); !errors.Is(err, ErrDenied) || len(b.calls) != count {
				t.Fatal("lost acknowledgement retried", err)
			}
		})
	}
}

func TestPropertyExecutionNativeAmbiguity(t *testing.T) {
	for _, cut := range []string{"apply", "unquiet", "observe", "mismatch", "policy-drift", "malformed"} {
		t.Run(cut, func(t *testing.T) {
			e, b, _ := executionSetup(t)
			switch cut {
			case "apply", "observe":
				b.cut = cut
			case "unquiet":
				b.quiet = false
			case "mismatch":
				b.effect.Filesystems[0].Properties.Quota.Effective = "2"
			case "policy-drift":
				b.hook = func(s string) {
					if s == "observe" {
						b.policy.Revision++
					}
				}
			case "malformed":
				b.effect.Complete = false
			}
			got, err := e.Run(context.Background(), b.job.Job().ID)
			if err == nil || executionState(got) != JobReconcileRequired || b.job.Job().Terminal() || executionCount(b, "apply") != 1 {
				t.Fatal("ambiguity freed hold/succeeded", err)
			}
			if cut == "apply" || cut == "unquiet" {
				if executionCount(b, "observe") != 0 || executionCount(b, "observing") != 0 {
					t.Fatal("unverified control observed as complete")
				}
			}
			if cut == "mismatch" || cut == "policy-drift" {
				if executionCount(b, "observing") != 1 || got.Job().History[len(got.Job().History)-1].ObservationDigest == "" {
					t.Fatal("valid differing facts lost")
				}
			}
		})
	}
}

func TestPropertyExecutionLaunchDenials(t *testing.T) {
	for _, stage := range []string{"prepare", "refresh"} {
		for _, kind := range []string{"backend", "policy", "property", "ineligible", "incomplete", "expired", "cancelled", "rollback", "future-history"} {
			t.Run(stage+"/"+kind, func(t *testing.T) {
				e, b, now := executionSetup(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				b.hook = func(s string) {
					if s != stage {
						return
					}
					switch kind {
					case "backend":
						b.cut = stage
					case "policy":
						b.policy.Revision++
					case "property":
						b.baseline.Filesystems[0].Properties.Quota.Effective = "1"
					case "ineligible":
						b.baseline.Eligible = false
					case "incomplete":
						b.baseline.Complete = false
					case "expired":
						*now = now.Add(5 * time.Minute)
					case "cancelled":
						cancel()
					case "rollback":
						*now = now.Add(-time.Second)
					case "future-history":
						*now = time.UnixMilli(int64(b.job.Job().History[0].At) - 1)
					}
				}
				got, err := e.Run(ctx, b.job.Job().ID)
				if err == nil || executionCount(b, "apply") != 0 || executionState(got) == JobSucceeded {
					t.Fatal("unsafe launch", err)
				}
				if stage == "prepare" && len(b.calls) != 1 {
					t.Fatal("denied preflight published", b.calls)
				}
				if stage == "refresh" && executionState(b.job) != JobReconcileRequired && executionState(b.job) != JobApplying {
					t.Fatal("lost applying hold")
				}
			})
		}
	}
}

func TestPropertyExecutionAcknowledgementBinding(t *testing.T) {
	for _, cut := range []string{"applying", "observing", "succeeded"} {
		for _, kind := range []string{"invalid", "foreign", "earlier", "rewritten"} {
			t.Run(cut+"/"+kind, func(t *testing.T) {
				e, b, _ := executionSetup(t)
				original := clonePropertyJob(b.job)
				b.ackChange = func(p PropertyJob) PropertyJob {
					if string(executionState(p)) != cut {
						return p
					}
					switch kind {
					case "invalid":
						return PropertyJob{}
					case "foreign":
						p.job.ID = strings.Repeat("f", 64)
					case "earlier":
						return original
					case "rewritten":
						p.intent.review.Context[0].Name = "foreign"
					}
					return p
				}
				got, err := e.Run(context.Background(), b.job.Job().ID)
				if !errors.Is(err, ErrStateUncertain) || executionState(got) == JobSucceeded {
					t.Fatal("foreign durability acknowledgement", err)
				}
				if cut == "applying" && executionCount(b, "apply") != 0 {
					t.Fatal("effect after untrusted applying acknowledgement")
				}
			})
		}
	}
}

func TestPropertyExecutionSingleAttemptConcurrent(t *testing.T) {
	e, b, _ := executionSetup(t)
	id := b.job.Job().ID
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for range 32 {
		wg.Go(func() {
			p, err := e.Run(context.Background(), id)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				if executionState(p) != JobSucceeded {
					t.Error("not durable success")
				}
				success++
			} else if !errors.Is(err, ErrDenied) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if success != 1 || executionCount(b, "apply") != 1 {
		t.Fatal("duplicate launch", success, b.calls)
	}
}

func TestPropertyExecutionRefusesRestoredRecords(t *testing.T) {
	for _, state := range []JobState{JobApplying, JobObserving, JobSucceeded, JobReconcileRequired} {
		t.Run(string(state), func(t *testing.T) {
			e, b, now := executionSetup(t)
			p, err := TransitionPropertyJob(b.job, JobApplying, uint64(now.UnixMilli()), "launch-intent", "")
			if err != nil {
				t.Fatal(err)
			}
			if state == JobObserving || state == JobSucceeded {
				p, err = TransitionPropertyJob(p, JobObserving, uint64(now.UnixMilli()), "native-observation", strings.Repeat("d", 64))
				if err != nil {
					t.Fatal(err)
				}
			}
			if state == JobSucceeded {
				p, err = TransitionPropertyJob(p, state, uint64(now.UnixMilli()), "verified", strings.Repeat("d", 64))
				if err != nil {
					t.Fatal(err)
				}
			}
			if state == JobReconcileRequired {
				p, err = TransitionPropertyJob(p, state, uint64(now.UnixMilli()), "executor-unverified", "")
				if err != nil {
					t.Fatal(err)
				}
			}
			b.job = p
			if _, err = e.Run(context.Background(), p.Job().ID); err == nil || executionCount(b, "apply") != 0 || len(b.calls) != 1 {
				t.Fatal("retained state restored launch authority", err)
			}
		})
	}
}

func TestPropertyExecutionWorkerContextAndLateCompletion(t *testing.T) {
	e, b, now := executionSetup(t)
	b.hook = func(s string) {
		if s == "apply" {
			*now = now.Add(10 * time.Minute)
		}
	}
	got, err := e.Run(context.Background(), b.job.Job().ID)
	if err != nil || executionState(got) != JobSucceeded {
		t.Fatal("expired after launch hid actual durable completion", err)
	}
	e, b, _ = executionSetup(t)
	ctx, cancel := context.WithCancel(context.Background())
	b.hook = func(s string) {
		if s == "apply" {
			cancel()
		}
	}
	got, err = e.Run(ctx, b.job.Job().ID)
	if err == nil || executionState(got) != JobReconcileRequired {
		t.Fatal("worker cancellation discarded applying intent", err)
	}
	e, b, _ = executionSetup(t)
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	b.hook = func(s string) {
		if s == "succeeded" {
			cancel()
		}
	}
	got, err = e.Run(ctx, b.job.Job().ID)
	if err != nil || executionState(got) != JobSucceeded || ctx.Err() == nil {
		t.Fatal("late cancellation hid durable outcome", err)
	}
}

func TestPropertyExecutionInvalidInputs(t *testing.T) {
	e, b, _ := executionSetup(t)
	for _, id := range []string{"", "../job", strings.Repeat("a", 65)} {
		if _, err := e.Run(context.Background(), id); err == nil || len(b.calls) != 0 {
			t.Fatal("invalid ID reached backend")
		}
	}
	if _, err := e.Run(nil, b.job.Job().ID); !errors.Is(err, ErrInvalid) {
		t.Fatal("nil context", err)
	}
	var absent *PropertyExecution
	if _, err := absent.Run(context.Background(), b.job.Job().ID); !errors.Is(err, ErrUnavailable) {
		t.Fatal("nil execution", err)
	}
	if _, err := NewPropertyExecution(nil).Run(context.Background(), b.job.Job().ID); !errors.Is(err, ErrUnavailable) {
		t.Fatal("nil backend", err)
	}
	b.hook = func(s string) {
		if s == "prepare" {
			b.job.job.ID = strings.Repeat("f", 64)
		}
	}
	if _, err := e.Run(context.Background(), b.job.Job().ID); err == nil || executionCount(b, "apply") != 0 {
		t.Fatal("foreign queued record")
	}
}

func FuzzPropertyExecutionCut(f *testing.F) {
	for _, s := range []string{"", "prepare", "applying", "refresh", "apply", "observe", "observing", "succeeded", "reconcile-required"} {
		f.Add(s, true)
	}
	f.Fuzz(func(t *testing.T, cut string, quiet bool) {
		if len(cut) > 64 {
			return
		}
		e, b, _ := executionSetup(t)
		b.cut = cut
		b.quiet = quiet
		id := b.job.Job().ID
		got, err := e.Run(context.Background(), id)
		if executionCount(b, "apply") > 1 || err != nil && executionState(got) == JobSucceeded {
			t.Fatal("false success/replay")
		}
		if err == nil && (executionState(got) != JobSucceeded || executionCount(b, "apply") != 1 || !quiet || executionCount(b, "succeeded") != 1) {
			t.Fatal("success without full durable sequence")
		}
		count := len(b.calls)
		if _, err = e.Run(context.Background(), id); !errors.Is(err, ErrDenied) || len(b.calls) != count {
			t.Fatal("attempt replayed")
		}
	})
}
