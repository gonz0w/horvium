//go:build !horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"time"
)

// AnnotateDeadlineReturn is an identity adapter in ordinary builds. Only the
// explicitly tagged disposable fixture can enable diagnostic observations.
func AnnotateDeadlineReturn(_ context.Context, _ context.Context, err error, _ DeadlineSite, _ DeadlinePhase, _ time.Duration) error {
	return err
}
