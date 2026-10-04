//go:build !freebsd

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "context"

const nativeSupported = false

func readNative(context.Context, probe, int) ([]byte, error) { return nil, ErrUnsupported }
