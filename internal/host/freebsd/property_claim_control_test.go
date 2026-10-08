// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gonz0w/horvium/internal/storage"
)

func claimControlFixture(t testing.TB) (*propertyClaimControlAuthority, storage.PropertyJob, *int) {
	t.Helper()
	job := claimSourceJob(t)
	job, err := storage.TransitionPropertyJob(job, storage.JobApplying, job.Job().History[0].At+1, "launch-intent", "")
	if err != nil {
		t.Fatal(err)
	}
	checks := new(int)
	a := &propertyClaimControlAuthority{expected: job, live: func(context.Context) error { return nil },
		verify: func(current storage.PropertyJob) error {
			*checks++
			if !reflect.DeepEqual(current, job) {
				t.Fatal("foreign expected record")
			}
			return nil
		},
		collect: func(context.Context, string) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
			p, o, err := claimSourceSafety().Collect(context.Background(), job.Job().Intent.PoolGUID)
			return job, p, o, err
		}}
	return a, job, checks
}

func TestPropertyClaimControlAuthorityExactRecordAndLiveChecks(t *testing.T) {
	a, expected, checks := claimControlFixture(t)
	live := 0
	a.live = func(context.Context) error { live++; return nil }
	got, p, o, err := a.authorize(context.Background(), expected.Intent())
	if err != nil || *checks != 2 || live != 3 || !reflect.DeepEqual(got, expected) || o.Eligible {
		t.Fatal("bound applying authorization", err, *checks, live)
	}
	assessment, err := got.AssessObservation(p, o)
	if err != nil || !assessment.BaselineMatches {
		t.Fatal("baseline", err)
	}
	j := got.Job()
	j.History[0].At++
	if reflect.DeepEqual(j, a.expected.Job()) {
		t.Fatal("aliased reply")
	}
}

func TestPropertyClaimControlAuthorityDenialsDiscardProjections(t *testing.T) {
	for _, mode := range []string{"nil", "no-live", "no-verify", "no-collect", "live-first", "live-after-read", "live-final", "ack-first", "ack-final", "collect", "changed-record", "different-intent", "queued", "eligible", "native-drift", "cancel-read", "cancel-final", "nil-context"} {
		t.Run(mode, func(t *testing.T) {
			a, expected, checks := claimControlFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			live := 0
			a.live = func(context.Context) error {
				live++
				if mode == "live-first" || mode == "live-after-read" && live == 2 || mode == "live-final" && live == 3 {
					return storage.ErrExpired
				}
				return nil
			}
			verify := a.verify
			a.verify = func(j storage.PropertyJob) error {
				err := verify(j)
				if mode == "ack-first" || mode == "ack-final" && *checks == 2 {
					return storage.ErrDenied
				}
				if mode == "cancel-final" && *checks == 2 {
					cancel()
				}
				return err
			}
			collect := a.collect
			a.collect = func(c context.Context, id string) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
				j, p, o, err := collect(c, id)
				switch mode {
				case "collect":
					err = ErrUnavailable
				case "changed-record":
					r := j.Job()
					r.History[1].At++
					j, err = storage.BindPropertyJob(r, j.Intent())
				case "eligible":
					o.Eligible = true
				case "native-drift":
					p.Revision++
				case "cancel-read":
					cancel()
				}
				return j, p, o, err
			}
			intent := expected.Intent()
			switch mode {
			case "nil":
				a = nil
			case "no-live":
				a.live = nil
			case "no-verify":
				a.verify = nil
			case "no-collect":
				a.collect = nil
			case "different-intent":
				other := sourcePropertyJob(t, safetyPolicy(), mustClaimControlObservation(t))
				intent = other.Intent()
			case "queued":
				a.expected = claimSourceJob(t)
			case "nil-context":
				ctx = nil
			}
			j, p, o, err := a.authorize(ctx, intent)
			if err == nil || !reflect.DeepEqual(j, storage.PropertyJob{}) || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
				t.Fatal("denied authority retained projections", err)
			}
		})
	}
}

func mustClaimControlObservation(t testing.TB) storage.Observation {
	t.Helper()
	_, o, err := claimSourceSafety().Collect(context.Background(), "100")
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestPropertyClaimControlBindingRequiresNativeOriginalClaim(t *testing.T) {
	if _, err := newPropertyClaimControl(nil, storage.PropertyJob{}, nil); err == nil {
		t.Fatal("missing original claim/clearance")
	}
	if !nativeSupported {
		if _, err := newPropertyClaimControl(&storage.PropertyStateClaim{}, claimSourceJob(t), func(context.Context) error { return nil }); !errors.Is(err, ErrUnsupported) {
			t.Fatal(err)
		}
	}
}
