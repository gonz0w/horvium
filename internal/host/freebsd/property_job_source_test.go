// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

func sourcePropertyJob(t testing.TB, p storage.Policy, o storage.Observation) storage.PropertyJob {
	t.Helper()
	o.Eligible = true // Test-only capture, never returned by a native source.
	at := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	plan, err := storage.Build(at, "source-test", p, o, storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "2097152"})
	if err != nil {
		t.Fatal(err)
	}
	i, err := plan.CapturePropertyIntent(at, "source-test", p, o)
	if err != nil {
		t.Fatal(err)
	}
	r := i.Review()
	j := storage.Job{ID: strings.Repeat("a", 64), Intent: storage.JobIntent{PlanID: r.ID, SubmissionID: strings.Repeat("c", 64), PolicyRevision: r.PolicyRevision, PoolGUID: r.PoolGUID, Change: r.Intent}, History: []storage.JobTransition{{Revision: 1, At: uint64(at.Add(time.Millisecond).UnixMilli()), State: storage.JobQueued, Reason: "admitted"}}}
	job, err := storage.BindPropertyJob(j, i)
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func TestPropertyJobSourceBindsLedgerAndIndependentCollection(t *testing.T) {
	policy := policySource(func(string) (storage.Policy, error) { return collectionPolicy(), nil })
	p, o, err := policy.Collect(context.Background(), "100")
	if err != nil {
		t.Fatal(err)
	}
	job := sourcePropertyJob(t, p, o)
	digest := sha256.Sum256([]byte("complete-test-ledger"))
	reads := 0
	source := &PropertyJobSource{policy: policy, inspect: func(id string) (storage.PropertyJob, [32]byte, error) {
		reads++
		if id != job.Job().ID {
			t.Fatal("unbound job selection")
		}
		return job, digest, nil
	}}
	got, err := source.Assess(context.Background(), job.Job().ID)
	if err != nil || reads != 2 || !got.BaselineMatches || got.PostconditionMatches || got.Restart.Last.State != storage.JobQueued || got.Restart.Reason != "review-expired" {
		t.Fatal("missing native/ledger binding", got, err)
	}
	encoded, _ := storage.EncodePropertyJob(job)
	got.Restart.Last.Reason = "caller-change"
	after, _ := storage.EncodePropertyJob(job)
	if !reflect.DeepEqual(encoded, after) {
		t.Fatal("changed retained metadata")
	}
}

func TestPropertyJobSourceDenialsExposeNoPartialAssessment(t *testing.T) {
	for _, mode := range []string{"first-read", "final-read", "ledger-drift", "job-drift", "foreign-id", "zero-digest", "zero-job", "native", "policy-drift", "cancel-first", "cancel-final", "missing-inspector", "missing-policy"} {
		t.Run(mode, func(t *testing.T) {
			policy := policySource(func(string) (storage.Policy, error) { return collectionPolicy(), nil })
			p, o, err := policy.Collect(context.Background(), "100")
			if err != nil {
				t.Fatal(err)
			}
			job := sourcePropertyJob(t, p, o)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			digest := sha256.Sum256([]byte("complete-test-ledger"))
			s := &PropertyJobSource{policy: policy, inspect: func(string) (storage.PropertyJob, [32]byte, error) {
				calls++
				if mode == "first-read" || mode == "final-read" && calls == 2 {
					return storage.PropertyJob{}, [32]byte{}, storage.ErrStateRecord
				}
				if mode == "cancel-first" || mode == "cancel-final" && calls == 2 {
					cancel()
				}
				if mode == "zero-job" {
					return storage.PropertyJob{}, digest, nil
				}
				if mode == "zero-digest" {
					return job, [32]byte{}, nil
				}
				if mode == "ledger-drift" && calls == 2 {
					digest[0] ^= 1
				}
				if mode == "job-drift" && calls == 2 {
					job, err = storage.TransitionPropertyJob(job, storage.JobApplying, job.Job().History[0].At+1, "launch-intent", "")
					if err != nil {
						t.Fatal(err)
					}
				}
				if mode == "foreign-id" {
					j := job.Job()
					j.ID = strings.Repeat("b", 64)
					job, err = storage.BindPropertyJob(j, job.Intent())
					if err != nil {
						t.Fatal(err)
					}
				}
				return job, digest, nil
			}}
			id := job.Job().ID
			switch mode {
			case "native":
				policy.pool.read = func(context.Context, poolProbe, string, int) (readOutput, error) { return readOutput{}, ErrExit }
			case "policy-drift":
				reads := 0
				policy.ownership = func(string) (storage.Policy, error) {
					reads++
					p := collectionPolicy()
					p.Revision = uint64(reads)
					return p, nil
				}
			case "missing-inspector":
				s.inspect = nil
			case "missing-policy":
				s.policy = nil
			}
			got, err := s.Assess(ctx, id)
			if err == nil || got != (storage.PropertyJobAssessment{}) {
				t.Fatal("partial denied result", got, err)
			}
		})
	}
}

