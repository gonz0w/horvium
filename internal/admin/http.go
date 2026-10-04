// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"context"
	"encoding/json"
	"errors"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"
)

const (
	SessionCookie   = "horvium_session"
	ChallengeCookie = "horvium_challenge"
	MaxBodyBytes    = 4096
	MaxHeaderBytes  = 16 << 10
)

type csrfKey struct{}

// Gate is shared by the future HTML/fragment and CLI adapters. It only authorizes
// requests. Protect must wrap every inventory/refresh handler; there is no
// unauthenticated loopback/health/fragment exception. No native work occurs here.
type Gate struct {
	hosts  map[string]bool
	auth   *Auth
	origin http.CrossOriginProtection
}

func NewGate(c Config, a *Auth) (*Gate, error) {
	if c.Validate() != nil || a == nil {
		return nil, ErrConfig
	}
	g := &Gate{auth: a, hosts: map[string]bool{}}
	for _, h := range c.AllowedHosts {
		g.hosts[h] = true
	}
	return g, nil
}

// HTTPServer sets transport caps but does not bind, launch or install anything.
// Callers must use the validated loopback address and only gated handlers.
func (c Config) HTTPServer(h http.Handler) (*http.Server, error) {
	if c.Validate() != nil || h == nil {
		return nil, ErrConfig
	}
	return &http.Server{Addr: c.Listen, Handler: h, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: MaxHeaderBytes}, nil
}
func headers(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; object-src 'none'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "same-origin")
}
func deny(w http.ResponseWriter, status int) { http.Error(w, http.StatusText(status), status) }
func (g *Gate) policy(w http.ResponseWriter, r *http.Request) bool {
	headers(w)
	if (r.Method == "GET" || r.Method == "HEAD") && (r.ContentLength != 0 || len(r.TransferEncoding) != 0) {
		deny(w, 400)
		return false
	}
	total := len(r.Host) + len(r.RequestURI)
	for k, vs := range r.Header {
		total += len(k)
		for _, v := range vs {
			total += len(v) + 4
		}
		if total > MaxHeaderBytes {
			deny(w, 431)
			return false
		}
	}
	if total > MaxHeaderBytes {
		deny(w, 431)
		return false
	}
	if !g.hosts[r.Host] || r.TLS != nil || r.URL.Scheme != "" || r.URL.Host != "" {
		deny(w, 403)
		return false
	}
	origins := r.Header.Values("Origin")
	if len(origins) > 1 || len(origins) == 1 && origins[0] != "http://"+r.Host {
		deny(w, 403)
		return false
	}
	sites := r.Header.Values("Sec-Fetch-Site")
	if len(sites) > 1 || len(sites) == 1 && sites[0] != "same-origin" && sites[0] != "none" {
		deny(w, 403)
		return false
	}
	if g.origin.Check(r) != nil {
		deny(w, 403)
		return false
	}
	return true
}
func remoteSource(r *http.Request) (string, error) {
	host, _, e := net.SplitHostPort(r.RemoteAddr)
	ip, ie := netip.ParseAddr(host)
	if e != nil || ie != nil || !ip.IsLoopback() || ip.Zone() != "" {
		return "", ErrUnauthorized
	}
	return ip.Unmap().String(), nil
}

// Count raw credential names before net/http can silently skip an invalid
// cookie value. Credentials use a canonical unquoted hexadecimal value; a
// malformed duplicate must not turn an ambiguous request into an authorized one.
func cookieValue(r *http.Request, name string) (string, error) {
	value := ""
	count := 0
	for _, line := range r.Header.Values("Cookie") {
		for _, part := range strings.Split(line, ";") {
			key, v, hasValue := strings.Cut(strings.TrimSpace(part), "=")
			if strings.TrimSpace(key) == name {
				count++
				if !hasValue || !validSecret(v) {
					return "", ErrUnauthorized
				}
				value = v
			}
		}
	}
	if count != 1 {
		return "", ErrUnauthorized
	}
	return value, nil
}

