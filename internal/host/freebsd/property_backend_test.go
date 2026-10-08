// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/storage"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type backendFixture struct {
	b                *propertyBackend
	coordinator      *storage.PropertyCoordinator
	policy           storage.Policy
	baseline, effect storage.Observation
	durable          storage.PropertyJob
	calls            []string
	cuts             string
	hook             func(string)
}

func (f *backendFixture) step(s string) error {
	f.calls = append(f.calls, s)
	if f.hook != nil {
		f.hook(s)
	}
	if f.cuts == s {
		return ErrUnavailable
	}
	return nil
}
func copyBackendJob(t testing.TB, j storage.PropertyJob) storage.PropertyJob {
	t.Helper()
	data, err := storage.EncodePropertyJob(j)
	if err != nil {
		t.Fatal(err)
	}
	j, err = storage.DecodePropertyJob(data)
	if err != nil {
		t.Fatal(err)
	}
	return j
}
func backendSetup(t testing.TB) *backendFixture {
	t.Helper()
	p, o, err := claimSourceSafety().Collect(context.Background(), "100")
	if err != nil {
		t.Fatal(err)
	}
	f := &backendFixture{policy: p, baseline: o, effect: o}
	f.effect.Filesystems = append([]storage.Filesystem(nil), o.Filesystems...)
	f.effect.Filesystems[0].Properties.Quota = storage.Value{Effective: "2097152", Source: storage.Local, SourceGUID: "101"}
	b := &propertyBackend{guid: "100", root: func() bool { return true }, clock: time.Now, clearance: func(context.Context) error { return nil }}
	f.b = b
	b.initial = func(context.Context, string) (storage.Policy, storage.Observation, error) {
		err := f.step("initial")
		return f.policy, f.baseline, err
	}
	b.admit = func(j storage.PropertyJob) (storage.PropertyJob, bool, error) {
		if err := f.step("queued"); err != nil {
			return storage.PropertyJob{}, false, err
		}
		f.durable = copyBackendJob(t, j)
		return copyBackendJob(t, j), false, nil
	}
	b.claim = func(id string) (*propertyBackendClaim, error) {
		if err := f.step("claim"); err != nil {
			return nil, err
		}
		if id != f.durable.Job().ID {
			return nil, storage.ErrDenied
		}
		collect := func(context.Context, string) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
			err := f.step("collect")
			o := f.baseline
			if b.controlAttempted && b.quiet {
				o = f.effect
			}
			return copyBackendJob(t, f.durable), f.policy, o, err
		}
		verify := func(j storage.PropertyJob) error {
			if err := f.step("verify"); err != nil {
				return err
			}
			a, _ := storage.EncodePropertyJob(j)
			d, _ := storage.EncodePropertyJob(f.durable)
			if !bytes.Equal(a, d) {
				return storage.ErrDrift
			}
			return nil
		}
		publish := func(j storage.PropertyJob) (storage.PropertyJob, error) {
			state := j.Job().History[len(j.Job().History)-1].State
			if err := f.step(string(state)); err != nil {
				return storage.PropertyJob{}, err
			}
			if storage.ValidatePropertyJobSuccessor(f.durable, j) != nil {
				return storage.PropertyJob{}, storage.ErrDrift
			}
			f.durable = copyBackendJob(t, j)
			return copyBackendJob(t, j), nil
		}
		return &propertyBackendClaim{collect: collect, verify: verify, applying: publish, successor: publish, control: func(j storage.PropertyJob, live func(context.Context) error) (*propertyControl, error) {
			if err := f.step("control-factory"); err != nil {
				return nil, err
			}
			a := &propertyClaimControlAuthority{expected: copyBackendJob(t, j), live: live, verify: verify, collect: collect}
			return &propertyControl{root: b.root, clock: b.clock, authorize: a.authorize, run: func(context.Context, storage.PropertyIntent) error { return f.step("apply") }}, nil
		}}, nil
	}
	f.coordinator = storage.NewPropertyCoordinator(b)
	return f
}
func backendAdmission(t testing.TB, f *backendFixture) (storage.PropertyJob, *storage.PropertyExecution) {
	t.Helper()
	r, err := f.coordinator.Review(context.Background(), "backend-session", storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "2097152"})
	if err != nil {
		t.Fatal("review", err)
	}
	j, dup, err := f.coordinator.Submit(context.Background(), "backend-session", r.ID, strings.Repeat("a", 64))
	if err != nil || dup {
		t.Fatal("admission", err)
	}
	e, err := f.coordinator.ClaimPropertyExecution(context.Background(), "backend-session", j.Job().ID)
	if err != nil {
		t.Fatal("handoff", err)
	}
	return j, e
}
func backendState(j storage.PropertyJob) storage.JobState {
	h := j.Job().History
	if len(h) == 0 {
		return ""
	}
	return h[len(h)-1].State
}
func backendCalls(f *backendFixture, s string) int {
	n := 0
	for _, v := range f.calls {
		if v == s {
			n++
		}
	}
	return n
}

