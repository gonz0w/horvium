// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type listenerFixture struct {
	server *propertyServer
	cancel context.CancelFunc
	done   <-chan error
	once   sync.Once
}

func startPropertyListener(t *testing.T, f nativeFixture) *listenerFixture {
	t.Helper()
	s, err := newPropertyServer(f.config)
	if err != nil {
		t.Fatal("root listener factory", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	r := &listenerFixture{server: s, cancel: cancel, done: done}
	t.Cleanup(func() { r.stop(t) })
	go func() { done <- s.serve(ctx) }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if safeSocket(f.config) == nil {
			return r
		}
		select {
		case err := <-done:
			t.Fatal("listener returned before ready", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("root listener not ready")
		}
		time.Sleep(time.Millisecond)
	}
}
func (r *listenerFixture) stop(t *testing.T) {
	t.Helper()
	r.once.Do(func() {
		r.cancel()
		select {
		case err := <-r.done:
			if !errors.Is(err, context.Canceled) || err.Error() != context.Canceled.Error() {
				t.Error("root listener stop", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("root listener workers did not drain")
		}
		r.server.mu.Lock()
		n := len(r.server.active)
		r.server.mu.Unlock()
		if n != 0 {
			t.Error("listener retained worker reservations")
		}
	})
}
func listenerWorkers(t *testing.T, s *propertyServer, want int) *propertyConnection {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		s.mu.Lock()
		n := len(s.active)
		var lease *propertyConnection
		for c := range s.active {
			lease = c
		}
		s.mu.Unlock()
		if n == want {
			return lease
		}
		if time.Now().After(deadline) {
			t.Fatal("unexpected actual worker count", n, want)
		}
		time.Sleep(time.Millisecond)
	}
}
func listenerChild(t *testing.T, f nativeFixture, mode string, uid uint32) *exec.Cmd {
	t.Helper()
	data, err := json.Marshal(f.config)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(f.binary, "-test.run=^TestNativePropertyListenerChild$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_PROPERTY_LISTENER_CHILD=" + mode, "HORVIUM_HELPER_CONFIG=" + string(data)}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: f.gid, Groups: []uint32{f.gid}}}
	return cmd
}
func listenerSignaledChild(t *testing.T, f nativeFixture, mode string) *exec.Cmd {
	t.Helper()
	cmd := listenerChild(t, f, mode, f.uid)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = out.Close() })
	startActionChild(t, cmd)
	ready := make(chan string, 1)
	go func() {
		line, err := bufio.NewReader(out).ReadString('\n')
		if err != nil {
			ready <- ""
		} else {
			ready <- line
		}
	}()
	select {
	case line := <-ready:
		if line != "ready\n" {
			t.Fatal("child not connected/negotiated")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("child signal timeout")
	}
	return cmd
}
func TestNativePropertyListener(t *testing.T) {
	if os.Getenv("HORVIUM_ACTION_PEER_VM") != "1" {
		t.Skip("requires disposable peer opt-in")
	}
	f := fixture(t)
	cases := []string{"review-denied", "submit-denied", "wrong-uid", "root-denied", "malformed", "legacy-denied", "idle-expiry", "capacity-recovery", "shutdown-negotiation", "shutdown-request", "existing-file", "existing-symlink", "existing-socket", "unsafe-directory", "nonroot-constructor", "replacement-preserved", "single-serve", "fixtures-teardown"}
	for _, name := range cases[:len(cases)-1] {
		t.Run(name, func(t *testing.T) {
			local := f
			local.config.Timeout = 3 * time.Second
			local.config.MaxClients = 1
			switch name {
			case "existing-file", "existing-symlink", "existing-socket":
				var before os.FileInfo
				switch name {
				case "existing-file":
					if os.WriteFile(local.config.Socket, []byte("preserved"), 0600) != nil {
						t.Fatal("file fixture")
					}
				case "existing-symlink":
					if os.Symlink(filepath.Join(f.dir, "absent"), local.config.Socket) != nil {
						t.Fatal("symlink fixture")
					}
				case "existing-socket":
					l, err := net.ListenUnix("unix", &net.UnixAddr{Name: local.config.Socket, Net: "unix"})
					if err != nil {
						t.Fatal(err)
					}
					l.SetUnlinkOnClose(false)
					defer l.Close()
				}
				var err error
				before, err = os.Lstat(local.config.Socket)
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(local.config.Socket)
				s, err := newPropertyServer(local.config)
				if err != nil {
					t.Fatal(err)
				}
				if err := s.serve(context.Background()); !errors.Is(err, ErrConfig) {
					t.Fatal("existing entry adopted", err)
				}
				after, err := os.Lstat(local.config.Socket)
				if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
					t.Fatal("existing entry changed", err)
				}
				if name == "existing-file" {
					if data, err := os.ReadFile(local.config.Socket); err != nil || string(data) != "preserved" {
						t.Fatal("file bytes changed")
					}
				}
				return
			case "unsafe-directory":
				run := filepath.Dir(local.config.Socket)
				defer func() {
					if os.Chmod(run, 0750) != nil || os.Chown(run, 0, int(f.gid)) != nil {
						t.Error("directory fixture restoration")
					}
				}()
				for _, mode := range []os.FileMode{0700, 0770, 0777, 0750 | os.ModeSetgid} {
					if os.Chmod(run, mode) != nil {
						t.Fatal("directory mode")
					}
					if s, err := newPropertyServer(local.config); s != nil || !errors.Is(err, ErrConfig) {
						t.Fatal("unsafe directory accepted", err)
					}
				}
				if os.Chmod(run, 0750) != nil || os.Chown(run, int(f.uid), int(f.gid)) != nil {
					t.Fatal("owner fixture")
				}
				if s, err := newPropertyServer(local.config); s != nil || !errors.Is(err, ErrConfig) {
					t.Fatal("nonroot directory accepted", err)
				}
				if os.Chown(run, 0, int(f.gid)) != nil {
					t.Fatal("restore root directory")
				}
				alias := filepath.Join(f.dir, "alias")
				if os.Symlink(run, alias) != nil {
					t.Fatal("alias fixture")
				}
				defer os.Remove(alias)
				aliasConfig := local.config
				aliasConfig.Socket = filepath.Join(alias, "s")
				if s, err := newPropertyServer(aliasConfig); s != nil || !errors.Is(err, ErrConfig) {
					t.Fatal("symlink directory accepted", err)
				}
				return
			case "nonroot-constructor":
				cmd := listenerChild(t, local, "nonroot-constructor", f.uid)
				startActionChild(t, cmd)
				waitActionChild(t, cmd)
				return
			}
			if name == "idle-expiry" {
				local.config.Timeout = 350 * time.Millisecond
			}
			if name == "capacity-recovery" {
				local.config.Timeout = 750 * time.Millisecond
			}
			r := startPropertyListener(t, local)
			switch name {
			case "replacement-preserved":
				if os.Remove(local.config.Socket) != nil || os.WriteFile(local.config.Socket, []byte("replacement"), 0600) != nil {
					t.Fatal("replacement fixture")
				}
				r.stop(t)
				if data, err := os.ReadFile(local.config.Socket); err != nil || string(data) != "replacement" {
					t.Fatal("foreign replacement removed", err)
				}
				if os.Remove(local.config.Socket) != nil {
					t.Fatal("replacement fixture cleanup")
				}
				return
			case "single-serve":
				// Snapshot owner fields under the original lock; a mutex itself is never copied.
				r.server.mu.Lock()
				copy := propertyServer{identity: r.server.identity, config: r.server.config, attempted: r.server.attempted, active: r.server.active}
				r.server.mu.Unlock()
				if err := copy.serve(context.Background()); !errors.Is(err, ErrPermission) {
					t.Fatal("copied server started", err)
				}
				if err := r.server.serve(context.Background()); !errors.Is(err, ErrProtocol) {
					t.Fatal("original server restarted", err)
				}
				cmd := listenerChild(t, local, "review-denied", f.uid)
				startActionChild(t, cmd)
				waitActionChild(t, cmd)
			case "shutdown-negotiation", "shutdown-request", "idle-expiry", "capacity-recovery":
				mode := "idle"
				if name == "shutdown-request" {
					mode = "request-idle"
				}
				cmd := listenerSignaledChild(t, local, mode)
				lease := listenerWorkers(t, r.server, 1)
				if name == "capacity-recovery" {
					extra := listenerChild(t, local, "overflow", f.uid)
					startActionChild(t, extra)
					waitActionChild(t, extra)
				}
				if strings.HasPrefix(name, "shutdown") {
					// The signaled client and published original worker are both real. For
					// negotiation wait until the existing factory has claimed that lease.
					if name == "shutdown-negotiation" {
						deadline := time.Now().Add(time.Second)
						for {
							lease.owner.mu.Lock()
							started := lease.bindingAttempted
							lease.owner.mu.Unlock()
							if started {
								break
							}
							if time.Now().After(deadline) {
								t.Fatal("negotiation not entered")
							}
							time.Sleep(time.Millisecond)
						}
					}
					r.stop(t)
				}
				waitActionChild(t, cmd)
				listenerWorkers(t, r.server, 0)
				if name == "capacity-recovery" {
					next := listenerChild(t, local, "review-denied", f.uid)
					startActionChild(t, next)
					waitActionChild(t, next)
					listenerWorkers(t, r.server, 0)
				}
			default:
				uid := f.uid
				if name == "wrong-uid" {
					uid = f.denied
				}
				if name == "root-denied" {
					uid = 0
				}
				cmd := listenerChild(t, local, name, uid)
				startActionChild(t, cmd)
				waitActionChild(t, cmd)
				listenerWorkers(t, r.server, 0)
			}
			r.stop(t)
			if _, err := os.Lstat(local.config.Socket); !os.IsNotExist(err) {
				t.Fatal("owned socket retained", err)
			}
		})
	}
	t.Run("fixtures-teardown", func(t *testing.T) {
		if _, err := os.Lstat(f.config.Socket); !os.IsNotExist(err) {
			t.Fatal("socket retained")
		}
		if os.RemoveAll(f.dir) != nil {
			t.Fatal("fixture removal")
		}
		if _, err := os.Lstat(f.dir); !os.IsNotExist(err) {
			t.Fatal("fixture retained")
		}
	})
	if t.Failed() {
		return
	}
	fields := make([]string, len(cases))
	for i, n := range cases {
		fields[i] = n + "=pass"
	}
	fmt.Println("PROPERTY_LISTENER_RESULT " + strings.Join(fields, " "))
}
func TestNativePropertyListenerChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_PROPERTY_LISTENER_CHILD")
	if mode == "" {
		t.Skip("reexec only")
	}
	var c Config
	if json.Unmarshal([]byte(os.Getenv("HORVIUM_HELPER_CONFIG")), &c) != nil {
		t.Fatal("fixture config")
	}
	if mode == "nonroot-constructor" {
		if s, err := newPropertyServer(c); s != nil || !errors.Is(err, ErrPermission) {
			t.Fatal("nonroot listener constructed", err)
		}
		return
	}
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: c.Socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(4 * time.Second))
	if uid, err := peerUID(conn); err != nil || uid != 0 {
		t.Fatal("actual listener is not root", err)
	}
	if mode == "idle" {
		fmt.Println("ready")
	}
	if mode == "idle" || mode == "overflow" || mode == "wrong-uid" || mode == "root-denied" {
		if mode == "wrong-uid" || mode == "root-denied" {
			// Send a canonical hello without claiming a local UID. A permissive
			// listener would return ready; an idle timeout cannot qualify UID denial.
			_ = WriteActionNegotiation(conn, "listener", strings.Repeat("a", 64))
			_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		}
		if mode == "overflow" {
			_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		}
		var b [1]byte
		n, err := conn.Read(b[:])
		var timeout net.Error
		if n != 0 || err == nil || errors.As(err, &timeout) && timeout.Timeout() {
			t.Fatal("unauthorized/idle/overflow listener emitted bytes", err)
		}
		return
	}
	if mode == "legacy-denied" {
		if _, err := conn.Write(frame(`{"version":1,"request_id":"listener","operation":"inventory"}`)); err != nil {
			t.Fatal(err)
		}
		var b [1]byte
		if n, err := conn.Read(b[:]); n != 0 || err != io.EOF {
			t.Fatal("legacy operation dispatched", err)
		}
		return
	}
	x, err := ConnectPropertyClient(context.Background(), conn, c.ManagementUID, "listener", strings.Repeat("a", 64), 3*time.Second, func(ctx context.Context) error { return ctx.Err() })
	if err != nil || x == nil {
		t.Fatal("actual client negotiation", err)
	}
	defer x.Close()
	if mode == "request-idle" {
		fmt.Println("ready")
		var b [1]byte
		if n, err := conn.Read(b[:]); n != 0 || err != io.EOF {
			t.Fatal("shutdown request not interrupted", err)
		}
		return
	}
	if mode == "malformed" {
		var b bytes.Buffer
		if WritePropertyReview(&b, "listener", "100", propertyClientIntent()) != nil {
			t.Fatal("intent fixture")
		}
		payload := strings.Replace(string(b.Bytes()[4:]), `"version":2`, `"version":2,"uid":0`, 1)
		if _, err := conn.Write(frame(payload)); err != nil {
			t.Fatal(err)
		}
		var q [1]byte
		if n, err := conn.Read(q[:]); n != 0 || err != io.EOF {
			t.Fatal("forged request dispatched", err)
		}
		return
	}
	if mode == "submit-denied" {
		j, d, err := x.Submit("100", strings.Repeat("a", 64), strings.Repeat("b", 64))
		var refusal *PropertyRefusalError
		if !errors.As(err, &refusal) || refusal.Code != "denied" || j.ID != "" || d {
			t.Fatal("submission enabled", err)
		}
	} else {
		r, err := x.Review("100", propertyClientIntent())
		var refusal *PropertyRefusalError
		if !errors.As(err, &refusal) || refusal.Code != "denied" || r.ID != "" {
			t.Fatal("review enabled", err)
		}
	}
	sessionClientClosed(t, conn)
}
