// SPDX-License-Identifier: BSD-3-Clause
package main

import "testing"

func TestDashboardWitnessControllerLabels(t *testing.T) {
	for _, stage := range []string{"capture", "sealed", "forward", "metadata"} {
		for _, action := range []string{"sigterm", "sigkill"} {
			s, a, ok := witnessControllerLabel("controller-" + stage + "-" + action)
			if !ok || s != stage || a != action {
				t.Fatal("bounded controller case rejected")
			}
		}
	}
	for _, label := range []string{"controller-normal", "controller-capture-release", "controller-connect-sigkill", "controller-metadata-channel-eof", "controller-capture-sigkill-extra", "../controller-capture-sigkill", "controller-sealed-sigkill/escape", "controller--sigterm"} {
		if _, _, ok := witnessControllerLabel(label); ok {
			t.Fatal("unowned label accepted", label)
		}
	}
}

func TestDashboardWitnessControllerWaitRawStatus(t *testing.T) {
	for _, c := range []struct {
		action       string
		exited       bool
		code, signal int
		want         bool
	}{
		{"sigterm", false, -1, 15, true}, {"sigkill", false, -1, 9, true},
		{"sigterm", false, 0, 15, false}, {"sigkill", false, 0, 9, false},
		{"sigterm", true, 0, 0, false}, {"sigkill", false, -1, 15, false}, {"release", false, -1, 9, false},
	} {
		if got := witnessControllerWaitMatches(c.action, c.exited, c.code, c.signal); got != c.want {
			t.Fatalf("raw controller wait %s exited=%t code=%d signal=%d got=%t want=%t", c.action, c.exited, c.code, c.signal, got, c.want)
		}
	}
}
