// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func connectionAdmission(t *testing.T, ctx context.Context, limit int, timeout time.Duration) *propertyConnections {
	t.Helper()
	p, err := newPropertyConnections(ctx, limit, timeout)
	if err != nil || p == nil {
		t.Fatal("admission factory", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	return p
}
func admitConnection(t *testing.T, p *propertyConnections) (*net.UnixConn, *propertyConnection) {
	t.Helper()
	client, server := connected(t)
	c, err := p.admit(server)
	if err != nil || c == nil {
		t.Fatal("admit", err)
	}
	return client, c
}
func connectionEOF(t *testing.T, c *net.UnixConn) {
	t.Helper()
	var b [1]byte
	if n, err := c.Read(b[:]); n != 0 || err != io.EOF {
		t.Fatalf("denied connection emitted bytes or remained open: n=%d err=%v", n, err)
	}
}
func connectionDone(t *testing.T, c *propertyConnection) {
	t.Helper()
	select {
	case <-c.done:
	case <-time.After(time.Second):
		t.Fatal("owned connection not retired")
	}
	if ctx, conn, err := c.original(); ctx != nil || conn != nil || err == nil {
		t.Fatal("retired connection published")
	}
}
func TestPropertyConnectionsConfiguration(t *testing.T) {
	for _, name := range []string{"nil-context", "cancelled", "zero-limit", "negative-limit", "large-limit", "zero-timeout", "negative-timeout", "large-timeout"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			limit := 1
			timeout := time.Second
			switch name {
			case "nil-context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "zero-limit":
				limit = 0
			case "negative-limit":
				limit = -1
			case "large-limit":
				limit = 65
			case "zero-timeout":
				timeout = 0
			case "negative-timeout":
				timeout = -1
			case "large-timeout":
				timeout = 5*time.Minute + 1
			}
			if p, err := newPropertyConnections(ctx, limit, timeout); p != nil || err == nil {
				t.Fatal("unsafe admission configuration")
			}
		})
	}
	for _, limit := range []int{1, 64} {
		p := connectionAdmission(t, context.Background(), limit, 5*time.Minute)
		if p.Close() != nil {
			t.Fatal("close")
		}
	}
}
func TestPropertyConnectionsOriginalAndUnissued(t *testing.T) {
	p := connectionAdmission(t, context.Background(), 1, time.Second)
	client, c := admitConnection(t, p)
	ctx, conn, err := c.original()
	if err != nil || ctx != c.ctx || conn != c.connection {
		t.Fatal("original binding replaced")
	}
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > time.Second {
		t.Fatal("unbounded lifetime")
	}
	// Admission consumes no bytes. Only the later kernel-authenticated exchange may
	// read them; sending before that future phase is not authorization.
	if _, err := client.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	var b [1]byte
	if n, err := conn.Read(b[:]); n != 1 || err != nil || b[0] != 'x' {
		t.Fatal("admission consumed input")
	}
	var decoded propertyConnection
	if json.Unmarshal([]byte(`{}`), &decoded) != nil {
		t.Fatal("fixture")
	}
	copy := *c
	for _, fake := range []*propertyConnection{nil, {}, &decoded, &copy, {owner: p, connection: conn, ctx: ctx}} {
		if ctx, conn, err := fake.original(); ctx != nil || conn != nil || err == nil {
			t.Fatal("reconstructed binding accepted")
		}
		if fake.Close() != nil {
			t.Fatal("unissued close")
		}
	}
	if _, _, err := c.original(); err != nil {
		t.Fatal("forged close retired original")
	}
	if c.Close() != nil || c.Close() != nil {
		t.Fatal("idempotent close")
	}
	connectionDone(t, c)
	connectionEOF(t, client)
}
func TestPropertyConnectionsCapacityAndFailureRecovery(t *testing.T) {
	p := connectionAdmission(t, context.Background(), 2, time.Second)
	client, a := admitConnection(t, p)
	other, b := admitConnection(t, p)
	denied, raw := connected(t)
	if c, err := p.admit(raw); c != nil || !errors.Is(err, ErrLimit) {
		t.Fatal("capacity queued or bypassed", err)
	}
	connectionEOF(t, denied)
	if a.Close() != nil {
		t.Fatal("close")
	}
	connectionDone(t, a)
	connectionEOF(t, client)
	next, c := admitConnection(t, p)
	// Releasing one slot must not interrupt an unrelated admitted socket.
	if _, err := other.Write([]byte("y")); err != nil {
		t.Fatal(err)
	}
	var buf [1]byte
	if n, err := b.connection.Read(buf[:]); n != 1 || err != nil || buf[0] != 'y' {
		t.Fatal("unrelated connection retired")
	}
	if c.Close() != nil {
		t.Fatal("close")
	}
	connectionEOF(t, next)
	closed, raw := connected(t)
	_ = raw.Close()
	if c, err := p.admit(raw); c != nil || err == nil {
		t.Fatal("closed descriptor admitted")
	}
	connectionEOF(t, closed)
	if c, err := p.admit(nil); c != nil || err == nil {
		t.Fatal("nil descriptor admitted")
	}
	retry, d := admitConnection(t, p)
	_ = d.Close()
	connectionEOF(t, retry)
}
func TestPropertyConnectionsDuplicateAndMissingOwner(t *testing.T) {
	p := connectionAdmission(t, context.Background(), 1, time.Second)
	client, c := admitConnection(t, p)
	if dup, err := p.admit(c.connection); dup != nil || !errors.Is(err, ErrProtocol) {
		t.Fatal("duplicate descriptor queued", err)
	}
	connectionDone(t, c)
	connectionEOF(t, client)
	next, original := admitConnection(t, p)
	_ = original.Close()
	connectionEOF(t, next)
	var decoded propertyConnections
	if json.Unmarshal([]byte(`{}`), &decoded) != nil {
		t.Fatal("fixture")
	}
	forged := &propertyConnections{ctx: p.ctx, cancel: p.cancel, limit: p.limit, timeout: p.timeout, owned: p.owned}
	for _, owner := range []*propertyConnections{nil, {}, &decoded, forged} {
		client, raw := connected(t)
		if c, err := owner.admit(raw); c != nil || err == nil {
			t.Fatal("unissued owner admitted")
		}
		connectionEOF(t, client)
		if owner.Close() != nil {
			t.Fatal("unissued close")
		}
	}
}

