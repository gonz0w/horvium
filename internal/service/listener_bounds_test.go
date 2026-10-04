// SPDX-License-Identifier: BSD-3-Clause
package service

import (
	"net"
	"testing"
	"time"
)

func TestListenerBoundsSlowConnectionsAndRecovers(t *testing.T) {
	inner := newListener(t)
	listener := boundedListener(inner, 2)
	t.Cleanup(func() { _ = listener.Close() })
	accepted := make(chan net.Conn, 3)
	failures := make(chan error, 1)
	go func() {
		for i := 0; i < 3; i++ {
			c, err := listener.Accept()
			if err != nil {
				failures <- err
				return
			}
			accepted <- c
		}
	}()
	var clients []net.Conn
	for i := 0; i < 3; i++ {
		c, err := net.DialTimeout("tcp", inner.Addr().String(), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		clients = append(clients, c)
		t.Cleanup(func() { _ = c.Close() })
	}
	receive := func() net.Conn {
		t.Helper()
		select {
		case c := <-accepted:
			t.Cleanup(func() { _ = c.Close() })
			return c
		case err := <-failures:
			t.Fatal(err)
		case <-time.After(time.Second):
			t.Fatal("listener did not recover")
		}
		return nil
	}
	first, second := receive(), receive()
	select {
	case c := <-accepted:
		_ = c.Close()
		t.Fatal("slow connections exceed admission bound")
	case err := <-failures:
		t.Fatal(err)
	case <-time.After(30 * time.Millisecond):
	}
	_ = first.Close()
	_ = first.Close() // idempotent release must not consume another slot
	third := receive()
	_ = second.Close()
	_ = third.Close()
}

func TestListenerCloseUnblocksCapacityWait(t *testing.T) {
	inner := newListener(t)
	listener := boundedListener(inner, 1)
	t.Cleanup(func() { _ = listener.Close() })
	client, err := net.DialTimeout("tcp", inner.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	active, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer active.Close()
	done := make(chan error, 1)
	go func() { _, err := listener.Accept(); done <- err }()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("closed listener accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("capacity waiter stranded on shutdown")
	}
}
