// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"fmt"
	"reflect"
	"testing"
)

func liveStateView(t testing.TB, jobs ...PropertyJob) JobReservations {
	t.Helper()
	var records [][]byte
	for _, p := range jobs {
		b, err := EncodePropertyJob(p)
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, b)
	}
	v, err := BuildJobReservations(records)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// This seam models only the post-sync callback. It is not native admission.
func rememberLiveState(t testing.TB, a *propertyAdmissionLedger, p PropertyJob) {
	t.Helper()
	b, err := EncodePropertyJob(p)
	if err != nil {
		t.Fatal(err)
	}
	a.remember(p.job.ID, b)
}

func TestPropertyLiveStateOriginalReceipt(t *testing.T) {
	p := propertyJobFixture(t)
	v := liveStateView(t, p)
	var a propertyAdmissionLedger
	if _, err := a.claim(v, p.job.ID); err == nil {
		t.Fatal("decoded bytes created receipt")
	}
	rememberLiveState(t, &a, p)
	c, err := a.claim(v, p.job.ID)
	if err != nil || c == nil {
		t.Fatal(err)
	}
	if _, err := a.claim(v, p.job.ID); err == nil {
		t.Fatal("second claim")
	}
	if err := a.check(c, v, p, p.job.Intent.PoolGUID); err != nil {
		t.Fatal(err)
	}
	next, err := TransitionPropertyJob(p, JobApplying, p.job.History[0].At+1, "launch-intent", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.check(c, liveStateView(t, next), next, p.job.Intent.PoolGUID); err != nil {
		t.Fatal(err)
	}
	if err := checkPropertyStateJobs(v, p.job.Intent.PoolGUID); err == nil {
		t.Fatal("ordinary gate was relaxed")
	}
	var reopened propertyAdmissionLedger
	if err := reopened.check(c, v, p, p.job.Intent.PoolGUID); err == nil {
		t.Fatal("reopened ledger accepted old claim")
	}
}

func TestPropertyLiveStateWholeRecordAndStages(t *testing.T) {
	p := propertyJobFixture(t)
	for _, mode := range []string{"capture", "queued-time", "intent", "foreign-current", "wrong-pool", "missing", "zero-view", "nil-claim", "foreign-claim", "applying-before-claim", "observing", "cancelled", "reconcile"} {
		t.Run(mode, func(t *testing.T) {
			var a propertyAdmissionLedger
			rememberLiveState(t, &a, p)
			c, err := a.claim(liveStateView(t, p), p.job.ID)
			if err != nil {
				t.Fatal(err)
			}
			q := PropertyJob{job: p.Job(), intent: p.Intent()}
			guid := p.job.Intent.PoolGUID
			switch mode {
			case "capture":
				q.intent.capturedAt = q.intent.capturedAt.Add(-1)
			case "queued-time":
				q.job.History[0].At++
			case "intent":
				q.intent.sessionDigest[0] ^= 1
			case "foreign-current":
				q.job.ID = fmt.Sprintf("%064x", 77)
			case "wrong-pool":
				guid = "999"
			case "nil-claim":
				c = nil
			case "foreign-claim":
				c = &propertyStateClaimIdentity{id: p.job.ID}
			case "applying-before-claim":
				q, err = TransitionPropertyJob(q, JobApplying, q.job.History[0].At+1, "launch-intent", "")
				var b propertyAdmissionLedger
				rememberLiveState(t, &b, p)
				if _, err := b.claim(liveStateView(t, q), p.job.ID); err == nil {
					t.Fatal("late claim")
				}
				return
			case "observing":
				q, err = TransitionPropertyJob(q, JobApplying, q.job.History[0].At+1, "launch-intent", "")
				if err != nil {
					t.Fatal(err)
				}
				q, err = TransitionPropertyJob(q, JobObserving, q.job.History[1].At+1, "native-observation", fmt.Sprintf("%064x", 1))
			case "cancelled":
				q, err = TransitionPropertyJob(q, JobCancelled, q.job.History[0].At+1, "operator-cancelled", "")
			case "reconcile":
				q, err = TransitionPropertyJob(q, JobReconcileRequired, q.job.History[0].At+1, "review-expired", "")
			}
			if err != nil {
				t.Fatal(err)
			}
			v := liveStateView(t, q)
			if mode == "missing" {
				v = liveStateView(t)
			}
			if mode == "zero-view" {
				v = JobReservations{}
			}
			if err := a.check(c, v, q, guid); err == nil {
				t.Fatal("unsafe claim accepted")
			}
		})
	}
}

func TestPropertyLiveStateOtherHolds(t *testing.T) {
	p := propertyJobFixture(t)
	for _, mode := range []string{"same-terminal", "other-pending", "other-terminal"} {
		t.Run(mode, func(t *testing.T) {
			q := PropertyJob{job: p.Job(), intent: p.Intent()}
			q.job.ID = fmt.Sprintf("%064x", 20)
			q.job.Intent.SubmissionID = fmt.Sprintf("%064x", 21)
			q.job.Intent.PlanID = fmt.Sprintf("%064x", 22)
			q.intent.review.ID = q.job.Intent.PlanID
			if mode != "same-terminal" {
				q.job.Intent.PoolGUID = "999"
				q.intent.review.PoolGUID = "999"
				q.intent.review.Pool.GUID = "999"
				q.intent.review.Features.PoolGUID = "999"
				q.intent.review.Topology.PoolGUID = "999"
			}
			if mode != "other-pending" {
				var err error
				q, err = TransitionPropertyJob(q, JobCancelled, q.job.History[0].At+1, "operator-cancelled", "")
				if err != nil {
					t.Fatal(err)
				}
			}
			var a propertyAdmissionLedger
			rememberLiveState(t, &a, p)
			c, err := a.claim(liveStateView(t, p), p.job.ID)
			if err != nil {
				t.Fatal(err)
			}
			v := liveStateView(t, p, q)
			before := v.Pools()
			err = a.check(c, v, p, p.job.Intent.PoolGUID)
			if (err == nil) != (mode == "other-terminal") {
				t.Fatal("hold rule", err)
			}
			if !reflect.DeepEqual(before, v.Pools()) {
				t.Fatal("released holds")
			}
		})
	}
}

func TestPropertyLiveStateBoundsAndGlobalClaim(t *testing.T) {
	p := propertyJobFixture(t)
	var a propertyAdmissionLedger
	for n := 0; n < maxPropertyAdmissionReceipts+1; n++ {
		q := PropertyJob{job: p.Job(), intent: p.Intent()}
		q.job.ID = fmt.Sprintf("%064x", n+1)
		rememberLiveState(t, &a, q)
	}
	if len(a.receipts) != maxPropertyAdmissionReceipts {
		t.Fatal("unbounded receipts")
	}
	q := PropertyJob{job: p.Job(), intent: p.Intent()}
	q.job.ID = fmt.Sprintf("%064x", maxPropertyAdmissionReceipts+1)
	if _, err := a.claim(liveStateView(t, q), q.job.ID); err == nil {
		t.Fatal("exhausted capture created authority")
	}
	p.job.ID = fmt.Sprintf("%064x", 1)
	if _, err := a.claim(liveStateView(t, p), p.job.ID); err != nil {
		t.Fatal(err)
	}
	q.job.ID = fmt.Sprintf("%064x", 2)
	if _, err := a.claim(liveStateView(t, q), q.job.ID); err == nil {
		t.Fatal("second global claimant")
	}
}

func TestPropertyLiveStateReceiptCannotRebindAndCloseInvalidates(t *testing.T) {
	p := propertyJobFixture(t)
	s := &StateDirectory{}
	rememberLiveState(t, &s.propertyAdmissions, p)
	q := PropertyJob{job: p.Job(), intent: p.Intent()}
	q.job.History[0].At++
	rememberLiveState(t, &s.propertyAdmissions, q)
	if _, err := s.propertyAdmissions.claim(liveStateView(t, q), q.job.ID); err == nil {
		t.Fatal("replacement receipt accepted")
	}
	c, err := s.propertyAdmissions.claim(liveStateView(t, p), p.job.ID)
	if err != nil {
		t.Fatal("failed claim consumed lineage", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if len(s.propertyAdmissions.receipts) != 0 || s.propertyAdmissions.active != nil {
		t.Fatal("close retained volatile admission")
	}
	if err := s.propertyAdmissions.check(c, liveStateView(t, p), p, p.job.Intent.PoolGUID); err == nil {
		t.Fatal("closed claim accepted")
	}
}

func FuzzPropertyLiveStateReceipt(f *testing.F) {
	f.Add([]byte{0, 1, 2, 1})
	f.Fuzz(func(t *testing.T, actions []byte) {
		if len(actions) > 32 {
			return
		}
		p := propertyJobFixture(t)
		v := liveStateView(t, p)
		var a propertyAdmissionLedger
		grants := 0
		for _, action := range actions {
			switch action % 3 {
			case 0:
				rememberLiveState(t, &a, p)
			case 1:
				if _, err := a.claim(v, p.job.ID); err == nil {
					grants++
				}
			case 2:
				if a.active != nil {
					_ = a.check(a.active, v, p, p.job.Intent.PoolGUID)
				}
			}
			if grants > 1 {
				t.Fatal("multiple live claims")
			}
		}
	})
}
