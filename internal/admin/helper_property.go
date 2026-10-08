// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"context"
	"github.com/gonz0w/horvium/internal/helper"
	"github.com/gonz0w/horvium/internal/storage"
	"net"
	"time"
)

// Unregistered adapters: only original authenticated application authority is
// accepted here. Correlation sent to root remains an untrusted service assertion.
func propertySessionCheck(authority *SessionAuthority, scope string) func(context.Context) error {
	return func(context.Context) error { return ErrUnauthorized }
}
func reviewHelperProperty(ctx context.Context, c *net.UnixConn, uid uint32, id, pool string, intent storage.Intent, timeout time.Duration, authority *SessionAuthority) (storage.Review, error) {
	if c != nil {
		_ = c.Close()
	}
	return storage.Review{}, helper.ErrPermission
}
func submitHelperProperty(ctx context.Context, c *net.UnixConn, uid uint32, id, pool, plan, submission string, timeout time.Duration, authority *SessionAuthority) (storage.Job, bool, error) {
	if c != nil {
		_ = c.Close()
	}
	return storage.Job{}, false, helper.ErrPermission
}
