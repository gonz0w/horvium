// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"context"
	"io"

	"github.com/gonz0w/horvium/internal/helper"
)

// negotiateHelper is an unregistered application adapter, not a socket dialer
// or an authorization result. The eventual transport owner must authenticate the
// root peer, bound deadlines/lifetime and retire a failed connection. No command,
// credential, claimed UID or mutation payload is sent. Negotiation never renews
// the session or supplies root policy/recovery/launch clearance.
func negotiateHelper(ctx context.Context, transport io.ReadWriter, id string, authority *SessionAuthority) error {
	if transport == nil {
		return ErrUnauthorized
	}
	scope, err := authority.Context(ctx)
	if err != nil {
		return err
	}
	if err := helper.WriteActionNegotiation(transport, id, scope); err != nil {
		return err
	}
	// A write can block long enough for logout/expiry; do not wait for a response
	// after the original session has already been revoked.
	if err := authority.CheckLive(ctx); err != nil {
		return err
	}
	if err := helper.ReadActionNegotiation(transport, id, scope); err != nil {
		return err
	}
	current, err := authority.Context(ctx)
	if err != nil {
		return err
	}
	if current != scope {
		return ErrUnauthorized
	}
	return nil
}
