// SPDX-License-Identifier: BSD-3-Clause
package main

import "errors"

// Fixed stage names explain rejection without exposing producer values, actor
// IDs, paths, underlying OS errors or authenticated response bodies.
type dashboardWitnessFailure string

func (e dashboardWitnessFailure) Error() string {
	return "invalid original dashboard witness: " + string(e)
}
func (e dashboardWitnessFailure) Unwrap() error { return errDashboardWitness }
func witnessFailure(stage string) error {
	switch stage {
	case "producer-read", "raw-limit", "archive-limit",
		"publication-original", "publication-eligibility", "publication-metadata", "publication-mismatch", "publication-context", "publication-cache", "publication-idle", "publication-capacity", "publication-failed",
		"control-closed", "control-write", "control-read", "control-receipt", "control-failed",
		"original-duplicate", "original-missing", "original-eligibility", "original-reference", "original-path", "original-object", "original-open", "original-descriptor", "original-decode", "original-hash", "original-changed",
		"request-read", "request-grammar", "actual-socket", "connector-channel", "connector-file", "connector-start", "connector-descriptor", "helper-peer", "connector-ack", "connector-wait", "connector-usage", "producer-request", "producer-capture", "producer-lifetime", "producer-forward", "original-index",
		"management-registration", "helper-registration", "control-operation", "receipt-object", "receipt-custody", "receipt-store", "front-cleanup":
		return dashboardWitnessFailure(stage)
	default:
		return dashboardWitnessFailure("unclassified")
	}
}
func witnessFailureFrom(e error, fallback string) error {
	var classified dashboardWitnessFailure
	if errors.As(e, &classified) {
		return witnessFailure(string(classified))
	}
	return witnessFailure(fallback)
}
