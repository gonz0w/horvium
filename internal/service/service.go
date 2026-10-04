// SPDX-License-Identifier: BSD-3-Clause
// Package service assembles the M1 foreground runtime. It installs no services.
package service

import (
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/console"
	"github.com/gonz0w/horvium/internal/eventlog"
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
	logCtx, stopLog := context.WithCancel(ctx)
	defer stopLog()
	events := eventlog.New(logCtx, os.Stderr)
	server, e := helper.NewServer(c.HelperConfig(), nativeEvents{source, events})
	if e != nil {
		return ErrRuntime
	}
	events.Record(eventlog.Event{Operation: "helper", ID: events.NextID(), Section: "inventory", Outcome: "started"})
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
	logCtx, stopLog := context.WithCancel(context.Background())
	defer stopLog()
	events := eventlog.New(logCtx, os.Stderr)
	manager, e := observation.NewObserved(c.ObservationConfig(), source, observed(events))
	if e != nil {
		return ErrRuntime
	}
	a := admin.NewAuth(digest)
	defer a.Close()
	handler, e := console.New(c, a, manager)
	if e != nil {
		return ErrRuntime
	}
	server, e := c.HTTPServer(requests(events, handler))
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
	events.Record(eventlog.Event{Operation: "management", ID: events.NextID(), Section: "inventory", Outcome: "started"})
	e = run(ctx, server, listener, manager)
	outcome, class := "stopped", ""
	if e != nil {
		outcome, class = "failed", "failed"
	}
	events.Record(eventlog.Event{Operation: "management", ID: events.NextID(), Section: "inventory", Outcome: outcome, ErrorClass: class})
	return e
}

// run owns the collector, listener and admitted HTTP work together. Unexpected
// collector termination must stop serving rather than leave a stranded runtime.
func run(ctx context.Context, server *http.Server, listener net.Listener, manager *observation.Manager) error {
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { e := manager.Run(child); done <- e; cancel() }()
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
	e := server.Serve(listener)
	cancel()
	<-stopped
	collectorErr := <-done
	if collectorErr != nil && !errors.Is(collectorErr, context.Canceled) {
		return ErrRuntime
	}
	if e != nil && !errors.Is(e, net.ErrClosed) && !errors.Is(e, http.ErrServerClosed) {
		return ErrRuntime
	}
	return nil
}
