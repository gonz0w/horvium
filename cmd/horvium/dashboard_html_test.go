// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/console"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

type dashboardHTMLSource struct{ snapshot inventory.Snapshot }

func (s dashboardHTMLSource) Collect(context.Context, string) (inventory.Snapshot, error) {
	return s.snapshot, nil
}

func dashboardHTMLFixture(t *testing.T, s inventory.Snapshot) (http.Handler, *observation.Manager, *http.Cookie) {
	t.Helper()
	owned, err := inventory.CloneBounded(s, inventory.DefaultLimits())
	if err != nil {
		t.Fatal("invalid fixture inventory", err)
	}
	m, err := observation.New(observation.DefaultConfig(), dashboardHTMLSource{owned})
	if err != nil {
		t.Fatal(err)
	}
	return dashboardHTMLManagerFixture(t, m)
}

func dashboardHTMLManagerFixture(t *testing.T, m *observation.Manager) (http.Handler, *observation.Manager, *http.Cookie) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); _ = m.Run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	dashboardHTMLWait(t, func() bool { return m.State().CurrentID != "" })
	auth := admin.NewAuth(sha256.Sum256([]byte(strings.Repeat("a", 64))))
	t.Cleanup(auth.Close)
	host := "127.0.0.1:8088"
	c := admin.Config{Version: 1, Listen: host, AllowedHosts: []string{host}, ManagementUID: 65534, SocketGID: 65534, HelperSocket: "/var/run/horvium/inventory.sock", StateDirectory: "/var/db/horvium", MaxBytes: 64 << 20, MaxRecords: 100000, MaxFieldBytes: 4096, MaxXMLDepth: 32, MaxXMLElements: 250000, MaxFrameBytes: 64 << 10, MaxJSONBytes: 64 << 20, MaxTransportRecords: 100000, MaxJSONDepth: 32, IntervalSeconds: 30, TimeoutSeconds: 45, StaleSeconds: 90, MaxReaders: 16, MaxHelperClients: 8}
	h, err := console.New(c, auth, m)
	if err != nil {
		t.Fatal(err)
	}
	challenge := dashboardHTMLRequest(h, nil, "GET", "/auth/challenge", "")
	var ch admin.Challenge
	if json.Unmarshal(challenge.Body.Bytes(), &ch) != nil || len(challenge.Result().Cookies()) != 1 {
		t.Fatal("challenge")
	}
	login := dashboardHTMLRequest(h, challenge.Result().Cookies()[0], "POST", "/auth/login", url.Values{"token": {strings.Repeat("a", 64)}, "csrf": {ch.CSRF}}.Encode())
	if login.Code != 200 {
		t.Fatal("login", login.Code)
	}
	for _, cookie := range login.Result().Cookies() {
		if cookie.Name == admin.SessionCookie {
			return h, m, cookie
		}
	}
	t.Fatal("missing session")
	return nil, nil, nil
}
func dashboardHTMLRequest(h http.Handler, c *http.Cookie, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Host = "127.0.0.1:8088"
	r.RemoteAddr = "127.0.0.1:50000"
	if method == "POST" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if c != nil {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func dashboardHTMLWait(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatal("fixture timeout")
		}
		time.Sleep(time.Millisecond)
	}
}
func dashboardRichSnapshot() inventory.Snapshot {
	s := performanceSnapshot(610, 0)
	s.Pools.Data = nil
	for i := 0; i < 610; i++ {
		p := inventory.Pool{Name: fmt.Sprintf("pool-%03d", i), GUID: strconv.FormatUint(9007199254740993+uint64(i), 10), Size: 1000, Allocated: 999, Free: 700, Health: "ONLINE"}
		if i == 0 {
			p.Name += ` <img src=x> & "quoted"`
			p.Size = ^uint64(0)
			p.Allocated = p.Size - 1
			p.Free = p.Size
		}
		if i == 1 {
			p.Size = 0
			p.Allocated = 0
			p.Free = 0
		}
		s.Pools.Data = append(s.Pools.Data, p)
	}
	s.Datasets.Data[0].Name += ` <script> & 'quoted'`
	s.Datasets.Data[0].GUID = "18446744073709551615"
	s.Datasets.Data[0].Available = 9007199254740993
	s.Datasets.Data[0].Referenced = ^uint64(0)
	s.Datasets.Data[88].Used = ^uint64(0) // Outside the first twenty comparisons.
	return s
}

