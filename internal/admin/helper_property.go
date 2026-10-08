// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/gonz0w/horvium/internal/helper"
	"github.com/gonz0w/horvium/internal/storage"
)

// Unregistered adapters: only original authenticated application authority is
// accepted here. Correlation sent to root remains an untrusted service assertion.
func propertySessionCheck(authority *SessionAuthority, scope string) func(context.Context) error {
	return func(ctx context.Context) error {
		current, err := authority.Context(ctx)
		if err != nil {
			return err
		}
		if current != scope {
			return ErrUnauthorized
		}
		return nil
	}
}
func closePropertySocket(c *net.UnixConn, err error) error {
	if c != nil {
		if e := c.Close(); e != nil && !errors.Is(e, net.ErrClosed) {
			return errors.Join(err, e)
		}
	}
	return err
}
func reviewHelperProperty(ctx context.Context, c *net.UnixConn, uid uint32, id, pool string, intent storage.Intent, timeout time.Duration, authority *SessionAuthority) (r storage.Review, err error) {
	defer func() {
		err = closePropertySocket(c, err)
		if err != nil {
			r = storage.Review{}
		}
	}()
	if err = helper.WritePropertyReview(io.Discard, id, pool, intent); err != nil {
		return storage.Review{}, err
	}
	scope, err := authority.Context(ctx)
	if err != nil {
		return storage.Review{}, err
	}
	peer, err := helper.ConnectPropertyClient(ctx, c, uid, id, scope, timeout, propertySessionCheck(authority, scope))
	if err != nil {
		return storage.Review{}, err
	}
	return peer.Review(pool, intent)
}
func submitHelperProperty(ctx context.Context, c *net.UnixConn, uid uint32, id, pool, plan, submission string, timeout time.Duration, authority *SessionAuthority) (j storage.Job, duplicate bool, err error) {
	defer func() {
		err = closePropertySocket(c, err)
		if err != nil {
			j, duplicate = storage.Job{}, false
		}
	}()
	if err = helper.WritePropertySubmission(io.Discard, id, pool, plan, submission); err != nil {
		return storage.Job{}, false, err
	}
	scope, err := authority.Context(ctx)
	if err != nil {
		return storage.Job{}, false, err
	}
	peer, err := helper.ConnectPropertyClient(ctx, c, uid, id, scope, timeout, propertySessionCheck(authority, scope))
	if err != nil {
		return storage.Job{}, false, err
	}
	return peer.Submit(pool, plan, submission)
}

// ReviewConfiguredProperty performs one typed, session-bound configured exchange.
// It registers no route and grants no root policy or mutation authority.
func ReviewConfiguredProperty(ctx context.Context, c helper.Config, id, pool string, intent storage.Intent, authority *SessionAuthority) (r storage.Review, err error) {
	if err = helper.WritePropertyReview(io.Discard, id, pool, intent); err != nil {
		return storage.Review{}, err
	}
	scope, err := authority.Context(ctx)
	if err != nil {
		return storage.Review{}, err
	}
	peer, err := helper.DialPropertyClient(ctx, c, id, scope, propertySessionCheck(authority, scope))
	if err != nil {
		return storage.Review{}, err
	}
	defer func() {
		err = errors.Join(err, peer.Close())
		if err != nil {
			r = storage.Review{}
		}
	}()
	return peer.Review(pool, intent)
}

// SubmitConfiguredProperty performs one typed, session-bound configured exchange.
// It registers no route and grants no root policy or mutation authority.
func SubmitConfiguredProperty(ctx context.Context, c helper.Config, id, pool, plan, submission string, authority *SessionAuthority) (j storage.Job, duplicate bool, err error) {
	if err = helper.WritePropertySubmission(io.Discard, id, pool, plan, submission); err != nil {
		return storage.Job{}, false, err
	}
	scope, err := authority.Context(ctx)
	if err != nil {
		return storage.Job{}, false, err
	}
	peer, err := helper.DialPropertyClient(ctx, c, id, scope, propertySessionCheck(authority, scope))
	if err != nil {
		return storage.Job{}, false, err
	}
	defer func() {
		err = errors.Join(err, peer.Close())
		if err != nil {
			j, duplicate = storage.Job{}, false
		}
	}()
	return peer.Submit(pool, plan, submission)
}
