// SPDX-License-Identifier: BSD-3-Clause
package main

import "testing"

func TestDashboardWitnessScheduleCoverage(t *testing.T) {
	samples := []witnessScheduleSample{{0, 1}, {100, 101}, {200, 202}}
	got, err := witnessScheduleCoverage(samples, 299, 100)
	if err != nil || got.Planned != 3 || got.Missing != 0 || got.Duplicates != 0 || got.Late != 0 {
		t.Fatalf("complete original schedule: %+v %v", got, err)
	}
	got, err = witnessScheduleCoverage([]witnessScheduleSample{{0, 1}, {200, 350}, {200, 360}}, 399, 100)
	if err != nil || got.Planned != 4 || got.Missing != 2 || got.Duplicates != 1 || got.Late != 2 {
		t.Fatalf("gaps, repeats and lateness must stay visible: %+v %v", got, err)
	}
	for _, samples := range [][]witnessScheduleSample{nil, {{-1, 1}}, {{0, -1}}, {{100, 99}}, {{0, 400}}, {{100, 101}, {0, 102}}} {
		if _, err := witnessScheduleCoverage(samples, 399, 100); err == nil {
			t.Fatalf("accepted invalid original schedule: %+v", samples)
		}
	}
	for _, bounds := range [][2]int64{{-1, 100}, {100, 0}, {100, -1}, {1 << 62, 1}} {
		if _, err := witnessScheduleCoverage([]witnessScheduleSample{{0, 1}}, bounds[0], bounds[1]); err == nil {
			t.Fatal("unbounded schedule accepted")
		}
	}
}
