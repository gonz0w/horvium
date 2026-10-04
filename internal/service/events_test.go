// SPDX-License-Identifier: BSD-3-Clause
package service

import (
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestClassificationNeverUsesRawValues(t *testing.T) {
	for _, path := range []string{"/secret-token?cookie=secret", "/api/datasets?filter=secret", "/assets/secret"} {
		r := httptest.NewRequest("GET", path, nil)
		section := requestSection(r)
		if section != "other" && section != "datasets" && section != "asset" {
			t.Fatal("raw path classified")
		}
	}
}

type stoppedSource struct{}

func (stoppedSource) Collect(ctx context.Context, id string) (inventory.Snapshot, error) {
	<-ctx.Done()
	return inventory.Snapshot{}, ctx.Err()
}
func TestForegroundCancellationClearsStateAndClosesListener(t *testing.T) {
	c := observation.DefaultConfig()
	m, e := observation.New(c, stoppedSource{})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	l := newListener(t)
	s := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })}
	done := make(chan error, 1)
	go func() { done <- run(ctx, s, l, m) }()
	for until := time.Now().Add(time.Second); !m.State().Collecting; {
		if time.Now().After(until) {
			t.Fatal("collector did not start")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown blocked")
	}
	if state := m.State(); state.Running || state.Collecting || state.CurrentID != "" {
		t.Fatal("volatile state survived")
	}
	if _, e := l.Accept(); e == nil {
		t.Fatal("listener survived")
	}
}
func TestCollectorTerminationStopsHTTP(t *testing.T) {
	m, _ := observation.New(observation.DefaultConfig(), stoppedSource{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = m.Run(ctx) // cancelled run does not reserve started
	// A running manager belongs to one foreground process. Reusing it must fail.
	active, stop := context.WithCancel(context.Background())
	defer stop()
	done := make(chan error, 1)
	go func() { done <- m.Run(active) }()
	for !m.State().Running {
		time.Sleep(time.Millisecond)
	}
	l := newListener(t)
	s := &http.Server{Handler: http.NotFoundHandler()}
	e := run(context.Background(), s, l, m)
	if !errors.Is(e, ErrRuntime) {
		t.Fatal("collector failure left runtime alive")
	}
	stop()
	<-done
}
