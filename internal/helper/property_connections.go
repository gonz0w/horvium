// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"net"
	"sync"
	"time"
)

// Private network admission only. No peer/session, storage or launch authority.
type propertyConnections struct {
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	stop    func() bool
	limit   int
	timeout time.Duration
	owned   map[*net.UnixConn]*propertyConnection
	closed  bool
}
type propertyConnection struct {
	owner      *propertyConnections
	connection *net.UnixConn
	ctx        context.Context
	cancel     context.CancelFunc
	stop       func() bool
	done       chan struct{}
}

func newPropertyConnections(context.Context, int, time.Duration) (*propertyConnections, error) {
	return nil, ErrConfig
}
func (p *propertyConnections) admit(c *net.UnixConn) (*propertyConnection, error) {
	if c != nil {
		_ = c.Close()
	}
	return nil, ErrPermission
}
func (p *propertyConnections) Close() error { return nil }
func (c *propertyConnection) original() (context.Context, *net.UnixConn, error) {
	return nil, nil, ErrPermission
}
func (c *propertyConnection) Close() error { return nil }
