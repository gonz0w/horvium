// SPDX-License-Identifier: BSD-3-Clause
// Package eventlog emits bounded, fixed-schema operational events. It accepts
// no raw request, error, command output, credential or observed host field.
package eventlog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Event struct {
	Operation, ID, Section, Outcome, ErrorClass string
	Duration                                    time.Duration
}
type record struct {
	Timestamp  time.Time `json:"timestamp"`
	Operation  string    `json:"operation"`
	ID         string    `json:"id"`
	Section    string    `json:"section"`
	Outcome    string    `json:"outcome"`
	ErrorClass string    `json:"error_class"`
	DurationMS int64     `json:"duration_ms"`
	Suppressed uint64    `json:"suppressed"`
}
type Log struct {
	ctx        context.Context
	queue      chan record
	nonce      string
	serial     atomic.Uint64
	mu         sync.Mutex
	window     time.Time
	emitted    int
	suppressed uint64
}

// New owns one writer goroutine and at most 64 queued fixed-size records.
// A blocked sink never blocks callers or shutdown. The foreground process may
// exit before queued records are written; these are operational logs, not a
// durable audit ledger. No telemetry, log files or syslog sockets are opened.
func New(ctx context.Context, w io.Writer) *Log {
	var seed [16]byte
	if _, e := rand.Read(seed[:]); e != nil {
		return nil
	}
	l := &Log{ctx: ctx, queue: make(chan record, 64), nonce: hex.EncodeToString(seed[:])}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-l.queue:
				if ctx.Err() != nil {
					return
				}
				b, e := json.Marshal(r)
				if e != nil {
					return
				}
				b = append(b, '\n')
				if n, e := w.Write(b); e != nil || n != len(b) {
					return
				}
			}
		}
	}()
	return l
}
func (l *Log) NextID() string {
	if l == nil {
		return ""
	}
	return l.nonce + "-" + strconv.FormatUint(l.serial.Add(1), 10)
}
func one(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}
func validID(s string) bool {
	p := strings.Split(s, "-")
	if len(p) != 2 || len(p[0]) != 32 || len(p[1]) > 20 {
		return false
	}
	for _, c := range p[0] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	n, e := strconv.ParseUint(p[1], 10, 64)
	return e == nil && n > 0 && strconv.FormatUint(n, 10) == p[1]
}

// Record is nonblocking and admits at most 60 records per minute per process.
// A later admitted record reports suppressed volume without storing its data.
func (l *Log) Record(e Event) {
	if l == nil || l.ctx.Err() != nil {
		return
	}
	if !validID(e.ID) || !one(e.Operation, "http", "collection", "helper", "management") || !one(e.Section, "inventory", "host", "geom", "pools", "datasets", "authentication", "asset", "other") || !one(e.Outcome, "ok", "partial", "failed", "rejected", "cancelled", "started", "stopped") || !one(e.ErrorClass, "", "failed", "cancelled", "timeout", "limit", "malformed", "protocol", "permission", "configuration", "unsupported", "authentication", "policy", "capacity", "query", "expired", "unavailable", "conflict") || e.Duration < 0 {
		return
	}
	now := time.Now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.window.IsZero() || now.Sub(l.window) >= time.Minute {
		l.window = now
		l.emitted = 0
	}
	if l.emitted >= 60 {
		l.drop()
		return
	}
	r := record{now, e.Operation, e.ID, e.Section, e.Outcome, e.ErrorClass, e.Duration.Milliseconds(), l.suppressed}
	select {
	case l.queue <- r:
		l.emitted++
		l.suppressed = 0
	default:
		l.drop()
	}
}
func (l *Log) drop() {
	if l.suppressed != ^uint64(0) {
		l.suppressed++
	}
}