func TestPropertyBackendCompleteSequence(t *testing.T) {
	f := backendSetup(t)
	j, e := backendAdmission(t, f)
	got, err := e.Run(context.Background(), j.Job().ID)
	if err != nil || backendState(got) != storage.JobSucceeded || backendCalls(f, "apply") != 1 || !reflect.DeepEqual(got, f.durable) {
		t.Fatal("complete native-backend contract", err, f.calls)
	}
	if f.baseline.Eligible || f.effect.Eligible || !f.b.matched || f.b.digest == "" {
		t.Fatal("facts promoted or outcome not independently witnessed")
	}
	if err := f.b.authority.CheckLive(context.Background()); err == nil {
		t.Fatal("completed authority still live")
	}
	before := len(f.calls)
	if _, err := e.Run(context.Background(), j.Job().ID); !errors.Is(err, storage.ErrDenied) || len(f.calls) != before {
		t.Fatal("worker replay", err)
	}
	if quiet, err := f.b.ApplyProperty(context.Background(), j.Intent()); quiet || err == nil || backendCalls(f, "apply") != 1 {
		t.Fatal("backend replay", err)
	}
	if _, _, err := f.b.ObservePropertyExecution(context.Background(), j.Intent()); err == nil {
		t.Fatal("outcome observation replay")
	}
	states := []string{}
	for _, s := range f.calls {
		if s == "queued" || s == "applying" || s == "apply" || s == "observing" || s == "succeeded" {
			states = append(states, s)
		}
	}
	if !reflect.DeepEqual(states, []string{"queued", "applying", "apply", "observing", "succeeded"}) {
		t.Fatal("effect/durable ordering", states)
	}
}

func TestPropertyBackendAdmissionDenials(t *testing.T) {
	for _, mode := range []string{"root", "clearance", "native", "eligible", "wrong-pool", "cancel", "rollback", "busy"} {
		t.Run(mode, func(t *testing.T) {
			f := backendSetup(t)
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			switch mode {
			case "root":
				f.b.root = func() bool { return false }
			case "clearance":
				f.b.clearance = func(context.Context) error { return storage.ErrDenied }
			case "native":
				f.cuts = "initial"
			case "eligible":
				f.baseline.Eligible = true
			case "wrong-pool":
				f.policy.PoolGUID = "999"
			case "cancel":
				f.hook = func(s string) {
					if s == "initial" {
						stop()
					}
				}
			case "rollback":
				reads := 0
				f.b.clock = func() time.Time {
					reads++
					if reads > 1 {
						return time.Now().Add(-time.Hour)
					}
					return time.Now()
				}
			case "busy":
				f.b.mu.Lock()
				defer f.b.mu.Unlock()
			}
			p, o, err := f.b.ObserveProperty(ctx, storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "2097152"})
			if err == nil || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) || backendCalls(f, "queued") != 0 {
				t.Fatal("denial retained authority/facts", err)
			}
		})
	}
}

