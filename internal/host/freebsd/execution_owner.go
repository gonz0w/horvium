// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"sync"

	"github.com/gonz0w/horvium/internal/storage"
)

// One commissioning attempt for the root helper lifetime. Never reset or copy.
type executionOwnerRegistry struct {
	mu       sync.Mutex
	original *executionOwnerRegistry
	claimed  bool
	owner    *commissionedExecutionOwner
}

func newExecutionOwnerRegistry() *executionOwnerRegistry {
	r := &executionOwnerRegistry{}
	r.original = r
	return r
}

var nativeExecutionOwners = newExecutionOwnerRegistry()

type executionOwnerSource struct {
	observe func(context.Context) (storage.ExecutionDomainsObservation, error)
	finish  func(context.Context) error
	release func() error
}
type commissionedExecutionOwner struct {
	mu              sync.Mutex
	original        *commissionedExecutionOwner
	registry        *executionOwnerRegistry
	source          executionOwnerSource
	root            func() bool
	blocked, closed bool
	closeErr        error
}

// Private synchronous seams only. Production fixes the process registry, actual
// root check and original native activation. No caller-facing factory is added.
func commissionExecutionOwner(ctx context.Context, r *executionOwnerRegistry, root func() bool, factory func(context.Context) (executionOwnerSource, error)) (*commissionedExecutionOwner, error) {
	if ctx == nil || r == nil || r.original != r || root == nil || factory == nil {
		return nil, storage.ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if !root() {
		return nil, ErrPermission
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	o := &commissionedExecutionOwner{registry: r, root: root}
	o.original = o
	r.mu.Lock()
	if e := ctx.Err(); e != nil {
		r.mu.Unlock()
		return nil, e
	}
	if !root() {
		r.mu.Unlock()
		return nil, ErrPermission
	}
	if e := ctx.Err(); e != nil {
		r.mu.Unlock()
		return nil, e
	}
	if r.claimed {
		r.mu.Unlock()
		return nil, storage.ErrDenied
	}
	// Reserve before any namespace/provisioning effect. The helper lifetime never
	// resets this registration, including failed construction and successful close.
	r.claimed = true
	r.owner = o
	r.mu.Unlock()
	source, e := factory(ctx)
	o.source = source
	if e == nil && (source.observe == nil || source.finish == nil || source.release == nil) {
		e = ErrUnavailable
	}
	e = errors.Join(e, ctx.Err())
	if !root() {
		e = errors.Join(e, ErrPermission)
	}
	if e != nil {
		o.blocked = true
		o.closed = true
		if source.release != nil {
			o.closeErr = source.release()
		}
		return nil, errors.Join(e, o.closeErr)
	}
	return o, nil
}

// Verify immutable original lineage before touching a copied owner mutex.
func (o *commissionedExecutionOwner) originalOwner() bool {
	if o == nil || o.original != o || o.registry == nil || o.registry.original != o.registry || o.root == nil || o.source.observe == nil || o.source.finish == nil || o.source.release == nil {
		return false
	}
	r := o.registry
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.claimed && r.owner == o
}
func (o *commissionedExecutionOwner) liveLocked() error {
	if o.closed {
		return storage.ErrDenied
	}
	if !o.root() {
		o.blocked = true
		return ErrPermission
	}
	if o.blocked {
		return storage.ErrStateUncertain
	}
	return nil
}
func (o *commissionedExecutionOwner) Observe(ctx context.Context) (storage.ExecutionDomainsObservation, error) {
	zero := storage.ExecutionDomainsObservation{}
	if ctx == nil || !o.originalOwner() {
		return zero, storage.ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	if e := o.liveLocked(); e != nil {
		return zero, e
	}
	got, e := o.source.observe(ctx)
	e = errors.Join(e, ctx.Err())
	if !o.root() {
		e = errors.Join(e, ErrPermission)
	}
	if e != nil {
		o.blocked = true
		return zero, e
	}
	return got, nil
}
func (o *commissionedExecutionOwner) FinishIdle(ctx context.Context) error {
	if ctx == nil || !o.originalOwner() {
		return storage.ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return e
	}
	if e := o.liveLocked(); e != nil {
		return e
	}
	// An entered finish is terminal even if acknowledgement is lost. Never invoke
	// the producer again, infer quiescence or release leases before explicit Close.
	o.blocked = true
	e := o.source.finish(ctx)
	e = errors.Join(e, ctx.Err())
	if !o.root() {
		e = errors.Join(e, ErrPermission)
	}
	return e
}
func (o *commissionedExecutionOwner) Close() error {
	if !o.originalOwner() {
		return storage.ErrDenied
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.closed {
		o.closed = true
		o.blocked = true
		rooted := o.root()
		o.closeErr = o.source.release()
		if !rooted || !o.root() {
			o.closeErr = errors.Join(o.closeErr, ErrPermission)
		}
	}
	return o.closeErr
}
