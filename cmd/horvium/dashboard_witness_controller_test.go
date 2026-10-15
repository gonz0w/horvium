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
