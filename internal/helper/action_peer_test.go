// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"encoding/json"
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
