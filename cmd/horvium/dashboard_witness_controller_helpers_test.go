// SPDX-License-Identifier: BSD-3-Clause
package main

import "strings"

func witnessControllerLabel(label string) (stage, action string, ok bool) {
	parts := strings.Split(label, "-")
	if len(parts) != 3 || parts[0] != "controller" {
		return "", "", false
	}
	if !witnessOneOf(parts[1], "capture", "sealed", "forward", "metadata") || !witnessOneOf(parts[2], "sigterm", "sigkill") {
		return "", "", false
	}
	return parts[1], parts[2], true
}
