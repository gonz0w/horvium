// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"net"
	"time"
)

type actionPeer struct{}
type actionIdentity struct {
	uid                           uint32
	requestID, applicationContext string
}

func acceptActionPeer(_ context.Context, c *net.UnixConn, _ uint32, _ time.Duration) (*actionPeer, error) {
	if c != nil {
		_ = c.Close()
	}
	return nil, ErrPermission
}
func connectActionPeer(_ context.Context, c *net.UnixConn, _ uint32, _, _ string, _ time.Duration) (*actionPeer, error) {
	if c != nil {
		_ = c.Close()
	}
	return nil, ErrPermission
}
func (*actionPeer) identity() (actionIdentity, error) { return actionIdentity{}, ErrPermission }
func (*actionPeer) checkLive() error                  { return ErrPermission }
func (*actionPeer) Close() error                      { return nil }
