// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gonz0w/horvium/internal/helper"
	"github.com/gonz0w/horvium/internal/storage"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestPropertyAdapterSelectionBeforeNegotiation(t *testing.T) {
	for _, name := range []string{"review-selection", "review-authority", "submit-selection", "submit-authority", "unsupported"} {
		t.Run(name, func(t *testing.T) {
			_, _, _, authority, _, _ := authorityFixture(t)
			dir, err := os.MkdirTemp("", "hp-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if os.RemoveAll(dir) != nil {
					t.Error("fixture removal")
				}
			})
			listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(dir, "s"), Net: "unix"})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			client, err := net.DialUnix("unix", nil, listener.Addr().(*net.UnixAddr))
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			server, err := listener.AcceptUnix()
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			_ = server.SetDeadline(time.Now().Add(time.Second))
			id, pool, plan, submission := "property-1", "100", strings.Repeat("a", 64), strings.Repeat("b", 64)
			intent := storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "1"}
			if name == "review-selection" {
				intent.Value = "0"
				authority = nil
			}
			if name == "submit-selection" {
				plan = "../plan"
				authority = nil
			}
			if strings.HasSuffix(name, "authority") {
				authority = nil
			}
			var e error
			if strings.HasPrefix(name, "submit") {
				j, d, err := submitHelperProperty(context.Background(), client, 42, id, pool, plan, submission, time.Second, authority)
				e = err
				if j.ID != "" || d {
					t.Fatal("denied submission projected metadata")
				}
			} else {
				r, err := reviewHelperProperty(context.Background(), client, 42, id, pool, intent, time.Second, authority)
				e = err
				if r.ID != "" {
					t.Fatal("denied review projected metadata")
				}
			}
			if e == nil {
				t.Fatal("invalid adapter accepted")
			}
			if strings.HasSuffix(name, "selection") && !errors.Is(e, helper.ErrProtocol) {
				t.Fatal("selection not validated before authority/negotiation", e)
			}
			if strings.HasSuffix(name, "authority") && !errors.Is(e, ErrUnauthorized) {
				t.Fatal("concrete original authority not required", e)
			}
			// On FreeBSD the root test process cannot impersonate configured UID 42;
			// on other hosts the actual factory must report unsupported, not authorize.
			if name == "unsupported" && runtime.GOOS != "freebsd" && !errors.Is(e, helper.ErrUnsupported) {
				t.Fatal("unsupported factory replaced", e)
			}
			var b [1]byte
			if n, err := server.Read(b[:]); n != 0 || err != io.EOF {
				t.Fatal("denial negotiated or retained socket", err)
			}
		})
	}
}

func TestConfiguredPropertyAdapterSelection(t *testing.T) {
	for _, name := range []string{"review-selection", "submit-selection", "review-authority", "submit-authority", "config", "unsupported"} {
		t.Run(name, func(t *testing.T) {
			_, _, _, authority, _, _ := authorityFixture(t)
			c := helper.Config{Socket: "/var/run/horvium/helper.sock", ManagementUID: 42, SocketGID: 42, Limits: helper.DefaultLimits(), Timeout: time.Second, MaxClients: 1}
			intent := storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "1"}
			plan, submission := strings.Repeat("a", 64), strings.Repeat("b", 64)
			if name == "review-selection" {
				intent.Value = "0"
				authority = nil
			}
			if name == "submit-selection" {
				plan = "../plan"
				authority = nil
			}
			if strings.HasSuffix(name, "authority") {
				authority = nil
			}
			if name == "config" {
				c.MaxClients = 65
			}
			var err error
			if strings.HasPrefix(name, "submit") {
				j, d, e := SubmitConfiguredProperty(context.Background(), c, "configured", "100", plan, submission, authority)
				err = e
				if j.ID != "" || d {
					t.Fatal("denial projected admission")
				}
			} else {
				r, e := ReviewConfiguredProperty(context.Background(), c, "configured", "100", intent, authority)
				err = e
				if r.ID != "" {
					t.Fatal("denial projected review")
				}
			}
			if err == nil {
				t.Fatal("unsafe configured adapter accepted")
			}
			if strings.HasSuffix(name, "selection") && !errors.Is(err, helper.ErrProtocol) {
				t.Fatal("selection not checked before session/dial", err)
			}
			if strings.HasSuffix(name, "authority") && !errors.Is(err, ErrUnauthorized) {
				t.Fatal("original authority bypassed", err)
			}
			if name == "config" && !errors.Is(err, helper.ErrConfig) {
				t.Fatal("config not validated before connection", err)
			}
			if name == "unsupported" && runtime.GOOS != "freebsd" && !errors.Is(err, helper.ErrUnsupported) {
				t.Fatal("unsupported connector used", err)
			}
		})
	}
}
