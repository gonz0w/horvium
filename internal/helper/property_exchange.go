// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
	"net"
	"sync"
	"time"
)

type propertyPeer struct {
	mu      sync.Mutex
	peer    *actionPeer
	root    bool
	phase   uint8
	request propertyRequest
}

func acceptPropertyPeer(ctx context.Context, c *net.UnixConn, uid uint32, timeout time.Duration) (*propertyPeer, error) {
	if c != nil {
		_ = c.Close()
	}
	return nil, ErrPermission
}
func connectPropertyPeer(ctx context.Context, c *net.UnixConn, uid uint32, id, scope string, timeout time.Duration) (*propertyPeer, error) {
	if c != nil {
		_ = c.Close()
	}
	return nil, ErrPermission
}
func (x *propertyPeer) receive() (propertyRequest, actionIdentity, error) {
	return propertyRequest{}, actionIdentity{}, ErrPermission
}
func (x *propertyPeer) review(pool string, intent storage.Intent) (storage.Review, error) {
	return storage.Review{}, ErrPermission
}
func (x *propertyPeer) submit(pool, plan, submission string) (storage.Job, bool, error) {
	return storage.Job{}, false, ErrPermission
}
func (x *propertyPeer) replyReview(r storage.Review) error                  { return ErrPermission }
func (x *propertyPeer) replySubmission(j storage.Job, duplicate bool) error { return ErrPermission }
func (x *propertyPeer) refuse(code string) error                            { return ErrPermission }
func (x *propertyPeer) Close() error {
	if x == nil || x.peer == nil {
		return nil
	}
	return x.peer.Close()
}
