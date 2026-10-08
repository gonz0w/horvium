// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type negotiationTransport struct {
	request  bytes.Buffer
	response *bytes.Reader
	onWrite  func()
	onRead   func()
	alter    func([]byte) []byte
}

func (w *negotiationTransport) Write(b []byte) (int, error) {
	if w.onWrite != nil {
		w.onWrite()
	}
	return w.request.Write(b)
}
func (w *negotiationTransport) Read(b []byte) (int, error) {
	if w.response == nil {
		if w.onRead != nil {
			w.onRead()
		}
		data := bytes.Replace(bytes.Replace(w.request.Bytes(), []byte(`"operation"`), []byte(`"type"`), 1), []byte(`"negotiate"`), []byte(`"negotiated"`), 1)
		if w.alter != nil {
			data = w.alter(data)
		}
		if len(data) >= 4 {
			binary.BigEndian.PutUint32(data, uint32(len(data)-4))
		}
		w.response = bytes.NewReader(data)
	}
	return w.response.Read(b)
}
func TestHelperNegotiationOriginalSession(t *testing.T) {
	a, _, session, authority, now, token := authorityFixture(t)
	touched := a.sessions[session.ID].touched
	var transport negotiationTransport
	transport.onRead = func() { *now = now.Add(time.Second) }
	if err := negotiateHelper(context.Background(), &transport, "request-1", authority); err != nil {
		t.Fatal(err)
	}
	scope, err := authority.Context(context.Background())
	if err != nil || !bytes.Contains(transport.request.Bytes(), []byte(scope)) {
		t.Fatal("original context lost")
	}
	for _, credential := range []string{session.ID, session.CSRF, token} {
		if bytes.Contains(transport.request.Bytes(), []byte(credential)) {
			t.Fatal("credential sent to helper")
		}
	}
	copy := *authority
	if err := negotiateHelper(context.Background(), &negotiationTransport{}, "request-2", &copy); err != nil {
		t.Fatal("copy lost original session", err)
	}
	if !a.sessions[session.ID].touched.Equal(touched) {
		t.Fatal("negotiation renewed idle lifetime")
	}
}

type failedNegotiation struct {
	writes, reads int
	writeFailure  bool
}

func (f *failedNegotiation) Write([]byte) (int, error) {
	f.writes++
	if f.writeFailure {
		return 0, io.ErrClosedPipe
	}
	return 0, nil
}
func (f *failedNegotiation) Read([]byte) (int, error) { f.reads++; return 0, io.ErrUnexpectedEOF }
func TestHelperNegotiationNoRetryAfterWriteFailure(t *testing.T) {
	for _, name := range []string{"error", "short-write"} {
		t.Run(name, func(t *testing.T) {
			_, _, _, authority, _, _ := authorityFixture(t)
			wire := &failedNegotiation{writeFailure: name == "error"}
			err := negotiateHelper(context.Background(), wire, "request-1", authority)
			want := io.ErrShortWrite
			if wire.writeFailure {
				want = io.ErrClosedPipe
			}
			if !errors.Is(err, want) || wire.writes != 1 || wire.reads != 0 {
				t.Fatal("failed write retried or response awaited", err)
			}
		})
	}
}
func TestHelperNegotiationAuthorityDenials(t *testing.T) {
	for _, name := range []string{"nil", "zero", "decoded", "logout", "closed", "cancelled", "nil-context", "nil-transport", "invalid-id"} {
		t.Run(name, func(t *testing.T) {
			a, _, s, authority, _, _ := authorityFixture(t)
			ctx := context.Background()
			id := "request-1"
			var transport negotiationTransport
			var wire io.ReadWriter = &transport
			switch name {
			case "nil":
				authority = nil
			case "zero":
				authority = &SessionAuthority{}
			case "decoded":
				var decoded SessionAuthority
				if json.Unmarshal([]byte(`{}`), &decoded) != nil {
					t.Fatal("fixture")
				}
				authority = &decoded
			case "logout":
				a.Logout(s.ID)
			case "closed":
				a.Close()
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "nil-context":
				ctx = nil
			case "nil-transport":
				wire = nil
			case "invalid-id":
				id = "../fixture"
			}
			if negotiateHelper(ctx, wire, id, authority) == nil || transport.request.Len() != 0 {
				t.Fatal("denied negotiation emitted request")
			}
		})
	}
}
func TestHelperNegotiationFinalSessionChecks(t *testing.T) {
	for _, name := range []string{"logout-during-write", "logout-during-read", "expiry-during-read", "cancel-during-read", "wrong-request", "wrong-context", "old-version", "unknown-response", "cut-response"} {
		t.Run(name, func(t *testing.T) {
			a, _, s, authority, now, _ := authorityFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var wire negotiationTransport
			switch name {
			case "logout-during-write":
				wire.onWrite = func() { a.Logout(s.ID) }
			case "logout-during-read":
				wire.onRead = func() { a.Logout(s.ID) }
			case "expiry-during-read":
				wire.onRead = func() { *now = now.Add(IdleLifetime) }
			case "cancel-during-read":
				wire.onRead = cancel
			case "wrong-request":
				wire.alter = func(b []byte) []byte { return bytes.Replace(b, []byte("request-1"), []byte("request-2"), 1) }
			case "wrong-context":
				scope, _ := authority.Context(ctx)
				wire.alter = func(b []byte) []byte { return bytes.Replace(b, []byte(scope), []byte(strings.Repeat("f", 64)), 1) }
			case "old-version":
				wire.alter = func(b []byte) []byte { return bytes.Replace(b, []byte(`"version":2`), []byte(`"version":1`), 1) }
			case "unknown-response":
				wire.alter = func(b []byte) []byte { return bytes.Replace(b, []byte("negotiated"), []byte("accepted"), 1) }
			case "cut-response":
				wire.alter = func(b []byte) []byte { return b[:len(b)-1] }
			}
			if negotiateHelper(ctx, &wire, "request-1", authority) == nil {
				t.Fatal("unverified exchange accepted")
			}
			if name == "logout-during-write" && wire.response != nil {
				t.Fatal("response awaited after revoked write")
			}
		})
	}
}
