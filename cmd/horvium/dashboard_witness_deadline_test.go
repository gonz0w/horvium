// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"testing"
	"time"
)

func TestDashboardWitnessConnectorKeepsOriginalCollectionDeadline(t *testing.T) {
	start := time.Unix(1000, 0)
	original := start.Add(45 * time.Second)
	for _, elapsed := range []time.Duration{0, 15 * time.Second, 35 * time.Second, 45*time.Second - time.Nanosecond} {
		got, e := witnessConnectorDeadline(start.Add(elapsed), original.UnixNano())
		if e != nil || !got.Equal(original) {
			t.Fatalf("elapsed %s changed original collection deadline: got %s want %s error %v", elapsed, got, original, e)
		}
	}
}
func TestDashboardWitnessConnectorRefusesInvalidCollectionDeadline(t *testing.T) {
	now := time.Unix(1000, 0)
	for _, stamp := range []int64{-1, 0, now.Add(-time.Nanosecond).UnixNano(), now.UnixNano(), now.Add(45*time.Second + time.Nanosecond).UnixNano()} {
		if _, e := witnessConnectorDeadline(now, stamp); e == nil {
			t.Fatal("expired or beyond-budget deadline accepted", stamp)
		}
	}
}
