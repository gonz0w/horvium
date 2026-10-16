// SPDX-License-Identifier: BSD-3-Clause
package main

// Capture diagnostics are bounded observations, separate from original custody,
// forwarding and publication. A failed open has no captured original reference.
type witnessCaptureBoundary struct {
	ID, Management, Helper, Stage string
	AcknowledgedBytes             int64
	Error                         witnessForwardError
}
