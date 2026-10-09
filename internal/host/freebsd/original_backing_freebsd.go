// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

// Private fixed native bridge. No caller selects commands or supplies facts.
// Domain observation does not register a dual-lease owner or grant clearance.
func collectOriginalExecutionDomain(ctx context.Context, state *storage.StateDirectory) (storage.ExecutionDomainObservation, error) {
	return storage.ExecutionDomainObservation{}, ErrUnavailable
}
