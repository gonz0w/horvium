// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestActionPeerCannotBeReconstructed(t *testing.T) {
	for _, p := range []*actionPeer{nil, {}, new(actionPeer)} {
		if p.checkLive() == nil {
			t.Fatal("unissued peer live")
		}
		if identity, err := p.identity(); err == nil || identity != (actionIdentity{}) {
			t.Fatal("unissued identity published")
		}
		if err := p.Close(); err != nil {
			t.Fatal("zero close", err)
		}
	}
	var decoded actionPeer
	if json.Unmarshal([]byte(`{}`), &decoded) != nil || decoded.checkLive() == nil {
		t.Fatal("decoded peer live")
	}
}
func TestActionPeerFactoriesDenyMissingConnection(t *testing.T) {
	if p, err := acceptActionPeer(context.Background(), nil, 42, time.Second); p != nil || err == nil {
		t.Fatal("nil accepted connection")
	}
	if p, err := connectActionPeer(context.Background(), nil, 42, "request-1", "forged", time.Second); p != nil || err == nil {
		t.Fatal("nil root connection")
	}
}
func TestActionPeerInvalidConfigurationClosesConnection(t *testing.T) {
	for _, name := range []string{"nil-context", "root-uid", "overflow-uid", "zero-timeout", "long-timeout", "invalid-request", "invalid-context"} {
		t.Run(name, func(t *testing.T) {
			client, server := connected(t)
			ctx := context.Background()
			uid := uint32(42)
			timeout := time.Second
			if name == "nil-context" {
				ctx = nil
			}
			if name == "root-uid" {
				uid = 0
			}
			if name == "overflow-uid" {
				uid = ^uint32(0)
			}
			if name == "zero-timeout" {
				timeout = 0
			}
			if name == "long-timeout" {
				timeout = 5*time.Minute + 1
			}
			var peer *actionPeer
			var err error
			if name == "invalid-request" || name == "invalid-context" {
				id, scope := "request-1", strings.Repeat("a", 64)
				if name == "invalid-request" {
					id = "../fixture"
				} else {
					scope = "forged"
				}
				peer, err = connectActionPeer(ctx, server, uid, id, scope, timeout)
				if !errors.Is(err, ErrProtocol) {
					t.Fatal("invalid input not denied", err)
				}
			} else {
				peer, err = acceptActionPeer(ctx, server, uid, timeout)
				if !errors.Is(err, ErrConfig) {
					t.Fatal("invalid config not denied", err)
				}
			}
			if peer != nil {
				t.Fatal("invalid connection published")
			}
			var b [1]byte
			if n, err := client.Read(b[:]); n != 0 || err == nil {
				t.Fatal("failed connection leaked bytes or remained open", err)
			}
		})
	}
}
func TestActionPeerUnsupportedHostClosesConnection(t *testing.T) {
	if nativeSupported {
		return
	}
	for _, role := range []string{"helper", "application"} {
		t.Run(role, func(t *testing.T) {
			client, server := connected(t)
			var peer *actionPeer
			var err error
			if role == "helper" {
				peer, err = acceptActionPeer(context.Background(), server, 42, time.Second)
			} else {
				peer, err = connectActionPeer(context.Background(), server, 42, "request-1", strings.Repeat("a", 64), time.Second)
			}
			if peer != nil || !errors.Is(err, ErrUnsupported) {
				t.Fatal("host masqueraded as FreeBSD", err)
			}
			var b [1]byte
			if n, err := client.Read(b[:]); n != 0 || err == nil {
				t.Fatal("unsupported connection retained", err)
			}
		})
	}
}
