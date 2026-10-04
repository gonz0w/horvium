// SPDX-License-Identifier: BSD-3-Clause
package console

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type blockedReply struct {
	*httptest.ResponseRecorder
	entered chan struct{}
	release chan struct{}
}

func (w *blockedReply) Write(b []byte) (int, error) {
	close(w.entered)
	<-w.release
	return w.ResponseRecorder.Write(b)
}
func TestAdmissionCoversResponseWriteAndLeavesAuthResponsive(t *testing.T) {
	_, m, a := fixture(t)
	c := fixtureConfig("127.0.0.1:8088")
	c.MaxReaders = 1
	h, e := New(c, a, m)
	if e != nil {
		t.Fatal(e)
	}
	cookie, _ := login(t, h)
	r := req("GET", "/storage/datasets", "")
	r.AddCookie(cookie)
	w := &blockedReply{httptest.NewRecorder(), make(chan struct{}), make(chan struct{})}
	done := make(chan struct{})
	go func() { defer close(done); h.ServeHTTP(w, r) }()
	defer func() { close(w.release); <-done }()
	select {
	case <-w.entered:
	case <-time.After(time.Second):
		t.Fatal("response did not begin")
	}
	for _, path := range []string{"/api/datasets", "/storage/pools"} {
		r := req("GET", path, "")
		r.AddCookie(cookie)
		got := serve(h, r)
		if got.Code != http.StatusServiceUnavailable {
			t.Fatalf("admitted while response blocked: %s status=%d", path, got.Code)
		}
	}
	if got := serve(h, req("GET", "/api/datasets", "")); got.Code != 401 {
		t.Fatal("admission bypassed authentication")
	}
	if got := serve(h, req("GET", "/auth/challenge", "")); got.Code != 200 {
		t.Fatal("challenge blocked by inventory reader")
	}
	r = req("GET", "/status", "")
	r.AddCookie(cookie)
	if got := serve(h, r); got.Code != 200 {
		t.Fatal("small collector state blocked by inventory reader")
	}
}
