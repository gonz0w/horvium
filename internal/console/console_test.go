// SPDX-License-Identifier: BSD-3-Clause
package console

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gonz0w/horvium/internal/client"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

type fixtureSource struct{}

func (fixtureSource) Collect(context.Context, string) (inventory.Snapshot, error) {
	return fixtureSnapshot(), nil
}
func fixtureSnapshot() inventory.Snapshot {
	now := time.Now().UTC()
	mem := uint64(16 << 30)
	s := inventory.Snapshot{StartedAt: now, FinishedAt: now}
	s.Host = inventory.Section[inventory.Host]{Status: inventory.OK, StartedAt: now, FinishedAt: now, Data: inventory.Host{InstalledKernel: "fixture-kernel", RunningKernel: "fixture-kernel", Userland: "fixture-userland", Architecture: "fixture", PhysicalMemory: &mem, ZFSUserland: "fixture-zfs", ZFSKernel: "fixture-zfs"}}
	s.GEOM = inventory.Section[inventory.GEOM]{Status: inventory.OK, StartedAt: now, FinishedAt: now, Data: inventory.GEOM{}}
	s.Pools = inventory.Section[[]inventory.Pool]{Status: inventory.OK, StartedAt: now, FinishedAt: now, Data: []inventory.Pool{{Name: "vault", GUID: "18446744073709551615", Size: 3000000000000, Allocated: 1000000000000, Free: 2000000000000, Health: "ONLINE"}}}
	s.Datasets = inventory.Section[[]inventory.Dataset]{Status: inventory.OK, StartedAt: now, FinishedAt: now, Data: []inventory.Dataset{{Name: "vault/photos", GUID: "18446744073709551614", Used: 123, Available: 456, Referenced: 789}, {Name: `vault/<img src=x onerror=alert(1)>`, GUID: "21", Used: 10}}}
	return s
}
func fixtureConfig(host string) admin.Config {
	return admin.Config{Version: 1, Listen: host, AllowedHosts: []string{host}, ManagementUID: 65534, SocketGID: 65534, HelperSocket: "/var/run/horvium/inventory.sock", StateDirectory: "/var/db/horvium", MaxBytes: 64 << 20, MaxRecords: 100000, MaxFieldBytes: 4096, MaxXMLDepth: 32, MaxXMLElements: 250000, MaxFrameBytes: 64 << 10, MaxJSONBytes: 64 << 20, MaxTransportRecords: 100000, MaxJSONDepth: 32, IntervalSeconds: 30, TimeoutSeconds: 45, StaleSeconds: 90, MaxReaders: 16, MaxHelperClients: 8}
}
func fixture(t *testing.T) (http.Handler, *observation.Manager, *admin.Auth) {
	return fixtureWithSource(t, fixtureSource{})
}
func fixtureWithSource(t *testing.T, source observation.Source) (http.Handler, *observation.Manager, *admin.Auth) {
	t.Helper()
	m, e := observation.New(observation.DefaultConfig(), source)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); _ = m.Run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	for deadline := time.Now().Add(time.Second); m.State().CurrentID == ""; {
		if time.Now().After(deadline) {
			t.Fatal("fixture collection")
		}
		time.Sleep(time.Millisecond)
	}
	a := admin.NewAuth(sha256.Sum256([]byte(strings.Repeat("a", 64))))
	t.Cleanup(a.Close)
	h, e := New(fixtureConfig("127.0.0.1:8088"), a, m)
	if e != nil {
		t.Fatal(e)
	}
	return h, m, a
}
func req(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Host = "127.0.0.1:8088"
	r.RemoteAddr = "127.0.0.1:50000"
	if method == "POST" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return r
}
func serve(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func login(t *testing.T, h http.Handler) (*http.Cookie, string) {
	t.Helper()
	w := serve(h, req("GET", "/auth/challenge", ""))
	var c admin.Challenge
	_ = json.Unmarshal(w.Body.Bytes(), &c)
	r := req("POST", "/auth/login", url.Values{"token": {strings.Repeat("a", 64)}, "csrf": {c.CSRF}}.Encode())
	if len(w.Result().Cookies()) != 1 {
		t.Fatal("challenge cookie")
	}
	r.AddCookie(w.Result().Cookies()[0])
	w = serve(h, r)
	if w.Code != 200 {
		t.Fatal("login")
	}
	var s admin.Session
	_ = json.Unmarshal(w.Body.Bytes(), &s)
	for _, c := range w.Result().Cookies() {
		if c.Name == admin.SessionCookie {
			return c, s.CSRF
		}
	}
	t.Fatal("session cookie")
	return nil, ""
}
func TestConsoleAuthenticationAndProgressiveLogin(t *testing.T) {
	h, _, _ := fixture(t)
	w := serve(h, req("GET", "/", ""))
	if w.Code != 303 || w.Header().Get("Location") != "/login" {
		t.Fatal("anonymous full page must reach login")
	}
	w = serve(h, req("GET", "/login", ""))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `type="password"`) || !strings.Contains(w.Body.String(), `name="csrf"`) {
		t.Fatal("semantic login form")
	}
	for _, path := range []string{"/api/overview", "/api/pools", "/api/providers", "/api/datasets"} {
		if serve(h, req("GET", path, "")).Code != 401 {
			t.Fatal("API leaked inventory")
		}
	}
	cookie, _ := login(t, h)
	r := req("GET", "/", "")
	r.AddCookie(cookie)
	w = serve(h, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "fixture-kernel") || !strings.Contains(w.Body.String(), "Not collected") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("honest authenticated overview")
	}
}
func TestHTMLFragmentJSONParityAndEscaping(t *testing.T) {
	h, m, _ := fixture(t)
	cookie, _ := login(t, h)
	id := m.State().CurrentID
	query := url.Values{"generation": {id}, "filter": {"photos"}, "limit": {"1"}, "sort": {"guid"}, "selected": {"18446744073709551614"}}.Encode()
	r := req("GET", "/storage/datasets?"+query, "")
	r.AddCookie(cookie)
	w := serve(h, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), id) || !strings.Contains(w.Body.String(), "18446744073709551614") || !strings.Contains(w.Body.String(), "vault/photos") {
		t.Fatal("full generation/detail view")
	}
	r.Header.Set("HX-Request", "true")
	fragment := serve(h, r)
	if fragment.Code != 200 || strings.Contains(fragment.Body.String(), "<!doctype") || !strings.Contains(fragment.Body.String(), `id="workspace"`) {
		t.Fatal("progressive fragment")
	}
	q := url.Values{"generation": {id}, "filter": {"photos"}, "limit": {"1"}, "sort": {"guid"}}.Encode()
	r = req("GET", "/api/datasets?"+q, "")
	r.AddCookie(cookie)
	w = serve(h, r)
	var page observation.Page[inventory.Dataset]
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || page.Generation != id || len(page.Rows) != 1 || page.Rows[0].GUID != "18446744073709551614" {
		t.Fatal("CLI transport lost exact generation/GUID")
	}
	r = req("GET", "/storage/datasets", "")
	r.AddCookie(cookie)
	w = serve(h, r)
	if strings.Contains(w.Body.String(), `<img src=x`) || !strings.Contains(w.Body.String(), "&lt;img") {
		t.Fatal("observed name rendered as HTML")
	}
}
func TestRoutesStrictQueriesAndGenerationExpiry(t *testing.T) {
	h, _, _ := fixture(t)
	cookie, csrf := login(t, h)
	for _, path := range []string{"/api/pools?limit=1&limit=2", "/api/pools?future=1", "/api/pools?offset=-1", "/storage/pools?selected=missing", "/storage/pools?generation=expired"} {
		r := req("GET", path, "")
		r.AddCookie(cookie)
		w := serve(h, r)
		want := 400
		if strings.Contains(path, "expired") {
			want = 409
		}
		if w.Code != want {
			t.Fatalf("query denial status %d want %d", w.Code, want)
		}
	}
	r := req("GET", "/refresh", "")
	r.AddCookie(cookie)
	if serve(h, r).Code != 405 {
		t.Fatal("GET refresh mutation")
	}
	r = req("POST", "/refresh", "csrf=incorrect")
	r.AddCookie(cookie)
	if serve(h, r).Code != 403 {
		t.Fatal("refresh CSRF bypass")
	}
	r = req("POST", "/api/refresh", url.Values{"csrf": {csrf}}.Encode())
	r.AddCookie(cookie)
	w := serve(h, r)
	if w.Code != 202 && w.Code != 409 {
		t.Fatal("typed refresh outcome")
	}
	r = req("GET", "/assets/console.css", "")
	r.Host = "evil.example:8088"
	if serve(h, r).Code != 403 {
		t.Fatal("asset rebinding bypass")
	}
	w = serve(h, req("GET", "/assets/htmx.min.js", ""))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "2.0.11") {
		t.Fatal("pinned local HTMX")
	}
}
func TestInitialUnavailableDoesNotInventMetrics(t *testing.T) {
	m, e := observation.New(observation.DefaultConfig(), fixtureSource{})
	if e != nil {
		t.Fatal(e)
	}
	a := admin.NewAuth(sha256.Sum256([]byte(strings.Repeat("a", 64))))
	defer a.Close()
	h, e := New(fixtureConfig("127.0.0.1:8088"), a, m)
	if e != nil {
		t.Fatal(e)
	}
	cookie, _ := login(t, h)
	r := req("GET", "/", "")
	r.AddCookie(cookie)
	w := serve(h, r)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "Inventory unavailable") || strings.Contains(w.Body.String(), "fixture-kernel") {
		t.Fatal("startup invented native data")
	}
}
func TestBrowserPreview(t *testing.T) {
	if os.Getenv("HORVIUM_UI_PREVIEW") != "1" {
		t.Skip("explicit synthetic browser fixture only")
	}
	h, m, _ := fixture(t)
	_ = h
	s := httptest.NewUnstartedServer(nil)
	defer s.Close()
	a := admin.NewAuth(sha256.Sum256([]byte(strings.Repeat("a", 64))))
	defer a.Close()
	handler, e := New(fixtureConfig(s.Listener.Addr().String()), a, m)
	if e != nil {
		t.Fatal(e)
	}
	s.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded := serve(handler, r)
		for key, vs := range recorded.Header() {
			w.Header()[key] = vs
		}
		body := recorded.Body.String()
		if strings.Contains(recorded.Header().Get("Content-Type"), "text/html") {
			banner := `<p class="notice">Synthetic browser fixture · These values are not a live FreeBSD host.</p>`
			body = strings.Replace(body, `<main id="workspace" tabindex="-1">`, `<main id="workspace" tabindex="-1">`+banner, 1)
			body = strings.Replace(body, `<main class="login-panel">`, `<main class="login-panel">`+banner, 1)
		}
		w.WriteHeader(recorded.Code)
		_, _ = w.Write([]byte(body))
	})
	s.Start()
	fmt.Println("Synthetic browser fixture:", s.URL)
	<-time.After(20 * time.Minute)
}

