// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func nativeFixture(t *testing.T) (Config, string) {
	t.Helper()
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Geteuid() != 0 {
		t.Skip("requires root in disposable FreeBSD QEMU")
	}
	dir, e := os.MkdirTemp("/tmp", "horvium-admin-")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := os.RemoveAll(dir); e != nil {
			t.Error("fixture cleanup failed")
		}
	})
	if os.Chmod(dir, 0755) != nil {
		t.Fatal("fixture mode")
	}
	c := testConfig()
	c.StateDirectory = filepath.Join(dir, "state")
	p := filepath.Join(dir, "config.json")
	b, _ := json.Marshal(c)
	if os.WriteFile(p, b, 0644) != nil {
		t.Fatal("config fixture")
	}
	return c, p
}
func TestNativeCredentialFiles(t *testing.T) {
	c, p := nativeFixture(t)
	loaded, e := LoadConfig(p)
	if e != nil || loaded.StateDirectory != c.StateDirectory {
		t.Fatal("root config load")
	}
	token, e := SetupToken(c, false)
	if e != nil || !validSecret(token) {
		t.Fatal("local setup")
	}
	path := filepath.Join(c.StateDirectory, "token.sha256")
	b, e := os.ReadFile(path)
	digest := sha256.Sum256([]byte(token))
	if e != nil || string(b) != string(digest[:]) || len(b) != 32 {
		t.Fatal("digest-only persistence")
	}
	st, e := os.Stat(path)
	if e != nil || st.Mode().Perm() != 0600 || st.Sys().(*syscall.Stat_t).Uid != c.ManagementUID || st.Sys().(*syscall.Stat_t).Nlink != 1 {
		t.Fatal("digest metadata")
	}
	st, _ = os.Stat(c.StateDirectory)
	if st.Mode().Perm() != 0700 || st.Sys().(*syscall.Stat_t).Uid != c.ManagementUID {
		t.Fatal("state directory metadata")
	}
	if got, e := SetupToken(c, false); !errors.Is(e, ErrState) || got != "" {
		t.Fatal("setup clobbered existing credential")
	}
	rotated, e := SetupToken(c, true)
	if e != nil || rotated == token {
		t.Fatal("rotation")
	}
	b, _ = os.ReadFile(path)
	d2 := sha256.Sum256([]byte(rotated))
	if string(b) != string(d2[:]) {
		t.Fatal("rotation did not publish complete digest")
	}
	if _, e := LoadDigest(c); !errors.Is(e, ErrState) {
		t.Fatal("root runtime bypass of dedicated management UID")
	}
	for _, mode := range []os.FileMode{0664, 0646, 0777} {
		if os.Chmod(p, mode) != nil {
			t.Fatal("chmod")
		}
		if _, e := LoadConfig(p); !errors.Is(e, ErrConfig) {
			t.Fatal("unsafe config mode")
		}
	}
	_ = os.Chmod(p, 0644)
	if os.Chown(p, 65534, 65534) != nil {
		t.Fatal("chown")
	}
	if _, e := LoadConfig(p); !errors.Is(e, ErrConfig) {
		t.Fatal("unprivileged config accepted")
	}
	_ = os.Chown(p, 0, 0)
	link := p + ".link"
	if os.Symlink(p, link) != nil {
		t.Fatal("symlink")
	}
	if _, e := LoadConfig(link); !errors.Is(e, ErrConfig) {
		t.Fatal("symlink config")
	}
	_ = os.Remove(link)
	if os.Link(p, link) != nil {
		t.Fatal("hardlink")
	}
	if _, e := LoadConfig(p); !errors.Is(e, ErrConfig) {
		t.Fatal("hardlinked config")
	}
	_ = os.Remove(link)
	before := append([]byte(nil), b...)
	_ = os.Chmod(path, 0644)
	if _, e := SetupToken(c, true); !errors.Is(e, ErrState) {
		t.Fatal("unsafe digest adopted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed rotation mutated digest")
	}
	_ = os.Chmod(path, 0600)
	link = path + ".link"
	_ = os.Link(path, link)
	if _, e := SetupToken(c, true); !errors.Is(e, ErrState) {
		t.Fatal("hardlinked digest adopted")
	}
	_ = os.Remove(link)
	_ = os.Remove(path)
	_ = os.Symlink(p, path)
	if _, e := SetupToken(c, true); !errors.Is(e, ErrState) {
		t.Fatal("symlink digest adopted")
	}
	_ = os.Remove(path)
	_ = os.Chmod(c.StateDirectory, 0750)
	if _, e := SetupToken(c, false); !errors.Is(e, ErrState) {
		t.Fatal("unsafe state directory")
	}
	_ = os.Chmod(c.StateDirectory, 0700)
	_ = os.Chmod(filepath.Dir(p), 0777)
	if _, e := LoadConfig(p); !errors.Is(e, ErrConfig) {
		t.Fatal("writable config ancestor")
	}
	if _, e := SetupToken(c, false); !errors.Is(e, ErrState) {
		t.Fatal("writable state ancestor")
	}
	_ = os.Chmod(filepath.Dir(p), 0755)
}
func TestNativeAdministrationAsServiceUID(t *testing.T) {
	c, p := nativeFixture(t)
	token, e := SetupToken(c, false)
	if e != nil {
		t.Fatal("setup")
	}
	binary := filepath.Join(filepath.Dir(p), "test-bin")
	source, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	in, e := os.Open(source)
	if e != nil {
		t.Fatal(e)
	}
	defer in.Close()
	out, e := os.OpenFile(binary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = io.Copy(out, in); e != nil {
		t.Fatal(e)
	}
	if out.Close() != nil {
		t.Fatal("copy close")
	}
	for _, uid := range []uint32{65534, 65533} {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		cmd := exec.CommandContext(ctx, binary, "-test.run=^TestNativeAdministrationChild$", "-test.count=1")
		cmd.Env = append(os.Environ(), "HORVIUM_ADMIN_CONFIG="+p)
		cmd.Stdin = strings.NewReader(token + "\n")
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: 65534, Groups: []uint32{65534}}}
		log, e := cmd.CombinedOutput()
		cancel()
		if e != nil {
			t.Log(string(log))
			t.Fatal("dropped-UID fixture failed")
		}
	}
}
func TestNativeAdministrationChild(t *testing.T) {
	p := os.Getenv("HORVIUM_ADMIN_CONFIG")
	if p == "" || os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("reexec fixture only")
	}
	c, e := LoadConfig(p)
	if e != nil {
		t.Fatal("service could not read safe root configuration")
	}
	digest, e := LoadDigest(c)
	if os.Geteuid() != int(c.ManagementUID) {
		if !errors.Is(e, ErrState) {
			t.Fatal("other UID read digest")
		}
		if _, e := os.ReadFile(filepath.Join(c.StateDirectory, "token.sha256")); e == nil {
			t.Fatal("kernel allowed other UID digest read")
		}
		return
	}
	if e != nil {
		t.Fatal("service digest read")
	}
	path := filepath.Join(c.StateDirectory, digestName)
	if os.Chmod(path, 0644) != nil {
		t.Fatal("service permission fixture")
	}
	if _, e := LoadDigest(c); !errors.Is(e, ErrState) {
		t.Fatal("unsafe digest mode accepted")
	}
	if os.Chmod(path, 0600) != nil {
		t.Fatal("fixture restore")
	}
	token, e := bufio.NewReader(os.Stdin).ReadString('\n')
	if e != nil {
		t.Fatal("fixture input")
	}
	token = strings.TrimSuffix(token, "\n")
	if d := sha256.Sum256([]byte(token)); d != digest {
		t.Fatal("stored digest mismatch")
	}
	if _, e := SetupToken(c, true); !errors.Is(e, ErrState) {
		t.Fatal("service allowed local administrator rotation")
	}
	// Ephemeral loopback listener is a disposable HTTP fixture, not host service
	// enablement. Its exact address becomes the fixture's startup Host contract.
	s := httptest.NewUnstartedServer(nil)
	defer s.Close()
	c.Listen = s.Listener.Addr().String()
	c.AllowedHosts = []string{c.Listen}
	a := NewAuth(digest)
	defer a.Close()
	g, e := NewGate(c, a)
	if e != nil {
		t.Fatal(e)
	}
	var reads atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/challenge", g.Challenge)
	mux.HandleFunc("/login", g.Login)
	mux.HandleFunc("/logout", g.Logout)
	mux.Handle("/", g.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reads.Add(1); w.WriteHeader(204) })))
	s.Config, e = c.HTTPServer(mux)
	if e != nil {
		t.Fatal(e)
	}
	s.Start()
	client := s.Client()
	response, e := client.Get(s.URL + "/fragment")
	if e != nil {
		t.Fatal("loopback request")
	}
	_ = response.Body.Close()
	if response.StatusCode != 401 || reads.Load() != 0 {
		t.Fatal("inventory leaked before auth")
	}
	response, e = client.Get(s.URL + "/challenge")
	if e != nil {
		t.Fatal(e)
	}
	var challenge Challenge
	e = json.NewDecoder(response.Body).Decode(&challenge)
	_ = response.Body.Close()
	if e != nil || response.StatusCode != 200 || len(response.Cookies()) != 1 {
		t.Fatal("challenge exchange")
	}
	challengeCookie := response.Cookies()[0]
	r, _ := http.NewRequest("POST", s.URL+"/login", strings.NewReader(url.Values{"csrf": {challenge.CSRF}, "token": {token}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(challengeCookie)
	response, e = client.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	var session Session
	e = json.NewDecoder(response.Body).Decode(&session)
	_ = response.Body.Close()
	if e != nil || response.StatusCode != 200 {
		t.Fatal("native login")
	}
	var sessionCookie *http.Cookie
	for _, cookie := range response.Cookies() {
		if cookie.Name == SessionCookie {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil {
		t.Fatal("session cookie")
	}
	r, _ = http.NewRequest("GET", s.URL+"/fragment", nil)
	r.AddCookie(sessionCookie)
	r.Header.Set("HX-Request", "true")
	response, e = client.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	_ = response.Body.Close()
	if response.StatusCode != 204 || reads.Load() != 1 {
		t.Fatal("native authorized fragment")
	}
	r, _ = http.NewRequest("POST", s.URL+"/refresh", strings.NewReader("csrf=incorrect"))
	r.AddCookie(sessionCookie)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, e = client.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	_ = response.Body.Close()
	if response.StatusCode != 403 || reads.Load() != 1 {
		t.Fatal("native csrf bypass")
	}
	for i := 0; i < 2; i++ {
		r, _ = http.NewRequest("GET", s.URL+"/", nil)
		r.AddCookie(sessionCookie)
		if i == 0 {
			r.Host = "evil.example:8080"
			r.Header.Set("X-Forwarded-Host", c.Listen)
		} else {
			r.Header.Set("Origin", "http://evil.example")
			r.Header.Set("Sec-Fetch-Site", "same-origin")
		}
		response, e = client.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		_ = response.Body.Close()
		if response.StatusCode != 403 || reads.Load() != 1 {
			t.Fatal("native host/origin bypass")
		}
	}
	// Exercise the actual net/http header cap before application dispatch.
	conn, e := net.DialTimeout("tcp", c.Listen, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, e = io.WriteString(conn, "GET / HTTP/1.1\r\nHost: "+c.Listen+"\r\nX-Fill: "+strings.Repeat("x", MaxHeaderBytes+8192)+"\r\n\r\n")
	if e != nil {
		t.Fatal("header fixture write")
	}
	line, e := bufio.NewReader(conn).ReadString('\n')
	_ = conn.Close()
	if e != nil || !strings.Contains(line, "431") {
		t.Fatal("transport header cap")
	}
	if reads.Load() != 1 {
		t.Fatal("oversize headers reached inventory")
	}
	a.Close()
	r, _ = http.NewRequest("GET", s.URL+"/", nil)
	r.AddCookie(sessionCookie)
	response, e = client.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	_ = response.Body.Close()
	if response.StatusCode != 503 || reads.Load() != 1 {
		t.Fatal("closed auth bypass")
	}
}

func TestNativeSetupIsAtomicAndDoesNotFollowSpecialFiles(t *testing.T) {
	c, p := nativeFixture(t)
	if _, e := SetupToken(c, true); !errors.Is(e, ErrState) {
		t.Fatal("rotation initialized missing state")
	}
	// Establish an empty, correctly owned directory before concurrent setup.
	if os.Mkdir(c.StateDirectory, 0700) != nil || os.Chown(c.StateDirectory, int(c.ManagementUID), int(c.SocketGID)) != nil {
		t.Fatal("empty state fixture")
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var winner string
	successes := 0
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			token, e := SetupToken(c, false)
			mu.Lock()
			defer mu.Unlock()
			if e == nil {
				successes++
				winner = token
			} else if !errors.Is(e, ErrState) || token != "" {
				t.Error("unsafe setup failure")
			}
		})
	}
	wg.Wait()
	if successes != 1 {
		t.Fatal("initial setup was not no-clobber")
	}
	path := filepath.Join(c.StateDirectory, digestName)
	b, e := os.ReadFile(path)
	digest := sha256.Sum256([]byte(winner))
	if e != nil || string(b) != string(digest[:]) {
		t.Fatal("setup published partial or unrelated state")
	}
	entries, e := os.ReadDir(c.StateDirectory)
	if e != nil || len(entries) != 1 || entries[0].Name() != digestName {
		t.Fatal("temporary digest leaked")
	}
	// Readers opening the old/new inode must never observe a partial digest.
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
			}
			b, e := os.ReadFile(path)
			if e != nil || len(b) != 32 {
				t.Error("rotation exposed incomplete file")
				return
			}
		}
	}()
	for i := 0; i < 8; i++ {
		if _, e := SetupToken(c, true); e != nil {
			t.Error("atomic rotation failed")
		}
	}
	close(stop)
	<-done
	_ = os.Remove(path)
	if syscall.Mkfifo(path, 0600) != nil {
		t.Fatal("fifo fixture")
	}
	if _, e := SetupToken(c, true); !errors.Is(e, ErrState) {
		t.Fatal("special digest file adopted")
	}
	_ = os.Remove(path)
	link := filepath.Join(filepath.Dir(p), "linked-state")
	if os.Symlink(c.StateDirectory, link) != nil {
		t.Fatal("state symlink")
	}
	c.StateDirectory = link
	if _, e := SetupToken(c, false); !errors.Is(e, ErrState) {
		t.Fatal("state symlink followed")
	}
	alias := filepath.Join(filepath.Dir(p), "parent-alias")
	if os.Symlink(filepath.Dir(p), alias) != nil {
		t.Fatal("ancestor symlink")
	}
	if _, e := LoadConfig(filepath.Join(alias, "config.json")); !errors.Is(e, ErrConfig) {
		t.Fatal("symlink ancestor followed")
	}
}
