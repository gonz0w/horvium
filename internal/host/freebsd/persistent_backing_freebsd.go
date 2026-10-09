// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "context"

func collectNativePersistentBacking(ctx context.Context, fd int) (persistentBackingIdentity, error) {
	return persistentBackingIdentity{}, ErrUnavailable
}
