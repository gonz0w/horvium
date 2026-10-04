// SPDX-License-Identifier: BSD-3-Clause
package client

import (
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/observation"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEndpointBoundary(t *testing.T) {
	for _, u := range []string{"https://127.0.0.1:80", "http://localhost:80", "http://192.0.2.1:80", "http://user@127.0.0.1:80", "http://127.0.0.1:80/", "http://127.0.0.1:80?", "http://[::ffff:127.0.0.1]:80", "http://127.0.0.1:0", "http://127.0.0.1:080"} {
		if _, e := New(u); e == nil {
			t.Fatal("unsafe endpoint accepted")
		}
	}
}
func TestNoCredentialRedirectOrRawError(t *testing.T) {
	calls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; t.Error("redirect followed") }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer source.Close()
	c, e := New(source.URL)
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Login(context.Background(), strings.Repeat("a", 64)); !errors.Is(e, ErrResponse) {
		t.Fatal("redirect accepted")
	}
	if calls != 0 {
		t.Fatal("credential forwarded")
	}
}
func TestRejectMalformedResponses(t *testing.T) {
	for _, body := range []string{`{"Generation":"a"} {}`, `{"unexpected":"secret"}`, `{"Generation":`} {
		t.Run("bounded failure", func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer s.Close()
			c, _ := New(s.URL)
			_, e := c.Query(context.Background(), "overview", observation.Query{})
			if !errors.Is(e, ErrResponse) {
				t.Fatal("malformed response accepted")
			}
		})
	}
}
