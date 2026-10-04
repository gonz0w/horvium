// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"net/netip"
	"sync"
	"time"
)

const (
	MaxSessions       = 8
	MaxChallenges     = 32
	SourceAttempts    = 20
	GlobalAttempts    = 100
	MaxSources        = 128
	ChallengeLifetime = 2 * time.Minute
	IdleLifetime      = 15 * time.Minute
	AbsoluteLifetime  = 8 * time.Hour
)

type Challenge struct {
	ID   string `json:"-"`
	CSRF string `json:"csrf"`
}
type Session struct {
	ID   string `json:"-"`
	CSRF string `json:"csrf"`
}
type preauth struct {
	Challenge
	source  string
	expires time.Time
}
type session struct {
	Session
	created, touched time.Time
}

// Auth is volatile. Every operation is bounded by the fixed M1 caps. Restart or
// Close invalidates all credentials. Neither the setup token nor native data is
// retained here. The supplied digest comes from a validated credential file.
type Auth struct {
	mu         sync.Mutex
	digest     [32]byte
	now        func() time.Time
	random     io.Reader
	challenges map[string]preauth
	sessions   map[string]session
	sources    map[string]int
	window     time.Time
	attempts   int
	closed     bool
}

func NewAuth(d [32]byte) *Auth {
	return &Auth{digest: d, now: time.Now, random: rand.Reader, challenges: map[string]preauth{}, sessions: map[string]session{}, sources: map[string]int{}}
}
func randomSecret(r io.Reader) (string, error) {
	var b [32]byte
	if _, e := io.ReadFull(r, b[:]); e != nil {
		return "", ErrState
	}
	return hex.EncodeToString(b[:]), nil
}
func validSecret(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func sameSecret(a, b string) bool {
	return validSecret(a) && validSecret(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func (a *Auth) prune(now time.Time) {
	for id, c := range a.challenges {
		if !now.Before(c.expires) {
			delete(a.challenges, id)
		}
	}
	for id, s := range a.sessions {
		if !now.Before(s.created.Add(AbsoluteLifetime)) || !now.Before(s.touched.Add(IdleLifetime)) {
			delete(a.sessions, id)
		}
	}
}

// A fixed one-minute window counts both challenge and login attempts, successful
// or failed. Reaching the source-map cap fails closed until window rollover;
// attacker-controlled identities are never evicted to bypass throttling.
func (a *Auth) admit(source string, now time.Time) error {
	ip, e := netip.ParseAddr(source)
	if e != nil || ip.Zone() != "" || ip.String() != source {
		return ErrUnauthorized
	}
	if a.window.IsZero() || !now.Before(a.window.Add(time.Minute)) {
		a.window = now
		a.sources = map[string]int{}
		a.attempts = 0
	}
	if a.attempts >= GlobalAttempts || a.sources[source] >= SourceAttempts {
		return ErrThrottled
	}
	if _, ok := a.sources[source]; !ok && len(a.sources) >= MaxSources {
		return ErrThrottled
	}
	a.attempts++
	a.sources[source]++
	return nil
}
func (a *Auth) Challenge(source string) (Challenge, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return Challenge{}, ErrClosed
	}
	now := a.now()
	a.prune(now)
	if e := a.admit(source, now); e != nil {
		return Challenge{}, e
	}
	if len(a.challenges) >= MaxChallenges {
		return Challenge{}, ErrCapacity
	}
	id, e := randomSecret(a.random)
	if e != nil {
		return Challenge{}, e
	}
	csrf, e := randomSecret(a.random)
	if e != nil {
		return Challenge{}, e
	}
	if _, ok := a.challenges[id]; ok || id == csrf {
		return Challenge{}, ErrState
	}
	c := Challenge{id, csrf}
	a.challenges[id] = preauth{c, source, now.Add(ChallengeLifetime)}
	return c, nil
}
func (a *Auth) Login(source, id, csrf, token string) (Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return Session{}, ErrClosed
	}
	now := a.now()
	a.prune(now)
	if e := a.admit(source, now); e != nil {
		return Session{}, e
	}
	c, ok := a.challenges[id]
	delete(a.challenges, id) // a failed exchange consumes its challenge too
	d := sha256.Sum256([]byte(token))
	matches := subtle.ConstantTimeCompare(d[:], a.digest[:])
	if !ok || c.source != source || !sameSecret(csrf, c.CSRF) || !validSecret(token) || matches != 1 {
		return Session{}, ErrUnauthorized
	}
	if len(a.sessions) >= MaxSessions {
		return Session{}, ErrCapacity
	}
	sid, e := randomSecret(a.random)
	if e != nil {
		return Session{}, e
	}
	sc, e := randomSecret(a.random)
	if e != nil {
		return Session{}, e
	}
	if _, ok := a.sessions[sid]; ok || sid == sc || sid == token {
		return Session{}, ErrState
	}
	s := Session{sid, sc}
	a.sessions[sid] = session{s, now, now}
	return s, nil
}

// Session first checks without extending expiry. The HTTP gate touches it only
// after method, provenance and CSRF checks have passed.
func (a *Auth) Session(id string, touch bool) (Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return Session{}, ErrClosed
	}
	now := a.now()
	a.prune(now)
	if !validSecret(id) {
		return Session{}, ErrUnauthorized
	}
	s, ok := a.sessions[id]
	if !ok {
		return Session{}, ErrUnauthorized
	}
	if touch {
		s.touched = now
		a.sessions[id] = s
	}
	return s.Session, nil
}
func (a *Auth) Logout(id string) { a.mu.Lock(); defer a.mu.Unlock(); delete(a.sessions, id) }
func (a *Auth) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
	clear(a.sessions)
	clear(a.challenges)
	clear(a.sources)
	a.digest = [32]byte{}
}
