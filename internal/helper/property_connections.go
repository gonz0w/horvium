// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

// Private network admission only. No peer/session, storage or launch authority.
// A future listener must call admit before starting a worker or consuming bytes.
type propertyConnections struct {
	mu       sync.Mutex
	identity *propertyConnections
	ctx      context.Context
	cancel   context.CancelFunc
	stop     func() bool
	limit    int
	timeout  time.Duration
	owned    map[*net.UnixConn]*propertyConnection
	closed   bool
	failure  error
}
type propertyConnection struct {
	owner            *propertyConnections
	connection       *net.UnixConn
	ctx              context.Context
	cancel           context.CancelFunc
	stop             func() bool
	done             chan struct{}
	bindingAttempted bool
}

func newPropertyConnections(parent context.Context, limit int, timeout time.Duration) (*propertyConnections, error) {
	if parent == nil || limit < 1 || limit > 64 || timeout <= 0 || timeout > 5*time.Minute {
		return nil, ErrConfig
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	p := &propertyConnections{ctx: ctx, cancel: cancel, limit: limit, timeout: timeout, owned: make(map[*net.UnixConn]*propertyConnection)}
	p.identity = p
	p.mu.Lock()
	p.stop = context.AfterFunc(ctx, func() { _ = p.Close() })
	p.mu.Unlock()
	if err := ctx.Err(); err != nil {
		_ = p.Close()
		return nil, err
	}
	return p, nil
}
func closePropertySocket(c *net.UnixConn) error {
	if c == nil {
		return nil
	}
	if err := c.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}

// Ownership transfers on entry, including denial. Only this owner may publish a
// slot, after installing its original deadline. No goroutine waits for capacity.
func (p *propertyConnections) admit(c *net.UnixConn) (lease *propertyConnection, err error) {
	entered := time.Now()
	defer func() {
		if err != nil {
			if failure := closePropertySocket(c); failure != nil {
				err = errors.Join(err, failure)
				if p != nil {
					p.mu.Lock()
					if p.identity == p {
						p.failure = errors.Join(p.failure, failure)
					}
					p.mu.Unlock()
					_ = p.Close()
				}
			}
		}
	}()
	if p == nil || c == nil {
		return nil, ErrConfig
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.identity != p || p.ctx == nil || p.cancel == nil || p.owned == nil || p.closed || p.failure != nil {
		return nil, ErrPermission
	}
	if previous := p.owned[c]; previous != nil {
		failure := p.retireLocked(previous)
		return nil, errors.Join(ErrProtocol, failure)
	}
	if err := p.ctx.Err(); err != nil {
		return nil, err
	}
	if len(p.owned) >= p.limit {
		return nil, ErrLimit
	}
	ctx, cancel := context.WithDeadline(p.ctx, entered.Add(p.timeout))
	deadline, _ := ctx.Deadline()
	if err := ctx.Err(); err != nil {
		cancel()
		return nil, err
	}
	if !time.Now().Before(deadline) {
		cancel()
		return nil, context.DeadlineExceeded
	}
	if err := c.SetDeadline(deadline); err != nil {
		cancel()
		return nil, ErrPermission
	}
	lease = &propertyConnection{owner: p, connection: c, ctx: ctx, cancel: cancel, done: make(chan struct{})}
	p.owned[c] = lease
	// Capture the installed identity, never the named return slot: a concurrent
	// cancellation may change that return slot to nil while this callback runs.
	held := lease
	held.stop = context.AfterFunc(ctx, func() { _ = held.Close() })
	if err := ctx.Err(); err != nil {
		_ = p.retireLocked(lease)
		return nil, err
	}
	return lease, nil
}
func (p *propertyConnections) retireLocked(c *propertyConnection) error {
	if c == nil || c.owner != p || p.owned[c.connection] != c {
		return nil
	}
	if c.stop != nil {
		c.stop()
	}
	err := closePropertySocket(c.connection)
	c.cancel()
	delete(p.owned, c.connection)
	close(c.done)
	if err != nil {
		p.failure = errors.Join(p.failure, err)
	}
	return err
}

// Shutdown closes all descriptors before returning. It does not claim that a
// caller's worker or any storage command has terminated. Close failures latch.
func (p *propertyConnections) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.identity != p {
		return nil
	}
	p.closed = true
	if p.stop != nil {
		p.stop()
	}
	if p.cancel != nil {
		p.cancel()
	}
	for _, c := range p.owned {
		_ = p.retireLocked(c)
	}
	return p.failure
}

// A future exchange must use BOTH original values. Its own shorter timeout may
// narrow this lifetime; using a new background context would discard admission.
func (c *propertyConnection) original() (context.Context, *net.UnixConn, error) {
	if c == nil || c.owner == nil {
		return nil, nil, ErrPermission
	}
	p := c.owner
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.identity != p || c.connection == nil || p.owned[c.connection] != c {
		return nil, nil, ErrPermission
	}
	fail := func(err error) (context.Context, *net.UnixConn, error) { _ = p.retireLocked(c); return nil, nil, err }
	if p.closed || p.failure != nil {
		return fail(ErrPermission)
	}
	if err := c.ctx.Err(); err != nil {
		return fail(err)
	}
	deadline, ok := c.ctx.Deadline()
	if !ok || !time.Now().Before(deadline) {
		return fail(context.DeadlineExceeded)
	}
	return c.ctx, c.connection, nil
}
func (c *propertyConnection) Close() error {
	if c == nil || c.owner == nil {
		return nil
	}
	p := c.owner
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.identity != p {
		return nil
	}
	return p.retireLocked(c)
}
