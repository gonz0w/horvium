// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

// Fixed original root startup review, using the same single-use process registry
// as commissioning. No active incarnation or native launch is granted.
func reviewNativeExecutionStartupOwner(ctx context.Context, journal, ledger string) (*commissionedExecutionOwner, error) {
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	return commissionExecutionOwner(ctx, nativeExecutionOwners, executionRoot, func(c context.Context) (executionOwnerSource, error) {
		source, e := NewSystemUseSource()
		if e != nil {
			return executionOwnerSource{}, e
		}
		r, e := storage.ReviewExecutionStartup(c, journal, ledger, readNativeExecutionBacking, source.Collect, readNativeActivationBoot)
		if r == nil {
			return executionOwnerSource{}, e
		}
		return executionOwnerSource{observe: r.Observe, finish: func(context.Context) error { return storage.ErrDenied }, release: r.Close}, e
	})
}
