// SPDX-License-Identifier: BSD-3-Clause
package main

import "testing"

func TestDashboardWitnessArchiveSumUsesActualBoundedOriginals(t *testing.T) {
	for _, tc := range []struct {
		name  string
		prior int64
		refs  []*dashboardWitnessRef
		want  int64
		bad   bool
	}{
		{"empty", 0, nil, 0, false},
		{"actual-prefix", 100, []*dashboardWitnessRef{{Name: "original-001.wire", RawBytes: 80}}, 180, false},
		{"exact-cap", witnessArchiveCap - 80, []*dashboardWitnessRef{{Name: "original-001.wire", RawBytes: 80}}, witnessArchiveCap, false},
		{"reserve-80", witnessArchiveCap - 180, []*dashboardWitnessRef{{Name: "original-001.wire", RawBytes: 100}}, witnessArchiveCap - 80, false},
		{"negative-prior", -1, nil, 0, true},
		{"over-cap-prior", witnessArchiveCap + 1, nil, 0, true},
		{"nil-original", 0, []*dashboardWitnessRef{nil}, 0, true},
		{"negative-original", 0, []*dashboardWitnessRef{{Name: "original-001.wire", RawBytes: -1}}, 0, true},
		{"oversized-original", 0, []*dashboardWitnessRef{{Name: "original-001.wire", RawBytes: witnessRawCap + 1}}, 0, true},
		{"over-budget", witnessArchiveCap - 80, []*dashboardWitnessRef{{Name: "original-001.wire", RawBytes: 81}}, 0, true},
		{"duplicate-original", 0, []*dashboardWitnessRef{{Name: "original-001.wire", RawBytes: 80}, {Name: "original-001.wire", RawBytes: 80}}, 0, true},
		{"unknown-slot", 0, []*dashboardWitnessRef{{Name: "../unowned", RawBytes: 80}}, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, e := witnessArchiveSum(tc.prior, tc.refs)
			if (e != nil) != tc.bad || got != tc.want {
				t.Fatal("unqualified archive budget", got, e)
			}
		})
	}
}