func TestPropertyJobSourceSelectionBeforeReads(t *testing.T) {
	s := &PropertyJobSource{policy: &PropertyPolicySource{}, inspect: func(string) (storage.PropertyJob, [32]byte, error) {
		t.Fatal("invalid selection accessed metadata")
		return storage.PropertyJob{}, [32]byte{}, nil
	}}
	for _, id := range []string{"", "../writer.lock", strings.Repeat("A", 64), strings.Repeat("a", 63), strings.Repeat("a", 65), strings.Repeat("a", 1<<20)} {
		got, err := s.Assess(context.Background(), id)
		if err == nil || got != (storage.PropertyJobAssessment{}) {
			t.Fatal("invalid selection", err)
		}
	}
	if _, err := s.Assess(nil, strings.Repeat("a", 64)); err == nil {
		t.Fatal("nil context")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Assess(ctx, strings.Repeat("a", 64)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var absent *PropertyJobSource
	if _, err := absent.Assess(context.Background(), strings.Repeat("a", 64)); err == nil {
		t.Fatal("nil source")
	}
}

func FuzzPropertyJobSource(f *testing.F) {
	policy := policySource(func(string) (storage.Policy, error) { return collectionPolicy(), nil })
	p, o, err := policy.Collect(context.Background(), "100")
	if err != nil {
		f.Fatal(err)
	}
	job := sourcePropertyJob(f, p, o)
	encoded, _ := storage.EncodePropertyJob(job)
	for _, id := range []string{job.Job().ID, "", "../writer.lock", strings.Repeat("b", 64)} {
		f.Add(id, uint8(0))
	}
	f.Add(job.Job().ID, uint8(1))
	f.Add(job.Job().ID, uint8(2))
	f.Add(job.Job().ID, uint8(3))
	f.Fuzz(func(t *testing.T, id string, mode uint8) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		reads := 0
		digest := sha256.Sum256([]byte("bounded-test-ledger"))
		s := &PropertyJobSource{policy: policy, inspect: func(string) (storage.PropertyJob, [32]byte, error) {
			reads++
			if mode%4 == 1 && reads == 2 {
				digest[0] ^= 1
			}
			if mode%4 == 2 {
				return storage.PropertyJob{}, [32]byte{}, storage.ErrStateRecord
			}
			if mode%4 == 3 && reads == 2 {
				cancel()
			}
			return job, digest, nil
		}}
		got, err := s.Assess(ctx, id)
		pass := id == job.Job().ID && mode%4 == 0
		if pass {
			if err != nil || reads != 2 || !got.BaselineMatches || got.PostconditionMatches || got.Restart.Reason != "review-expired" {
				t.Fatal("wrong bound facts", got, err)
			}
		} else if err == nil || got != (storage.PropertyJobAssessment{}) {
			t.Fatal("partial failed source", got, err)
		}
		if storage.ValidatePropertyJobSelection(id) != nil && reads != 0 {
			t.Fatal("invalid selector accessed ledger")
		}
		after, _ := storage.EncodePropertyJob(job)
		if !reflect.DeepEqual(encoded, after) {
			t.Fatal("changed private history")
		}
	})
}
