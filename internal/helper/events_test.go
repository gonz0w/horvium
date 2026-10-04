// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/eventlog"
	"github.com/gonz0w/horvium/internal/inventory"
)

type helperEventSink struct {
	sync.Mutex
	bytes.Buffer
}

func (s *helperEventSink) Write(b []byte) (int, error) {
	s.Lock()
	defer s.Unlock()
	return s.Buffer.Write(b)
}
func (s *helperEventSink) data() []byte {
	s.Lock()
	defer s.Unlock()
	return append([]byte(nil), s.Buffer.Bytes()...)
}
func helperRecords(t *testing.T, s *helperEventSink, count int) []map[string]any {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		var rows []map[string]any
		for _, line := range bytes.Split(bytes.TrimSpace(s.data()), []byte("\n")) {
			if len(line) == 0 {
				continue
			}
			var row map[string]any
			if json.Unmarshal(line, &row) != nil {
				t.Fatal("invalid helper event schema")
			}
			rows = append(rows, row)
		}
		if len(rows) >= count {
			return rows
		}
		if time.Now().After(deadline) {
			t.Fatal("helper event missing")
		}
		time.Sleep(time.Millisecond)
	}
}
func TestHelperDenialsAreClassifiedWithoutCallerData(t *testing.T) {
	payload := []byte(`{"version":1,"request_id":"sensitive-request","operation":"destroy","path":"sensitive-path"}`)
	var oversized [4]byte
	binary.BigEndian.PutUint32(oversized[:], maxFrame+1)
	for _, tc := range []struct {
		name           string
		uid            uint32
		peerError      error
		input          []byte
		class, outcome string
	}{
		{"kernel UID", 43, nil, nil, "permission", "rejected"},
		{"credential failure", 42, errors.New("sensitive-kernel-error"), nil, "permission", "rejected"},
		{"unknown operation", 42, nil, frame(`{"version":1,"request_id":"sensitive-request","operation":"destroy"}`), "protocol", "rejected"},
		{"extra byte", 42, nil, append(frame(`{"version":1,"request_id":"sensitive-request","operation":"inventory"}`), 0), "protocol", "rejected"},
		{"forged operation", 42, nil, frame(string(payload)), "protocol", "rejected"},
		{"oversized frame", 42, nil, oversized[:], "limit", "rejected"},
		{"idle deadline", 42, nil, nil, "timeout", "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, connection := connected(t)
			var calls atomic.Int32
			s := testServer(collectFunc(func(context.Context) inventory.Snapshot { calls.Add(1); return observation() }), func(*net.UnixConn) (uint32, error) { return tc.uid, tc.peerError })
			s.config.Timeout = 100 * time.Millisecond
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sink := &helperEventSink{}
			s.events = eventlog.New(ctx, sink)
			done := make(chan struct{})
			go func() { s.handle(ctx, connection); close(done) }()
			if tc.input != nil {
				if _, err := client.Write(tc.input); err != nil {
					t.Fatal(err)
				}
				_ = client.CloseWrite()
			}
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("denial waited for caller or logging")
			}
			rows := helperRecords(t, sink, 1)
			if len(rows) != 1 || rows[0]["operation"] != "helper" || rows[0]["error_class"] != tc.class || rows[0]["outcome"] != tc.outcome {
				t.Fatal("denial misclassified")
			}
			if calls.Load() != 0 || bytes.Contains(sink.data(), []byte("sensitive")) {
				t.Fatal("denial collected or disclosed caller data")
			}
			if rows[0]["id"] == "sensitive-request" || len(rows[0]) != 8 {
				t.Fatal("untrusted ID or expanded schema")
			}
		})
	}
}

type stalledHelperSink struct {
	entered, release chan struct{}
	once             sync.Once
}

