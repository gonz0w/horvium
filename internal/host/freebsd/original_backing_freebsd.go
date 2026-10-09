// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

// Private fixed native bridge. No caller selects commands or supplies facts.
// Domain observation does not register a dual-lease owner or grant clearance.
func collectOriginalExecutionDomain(ctx context.Context, state *storage.StateDirectory) (storage.ExecutionDomainObservation, error) {
	return state.ObserveExecutionDomain(ctx, func(ctx context.Context, fd int) (storage.ExecutionBackingObservation, error) {
		b, e := collectNativePersistentBacking(ctx, fd)
		if e != nil {
			return storage.ExecutionBackingObservation{}, e
		}
		return storage.ExecutionBackingObservation{Kind: b.kind, UFS: b.ufs, Pool: b.pool, Dataset: b.dataset}, nil
	})
}
