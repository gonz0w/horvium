// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/admin"
)

func abuseRaw(t *testing.T, address, request string, deadline time.Duration) int {
	t.Helper()
	c, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		t.Fatal("abuse fixture connection")
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(deadline))
	if _, err := io.WriteString(c, request); err != nil {
		t.Fatal("abuse fixture write")
	}
	r, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatal("abuse fixture response; input withheld")
	}
	defer r.Body.Close()
	b, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil || bytes.Contains(b, []byte("hostile-abuse-marker")) {
		t.Fatal("unbounded or reflected denial")
	}
	return r.StatusCode
}

func TestNativeBuiltHTTPAbuseAndRecovery(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("explicit disposable native opt-in required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("abuse fixture requires root")
	}
	owned := nativeStorageFixture(t)
	c, path, token := nativeRuntimeConfig(t)
	c.MaxReaders = 2
	encoded, _ := json.Marshal(c)
	if os.WriteFile(path, encoded, 0644) != nil {
		t.Fatal("abuse config")
	}
	digestPath := filepath.Join(c.StateDirectory, "token.sha256")
	digest, err := os.ReadFile(digestPath)
	if err != nil {
		t.Fatal("abuse digest")
	}
	built, _ := filepath.Abs("../../.tools/horvium")
	binary := filepath.Join(filepath.Dir(path), "horvium")
	copyPerformanceExecutable(t, built, binary)
	helperLogs, httpLogs := &nativeHelperEventSink{}, &nativeHelperEventSink{}
	helper := startNativeProcessWithStderr(t, binary, []string{"helper", "-config", path}, 0, 0, helperLogs)
	nativeWait(t, func() bool { i, e := os.Stat(c.HelperSocket); return e == nil && i.Mode().Perm() == 0660 })
	management := startNativeProcessWithStderr(t, binary, []string{"serve", "-config", path}, c.ManagementUID, c.SocketGID, httpLogs)
	session := newPerformanceSession(t, c, token)
	nativeWait(t, func() bool { o := session.overview(t); return o.Complete && o.Generation != "" })
	u, _ := url.Parse(session.base)
	credential := ""
	for _, cookie := range session.http.Jar.Cookies(u) {
		if cookie.Name == admin.SessionCookie {
			credential = cookie.Name + "=" + cookie.Value
		}
	}
	if credential == "" {
		t.Fatal("abuse fixture session")
	}
	base := "Host: " + c.Listen + "\r\nConnection: close\r\n"
	auth := base + "Cookie: " + credential + "\r\n"
	cases := []struct {
		name, raw string
		status    int
	}{
		{"unauthenticated-json", "GET /api/overview HTTP/1.1\r\n" + base + "\r\n", 401},
		{"fragment-auth-bypass", "GET /storage/datasets HTTP/1.1\r\n" + base + "HX-Request: true\r\n\r\n", 401},
		{"host-rebinding", "GET /api/overview HTTP/1.1\r\nHost: hostile-abuse-marker.invalid\r\nCookie: " + credential + "\r\nConnection: close\r\n\r\n", 403},
		{"absolute-target", "GET http://hostile-abuse-marker.invalid/api/overview HTTP/1.1\r\n" + auth + "\r\n", 403},
		{"cross-origin", "GET /api/overview HTTP/1.1\r\n" + auth + "Origin: http://hostile-abuse-marker.invalid\r\n\r\n", 403},
		{"cross-site", "GET /api/overview HTTP/1.1\r\n" + auth + "Sec-Fetch-Site: cross-site\r\n\r\n", 403},
		{"duplicate-cookie", "GET /api/overview HTTP/1.1\r\n" + auth + "Cookie: " + admin.SessionCookie + "=hostile-abuse-marker\r\n\r\n", 401},
		{"get-mutation", "GET /api/refresh HTTP/1.1\r\n" + auth + "\r\n", 405},
		{"get-body", "GET /api/overview HTTP/1.1\r\n" + auth + "Content-Length: 1\r\n\r\nx", 400},
		{"unknown-command", "POST /api/command HTTP/1.1\r\n" + auth + "Content-Length: 0\r\n\r\n", 405},
		{"missing-csrf", "POST /api/refresh HTTP/1.1\r\n" + auth + "Content-Type: application/x-www-form-urlencoded\r\nContent-Length: 0\r\n\r\n", 403},
		{"wrong-content-type", "POST /api/refresh HTTP/1.1\r\n" + auth + "Content-Type: application/json\r\nContent-Length: 2\r\n\r\n{}", 415},
		{"conflicting-lengths", "POST /api/refresh HTTP/1.1\r\n" + auth + "Content-Length: 1\r\nContent-Length: 2\r\n\r\nxx", 400},
		{"malformed-chunk", "POST /api/refresh HTTP/1.1\r\n" + auth + "Content-Type: application/x-www-form-urlencoded\r\nTransfer-Encoding: chunked\r\n\r\ninvalid\r\n", 400},
		{"oversized-body", "POST /api/refresh HTTP/1.1\r\n" + auth + "Content-Type: application/x-www-form-urlencoded\r\nContent-Length: 4097\r\n\r\n" + strings.Repeat("x", 4097), 413},
		{"oversized-header", "GET /api/overview HTTP/1.1\r\n" + auth + "X-Fill: " + strings.Repeat("x", admin.MaxHeaderBytes+8192) + "\r\n\r\n", 431},
		{"duplicate-query", "GET /api/datasets?limit=1&limit=2 HTTP/1.1\r\n" + auth + "\r\n", 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := abuseRaw(t, c.Listen, tc.raw, 3*time.Second); got != tc.status {
				t.Fatalf("denial status %d, want %d", got, tc.status)
			}
		})
	}
	// Challenge throttling bounds source admission while the already admitted
	// session stays usable. Forwarded identities remain covered by gate tests.
	throttled := false
	for i := 0; i < 40; i++ {
		r, e := session.http.Get(session.base + "/auth/challenge")
		if e != nil {
			t.Fatal("challenge flood transport")
		}
		r.Body.Close()
		if r.StatusCode == 429 {
			throttled = true
		} else if r.StatusCode != 200 {
			t.Fatal("challenge flood unexpected result")
		}
	}
	if !throttled {
		t.Fatal("challenge flood was not bounded")
	}
	var wg sync.WaitGroup
	failures := make(chan bool, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := session.http.PostForm(session.base+"/api/refresh", url.Values{"csrf": {session.csrf}})
			if e != nil {
				failures <- true
				return
			}
			r.Body.Close()
			if r.StatusCode != 202 && r.StatusCode != 409 {
				failures <- true
			}
		}()
	}
	wg.Wait()
	close(failures)
	if len(failures) != 0 {
		t.Fatal("refresh flood admission/recovery failed")
	}
	// Stop reusing the controller's keep-alive sockets before measuring only
	// deliberately incomplete clients. fstat stays in guest memory; no dump is exported.
	session.http.CloseIdleConnections()
	fdCount := func() int {
		return len(strings.Split(strings.TrimSpace(string(readerFixtureCommand(t, "/usr/bin/fstat", "-p", fmt.Sprint(management.cmd.Process.Pid)))), "\n"))
	}
	baseline := fdCount()
	var slow []net.Conn
	for i := 0; i < 32; i++ {
		conn, e := net.DialTimeout("tcp", c.Listen, time.Second)
		if e != nil {
			t.Fatal("slow connection fixture")
		}
		slow = append(slow, conn)
		_, _ = io.WriteString(conn, "GET /api/overview HTTP/1.1\r\n")
	}
	t.Cleanup(func() {
		for _, conn := range slow {
			_ = conn.Close()
		}
	})
	time.Sleep(150 * time.Millisecond)
	if fdCount()-baseline > c.MaxReaders+8+2 {
		t.Fatal("slow clients exceeded built-runtime connection bound")
	}
	for _, conn := range slow {
		_ = conn.Close()
	}
	nativeWait(t, func() bool {
		r, e := session.http.Get(session.base + "/api/overview")
		if e != nil {
			return false
		}
		defer r.Body.Close()
		return r.StatusCode == 200
	})
	// One stalled header and body must independently expire, freeing capacity.
	for _, raw := range []string{"GET /api/overview HTTP/1.1\r\n", "POST /api/refresh HTTP/1.1\r\n" + auth + "Content-Type: application/x-www-form-urlencoded\r\nContent-Length: 4096\r\n\r\nx"} {
		conn, e := net.DialTimeout("tcp", c.Listen, time.Second)
		if e != nil {
			t.Fatal("deadline fixture")
		}
		_ = conn.SetDeadline(time.Now().Add(12 * time.Second))
		_, _ = io.WriteString(conn, raw)
		_, e = io.ReadAll(io.LimitReader(conn, 65536))
		_ = conn.Close()
		if e != nil {
			t.Fatal("slow input outlived server transport deadline")
		}
	}
	if !session.overview(t).Complete {
		t.Fatal("inventory did not recover after abuse")
	}
	management.stop(t, syscall.SIGTERM, true)
	helper.stop(t, syscall.SIGTERM, true)
	for _, sink := range []*nativeHelperEventSink{helperLogs, httpLogs} {
		if len(sink.records(t)) == 0 {
			t.Fatal("built runtime emitted no bounded events")
		}
		sink.Lock()
		log := append([]byte(nil), sink.Buffer.Bytes()...)
		sink.Unlock()
		for _, secret := range []string{token, credential, session.csrf, "hostile-abuse-marker", owned.Pool, owned.DatasetGUID} {
			if bytes.Contains(log, []byte(secret)) {
				t.Fatal("runtime log disclosed credentials/input/inventory")
			}
		}
	}
	afterConfig, _ := os.ReadFile(path)
	afterDigest, _ := os.ReadFile(digestPath)
	if !bytes.Equal(encoded, afterConfig) || !bytes.Equal(digest, afterDigest) {
		t.Fatal("abuse changed boot state")
	}
	t.Log("HTTP_ABUSE denial_matrix=pass floods=pass connection_bound=pass slow_deadlines=pass recovery=pass redaction=pass boot_state=unchanged")
}