func (s *stalledHelperSink) Write(b []byte) (int, error) {
	s.once.Do(func() { close(s.entered) })
	<-s.release
	return len(b), nil
}
func TestHelperDenialFloodCannotBlockHealthyCollection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sink := &stalledHelperSink{entered: make(chan struct{}), release: make(chan struct{})}
	defer close(sink.release)
	logger := eventlog.New(ctx, sink)
	var calls atomic.Int32
	source := collectFunc(func(context.Context) inventory.Snapshot { calls.Add(1); return observation() })
	for i := 0; i < 100; i++ {
		client, connection := connected(t)
		s := testServer(source, func(*net.UnixConn) (uint32, error) { return 43, nil })
		s.events = logger
		s.handle(ctx, connection)
		client.Close()
	}
	select {
	case <-sink.entered:
	case <-time.After(time.Second):
		t.Fatal("denial event missing")
	}
	client, connection := connected(t)
	s := testServer(source, func(*net.UnixConn) (uint32, error) { return 42, nil })
	s.events = logger
	go s.handle(ctx, connection)
	if err := writeRequest(client, "sensitive-request", DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	client.CloseWrite()
	if _, err := readSnapshot(client, "sensitive-request", DefaultLimits()); err != nil || calls.Load() != 1 {
		t.Fatal("blocked log sink prevented healthy collection")
	}
}
func TestHelperFailedExchangeReleasesCollection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sink := &helperEventSink{}
	logger := eventlog.New(ctx, sink)
	client, connection := connected(t)
	s := testServer(collectFunc(func(context.Context) inventory.Snapshot { return observation() }), func(*net.UnixConn) (uint32, error) { return 42, nil })
	s.events = logger
	s.config.Limits.MaxRecords = 2
	go s.handle(ctx, connection)
	_ = writeRequest(client, "sensitive-request", DefaultLimits())
	client.CloseWrite()
	if _, err := readSnapshot(client, "sensitive-request", DefaultLimits()); err == nil {
		t.Fatal("over-limit exchange published")
	}
	rows := helperRecords(t, sink, 1)
	if rows[0]["outcome"] != "failed" || rows[0]["error_class"] != "limit" || !s.collection.TryLock() {
		t.Fatal("failed exchange not classified or collection retained")
	}
	s.collection.Unlock()
}

func TestHelperBusyAndCancellationEventsDoNotRetainWork(t *testing.T) {
	logContext, stopLog := context.WithCancel(context.Background())
	defer stopLog()
	sink := &helperEventSink{}
	logger := eventlog.New(logContext, sink)
	started := make(chan struct{})
	var calls atomic.Int32
	s := testServer(collectFunc(func(ctx context.Context) inventory.Snapshot {
		calls.Add(1)
		close(started)
		<-ctx.Done()
		return observation()
	}), func(*net.UnixConn) (uint32, error) { return 42, nil })
	s.events = logger
	client, connection := connected(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { s.handle(ctx, connection); close(done) }()
	_ = writeRequest(client, "sensitive-first", DefaultLimits())
	client.CloseWrite()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("collection not started")
	}
	other, second := connected(t)
	go s.handle(logContext, second)
	_ = writeRequest(other, "sensitive-second", DefaultLimits())
	other.CloseWrite()
	if _, err := readSnapshot(other, "sensitive-second", DefaultLimits()); err == nil {
		t.Fatal("busy request queued")
	}
	rows := helperRecords(t, sink, 1)
	if rows[0]["error_class"] != "capacity" || rows[0]["outcome"] != "rejected" || calls.Load() != 1 {
		t.Fatal("busy event missing or extra collection")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancelled helper blocked")
	}
	rows = helperRecords(t, sink, 2)
	if rows[1]["error_class"] != "cancelled" || rows[1]["outcome"] != "cancelled" || !s.collection.TryLock() {
		t.Fatal("cancellation event missing or work retained")
	}
	s.collection.Unlock()
	if bytes.Contains(sink.data(), []byte("sensitive")) {
		t.Fatal("caller IDs disclosed")
	}
}
