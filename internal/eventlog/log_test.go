// SPDX-License-Identifier: BSD-3-Clause
package eventlog

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

type sink struct {
	sync.Mutex
	bytes.Buffer
}

func (s *sink) Write(b []byte) (int, error) { s.Lock(); defer s.Unlock(); return s.Buffer.Write(b) }
func (s *sink) data() string                { s.Lock(); defer s.Unlock(); return s.Buffer.String() }
func TestFixedSchemaRedactionAndVolume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &sink{}
	l := New(ctx, s)
	for i := 0; i < 10000; i++ {
		l.Record(Event{Operation: "http", ID: l.NextID(), Section: "datasets", Outcome: "ok", Duration: time.Millisecond})
	}
	l.Record(Event{Operation: "token=secret", ID: "cookie=secret", Section: "host", Outcome: "ok"})
	deadline := time.Now().Add(time.Second)
	for s.data() == "" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	text := s.data()
	if text == "" || strings.Contains(text, "secret") {
		t.Fatal("missing or unsafe events")
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) > 60 {
		t.Fatal("log flood exceeded per-minute budget")
	}
	for _, line := range lines {
		var v map[string]any
		if json.Unmarshal([]byte(line), &v) != nil || v["timestamp"] == nil || v["id"] == nil || v["duration_ms"] == nil || v["operation"] != "http" {
			t.Fatal("invalid structured event")
		}
	}
}

type blockedSink struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *blockedSink) Write(b []byte) (int, error) {
	s.once.Do(func() { close(s.entered) })
	<-s.release
	return len(b), nil
}
func TestBlockedLogSinkDoesNotBlockManagementOrCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &blockedSink{entered: make(chan struct{}), release: make(chan struct{})}
	defer close(s.release)
	l := New(ctx, s)
	l.Record(Event{Operation: "management", ID: l.NextID(), Section: "inventory", Outcome: "ok"})
	select {
	case <-s.entered:
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10000; i++ {
			l.Record(Event{Operation: "http", ID: l.NextID(), Section: "inventory", Outcome: "rejected"})
		}
		cancel()
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("blocked by logging")
	}
}
