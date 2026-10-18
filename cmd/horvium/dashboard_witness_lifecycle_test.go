// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

func TestWitnessMetadataObserveRetainsRefusedResponseWithoutRetry(t *testing.T) {
	for _, tc := range []struct {
		name, body         string
		status             int
		observed, accepted bool
	}{
		{"ok", `{"Generation":"` + witnessTestID + `","Complete":true}`, 200, true, true},
		{"expired", `{"code":"expired","message":"Inventory generation expired; reload required"}`, 409, true, false},
		{"duplicate", `{"Generation":"` + witnessTestID + `","Generation":"other"}`, 200, true, false},
		{"trailing", `{} {}`, 200, true, false},
		{"oversize", strings.Repeat("x", 65537), 200, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.Path != "/api/overview" || r.URL.Query().Get("generation") != witnessTestID {
					t.Error("metadata request")
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer s.Close()
			observed := 0
			lookup := witnessHTTPMetadataObserve(s.Client(), s.URL, func(id string, status int, b []byte) error {
				observed++
				if id != witnessTestID || status != tc.status || string(b) != tc.body {
					t.Error("original response changed")
				}
				return nil
			})
			_, err := lookup(context.Background(), witnessTestID)
			if (err == nil) != tc.accepted || (observed == 1) != tc.observed || calls.Load() != 1 {
				t.Fatal("bounded one-shot metadata observation", err, observed, calls.Load())
			}
		})
	}
}

func TestWitnessMetadataObservationFailureAndLegacyCapture(t *testing.T) {
	for _, status := range []int{200, 409} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"Generation":"`+witnessTestID+`","Complete":true}`)
		}))
		calls := 0
		_, err := witnessHTTPMetadataObserve(s.Client(), s.URL, func(string, int, []byte) error { return errors.New("PRIVATE receipt path") })(context.Background(), witnessTestID)
		if err == nil || strings.Contains(err.Error(), "PRIVATE") {
			t.Fatal("receipt failure accepted/leaked")
		}
		_, err = witnessHTTPMetadataCapture(s.Client(), s.URL, func(string, []byte) error { calls++; return nil })(context.Background(), witnessTestID)
		if (err == nil) != (status == 200) || calls != map[int]int{200: 1, 409: 0}[status] {
			t.Fatal("legacy capture behavior changed")
		}
		s.Close()
	}
}

func TestWitnessMetadataConcurrentConfirmationAndFailureDrain(t *testing.T) {
	for _, fail := range []bool{false, true} {
		source := newWitnessManagerSource(t)
		h, m, c := dashboardHTMLManagerFixtureForWitness(t, source)
		_ = h
		_ = c
		id := m.State().CurrentID
		var calls atomic.Int32
		registry := newDashboardWitnessRegistry(source.original, func(ctx context.Context, id string) (witnessPublication, error) {
			calls.Add(1)
			if fail {
				return witnessPublication{}, errors.New("PRIVATE")
			}
			o, e := m.Overview(ctx, id)
			return witnessPublicationFromOverview(o), e
		})
		replies, err := witnessConfirmConcurrent(context.Background(), registry, id)
		if (err != nil) != fail || len(replies) != 3 || calls.Load() != 1 || registry.controls() != 1 {
			t.Fatal("coalescing/drain", err, calls.Load())
		}
		for i, r := range replies {
			if r.Index != i+1 || r.Denied != fail || r.FinishedNS < r.StartedNS || !fail && (r.ID != id || len(r.RawSHA256) != 64) {
				t.Fatal("terminal confirmation receipt")
			}
		}
	}
}

// Independent declared partial snapshots exercise the real manager's retention,
// without pretending a complete raw source became a valid partial witness.
type witnessLifecycleSource struct {
	mu       sync.Mutex
	snapshot inventory.Snapshot
}

func (s *witnessLifecycleSource) Collect(context.Context, string) (inventory.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return inventory.CloneBounded(s.snapshot, inventory.DefaultLimits())
}
func TestWitnessLifecyclePartialLatestRetainsLastComplete(t *testing.T) {
	source := &witnessLifecycleSource{snapshot: performanceSnapshot(1, 0)}
	m, e := observation.New(observation.DefaultConfig(), source)
	if e != nil {
		t.Fatal(e)
	}
	_, _, _ = dashboardHTMLManagerFixture(t, m)
	first := m.State().CurrentID
	replace := func(partial bool) string {
		source.mu.Lock()
		source.snapshot = performanceSnapshot(1, 0)
		if partial {
			source.snapshot.Datasets.Status = inventory.Unavailable
			source.snapshot.Datasets.Data = nil
			source.snapshot.Datasets.Problems = []inventory.Problem{{Source: "datasets", Code: "failed"}}
		}
		source.mu.Unlock()
		old := m.State().CurrentID
		if m.Refresh() != nil {
			t.Fatal("single replacement")
		}
		dashboardHTMLWait(t, func() bool { return m.State().CurrentID != old && !m.State().Collecting })
		return m.State().CurrentID
	}
	partial1 := replace(true)
	partial2 := replace(true)
	if m.State().CompleteID != first {
		t.Fatal("last complete replaced by partial")
	}
	for _, id := range []string{first, partial2} {
		if _, e := m.Overview(context.Background(), id); e != nil {
			t.Fatal("retained generation", e)
		}
	}
	if _, e := m.Overview(context.Background(), partial1); !errors.Is(e, observation.ErrExpired) {
		t.Fatal("overwritten partial retained", e)
	}
	complete := replace(false)
	if m.State().CompleteID != complete {
		t.Fatal("new complete not current")
	}
	if _, e := m.Overview(context.Background(), first); !errors.Is(e, observation.ErrExpired) {
		t.Fatal("obsolete complete retained", e)
	}
	if o, e := m.Overview(context.Background(), partial2); e != nil || o.Complete {
		t.Fatal("previous partial retention", e)
	}
}

func dashboardHTMLManagerFixtureForWitness(t *testing.T, s *witnessManagerSource) (http.Handler, *observation.Manager, *http.Cookie) {
	t.Helper()
	m, e := observation.New(observation.DefaultConfig(), s)
	if e != nil {
		t.Fatal(e)
	}
	return dashboardHTMLManagerFixture(t, m)
}

func TestWitnessMetadataCounterCountsOnlyRequests(t *testing.T) {
	var actual atomic.Int64
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actual.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "yes"})
	}))
	defer s.Close()
	c := s.Client()
	counter := &witnessRequestCounter{transport: c.Transport}
	c.Transport = counter
	for range 3 {
		r, e := c.Get(s.URL)
		if e != nil {
			t.Fatal(e)
		}
		r.Body.Close()
	}
	if counter.requests.Load() != 3 || actual.Load() != 3 {
		t.Fatal("request counter")
	}
}
