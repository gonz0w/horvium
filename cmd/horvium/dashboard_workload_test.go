// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDashboardWorkloadPlanCoversFiveCases(t *testing.T) {
	for _, count := range []int{1, 100, 137, 10000} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			plan, err := dashboardPlan(count)
			if err != nil || len(plan) != 1000 {
				t.Fatal("missing workload", err)
			}
			cases := []string{"overview", "pools-default", "pools-maximum", "datasets-default", "datasets-maximum"}
			seen := map[string]int{}
			for i, request := range plan {
				want := "/"
				switch i % 5 {
				case 1:
					want = "/storage/pools?offset=0&sort=name"
				case 2:
					want = "/storage/pools?limit=500&offset=0&sort=name"
				case 3:
					want = fmt.Sprintf("/storage/datasets?offset=%d&sort=name", ((i/5)*100)%count)
				case 4:
					want = fmt.Sprintf("/storage/datasets?limit=500&offset=%d&sort=name", ((i/5)*500)%count)
				}
				if request.Index != i || request.Case != cases[i%5] || request.Path != want {
					t.Fatalf("wrong request %d: %+v", i, request)
				}
				seen[request.Case]++
			}
			for _, name := range cases {
				if seen[name] != 200 {
					t.Fatalf("case %s has %d", name, seen[name])
				}
			}
			for worker := 0; worker < 10; worker++ {
				for _, name := range cases {
					n := 0
					for _, request := range plan[worker*100 : (worker+1)*100] {
						if request.Case == name {
							n++
						}
					}
					if n != 20 {
						t.Fatalf("worker %d case %s has %d", worker, name, n)
					}
				}
			}
		})
	}
	for _, count := range []int{0, -1, 100001} {
		if _, err := dashboardPlan(count); err == nil {
			t.Fatal("unbounded/empty dataset count accepted")
		}
	}
}

type dashboardBody struct {
	io.Reader
	closed   bool
	closeErr error
}

func (b *dashboardBody) Close() error { b.closed = true; return b.closeErr }

type dashboardBrokenReader struct{}

func (dashboardBrokenReader) Read([]byte) (int, error) { return 0, errors.New("read failure") }

func TestDashboardBodyRequiresCompleteReadAndClose(t *testing.T) {
	for _, n := range []int{0, 1, 32 << 20, (32 << 20) + 1} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			body := &dashboardBody{Reader: strings.NewReader(strings.Repeat("x", n))}
			got, err := dashboardReadBody(body)
			if !body.closed {
				t.Fatal("body leaked")
			}
			if n > 32<<20 {
				if err == nil || got != nil {
					t.Fatal("oversized response prefix accepted")
				}
			} else if err != nil || len(got) != n {
				t.Fatal("complete response rejected", err)
			}
		})
	}
	for _, body := range []*dashboardBody{{Reader: dashboardBrokenReader{}}, {Reader: strings.NewReader("ok"), closeErr: errors.New("close failure")}} {
		if got, err := dashboardReadBody(body); err == nil || got != nil || !body.closed {
			t.Fatal("failed read/close passed or leaked")
		}
	}
}

func dashboardGoodSamples() []dashboardSample {
	plan, _ := dashboardPlan(10000)
	samples := make([]dashboardSample, len(plan))
	for i, request := range plan {
		samples[i] = dashboardSample{Index: i, Case: request.Case, Outcome: "success", Status: 200, Duration: time.Duration(i+1) * time.Microsecond}
	}
	return samples
}

