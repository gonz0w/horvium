//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"encoding/json"
	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/observation"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func nativeRuntimeConfig(t *testing.T) (admin.Config, string, string) {
	t.Helper()
	u, e := user.Lookup("nobody")
	if e != nil {
		t.Fatal("fixture UID unavailable")
	}
	uid, e := strconv.ParseUint(u.Uid, 10, 32)
	if e != nil || uid == 0 {
		t.Fatal("fixture UID")
	}
	gid, e := strconv.ParseUint(u.Gid, 10, 32)
	if e != nil || gid == 0 {
		t.Fatal("fixture group")
	}
	dir, e := os.MkdirTemp("/tmp", "horvium-lifecycle-")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if os.Chmod(dir, 0755) != nil {
		t.Fatal("fixture directory")
	}
	run := filepath.Join(dir, "run")
	if os.Mkdir(run, 0750) != nil || os.Chown(run, 0, int(gid)) != nil {
		t.Fatal("socket directory")
	}
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	address := l.Addr().String()
	_ = l.Close()
	c := admin.Config{Version: 1, Listen: address, AllowedHosts: []string{address}, ManagementUID: uint32(uid), SocketGID: uint32(gid), HelperSocket: filepath.Join(run, "s"), StateDirectory: filepath.Join(dir, "state"), MaxBytes: 64 << 20, MaxRecords: 100000, MaxFieldBytes: 4096, MaxXMLDepth: 32, MaxXMLElements: 250000, MaxFrameBytes: 64 << 10, MaxJSONBytes: 64 << 20, MaxTransportRecords: 100000, MaxJSONDepth: 32, IntervalSeconds: 60, TimeoutSeconds: 5, StaleSeconds: 90, MaxReaders: 16, MaxHelperClients: 8}
	b, _ := json.Marshal(c)
	path := filepath.Join(dir, "config.json")
	if os.WriteFile(path, b, 0644) != nil {
		t.Fatal("config fixture")
	}
	token, e := admin.SetupToken(c, false)
	if e != nil {
		t.Fatal("token fixture")
	}
	return c, path, token
}

type nativeProcess struct {
	cmd    *exec.Cmd
	done   chan error
	waited bool
}

