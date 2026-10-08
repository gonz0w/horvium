// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPropertyDialPreflight(t *testing.T) {
	for _, name := range []string{"relative", "uid", "limits", "timeout", "capacity", "id", "scope", "check", "nil-context", "cancelled", "unsupported"} {
		if name == "unsupported" && nativeSupported {
			continue
		}
		t.Run(name, func(t *testing.T) {
			c := Config{Socket: "/var/run/horvium/helper.sock", ManagementUID: 42, SocketGID: 42, Limits: DefaultLimits(), Timeout: time.Second, MaxClients: 1}
			ctx := context.Background()
			id, scope := "dial", strings.Repeat("a", 64)
			calls := 0
			check := func(context.Context) error { calls++; return nil }
			want := ErrConfig
			switch name {
			case "relative":
				c.Socket = "relative"
			case "uid":
				c.ManagementUID = 0
			case "limits":
				c.Limits = Limits{}
			case "timeout":
				c.Timeout = 0
			case "capacity":
				c.MaxClients = 65
			case "id":
				id = "../dial"
			case "scope":
				scope = "cookie"
			case "check":
				check = nil
			case "nil-context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
				want = context.Canceled
			case "unsupported":
				want = ErrUnsupported
			}
			if p, err := DialPropertyClient(ctx, c, id, scope, check); p != nil || !errors.Is(err, want) || calls != 0 {
				t.Fatal("preflight allowed I/O/application check", err, calls)
			}
		})
	}
}
func TestPropertyDialOriginalLifetimeRetirement(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Private owner fixture only; no socket, root credential or factory is supplied.
	p := &PropertyClient{exchange: &propertyPeer{peer: &actionPeer{}, application: func() error { return nil }}, cancel: cancel}
	p.identity = p
	copy := *p
	if copy.Close() != nil || ctx.Err() != nil {
		t.Fatal("copied client retired original dial lifetime")
	}
	if p.Close() != nil || !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("original dial lifetime retained")
	}
	if p.Close() != nil {
		t.Fatal("repeat retirement")
	}
}
