// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"errors"
	"sync"
)

// Private original-request provenance, never human/session, policy or launch
// permission. Only a received request on an original admitted root peer issues it.
type propertyRequestBinding struct {
	identity *propertyRequestBinding
	owner    *propertyRequestOwner
	peer     *admittedPropertyPeer
	request  propertyRequest
	actor    actionIdentity
}
type propertyRequestOwner struct {
	mu      sync.Mutex
	peer    *admittedPropertyPeer
	binding *propertyRequestBinding
	retired bool
	close   sync.Once
	failure error
}

func receivePropertyBinding(a *admittedPropertyPeer) (_ *propertyRequestBinding, err error) {
	defer func() {
		if err != nil {
			err = errors.Join(err, a.Close())
		}
	}()
	req, actor, err := a.receive()
	if err != nil {
		return nil, err
	}
	if a.bindings == nil {
		return nil, ErrPermission
	}
	owner := a.bindings
	owner.mu.Lock()
	if owner.peer != a || owner.binding != nil || owner.retired {
		owner.mu.Unlock()
		return nil, ErrPermission
	}
	r := &propertyRequestBinding{owner: owner, peer: a, request: req, actor: actor}
	r.identity = r
	owner.binding = r
	owner.mu.Unlock()
	ctx, _, err := a.lease.original()
	if err != nil {
		return nil, errors.Join(err, r.Close())
	}
	if err := r.check(ctx); err != nil {
		return nil, err
	}
	return r, nil
}
func (r *propertyRequestBinding) original() bool {
	return r != nil && r.identity == r && r.owner != nil && r.peer != nil &&
		r.peer.identity == r.peer && r.peer.bindings == r.owner
}
func (r *propertyRequestBinding) check(ctx context.Context) (err error) {
	if !r.original() {
		return ErrPermission
	}
	owned := false
	defer func() {
		if owned && err != nil {
			err = errors.Join(err, r.Close())
		}
	}()
	owner := r.owner
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.peer != r.peer || owner.binding != r {
		return ErrPermission
	}
	owned = true
	if owner.retired {
		return ErrPermission
	}
	if ctx == nil {
		return ErrConfig
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.peer.live(); err != nil {
		return err
	}
	exchange := r.peer.exchange
	// begin uses TryLock: do not wait for an overlapping reply's bounded I/O.
	if err := exchange.begin(true, propertyReply, true); err != nil {
		return err
	}
	defer exchange.mu.Unlock()
	if exchange.request != r.request {
		return ErrProtocol
	}
	actor, err := propertyIdentity(exchange.peer, true)
	if err != nil {
		return err
	}
	if actor != r.actor {
		return ErrPermission
	}
	if err := r.peer.live(); err != nil {
		return err
	}
	return ctx.Err()
}
func (r *propertyRequestBinding) snapshot(ctx context.Context) (propertyRequest, actionIdentity, error) {
	if err := r.check(ctx); err != nil {
		return propertyRequest{}, actionIdentity{}, err
	}
	return r.request, r.actor, nil
}
func (r *propertyRequestBinding) Close() error {
	if !r.original() {
		return nil
	}
	owner := r.owner
	owner.mu.Lock()
	if owner.peer != r.peer || owner.binding != r {
		owner.mu.Unlock()
		return nil
	}
	owner.retired = true
	owner.mu.Unlock()
	// Keep the first genuine cleanup result, including concurrent/repeated Close.
	// Mark retired before closing, and never hold the binding mutex in peer cleanup.
	owner.close.Do(func() { owner.failure = r.peer.Close() })
	return owner.failure
}
