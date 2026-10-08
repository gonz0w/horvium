// SPDX-License-Identifier: BSD-3-Clause
package helper

import "context"

// DialPropertyClient owns one configured application connection. Configuration
// is trusted; application checks supplement actual kernel UID and socket policy.
// This is unregistered and grants no root session/backend or launch authority.
func DialPropertyClient(ctx context.Context, c Config, id, scope string, check func(context.Context) error) (*PropertyClient, error) {
	return nil, ErrPermission
}
