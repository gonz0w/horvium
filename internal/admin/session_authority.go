// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"context"
	"crypto/sha256"
	"net/http"
	"time"
)

// SessionAuthority proves only the original live application session. It is not
// kernel-peer identity, root policy, recovery clearance or mutation permission.
// Copies share the same volatile binding; no public constructor/decoder exists.
type SessionAuthority struct{ binding *sessionBinding }
type sessionBinding struct {
	owner     *Auth
	sessionID string
	identity  *sessionIdentity
}
type sessionIdentity struct {
	credentials [32]byte
	created     time.Time
	owner       *Auth
	last        time.Time
	blocked     bool
	binding     *sessionBinding
	context     string
}

// Called only under owner.mu. Keep a sticky action-authority clock failure while
// preserving existing M1 read/login behavior. Ordinary session checks also record
// observed time, so a later action cannot hide rollback through an intervening read.
func (s *sessionIdentity) observe(now time.Time) error {
	at := now.Round(0).UTC()
	if s.blocked || at.IsZero() || at.Year() < 1970 || at.Year() > 9999 || at.Before(s.last) {
		s.blocked = true
		return ErrState
	}
	s.last = at
	return nil
}
func sessionFingerprint(s Session) [32]byte {
	return sha256.Sum256([]byte(s.ID + "\x00" + s.CSRF))
}

func authorityContext(ctx context.Context) error {
	if ctx == nil {
		return ErrUnauthorized
	}
	return ctx.Err()
}

// Called only under a.mu; check the current original entry, not bearer strings
// or a reconstructed Session. Checking never extends idle/absolute lifetime.
func (a *Auth) authoritySession(id string, identity *sessionIdentity) (session, error) {
	if a.closed {
		return session{}, ErrClosed
	}
	if !validSecret(id) || identity == nil || identity.owner != a {
		return session{}, ErrUnauthorized
	}
	if a.now == nil {
		identity.blocked = true
		return session{}, ErrState
	}
	s, ok := a.sessions[id]
	if !ok || s.identity != identity || !sameSecret(id, s.ID) || !validSecret(s.CSRF) || sessionFingerprint(s.Session) != identity.credentials || !s.created.Round(0).UTC().Equal(identity.created) {
		return session{}, ErrUnauthorized
	}
	now := a.now().Round(0).UTC()
	created, touched := s.created.Round(0).UTC(), s.touched.Round(0).UTC()
	if err := identity.observe(now); err != nil {
		return session{}, err
	}
	if touched.Before(created) || now.Before(created) || now.Before(touched) {
		identity.blocked = true
		return session{}, ErrState
	}
	if !now.Before(created.Add(AbsoluteLifetime)) || !now.Before(touched.Add(IdleLifetime)) {
		identity.blocked = true
		delete(a.sessions, id)
		return session{}, ErrUnauthorized
	}
	return s, nil
}
func (a *Auth) bindSession(ctx context.Context, id string) (*SessionAuthority, error) {
	if err := authorityContext(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrUnauthorized
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	original, ok := a.sessions[id]
	if !ok {
		return nil, ErrUnauthorized
	}
	if _, err := a.authoritySession(id, original.identity); err != nil {
		return nil, err
	}
	identity := original.identity
	if identity.binding != nil {
		if identity.binding.owner != a || identity.binding.identity != identity || identity.binding.sessionID != id || !validSecret(identity.context) {
			return nil, ErrState
		}
		if err := authorityContext(ctx); err != nil {
			return nil, err
		}
		return &SessionAuthority{identity.binding}, nil
	}
	if a.random == nil {
		return nil, ErrState
	}
	nonce, err := randomSecret(a.random)
	if err != nil {
		return nil, err
	}
	// The context is distinct from credentials and every retained live context.
	// An entropy failure/collision cannot partially install or replace a binding.
	if sha256.Sum256([]byte(nonce)) == a.digest {
		return nil, ErrState
	}
	for _, s := range a.sessions {
		if nonce == s.ID || nonce == s.CSRF || s.identity != nil && nonce == s.identity.context {
			return nil, ErrState
		}
	}
	for _, c := range a.challenges {
		if nonce == c.ID || nonce == c.CSRF {
			return nil, ErrState
		}
	}
	// Entropy acquisition can take time. Recheck original membership, civil clock,
	// expiry and context before publishing a capability into that same session.
	if _, err := a.authoritySession(id, identity); err != nil {
		return nil, err
	}
	if err := authorityContext(ctx); err != nil {
		return nil, err
	}
	identity.context = nonce
	identity.binding = &sessionBinding{a, id, identity}
	return &SessionAuthority{identity.binding}, nil
}
func (s *SessionAuthority) CheckLive(ctx context.Context) error {
	if err := authorityContext(ctx); err != nil {
		return err
	}
	if s == nil || s.binding == nil || s.binding.owner == nil {
		return ErrUnauthorized
	}
	b := s.binding
	a := b.owner
	a.mu.Lock()
	defer a.mu.Unlock()
	if b.identity == nil || b.identity.binding != b {
		return ErrUnauthorized
	}
	if _, err := a.authoritySession(b.sessionID, b.identity); err != nil {
		return err
	}
	if b.identity.binding != b || !validSecret(b.identity.context) {
		return ErrUnauthorized
	}
	return authorityContext(ctx)
}

// Context returns a bounded non-credential context only after a live check. It
// is correlation data, never enough to reconstruct or authenticate a capability.
func (s *SessionAuthority) Context(ctx context.Context) (string, error) {
	if err := s.CheckLive(ctx); err != nil {
		return "", err
	}
	// No writer modifies the installed context; logout/Close invalidate membership
	// instead of resetting it. A concurrent revocation still requires launch checks.
	return s.binding.identity.context, nil
}

type sessionAuthorityKey struct{}

// ProtectAction is an explicit POST-only application boundary for future typed
// adapters. It registers no route and calls no helper/storage/native operation.
// Preserve all M1 Host/Origin/cookie/CSRF/body caps before issuing authority.
func (g *Gate) ProtectAction(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g == nil || next == nil || r == nil {
			deny(w, 503)
			return
		}
		if !g.policy(w, r) {
			return
		}
		if r.Method != "POST" {
			deny(w, 405)
			return
		}
		if r.URL.RawQuery != "" {
			deny(w, 400)
			return
		}
		if err := authorityContext(r.Context()); err != nil {
			authError(w, err)
			return
		}
		session, ok := g.authorize(w, r, false)
		if !ok {
			return
		}
		authority, err := g.auth.bindSession(r.Context(), session.ID)
		if err != nil {
			authError(w, err)
			return
		}
		if err = authority.CheckLive(r.Context()); err != nil {
			authError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), csrfKey{}, session.CSRF)
		ctx = context.WithValue(ctx, sessionAuthorityKey{}, authority)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func RequestSessionAuthority(r *http.Request) *SessionAuthority {
	if r == nil {
		return nil
	}
	value, _ := r.Context().Value(sessionAuthorityKey{}).(*SessionAuthority)
	return value
}
