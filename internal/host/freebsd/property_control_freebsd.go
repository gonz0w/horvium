// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"os"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// Private and disconnected from runtime. Only a qualified original live root
// backend claim may bind this runner; current disposable tests are its sole caller.
// Errors NEVER establish quiescence or a safe replay, even when no effect is seen.
func propertySetNative(ctx context.Context, i storage.PropertyIntent) error {
	if os.Geteuid() != 0 {
		return ErrPermission
	}
	if ctx == nil {
		return storage.ErrInvalid
	}
	args, err := propertySetArguments(i)
	if err != nil {
		return err
	}
	ctx, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	out, err := executeNative(ctx, "/sbin/zfs", args, 4096)
	if err != nil {
		return err
	}
	if len(out.data) != 0 {
		return ErrUnavailable
	}
	return nil
}
