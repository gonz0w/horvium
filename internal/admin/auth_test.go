// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestAuth(t *testing.T) (*Auth, string, *time.Time) {
	t.Helper()
	token, e := randomSecret(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	a := NewAuth(sha256.Sum256([]byte(token)))
	a.now = func() time.Time { return now }
	return a, token, &now
}
func loginTest(t *testing.T, a *Auth, token, source string) Session {
	t.Helper()
	c, e := a.Challenge(source)
	if e != nil {
		t.Fatal(e)
	}
	s, e := a.Login(source, c.ID, c.CSRF, token)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func TestChallengeAndTokenExchange(t *testing.T) {
	a, token, now := newTestAuth(t)
	c, e := a.Challenge("127.0.0.1")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.Login("127.0.0.2", c.ID, c.CSRF, token); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("challenge source bypass")
	}
	if _, e = a.Login("127.0.0.1", c.ID, c.CSRF, token); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("consumed challenge replay")
	}
	c, _ = a.Challenge("127.0.0.1")
	if _, e = a.Login("127.0.0.1", c.ID, c.CSRF, token+"x"); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("bad token")
	}
	c, _ = a.Challenge("127.0.0.1")
	*now = now.Add(ChallengeLifetime)
	if _, e = a.Login("127.0.0.1", c.ID, c.CSRF, token); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("expired challenge")
	}
	*now = now.Add(time.Minute)
	s := loginTest(t, a, token, "127.0.0.1")
	if len(s.ID) != 64 || len(s.CSRF) != 64 || s.ID == s.CSRF || s.ID == token {
		t.Fatal("credentials not distinct random secrets")
	}
	if _, e = hex.DecodeString(s.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = a.Session(s.ID, false); e != nil {
		t.Fatal(e)
	}
	a.Logout(s.ID)
	if _, e = a.Session(s.ID, false); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("logout failed")
	}
	other := NewAuth(a.digest)
	if _, e = other.Session(s.ID, false); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("session survived restart")
	}
}
func TestSessionIdleAbsoluteAndCapacity(t *testing.T) {
	a, token, now := newTestAuth(t)
	s := loginTest(t, a, token, "127.0.0.1")
	*now = now.Add(IdleLifetime - time.Second)
	if _, e := a.Session(s.ID, true); e != nil {
		t.Fatal(e)
	}
	*now = now.Add(IdleLifetime)
	if _, e := a.Session(s.ID, false); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("idle expiry")
	}
	a, token, now = newTestAuth(t)
	s = loginTest(t, a, token, "127.0.0.1")
	for i := 0; i < 32; i++ {
		*now = now.Add(IdleLifetime - time.Second)
		if _, e := a.Session(s.ID, true); e != nil {
			t.Fatal(e)
		}
	}
	*now = now.Add(32 * time.Second)
	if _, e := a.Session(s.ID, false); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("absolute expiry")
	}
	a, token, now = newTestAuth(t)
	for i := 0; i < MaxSessions; i++ {
		loginTest(t, a, token, "127.0.0.1")
	}
	c, _ := a.Challenge("127.0.0.1")
	if _, e := a.Login("127.0.0.1", c.ID, c.CSRF, token); !errors.Is(e, ErrCapacity) {
		t.Fatal("session capacity exceeded")
	}
	*now = now.Add(IdleLifetime)
	loginTest(t, a, token, "127.0.0.1")
}
func TestBoundedThrottleAndPreauth(t *testing.T) {
	a, _, now := newTestAuth(t)
	for i := 0; i < SourceAttempts; i++ {
		if _, e := a.Challenge("127.0.0.1"); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := a.Challenge("127.0.0.1"); !errors.Is(e, ErrThrottled) {
		t.Fatal("source limit")
	}
	*now = now.Add(time.Minute)
	if _, e := a.Challenge("127.0.0.1"); e != nil {
		t.Fatal(e)
	}
	a, _, now = newTestAuth(t)
	for i := 0; i < MaxChallenges; i++ {
		if _, e := a.Challenge(sourceFixture(i)); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := a.Challenge("127.0.0.1"); !errors.Is(e, ErrCapacity) {
		t.Fatal("challenge cap")
	}
	*now = now.Add(ChallengeLifetime)
	if _, e := a.Challenge("127.0.0.1"); e != nil {
		t.Fatal(e)
	}
	a, _, now = newTestAuth(t)
	for i := 0; i < GlobalAttempts; i++ {
		_, _ = a.Challenge(sourceFixture(i))
	}
	if _, e := a.Challenge("127.0.0.1"); !errors.Is(e, ErrThrottled) {
		t.Fatal("global limit")
	}
	*now = now.Add(ChallengeLifetime)
	if _, e := a.Challenge("127.0.0.1"); e != nil {
		t.Fatal(e)
	}
}
func TestConcurrentSessionLimitAndClear(t *testing.T) {
	a, token, _ := newTestAuth(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	count := 0
	for i := 0; i < 32; i++ {
		wg.Go(func() {
			c, e := a.Challenge(sourceFixture(i))
			if e != nil {
				return
			}
			if _, e = a.Login(sourceFixture(i), c.ID, c.CSRF, token); e == nil {
				mu.Lock()
				count++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if count != MaxSessions {
		t.Fatal("concurrent admission violated capacity")
	}
	a.Close()
	if _, e := a.Challenge("127.0.0.1"); !errors.Is(e, ErrClosed) {
		t.Fatal("closed auth still running")
	}
}

func TestEntropyFailureAndSourceMapFailClosed(t *testing.T) {
	a, token, now := newTestAuth(t)
	a.random = strings.NewReader("")
	if _, e := a.Challenge("127.0.0.1"); !errors.Is(e, ErrState) {
		t.Fatal("entropy failure ignored")
	}
	a.random = rand.Reader
	c, e := a.Challenge("127.0.0.1")
	if e != nil {
		t.Fatal(e)
	}
	a.random = strings.NewReader("")
	if _, e = a.Login("127.0.0.1", c.ID, c.CSRF, token); !errors.Is(e, ErrState) {
		t.Fatal("session entropy failure ignored")
	}
	if len(a.sessions) != 0 {
		t.Fatal("partial session retained")
	}
	a, _, now = newTestAuth(t)
	a.window = *now
	for i := 0; i < MaxSources; i++ {
		a.sources[sourceFixture(i)] = 1
	}
	if _, e := a.Challenge("127.0.0.1"); !errors.Is(e, ErrThrottled) {
		t.Fatal("source cap evicted existing identities")
	}
	*now = now.Add(time.Minute)
	if _, e := a.Challenge("127.0.0.1"); e != nil {
		t.Fatal("bounded source window did not reset")
	}
}
