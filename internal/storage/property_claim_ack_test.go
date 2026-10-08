// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/sha256"
	"testing"
)

func TestPropertyClaimApplyingAcknowledgementLineage(t *testing.T) {
	for _, mode := range []string{"exact", "no-attempt", "no-ack", "wrong-revision", "wrong-hash", "foreign", "changed", "queued", "observing", "successor-attempt", "missing"} {
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
			data, _ := EncodePropertyJob(current)
			c.applyingAttempted = true
			c.acknowledgedRevision = 2
			c.acknowledgedReceipt = sha256.Sum256(data)
			jobs := []PropertyJob{current}
			switch mode {
			case "no-attempt":
				c.applyingAttempted = false
			case "no-ack":
				c.acknowledgedReceipt = [32]byte{}
			case "wrong-revision":
				c.acknowledgedRevision++
			case "wrong-hash":
				c.acknowledgedReceipt[0] ^= 1
			case "foreign":
				copied := *c
				c = &copied
			case "changed":
				current.job.History[1].At++
				jobs[0] = current
			case "queued":
				current = queued
				jobs[0] = current
			case "observing":
				current, err = TransitionPropertyJob(current, JobObserving, current.job.History[1].At+1, "native-observation", queued.job.ID)
				jobs[0] = current
			case "successor-attempt":
				c.successorAttempted = true
			case "missing":
				jobs = nil
			}
			if err != nil {
				t.Fatal(err)
			}
			before := *c
			err = a.checkApplyingAcknowledgement(c, liveStateView(t, jobs...), current)
			if (err == nil) != (mode == "exact") {
				t.Fatal("acknowledged applying lineage", err)
			}
			if before != *c {
				t.Fatal("read check changed claim")
			}
		})
	}
}
