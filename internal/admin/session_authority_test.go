// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func actionRequest(s Session) *http.Request {
	r := request("POST", "/fixture-action", url.Values{"csrf": {s.CSRF}}.Encode())
	r.AddCookie(&http.Cookie{Name: SessionCookie, Value: s.ID})
	return r
}
func issueAuthority(t *testing.T, g *Gate, s Session) *SessionAuthority {
	t.Helper()
	var authority *SessionAuthority
	w := httptest.NewRecorder()
	g.ProtectAction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authority = RequestSessionAuthority(r)
		if RequestCSRF(r) != s.CSRF {
			t.Error("CSRF contract lost")
		}
		w.WriteHeader(204)
	})).ServeHTTP(w, actionRequest(s))
	if w.Code != 204 || authority == nil {
		t.Fatalf("authenticated action lacks authority: %d", w.Code)
	}
	return authority
}
func authorityFixture(t *testing.T) (*Auth, *Gate, Session, *SessionAuthority, *time.Time, string) {
	t.Helper()
	a, token, now := newTestAuth(t)
	s := loginTest(t, a, token, "127.0.0.1")
	g, err := NewGate(testConfig(), a)
	if err != nil {
		t.Fatal(err)
	}
	return a, g, s, issueAuthority(t, g, s), now, token
}
func TestSessionAuthorityAuthenticatedAction(t *testing.T) {
	a, g, s, b, now, token := authorityFixture(t)
	id, err := b.Context(context.Background())
	if err != nil || !validSecret(id) || id == s.ID || id == s.CSRF || id == token {
		t.Fatal("context exposes credentials", err)
	}
	again := issueAuthority(t, g, s)
	if next, e := again.Context(context.Background()); e != nil || next != id {
		t.Fatal("original context changed", e)
	}
	copy := *b
	if e := copy.CheckLive(context.Background()); e != nil {
		t.Fatal("copy lost original identity", e)
	}
	touched := a.sessions[s.ID].touched
	*now = now.Add(time.Second)
	if e := copy.CheckLive(context.Background()); e != nil {
		t.Fatal(e)
	}
	if !a.sessions[s.ID].touched.Equal(touched) {
		t.Fatal("live check extended session")
	}
	raw, e := json.Marshal(b)
	if e != nil || string(raw) != "{}" {
		t.Fatal("serialized authority", e)
	}
	var decoded SessionAuthority
	if json.Unmarshal(raw, &decoded) != nil || decoded.CheckLive(context.Background()) == nil {
		t.Fatal("decoded authority authenticated")
	}
	formatted := fmt.Sprintf("%+v", b)
	for _, secret := range []string{s.ID, s.CSRF, token, id} {
		if strings.Contains(formatted, secret) {
			t.Fatal("formatted authority exposes context/credential")
		}
	}
	if RequestSessionAuthority(actionRequest(s)) != nil || RequestSessionAuthority(nil) != nil {
		t.Fatal("request data reconstructed authority")
	}
	r := actionRequest(s)
	r.Header.Set("Cookie", SessionCookie+"="+id)
	w := httptest.NewRecorder()
	g.ProtectAction(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("context accepted as bearer credential") })).ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("context authenticated as cookie", w.Code)
	}
}
func TestSessionAuthorityActionDenials(t *testing.T) {
	cases := map[string]func(*http.Request){
		"get":        func(r *http.Request) { r.Method = "GET" },
		"head":       func(r *http.Request) { r.Method = "HEAD" },
		"put":        func(r *http.Request) { r.Method = "PUT" },
		"host":       func(r *http.Request) { r.Host = "evil.test" },
		"origin":     func(r *http.Request) { r.Header.Set("Origin", "http://evil.test") },
		"cross-site": func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") },
		"duplicate-origin": func(r *http.Request) {
			r.Header.Add("Origin", "http://localhost:8080")
			r.Header.Add("Origin", "http://localhost:8080")
		},
		"missing-cookie":   func(r *http.Request) { r.Header.Del("Cookie") },
		"forged-cookie":    func(r *http.Request) { r.Header.Set("Cookie", SessionCookie+"="+strings.Repeat("f", 64)) },
		"duplicate-cookie": func(r *http.Request) { r.Header.Add("Cookie", r.Header.Get("Cookie")) },
		"missing-csrf":     func(r *http.Request) { r.Body = io.NopCloser(strings.NewReader("value=1")); r.ContentLength = 7 },
		"wrong-csrf":       func(r *http.Request) { r.Body = io.NopCloser(strings.NewReader("csrf=wrong")); r.ContentLength = 10 },
		"duplicate-csrf":   func(r *http.Request) { r.Body = io.NopCloser(strings.NewReader("csrf=a&csrf=b")); r.ContentLength = 13 },
		"content-type":     func(r *http.Request) { r.Header.Set("Content-Type", "application/json") },
		"body-limit": func(r *http.Request) {
			r.Body = io.NopCloser(strings.NewReader(strings.Repeat("x", MaxBodyBytes+1)))
			r.ContentLength = MaxBodyBytes + 1
		},
		"query": func(r *http.Request) { r.URL.RawQuery = "csrf=forged" },
		"cancelled": func(r *http.Request) {
			ctx, cancel := context.WithCancel(r.Context())
			cancel()
			*r = *r.WithContext(ctx)
		},
	}
	for name, alter := range cases {
		t.Run(name, func(t *testing.T) {
			a, token, _ := newTestAuth(t)
			s := loginTest(t, a, token, "127.0.0.1")
			g, _ := NewGate(testConfig(), a)
			a.random = strings.NewReader("") // Denial must precede authority entropy/issuance.
			calls := 0
			h := g.ProtectAction(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
			r := actionRequest(s)
			alter(r)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if calls != 0 || w.Code == 204 || w.Code == 200 {
				t.Fatal("denied action dispatched", w.Code)
			}
		})
	}
}
func TestSessionAuthorityRevocation(t *testing.T) {
	for _, mode := range []string{"logout", "close", "idle", "absolute", "rollback", "zero-clock", "invalid-clock"} {
		t.Run(mode, func(t *testing.T) {
			a, _, s, b, now, _ := authorityFixture(t)
			switch mode {
			case "logout":
				a.Logout(s.ID)
			case "close":
				a.Close()
			case "idle":
				*now = now.Add(IdleLifetime)
			case "absolute":
				*now = now.Add(AbsoluteLifetime)
			case "rollback":
				*now = now.Add(-time.Second)
			case "zero-clock":
				*now = time.Time{}
			case "invalid-clock":
				*now = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
			}
			copy := *b
			if copy.CheckLive(context.Background()) == nil {
				t.Fatal("revoked original authority remained live")
			}
			if id, e := b.Context(context.Background()); e == nil || id != "" {
				t.Fatal("revoked context projected")
			}
			if mode == "rollback" || mode == "zero-clock" || mode == "invalid-clock" {
				*now = time.Date(2026, 10, 4, 12, 0, 1, 0, time.UTC)
				if b.CheckLive(context.Background()) == nil {
					t.Fatal("clock failure reset authority")
				}
			}
		})
	}
	for _, mode := range []string{"nil", "zero", "nil-context", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			_, _, _, b, _, _ := authorityFixture(t)
			ctx := context.Background()
			switch mode {
			case "nil":
				b = nil
			case "zero":
				b = &SessionAuthority{}
			case "nil-context":
				ctx = nil
			case "cancelled":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			}
			if b.CheckLive(ctx) == nil {
				t.Fatal("invalid capability/context authenticated")
			}
			if id, e := b.Context(ctx); e == nil || id != "" {
				t.Fatal("invalid context projected")
			}
		})
	}
}
func TestSessionAuthorityReadGateIsolation(t *testing.T) {
	a, token, _ := newTestAuth(t)
	s := loginTest(t, a, token, "127.0.0.1")
	g, _ := NewGate(testConfig(), a)
	a.random = strings.NewReader("")
	h := g.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestSessionAuthority(r) != nil {
			t.Error("ordinary read minted action authority")
		}
		w.WriteHeader(204)
	}))
	r := request("GET", "/fixture-read", "")
	r.AddCookie(&http.Cookie{Name: SessionCookie, Value: s.ID})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal("read path changed", w.Code)
	}
	w = httptest.NewRecorder()
	g.ProtectAction(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("entropy failure dispatched action") })).ServeHTTP(w, actionRequest(s))
	if w.Code != 503 || a.sessions[s.ID].identity.binding != nil {
		t.Fatal("failed action installed authority", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal("failed issuance disabled ordinary reads", w.Code)
	}
}
func TestSessionAuthorityConcurrentIssuance(t *testing.T) {
	a, token, _ := newTestAuth(t)
	s := loginTest(t, a, token, "127.0.0.1")
	g, _ := NewGate(testConfig(), a)
	var wg sync.WaitGroup
	ids := make(chan string, 32)
	h := g.ProtectAction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := RequestSessionAuthority(r)
		id, err := b.Context(context.Background())
		if err != nil {
			t.Error(err)
		}
		ids <- id
		w.WriteHeader(204)
	}))
	for i := 0; i < 32; i++ {
		wg.Go(func() {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, actionRequest(s))
			if w.Code != 204 {
				t.Error(w.Code)
			}
		})
	}
	wg.Wait()
	close(ids)
	id := ""
	count := 0
	for got := range ids {
		if !validSecret(got) || id != "" && got != id {
			t.Error("concurrent context changed")
		}
		id = got
		count++
	}
	if count != 32 {
		t.Fatal("missing authorized callers")
	}
	a.Logout(s.ID)
	if _, e := a.Session(s.ID, false); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("concurrent identity survived logout")
	}
}

