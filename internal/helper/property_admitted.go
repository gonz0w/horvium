// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"errors"

	"github.com/gonz0w/horvium/internal/storage"
)

// Original admission lease plus one original kernel-authenticated exchange.
// Private and unregistered; no storage/session or worker quiescence authority.
type admittedPropertyPeer struct {
	identity *admittedPropertyPeer
	lease    *propertyConnection
	exchange *propertyPeer
}

func (c *propertyConnection) accept(uid uint32) (a *admittedPropertyPeer, err error) {
	defer func() {
		if err != nil {
			err = errors.Join(err, c.Close())
		}
	}()
	ctx, conn, err := c.original()
	if err != nil {
		return nil, err
	}
	p := c.owner
	p.mu.Lock()
	if p.closed || p.failure != nil || p.owned[conn] != c || ctx.Err() != nil {
		p.mu.Unlock()
		return nil, ErrPermission
	}
	if c.bindingAttempted {
		p.mu.Unlock()
		return nil, ErrProtocol
	}
	c.bindingAttempted = true
	timeout := p.timeout
	p.mu.Unlock()
	// The original admission context caps the existing factory's timeout. Time
	// spent waiting for negotiation cannot create a fresh phase deadline.
	exchange, err := acceptPropertyPeer(ctx, conn, uid, timeout)
	if err != nil {
		return nil, err
	}
	a = &admittedPropertyPeer{lease: c, exchange: exchange}
	a.identity = a
	if err := a.live(); err != nil {
		return nil, errors.Join(err, a.Close())
	}
	return a, nil
}
func (a *admittedPropertyPeer) matchingLease() error {
	if a == nil || a.identity != a || a.lease == nil || a.exchange == nil || a.exchange.peer == nil || !a.exchange.root {
		return ErrPermission
	}
	ctx, conn, err := a.lease.original()
	if err != nil {
		return err
	}
	if ctx != a.lease.ctx || conn != a.exchange.peer.connection || a.exchange.peer.lifetime == nil {
		return ErrPermission
	}
	original, ok := ctx.Deadline()
	actual, bounded := a.exchange.peer.lifetime.Deadline()
	if !ok || !bounded || !original.Equal(actual) {
		return ErrPermission
	}
	return nil
}
func (a *admittedPropertyPeer) live() error {
	if err := a.matchingLease(); err != nil {
		return err
	}
	return a.exchange.peer.lifetime.Err()
}
func (a *admittedPropertyPeer) receive() (propertyRequest, actionIdentity, error) {
	zero := func(err error) (propertyRequest, actionIdentity, error) {
		return propertyRequest{}, actionIdentity{}, errors.Join(err, a.Close())
	}
	if err := a.live(); err != nil {
		return zero(err)
	}
	req, actor, err := a.exchange.receive()
	if err != nil {
		return zero(err)
	}
	if err := a.live(); err != nil {
		return zero(err)
	}
	return req, actor, nil
}
func (a *admittedPropertyPeer) replyReview(r storage.Review) (err error) {
	defer func() { err = errors.Join(err, a.Close()) }()
	if err := a.live(); err != nil {
		return err
	}
	if err := a.exchange.replyReview(r); err != nil {
		return err
	}
	// The typed owner intentionally retired after its terminal reply. Check the
	// original admission lifetime without asserting liveness of that closed peer.
	return a.matchingLease()
}
func (a *admittedPropertyPeer) replySubmission(j storage.Job, duplicate bool) (err error) {
	defer func() { err = errors.Join(err, a.Close()) }()
	if err := a.live(); err != nil {
		return err
	}
	if err := a.exchange.replySubmission(j, duplicate); err != nil {
		return err
	}
	return a.matchingLease()
}
func (a *admittedPropertyPeer) refuse(code string) (err error) {
	defer func() { err = errors.Join(err, a.Close()) }()
	if err := a.live(); err != nil {
		return err
	}
	if err := a.exchange.refuse(code); err != nil {
		return err
	}
	return a.matchingLease()
}
func (a *admittedPropertyPeer) Close() error {
	if a == nil || a.identity != a {
		return nil
	}
	return errors.Join(a.exchange.Close(), a.lease.Close())
}