func TestAuthenticatedClientParity(t *testing.T) {
	_, m, _ := fixture(t)
	s := httptest.NewUnstartedServer(nil)
	defer s.Close()
	a := admin.NewAuth(sha256.Sum256([]byte(strings.Repeat("a", 64))))
	defer a.Close()
	h, e := New(fixtureConfig(s.Listener.Addr().String()), a, m)
	if e != nil {
		t.Fatal(e)
	}
	s.Config.Handler = h
	s.Start()
	c, e := client.New(s.URL)
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	if _, e = c.Query(ctx, "overview", observation.Query{}); e == nil {
		t.Fatal("anonymous client bypass")
	}
	if e = c.Login(ctx, strings.Repeat("a", 64)); e != nil {
		t.Fatal(e)
	}
	id := m.State().CurrentID
	value, e := c.Query(ctx, "datasets", observation.Query{Generation: id, Filter: "photos", Sort: "guid", Limit: 1})
	if e != nil {
		t.Fatal(e)
	}
	p, ok := value.(observation.Page[inventory.Dataset])
	if !ok || p.Generation != id || len(p.Rows) != 1 || p.Rows[0].GUID != "18446744073709551614" {
		t.Fatal("typed CLI parity")
	}
	if _, e = c.Query(ctx, "datasets", observation.Query{Generation: "expired"}); !errors.Is(e, observation.ErrExpired) {
		t.Fatal("expiry hidden")
	}
	if e = c.Logout(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e = c.Query(ctx, "overview", observation.Query{}); e == nil {
		t.Fatal("logout retained authorization")
	}
}

func TestPinnedHTMX(t *testing.T) {
	b, e := assets.ReadFile("assets/htmx.min.js")
	if e != nil {
		t.Fatal(e)
	}
	hash := sha256.Sum256(b)
	if hex.EncodeToString(hash[:]) != "7f14135fe8ee3ed5f1a85fea21ba4dfba8236d697c4f95ab575d1c9091bb06ca" {
		t.Fatal("HTMX pin changed without provenance review")
	}
}
func TestHTMLLoginLogoutAndExpiredFragment(t *testing.T) {
	h, _, a := fixture(t)
	w := serve(h, req("GET", "/auth/challenge", ""))
	var c admin.Challenge
	if json.Unmarshal(w.Body.Bytes(), &c) != nil {
		t.Fatal("challenge")
	}
	r := req("POST", "/login", url.Values{"token": {strings.Repeat("a", 64)}, "csrf": {c.CSRF}}.Encode())
	r.AddCookie(w.Result().Cookies()[0])
	w = serve(h, r)
	if w.Code != 303 || w.Header().Get("Location") != "/" {
		t.Fatal("HTML login failed")
	}
	var cookie *http.Cookie
	for _, v := range w.Result().Cookies() {
		if v.Name == admin.SessionCookie {
			cookie = v
		}
	}
	if cookie == nil {
		t.Fatal("session cookie")
	}
	session, e := a.Session(cookie.Value, false)
	if e != nil {
		t.Fatal(e)
	}
	r = req("POST", "/logout", url.Values{"csrf": {session.CSRF}}.Encode())
	r.AddCookie(cookie)
	w = serve(h, r)
	if w.Code != 303 || w.Header().Get("Location") != "/login" {
		t.Fatal("HTML logout failed")
	}
	r = req("GET", "/storage/pools", "")
	r.AddCookie(cookie)
	r.Header.Set("HX-Request", "true")
	w = serve(h, r)
	if w.Code != 401 || w.Header().Get("HX-Redirect") != "/login" {
		t.Fatal("expired fragment must reach full login")
	}
}

func TestBackgroundMissingAssetCannotReplaceLoginChallenge(t *testing.T) {
	h, _, _ := fixture(t)
	w := serve(h, req("GET", "/favicon.ico", ""))
	if w.Code != 404 || len(w.Result().Cookies()) != 0 || w.Header().Get("Location") != "" {
		t.Fatal("background asset redirected through challenge login")
	}
}
func TestLocalFormsRetainOriginAndNullOriginIsDenied(t *testing.T) {
	h, _, _ := fixture(t)
	w := serve(h, req("GET", "/login", ""))
	if w.Header().Get("Referrer-Policy") != "same-origin" {
		t.Fatal("no-referrer suppresses Origin on normal HTML form posts")
	}
	r := req("POST", "/login", "csrf=unknown&token=unknown")
	r.Header.Set("Origin", "null")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	if serve(h, r).Code != 403 {
		t.Fatal("opaque Origin must remain denied")
	}
}
func TestOversizedViewsReturnNoPartialInventory(t *testing.T) {
	h, m, a := fixture(t)
	_ = h
	long := strings.Repeat("x", 32<<20)
	jsonResponse := httptest.NewRecorder()
	writeJSON(jsonResponse, map[string]string{"observed": long}, 200)
	if jsonResponse.Code != 503 || strings.Contains(jsonResponse.Body.String(), "xxxxx") {
		t.Fatal("oversized JSON published partial inventory")
	}
	application := &app{manager: m}
	application.gate, _ = admin.NewGate(fixtureConfig("127.0.0.1:8088"), a)
	application.templates = template.Must(template.New("fixture").Parse(`{{define "fixture"}}sensitive prefix {{.Error}}{{end}}`))
	htmlResponse := httptest.NewRecorder()
	application.render(htmlResponse, "fixture", view{Error: long}, 200)
	if htmlResponse.Code != 503 || strings.Contains(htmlResponse.Body.String(), "sensitive prefix") {
		t.Fatal("oversized HTML published partial inventory")
	}
}

type failedPoolSource struct{}

func (failedPoolSource) Collect(context.Context, string) (inventory.Snapshot, error) {
	s := fixtureSnapshot()
	s.Pools.Status = inventory.Unavailable
	s.Pools.Data = nil
	s.Pools.Problems = []inventory.Problem{{Source: "pools", Code: "permission"}}
	return s, nil
}
func TestFailedSectionDoesNotReportSuccessfulZero(t *testing.T) {
	h, _, _ := fixtureWithSource(t, failedPoolSource{})
	cookie, _ := login(t, h)
	r := req("GET", "/storage/pools", "")
	r.AddCookie(cookie)
	w := serve(h, r)
	if w.Code != 200 || strings.Contains(w.Body.String(), "0 matching items") || !strings.Contains(w.Body.String(), "Resource count unavailable") {
		t.Fatal("failed section presented a successful zero count")
	}
}