func TestSessionAuthorityOriginalBinding(t *testing.T) {
	for _, mode := range []string{"removed", "changed-id", "changed-csrf", "changed-created", "future-touch", "restored-record", "replaced-identity", "copied-binding"} {
		t.Run(mode, func(t *testing.T) {
			a, _, s, b, now, _ := authorityFixture(t)
			entry := a.sessions[s.ID]
			switch mode {
			case "removed":
				delete(a.sessions, s.ID)
			case "changed-id":
				entry.ID = strings.Repeat("e", 64)
				a.sessions[s.ID] = entry
			case "changed-csrf":
				entry.CSRF = strings.Repeat("e", 64)
				a.sessions[s.ID] = entry
			case "changed-created":
				entry.created = entry.created.Add(-time.Second)
				a.sessions[s.ID] = entry
			case "future-touch":
				entry.touched = now.Add(time.Second)
				a.sessions[s.ID] = entry
			case "restored-record":
				entry.identity = nil
				a.sessions[s.ID] = entry
			case "replaced-identity":
				copy := *entry.identity
				entry.identity = &copy
				a.sessions[s.ID] = entry
			case "copied-binding":
				copy := *b.binding
				b = &SessionAuthority{binding: &copy}
			}
			if b.CheckLive(context.Background()) == nil {
				t.Fatal("replaced original entry authenticated")
			}
		})
	}
	t.Run("other-auth", func(t *testing.T) {
		a, _, s, b, _, _ := authorityFixture(t)
		other := NewAuth(a.digest)
		other.now = a.now
		other.sessions[s.ID] = a.sessions[s.ID]
		if cap, e := other.bindSession(context.Background(), s.ID); e == nil || cap != nil {
			t.Fatal("copied original entry transferred authority")
		}
		a.Logout(s.ID)
		if b.CheckLive(context.Background()) == nil {
			t.Fatal("copied entry kept original authority alive")
		}
	})
	t.Run("reused-credential", func(t *testing.T) {
		a, _, s, b, _, token := authorityFixture(t)
		a.Logout(s.ID)
		// Simulate a new valid login with the same Session fields but a fresh original
		// private identity. Old handles cannot survive replacement/credential reuse.
		replacement := loginTest(t, a, token, "127.0.0.1")
		entry := a.sessions[replacement.ID]
		delete(a.sessions, replacement.ID)
		entry.Session = s
		a.sessions[s.ID] = entry
		if b.CheckLive(context.Background()) == nil {
			t.Fatal("old capability survived new identity")
		}
	})
}

