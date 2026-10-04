// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

// Config comes from the future root-owned service configuration, not a request.
// Creating a Server does not install, enable, or start a host service.
type Config struct {
	Socket        string
	ManagementUID uint32
	SocketGID     uint32
	Limits        Limits
	Timeout       time.Duration
	MaxClients    int
}

func (c Config) validate() error {
	if !filepath.IsAbs(c.Socket) || filepath.Clean(c.Socket) != c.Socket || len(c.Socket) > 103 || filepath.Base(c.Socket) == "." || c.ManagementUID == 0 || c.ManagementUID == ^uint32(0) || c.SocketGID == 0 || c.SocketGID == ^uint32(0) || c.Timeout <= 0 || c.Timeout > 5*time.Minute || c.MaxClients < 1 || c.MaxClients > 64 {
		return ErrConfig
	}
	return c.Limits.validate()
}

type Collector interface {
	Collect(context.Context) inventory.Snapshot
}
type Server struct {
	config     Config
	collect    Collector
	peer       func(*net.UnixConn) (uint32, error)
	collection sync.Mutex
}

func NewServer(c Config, source Collector) (*Server, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	if os.Geteuid() != 0 || source == nil {
		return nil, ErrConfig
	}
	if err := safeDirectory(c); err != nil {
		return nil, err
	}
	return &Server{config: c, collect: source, peer: peerUID}, nil
}

// Serve owns only a newly created socket. An existing entry is never unlinked or
// adopted. Shutdown closes admitted connections, waits for collection cancellation,
// then removes the socket only if its inode still belongs to this listener.
func (s *Server) Serve(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := safeDirectory(s.config); err != nil {
		return err
	}
	if _, err := os.Lstat(s.config.Socket); !os.IsNotExist(err) {
		return ErrConfig
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: s.config.Socket, Net: "unix"})
	if err != nil {
		return ErrConfig
	}
	listener.SetUnlinkOnClose(false)
	owned, err := os.Lstat(s.config.Socket)
	if err != nil {
		_ = listener.Close()
		return ErrConfig
	}
	defer func() {
		_ = listener.Close()
		current, err := os.Lstat(s.config.Socket)
		if err == nil && os.SameFile(owned, current) {
			_ = os.Remove(s.config.Socket)
		}
	}()
	if err := os.Chown(s.config.Socket, 0, int(s.config.SocketGID)); err != nil {
		return ErrConfig
	}
	if err := os.Chmod(s.config.Socket, 0660); err != nil {
		return ErrConfig
	}
	if err := safeSocket(s.config); err != nil {
		return err
	}
	serveCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(serveCtx, func() { _ = listener.Close() })
	defer stop()
	var workers sync.WaitGroup
	defer workers.Wait()
	// This defer runs before Wait, including on an unexpected accept failure.
	defer cancel()
	slots := make(chan struct{}, s.config.MaxClients)
	for {
		conn, err := listener.AcceptUnix()
		if err != nil {
			if serveCtx.Err() != nil {
				return serveCtx.Err()
			}
			return ErrProtocol
		}
		select {
		case slots <- struct{}{}:
			workers.Add(1)
			go func() { defer workers.Done(); defer func() { <-slots }(); s.handle(serveCtx, conn) }()
		default:
			_ = conn.Close()
		}
	}
}
func (s *Server) handle(parent context.Context, c *net.UnixConn) {
	defer c.Close()
	ctx, cancel := context.WithTimeout(parent, s.config.Timeout)
	defer cancel()
	deadline, _ := ctx.Deadline()
	if c.SetDeadline(deadline) != nil {
		return
	}
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	uid, err := s.peer(c)
	if err != nil || uid != s.config.ManagementUID {
		return
	}
	id, err := readRequest(c, s.config.Limits)
	if err != nil {
		return
	}
	// One request per connection; client closes its write half. No implicit
	// queue or second operation can accompany an authorized inventory request.
	var extra [1]byte
	if n, err := c.Read(extra[:]); n != 0 || err != io.EOF {
		return
	}
	if !s.collection.TryLock() {
		return
	}
	defer s.collection.Unlock()
	if ctx.Err() != nil {
		return
	}
	observed := s.collect.Collect(ctx)
	if ctx.Err() != nil {
		return
	}
	_ = writeSnapshot(c, id, observed, s.config.Limits)
}

type Client struct{ config Config }

func NewClient(c Config) (*Client, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	if uint32(os.Getuid()) != c.ManagementUID {
		return nil, ErrPermission
	}
	if err := safeSocket(c); err != nil {
		return nil, err
	}
	return &Client{config: c}, nil
}
func (c *Client) Collect(parent context.Context, id string) (inventory.Snapshot, error) {
	if !validID(id) {
		return inventory.Snapshot{}, ErrProtocol
	}
	if err := safeSocket(c.config); err != nil {
		return inventory.Snapshot{}, err
	}
	ctx, cancel := context.WithTimeout(parent, c.config.Timeout)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", c.config.Socket)
	if err != nil {
		return inventory.Snapshot{}, ErrProtocol
	}
	defer conn.Close()
	u, ok := conn.(*net.UnixConn)
	if !ok {
		return inventory.Snapshot{}, ErrProtocol
	}
	deadline, _ := ctx.Deadline()
	if err := u.SetDeadline(deadline); err != nil {
		return inventory.Snapshot{}, ErrProtocol
	}
	stop := context.AfterFunc(ctx, func() { _ = u.Close() })
	defer stop()
	uid, err := peerUID(u)
	if err != nil || uid != 0 {
		return inventory.Snapshot{}, ErrPermission
	}
	if err := writeRequest(u, id, c.config.Limits); err != nil {
		return inventory.Snapshot{}, err
	}
	if err := u.CloseWrite(); err != nil {
		return inventory.Snapshot{}, ErrProtocol
	}
	s, err := readSnapshot(u, id, c.config.Limits)
	if ctx.Err() != nil {
		return inventory.Snapshot{}, ctx.Err()
	}
	return s, err
}
