// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func sourceFixture(i int) string { return fmt.Sprintf("127.0.1.%d", i+1) }
func request(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Host = "localhost:8080"
	r.RemoteAddr = "127.0.0.1:12345"
	if method == "POST" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return r
}
func exchangeHTTP(t *testing.T, g *Gate, token string) (*http.Cookie, string) {
	t.Helper()
	w := httptest.NewRecorder()
	g.Challenge(w, request("GET", "/challenge", ""))
	if w.Code != 200 {
		t.Fatal("challenge refused")
	}
	var c Challenge
	if json.Unmarshal(w.Body.Bytes(), &c) != nil {
		t.Fatal("challenge payload")
	}
	r := request("POST", "/login", url.Values{"token": {token}, "csrf": {c.CSRF}}.Encode())
	r.AddCookie(w.Result().Cookies()[0])
	w = httptest.NewRecorder()
	g.Login(w, r)
	if w.Code != 200 {
		t.Fatal("login refused")
	}
	var s Session
	_ = json.Unmarshal(w.Body.Bytes(), &s)
	for _, c := range w.Result().Cookies() {
		if c.Name == SessionCookie {
			return c, s.CSRF
		}
	}
	t.Fatal("missing session")
	return nil, ""
}
func TestHTTPAuthenticationPrecedesInventoryAndFragments(t *testing.T) {
	a, token, _ := newTestAuth(t)
	g, e := NewGate(testConfig(), a)
	if e != nil {
		t.Fatal(e)
	}
	count := 0
	h := g.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if RequestCSRF(r) == "" {
			t.Error("missing explicit session contract")
		}
		w.WriteHeader(204)
	}))
	for _, path := range []string{"/", "/fragment", "/health"} {
		r := request("GET", path, "")
		r.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatal("unauthenticated inventory")
		}
	}
	if count != 0 {
		t.Fatal("lookup before authentication")
	}
	cookie, csrf := exchangeHTTP(t, g, token)
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.Domain != "" || cookie.Secure {
		t.Fatal("unsafe loopback cookie")
	}
	r := request("GET", "/fragment", "")
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 || count != 1 || w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatal("protected lookup/headers")
	}
	r = request("POST", "/refresh", url.Values{"csrf": {csrf}}.Encode())
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal("CLI missing provenance plus valid csrf refused")
	}
	r = request("POST", "/logout", url.Values{"csrf": {csrf}}.Encode())
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	g.Logout(w, r)
	if w.Code != 204 {
		t.Fatal("logout")
	}
	r = request("GET", "/", "")
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("session survived logout")
	}
}
func TestHTTPHostOriginAndCSRFDenials(t *testing.T) {
	a, token, _ := newTestAuth(t)
	g, _ := NewGate(testConfig(), a)
	cookie, csrf := exchangeHTTP(t, g, token)
	count := 0
	h := g.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { count++; w.WriteHeader(204) }))
	for _, edit := range []func(*http.Request){
		func(r *http.Request) { r.Host = "evil.example:8080" }, func(r *http.Request) { r.Host = "LOCALHOST:8080" }, func(r *http.Request) { r.Header.Set("Origin", "http://evil.example") }, func(r *http.Request) { r.Header.Set("Origin", "https://localhost:8080") }, func(r *http.Request) { r.Header.Set("Origin", "http://localhost:8080/") }, func(r *http.Request) { r.Header.Set("Origin", "null") }, func(r *http.Request) {
			r.Header.Set("Origin", "http://evil.example")
			r.Header.Set("Sec-Fetch-Site", "same-origin")
		}, func(r *http.Request) {
			r.Header.Add("Origin", "http://localhost:8080")
			r.Header.Add("Origin", "http://localhost:8080")
		}, func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") },
		func(r *http.Request) { r.Body = http.NoBody; r.ContentLength = 0 }, func(r *http.Request) { r.Body = stringBody("csrf=incorrect") }, func(r *http.Request) { r.Body = stringBody(url.Values{"csrf": {csrf, csrf}}.Encode()) }, func(r *http.Request) { r.AddCookie(cookie) }, func(r *http.Request) { r.Method = "DELETE" }, func(r *http.Request) { r.Header.Set("Content-Type", "application/json") }, func(r *http.Request) { r.Body = stringBody(strings.Repeat("x", MaxBodyBytes+1)) }, func(r *http.Request) { r.Header.Set("X-Oversized", strings.Repeat("x", MaxHeaderBytes)) },
	} {
		r := request("POST", "/refresh", url.Values{"csrf": {csrf}}.Encode())
		r.AddCookie(cookie)
		edit(r)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code < 400 {
			t.Fatal("security denial bypass")
		}
	}
	if count != 0 {
		t.Fatal("denial called application")
	}
	for _, edit := range []func(*http.Request){func(r *http.Request) { r.Host = "evil.example:8080" }, func(r *http.Request) { r.Header.Set("Origin", "http://evil.example") }} {
		r := request("GET", "/", "")
		r.AddCookie(cookie)
		edit(r)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code < 400 {
			t.Fatal("safe method bypassed origin/host")
		}
	}
}
func stringBody(s string) io.ReadCloser { return io.NopCloser(strings.NewReader(s)) }

