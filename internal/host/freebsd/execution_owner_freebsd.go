// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
)

// Original root helper commissioning only. No configured startup, existing-state
// recovery, metadata mutation, worker or native launch consumer is registered.
func commissionNativeExecutionOwner(ctx context.Context, journal, ledger string) (*commissionedExecutionOwner, error) {
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	return commissionExecutionOwner(ctx, nativeExecutionOwners, executionRoot, func(c context.Context) (executionOwnerSource, error) {
		a, e := activateNativeFreshExecutionJournal(c, journal, ledger)
		if a == nil {
			return executionOwnerSource{}, e
		}
		return executionOwnerSource{observe: a.Observe, finish: a.FinishIdle, release: a.Close}, e
	})
}
