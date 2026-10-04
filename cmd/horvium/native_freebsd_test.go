//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/client"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
	"github.com/gonz0w/horvium/internal/service"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

type ownedFixture struct {
	Pool, PoolGUID, DatasetGUID string
	Size                        uint64
}

func nativeStorageFixture(t *testing.T) ownedFixture {
	t.Helper()
	must := func(path string, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, path, args...)
		cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
		b, e := cmd.Output()
		if e != nil {
			t.Fatal("owned fixture command failed; output withheld")
		}
		return string(b)
	}
	var nonce [8]byte
	if _, e := rand.Read(nonce[:]); e != nil {
		t.Fatal(e)
	}
	pool := "horvium_ui_" + hex.EncodeToString(nonce[:])
	file := filepath.Join(t.TempDir(), "disk.img")
	f, e := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	e = f.Truncate(512 << 20)
	ce := f.Close()
	if e != nil || ce != nil {
		t.Fatal("fixture image")
	}
	unit := strings.TrimSpace(must("/sbin/mdconfig", "-a", "-t", "vnode", "-f", file))
	if !regexp.MustCompile(`^md[0-9]+$`).MatchString(unit) {
		t.Fatal("generated md identity invalid; discard guest")
	}
	created := false
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		clean := func(path string, args ...string) error { return exec.CommandContext(ctx, path, args...).Run() }
		if created {
			if clean("/sbin/zpool", "destroy", pool) != nil {
				t.Error("owned pool cleanup failed; discard guest")
				return
			}
			if clean("/sbin/zpool", "list", "-H", pool) == nil {
				t.Error("owned pool remains")
			}
		}
		if clean("/sbin/mdconfig", "-d", "-u", strings.TrimPrefix(unit, "md")) != nil {
			t.Error("owned md cleanup failed; discard guest")
			return
		}
		if clean("/sbin/mdconfig", "-l", "-n", "-f", file) == nil {
			t.Error("owned md remains")
		}
	})
	if strings.TrimSpace(must("/sbin/mdconfig", "-l", "-n", "-f", file)) != strings.TrimPrefix(unit, "md") {
		t.Fatal("fixture backing-file mismatch")
	}
	// Only a newly allocated file-backed md device; no force or operator disk args.
	must("/sbin/zpool", "create", "-o", "cachefile=none", "-m", "none", pool, "/dev/"+unit)
	created = true
	must("/sbin/zfs", "create", "-o", "mountpoint=none", pool+"/data")
	before := must("/sbin/zfs", "get", "-H", "-p", "-o", "property,value", "mountpoint,readonly,compression", pool+"/data")
	t.Cleanup(func() {
		if after := must("/sbin/zfs", "get", "-H", "-p", "-o", "property,value", "mountpoint,readonly,compression", pool+"/data"); after != before {
			t.Error("foreground queries changed owned dataset properties")
		}
	})
	values := strings.Fields(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid,size", pool))
	if len(values) != 2 {
		t.Fatal("independent pool properties")
	}
	size, e := strconv.ParseUint(values[1], 10, 64)
	if e != nil {
		t.Fatal("independent pool size")
	}
	guid := strings.TrimSpace(must("/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", pool+"/data"))
	return ownedFixture{Pool: pool, PoolGUID: values[0], DatasetGUID: guid, Size: size}
}
func TestNativeForegroundCLIConsole(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("explicit disposable QEMU opt-in required")
	}
	if os.Getuid() != 0 {
		t.Fatal("fixture parent requires root")
	}
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
	expected := nativeStorageFixture(t)
	dir, e := os.MkdirTemp("/tmp", "horvium-web-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	if os.Chmod(dir, 0755) != nil {
		t.Fatal("fixture directory")
	}
	runDir := filepath.Join(dir, "run")
	if os.Mkdir(runDir, 0750) != nil || os.Chown(runDir, 0, int(gid)) != nil {
		t.Fatal("fixture socket directory")
	}
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	address := l.Addr().String()
	_ = l.Close()
	c := admin.Config{Version: 1, Listen: address, AllowedHosts: []string{address}, ManagementUID: uint32(uid), SocketGID: uint32(gid), HelperSocket: filepath.Join(runDir, "s"), StateDirectory: filepath.Join(dir, "state"), MaxBytes: 64 << 20, MaxRecords: 100000, MaxFieldBytes: 4096, MaxXMLDepth: 32, MaxXMLElements: 250000, MaxFrameBytes: 64 << 10, MaxJSONBytes: 64 << 20, MaxTransportRecords: 100000, MaxJSONDepth: 32, IntervalSeconds: 30, TimeoutSeconds: 45, StaleSeconds: 90, MaxReaders: 16, MaxHelperClients: 8}
	data, _ := json.Marshal(c)
	path := filepath.Join(dir, "config.json")
	if os.WriteFile(path, data, 0644) != nil {
		t.Fatal("fixture config")
	}
	token, e := admin.SetupToken(c, false)
	if e != nil {
		t.Fatal("fixture credential")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	helperCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- service.Helper(helperCtx, c) }()
	defer func() {
		stop()
		select {
		case e := <-done:
			if e != nil {
				t.Error("fixture helper shutdown")
			}
		case <-time.After(5 * time.Second):
			t.Error("fixture helper timed out")
		}
	}()
	for deadline := time.Now().Add(5 * time.Second); ; {
		info, e := os.Stat(c.HelperSocket)
		if e == nil && info.Mode().Perm() == 0660 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fixture helper readiness")
		}
		time.Sleep(10 * time.Millisecond)
	}
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	from, e := os.Open(exe)
	if e != nil {
		t.Fatal(e)
	}
	defer from.Close()
	binary := filepath.Join(dir, "test")
	to, e := os.OpenFile(binary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0755)
	if e != nil {
		t.Fatal(e)
	}
	_, e = io.Copy(to, from)
	closeErr := to.Close()
	if e != nil || closeErr != nil {
		t.Fatal("fixture executable")
	}
	cmd := exec.CommandContext(ctx, binary, "-test.run=^TestNativeForegroundChild$", "-test.count=1")
	expectedJSON, _ := json.Marshal(expected)
	cmd.Env = []string{"HORVIUM_WEB_CHILD=" + path, "HORVIUM_WEB_EXPECTED=" + string(expectedJSON)}
	cmd.Stdin = strings.NewReader(token + "\n")
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: []uint32{uint32(gid)}}}
	// Never export the child's inventory, request bodies or credentials.
	if cmd.Run() != nil {
		t.Fatal("unprivileged runtime/CLI/console fixture failed")
	}
}
func TestNativeForegroundChild(t *testing.T) {
	path := os.Getenv("HORVIUM_WEB_CHILD")
	if path == "" {
		t.Skip("reexec fixture only")
	}
	var expected ownedFixture
	if json.Unmarshal([]byte(os.Getenv("HORVIUM_WEB_EXPECTED")), &expected) != nil || expected.Pool == "" {
		t.Fatal("owned fixture expectation")
	}
	c, e := admin.LoadConfig(path)
	if e != nil || os.Getuid() == 0 || uint32(os.Getuid()) != c.ManagementUID {
		t.Fatal("fixture identity/config")
	}
	token, e := readToken(os.Stdin)
	if e != nil {
		t.Fatal("protected fixture input")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	serverCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- service.Serve(serverCtx, c) }()
	defer func() {
		stop()
		select {
		case e := <-done:
			if e != nil {
				t.Error("management shutdown")
			}
		case <-time.After(7 * time.Second):
			t.Error("management leaked")
		}
	}()
	endpoint := "http://" + c.Listen
	probe := &http.Client{Timeout: time.Second}
	for deadline := time.Now().Add(5 * time.Second); ; {
		r, e := probe.Get(endpoint + "/assets/console.css")
		if e == nil {
			r.Body.Close()
			if r.StatusCode == 200 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("management readiness")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cli, e := client.New(endpoint)
	if e != nil {
		t.Fatal(e)
	}
	if cli.Login(ctx, token) != nil {
		t.Fatal("CLI login")
	}
	defer cli.Logout(context.Background())
	var o observation.Overview
	for {
		value, e := cli.Query(ctx, "overview", observation.Query{})
		if e == nil {
			o = value.(observation.Overview)
			break
		}
		if ctx.Err() != nil {
			t.Fatal("native observation unavailable")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if o.Generation == "" || o.Host.Architecture != "amd64" || o.HostState.Status != inventory.OK || o.Host.InstalledKernel == "" {
		t.Fatal("real host observation")
	}
	value, e := cli.Query(ctx, "pools", observation.Query{Generation: o.Generation, Filter: expected.Pool})
	if e != nil {
		t.Fatal("native pool query")
	}
	p := value.(observation.Page[inventory.Pool])
	if p.Generation != o.Generation || len(p.Rows) != 1 || p.Rows[0].Name != expected.Pool || p.Rows[0].GUID != expected.PoolGUID || p.Rows[0].Size != expected.Size {
		t.Fatal("generation mismatch")
	}
	var output, diagnostics bytes.Buffer
	if run(ctx, []string{"inventory", "-endpoint", endpoint, "-generation", o.Generation}, strings.NewReader(token), &output, &diagnostics) != 0 {
		t.Fatal("operator CLI")
	}
	var actual observation.Overview
	if json.Unmarshal(output.Bytes(), &actual) != nil || actual.Generation != o.Generation || actual.Host.Architecture != o.Host.Architecture {
		t.Fatal("CLI/API parity")
	}
	output.Reset()
	diagnostics.Reset()
	if run(ctx, []string{"inventory", "-endpoint", endpoint, "-resource", "datasets", "-generation", o.Generation, "-filter", expected.Pool + "/data"}, strings.NewReader(token), &output, &diagnostics) != 0 {
		t.Fatal("native dataset CLI")
	}
	var datasetPage observation.Page[inventory.Dataset]
	if json.Unmarshal(output.Bytes(), &datasetPage) != nil || datasetPage.Generation != o.Generation || len(datasetPage.Rows) != 1 || datasetPage.Rows[0].GUID != expected.DatasetGUID {
		t.Fatal("CLI/independent dataset GUID parity")
	}
	// Exercise real loopback HTTP HTML with browser-style challenge/cookie login.
	response, e := probe.Get(endpoint + "/login")
	if e != nil {
		t.Fatal("login form")
	}
	b, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if !bytes.Contains(b, []byte(`type="password"`)) {
		t.Fatal("semantic login")
	}
	anonymous, e := probe.Get(endpoint + "/api/overview")
	if e != nil {
		t.Fatal("anonymous probe")
	}
	anonymous.Body.Close()
	if anonymous.StatusCode != 401 {
		t.Fatal("anonymous inventory")
	}
	jar, _ := cookiejar.New(nil)
	probe.Jar = jar
	challengeResponse, e := probe.Get(endpoint + "/auth/challenge")
	if e != nil {
		t.Fatal("browser challenge")
	}
	var challenge admin.Challenge
	e = json.NewDecoder(challengeResponse.Body).Decode(&challenge)
	challengeResponse.Body.Close()
	if e != nil {
		t.Fatal("browser challenge response")
	}
	logged, e := probe.PostForm(endpoint+"/auth/login", url.Values{"token": {token}, "csrf": {challenge.CSRF}})
	if e != nil {
		t.Fatal("browser authentication")
	}
	logged.Body.Close()
	if logged.StatusCode != 200 {
		t.Fatal("browser authentication response")
	}
	for _, fragment := range []bool{false, true} {
		request, _ := http.NewRequest("GET", endpoint+"/storage/pools?generation="+url.QueryEscape(o.Generation)+"&filter="+url.QueryEscape(expected.Pool), nil)
		if fragment {
			request.Header.Set("HX-Request", "true")
		}
		rendered, e := probe.Do(request)
		if e != nil {
			t.Fatal("native HTML request")
		}
		markup, e := io.ReadAll(io.LimitReader(rendered.Body, 4<<20))
		rendered.Body.Close()
		if e != nil || rendered.StatusCode != 200 || !bytes.Contains(markup, []byte(o.Generation)) || !bytes.Contains(markup, []byte(`id="workspace"`)) {
			t.Fatal("native full/fragment generation parity")
		}
		if fragment && bytes.Contains(markup, []byte("<!doctype")) {
			t.Fatal("fragment returned full document")
		}
		for _, pool := range p.Rows {
			if !bytes.Contains(markup, []byte(pool.GUID)) {
				t.Fatal("HTML/native GUID parity")
			}
		}
	}
	for _, fragment := range []bool{false, true} {
		request, _ := http.NewRequest("GET", endpoint+"/storage/datasets?generation="+url.QueryEscape(o.Generation)+"&filter="+url.QueryEscape(expected.Pool+"/data")+"&selected="+expected.DatasetGUID, nil)
		if fragment {
			request.Header.Set("HX-Request", "true")
		}
		rendered, e := probe.Do(request)
		if e != nil {
			t.Fatal("native dataset HTML")
		}
		markup, e := io.ReadAll(io.LimitReader(rendered.Body, 4<<20))
		rendered.Body.Close()
		if e != nil || rendered.StatusCode != 200 || !bytes.Contains(markup, []byte(expected.DatasetGUID)) || !bytes.Contains(markup, []byte(`id="detail-title"`)) {
			t.Fatal("native selected dataset GUID parity")
		}
	}
	if _, e = cli.Query(ctx, "pools", observation.Query{Generation: "expired"}); e != observation.ErrExpired {
		t.Fatal("expired native cursor")
	}
	fmt.Println("PASS: native unprivileged foreground runtime and authenticated CLI")
}
