// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package freebsd

import "context"

func commissionNativeExecutionOwner(context.Context, string, string) (*commissionedExecutionOwner, error) {
	return nil, ErrUnsupported
}
