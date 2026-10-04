// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"errors"
	"github.com/gonz0w/horvium/internal/inventory"
	"testing"
	"time"
)

func TestPerformanceSamplesRequireBothOwnedProcesses(t *testing.T) {
	pids := [2]int{41, 42}
	s, e := parseUsage([]byte("41 1024 0:01.250\n42 2048 1:02.500\n"), pids)
	if e != nil || s.RSSBytes != 3<<20 || s.CPU != 63750*time.Millisecond {
		t.Fatal("wrong RSS units or CPU time")
	}
	for _, raw := range []string{"41 1024 0:00.1\n", "41 1 0:00.1\n41 2 0:00.2\n", "41 -1 0:00.1\n42 1 0:00.1\n", "41 1 0:61.0\n42 1 0:00.1\n", "41 1 bad\n42 1 0:00.1\n", "43 1 0:00.1\n42 1 0:00.1\n", "41 18446744073709551615 0:00.1\n42 1 0:00.1\n"} {
		if _, e := parseUsage([]byte(raw), pids); e == nil {
			t.Fatal("accepted incomplete/invalid process sample")
		}
	}
}
func TestIdleCPUUsesOneCoreAndRejectsCounterRegression(t *testing.T) {
	pct, e := idlePercent(time.Second, 2*time.Second, 300*time.Second)
	if e != nil || pct < 0.333 || pct > 0.334 {
		t.Fatal("wrong one-core CPU percentage")
	}
	if _, e = idlePercent(2*time.Second, time.Second, 300*time.Second); e == nil {
		t.Fatal("counter regression accepted")
	}
	if _, e = idlePercent(0, 0, 0); e == nil {
		t.Fatal("zero measurement accepted")
	}
}
func TestFixtureAccountsForGraphRowsAtRecordBoundary(t *testing.T) {
	s := performanceSnapshot(99990, 0)
	if _, e := inventory.CloneBounded(s, inventory.DefaultLimits()); e != nil {
		t.Fatal("exact 100,000 model-record fixture failed", e)
	}
	if _, e := inventory.CloneBounded(performanceSnapshot(99991, 0), inventory.DefaultLimits()); !errors.Is(e, inventory.ErrLimit) {
		t.Fatal("model row limit did not reject excess")
	}
}
