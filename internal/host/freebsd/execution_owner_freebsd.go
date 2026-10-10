// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

// Original root helper commissioning only. No configured startup, existing-state
// recovery, metadata mutation, worker or native launch consumer is registered.
func commissionNativeExecutionOwner(context.Context, string, string) (*commissionedExecutionOwner, error) {
	return nil, storage.ErrDenied
}
