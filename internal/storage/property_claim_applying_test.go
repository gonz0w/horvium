// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"fmt"
	"testing"
)

func TestPropertyClaimApplyingExactSuccessor(t *testing.T) {
	p := propertyJobFixture(t)
	for _, mode := range []string{"exact", "nil", "foreign-claim", "foreign-current", "queued", "cancelled", "reconcile", "observing", "rewritten-intent", "rewritten-queued", "already-applying", "missing", "other-pending", "same-terminal", "other-terminal"} {
		t.Run(mode, func(t *testing.T) {
			var a propertyAdmissionLedger
			rememberLiveState(t, &a, p)
			c, err := a.claim(liveStateView(t, p), p.job.ID)
			if err != nil {
				t.Fatal(err)
			}
			current := clonePropertyJob(p)
			next, err := TransitionPropertyJob(p, JobApplying, p.job.History[0].At+1, "launch-intent", "")
			if err != nil {
				t.Fatal(err)
			}
			jobs := []PropertyJob{current}
			switch mode {
			case "nil":
				c = nil
			case "foreign-claim":
				c = &propertyStateClaimIdentity{id: p.job.ID}
			case "foreign-current":
				current.job.ID = fmt.Sprintf("%064x", 99)
			case "queued":
				next = clonePropertyJob(p)
			case "cancelled":
				next, err = TransitionPropertyJob(p, JobCancelled, p.job.History[0].At+1, "operator-cancelled", "")
			case "reconcile":
				next, err = TransitionPropertyJob(p, JobReconcileRequired, p.job.History[0].At+1, "review-expired", "")
			case "observing":
				next, err = TransitionPropertyJob(next, JobObserving, next.job.History[1].At+1, "native-observation", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
			case "rewritten-intent":
				next.intent.sessionDigest[0] ^= 1
			case "rewritten-queued":
				next.job.History[0].At++
			case "already-applying":
				current = clonePropertyJob(next)
				jobs[0] = current
			case "missing":
				jobs = nil
			case "other-pending", "same-terminal", "other-terminal":
				other := clonePropertyJob(p)
				other.job.ID = fmt.Sprintf("%064x", 20)
				other.job.Intent.SubmissionID = fmt.Sprintf("%064x", 21)
				other.job.Intent.PlanID = fmt.Sprintf("%064x", 22)
				other.intent.review.ID = other.job.Intent.PlanID
				if mode != "same-terminal" {
					other.job.Intent.PoolGUID = "999"
					other.intent.review.PoolGUID = "999"
					other.intent.review.Pool.GUID = "999"
					other.intent.review.Features.PoolGUID = "999"
					other.intent.review.Topology.PoolGUID = "999"
				}
				if mode != "other-pending" {
					other, err = TransitionPropertyJob(other, JobCancelled, other.job.History[0].At+1, "operator-cancelled", "")
				}
				jobs = append(jobs, other)
			}
			if err != nil {
				t.Fatal(err)
			}
			before, _ := EncodePropertyJob(current)
			err = a.validateApplying(c, liveStateView(t, jobs...), current, next)
			positive := mode == "exact" || mode == "other-terminal"
			if (err == nil) != positive {
				t.Fatal("exact original applying boundary", err)
			}
			after, _ := EncodePropertyJob(current)
			if string(before) != string(after) {
				t.Fatal("validation changed current metadata")
			}
		})
	}
}
