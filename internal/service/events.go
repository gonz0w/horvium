// SPDX-License-Identifier: BSD-3-Clause
package service

import (
	"context"
	"github.com/gonz0w/horvium/internal/eventlog"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
	"net/http"
	"time"
)

func requestSection(r *http.Request) string {
	switch r.URL.Path {
	case "/login", "/logout", "/auth/challenge", "/auth/login", "/auth/logout":
		return "authentication"
	case "/api/datasets", "/storage/datasets":
		return "datasets"
	case "/api/pools", "/storage/pools":
		return "pools"
	case "/api/providers", "/storage/providers":
		return "geom"
	case "/", "/status", "/refresh", "/api/overview", "/api/refresh":
		return "inventory"
	case "/assets/console.css", "/assets/console.js", "/assets/htmx.min.js", "/assets/horvium.svg":
		return "asset"
	default:
		return "other"
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
	failed bool
}

func (w *statusWriter) WriteHeader(n int) {
	if w.status == 0 {
		w.status = n
	}
	w.ResponseWriter.WriteHeader(n)
}
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	n, e := w.ResponseWriter.Write(b)
	if e != nil {
		w.failed = true
	}
	return n, e
}
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func requests(l *eventlog.Log, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		id := l.NextID()
		out := &statusWriter{ResponseWriter: w}
		defer func() {
			outcome, class := "ok", ""
			switch out.status {
			case 400:
				outcome, class = "rejected", "query"
			case 401:
				outcome, class = "rejected", "authentication"
			case 403, 404, 405, 413, 431:
				outcome, class = "rejected", "policy"
			case 409:
				outcome, class = "rejected", "conflict"
			case 429:
				outcome, class = "rejected", "capacity"
			case 503:
				outcome, class = "failed", "unavailable"
				if out.Header().Get("Retry-After") != "" {
					outcome, class = "rejected", "capacity"
				}
			default:
				if out.status >= 500 {
					outcome, class = "failed", "failed"
				}
			}
			if out.status == 303 && requestSection(r) != "authentication" && r.URL.Path != "/refresh" {
				outcome, class = "rejected", "authentication"
			}
			if out.failed {
				outcome, class = "failed", "failed"
			}
			if r.Context().Err() != nil {
				outcome, class = "cancelled", "cancelled"
			}
			l.Record(eventlog.Event{Operation: "http", ID: id, Section: requestSection(r), Outcome: outcome, ErrorClass: class, Duration: time.Since(started)})
		}()
		next.ServeHTTP(out, r)
	})
}
func observed(l *eventlog.Log) func(observation.Result) {
	return func(r observation.Result) {
		outcome := "ok"
		if r.Attempt.ErrorCode != "" {
			outcome = "failed"
		}
		duration := r.Attempt.FinishedAt.Sub(r.Attempt.StartedAt)
		l.Record(eventlog.Event{Operation: "collection", ID: r.Attempt.ID, Section: "inventory", Outcome: outcome, ErrorClass: r.Attempt.ErrorCode, Duration: duration})
		if r.Attempt.ErrorCode != "" {
			return
		}
		sections := [4]string{"host", "geom", "pools", "datasets"}
		for i, status := range r.Status {
			outcome = "ok"
			class := ""
			if status == inventory.Partial {
				outcome, class = "partial", "unavailable"
			} else if status != inventory.OK {
				outcome, class = "failed", "unavailable"
			}
			l.Record(eventlog.Event{Operation: "collection", ID: r.Attempt.ID, Section: sections[i], Outcome: outcome, ErrorClass: class, Duration: duration})
		}
	}
}

type nativeEvents struct {
	source interface {
		Collect(context.Context) inventory.Snapshot
	}
	log *eventlog.Log
}

func (n nativeEvents) Collect(ctx context.Context) inventory.Snapshot {
	started := time.Now()
	s := n.source.Collect(ctx)
	outcome := "ok"
	class := ""
	if !s.Complete() {
		outcome, class = "partial", "unavailable"
	}
	if ctx.Err() != nil {
		outcome, class = "cancelled", "cancelled"
	}
	n.log.Record(eventlog.Event{Operation: "helper", ID: n.log.NextID(), Section: "inventory", Outcome: outcome, ErrorClass: class, Duration: time.Since(started)})
	return s
}
