// SPDX-License-Identifier: BSD-3-Clause
// Package service assembles the M1 foreground runtime. It installs no services.
package service

import (
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/console"
	"github.com/gonz0w/horvium/internal/helper"
	native "github.com/gonz0w/horvium/internal/host/freebsd"
	"github.com/gonz0w/horvium/internal/observation"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"
)

var ErrRuntime = errors.New("foreground runtime unavailable or unsafe")

func Helper(ctx context.Context, c admin.Config) error {
	if runtime.GOOS != "freebsd" || os.Getuid() != 0 || os.Geteuid() != 0 || c.Validate() != nil {
		return ErrRuntime
	}
	source, e := native.New(c.InventoryLimits())
	if e != nil {
		return ErrRuntime
	}
	server, e := helper.NewServer(c.HelperConfig(), source)
	if e != nil {
		return ErrRuntime
	}
	if e = server.Serve(ctx); e != nil && !errors.Is(e, context.Canceled) {
		return ErrRuntime
	}
	return nil
}
func Serve(ctx context.Context, c admin.Config) error {
	if runtime.GOOS != "freebsd" || os.Getuid() == 0 || uint32(os.Getuid()) != c.ManagementUID || os.Geteuid() != os.Getuid() || c.Validate() != nil {
		return ErrRuntime
	}
	digest, e := admin.LoadDigest(c)
	if e != nil {
		return ErrRuntime
	}
	source, e := helper.NewClient(c.HelperConfig())
	if e != nil {
		return ErrRuntime
	}
	manager, e := observation.New(c.ObservationConfig(), source)
	if e != nil {
		return ErrRuntime
	}
	a := admin.NewAuth(digest)
	defer a.Close()
	handler, e := console.New(c, a, manager)
	if e != nil {
		return ErrRuntime
	}
	server, e := c.HTTPServer(handler)
	if e != nil {
		return ErrRuntime
	}
	// net/http's default error logger can disclose request details; no raw errors.
	server.ErrorLog = log.New(io.Discard, "", 0)
	listener, e := net.Listen("tcp", c.Listen)
	if e != nil {
		return ErrRuntime
	}
	defer listener.Close()
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- manager.Run(child) }()
	defer func() { cancel(); <-done }()
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-child.Done()
		shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if server.Shutdown(shutdown) != nil {
			_ = server.Close()
		}
	}()
	e = server.Serve(listener)
	cancel()
	<-stopped
	if e != nil && !errors.Is(e, net.ErrClosed) && !errors.Is(e, http.ErrServerClosed) {
		return ErrRuntime
	}
	return nil
}