func TestDashboardSummaryRecomputesPerCaseAndSeparatesExpiry(t *testing.T) {
	samples := dashboardGoodSamples()
	original := slices.Clone(samples)
	summary, err := dashboardSummarize(samples, "initial")
	if err != nil || summary.Success != 1000 || summary.Expiry != 0 || summary.Latency.P50 != 500*time.Microsecond || summary.Latency.P95 != 950*time.Microsecond || summary.Latency.P99 != 990*time.Microsecond {
		t.Fatal("wrong aggregate", summary, err)
	}
	for i, name := range []string{"overview", "pools-default", "pools-maximum", "datasets-default", "datasets-maximum"} {
		c := summary.Cases[name]
		if c.Scheduled != 200 || c.Success != 200 || c.Expiry != 0 || c.Latency.P50 != time.Duration(496+i)*time.Microsecond || c.Latency.P95 != time.Duration(946+i)*time.Microsecond || c.Latency.P99 != time.Duration(986+i)*time.Microsecond {
			t.Fatalf("wrong case %s: %+v", name, c)
		}
	}
	if !slices.Equal(original, samples) {
		t.Fatal("raw observations mutated")
	}
	samples[0].Outcome, samples[0].Status, samples[0].Duration = "expiry", 409, time.Nanosecond
	summary, err = dashboardSummarize(samples, "replacement")
	if err != nil || summary.Success != 999 || summary.Expiry != 1 || summary.Cases["overview"].Success != 199 || summary.Cases["overview"].Expiry != 1 || summary.Cases["overview"].Latency.P50 != 501*time.Microsecond || summary.ExpiryLatency.P50 != time.Nanosecond {
		t.Fatal("expiry polluted successful latency", summary, err)
	}
	if _, err = dashboardSummarize(samples, "initial"); err == nil {
		t.Fatal("initial expiry qualified")
	}
}

func TestDashboardSummaryRejectsIncompleteOrMisclassifiedObservations(t *testing.T) {
	mutations := map[string]func([]dashboardSample) []dashboardSample{
		"missing":              func(s []dashboardSample) []dashboardSample { return s[:999] },
		"extra":                func(s []dashboardSample) []dashboardSample { return append(s, s[0]) },
		"duplicate":            func(s []dashboardSample) []dashboardSample { s[1] = s[0]; return s },
		"wrong-case":           func(s []dashboardSample) []dashboardSample { s[1].Case = "overview"; return s },
		"wrong-index":          func(s []dashboardSample) []dashboardSample { s[1].Index = 1000; return s },
		"unknown-outcome":      func(s []dashboardSample) []dashboardSample { s[1].Outcome = "skipped"; return s },
		"failure":              func(s []dashboardSample) []dashboardSample { s[1].Outcome = "failure"; return s },
		"wrong-success-status": func(s []dashboardSample) []dashboardSample { s[1].Status = 303; return s },
		"wrong-expiry-status":  func(s []dashboardSample) []dashboardSample { s[0].Outcome = "expiry"; return s },
		"pool-expiry":          func(s []dashboardSample) []dashboardSample { s[1].Outcome = "expiry"; s[1].Status = 409; return s },
		"zero-duration":        func(s []dashboardSample) []dashboardSample { s[1].Duration = 0; return s },
		"negative-duration":    func(s []dashboardSample) []dashboardSample { s[1].Duration = -1; return s },
		"all-overview-expiry": func(s []dashboardSample) []dashboardSample {
			for i := range s {
				if i%5 == 0 {
					s[i].Outcome = "expiry"
					s[i].Status = 409
				}
			}
			return s
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			if _, err := dashboardSummarize(mutate(dashboardGoodSamples()), "replacement"); err == nil {
				t.Fatal("invalid workload qualified")
			}
		})
	}
	if _, err := dashboardSummarize(dashboardGoodSamples(), "future"); err == nil {
		t.Fatal("unknown phase qualified")
	}
}

func TestDashboardLargeLatencyBudgetCannotHideSlowCase(t *testing.T) {
	samples := dashboardGoodSamples()
	// Forty slow Overview requests: aggregate p95 is fast, Overview p95 is slow.
	for i := 0; i < 200; i += 5 {
		samples[i].Duration = 250 * time.Millisecond
	}
	summary, err := dashboardSummarize(samples, "initial")
	if err != nil || summary.Latency.P95 >= 250*time.Millisecond {
		t.Fatal("invalid masking fixture", err)
	}
	if err = dashboardLatencyBudget(summary, true); err == nil {
		t.Fatal("slow case hidden by aggregate")
	}
	if err = dashboardLatencyBudget(summary, false); err != nil {
		t.Fatal("invented small latency budget")
	}
	summary, _ = dashboardSummarize(dashboardGoodSamples(), "initial")
	if err = dashboardLatencyBudget(summary, true); err != nil {
		t.Fatal("fast large workload rejected", err)
	}
	summary, _ = dashboardSummarize(dashboardGoodSamples(), "replacement")
	if err = dashboardLatencyBudget(summary, true); err == nil {
		t.Fatal("replacement phase substituted for initial budget")
	}
}