func TestDashboardHTMLValidatesActualFiveCasePages(t *testing.T) {
	s := dashboardRichSnapshot()
	plan, _ := dashboardPlan(len(s.Datasets.Data))
	predicate, err := newDashboardHTMLPredicate(s, plan)
	if err != nil {
		t.Fatal(err)
	}
	// Preparation owns a detached copy, not mutable source slices/pointers.
	h, _, cookie := dashboardHTMLFixture(t, s)
	s.Pools.Data[0].Size = 123
	s.Datasets.Data[0].Used = 999
	for _, i := range []int{0, 1, 2, 3, 4, 8, 9, 23, 24, 198, 199, 998, 999} {
		t.Run(fmt.Sprintf("%d-%s", i, plan[i].Case), func(t *testing.T) {
			response := dashboardHTMLRequest(h, cookie, "GET", plan[i].Path, "")
			if response.Code != 200 {
				t.Fatal("actual view failed", response.Code)
			}
			if err := predicate(plan[i], response.Code, response.Body.Bytes()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func dashboardHTMLMutateRegion(t *testing.T, body, begin, end string, change func(string) string) string {
	t.Helper()
	a := strings.Index(body, begin)
	if a < 0 {
		t.Fatal("mutation region absent", begin)
	}
	b := strings.Index(body[a:], end)
	if b < 0 {
		t.Fatal("mutation region not closed")
	}
	b += a + len(end)
	result := body[:a] + change(body[a:b]) + body[b:]
	if result == body {
		t.Fatal("mutation had no effect")
	}
	return result
}
func dashboardHTMLReplaceMatch(pattern, replacement string) func(string) string {
	return func(s string) string {
		re := regexp.MustCompile(pattern)
		m := re.FindStringIndex(s)
		if m == nil {
			return s
		}
		return s[:m[0]] + re.ReplaceAllString(s[m[0]:m[1]], replacement) + s[m[1]:]
	}
}

func TestDashboardHTMLRejectsSemanticResponseMutations(t *testing.T) {
	s := dashboardRichSnapshot()
	plan, _ := dashboardPlan(len(s.Datasets.Data))
	predicate, err := newDashboardHTMLPredicate(s, plan)
	if err != nil {
		t.Fatal(err)
	}
	h, _, cookie := dashboardHTMLFixture(t, s)
	for i := 0; i < 5; i++ {
		t.Run(plan[i].Case, func(t *testing.T) {
			response := dashboardHTMLRequest(h, cookie, "GET", plan[i].Path, "")
			body := response.Body.String()
			if err := predicate(plan[i], 200, []byte(body)); err != nil {
				t.Fatal("control", err)
			}
			mutations := map[string]func(string) string{
				"truncated-document": func(s string) string { return strings.TrimSuffix(s, "</html>") },
				"wrong-heading": func(s string) string {
					return regexp.MustCompile(`<h1>[^<]*</h1>`).ReplaceAllString(s, "<h1>Wrong view</h1>")
				},
				"wrong-generation": dashboardHTMLReplaceMatch(`(<p class="generation">Observation <code>)[^<]*(</code>)`, "${1}other-generation${2}"),
				"missing-capacity": func(s string) string {
					return dashboardHTMLMutateRegion(t, s, `<section class="capacity-section"`, `</section>`, func(string) string { return "" })
				},
				"missing-comparison": func(s string) string {
					return dashboardHTMLMutateRegion(t, s, `<ol class="capacity-list">`, `</ol>`, dashboardHTMLReplaceMatch(`(?s)<li>.*?</li>`, ""))
				},
				"duplicate-comparison": func(s string) string {
					return dashboardHTMLMutateRegion(t, s, `<ol class="capacity-list">`, `</ol>`, dashboardHTMLReplaceMatch(`(?s)<li>.*?</li>`, "$0$0"))
				},
				"wrong-comparison-name": dashboardHTMLReplaceMatch(`(<a id="capacity-[^"]*"[^>]*>)[^<]*(</a>)`, "${1}wrong original name${2}"),
				"wrong-link-generation": dashboardHTMLReplaceMatch(`(id="capacity-[^"]*" href=")[^"]*(")`, "${1}/storage/pools?generation=wrong${2}"),
				"wrong-hx-link":         dashboardHTMLReplaceMatch(`(id="capacity-[^"]*"[^>]*hx-get=")[^"]*(")`, "${1}/storage/pools?generation=wrong${2}"),
				"wrong-coverage":        func(s string) string { return strings.Replace(s, "610 observed pools", "609 observed pools", 1) },
				"wrong-query":           func(s string) string { return strings.Replace(s, "Sort: name", "Sort: used", 1) },
				"wrong-status": func(s string) string {
					return strings.Replace(s, `<span class="badge">Observed</span>`, `<span class="badge">unavailable</span>`, 1)
				},
				"wrong-coordinate":   dashboardHTMLReplaceMatch(`(<rect class="capacity-(?:used|allocated)"[^>]*width=")[^"]*(")`, "${1}5000${2}"),
				"stacked-coordinate": dashboardHTMLReplaceMatch(`(<rect class="capacity-(?:used|free)" x=")[^"]*(")`, "${1}50${2}"),
			}
			mutations["wrong-freshness-time"] = dashboardHTMLReplaceMatch(`(Section read )[0-9:-]+ [0-9:]+ UTC(; last successful read )`, "${1}2020-01-01 00:00:00 UTC${2}")
			mutations["wrong-svg-viewport"] = func(s string) string { return strings.Replace(s, `viewBox="0 0 1000`, `viewBox="0 0 2000`, 1) }
			mutations["wrong-track"] = dashboardHTMLReplaceMatch(`(<rect class="capacity-track"[^>]*width=")[^"]*(")`, "${1}1${2}")
			mutations["extra-table"] = func(s string) string {
				return strings.Replace(s, "</main>", "<table><caption>Extra</caption></table></main>", 1)
			}
			if i == 0 {
				mutations["wrong-host-fact"] = func(s string) string { return strings.Replace(s, "synthetic-performance", "wrong-kernel", 1) }
				mutations["wrong-section-count"] = dashboardHTMLReplaceMatch(`(<td>)610(</td>)`, "${1}609${2}")
				mutations["wrong-section-link"] = dashboardHTMLReplaceMatch(`(href="/storage/pools\?generation=)[^"]*(")`, "${1}other-generation${2}")
			} else {
				mutations["wrong-form-generation"] = dashboardHTMLReplaceMatch(`(name="generation" value=")[^"]*(")`, "${1}other-generation${2}")
				mutations["wrong-form-limit"] = dashboardHTMLReplaceMatch(`(id="limit"[^>]*value=")[^"]*(")`, "${1}20${2}")
				mutations["selected-row"] = dashboardHTMLReplaceMatch(`(<tbody><tr)\s*(>)`, `${1} class="selected"${2}`)
				mutations["wrong-page-link"] = dashboardHTMLReplaceMatch(`(<a href=")[^"]*("[^>]*>Next page)`, "${1}/storage/pools?generation=wrong${2}")
			}
			if i == 1 || i == 2 {
				mutations["wrong-coverage"] = func(s string) string { return strings.Replace(s, "610 matching pools", "609 matching pools", 1) }
			}
			if i >= 3 {
				delete(mutations, "wrong-coverage")
				mutations["wrong-coverage"] = func(s string) string { return strings.Replace(s, "610 matching datasets", "609 matching datasets", 1) }
				mutations["wrong-scale"] = func(s string) string {
					return strings.Replace(s, "largest Used on this page · 18446744073709551615 B", "largest Used on this page · 9007199254740992 B", 1)
				}
			}
			caption := "<caption>Pools in this observation"
			if i == 0 {
				caption = "<caption>Pool capacity in this observation"
			} else if i >= 3 {
				caption = "<caption>Datasets in this observation"
			}
			for name, change := range map[string]func(string) string{
				"missing-table-row":   dashboardHTMLReplaceMatch(`(?s)<tbody><tr.*?</tr>`, "<tbody>"),
				"duplicate-table-row": dashboardHTMLReplaceMatch(`(?s)<tbody>(<tr.*?</tr>)`, "<tbody>$1$1"),
				"wrong-table-guid":    dashboardHTMLReplaceMatch(`(<td class="mono">)[^<]*(</td>)`, "${1}9007199254740992${2}"),
				"rounded-byte-cell":   dashboardHTMLReplaceMatch(`(<td class="mono">)[0-9]+ B(</td>)`, "${1}9007199254740992 B${2}"),
				"wrong-table-name":    dashboardHTMLReplaceMatch(`(<a[^>]*>)[^<]*(</a>)`, "${1}wrong original name${2}"),
				"missing-table-end":   func(s string) string { return strings.Replace(s, "</table>", "", 1) },
			} {
				mutations[name] = func(s string) string { return dashboardHTMLMutateRegion(t, s, caption, "</table>", change) }
			}
			for name, change := range mutations {
				t.Run(name, func(t *testing.T) {
					changed := change(body)
					if changed == body {
						t.Fatal("mutation had no effect")
					}
					if err := predicate(plan[i], 200, []byte(changed)); err == nil {
						t.Fatal("semantic mutation qualified")
					}
				})
			}
			for _, status := range []int{303, 401, 409, 500} {
				if err := predicate(plan[i], status, []byte(body)); err == nil {
					t.Fatal("unexpected status qualified")
				}
			}
		})
	}
}

func TestDashboardHTMLHonestEmptyUnavailableZeroAndStale(t *testing.T) {
	for _, mode := range []string{"empty", "zero", "unavailable", "host-unavailable", "stale"} {
		t.Run(mode, func(t *testing.T) {
			s := performanceSnapshot(1, 0)
			switch mode {
			case "empty":
				s.Pools.Data = nil
				s.Datasets.Data = nil
			case "zero":
				s.Pools.Data[0].Size = 0
				s.Pools.Data[0].Free = 0
				s.Datasets.Data[0].Used = 0
			case "unavailable":
				s.Pools.Status = inventory.Unavailable
				s.Datasets.Status = inventory.Unavailable
				s.Pools.Data = nil
				s.Datasets.Data = nil
				s.Pools.Problems = []inventory.Problem{{Source: "pools", Code: "permission"}}
				s.Datasets.Problems = []inventory.Problem{{Source: "datasets", Code: "permission"}}
			case "host-unavailable":
				s.Host.Status = inventory.Unavailable
				s.Host.Data = inventory.Host{}
				s.Host.Problems = []inventory.Problem{{Source: "freebsd-version", Code: "permission"}}
			case "stale":
				old := s.StartedAt.Add(-2 * time.Minute)
				s.StartedAt = old
				s.FinishedAt = old
				s.Host.StartedAt = old
				s.Host.FinishedAt = old
				s.GEOM.StartedAt = old
				s.GEOM.FinishedAt = old
				s.Pools.StartedAt = old
				s.Pools.FinishedAt = old
				s.Datasets.StartedAt = old
				s.Datasets.FinishedAt = old
			}
			plan, _ := dashboardPlan(max(1, len(s.Datasets.Data)))
			predicate, err := newDashboardHTMLPredicate(s, plan)
			if err != nil {
				t.Fatal(err)
			}
			h, _, cookie := dashboardHTMLFixture(t, s)
			for i := 0; i < 5; i++ {
				response := dashboardHTMLRequest(h, cookie, "GET", plan[i].Path, "")
				if err := predicate(plan[i], response.Code, response.Body.Bytes()); err != nil {
					t.Fatal(plan[i].Case, err)
				}
			}
		})
	}
}

func TestDashboardHTMLRequiresSafeActualExpiry(t *testing.T) {
	s := performanceSnapshot(100, 0)
	plan, _ := dashboardPlan(100)
	predicate, err := newDashboardHTMLPredicate(s, plan)
	if err != nil {
		t.Fatal(err)
	}
	h, m, cookie := dashboardHTMLFixture(t, s)
	old := m.State().CurrentID
	for i := 0; i < 2; i++ {
		previous := m.State().CurrentID
		if err := m.Refresh(); err != nil {
			t.Fatal(err)
		}
		dashboardHTMLWait(t, func() bool { return m.State().CurrentID != previous })
	}
	expired := dashboardHTMLRequest(h, cookie, "GET", "/?generation="+old, "")
	if expired.Code != 409 || predicate(plan[0], 409, expired.Body.Bytes()) != nil {
		t.Fatal("actual expiry denied")
	}
	body := expired.Body.String()
	for _, change := range []func(string) string{
		func(s string) string { return strings.Replace(s, "generation_expired", "busy", 1) },
		func(s string) string {
			return strings.Replace(s, `href="/">Reload current observation`, `href="/storage/pools">Reload current observation`, 1)
		},
		func(s string) string {
			return strings.Replace(s, "</main>", `<section class="capacity-section">successful data</section></main>`, 1)
		},
		func(s string) string {
			return strings.Replace(s, "</main>", `<section><h2>Host facts</h2><dl><dt>Host</dt><dd>data</dd></dl></section></main>`, 1)
		},
		func(s string) string {
			return strings.Replace(s, "</main>", `<table><tbody><tr><td>data</td></tr></tbody></table></main>`, 1)
		},
	} {
		if err := predicate(plan[0], 409, []byte(change(body))); err == nil {
			t.Fatal("unsafe expiry qualified")
		}
	}
	if err := predicate(plan[1], 409, []byte(body)); err == nil {
		t.Fatal("pool expiry qualified")
	}
	login := dashboardHTMLRequest(h, nil, "GET", "/login", "")
	if err := predicate(plan[0], 200, login.Body.Bytes()); err == nil {
		t.Fatal("login body qualified")
	}
}

func TestDashboardHTMLPreparationAndParserFailClosed(t *testing.T) {
	s := performanceSnapshot(100, 0)
	plan, _ := dashboardPlan(100)
	broken := s
	broken.Datasets.Data = append([]inventory.Dataset(nil), s.Datasets.Data...)
	broken.Datasets.Data[1].GUID = broken.Datasets.Data[0].GUID
	if _, err := newDashboardHTMLPredicate(broken, plan); err == nil {
		t.Fatal("invalid original inventory accepted")
	}
	if _, err := newDashboardHTMLPredicate(s, plan[:999]); err == nil {
		t.Fatal("incomplete plan accepted")
	}
	wrong := append([]dashboardRequest(nil), plan...)
	wrong[3].Path = "/storage/datasets?offset=1&sort=name"
	if _, err := newDashboardHTMLPredicate(s, wrong); err == nil {
		t.Fatal("wrong plan accepted")
	}
	predicate, err := newDashboardHTMLPredicate(s, plan)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dashboardParseHTML([]byte(`<!doctype html><html lang="en"><fake:main id="workspace"></fake:main></html>`)); err == nil {
		t.Fatal("prefixed element interpreted as HTML")
	}
	for _, body := range []string{"", `<!doctype html><html><body><main></body></main></html>`, `<!doctype html><html id="x" id="y"></html>`, `<!doctype html><html lang="en">` + strings.Repeat("<div>", 65) + strings.Repeat("</div>", 65) + `</html>`, `<!doctype html><html lang="en">` + strings.Repeat("<span></span>", 20001) + `</html>`, strings.Repeat("x", (32<<20)+1)} {
		if err := predicate(plan[0], 200, []byte(body)); err == nil {
			t.Fatal("malformed/unbounded document accepted")
		}
	}
}

func TestDashboardHTMLPredicateRunsThroughReviewedDriver(t *testing.T) {
	s := performanceSnapshot(100, 0)
	plan, _ := dashboardPlan(100)
	predicate, err := newDashboardHTMLPredicate(s, plan)
	if err != nil {
		t.Fatal(err)
	}
	h, _, cookie := dashboardHTMLFixture(t, s)
	var requests, apiRequests atomic.Int32
	client := &http.Client{Transport: dashboardTransport(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		if strings.HasPrefix(request.URL.Path, "/api/") {
			apiRequests.Add(1)
		}
		r := httptest.NewRequest(request.Method, request.URL.RequestURI(), nil).WithContext(request.Context())
		r.Header = request.Header.Clone()
		r.Host = "127.0.0.1:8088"
		r.RemoteAddr = "127.0.0.1:50000"
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Result(), nil
	})}
	samples, err := dashboardLoad(client, "http://127.0.0.1:8088", 100, plan, "initial", predicate)
	if err != nil {
		t.Fatal("actual console workload denied", err)
	}
	summary, err := dashboardSummarize(samples, "initial")
	if err != nil || summary.Success != 1000 || requests.Load() != 1000 || apiRequests.Load() != 0 {
		t.Fatal("unexpected workload", summary, err)
	}
	for _, name := range dashboardCases {
		if summary.Cases[name].Success != 200 {
			t.Fatal("case not completed", name)
		}
	}
}

func TestDashboardHTMLParserEnforcesActualBoundaries(t *testing.T) {
	for _, spec := range []struct {
		name      string
		good, bad string
	}{
		{"nodes", `<!doctype html><html lang="en">` + strings.Repeat("<span></span>", 19999) + `</html>`, `<!doctype html><html lang="en">` + strings.Repeat("<span></span>", 20000) + `</html>`},
		{"depth", `<!doctype html><html lang="en">` + strings.Repeat("<div>", 63) + strings.Repeat("</div>", 63) + `</html>`, `<!doctype html><html lang="en">` + strings.Repeat("<div>", 64) + strings.Repeat("</div>", 64) + `</html>`},
		{"attribute", `<!doctype html><html lang="en" data-x="` + strings.Repeat("x", 8192) + `"></html>`, `<!doctype html><html lang="en" data-x="` + strings.Repeat("x", 8193) + `"></html>`},
	} {
		t.Run(spec.name, func(t *testing.T) {
			if _, err := dashboardParseHTML([]byte(spec.good)); err != nil {
				t.Fatal("boundary rejected", err)
			}
			if _, err := dashboardParseHTML([]byte(spec.bad)); err == nil {
				t.Fatal("limit+1 accepted")
			}
		})
	}
}