func TestHTTPLoginCSRFReplayAndUnsupportedInput(t *testing.T) {
	a, token, _ := newTestAuth(t)
	g, _ := NewGate(testConfig(), a)
	for _, handler := range []http.HandlerFunc{g.Login, g.Logout} {
		w := httptest.NewRecorder()
		handler(w, request("GET", "/action", ""))
		if w.Code != 405 {
			t.Fatal("GET accepted state-changing auth action")
		}
	}
	for _, edit := range []func(*http.Request){
		func(r *http.Request) { r.Body = stringBody("token=" + token) },
		func(r *http.Request) {
			r.Body = stringBody(url.Values{"token": {token, token}, "csrf": {"placeholder"}}.Encode())
		},
		func(r *http.Request) { r.URL.RawQuery = "token=invalid" },
		func(r *http.Request) { r.Header.Set("Origin", "http://evil.example") },
		func(r *http.Request) {
			r.Header.Set("X-Forwarded-Host", "localhost:8080")
			r.Host = "evil.example:8080"
		},
		func(r *http.Request) { r.Body = stringBody("csrf=%zz") },
	} {
		w := httptest.NewRecorder()
		g.Challenge(w, request("GET", "/challenge", ""))
		var c Challenge
		_ = json.Unmarshal(w.Body.Bytes(), &c)
		cookie := w.Result().Cookies()[0]
		r := request("POST", "/login", url.Values{"token": {token}, "csrf": {c.CSRF}}.Encode())
		r.AddCookie(cookie)
		edit(r)
		w = httptest.NewRecorder()
		g.Login(w, r)
		if w.Code < 400 || strings.Contains(w.Body.String(), token) {
			t.Fatal("login input bypass or credential echo")
		}
	}
	w := httptest.NewRecorder()
	g.Challenge(w, request("GET", "/challenge", ""))
	var c Challenge
	_ = json.Unmarshal(w.Body.Bytes(), &c)
	cookie := w.Result().Cookies()[0]
	for i := 0; i < 2; i++ {
		r := request("POST", "/login", url.Values{"token": {token}, "csrf": {c.CSRF}}.Encode())
		r.AddCookie(cookie)
		w = httptest.NewRecorder()
		g.Login(w, r)
		if i == 0 && w.Code != 200 || i == 1 && w.Code != 401 {
			t.Fatal("challenge replay")
		}
		if strings.Contains(w.Body.String(), token) {
			t.Fatal("token echoed")
		}
	}
}
func TestHTTPIgnoresForwardedSourceAndBoundsPreauth(t *testing.T) {
	a, _, _ := newTestAuth(t)
	g, _ := NewGate(testConfig(), a)
	for i := 0; i < SourceAttempts+1; i++ {
		r := request("GET", "/challenge", "")
		r.RemoteAddr = fmt.Sprintf("127.0.0.1:%d", 10000+i)
		r.Header.Set("X-Forwarded-For", sourceFixture(i))
		r.Header.Set("Forwarded", "for="+sourceFixture(i))
		w := httptest.NewRecorder()
		g.Challenge(w, r)
		if i < SourceAttempts && w.Code != 200 || i == SourceAttempts && w.Code != 429 {
			t.Fatal("forwarded-IP or source-port throttle bypass")
		}
	}
}
func TestHTTPAcceptsStandardUTF8FormAndRejectsSafeRequestBodies(t *testing.T) {
	a, token, _ := newTestAuth(t)
	g, _ := NewGate(testConfig(), a)
	cookie, csrf := exchangeHTTP(t, g, token)
	calls := 0
	h := g.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(204) }))
	r := request("POST", "/refresh", url.Values{"csrf": {csrf}}.Encode())
	r.AddCookie(cookie)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal("standard UTF-8 form refused")
	}
	r = request("GET", "/", strings.Repeat("x", MaxBodyBytes+1))
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 || calls != 1 {
		t.Fatal("safe request body admitted to application")
	}
}

func TestMalformedDuplicateCookieCannotHideBehindValidSession(t *testing.T) {
	a, token, _ := newTestAuth(t)
	g, _ := NewGate(testConfig(), a)
	cookie, _ := exchangeHTTP(t, g, token)
	calls := 0
	h := g.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(204) }))
	r := request("GET", "/", "")
	r.Header.Set("Cookie", SessionCookie+"="+cookie.Value+"; "+SessionCookie+"=bad\\value")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 || calls != 0 {
		t.Fatal("malformed duplicate cookie hid from authentication")
	}
}
func TestDeniedPOSTDoesNotExtendSessionIdleExpiry(t *testing.T) {
	a, token, now := newTestAuth(t)
	g, _ := NewGate(testConfig(), a)
	cookie, _ := exchangeHTTP(t, g, token)
	h := g.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("denied/expired request reached application") }))
	*now = now.Add(IdleLifetime - time.Second)
	r := request("POST", "/refresh", "csrf=incorrect")
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("bad csrf")
	}
	*now = now.Add(time.Second)
	r = request("GET", "/", "")
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("denied POST extended session")
	}
}
