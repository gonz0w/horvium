//go:build !freebsd

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "context"

func readPoolNative(context.Context, poolProbe, string, int) (readOutput, error) {
	return readOutput{}, ErrUnsupported
}
