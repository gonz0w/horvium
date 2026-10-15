// SPDX-License-Identifier: BSD-3-Clause
package main

import "strings"

// Reframe after changing the ID: decimal counters do not retain byte width.
func witnessCapacityGolden(id string) []byte {
	records := witnessGolden()
	for i := range records {
		records[i] = strings.ReplaceAll(records[i], witnessTestID, id)
	}
	return witnessFrames(records)
}