func cookie(w http.ResponseWriter, name, value string, remove bool) {
	c := &http.Cookie{Name: name, Value: value, HttpOnly: true, SameSite: http.SameSiteStrictMode, Path: "/"}
	if remove {
		c.MaxAge = -1
		c.Expires = time.Unix(1, 0)
	}
	http.SetCookie(w, c)
}
func form(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != "POST" {
		deny(w, 405)
		return false
	}
	if len(r.Header.Values("Content-Type")) != 1 {
		deny(w, 415)
		return false
	}
	typ, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil || typ != "application/x-www-form-urlencoded" || (len(params) > 1 || len(params) == 1 && !strings.EqualFold(params["charset"], "utf-8")) {
		deny(w, 415)
		return false
	}
	if r.ContentLength > MaxBodyBytes {
		deny(w, 413)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	if e = r.ParseForm(); e != nil {
		var large *http.MaxBytesError
		if errors.As(e, &large) {
			deny(w, 413)
		} else {
			deny(w, 400)
		}
		return false
	}
	return true
}
func formValue(r *http.Request, key string) (string, bool) {
	values := r.PostForm[key]
	return first(values), len(values) == 1
}
func first(v []string) string {
	if len(v) == 0 {
		return ""
	}
	return v[0]
}
func authError(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, ErrThrottled):
		w.Header().Set("Retry-After", "60")
		deny(w, 429)
	case errors.Is(e, ErrCapacity):
		deny(w, 429)
	case errors.Is(e, ErrUnauthorized):
		deny(w, 401)
	default:
		deny(w, 503)
	}
}
func jsonReply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// Challenge is the bounded pre-authentication exchange; its ID is carried only
// in an HttpOnly cookie and the independent CSRF value in the response body.
func (g *Gate) BeginLogin(w http.ResponseWriter, r *http.Request) (Challenge, bool) {
	if !g.policy(w, r) {
		return Challenge{}, false
	}
	if r.Method != "GET" || r.URL.RawQuery != "" {
		deny(w, 405)
		return Challenge{}, false
	}
	source, e := remoteSource(r)
	if e != nil {
		authError(w, e)
		return Challenge{}, false
	}
	c, e := g.auth.Challenge(source)
	if e != nil {
		authError(w, e)
		return Challenge{}, false
	}
	cookie(w, ChallengeCookie, c.ID, false)
	return c, true
}
func (g *Gate) ExchangeLogin(w http.ResponseWriter, r *http.Request) (Session, bool) {
	if !g.policy(w, r) {
		return Session{}, false
	}
	if r.URL.RawQuery != "" {
		deny(w, 400)
		return Session{}, false
	}
	if !form(w, r) {
		return Session{}, false
	}
	token, tok := formValue(r, "token")
	csrf, csok := formValue(r, "csrf")
	id, e := cookieValue(r, ChallengeCookie)
	if !tok || !csok || len(r.PostForm) != 2 || e != nil {
		deny(w, 403)
		return Session{}, false
	}
	source, e := remoteSource(r)
	if e != nil {
		authError(w, e)
		return Session{}, false
	}
	s, e := g.auth.Login(source, id, csrf, token)
	cookie(w, ChallengeCookie, "", true)
	if e != nil {
		authError(w, e)
		return Session{}, false
	}
	cookie(w, SessionCookie, s.ID, false)
	return s, true
}
func (g *Gate) Challenge(w http.ResponseWriter, r *http.Request) {
	if c, ok := g.BeginLogin(w, r); ok {
		jsonReply(w, c)
	}
}
func (g *Gate) Login(w http.ResponseWriter, r *http.Request) {
	if s, ok := g.ExchangeLogin(w, r); ok {
		jsonReply(w, s)
	}
}

func (g *Gate) authorize(w http.ResponseWriter, r *http.Request, browser bool) (Session, bool) {
	id, e := cookieValue(r, SessionCookie)
	if e != nil {
		g.sessionError(w, r, e, browser)
		return Session{}, false
	}
	s, e := g.auth.Session(id, false)
	if e != nil {
		g.sessionError(w, r, e, browser)
		return Session{}, false
	}
	if r.Method == "POST" {
		if !form(w, r) {
			return Session{}, false
		}
		csrf, ok := formValue(r, "csrf")
		if !ok || !sameSecret(csrf, s.CSRF) {
			deny(w, 403)
			return Session{}, false
		}
	}
	s, e = g.auth.Session(id, true)
	if e != nil {
		g.sessionError(w, r, e, browser)
		return Session{}, false
	}
	return s, true
}
func (g *Gate) EndSession(w http.ResponseWriter, r *http.Request) bool {
	if !g.policy(w, r) {
		return false
	}
	if r.Method != "POST" {
		deny(w, 405)
		return false
	}
	if r.URL.RawQuery != "" {
		deny(w, 400)
		return false
	}
	s, ok := g.authorize(w, r, false)
	if !ok {
		return false
	}
	if len(r.PostForm) != 1 {
		deny(w, 400)
		return false
	}
	g.auth.Logout(s.ID)
	cookie(w, SessionCookie, "", true)
	return true
}
func (g *Gate) Logout(w http.ResponseWriter, r *http.Request) {
	if g.EndSession(w, r) {
		w.WriteHeader(204)
	}
}
func (g *Gate) sessionError(w http.ResponseWriter, r *http.Request, e error, browser bool) {
	if browser && errors.Is(e, ErrUnauthorized) {
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/login")
			deny(w, 401)
		} else {
			http.Redirect(w, r, "/login", 303)
		}
		return
	}
	authError(w, e)
}

// Public allows only explicitly registered login/assets, never inventory.
func (g *Gate) Public(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !g.policy(w, r) {
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			deny(w, 405)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (g *Gate) ProtectPage(next http.Handler) http.Handler { return g.protect(next, true) }
func (g *Gate) Protect(next http.Handler) http.Handler     { return g.protect(next, false) }
func (g *Gate) protect(next http.Handler, browser bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !g.policy(w, r) {
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "POST" {
			deny(w, 405)
			return
		}
		s, ok := g.authorize(w, r, browser)
		if !ok {
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), csrfKey{}, s.CSRF)))
	})
}
func RequestCSRF(r *http.Request) string { v, _ := r.Context().Value(csrfKey{}).(string); return v }
