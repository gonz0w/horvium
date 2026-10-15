// SPDX-License-Identifier: BSD-3-Clause
package main

import "time"

func witnessConnectorDeadline(now time.Time, stamp int64) (time.Time, error) {
	deadline := time.Unix(0, stamp)
	if stamp <= 0 || !deadline.After(now) || deadline.After(now.Add(45*time.Second)) {
		return time.Time{}, errDashboardWitness
	}
	return deadline, nil
}
