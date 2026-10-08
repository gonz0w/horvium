// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gonz0w/horvium/internal/storage"
	"strings"
	"testing"
	"time"
)

func TestPropertyClientUnissued(t *testing.T) {
	var decoded PropertyClient
	if json.Unmarshal([]byte(`{}`), &decoded) != nil {
		t.Fatal("fixture")
	}
	for _, p := range []*PropertyClient{nil, {}, &decoded} {
		if r, err := p.Review("100", propertyClientIntent()); err == nil || r.ID != "" {
			t.Fatal("unissued review")
		}
		if j, d, err := p.Submit("100", strings.Repeat("a", 64), strings.Repeat("b", 64)); err == nil || d || j.ID != "" {
			t.Fatal("unissued submission")
		}
		if p.Close() != nil {
			t.Fatal("unissued close")
		}
	}
}
func propertyClientIntent() storage.Intent {
	return storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "1099511627776"}
}
func TestPropertyClientFactoryClosesOwnedSocket(t *testing.T) {
	for _, name := range []string{"nil-check", "invalid-id", "invalid-scope", "unsupported"} {
		if name == "unsupported" && nativeSupported {
			continue
		}
		t.Run(name, func(t *testing.T) {
			client, server := connected(t)
			id, scope := "request-1", strings.Repeat("a", 64)
			check := func(context.Context) error { return nil }
			switch name {
			case "nil-check":
				check = nil
			case "invalid-id":
				id = "../request"
			case "invalid-scope":
				scope = "cookie"
			}
			p, err := ConnectPropertyClient(context.Background(), server, 42, id, scope, time.Second, check)
			if p != nil || err == nil {
				t.Fatal("invalid factory accepted")
			}
			if name == "unsupported" && !errors.Is(err, ErrUnsupported) {
				t.Fatal("kernel identity bypass", err)
			}
			var b [1]byte
			if n, err := client.Read(b[:]); n != 0 || err == nil {
				t.Fatal("factory emitted bytes/retained socket")
			}
		})
	}
}