type authorityEntropy struct {
	data []byte
	cut  func()
}

func (r *authorityEntropy) Read(p []byte) (int, error) {
	if r.cut != nil {
		r.cut()
		r.cut = nil
	}
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}
func TestSessionAuthorityEntropyAndFinalChecks(t *testing.T) {
	for _, mode := range []string{"entropy-failure", "nil-reader", "session-collision", "csrf-collision", "token-collision", "context-collision", "expired-during-entropy", "rollback-during-entropy", "cancelled-during-entropy", "removed-during-entropy"} {
		t.Run(mode, func(t *testing.T) {
			a, token, now := newTestAuth(t)
			s := loginTest(t, a, token, "127.0.0.1")
			ctx := context.Background()
			nonce := strings.Repeat("d", 64)
			if mode == "context-collision" {
				other := loginTest(t, a, token, "127.0.0.1")
				prior, e := a.bindSession(ctx, other.ID)
				if e != nil {
					t.Fatal(e)
				}
				nonce, _ = prior.Context(ctx)
			}
			switch mode {
			case "session-collision":
				nonce = s.ID
			case "csrf-collision":
				nonce = s.CSRF
			case "token-collision":
				nonce = token
			}
			bytes, err := hex.DecodeString(nonce)
			if err != nil {
				t.Fatal(err)
			}
			reader := &authorityEntropy{data: bytes}
			a.random = reader
			switch mode {
			case "entropy-failure":
				a.random = strings.NewReader("")
			case "nil-reader":
				a.random = nil
			case "expired-during-entropy":
				reader.cut = func() { *now = now.Add(IdleLifetime) }
			case "rollback-during-entropy":
				reader.cut = func() { *now = now.Add(-time.Second) }
			case "cancelled-during-entropy":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				reader.cut = cancel
			case "removed-during-entropy":
				reader.cut = func() { delete(a.sessions, s.ID) }
			}
			b, e := a.bindSession(ctx, s.ID)
			if e == nil || b != nil {
				t.Fatal("entropy/final cut issued authority")
			}
			if entry, ok := a.sessions[s.ID]; ok && entry.identity.binding != nil {
				t.Fatal("failed issuance partially installed binding")
			}
		})
	}
}
func TestSessionAuthorityReadClockCannotResetBinding(t *testing.T) {
	a, g, s, b, now, _ := authorityFixture(t)
	*now = now.Add(time.Minute)
	if _, e := a.Session(s.ID, true); e != nil {
		t.Fatal(e)
	}
	*now = now.Add(-time.Second)
	// Existing M1 read behavior remains; its private clock witness revokes action
	// authority even if the clock subsequently moves forward before launch.
	if _, e := a.Session(s.ID, true); e != nil {
		t.Fatal(e)
	}
	*now = now.Add(2 * time.Second)
	if b.CheckLive(context.Background()) == nil {
		t.Fatal("read-touch hid clock rollback")
	}
	w := httptest.NewRecorder()
	g.ProtectAction(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("reminted blocked identity") })).ServeHTTP(w, actionRequest(s))
	if w.Code != 503 {
		t.Fatal("clock-blocked action reminted", w.Code)
	}
}

