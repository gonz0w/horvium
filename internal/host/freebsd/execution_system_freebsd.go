// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

// Fixed native producers only. No runtime consumer or write eligibility.
func observeNativeExecutionSystemDomains(ctx context.Context, p *storage.ExecutionDomainLeases, installation string) (storage.ExecutionDomainsObservation, error) {
	system, e := NewSystemUseSource()
	if e != nil {
		return storage.ExecutionDomainsObservation{}, e
	}
	return p.ObserveSystem(ctx, installation, readNativeExecutionBacking, system.Collect)
}
