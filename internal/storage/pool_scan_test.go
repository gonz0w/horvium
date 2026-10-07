// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"reflect"
	"strings"
	"testing"
)

const scanDate = "Wed Oct  7 20:00:00 2026"

func TestPoolScanReportedStates(t *testing.T) {
	tests := []struct {
		text, kind, state string
		errors            uint64
		progress          uint16
		reported          bool
	}{
		{"none requested", "none", "none", 0, 0, false},
		{"scrub repaired 0B in 00:00:03 with 0 errors on " + scanDate, "scrub", "completed", 0, 0, false},
		{"scrub repaired 1.50K in 2 days 03:04:05 with 7 errors on " + scanDate, "scrub", "completed", 7, 0, false},
		{"resilvered 128M in 00:00:05 with 0 errors on " + scanDate, "resilver", "completed", 0, 0, false},
		{"scrub canceled on " + scanDate, "scrub", "cancelled", 0, 0, false},
		{"resilver canceled on " + scanDate, "resilver", "cancelled", 0, 0, false},
		{"scrub in progress since " + scanDate + "\n1.50M / 2.00M scanned at 100K/s, 1.00M / 2.00M issued at 50.0K/s\n0B repaired, 50.00% done, 00:00:20 to go", "scrub", "running", 0, 5000, true},
		{"resilver in progress since " + scanDate + "\n1.50M / 2.00M scanned, 1.00M / 2.00M issued\n128K resilvered, 50.00% done, no estimated completion time", "resilver", "running", 0, 5000, true},
		{"scrub paused since " + scanDate + "\nscrub started on " + scanDate + "\n1.50M / 2.00M scanned, 1.00M / 2.00M issued\n0B repaired, 50.00% done", "scrub", "paused", 0, 5000, true},
	}
	for _, tc := range tests {
		got, err := ParsePoolScan(tc.text)
		if err != nil || got.Kind != tc.kind || got.State != tc.state || got.Errors != tc.errors || got.ErrorsReported != (tc.state == "completed") || got.ProgressReported != tc.reported || got.ProgressHundredths != tc.progress {
			t.Fatal("scan facts", tc.text, got, err)
		}
		if tc.state == "completed" && (got.EndedOn != scanDate || got.ProgressReported) {
			t.Fatal("completed scan invented progress or time")
		}
		if tc.state == "paused" && (got.PausedOn != scanDate || got.StartedOn != scanDate || got.EstimateDisplay != "" || got.ScanRateDisplay != "" || got.IssueRateDisplay != "") {
			t.Fatal("paused scan invented estimate/rates")
		}
	}
}

func TestPoolScanUnknownMalformedOrPartialIsUnavailable(t *testing.T) {
	valid := "scrub in progress since " + scanDate + "\n1.50M / 2.00M scanned, 1.00M / 2.00M issued\n0B repaired, 50.00% done, no estimated completion time"
	inputs := []string{"", "none requested\n", "error scrub in progress since " + scanDate, "resilvered (mirror-0) 128M in 00:00:03 with 0 errors on " + scanDate, "scrub repaired 0B in 00:00:01 with 18446744073709551616 errors on " + scanDate, "scrub repaired 0B in 00:99:01 with 0 errors on " + scanDate, "scrub repaired 0B in 00:00:01 with 00 errors on " + scanDate, "scrub canceled on not-a-date", "none requested\x00", string([]byte{255}), strings.Repeat("a", 1025), strings.Replace(valid, "50.00", "100.01", 1), strings.Replace(valid, "50.00", "NaN", 1), strings.Replace(valid, "repaired", "resilvered", 1), strings.Replace(valid, " / 2.00M issued", " / -1M issued", 1), strings.Split(valid, "\n")[0], valid + "\nforeign extra line"}
	for _, text := range inputs {
		got, err := ParsePoolScan(text)
		if err == nil || !reflect.DeepEqual(got, PoolScan{}) {
			t.Fatal("invalid scan exposed facts", text, got, err)
		}
	}
}

func FuzzPoolScan(f *testing.F) {
	for _, s := range []string{"none requested", "scrub repaired 0B in 00:00:01 with 0 errors on " + scanDate, "scrub paused since " + scanDate, ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, text string) {
		a, err := ParsePoolScan(text)
		if err != nil {
			if a != (PoolScan{}) {
				t.Fatal("partial scan")
			}
			return
		}
		b, e := ParsePoolScan(text)
		if e != nil || a != b {
			t.Fatal("unstable decode")
		}
		if a.State != "completed" && a.ErrorsReported || a.State != "running" && a.State != "paused" && a.ProgressReported {
			t.Fatal("invented counters")
		}
	})
}
