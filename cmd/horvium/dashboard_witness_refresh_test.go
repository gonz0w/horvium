// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWitnessRefreshOverlapDispatch(t *testing.T) {
	var count atomic.Int32
	arrived := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/refresh" || r.ParseForm() != nil || r.PostForm.Get("csrf") != "fixture-csrf" || len(r.PostForm) != 1 {
			t.Error("request contract")
		}
		if count.Add(1) == 3 {
			close(arrived)
		}
		select {
		case <-arrived:
		case <-r.Context().Done():
			return
		case <-time.After(3 * time.Second):
			t.Error("requests were serialized")
			return
		}
		w.WriteHeader(409)
		_, _ = io.WriteString(w, "{\"code\":\"busy\",\"message\":\"Inventory work already active\"}\n")
	}))
	defer server.Close()
	client := server.Client()
	client.Timeout = 5 * time.Second
	replies, err := witnessRefreshOverlap(client, server.URL, "fixture-csrf")
	if err != nil || len(replies) != 3 || count.Load() != 3 {
		t.Fatal("fixed dispatch", err, count.Load())
	}
	for i, r := range replies {
		if r.Index != i+1 || !r.busy() {
			t.Fatal("busy receipt", r)
		}
	}
}

func TestWitnessRefreshReplyBoundsAndNoRetry(t *testing.T) {
	for _, tc := range []struct {
		name, body          string
		status              int
		bad, busy, accepted bool
	}{
		{"busy", `{"code":"busy","message":"Inventory work already active"}`, 409, false, true, false},
		{"accepted", `{"code":"refresh_requested"}`, 202, false, false, true},
		{"wrong-status", `{"code":"busy","message":"Inventory work already active"}`, 202, false, false, false},
		{"wrong-body", `{"code":"busy","message":"different"}`, 409, false, false, false},
		{"trailing", `{"code":"busy","message":"Inventory work already active"}{}`, 409, false, false, false},
		{"limit", strings.Repeat("x", 65536), 409, false, false, false},
		{"oversize", strings.Repeat("x", 65537), 409, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var count atomic.Int32
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count.Add(1)
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer s.Close()
			r, err := witnessRefreshOnce(s.Client(), s.URL, "fixture-csrf", 1)
			if (err != nil) != tc.bad || count.Load() != 1 || len(r.Body) > 65536 || r.busy() != tc.busy || r.accepted() != tc.accepted {
				t.Fatal("bounded single request", err, count.Load(), len(r.Body))
			}
		})
	}
}

func TestWitnessRefreshOverlapDrainsFailures(t *testing.T) {
	var count atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := count.Add(1)
		w.WriteHeader(409)
		if n == 1 {
			_, _ = io.WriteString(w, strings.Repeat("x", 65537))
			return
		}
		_, _ = io.WriteString(w, `{"code":"busy","message":"Inventory work already active"}`)
	}))
	defer s.Close()
	replies, err := witnessRefreshOverlap(s.Client(), s.URL, "fixture-csrf")
	if err == nil || count.Load() != 3 || len(replies) != 3 {
		t.Fatal("all fixed requests must finish", err, count.Load())
	}
	for i, r := range replies {
		if r.Index != i+1 || r.FinishedNS < r.StartedNS {
			t.Fatal("lost terminal receipt")
		}
	}
}

func TestWitnessRefreshReadFailure(t *testing.T) {
	var count atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(409)
		_, _ = io.WriteString(w, "short")
	}))
	client := s.Client()
	if _, err := witnessRefreshOnce(client, s.URL, "fixture-csrf", 1); err == nil || count.Load() != 1 {
		t.Fatal("partial read accepted")
	}
	s.Close()
	if _, err := witnessRefreshOnce(client, s.URL, "fixture-csrf", 1); err == nil {
		t.Fatal("transport failure accepted")
	}
}
