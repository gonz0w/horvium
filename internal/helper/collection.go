// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"io"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

func readCollection(ctx context.Context, r io.Reader, id string, limits Limits) (inventory.Snapshot, error) {
	s, err := readSnapshot(r, id, limits)
	if contextError := ctx.Err(); contextError != nil {
		return inventory.Snapshot{}, contextError
	}
	// Socket and context timers need not publish expiry in the same scheduler
	// turn. Enforce the existing absolute window even while Err is still nil.
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return inventory.Snapshot{}, context.DeadlineExceeded
	}
	return s, err
}
