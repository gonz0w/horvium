// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

func nativeControlOwner(context.Context, storage.ControlTargets, string) controlOwnerAttempt {
	return controlOwnerAttempt{released: true, err: ErrUnsupported}
}
func nativeControlInspect(context.Context, storage.ControlTargets) controlInspectionAttempt {
	return controlInspectionAttempt{released: true, err: ErrUnsupported}
}
