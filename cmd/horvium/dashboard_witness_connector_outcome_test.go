// SPDX-License-Identifier: BSD-3-Clause
package main

import "testing"

func TestWitnessConnectorBoundaryWait(t *testing.T) {
	for _, tc := range []struct {
		name, action string
		exited       bool
		code, signal int
		want         bool
	}{
		{"released", "release", true, 0, 0, true},
		{"eof", "channel-eof", true, 1, 0, true},
		{"term", "sigterm", false, -1, 15, true},
		{"kill", "sigkill", false, -1, 9, true},
		{"success is not eof", "channel-eof", true, 0, 0, false},
		{"signal is not eof", "channel-eof", false, -1, 9, false},
		{"wrong signal", "sigterm", false, -1, 9, false},
		{"failed release", "release", true, 1, 0, false},
		{"unknown action", "other", true, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := witnessConnectorWaitMatches(tc.action, tc.exited, tc.code, tc.signal); got != tc.want {
				t.Fatalf("wait classification=%t, want %t", got, tc.want)
			}
		})
	}
}
