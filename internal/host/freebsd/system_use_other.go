//go:build !freebsd

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "context"

func readSystemNative(context.Context, systemProbe, string, int) (readOutput, error) {
	return readOutput{}, ErrUnsupported
}
