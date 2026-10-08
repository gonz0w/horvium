// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// PropertyClient owns one application-side exchange. It is not root-side session,
// policy, durable admission or launch authority; there is no runtime consumer.
type PropertyClient struct {
	identity *PropertyClient
	exchange *propertyPeer
	cancel   context.CancelFunc
}

// ConnectPropertyClient transfers the original socket, including on failure.
// check is a trusted application callback, never a replacement kernel UID hook.
// uid/timeout come from trusted configuration, not request fields. The caller
// still owns directory/socket policy and must supply the original live session.
func ConnectPropertyClient(ctx context.Context, c *net.UnixConn, uid uint32, id, scope string, timeout time.Duration, check func(context.Context) error) (_ *PropertyClient, err error) {
	var p *actionPeer
	defer func() {
		if err == nil {
			return
		}
		if p != nil {
			err = errors.Join(err, p.Close())
		} else if c != nil {
			if e := c.Close(); e != nil && !errors.Is(e, net.ErrClosed) {
				err = errors.Join(err, e)
			}
		}
	}()
	if check == nil || !validID(id) || !validApplicationContext(scope) {
		return nil, ErrConfig
	}
	p, err = ownedActionPeer(ctx, c, uid, timeout, false)
	if err != nil {
		return nil, err
	}
	guard := func() error { return check(p.lifetime) }
	if err = guard(); err != nil {
		return nil, err
	}
	if err = WriteActionNegotiation(c, id, scope); err != nil {
		return nil, err
	}
	if err = guard(); err != nil {
		return nil, err
	}
	if err = p.check(false); err != nil {
		return nil, err
	}
	if err = ReadActionNegotiation(c, id, scope); err != nil {
		return nil, err
	}
	p.requestID, p.applicationContext, p.negotiated = id, scope, true
	if err = p.checkLive(); err != nil {
		return nil, err
	}
	if err = guard(); err != nil {
		return nil, err
	}
	x := &PropertyClient{exchange: &propertyPeer{peer: p, phase: propertyAwait, application: guard}}
	x.identity = x
	return x, nil
}
func (c *PropertyClient) original() bool {
	return c != nil && c.identity == c && c.exchange != nil && c.exchange.application != nil && !c.exchange.root
}
func (c *PropertyClient) Review(pool string, intent storage.Intent) (r storage.Review, err error) {
	if !c.original() {
		return storage.Review{}, ErrPermission
	}
	defer func() {
		err = errors.Join(err, c.Close())
		if err != nil {
			r = storage.Review{}
		}
	}()
	return c.exchange.review(pool, intent)
}
func (c *PropertyClient) Submit(pool, plan, submission string) (j storage.Job, duplicate bool, err error) {
	if !c.original() {
		return storage.Job{}, false, ErrPermission
	}
	defer func() {
		err = errors.Join(err, c.Close())
		if err != nil {
			j, duplicate = storage.Job{}, false
		}
	}()
	return c.exchange.submit(pool, plan, submission)
}
func (c *PropertyClient) Close() error {
	if !c.original() {
		return nil
	}
	return c.exchange.Close()
}
