// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

func helloPayload() string {
	return `{"version":2,"request_id":"request-1","operation":"negotiate","session_context":"` + strings.Repeat("a", 64) + `"}`
}
func readyPayload() string {
	return strings.Replace(strings.Replace(helloPayload(), `"operation"`, `"type"`, 1), `"negotiate"`, `"negotiated"`, 1)
}
func TestActionNegotiationRoundTrip(t *testing.T) {
	id, scope := "request-1", strings.Repeat("a", 64)
	var request, response bytes.Buffer
	if err := WriteActionNegotiation(&request, id, scope); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(request.Bytes(), frame(helloPayload())) {
		t.Fatal("request changed")
	}
	h, err := readActionNegotiation(&request)
	if err != nil || h.ID != id || h.Context != scope {
		t.Fatal("lost correlation", err)
	}
	if err := writeActionReady(&response, h); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(response.Bytes(), frame(readyPayload())) {
		t.Fatal("response changed")
	}
	if err := ReadActionNegotiation(&response, id, scope); err != nil {
		t.Fatal(err)
	}
}
func TestActionNegotiationRejectsClaimsAndDowngrades(t *testing.T) {
	for name, change := range map[string]func(string) string{
		"old-version":      func(s string) string { return strings.Replace(s, `"version":2`, `"version":1`, 1) },
		"future-version":   func(s string) string { return strings.Replace(s, `"version":2`, `"version":3`, 1) },
		"floating-version": func(s string) string { return strings.Replace(s, `"version":2`, `"version":2.0`, 1) },
		"duplicate":        func(s string) string { return strings.Replace(s, `"version":2`, `"version":2,"version":2`, 1) },
		"case":             func(s string) string { return strings.Replace(s, `"version"`, `"Version"`, 1) },
		"uid":              func(s string) string { return strings.Replace(s, `{"version"`, `{"uid":0,"version"`, 1) },
		"actor":            func(s string) string { return strings.Replace(s, `{"version"`, `{"actor":"root","version"`, 1) },
		"command":          func(s string) string { return strings.Replace(s, `{"version"`, `{"command":"zfs","version"`, 1) },
		"path":             func(s string) string { return strings.Replace(s, `{"version"`, `{"path":"/dev/ada0","version"`, 1) },
		"properties": func(s string) string {
			return strings.Replace(s, `{"version"`, `{"properties":{"quota":"1"},"version"`, 1)
		},
		"missing-context": func(s string) string {
			return strings.Replace(s, `,"session_context":"`+strings.Repeat("a", 64)+`"`, "", 1)
		},
		"null-context":   func(s string) string { return strings.Replace(s, `"`+strings.Repeat("a", 64)+`"`, "null", 1) },
		"short-context":  func(s string) string { return strings.Replace(s, strings.Repeat("a", 64), "abc", 1) },
		"upper-context":  func(s string) string { return strings.Replace(s, strings.Repeat("a", 64), strings.Repeat("A", 64), 1) },
		"nonhex-context": func(s string) string { return strings.Replace(s, strings.Repeat("a", 64), strings.Repeat("g", 64), 1) },
		"bad-id":         func(s string) string { return strings.Replace(s, "request-1", "../fixture", 1) },
		"trailing-json":  func(s string) string { return s + `{}` },
		"invalid-utf8":   func(s string) string { return strings.Replace(s, "request-1", "\xff", 1) },
	} {
		t.Run(name, func(t *testing.T) {
			h, err := readActionNegotiation(bytes.NewReader(frame(change(helloPayload()))))
			if err == nil || h != (actionHello{}) {
				t.Fatal("malformed request published")
			}
			if ReadActionNegotiation(bytes.NewReader(frame(change(readyPayload()))), "request-1", strings.Repeat("a", 64)) == nil {
				t.Fatal("malformed response accepted")
			}
		})
	}
	for _, operation := range []string{"inventory", "set", "destroy", "property", ""} {
		t.Run("operation-"+operation, func(t *testing.T) {
			if _, err := readActionNegotiation(bytes.NewReader(frame(strings.Replace(helloPayload(), "negotiate", operation, 1)))); err == nil {
				t.Fatal("operation accepted as negotiation")
			}
		})
	}
	for _, kind := range []string{"inventory", "success", "accepted", ""} {
		t.Run("response-"+kind, func(t *testing.T) {
			if ReadActionNegotiation(bytes.NewReader(frame(strings.Replace(readyPayload(), "negotiated", kind, 1))), "request-1", strings.Repeat("a", 64)) == nil {
				t.Fatal("operation result accepted as negotiation")
			}
		})
	}
}
func TestActionNegotiationCorrelation(t *testing.T) {
	for name, value := range map[string]string{"wrong-request": "other", "wrong-context": strings.Repeat("b", 64), "invalid-id": "../fixture", "invalid-context": "forged"} {
		t.Run(name, func(t *testing.T) {
			id, scope := "request-1", strings.Repeat("a", 64)
			if strings.Contains(name, "request") || strings.Contains(name, "id") {
				id = value
			} else {
				scope = value
			}
			if ReadActionNegotiation(bytes.NewReader(frame(readyPayload())), id, scope) == nil {
				t.Fatal("response crossed correlation")
			}
		})
	}
}
func TestActionNegotiationBoundsAndCuts(t *testing.T) {
	for _, payload := range []string{helloPayload(), readyPayload()} {
		data := frame(payload)
		for cut := 0; cut < len(data); cut++ {
			if strings.Contains(payload, "operation") {
				if h, err := readActionNegotiation(bytes.NewReader(data[:cut])); err == nil || h != (actionHello{}) {
					t.Fatal("cut request accepted", cut)
				}
			} else if ReadActionNegotiation(bytes.NewReader(data[:cut]), "request-1", strings.Repeat("a", 64)) == nil {
				t.Fatal("cut response accepted", cut)
			}
		}
	}
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, 513)
	if _, err := readActionNegotiation(bytes.NewReader(header)); !errors.Is(err, ErrLimit) {
		t.Fatal("oversize not refused before payload read", err)
	}
	if ReadActionNegotiation(bytes.NewReader(header), "request-1", strings.Repeat("a", 64)) != ErrLimit {
		t.Fatal("oversize response")
	}
	for name, payload := range map[string]string{"oversize": strings.Repeat(" ", 513), "long-field": strings.Replace(helloPayload(), "request-1", strings.Repeat("r", 65), 1), "depth": `{"version":{"a":{"b":{}}}}`} {
		t.Run(name, func(t *testing.T) {
			if _, err := readActionNegotiation(bytes.NewReader(frame(payload))); err == nil {
				t.Fatal("bound bypassed")
			}
		})
	}
}
func TestActionNegotiationWriterDenials(t *testing.T) {
	for name, pair := range map[string][2]string{"id": {"../fixture", strings.Repeat("a", 64)}, "context": {"request-1", "forged"}, "field-limit": {strings.Repeat("r", 65), strings.Repeat("a", 64)}} {
		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			if WriteActionNegotiation(&b, pair[0], pair[1]) == nil || b.Len() != 0 {
				t.Fatal("invalid request emitted")
			}
		})
	}
	for _, h := range []actionHello{{}, {2, "request-1", "destroy", strings.Repeat("a", 64)}, {1, "request-1", "negotiate", strings.Repeat("a", 64)}} {
		var b bytes.Buffer
		if writeActionReady(&b, h) == nil || b.Len() != 0 {
			t.Fatal("invalid negotiation acknowledged")
		}
	}
	if WriteActionNegotiation(nil, "request-1", strings.Repeat("a", 64)) == nil || ReadActionNegotiation(nil, "request-1", strings.Repeat("a", 64)) == nil {
		t.Fatal("nil transport accepted")
	}
	if _, err := readActionNegotiation(nil); err == nil {
		t.Fatal("nil reader")
	}
	if writeActionReady(nil, actionHello{2, "request-1", "negotiate", strings.Repeat("a", 64)}) == nil {
		t.Fatal("nil writer")
	}
	if err := WriteActionNegotiation(shortActionWriter{}, "request-1", strings.Repeat("a", 64)); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal("short write accepted", err)
	}
}

