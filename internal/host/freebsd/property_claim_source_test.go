// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"crypto/sha256"
	"reflect"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/storage"
)

func claimSourceSafety() *PropertySafetySource {
	rows := safetyCaptures()
	return &PropertySafetySource{policy: policySource(func(string) (storage.Policy, error) { return safetyPolicy(), nil }), providers: safetyProvider(), system: useSource(func(_ context.Context, probe systemProbe, _ string, _ int) (readOutput, error) {
		return testOutput(rows[probe]), nil
	})}
}

func claimSourceJob(t testing.TB) storage.PropertyJob {
	t.Helper()
	p, o, err := claimSourceSafety().Collect(context.Background(), "100")
	if err != nil {
		t.Fatal(err)
	}
	return sourcePropertyJob(t, p, o)
}

func TestPropertyClaimSourceBindsOriginalReadersAndNativeChecks(t *testing.T) {
	job := claimSourceJob(t)
	digest := sha256.Sum256([]byte("original-claim-ledger"))
	reads, checks := 0, 0
	s := &PropertyClaimSource{safety: claimSourceSafety(), inspect: func(id string) (storage.PropertyJob, [32]byte, error) {
		reads++
		if id != job.Job().ID {
			t.Fatal("job selection")
		}
		return job, digest, nil
	}, check: func(p storage.Policy, u storage.PoolSystemUse, current storage.PropertyJob) error {
		checks++
		if p.PoolGUID != "100" || len(p.Members) != 2 || len(u.System().Mounts) == 0 || !reflect.DeepEqual(job, current) {
			t.Fatal("unbound state check")
		}
		return nil
	}}
	got, p, o, err := s.Collect(context.Background(), job.Job().ID)
	if err != nil || reads != 2 || checks != 1 || !reflect.DeepEqual(got, job) || !o.Complete || o.Eligible {
		t.Fatal("collection", err)
	}
	a, err := got.AssessObservation(p, o)
	if err != nil || !a.BaselineMatches {
		t.Fatal("fresh bound baseline", err)
	}
}

func TestPropertyClaimSourceFailuresDiscardEveryProjection(t *testing.T) {
	for _, mode := range []string{"first-read", "final-read", "ledger-drift", "job-drift", "foreign-id", "zero-digest", "missing-check", "missing-inspector", "missing-safety", "state-denied", "state-uncertain", "cancel-first", "cancel-check", "cancel-final", "observing"} {
		t.Run(mode, func(t *testing.T) {
			job := claimSourceJob(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			digest := sha256.Sum256([]byte("original-claim-ledger"))
			reads := 0
			s := &PropertyClaimSource{safety: claimSourceSafety(), inspect: func(string) (storage.PropertyJob, [32]byte, error) {
				reads++
				next := job
				if mode == "first-read" || mode == "final-read" && reads == 2 {
					return storage.PropertyJob{}, [32]byte{}, storage.ErrDenied
				}
				if mode == "cancel-first" || mode == "cancel-final" && reads == 2 {
					cancel()
				}
				if mode == "zero-digest" {
					return job, [32]byte{}, nil
				}
				if mode == "ledger-drift" && reads == 2 {
					digest[0] ^= 1
				}
				if mode == "foreign-id" {
					j := job.Job()
					j.ID = strings.Repeat("b", 64)
					var err error
					next, err = storage.BindPropertyJob(j, job.Intent())
					if err != nil {
						t.Fatal(err)
					}
				}
				if mode == "job-drift" && reads == 2 || mode == "observing" {
					var err error
					next, err = storage.TransitionPropertyJob(job, storage.JobApplying, job.Job().History[0].At+1, "launch-intent", "")
					if err != nil {
						t.Fatal(err)
					}
					if mode == "observing" {
						next, err = storage.TransitionPropertyJob(next, storage.JobObserving, next.Job().History[1].At+1, "native-observation", strings.Repeat("e", 64))
						if err != nil {
							t.Fatal(err)
						}
					}
				}
				return next, digest, nil
			}, check: func(storage.Policy, storage.PoolSystemUse, storage.PropertyJob) error {
				if mode == "state-denied" {
					return storage.ErrDenied
				}
				if mode == "state-uncertain" {
					return storage.ErrStateUncertain
				}
				if mode == "cancel-check" {
					cancel()
				}
				return nil
			}}
			switch mode {
			case "missing-check":
				s.check = nil
			case "missing-inspector":
				s.inspect = nil
			case "missing-safety":
				s.safety = nil
			}
			got, p, o, err := s.Collect(ctx, job.Job().ID)
			if err == nil || !reflect.DeepEqual(got, storage.PropertyJob{}) || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
				t.Fatal("partial projection", err)
			}
		})
	}
}

