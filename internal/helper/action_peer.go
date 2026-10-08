// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"errors"
	"net"
	"os"
	"sync"
	"time"
)

// Private, unregistered connection prerequisite. It proves only kernel UID and
// correlation on this original socket, never application identity or permission.
// No factory dials, listens, installs a service or dispatches an operation.
type actionPeer struct {
	mu                            sync.Mutex
	connection                    *net.UnixConn
	lifetime                      context.Context
	cancel                        context.CancelFunc
	stop                          func() bool
	localUID, remoteUID           uint32
	serviceUID                    uint32
	requestID, applicationContext string
	negotiated, retired           bool
}
type actionIdentity struct {
	uid                           uint32
	requestID, applicationContext string
}

// Ownership transfers on entry, including every failure. The caller retains the
// existing trusted listener/directory/socket policy and bounded admission gates.
func ownedActionPeer(parent context.Context, c *net.UnixConn, uid uint32, timeout time.Duration, rootSide bool) (p *actionPeer, err error) {
	defer func() {
		if err != nil && c != nil {
			_ = c.Close()
		}
	}()
	if parent == nil || c == nil || uid == 0 || uid == ^uint32(0) || timeout <= 0 || timeout > 5*time.Minute {
		return nil, ErrConfig
	}
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	local, remote := uid, uint32(0)
	if rootSide {
		local, remote = 0, uid
	}
	if uint32(os.Getuid()) != local || uint32(os.Geteuid()) != local {
		return nil, ErrPermission
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	lifetime, cancel := context.WithTimeout(parent, timeout)
	deadline, _ := lifetime.Deadline()
	if err := c.SetDeadline(deadline); err != nil {
		cancel()
		return nil, ErrPermission
	}
	p = &actionPeer{connection: c, lifetime: lifetime, cancel: cancel, localUID: local, remoteUID: remote, serviceUID: uid}
	p.stop = context.AfterFunc(lifetime, func() { _ = c.Close() })
	if err := p.check(false); err != nil {
		return nil, err
	}
	return p, nil
}
func acceptActionPeer(parent context.Context, c *net.UnixConn, uid uint32, timeout time.Duration) (*actionPeer, error) {
	p, err := ownedActionPeer(parent, c, uid, timeout, true)
	if err != nil {
		return nil, err
	}
	h, err := readActionNegotiation(c)
	if err != nil {
		_ = p.Close()
		return nil, err
	}
	if err := p.check(true); err != nil {
		return nil, err
	}
	if err := writeActionReady(c, h); err != nil {
		_ = p.Close()
		return nil, err
	}
	p.requestID, p.applicationContext, p.negotiated = h.ID, h.Context, true
	if err := p.checkLive(); err != nil {
		return nil, err
	}
	return p, nil
}
func connectActionPeer(parent context.Context, c *net.UnixConn, uid uint32, id, applicationContext string, timeout time.Duration) (*actionPeer, error) {
	if !validID(id) || !validApplicationContext(applicationContext) {
		if c != nil {
			_ = c.Close()
		}
		return nil, ErrProtocol
	}
	p, err := ownedActionPeer(parent, c, uid, timeout, false)
	if err != nil {
		return nil, err
	}
	if err := WriteActionNegotiation(c, id, applicationContext); err != nil {
		_ = p.Close()
		return nil, err
	}
	if err := p.check(false); err != nil {
		return nil, err
	}
	if err := ReadActionNegotiation(c, id, applicationContext); err != nil {
		_ = p.Close()
		return nil, err
	}
	p.requestID, p.applicationContext, p.negotiated = id, applicationContext, true
	if err := p.checkLive(); err != nil {
		return nil, err
	}
	return p, nil
}

// Check the actual original descriptor every time. Peer credentials may remain
// cached after disconnection, so a post-negotiation quiet check also observes
// EOF/extra data without consuming bytes or inventing an operation dispatcher.
func (p *actionPeer) check(quiet bool) error {
	if p == nil {
		return ErrPermission
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.checkLocked(quiet)
}
func (p *actionPeer) checkLocked(quiet bool) error {
	fail := func(err error) error { _ = p.retireLocked(); return err }
	if p.retired || p.connection == nil || p.lifetime == nil || p.cancel == nil || !nativeSupported {
		return fail(ErrPermission)
	}
	if err := p.lifetime.Err(); err != nil {
		return fail(err)
	}
	deadline, ok := p.lifetime.Deadline()
	if !ok || !time.Now().Before(deadline) {
		return fail(context.DeadlineExceeded)
	}
	if uint32(os.Getuid()) != p.localUID || uint32(os.Geteuid()) != p.localUID {
		return fail(ErrPermission)
	}
	uid, err := peerUID(p.connection)
	if err != nil || uid != p.remoteUID {
		return fail(ErrPermission)
	}
	if quiet {
		if err := actionPeerQuiet(p.connection); err != nil {
			return fail(err)
		}
	}
	if err := p.lifetime.Err(); err != nil {
		return fail(err)
	}
	return nil
}
func (p *actionPeer) checkLive() error {
	_, err := p.identity()
	return err
}
func (p *actionPeer) identity() (actionIdentity, error) {
	if p == nil {
		return actionIdentity{}, ErrPermission
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.negotiated || !validID(p.requestID) || !validApplicationContext(p.applicationContext) {
		_ = p.retireLocked()
		return actionIdentity{}, ErrPermission
	}
	if err := p.checkLocked(true); err != nil {
		return actionIdentity{}, err
	}
	return actionIdentity{p.serviceUID, p.requestID, p.applicationContext}, nil
}
func (p *actionPeer) retireLocked() error {
	if p.retired {
		return nil
	}
	p.retired = true
	if p.stop != nil {
		p.stop()
	}
	if p.cancel != nil {
		p.cancel()
	}
	if p.connection != nil {
		if err := p.connection.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			return err
		}
	}
	return nil
}
func (p *actionPeer) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.retireLocked()
}
