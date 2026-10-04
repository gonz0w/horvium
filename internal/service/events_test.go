// SPDX-License-Identifier: BSD-3-Clause
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/gonz0w/horvium/internal/eventlog"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

type eventSink struct {
	sync.Mutex
	bytes.Buffer
}

func (s *eventSink) Write(b []byte) (int, error) {
	s.Lock()
	defer s.Unlock()
	return s.Buffer.Write(b)
}
func (s *eventSink) data() []byte {
	s.Lock()
	defer s.Unlock()
	return append([]byte(nil), s.Buffer.Bytes()...)
}
func TestRequestEventsRedactCredentialsAndRawErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sink := &eventSink{}
	events := eventlog.New(ctx, sink)
	h := requests(events, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "sensitive-native-error", 503) }))
	r := httptest.NewRequest("POST", "/api/refresh?token=sensitive-query", strings.NewReader("token=sensitive-body"))
	r.Header.Set("Cookie", "horvium_session=sensitive-cookie")
	r.Header.Set("Authorization", "sensitive-authorization")
	r.Header.Set("X-Request-ID", "sensitive-request-id")
	h.ServeHTTP(httptest.NewRecorder(), r)
	until := time.Now().Add(time.Second)
	for len(sink.data()) == 0 {
		if time.Now().After(until) {
			t.Fatal("request event missing")
		}
		time.Sleep(time.Millisecond)
	}
	b := sink.data()
	if bytes.Contains(b, []byte("sensitive")) {
		t.Fatal("request event disclosed raw data")
	}
	var record map[string]any
	if json.Unmarshal(b, &record) != nil || record["section"] != "inventory" || record["outcome"] != "failed" || record["error_class"] != "unavailable" {
		t.Fatal("request failure not classified")
	}
}
