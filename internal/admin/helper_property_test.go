// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestPropertySessionOriginalBinding(t *testing.T) {
	a, _, s, authority, now, token := authorityFixture(t)
	scope, err := authority.Context(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	touched := a.sessions[s.ID].touched
	copy := *authority
	for _, p := range []*SessionAuthority{authority, &copy} {
		check := propertySessionCheck(p, scope)
		*now = now.Add(time.Second)
		if err := check(context.Background()); err != nil {
			t.Fatal("original session denied", err)
		}
	}
	if !a.sessions[s.ID].touched.Equal(touched) {
		t.Fatal("property check renewed idle time")
	}
	for _, v := range []string{s.ID, s.CSRF, token} {
		if scope == v {
			t.Fatal("credential used as correlation")
		}
	}
}
func TestPropertySessionPhaseRevocation(t *testing.T) {
	for _, name := range []string{"nil", "zero", "decoded", "wrong-scope", "logout", "close", "expiry", "rollback", "cancelled", "nil-context"} {
		t.Run(name, func(t *testing.T) {
			a, _, s, authority, now, _ := authorityFixture(t)
			scope, err := authority.Context(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			switch name {
			case "nil":
				authority = nil
			case "zero":
				authority = &SessionAuthority{}
			case "decoded":
				var p SessionAuthority
				if json.Unmarshal([]byte(`{}`), &p) != nil {
					t.Fatal("fixture")
				}
				authority = &p
			case "wrong-scope":
				scope = ""
			}
			check := propertySessionCheck(authority, scope)
			if name != "nil" && name != "zero" && name != "decoded" && name != "wrong-scope" {
				if check(ctx) != nil {
					t.Fatal("initial check")
				}
			}
			switch name {
			case "logout":
				a.Logout(s.ID)
			case "close":
				a.Close()
			case "expiry":
				*now = now.Add(IdleLifetime)
			case "rollback":
				*now = now.Add(-time.Second)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "nil-context":
				ctx = nil
			}
			if check(ctx) == nil {
				t.Fatal("revoked/reconstructed binding accepted")
			}
		})
	}
}