func TestPropertyConnectionsCancellationAtAdmission(t *testing.T) {
	for range 32 {
		ctx, cancel := context.WithCancel(context.Background())
		p := connectionAdmission(t, ctx, 1, time.Second)
		client, raw := connected(t)
		start := make(chan struct{})
		result := make(chan *propertyConnection, 1)
		var wg sync.WaitGroup
		wg.Go(func() {
			<-start
			c, err := p.admit(raw)
			if (c == nil) == (err == nil) {
				t.Error("partial admission result")
			}
			result <- c
		})
		wg.Go(func() { <-start; cancel() })
		close(start)
		c := <-result
		cancel()
		wg.Wait()
		if p.Close() != nil {
			t.Fatal("shutdown")
		}
		if c != nil {
			connectionDone(t, c)
		}
		connectionEOF(t, client)
		p.mu.Lock()
		n := len(p.owned)
		p.mu.Unlock()
		if n != 0 {
			t.Fatal("cancelled admission retained a slot")
		}
	}
}
func TestPropertyConnectionsShutdownInterruptsIO(t *testing.T) {
	for _, name := range []string{"explicit", "parent-cancellation"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p := connectionAdmission(t, ctx, 2, time.Second)
			client, a := admitConnection(t, p)
			other, b := admitConnection(t, p)
			reads := make(chan error, 2)
			for _, c := range []*propertyConnection{a, b} {
				go func() { var buf [1]byte; _, err := c.connection.Read(buf[:]); reads <- err }()
			}
			if name == "explicit" {
				if p.Close() != nil || p.Close() != nil {
					t.Fatal("close")
				}
			} else {
				cancel()
			}
			for range 2 {
				select {
				case err := <-reads:
					if err == nil {
						t.Fatal("blocked IO survived")
					}
				case <-time.After(time.Second):
					t.Fatal("IO not interrupted")
				}
			}
			connectionDone(t, a)
			connectionDone(t, b)
			connectionEOF(t, client)
			connectionEOF(t, other)
			denied, raw := connected(t)
			if c, err := p.admit(raw); c != nil || err == nil {
				t.Fatal("shutdown admission")
			}
			connectionEOF(t, denied)
			p.mu.Lock()
			n := len(p.owned)
			p.mu.Unlock()
			if n != 0 {
				t.Fatal("retained admission")
			}
		})
	}
}
func TestPropertyConnectionsOriginalDeadline(t *testing.T) {
	for _, parent := range []bool{false, true} {
		t.Run(map[bool]string{false: "configured", true: "earlier-parent"}[parent], func(t *testing.T) {
			ctx := context.Background()
			timeout := 120 * time.Millisecond
			if parent {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
				timeout = time.Second
			}
			p := connectionAdmission(t, ctx, 1, timeout)
			client, c := admitConnection(t, p)
			deadline, _ := c.ctx.Deadline()
			for range 3 {
				if _, err := client.Write([]byte("x")); err != nil {
					t.Fatal(err)
				}
				var buf [1]byte
				if _, err := c.connection.Read(buf[:]); err != nil {
					t.Fatal(err)
				}
				current, _, err := c.original()
				if err != nil {
					t.Fatal(err)
				}
				next, _ := current.Deadline()
				if !next.Equal(deadline) {
					t.Fatal("traffic renewed original deadline")
				}
			}
			var buf [1]byte
			if n, err := c.connection.Read(buf[:]); n != 0 || err == nil {
				t.Fatal("idle connection outlived deadline")
			}
			connectionDone(t, c)
			connectionEOF(t, client)
			if !errors.Is(c.ctx.Err(), context.DeadlineExceeded) {
				t.Fatal("deadline not retained", c.ctx.Err())
			}
			next, raw := connected(t)
			if parent {
				if n, err := p.admit(raw); n != nil || err == nil {
					t.Fatal("expired parent renewed")
				}
				connectionEOF(t, next)
			} else {
				n, err := p.admit(raw)
				if err != nil {
					t.Fatal("expired slot retained", err)
				}
				_ = n.Close()
				connectionEOF(t, next)
			}
		})
	}
}
func TestPropertyConnectionsConcurrentLastSlot(t *testing.T) {
	p := connectionAdmission(t, context.Background(), 1, time.Second)
	const count = 16
	clients := make([]*net.UnixConn, count)
	raw := make([]*net.UnixConn, count)
	for i := range count {
		clients[i], raw[i] = connected(t)
	}
	type result struct {
		c   *propertyConnection
		err error
	}
	results := make(chan result, count)
	start := make(chan struct{})
	for i := range count {
		go func() { <-start; c, err := p.admit(raw[i]); results <- result{c, err} }()
	}
	close(start)
	var admitted *propertyConnection
	allowed := 0
	for range count {
		r := <-results
		if r.err == nil && r.c != nil {
			allowed++
			admitted = r.c
		} else if r.c != nil || !errors.Is(r.err, ErrLimit) {
			t.Fatal("unexpected admission", r.err)
		}
	}
	if allowed != 1 {
		t.Fatal("last slot raced", allowed)
	}
	// Concurrent releases/shutdown must complete without closing a slot twice.
	var wg sync.WaitGroup
	for range count {
		wg.Go(func() { _ = admitted.Close(); _ = p.Close() })
	}
	wg.Wait()
	connectionDone(t, admitted)
	for _, c := range clients {
		connectionEOF(t, c)
	}
}
