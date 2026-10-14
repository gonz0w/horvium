// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"
)

// Controller-only helpers. The native resource suite keeps its historical
// workload until inventory-specific HTML predicates and receipts are reviewed.
var dashboardCases = [...]string{"overview", "pools-default", "pools-maximum", "datasets-default", "datasets-maximum"}

const dashboardBodyLimit = 32 << 20 // Existing console complete-response ceiling.

type dashboardRequest struct {
	Index      int
	Case, Path string
}

func dashboardPlan(count int) ([]dashboardRequest, error) {
	if count < 1 || count > 100000 {
		return nil, errors.New("invalid fixture dataset count")
	}
	plan := make([]dashboardRequest, 1000)
	for i := range plan {
		request := dashboardRequest{Index: i, Case: dashboardCases[i%5], Path: "/"}
		if i%5 != 0 {
			path, limit, offset := "/storage/pools", 100, 0
			if i%5 >= 3 {
				path = "/storage/datasets"
			}
			if i%5 == 2 || i%5 == 4 {
				limit = 500
			}
			if i%5 >= 3 {
				offset = ((i / 5) * limit) % count
			}
			query := url.Values{"sort": {"name"}, "offset": {strconv.Itoa(offset)}}
			if limit == 500 {
				query.Set("limit", "500")
			}
			request.Path = path + "?" + query.Encode()
		}
		plan[i] = request
	}
	return plan, nil
}

func dashboardReadBody(body io.ReadCloser) ([]byte, error) {
	if body == nil {
		return nil, errors.New("missing fixture response body")
	}
	b, readErr := io.ReadAll(io.LimitReader(body, dashboardBodyLimit+1))
	closeErr := body.Close()
	if readErr != nil || closeErr != nil || len(b) > dashboardBodyLimit {
		return nil, errors.New("incomplete or oversized fixture response")
	}
	return b, nil
}

type dashboardSample struct {
	Index         int
	Case, Outcome string
	Status        int
	BodyBytes     int
	Predicate     string
	Duration      time.Duration
}
type dashboardQuantiles struct{ P50, P95, P99 time.Duration }
type dashboardCaseSummary struct {
	Scheduled, Success, Expiry int
	Latency, ExpiryLatency     dashboardQuantiles
}
type dashboardSummary struct {
	Phase                      string
	Scheduled, Success, Expiry int
	Latency, ExpiryLatency     dashboardQuantiles
	Cases                      map[string]dashboardCaseSummary
}

func dashboardPercentiles(durations []time.Duration) dashboardQuantiles {
	if len(durations) == 0 {
		return dashboardQuantiles{}
	}
	sorted := slices.Clone(durations)
	slices.Sort(sorted)
	rank := func(percent int) time.Duration { return sorted[(len(sorted)*percent+99)/100-1] }
	return dashboardQuantiles{rank(50), rank(95), rank(99)}
}

func dashboardSummarize(samples []dashboardSample, phase string) (dashboardSummary, error) {
	invalid := func() (dashboardSummary, error) {
		return dashboardSummary{}, errors.New("incomplete or invalid dashboard workload")
	}
	if phase != "initial" && phase != "replacement" || len(samples) != 1000 {
		return invalid()
	}
	out := dashboardSummary{Phase: phase, Scheduled: 1000, Cases: make(map[string]dashboardCaseSummary)}
	seen := [1000]bool{}
	successful, expired := make(map[string][]time.Duration), make(map[string][]time.Duration)
	var allSuccess, allExpiry []time.Duration
	for _, sample := range samples {
		if sample.Index < 0 || sample.Index >= 1000 || seen[sample.Index] || sample.Case != dashboardCases[sample.Index%5] || sample.Duration <= 0 {
			return invalid()
		}
		seen[sample.Index] = true
		c := out.Cases[sample.Case]
		c.Scheduled++
		switch sample.Outcome {
		case "success":
			if sample.Status != 200 {
				return invalid()
			}
			c.Success++
			out.Success++
			successful[sample.Case] = append(successful[sample.Case], sample.Duration)
			allSuccess = append(allSuccess, sample.Duration)
		case "expiry":
			if phase != "replacement" || sample.Case != "overview" || sample.Status != 409 {
				return invalid()
			}
			c.Expiry++
			out.Expiry++
			expired[sample.Case] = append(expired[sample.Case], sample.Duration)
			allExpiry = append(allExpiry, sample.Duration)
		default:
			return invalid()
		}
		out.Cases[sample.Case] = c
	}
	for _, name := range dashboardCases {
		c := out.Cases[name]
		if c.Scheduled != 200 || c.Success < 1 || c.Success+c.Expiry != c.Scheduled {
			return invalid()
		}
		c.Latency = dashboardPercentiles(successful[name])
		c.ExpiryLatency = dashboardPercentiles(expired[name])
		out.Cases[name] = c
	}
	out.Latency = dashboardPercentiles(allSuccess)
	out.ExpiryLatency = dashboardPercentiles(allExpiry)
	return out, nil
}

