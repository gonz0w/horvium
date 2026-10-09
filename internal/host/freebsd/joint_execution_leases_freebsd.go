// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

// Private native joint observation only. There is no runtime consumer or writer.
func openNativeExecutionDomains(ctx context.Context, journal, ledger string) (*storage.ExecutionDomainLeases, error) {
	return nil, ErrUnavailable
}
func observeNativeExecutionDomains(ctx context.Context, p *storage.ExecutionDomainLeases, installation string) (storage.ExecutionDomainsObservation, error) {
	return p.Observe(ctx, installation, readNativeExecutionBacking)
}
func readNativeExecutionBacking(ctx context.Context, fd int) (storage.ExecutionBackingObservation, error) {
	b, e := collectNativePersistentBacking(ctx, fd)
	if e != nil {
		return storage.ExecutionBackingObservation{}, e
	}
	return storage.ExecutionBackingObservation{Kind: b.kind, UFS: b.ufs, Pool: b.pool, Dataset: b.dataset}, nil
}