func TestPropertyBackendCutsRetainHonestOutcomes(t *testing.T) {
	for _, cut := range []string{"initial", "queued", "claim", "applying", "control-factory", "collect", "verify", "apply", "observing", "succeeded"} {
		t.Run(cut, func(t *testing.T) {
			f := backendSetup(t)
			if cut == "initial" || cut == "queued" || cut == "claim" {
				f.cuts = cut
				r, err := f.coordinator.Review(context.Background(), "backend-session", storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "2097152"})
				if cut == "initial" {
					if err == nil {
						t.Fatal("native failure granted review")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if _, _, err := f.coordinator.Submit(context.Background(), "backend-session", r.ID, strings.Repeat("a", 64)); err == nil {
					t.Fatal("admission cut acknowledged")
				}
				if backendCalls(f, "apply") != 0 || !f.b.blocked {
					t.Fatal("admission uncertainty not held")
				}
				return
			}
			j, e := backendAdmission(t, f)
			f.cuts = cut
			got, err := e.Run(context.Background(), j.Job().ID)
			if err == nil || backendState(got) == storage.JobSucceeded {
				t.Fatal("cut fabricated success", cut, err)
			}
			if cut == "control-factory" || cut == "collect" || cut == "verify" {
				if backendCalls(f, "apply") != 0 {
					t.Fatal("prelaunch failure had effect")
				}
			}
			if cut == "apply" && backendState(got) != storage.JobReconcileRequired {
				t.Fatal("unverified control lacks durable reconciliation", err)
			}
			if cut == "observing" && backendState(got) != storage.JobApplying {
				t.Fatal("lost observing ack fabricated successor")
			}
			if cut == "succeeded" && backendState(got) != storage.JobObserving {
				t.Fatal("lost success ack fabricated terminal")
			}
			n := len(f.calls)
			if _, err := e.Run(context.Background(), j.Job().ID); !errors.Is(err, storage.ErrDenied) || len(f.calls) != n {
				t.Fatal("cut replayed", err)
			}
		})
	}
}

func TestPropertyBackendRequiresOriginalBindingAndOutcome(t *testing.T) {
	for _, mode := range []string{"unbound", "nil-authority", "second-binding", "wrong-id", "wrong-intent", "premature-observe", "premature-success", "wrong-digest", "mismatched-effect"} {
		t.Run(mode, func(t *testing.T) {
			f := backendSetup(t)
			if mode == "unbound" || mode == "nil-authority" {
				if mode == "nil-authority" {
					if f.b.BindPropertyExecutionAuthority(nil) == nil {
						t.Fatal("nil authority bound")
					}
				}
				if quiet, err := f.b.ApplyProperty(context.Background(), storage.PropertyIntent{}); quiet || err == nil {
					t.Fatal("unbound control")
				}
				return
			}
			j, e := backendAdmission(t, f)
			switch mode {
			case "second-binding":
				if f.b.BindPropertyExecutionAuthority(f.b.authority) == nil {
					t.Fatal("binding replay")
				}
				return
			case "wrong-id":
				if got, _, _, err := f.b.PreparePropertyExecution(context.Background(), strings.Repeat("b", 64)); err == nil || !reflect.DeepEqual(got, storage.PropertyJob{}) {
					t.Fatal("foreign job")
				}
				return
			case "wrong-intent":
				if quiet, err := f.b.ApplyProperty(context.Background(), storage.PropertyIntent{}); quiet || err == nil {
					t.Fatal("foreign intent")
				}
				return
			case "premature-observe":
				if _, _, err := f.b.ObservePropertyExecution(context.Background(), j.Intent()); err == nil {
					t.Fatal("pre-effect facts accepted")
				}
				return
			case "premature-success":
				app, _ := storage.TransitionPropertyJob(j, storage.JobApplying, uint64(time.Now().UnixMilli()), "launch-intent", "")
				if _, err := f.b.PublishPropertyExecution(app, 1); err != nil {
					t.Fatal(err)
				}
				obs, _ := storage.TransitionPropertyJob(app, storage.JobObserving, uint64(time.Now().UnixMilli()), "native-observation", strings.Repeat("f", 64))
				if _, err := f.b.PublishPropertyExecution(obs, 2); err == nil {
					t.Fatal("metadata forged outcome authority")
				}
				return
			case "wrong-digest":
				f.hook = func(s string) {
					if s == "observing" {
						t.Fatal("foreign digest reached publisher")
					}
				}
				app, _ := storage.TransitionPropertyJob(j, storage.JobApplying, uint64(time.Now().UnixMilli()), "launch-intent", "")
				if _, err := f.b.PublishPropertyExecution(app, 1); err != nil {
					t.Fatal(err)
				}
				if _, _, err := f.b.RefreshPropertyExecution(context.Background(), app); err != nil {
					t.Fatal(err)
				}
				if q, err := f.b.ApplyProperty(context.Background(), j.Intent()); !q || err != nil {
					t.Fatal(err)
				}
				if _, _, err := f.b.ObservePropertyExecution(context.Background(), j.Intent()); err != nil {
					t.Fatal(err)
				}
				obs, _ := storage.TransitionPropertyJob(app, storage.JobObserving, uint64(time.Now().UnixMilli()), "native-observation", strings.Repeat("f", 64))
				if _, err := f.b.PublishPropertyExecution(obs, 2); err == nil {
					t.Fatal("wrong digest published")
				}
				return
			case "mismatched-effect":
				f.effect = f.baseline
			}
			got, err := e.Run(context.Background(), j.Job().ID)
			if mode == "mismatched-effect" && (err == nil || backendState(got) != storage.JobReconcileRequired || f.b.matched) {
				t.Fatal("unmatched effect reported success", err)
			}
		})
	}
}

func TestPropertyBackendConcurrentWorkerAndBusyCallback(t *testing.T) {
	f := backendSetup(t)
	j, e := backendAdmission(t, f)
	f.hook = func(s string) {
		if s == "apply" {
			_, _, err := f.b.ObserveProperty(context.Background(), j.Job().Intent.Change)
			if !errors.Is(err, storage.ErrDenied) {
				t.Fatal("busy backend blocked/reentered", err)
			}
		}
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for range 32 {
		wg.Go(func() {
			got, err := e.Run(context.Background(), j.Job().ID)
			mu.Lock()
			defer mu.Unlock()
			if err == nil && backendState(got) == storage.JobSucceeded {
				success++
			} else if !errors.Is(err, storage.ErrDenied) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if success != 1 || backendCalls(f, "apply") != 1 {
		t.Fatal("concurrent backend replay")
	}
}

func TestPropertyBackendFactoryUnavailable(t *testing.T) {
	if nativeSupported {
		t.Skip("non-FreeBSD refusal")
	}
	if b, err := newPropertyBackend(nil, "100", func(context.Context) error { return nil }); b != nil || !errors.Is(err, ErrUnsupported) {
		t.Fatal("unsupported factory", err)
	}
}

func TestPropertyBackendLostAndForeignAcknowledgements(t *testing.T) {
	for _, state := range []string{"queued", "applying", "observing", "succeeded"} {
		for _, mode := range []string{"lost", "foreign", "invalid"} {
			t.Run(state+"/"+mode, func(t *testing.T) {
				f := backendSetup(t)
				mutate := func(j storage.PropertyJob) (storage.PropertyJob, error) {
					if mode == "lost" {
						return storage.PropertyJob{}, ErrUnavailable
					}
					if mode == "invalid" {
						return storage.PropertyJob{}, nil
					}
					job := j.Job()
					job.ID = strings.Repeat("f", 64)
					other, err := storage.BindPropertyJob(job, j.Intent())
					if err != nil {
						t.Fatal(err)
					}
					return other, nil
				}
				if state == "queued" {
					original := f.b.admit
					f.b.admit = func(j storage.PropertyJob) (storage.PropertyJob, bool, error) {
						ack, dup, err := original(j)
						if err != nil {
							return ack, dup, err
						}
						ack, err = mutate(ack)
						return ack, dup, err
					}
					r, err := f.coordinator.Review(context.Background(), "backend-session", storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "2097152"})
					if err != nil {
						t.Fatal(err)
					}
					if _, _, err := f.coordinator.Submit(context.Background(), "backend-session", r.ID, strings.Repeat("a", 64)); !errors.Is(err, storage.ErrStateUncertain) || !f.b.blocked || backendCalls(f, "apply") != 0 {
						t.Fatal("queued ambiguity admitted execution", err)
					}
					return
				}
				j, e := backendAdmission(t, f)
				wrap := func(original func(storage.PropertyJob) (storage.PropertyJob, error)) func(storage.PropertyJob) (storage.PropertyJob, error) {
					return func(next storage.PropertyJob) (storage.PropertyJob, error) {
						ack, err := original(next)
						if err != nil {
							return ack, err
						}
						if string(backendState(next)) == state {
							return mutate(ack)
						}
						return ack, nil
					}
				}
				f.b.source.applying = wrap(f.b.source.applying)
				f.b.source.successor = wrap(f.b.source.successor)
				got, err := e.Run(context.Background(), j.Job().ID)
				if !errors.Is(err, storage.ErrStateUncertain) || !f.b.blocked || backendState(got) == storage.JobSucceeded {
					t.Fatal("ambiguous successor acknowledged", err)
				}
				before := len(f.calls)
				if _, err := e.Run(context.Background(), j.Job().ID); !errors.Is(err, storage.ErrDenied) || len(f.calls) != before {
					t.Fatal("uncertainty replayed", err)
				}
			})
		}
	}
}

func TestPropertyBackendFinalClearanceChecks(t *testing.T) {
	for _, mode := range []string{"expired", "rollback", "root"} {
		t.Run(mode, func(t *testing.T) {
			f := backendSetup(t)
			j, _ := backendAdmission(t, f)
			f.b.clearance = func(context.Context) error {
				switch mode {
				case "expired":
					f.b.clock = func() time.Time { return j.Intent().Review().ExpiresAt }
				case "rollback":
					f.b.clock = func() time.Time { return time.Now().Add(-time.Hour) }
				case "root":
					f.b.root = func() bool { return false }
				}
				return nil
			}
			job, p, o, err := f.b.PreparePropertyExecution(context.Background(), j.Job().ID)
			if err == nil || !reflect.DeepEqual(job, storage.PropertyJob{}) || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) || backendCalls(f, "apply") != 0 {
				t.Fatal("clearance extended original launch window", err)
			}
		})
	}
}