func TestDashboardDriverRecordsAllRequestsAndRequiresPredicate(t *testing.T) {
	for _, mode := range []string{"success", "mixed-expiry", "expiry", "initial-expiry", "wrong-expiry", "predicate-failure", "unsafe-expiry", "login", "transport"} {
		t.Run(mode, func(t *testing.T) {
			var requests, checked, responses atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if mode == "mixed-expiry" && r.URL.Path == "/" && responses.Add(1)%2 == 0 {
					w.WriteHeader(409)
				}
				if (mode == "expiry" || mode == "initial-expiry" || mode == "unsafe-expiry") && r.URL.Path == "/" {
					w.WriteHeader(409)
				}
				if mode == "wrong-expiry" && r.URL.Path == "/storage/pools" {
					w.WriteHeader(409)
				}
				if mode == "login" {
					w.WriteHeader(303)
				}
				fmt.Fprint(w, "complete-body")
			}))
			defer server.Close()
			client := server.Client()
			if mode == "transport" {
				server.Close()
			}
			plan, _ := dashboardPlan(137)
			phase := "replacement"
			if mode == "initial-expiry" {
				phase = "initial"
			}
			predicate := func(request dashboardRequest, status int, body []byte) error {
				checked.Add(1)
				if string(body) != "complete-body" {
					return errors.New("incomplete")
				}
				if mode == "predicate-failure" || mode == "unsafe-expiry" && status == 409 {
					return errors.New("predicate failed")
				}
				return nil
			}
			samples, err := dashboardLoad(client, server.URL, 137, plan, phase, predicate)
			if len(samples) != 1000 {
				t.Fatal("scheduled observations lost", len(samples))
			}
			if mode == "success" || mode == "mixed-expiry" {
				if err != nil || requests.Load() != 1000 || checked.Load() != 1000 {
					t.Fatal("successful driver failed", err)
				}
			} else if err == nil {
				t.Fatal("invalid or all-expiry workload qualified", mode)
			}
			if mode != "transport" && requests.Load() != 1000 {
				t.Fatal("early stop or retry changed schedule", requests.Load())
			}
			for i, sample := range samples {
				if sample.Index != i || sample.Case != plan[i].Case || sample.Duration <= 0 {
					t.Fatal("request identity/timing lost")
				}
			}
			if mode == "mixed-expiry" {
				summary, e := dashboardSummarize(samples, phase)
				if e != nil || summary.Success != 900 || summary.Expiry != 100 || summary.Cases["overview"].Success != 100 {
					t.Fatal("mixed expiry accounting failed", e)
				}
			}
			before := requests.Load()
			if _, err = dashboardLoad(client, server.URL, 137, plan, phase, nil); err == nil || requests.Load() != before {
				t.Fatal("missing predicate sent requests")
			}
			if _, err = dashboardLoad(client, server.URL, 137, plan[:999], phase, predicate); err == nil || requests.Load() != before {
				t.Fatal("incomplete plan sent requests")
			}
		})
	}
}

