// SPDX-License-Identifier: BSD-3-Clause
package main

import "fmt"

// Arithmetic is not custody authority. The native caller must first recheck the
// actual sealed objects, and independently export/reopen every counted byte.
func witnessArchiveSum(prior int64, refs []*dashboardWitnessRef) (int64, error) {
	if prior < 0 || prior > witnessArchiveCap || len(refs) > 2*witnessAttemptCap {
		return 0, errDashboardWitness
	}
	total := prior
	for i, ref := range refs {
		if ref == nil || ref.Name != fmt.Sprintf("original-%03d.wire", i+1) || ref.RawBytes < 0 || ref.RawBytes > witnessRawCap || ref.RawBytes > witnessArchiveCap-total {
			return 0, errDashboardWitness
		}
		total += ref.RawBytes
	}
	return total, nil
}
