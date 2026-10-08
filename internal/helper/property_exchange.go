// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// Private, unregistered owner of exactly one typed exchange. Parsed identities
// and replies are snapshots/assertions, never an admission or launch capability.
type propertyPeer struct {
	mu      sync.Mutex
	peer    *actionPeer
	root    bool
	phase   uint8
	request propertyRequest
}

const (
	propertyAwait uint8 = 1
	propertyReply uint8 = 2
	propertyDone  uint8 = 3
)

func acceptPropertyPeer(ctx context.Context, c *net.UnixConn, uid uint32, timeout time.Duration) (*propertyPeer, error) {
	p, err := ownedActionPeer(ctx, c, uid, timeout, true)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*propertyPeer, error) { _ = p.Close(); return nil, err }
	h, err := readActionNegotiation(c)
	if err != nil {
		return fail(err)
	}
	// No request may precede ready. After ready a legitimate request may already
	// be queued, so only the expected typed phase consumes it, with fixed bounds.
	if err := p.check(true); err != nil {
		return fail(err)
	}
	p.mu.Lock()
	p.requestID, p.applicationContext, p.negotiated = h.ID, h.Context, true
	p.mu.Unlock()
	if err := writeActionReady(c, h); err != nil {
		return fail(err)
	}
	if _, err := propertyIdentity(p, false); err != nil {
		return fail(err)
	}
	return &propertyPeer{peer: p, root: true, phase: propertyAwait}, nil
}
func connectPropertyPeer(ctx context.Context, c *net.UnixConn, uid uint32, id, scope string, timeout time.Duration) (*propertyPeer, error) {
	p, err := connectActionPeer(ctx, c, uid, id, scope, timeout)
	if err != nil {
		return nil, err
	}
	return &propertyPeer{peer: p, phase: propertyAwait}, nil
}
func propertyIdentity(p *actionPeer, quiet bool) (actionIdentity, error) {
	if p == nil {
		return actionIdentity{}, ErrPermission
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.negotiated || !validID(p.requestID) || !validApplicationContext(p.applicationContext) {
		_ = p.retireLocked()
		return actionIdentity{}, ErrPermission
	}
	if err := p.checkLocked(quiet); err != nil {
		return actionIdentity{}, err
	}
	return actionIdentity{p.serviceUID, p.requestID, p.applicationContext}, nil
}

// Returns holding mu only on success. Overlap retires the original connection
// without waiting for the operation's mutex, interrupting any blocked socket I/O.
func (x *propertyPeer) begin(root bool, phase uint8, quiet bool) error {
	if x == nil || x.peer == nil {
		return ErrPermission
	}
	if !x.mu.TryLock() {
		_ = x.Close()
		return ErrProtocol
	}
	if x.root != root || x.phase != phase {
		x.mu.Unlock()
		_ = x.Close()
		return ErrProtocol
	}
	if _, err := propertyIdentity(x.peer, quiet); err != nil {
		x.mu.Unlock()
		_ = x.Close()
		return err
	}
	return nil
}
func (x *propertyPeer) receive() (propertyRequest, actionIdentity, error) {
	if err := x.begin(true, propertyAwait, false); err != nil {
		return propertyRequest{}, actionIdentity{}, err
	}
	defer x.mu.Unlock()
	x.phase = propertyReply
	req, err := readPropertyRequest(x.peer.connection, x.peer.requestID)
	if err != nil {
		_ = x.Close()
		return propertyRequest{}, actionIdentity{}, err
	}
	actor, err := propertyIdentity(x.peer, true)
	if err != nil {
		_ = x.Close()
		return propertyRequest{}, actionIdentity{}, err
	}
	x.request = req
	return req, actor, nil
}
func (x *propertyPeer) terminalRead() error {
	// EOF is the expected terminal boundary. It proves complete framing, not a
	// live root process or transaction outcome. Never accept extra terminal bytes.
	var b [1]byte
	n, err := x.peer.connection.Read(b[:])
	if n != 0 {
		return ErrProtocol
	}
	if err != io.EOF {
		if err == nil {
			return ErrProtocol
		}
		return err
	}
	_, err = propertyIdentity(x.peer, false)
	return err
}
func (x *propertyPeer) review(pool string, intent storage.Intent) (storage.Review, error) {
	if err := x.begin(false, propertyAwait, true); err != nil {
		return storage.Review{}, err
	}
	defer x.mu.Unlock()
	defer x.Close()
	x.phase = propertyDone
	if err := WritePropertyReview(x.peer.connection, x.peer.requestID, pool, intent); err != nil {
		return storage.Review{}, err
	}
	if _, err := propertyIdentity(x.peer, false); err != nil {
		return storage.Review{}, err
	}
	r, err := ReadPropertyReview(x.peer.connection, x.peer.requestID, pool, intent)
	if err != nil {
		if _, ok := err.(*PropertyRefusalError); !ok {
			return storage.Review{}, err
		}
	}
	if end := x.terminalRead(); end != nil {
		return storage.Review{}, end
	}
	return r, err
}
func (x *propertyPeer) submit(pool, plan, submission string) (storage.Job, bool, error) {
	if err := x.begin(false, propertyAwait, true); err != nil {
		return storage.Job{}, false, err
	}
	defer x.mu.Unlock()
	defer x.Close()
	x.phase = propertyDone
	if err := WritePropertySubmission(x.peer.connection, x.peer.requestID, pool, plan, submission); err != nil {
		return storage.Job{}, false, err
	}
	if _, err := propertyIdentity(x.peer, false); err != nil {
		return storage.Job{}, false, err
	}
	j, d, err := ReadPropertySubmission(x.peer.connection, x.peer.requestID, pool, plan, submission)
	if err != nil {
		if _, ok := err.(*PropertyRefusalError); !ok {
			return storage.Job{}, false, err
		}
	}
	if end := x.terminalRead(); end != nil {
		return storage.Job{}, false, end
	}
	return j, d, err
}
func (x *propertyPeer) replyReview(r storage.Review) error {
	if err := x.begin(true, propertyReply, true); err != nil {
		return err
	}
	defer x.mu.Unlock()
	defer x.Close()
	x.phase = propertyDone
	if x.request.Operation != "property-review" || r.PoolGUID != x.request.PoolGUID || r.Intent != x.request.Intent {
		return ErrProtocol
	}
	if err := writePropertyReview(x.peer.connection, x.peer.requestID, r); err != nil {
		return err
	}
	_, err := propertyIdentity(x.peer, true)
	return err
}
func (x *propertyPeer) replySubmission(j storage.Job, duplicate bool) error {
	if err := x.begin(true, propertyReply, true); err != nil {
		return err
	}
	defer x.mu.Unlock()
	defer x.Close()
	x.phase = propertyDone
	if x.request.Operation != "property-submit" || j.Intent.PoolGUID != x.request.PoolGUID || j.Intent.PlanID != x.request.PlanID || j.Intent.SubmissionID != x.request.SubmissionID {
		return ErrProtocol
	}
	if err := writePropertySubmission(x.peer.connection, x.peer.requestID, j, duplicate); err != nil {
		return err
	}
	_, err := propertyIdentity(x.peer, true)
	return err
}
func (x *propertyPeer) refuse(code string) error {
	if err := x.begin(true, propertyReply, true); err != nil {
		return err
	}
	defer x.mu.Unlock()
	defer x.Close()
	x.phase = propertyDone
	if x.request.Operation != "property-review" && x.request.Operation != "property-submit" {
		return ErrProtocol
	}
	if err := writePropertyRefusal(x.peer.connection, x.peer.requestID, code); err != nil {
		return err
	}
	_, err := propertyIdentity(x.peer, true)
	return err
}
func (x *propertyPeer) Close() error {
	if x == nil || x.peer == nil {
		return nil
	}
	return x.peer.Close()
}