func TestDashboardDriverRejectsChangedScheduleAndOriginBeforeHTTP(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer server.Close()
	predicate := func(dashboardRequest, int, []byte) error { return nil }
	for _, change := range []func([]dashboardRequest){
		func(p []dashboardRequest) { p[3].Path = "/storage/datasets?offset=1&sort=name" },
		func(p []dashboardRequest) { p[4].Path = "/storage/datasets?limit=100&offset=0&sort=name" },
		func(p []dashboardRequest) { p[0].Path = "/?sort=name" },
		func(p []dashboardRequest) { p[1].Path = "http://example.invalid/" },
		func(p []dashboardRequest) { p[1].Index = 0 },
		func(p []dashboardRequest) { p[1].Case = "overview" },
	} {
		plan, _ := dashboardPlan(137)
		change(plan)
		if _, err := dashboardLoad(server.Client(), server.URL, 137, plan, "initial", predicate); err == nil {
			t.Fatal("changed schedule accepted")
		}
	}
	plan, _ := dashboardPlan(137)
	for _, base := range []string{"http://example.invalid", server.URL + "/other", server.URL + "?token=secret", server.URL + "#fragment", "http://user@127.0.0.1", "malformed"} {
		if _, err := dashboardLoad(server.Client(), base, 137, plan, "initial", predicate); err == nil {
			t.Fatal("invalid fixture origin accepted")
		}
	}
	if _, err := dashboardLoad(server.Client(), server.URL, 100, plan, "initial", predicate); err == nil {
		t.Fatal("wrong declared count accepted")
	}
	if _, err := dashboardLoad(nil, server.URL, 137, plan, "initial", predicate); err == nil {
		t.Fatal("missing client accepted")
	}
	if _, err := dashboardLoad(server.Client(), server.URL, 137, plan, "future", predicate); err == nil {
		t.Fatal("unknown phase accepted")
	}
	if calls.Load() != 0 {
		t.Fatal("invalid configuration sent HTTP requests")
	}
}

func TestDashboardDriverUsesTenReadersAndDoesNotFollowLogin(t *testing.T) {
	first := make(chan struct{}, 10)
	release := make(chan struct{})
	var active, peak, requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		n := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); n > old; old = peak.Load() {
			if peak.CompareAndSwap(old, n) {
				break
			}
		}
		select {
		case first <- struct{}{}:
		default:
		}
		<-release
		w.Header().Set("Location", "/login")
		w.WriteHeader(303)
	}))
	defer server.Close()
	// Cleanup also releases handlers if the concurrency assertion times out.
	var once atomic.Bool
	unblock := func() {
		if once.CompareAndSwap(false, true) {
			close(release)
		}
	}
	defer unblock()
	plan, _ := dashboardPlan(100)
	done := make(chan error, 1)
	go func() {
		_, err := dashboardLoad(server.Client(), server.URL, 100, plan, "initial", func(dashboardRequest, int, []byte) error { return nil })
		done <- err
	}()
	for i := 0; i < 10; i++ {
		select {
		case <-first:
		case <-time.After(5 * time.Second):
			t.Fatal("ten readers did not start")
		}
	}
	unblock()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("login denial qualified")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("driver did not finish")
	}
	if requests.Load() != 1000 || peak.Load() != 10 {
		t.Fatal("wrong concurrency, retry or redirect", requests.Load(), peak.Load())
	}
}

type dashboardTransport func(*http.Request) (*http.Response, error)

func (f dashboardTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type dashboardObservedBody struct {
	io.Reader
	closed   *atomic.Int32
	closeErr error
}

func (b *dashboardObservedBody) Close() error { b.closed.Add(1); return b.closeErr }

func TestDashboardDriverDoesNotValidateOrQualifyFailedBodies(t *testing.T) {
	for _, mode := range []string{"read", "close"} {
		t.Run(mode, func(t *testing.T) {
			var calls, closed, checked atomic.Int32
			client := &http.Client{Transport: dashboardTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				body := &dashboardObservedBody{Reader: strings.NewReader("prefix"), closed: &closed}
				if mode == "read" {
					body.Reader = dashboardBrokenReader{}
				} else {
					body.closeErr = errors.New("close failed")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: body, Request: r}, nil
			})}
			plan, _ := dashboardPlan(100)
			samples, err := dashboardLoad(client, "http://127.0.0.1", 100, plan, "initial", func(dashboardRequest, int, []byte) error { checked.Add(1); return nil })
			if err == nil || len(samples) != 1000 || calls.Load() != 1000 || closed.Load() != 1000 || checked.Load() != 0 {
				t.Fatal("failed body validated, retried or leaked", err)
			}
			for _, sample := range samples {
				if sample.Outcome != "failure" || sample.Status != 200 {
					t.Fatal("failed body misclassified")
				}
			}
		})
	}
}
