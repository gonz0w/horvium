// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestPropertyClaimSuccessorLineage(t *testing.T) {
	for _, mode := range []string{"observing", "applying-reconcile", "succeeded", "observing-reconcile", "no-ack", "foreign-claim", "wrong-revision", "wrong-receipt", "queued", "intent-drift", "history-drift", "current-drift", "missing", "other-pending", "same-terminal", "other-terminal", "reconcile-replay", "terminal-replay"} {
		t.Run(mode, func(t *testing.T) {
			queued := propertyJobFixture(t)
			var a propertyAdmissionLedger
			rememberLiveState(t, &a, queued)
			c, err := a.claim(liveStateView(t, queued), queued.job.ID)
			if err != nil {
				t.Fatal(err)
			}
			current, err := TransitionPropertyJob(queued, JobApplying, queued.job.History[0].At+1, "launch-intent", "")
			if err != nil {
				t.Fatal(err)
			}
			digest := fmt.Sprintf("%064x", 300)
			if mode == "succeeded" || mode == "observing-reconcile" || mode == "terminal-replay" {
				current, err = TransitionPropertyJob(current, JobObserving, current.job.History[1].At+1, "native-observation", digest)
			}
			if mode == "reconcile-replay" {
				current, err = TransitionPropertyJob(current, JobReconcileRequired, current.job.History[1].At+1, "executor-unverified", "")
			}
			if mode == "terminal-replay" {
				current, err = TransitionPropertyJob(current, JobSucceeded, current.job.History[2].At+1, "verified", digest)
			}
			data, e := EncodePropertyJob(current)
			if err != nil || e != nil {
				t.Fatal("fixture", err, e)
			}
			// Only actual post-sync publishers populate these volatile fields.
			// Synthetic tests model metadata lineage, not native authority.
			c.acknowledgedReceipt = sha256.Sum256(data)
			c.acknowledgedRevision = uint64(len(current.job.History))
			next := clonePropertyJob(current)
			switch mode {
			case "applying-reconcile", "observing-reconcile":
				next, err = TransitionPropertyJob(current, JobReconcileRequired, current.job.History[len(current.job.History)-1].At+1, "executor-unverified", current.job.History[len(current.job.History)-1].ObservationDigest)
			case "succeeded":
				next, err = TransitionPropertyJob(current, JobSucceeded, current.job.History[2].At+1, "verified", digest)
			case "reconcile-replay", "terminal-replay":
			default:
				next, err = TransitionPropertyJob(current, JobObserving, current.job.History[1].At+1, "native-observation", digest)
			}
			if err != nil {
				t.Fatal(err)
			}
			jobs := []PropertyJob{current}
			switch mode {
			case "no-ack":
				c.acknowledgedReceipt = [32]byte{}
				c.acknowledgedRevision = 0
			case "foreign-claim":
				copied := *c
				c = &copied
			case "wrong-revision":
				c.acknowledgedRevision++
			case "wrong-receipt":
				c.acknowledgedReceipt[0] ^= 1
			case "queued":
				current = queued
				jobs[0] = current
			case "intent-drift":
				next.intent.sessionDigest[0] ^= 1
			case "history-drift":
				next.job.History[0].At++
			case "current-drift":
				current.job.History[1].At++
				jobs[0] = current
			case "missing":
				jobs = nil
			case "other-pending", "same-terminal", "other-terminal":
				pool := "999"
				if mode == "same-terminal" {
					pool = current.job.Intent.PoolGUID
				}
				other := clonePropertyJob(queued)
				other.job.ID = fmt.Sprintf("%064x", 450)
				other.job.Intent.SubmissionID = fmt.Sprintf("%064x", 451)
				other.job.Intent.PlanID = fmt.Sprintf("%064x", 452)
				other.intent.review.ID = other.job.Intent.PlanID
				other.job.Intent.PoolGUID = pool
				other.intent.review.PoolGUID = pool
				other.intent.review.Pool.GUID = pool
				other.intent.review.Features.PoolGUID = pool
				other.intent.review.Topology.PoolGUID = pool
				if mode != "other-pending" {
					other, err = TransitionPropertyJob(other, JobCancelled, other.job.History[0].At+1, "operator-cancelled", "")
				}
				if err != nil {
					t.Fatal(err)
				}
				jobs = append(jobs, other)
			}
			before, _ := EncodePropertyJob(current)
			err = a.validateSuccessor(c, liveStateView(t, jobs...), current, next)
			positive := mode == "observing" || mode == "applying-reconcile" || mode == "succeeded" || mode == "observing-reconcile" || mode == "other-terminal"
			if (err == nil) != positive {
				t.Fatal("original acknowledged successor lineage", err)
			}
			after, _ := EncodePropertyJob(current)
			if string(before) != string(after) {
				t.Fatal("validation changed metadata")
			}
		})
	}
}
