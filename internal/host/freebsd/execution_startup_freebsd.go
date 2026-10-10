// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "context"

// Fixed original root startup review, using the same single-use process registry
// as commissioning. No active incarnation or native launch is granted.
func reviewNativeExecutionStartupOwner(ctx context.Context, journal, ledger string) (*commissionedExecutionOwner, error) {
	return nil, ErrUnavailable
}
