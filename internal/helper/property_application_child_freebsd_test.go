// SPDX-License-Identifier: BSD-3-Clause
package helper_test

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/helper"
	"github.com/gonz0w/horvium/internal/storage"
)

// External test package avoids admin -> helper -> admin test-import cycles.
// Only the original public application contracts are available to this child.
func applicationAuthority(t *testing.T, c helper.Config) (*admin.Auth, admin.Session, *admin.SessionAuthority) {
	t.Helper()
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		t.Fatal("fixture entropy")
	}
	token := hex.EncodeToString(secret[:])
	a := admin.NewAuth(sha256.Sum256([]byte(token)))
	t.Cleanup(a.Close)
	challenge, err := a.Challenge("127.0.0.1")
	if err != nil {
		t.Fatal("actual challenge")
	}
	session, err := a.Login("127.0.0.1", challenge.ID, challenge.CSRF, token)
	if err != nil {
		t.Fatal("actual login")
	}
	config := admin.Config{Version: 1, Listen: "127.0.0.1:8080", AllowedHosts: []string{"localhost:8080"}, ManagementUID: c.ManagementUID, SocketGID: c.SocketGID, HelperSocket: c.Socket, StateDirectory: "/var/db/horvium", MaxBytes: 64 << 20, MaxRecords: 100000, MaxFieldBytes: 4096, MaxXMLDepth: 32, MaxXMLElements: 250000, MaxFrameBytes: 64 << 10, MaxJSONBytes: 64 << 20, MaxTransportRecords: 100000, MaxJSONDepth: 32, IntervalSeconds: 30, TimeoutSeconds: 3, StaleSeconds: 90, MaxReaders: 16, MaxHelperClients: 1}
	gate, err := admin.NewGate(config, a)
	if err != nil {
		t.Fatal("actual gate configuration")
	}
	req := httptest.NewRequest("POST", "/fixture-action", strings.NewReader(url.Values{"csrf": {session.CSRF}}.Encode()))
	req.Host = "localhost:8080"
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(&http.Cookie{Name: admin.SessionCookie, Value: session.ID})
	var authority *admin.SessionAuthority
	recorder := httptest.NewRecorder()
	gate.ProtectAction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authority = admin.RequestSessionAuthority(r)
		if admin.RequestCSRF(r) != session.CSRF {
			t.Error("original CSRF contract")
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent || authority == nil {
		t.Fatal("actual action gate denied")
	}
	scope, err := authority.Context(context.Background())
	if err != nil || len(scope) != 64 || scope == token || scope == session.ID || scope == session.CSRF {
		t.Fatal("credential-free original context")
	}
	return a, session, authority
}
func TestNativeApplicationPropertyChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_APPLICATION_PROPERTY_CHILD")
	if mode == "" {
		t.Skip("reexec only")
	}
	var c helper.Config
	if json.Unmarshal([]byte(os.Getenv("HORVIUM_HELPER_CONFIG")), &c) != nil {
		t.Fatal("fixture configuration")
	}
	if uint32(os.Getuid()) != c.ManagementUID || uint32(os.Geteuid()) != c.ManagementUID || c.ManagementUID == 0 {
		t.Fatal("child credentials were not dropped before exec")
	}
	a, session, authority := applicationAuthority(t, c)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	intent := storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "1"}
	expected := error(nil)
	switch mode {
	case "logout-before-dial":
		a.Logout(session.ID)
		expected = admin.ErrUnauthorized
	case "close-before-dial":
		a.Close()
		expected = admin.ErrClosed
	case "decoded-authority":
		var decoded admin.SessionAuthority
		if json.Unmarshal([]byte(`{}`), &decoded) != nil {
			t.Fatal("decoded fixture")
		}
		authority = &decoded
		expected = admin.ErrUnauthorized
	case "invalid-selection":
		intent.Value = "0"
		expected = helper.ErrProtocol
	case "cancelled-before-dial":
		cancel()
		expected = context.Canceled
	}
	var revoked chan error
	if strings.Contains(mode, "after-") {
		revoked = make(chan error, 1)
		expected = admin.ErrUnauthorized
		if mode == "close-after-request" {
			expected = admin.ErrClosed
		}
		go func() {
			line, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil || line != "revoke\n" {
				revoked <- errors.New("invalid revoke signal")
				return
			}
			if mode == "close-after-request" {
				a.Close()
			} else {
				a.Logout(session.ID)
			}
			if err := authority.CheckLive(ctx); !errors.Is(err, expected) {
				revoked <- errors.New("original store remained live")
				return
			}
			fmt.Println("revoked")
			revoked <- nil
		}()
	}
	var err error
	if mode == "submit-denied" {
		job, duplicate, e := admin.SubmitConfiguredProperty(ctx, c, "application", "100", strings.Repeat("a", 64), strings.Repeat("b", 64), authority)
		err = e
		if !reflect.DeepEqual(job, storage.Job{}) || duplicate {
			t.Fatal("denial exposed admission")
		}
	} else {
		review, e := admin.ReviewConfiguredProperty(ctx, c, "application", "100", intent, authority)
		err = e
		if !reflect.DeepEqual(review, storage.Review{}) {
			t.Fatal("denial exposed review")
		}
	}
	if expected != nil {
		if !errors.Is(err, expected) {
			t.Fatal("original authority did not gate configured exchange")
		}
	} else {
		var refusal *helper.PropertyRefusalError
		if !errors.As(err, &refusal) || refusal.Code != "denied" {
			t.Fatal("root denial contract lost")
		}
	}
	if revoked != nil {
		select {
		case err := <-revoked:
			if err != nil {
				t.Fatal("original store revocation")
			}
		case <-ctx.Done():
			t.Fatal("revocation fixture did not finish")
		}
	}
	if expected == admin.ErrUnauthorized || expected == admin.ErrClosed {
		if authority.CheckLive(context.Background()) == nil {
			t.Fatal("authority revived after terminal exchange")
		}
	}
}
