// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

type collectFunc func(context.Context) inventory.Snapshot

func (f collectFunc) Collect(ctx context.Context) inventory.Snapshot { return f(ctx) }

func connected(t *testing.T) (*net.UnixConn, *net.UnixConn) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "horvium-wire-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(dir, "s"), Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	client, err := net.DialUnix("unix", nil, listener.Addr().(*net.UnixAddr))
	if err != nil {
		t.Fatal(err)
	}
	server, err := listener.AcceptUnix()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })
	_ = client.SetDeadline(time.Now().Add(2 * time.Second))
	return client, server
}
func testServer(f Collector, peer func(*net.UnixConn) (uint32, error)) *Server {
	return &Server{config: Config{ManagementUID: 42, Limits: DefaultLimits(), Timeout: time.Second, MaxClients: 2}, collect: f, peer: peer}
}
func TestServerAuthenticatesBeforeReading(t *testing.T) {
	for _, tc := range []struct {
		name string
		uid  uint32
		err  error
	}{{"different uid", 43, nil}, {"root", 0, nil}, {"credential ABI", 42, ErrPermission}} {
		t.Run(tc.name, func(t *testing.T) {
			client, server := connected(t)
			var calls atomic.Int32
			s := testServer(collectFunc(func(context.Context) inventory.Snapshot { calls.Add(1); return observation() }), func(*net.UnixConn) (uint32, error) { return tc.uid, tc.err })
			done := make(chan struct{})
			go func() { s.handle(context.Background(), server); close(done) }()
			// No request is sent: denial must precede reading any bytes.
			select {
			case <-done:
			case <-time.After(300 * time.Millisecond):
				t.Fatal("denial waited for attacker input")
			}
			if calls.Load() != 0 {
				t.Fatal("unauthorized native collection")
			}
			b := make([]byte, 1)
			if n, _ := client.Read(b); n != 0 {
				t.Fatal("inventory leaked")
			}
		})
	}
}
func TestServerInventoryOnly(t *testing.T) {
	for _, payload := range []string{
		`{"version":1,"request_id":"r","operation":"inventory"}`,
		`{"version":1,"request_id":"r","operation":"destroy"}`,
		`{"version":1,"request_id":"r","operation":"inventory","uid":42}`,
		`{"version":1,"request_id":"r","operation":"inventory","path":"/dev/ada0"}`,
	} {
		t.Run(payload, func(t *testing.T) {
			client, server := connected(t)
			var calls atomic.Int32
			s := testServer(collectFunc(func(context.Context) inventory.Snapshot { calls.Add(1); return observation() }), func(*net.UnixConn) (uint32, error) { return 42, nil })
			go s.handle(context.Background(), server)
			if _, err := client.Write(frame(payload)); err != nil {
				t.Fatal(err)
			}
			_ = client.CloseWrite()
			got, err := readSnapshot(client, "r", DefaultLimits())
			if payload == `{"version":1,"request_id":"r","operation":"inventory"}` {
				if err != nil || calls.Load() != 1 || got.Host.Status != inventory.Partial {
					t.Fatalf("allowed: calls=%d err=%v", calls.Load(), err)
				}
			} else if err == nil || calls.Load() != 0 {
				t.Fatal("invalid request reached collector")
			}
		})
	}
}
func TestServerRejectsExtraRequestAndBoundsStalls(t *testing.T) {
	client, server := connected(t)
	var calls atomic.Int32
	s := testServer(collectFunc(func(context.Context) inventory.Snapshot { calls.Add(1); return observation() }), func(*net.UnixConn) (uint32, error) { return 42, nil })
	go s.handle(context.Background(), server)
	var b bytes.Buffer
	_ = writeRequest(&b, "r", DefaultLimits())
	_, _ = client.Write(append(b.Bytes(), byte(0)))
	_ = client.CloseWrite()
	if _, err := readSnapshot(client, "r", DefaultLimits()); err == nil || calls.Load() != 0 {
		t.Fatal("extra request data accepted")
	}
	client2, server2 := connected(t)
	idle := testServer(s.collect, s.peer)
	idle.config.Timeout = 50 * time.Millisecond
	done := make(chan struct{})
	go func() { idle.handle(context.Background(), server2); close(done) }()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("idle client unbounded")
	}
	if _, err := io.Copy(io.Discard, client2); err != nil && !errors.Is(err, net.ErrClosed) {
		t.Fatal(err)
	}
}

func TestCredentialVersionAndConfiguration(t *testing.T) {
	if uid, err := credentialUID(0, 42); err != nil || uid != 42 {
		t.Fatal("known credential ABI denied")
	}
	if _, err := credentialUID(1, 42); !errors.Is(err, ErrPermission) {
		t.Fatal("unknown ABI accepted")
	}
	valid := Config{Socket: "/var/run/horvium/helper.sock", ManagementUID: 42, SocketGID: 42, Limits: DefaultLimits(), Timeout: 50 * time.Second, MaxClients: 8}
	if err := valid.validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.ManagementUID = 0 }, func(c *Config) { c.SocketGID = 0 }, func(c *Config) { c.Socket = "relative" }, func(c *Config) { c.Socket = "/var/run/../helper.sock" }, func(c *Config) { c.Timeout = 0 }, func(c *Config) { c.MaxClients = 65 }, func(c *Config) { c.Limits.MaxBytes = 0 },
	} {
		c := valid
		change(&c)
		if !errors.Is(c.validate(), ErrConfig) {
			t.Fatal("unsafe configuration accepted")
		}
	}
	if !nativeSupported {
		if _, err := NewServer(valid, collectFunc(func(context.Context) inventory.Snapshot { return observation() })); !errors.Is(err, ErrUnsupported) {
			t.Fatal("development host masqueraded as FreeBSD")
		}
		if _, err := NewClient(valid); !errors.Is(err, ErrUnsupported) {
			t.Fatal("unsupported native client")
		}
	}
}

func TestConcurrentInventoryIsNotQueuedAndCancellationStopsWork(t *testing.T) {
	started := make(chan struct{})
	stopped := make(chan struct{})
	s := testServer(collectFunc(func(ctx context.Context) inventory.Snapshot {
		close(started)
		<-ctx.Done()
		close(stopped)
		return observation()
	}), func(*net.UnixConn) (uint32, error) { return 42, nil })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, server := connected(t)
	done := make(chan struct{})
	go func() { s.handle(ctx, server); close(done) }()
	if err := writeRequest(client, "r", DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	_ = client.CloseWrite()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("collection not started")
	}
	client2, server2 := connected(t)
	go s.handle(context.Background(), server2)
	if err := writeRequest(client2, "r", DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	_ = client2.CloseWrite()
	if _, err := readSnapshot(client2, "r", DefaultLimits()); err == nil {
		t.Fatal("second collection queued or published")
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("collection context not cancelled")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("connection did not close")
	}
}
