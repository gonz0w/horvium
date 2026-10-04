// SPDX-License-Identifier: BSD-3-Clause
package service

import (
	"net"
	"sync"
)

// Bound accepted connections before net/http allocates a serving goroutine.
// Capacity includes incomplete headers and idle keep-alive connections. At full
// capacity, new connections remain in the kernel's finite listener backlog;
// existing transport deadlines release capacity. No waiter is spawned per client.
type connectionListener struct {
	net.Listener
	slots     chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	closeErr  error
}

func boundedListener(inner net.Listener, capacity int) net.Listener {
	return &connectionListener{Listener: inner, slots: make(chan struct{}, capacity), done: make(chan struct{})}
}
func (l *connectionListener) Accept() (net.Conn, error) {
	select {
	case <-l.done:
		return nil, net.ErrClosed
	case l.slots <- struct{}{}:
	}
	conn, err := l.Listener.Accept()
	if err != nil {
		<-l.slots
		return nil, err
	}
	return &countedConnection{Conn: conn, release: func() { <-l.slots }}, nil
}
func (l *connectionListener) Close() error {
	l.closeOnce.Do(func() {
		close(l.done)
		l.closeErr = l.Listener.Close()
	})
	return l.closeErr
}

type countedConnection struct {
	net.Conn
	release   func()
	closeOnce sync.Once
	closeErr  error
}

func (c *countedConnection) Close() error {
	c.closeOnce.Do(func() {
		c.closeErr = c.Conn.Close()
		c.release()
	})
	return c.closeErr
}