func startNativeProcess(t *testing.T, binary string, args []string, uid, gid uint32) *nativeProcess {
	t.Helper()
	return startNativeProcessWithStderr(t, binary, args, uid, gid, nil)
}
func startNativeProcessWithStderr(t *testing.T, binary string, args []string, uid, gid uint32, stderr io.Writer) *nativeProcess {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Stderr = stderr
	cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	if uid != 0 {
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: gid, Groups: []uint32{gid}}}
	}
	p := &nativeProcess{cmd: cmd, done: make(chan error, 1)}
	// No credential or native inventory leaves the disposable guest.
	if cmd.Start() != nil {
		t.Fatal("fixture process start")
	}
	go func() { p.done <- cmd.Wait() }()
	t.Cleanup(func() {
		if !p.waited {
			_ = cmd.Process.Kill()
			select {
			case <-p.done:
			case <-time.After(5 * time.Second):
				t.Error("fixture process cleanup timed out")
			}
			p.waited = true
		}
	})
	return p
}
func (p *nativeProcess) stop(t *testing.T, signal os.Signal, clean bool) {
	t.Helper()
	if p.cmd.Process.Signal(signal) != nil {
		t.Fatal("fixture signal")
	}
	select {
	case e := <-p.done:
		p.waited = true
		if clean && e != nil {
			t.Fatal("foreground signal shutdown failed")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("foreground shutdown timed out")
	}
}
func nativeWait(t *testing.T, f func() bool) {
	t.Helper()
	until := time.Now().Add(20 * time.Second)
	for !f() {
		if time.Now().After(until) {
			t.Fatal("native runtime condition timed out")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
func TestNativeBuiltProcessRestart(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("disposable QEMU opt-in required")
	}
	if os.Getuid() != 0 {
		t.Fatal("fixture parent requires root")
	}
	c, path, token := nativeRuntimeConfig(t)
	fileRead := independentSFTP(t)
	fileRead()
	built, e := filepath.Abs("../../.tools/horvium")
	if e != nil {
		t.Fatal(e)
	}
	if info, e := os.Stat(built); e != nil || !info.Mode().IsRegular() {
		t.Fatal("native script must build product executable")
	}
	// Copy the built product out of potentially root-only checkout ancestors.
	binary := filepath.Join(filepath.Dir(path), "horvium")
	from, e := os.Open(built)
	if e != nil {
		t.Fatal("built executable")
	}
	to, e := os.OpenFile(binary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if e != nil {
		from.Close()
		t.Fatal("fixture executable")
	}
	_, copyErr := io.Copy(to, from)
	closeErr := to.Close()
	from.Close()
	if copyErr != nil || closeErr != nil {
		t.Fatal("executable copy")
	}
	// State/config live on guest boot storage; this fixture creates no data pool.
	beforeConfig, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	digestPath := filepath.Join(c.StateDirectory, "token.sha256")
	beforeDigest, e := os.ReadFile(digestPath)
	if e != nil {
		t.Fatal("digest fixture")
	}
	helper := startNativeProcess(t, binary, []string{"helper", "-config", path}, 0, 0)
	nativeWait(t, func() bool {
		info, e := os.Stat(c.HelperSocket)
		return e == nil && info.Mode()&os.ModeSocket != 0 && info.Mode().Perm() == 0660
	})
	socket, e := os.Lstat(c.HelperSocket)
	if e != nil {
		t.Fatal(e)
	}
	management := startNativeProcess(t, binary, []string{"serve", "-config", path}, c.ManagementUID, c.SocketGID)
	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{Jar: jar, Timeout: 2 * time.Second}
	base := "http://" + c.Listen
	nativeWait(t, func() bool {
		r, e := httpClient.Get(base + "/auth/challenge")
		if e != nil {
			return false
		}
		_ = r.Body.Close()
		return r.StatusCode == 200
	})
	login := func() {
		t.Helper()
		r, e := httpClient.Get(base + "/auth/challenge")
		if e != nil {
			t.Fatal("challenge")
		}
		var challenge admin.Challenge
		e = json.NewDecoder(r.Body).Decode(&challenge)
		_ = r.Body.Close()
		if e != nil {
			t.Fatal("challenge schema")
		}
		r, e = httpClient.PostForm(base+"/auth/login", url.Values{"csrf": {challenge.CSRF}, "token": {token}})
		if e != nil {
			t.Fatal("login")
		}
		_ = r.Body.Close()
		if r.StatusCode != 200 {
			t.Fatal("login denied")
		}
	}
	overview := func() (observation.Overview, int) {
		r, e := httpClient.Get(base + "/api/overview")
		if e != nil {
			return observation.Overview{}, 0
		}
		defer r.Body.Close()
		var o observation.Overview
		if r.StatusCode == 200 && json.NewDecoder(r.Body).Decode(&o) != nil {
			t.Fatal("overview schema")
		}
		return o, r.StatusCode
	}
	login()
	var first observation.Overview
	nativeWait(t, func() bool {
		o, status := overview()
		if status == 200 {
			first = o
			return o.Generation != ""
		}
		return false
	})
	// Freeze helper before requesting refresh so management is killed during an
	// actual in-flight transport, rather than hoping to race fast native probes.
	if helper.cmd.Process.Signal(syscall.SIGSTOP) != nil {
		t.Fatal("freeze helper")
	}
	var session admin.Session
	// Authenticate another session solely to obtain its CSRF for refresh.
	r, e := httpClient.Get(base + "/auth/challenge")
	if e != nil {
		t.Fatal("challenge")
	}
	var ch admin.Challenge
	_ = json.NewDecoder(r.Body).Decode(&ch)
	_ = r.Body.Close()
	r, e = httpClient.PostForm(base+"/auth/login", url.Values{"csrf": {ch.CSRF}, "token": {token}})
	if e != nil {
		t.Fatal("login")
	}
	_ = json.NewDecoder(r.Body).Decode(&session)
	_ = r.Body.Close()
	r, e = httpClient.PostForm(base+"/api/refresh", url.Values{"csrf": {session.CSRF}})
	if e != nil {
		t.Fatal("refresh")
	}
	_ = r.Body.Close()
	if r.StatusCode != 202 {
		t.Fatal("refresh not scheduled")
	}
	nativeWait(t, func() bool {
		r, e := httpClient.Get(base + "/status")
		if e != nil {
			return false
		}
		defer r.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(r.Body, 65536))
		return strings.Contains(string(b), "Collecting")
	})
	management.stop(t, syscall.SIGKILL, false)
	fileRead()
	helper.stop(t, syscall.SIGKILL, false)
	fileRead()
	// Abrupt helper death deliberately leaves a socket. Only this fixture's
	// captured root-owned inode may be removed; product never adopts old sockets.
	current, e := os.Lstat(c.HelperSocket)
	if e != nil || !os.SameFile(socket, current) || current.Mode()&os.ModeSocket == 0 {
		t.Fatal("owned stale socket changed")
	}
	if os.Remove(c.HelperSocket) != nil {
		t.Fatal("fixture stale socket cleanup")
	}
	helper = startNativeProcess(t, binary, []string{"helper", "-config", path}, 0, 0)
	nativeWait(t, func() bool { info, e := os.Stat(c.HelperSocket); return e == nil && info.Mode().Perm() == 0660 })
	socket, e = os.Lstat(c.HelperSocket)
	if e != nil {
		t.Fatal("replacement socket")
	}
	management = startNativeProcess(t, binary, []string{"serve", "-config", path}, c.ManagementUID, c.SocketGID)
	nativeWait(t, func() bool { _, status := overview(); return status == 401 }) // old session invalid
	login()
	var next observation.Overview
	nativeWait(t, func() bool {
		o, status := overview()
		if status == 200 {
			next = o
			return o.Generation != ""
		}
		return false
	})
	if next.Generation == first.Generation {
		t.Fatal("generation survived restart")
	}
	r, e = httpClient.Get(base + "/api/datasets?generation=" + url.QueryEscape(first.Generation))
	if e != nil {
		t.Fatal("expired cursor")
	}
	_ = r.Body.Close()
	if r.StatusCode != 409 {
		t.Fatal("old generation accepted")
	}
	// Kill helper while management remains alive: cached context stays stale and
	// HTTP remains reachable, then a replacement helper recovers a new generation.
	helper.stop(t, syscall.SIGKILL, false)
	fileRead()
	// Wait for the next scheduled collection is avoided by reauth + explicit CSRF.
	r, e = httpClient.Get(base + "/auth/challenge")
	if e != nil {
		t.Fatal("challenge")
	}
	_ = json.NewDecoder(r.Body).Decode(&ch)
	_ = r.Body.Close()
	r, e = httpClient.PostForm(base+"/auth/login", url.Values{"csrf": {ch.CSRF}, "token": {token}})
	if e != nil {
		t.Fatal("login")
	}
	_ = json.NewDecoder(r.Body).Decode(&session)
	_ = r.Body.Close()
	r, e = httpClient.PostForm(base+"/api/refresh", url.Values{"csrf": {session.CSRF}})
	if e != nil {
		t.Fatal("refresh")
	}
	_ = r.Body.Close()
	if r.StatusCode != 202 {
		t.Fatal("outage refresh")
	}
	nativeWait(t, func() bool {
		o, status := overview()
		return status == 200 && o.Generation == next.Generation && o.HostState.Stale
	})
	current, e = os.Lstat(c.HelperSocket)
	if e != nil || !os.SameFile(socket, current) {
		t.Fatal("replacement stale socket changed")
	}
	if os.Remove(c.HelperSocket) != nil {
		t.Fatal("owned stale socket cleanup")
	}
	helper = startNativeProcess(t, binary, []string{"helper", "-config", path}, 0, 0)
	nativeWait(t, func() bool { info, e := os.Stat(c.HelperSocket); return e == nil && info.Mode().Perm() == 0660 })
	r, e = httpClient.PostForm(base+"/api/refresh", url.Values{"csrf": {session.CSRF}})
	if e != nil {
		t.Fatal("recovery refresh")
	}
	r.Body.Close()
	if r.StatusCode != 202 {
		t.Fatal("recovery refresh denied")
	}
	nativeWait(t, func() bool {
		o, status := overview()
		return status == 200 && o.Generation != next.Generation && !o.HostState.Stale
	})
	fileRead()
	management.stop(t, syscall.SIGTERM, true)
	fileRead()
	helper.stop(t, syscall.SIGTERM, true)
	if _, e := os.Lstat(c.HelperSocket); !os.IsNotExist(e) {
		t.Fatal("graceful helper left socket")
	}
	if b, e := os.ReadFile(path); e != nil || !bytes.Equal(b, beforeConfig) {
		t.Fatal("config changed across restart")
	}
	if b, e := os.ReadFile(digestPath); e != nil || !bytes.Equal(b, beforeDigest) {
		t.Fatal("credential digest changed across restart")
	}
}
