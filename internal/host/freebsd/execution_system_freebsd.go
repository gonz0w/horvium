// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

// Fixed native producers only. No runtime consumer or write eligibility.
func observeNativeExecutionSystemDomains(ctx context.Context, p *storage.ExecutionDomainLeases, installation string) (storage.ExecutionDomainsObservation, error) {
	return storage.ExecutionDomainsObservation{}, storage.ErrUnavailable
}