func TestPropertyClaimSourceSelectorsBeforeIO(t *testing.T) {
	reads := 0
	s := &PropertyClaimSource{safety: claimSourceSafety(), inspect: func(string) (storage.PropertyJob, [32]byte, error) {
		reads++
		return storage.PropertyJob{}, [32]byte{}, storage.ErrDenied
	}, check: func(storage.Policy, storage.PoolSystemUse, storage.PropertyJob) error {
		t.Fatal("unexpected check")
		return nil
	}}
	for _, id := range []string{"", "../../job", strings.Repeat("a", 65), "invalid"} {
		if _, _, _, err := s.Collect(context.Background(), id); err == nil {
			t.Fatal("selector")
		}
	}
	if _, _, _, err := s.Collect(nil, strings.Repeat("a", 64)); err == nil {
		t.Fatal("nil context")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, _, err := s.Collect(ctx, strings.Repeat("a", 64)); err == nil {
		t.Fatal("cancel")
	}
	if reads != 0 {
		t.Fatal("I/O before validation")
	}
}

func TestPropertyClaimSourceFinalPolicyAndNativeDenials(t *testing.T) {
	for _, mode := range []string{"policy-drift", "provider-error", "system-error", "system-root-overlap"} {
		t.Run(mode, func(t *testing.T) {
			job := claimSourceJob(t)
			safety := claimSourceSafety()
			changed := false
			safety.policy.ownership = func(string) (storage.Policy, error) {
				p := safetyPolicy()
				if changed {
					p.Revision++
				}
				return p, nil
			}
			if mode == "provider-error" {
				safety.providers = providersForTest(func(context.Context, systemProbe, string, int) (readOutput, error) {
					return readOutput{}, ErrUnavailable
				})
			}
			if mode == "system-error" {
				safety.system = useSource(func(context.Context, systemProbe, string, int) (readOutput, error) {
					return readOutput{}, ErrUnavailable
				})
			}
			if mode == "system-root-overlap" {
				rows := safetyCaptures()
				rows[systemMounts] = "/dev/ada0 / ufs rw 1 1\n"
				safety.system = useSource(func(_ context.Context, p systemProbe, _ string, _ int) (readOutput, error) {
					return testOutput(rows[p]), nil
				})
			}
			s := &PropertyClaimSource{safety: safety, inspect: func(string) (storage.PropertyJob, [32]byte, error) { return job, sha256.Sum256([]byte("ledger")), nil }, check: func(storage.Policy, storage.PoolSystemUse, storage.PropertyJob) error {
				changed = mode == "policy-drift"
				return nil
			}}
			got, p, o, err := s.Collect(context.Background(), job.Job().ID)
			if err == nil || !reflect.DeepEqual(got, storage.PropertyJob{}) || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
				t.Fatal("native/final policy denial leaked state", err)
			}
		})
	}
}

func FuzzPropertyClaimSourceSelection(f *testing.F) {
	f.Add(strings.Repeat("a", 64))
	f.Add("../../job")
	f.Fuzz(func(t *testing.T, id string) {
		if len(id) > 256 {
			return
		}
		reads := 0
		s := &PropertyClaimSource{safety: claimSourceSafety(), inspect: func(string) (storage.PropertyJob, [32]byte, error) {
			reads++
			return storage.PropertyJob{}, [32]byte{}, storage.ErrDenied
		}, check: func(storage.Policy, storage.PoolSystemUse, storage.PropertyJob) error {
			t.Fatal("unexpected check")
			return nil
		}}
		_, _, _, err := s.Collect(context.Background(), id)
		if err == nil {
			t.Fatal("empty trusted input accepted")
		}
		if storage.ValidatePropertyJobSelection(id) != nil && reads != 0 {
			t.Fatal("invalid selector reached I/O")
		}
	})
}