type shortActionWriter struct{}

func (shortActionWriter) Write([]byte) (int, error) { return 0, nil }
func TestActionNegotiationInventoryIsolation(t *testing.T) {
	if _, err := readRequest(bytes.NewReader(frame(helloPayload())), DefaultLimits()); err == nil {
		t.Fatal("read-only helper accepted negotiation")
	}
	var b bytes.Buffer
	if err := writeRequest(&b, "request-1", DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if _, err := readActionNegotiation(&b); err == nil {
		t.Fatal("inventory upgraded implicitly")
	}
}
func TestActionNegotiationReadOnlyServerDenial(t *testing.T) {
	client, server := connected(t)
	var collections atomic.Int32
	s := testServer(collectFunc(func(context.Context) inventory.Snapshot {
		collections.Add(1)
		return observation()
	}), func(*net.UnixConn) (uint32, error) { return 42, nil }) // Model peer only.
	done := make(chan struct{})
	go func() { s.handle(context.Background(), server); close(done) }()
	if _, err := client.Write(frame(helloPayload())); err != nil {
		t.Fatal(err)
	}
	if err := client.CloseWrite(); err != nil {
		t.Fatal(err)
	}
	if err := ReadActionNegotiation(client, "request-1", strings.Repeat("a", 64)); err == nil {
		t.Fatal("running inventory server negotiated")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("read-only denial did not stop")
	}
	if collections.Load() != 0 {
		t.Fatal("negotiation reached collector")
	}
}
func TestActionNegotiationFrameBoundary(t *testing.T) {
	// A later typed message stays unread. Negotiation alone never dispatches it.
	extra := frame(`{"version":2,"operation":"destroy"}`)
	r := bytes.NewReader(append(frame(helloPayload()), extra...))
	if _, err := readActionNegotiation(r); err != nil || r.Len() != len(extra) {
		t.Fatal("request consumed later frame", err)
	}
	r = bytes.NewReader(append(frame(readyPayload()), extra...))
	if err := ReadActionNegotiation(r, "request-1", strings.Repeat("a", 64)); err != nil || r.Len() != len(extra) {
		t.Fatal("response consumed later frame", err)
	}
}
func FuzzActionNegotiation(f *testing.F) {
	f.Add(frame(helloPayload()))
	f.Add(frame(readyPayload()))
	f.Add([]byte{255, 255, 255, 255})
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 516 {
			return
		}
		_ = ReadActionNegotiation(bytes.NewReader(raw), "request-1", strings.Repeat("a", 64))
		h, err := readActionNegotiation(bytes.NewReader(raw))
		if err != nil {
			if h != (actionHello{}) {
				t.Fatal("failed decode published")
			}
			return
		}
		var response bytes.Buffer
		if writeActionReady(&response, h) != nil || ReadActionNegotiation(&response, h.ID, h.Context) != nil {
			t.Fatal("accepted request cannot negotiate")
		}
		if _, err := readRequest(bytes.NewReader(raw), DefaultLimits()); err == nil {
			t.Fatal("protocol overlap")
		}
	})
}
