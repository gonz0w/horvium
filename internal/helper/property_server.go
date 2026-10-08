// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"errors"
	"net"
	"os"
	"sync"
)

// Private prototype: socket/worker ownership only. Every valid property operation
// is denied; no backend, collector, service or runtime consumer is registered.
type propertyServer struct {
	mu        sync.Mutex
	identity  *propertyServer
	config    Config
	attempted bool
	active    map[*propertyConnection]struct{}
}

func newPropertyServer(c Config) (*propertyServer, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	if os.Getuid() != 0 || os.Geteuid() != 0 {
		return nil, ErrPermission
	}
	if err := safeDirectory(c); err != nil {
		return nil, err
	}
	s := &propertyServer{config: c, active: make(map[*propertyConnection]struct{})}
	s.identity = s
	return s, nil
}
func (s *propertyServer) serve(parent context.Context) (err error) {
	if s == nil || parent == nil {
		return ErrPermission
	}
	s.mu.Lock()
	if s.identity != s || s.active == nil {
		s.mu.Unlock()
		return ErrPermission
	}
	if s.attempted {
		s.mu.Unlock()
		return ErrProtocol
	}
	s.attempted = true
	s.mu.Unlock()
	if err := s.config.validate(); err != nil {
		return err
	}
	if !nativeSupported {
		return ErrUnsupported
	}
	if os.Getuid() != 0 || os.Geteuid() != 0 {
		return ErrPermission
	}
	if err := parent.Err(); err != nil {
		return err
	}
	if err := safeDirectory(s.config); err != nil {
		return err
	}
	if _, err := os.Lstat(s.config.Socket); !os.IsNotExist(err) {
		return ErrConfig
	}
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: s.config.Socket, Net: "unix"})
	if err != nil {
		return ErrConfig
	}
	l.SetUnlinkOnClose(false)
	var closeOnce sync.Once
	var closeErr error
	closeListener := func() error {
		closeOnce.Do(func() { closeErr = closePropertyListener(l) })
		return closeErr
	}
	// Arm descriptor retirement immediately. Missing/changed namespace evidence
	// can never authorize unlinking an unrelated entry.
	var owned os.FileInfo
	defer func() {
		err = errors.Join(err, closeListener())
		if owned != nil {
			err = errors.Join(err, s.removeOwnedSocket(owned))
		}
	}()
	owned, err = os.Lstat(s.config.Socket)
	if err != nil {
		return err
	}
	if owned.Mode()&os.ModeSocket == 0 {
		return ErrConfig
	}
	if err := os.Chown(s.config.Socket, 0, int(s.config.SocketGID)); err != nil {
		return err
	}
	if err := os.Chmod(s.config.Socket, 0660); err != nil {
		return err
	}
	if err := safeSocket(s.config); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	p, err := newPropertyConnections(ctx, s.config.MaxClients, s.config.Timeout)
	if err != nil {
		return err
	}
	var workers sync.WaitGroup
	failures := make(chan error, 1)
	stop := context.AfterFunc(ctx, func() { _ = closeListener(); _ = p.Close() })
	defer func() {
		cancel()
		stop()
		err = errors.Join(err, closeListener(), p.Close())
		workers.Wait()
		// A finishing worker may have observed a late descriptor-close failure.
		err = errors.Join(err, p.Close())
		select {
		case failure := <-failures:
			err = errors.Join(err, failure)
		default:
		}
	}()
	for {
		raw, e := l.AcceptUnix()
		if e != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return ErrProtocol
		}
		lease, e := p.admit(raw)
		if e != nil {
			if failure := propertyWorkerCloseFailure(e); failure != nil {
				return failure
			}
			continue
		}
		if e = s.reserve(lease); e != nil {
			if failure := lease.Close(); failure != nil {
				return failure
			}
			continue
		}
		workers.Add(1)
		go func(c *propertyConnection) {
			defer workers.Done()
			defer s.finish(c)
			defer c.Close()
			if failure := propertyWorkerCloseFailure(s.deny(c)); failure != nil {
				select {
				case failures <- failure:
				default:
				}
				cancel()
			}
		}(lease)
	}
}
func closePropertyListener(l *net.UnixListener) error {
	if l == nil {
		return nil
	}
	if err := l.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}
func (s *propertyServer) removeOwnedSocket(owned os.FileInfo) error {
	if err := safeDirectory(s.config); err != nil {
		return err
	}
	current, err := os.Lstat(s.config.Socket)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if current.Mode()&os.ModeSocket != 0 && os.SameFile(owned, current) {
		return os.Remove(s.config.Socket)
	}
	return nil
}
func (s *propertyServer) reserve(c *propertyConnection) error {
	if s == nil {
		return ErrPermission
	}
	if _, _, err := c.original(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.identity != s || s.active == nil || s.config.MaxClients < 1 || s.config.MaxClients > 64 {
		return ErrPermission
	}
	if _, ok := s.active[c]; ok {
		return ErrProtocol
	}
	if len(s.active) >= s.config.MaxClients {
		return ErrLimit
	}
	s.active[c] = struct{}{}
	return nil
}
func (s *propertyServer) finish(c *propertyConnection) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.identity == s {
		delete(s.active, c)
	}
}
func (s *propertyServer) deny(c *propertyConnection) (err error) {
	a, err := c.accept(s.config.ManagementUID)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, a.Close()) }()
	if _, _, err := a.receive(); err != nil {
		return err
	}
	return a.refuse("denied")
}

// No caller-supplied errors or credential hooks enter this prototype. Walk the
// bounded standard-library/owner error tree to retain the first real close error;
// ordinary read/write/protocol denial must not consume an unbounded error ledger.
func propertyWorkerCloseFailure(err error) error {
	if err == nil {
		return nil
	}
	if op, ok := err.(*net.OpError); ok && op.Op == "close" && !errors.Is(op, net.ErrClosed) {
		return op
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range joined.Unwrap() {
			if failure := propertyWorkerCloseFailure(child); failure != nil {
				return failure
			}
		}
	} else if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return propertyWorkerCloseFailure(wrapped.Unwrap())
	}
	return nil
}