func dashboardLatencyBudget(summary dashboardSummary, large bool) error {
	if summary.Phase != "initial" || summary.Scheduled != 1000 || summary.Success != 1000 || summary.Expiry != 0 || len(summary.Cases) != 5 {
		return errors.New("initial complete workload required")
	}
	for _, name := range dashboardCases {
		c := summary.Cases[name]
		if c.Scheduled != 200 || c.Success != 200 || c.Expiry != 0 || c.Latency.P95 <= 0 {
			return errors.New("initial complete case required")
		}
		if large && c.Latency.P95 >= 250*time.Millisecond {
			return fmt.Errorf("large p95 budget failed for %s", name)
		}
	}
	if summary.Latency.P95 <= 0 || large && summary.Latency.P95 >= 250*time.Millisecond {
		return errors.New("large aggregate p95 budget failed")
	}
	return nil
}

// Predicate validation runs after timing stops. Its pacing remains part of the
// observed workload, but its allocations/time are outside the measured pair.
// The mandatory predicate must independently verify original inventory, exact
// tables/comparisons and generation agreement, or the safe expiry/reload form.
type dashboardPredicate func(dashboardRequest, int, []byte) error

// Fixed diagnostic vocabulary only. Never serialize predicate/transport errors,
// response bodies, identities or session material into failure observations.
func dashboardFailureClassification(err error) string {
	var table dashboardTableMismatch
	if errors.As(err, &table) {
		switch table {
		case "structure", "header-count", "header", "empty-shape", "empty-text", "row-count", "row-shape", "row-name", "row-link", "row-selection", "cell-style", "cell-guid", "cell-size", "cell-allocated", "cell-free", "cell-pool-state", "cell-used", "cell-available", "cell-referenced":
			return "failure:table-" + string(table)
		default:
			return "failure:table"
		}
	}

	switch err.Error() {
	case "malformed or unbounded dashboard document":
		return "failure:document"
	case "required unique response element missing":
		return "failure:element"
	case "wrong original dashboard table", "complete dashboard table missing", "duplicate dashboard table":
		return "failure:table"
	case "wrong original dashboard comparisons":
		return "failure:comparisons"
	case "wrong storage query or freshness metadata":
		return "failure:storage-metadata"
	case "wrong original Overview metadata":
		return "failure:overview-metadata"
	case "invalid response generation":
		return "failure:generation"
	case "unexpected dashboard table count":
		return "failure:table-count"
	default:
		return "failure:predicate"
	}
}

func dashboardClassifyResponse(request dashboardRequest, phase string, status int, body []byte, transportErr, bodyErr error, predicate dashboardPredicate) (string, string) {
	if transportErr != nil {
		return "failure", "failure:transport"
	}
	if bodyErr != nil {
		return "failure", "failure:body"
	}
	if status != 200 && !(phase == "replacement" && request.Case == "overview" && status == 409) {
		return "failure", "failure:status"
	}
	if err := predicate(request, status, body); err != nil {
		return "failure", dashboardFailureClassification(err)
	}
	if status == 409 {
		return "expiry", "safe-expiry"
	}
	return "success", "original-inventory"
}

func dashboardLoad(client *http.Client, base string, count int, plan []dashboardRequest, phase string, predicate dashboardPredicate) ([]dashboardSample, error) {
	if client == nil || predicate == nil || phase != "initial" && phase != "replacement" || len(plan) != 1000 {
		return nil, errors.New("incomplete dashboard workload configuration")
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || !net.ParseIP(u.Hostname()).IsLoopback() {
		return nil, errors.New("fixture requires a loopback HTTP origin")
	}
	// Reject a changed route, distribution or page sequence before sending any
	// request. No API call is needed: the count belongs to the declared fixture.
	expected, err := dashboardPlan(count)
	if err != nil || !slices.Equal(plan, expected) {
		return nil, errors.New("invalid dashboard request schedule")
	}
	plan = slices.Clone(plan)
	// Never follow a login redirect and mistake its 200 response for the view.
	httpClient := *client
	if httpClient.Timeout == 0 {
		httpClient.Timeout = 10 * time.Second
	}
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	samples := make([]dashboardSample, 1000)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for worker := 0; worker < 10; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			for i := worker * 100; i < (worker+1)*100; i++ {
				request := plan[i]
				sample := dashboardSample{Index: i, Case: request.Case, Outcome: "failure"}
				started := time.Now()
				response, transportErr := httpClient.Get(base + request.Path)
				var body []byte
				var bodyErr error
				if transportErr == nil {
					sample.Status = response.StatusCode
					body, bodyErr = dashboardReadBody(response.Body)
				}
				sample.Duration = time.Since(started)
				sample.BodyBytes = len(body)
				sample.Outcome, sample.Predicate = dashboardClassifyResponse(request, phase, sample.Status, body, transportErr, bodyErr, predicate)
				samples[i] = sample
			}
		}()
	}
	close(start)
	workers.Wait()
	_, err = dashboardSummarize(samples, phase)
	return samples, err
}