func TestSessionAuthorityRevocationCannotRestoreEntry(t *testing.T) {
	for _, mode := range []string{"logout", "pruned-expiry", "unavailable-clock"} {
		t.Run(mode, func(t *testing.T) {
			a, _, s, b, now, _ := authorityFixture(t)
			original := a.sessions[s.ID]
			clock := a.now
			switch mode {
			case "logout":
				a.Logout(s.ID)
			case "pruned-expiry":
				*now = now.Add(IdleLifetime)
				_, _ = a.Session(s.ID, false)
			case "unavailable-clock":
				a.now = nil
				_ = b.CheckLive(context.Background())
			}
			a.sessions[s.ID] = original
			a.now = clock
			*now = time.Date(2026, 10, 4, 12, 0, 1, 0, time.UTC)
			if b.CheckLive(context.Background()) == nil {
				t.Fatal("restored private entry reset retired authority")
			}
		})
	}
}

func FuzzSessionAuthorityActions(f *testing.F) {
	f.Add([]byte{0, 1, 6, 7, 8, 2, 0, 3})
	f.Add([]byte{8, 8, 5, 8, 0})
	f.Add([]byte{4, 0, 10, 1})
	f.Fuzz(func(t *testing.T, actions []byte) {
		if len(actions) > 32 {
			actions = actions[:32]
		}
		a, g, s, b, now, _ := authorityFixture(t)
		revoked := false
		for _, action := range actions {
			switch action % 11 {
			case 0:
				id, e := b.Context(context.Background())
				if revoked {
					if e == nil || id != "" {
						t.Fatal("revoked projection")
					}
				} else if e != nil || !validSecret(id) {
					t.Fatal("live projection lost", e)
				}
			case 1:
				copy := *b
				e := copy.CheckLive(context.Background())
				if (e != nil) != revoked {
					t.Fatal("copy authority changed")
				}
			case 2:
				a.Logout(s.ID)
				revoked = true
			case 3:
				a.Close()
				revoked = true
			case 4:
				*now = now.Add(AbsoluteLifetime)
				_ = b.CheckLive(context.Background())
				revoked = true
			case 5:
				*now = now.Add(-time.Second)
				_ = b.CheckLive(context.Background())
				revoked = true
			case 6:
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if id, e := b.Context(ctx); e == nil || id != "" {
					t.Fatal("cancelled projection")
				}
			case 7:
				raw, _ := json.Marshal(b)
				var decoded SessionAuthority
				_ = json.Unmarshal(raw, &decoded)
				if decoded.CheckLive(context.Background()) == nil {
					t.Fatal("decoded capability")
				}
			case 8:
				*now = now.Add(time.Second)
				_, _ = a.Session(s.ID, true)
			case 9:
				r := actionRequest(s)
				r.Header.Add("Cookie", r.Header.Get("Cookie"))
				w := httptest.NewRecorder()
				g.ProtectAction(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("ambiguous credentials dispatched") })).ServeHTTP(w, r)
			case 10:
				entry := a.sessions[s.ID]
				entry.identity = nil
				a.sessions[s.ID] = entry
				revoked = true
			}
		}
	})
}
