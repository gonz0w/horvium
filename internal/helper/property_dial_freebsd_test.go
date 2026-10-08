// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"bufio"
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
	"syscall"
	"testing"
	"time"
)

func configuredPropertyChild(t *testing.T, f nativeFixture, mode string, uid uint32) *exec.Cmd {
	t.Helper()
	data, err := json.Marshal(f.config)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(f.binary, "-test.run=^TestNativeConfiguredPropertyChild$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_CONFIGURED_PROPERTY_CHILD=" + mode, "HORVIUM_HELPER_CONFIG=" + string(data)}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: f.gid, Groups: []uint32{f.gid}}}
	return cmd
}
func TestNativeConfiguredPropertyClient(t *testing.T) {
	if os.Getenv("HORVIUM_ACTION_PEER_VM") != "1" {
		t.Skip("requires disposable peer opt-in")
	}
	f := fixture(t)
	cases := []string{"review-denied", "submit-denied", "revoked-before-dial", "cancelled-before-dial", "wrong-local-uid", "unsafe-directory", "unsafe-socket", "replaced-after-dial", "revoked-after-dial", "original-deadline", "negotiation-cancellation", "fixtures-teardown"}
	for _, name := range cases[:len(cases)-1] {
		t.Run(name, func(t *testing.T) {
			local := f
			local.config.Timeout = time.Second
			local.config.MaxClients = 1
			uid := f.uid
			if name == "wrong-local-uid" {
				uid = 0
			}
			cmd := configuredPropertyChild(t, local, name, uid)
			switch name {
			case "revoked-before-dial", "cancelled-before-dial", "wrong-local-uid", "unsafe-directory", "unsafe-socket":
				l := actionListener(t, local)
				if name == "unsafe-directory" {
					dir := filepath.Dir(local.config.Socket)
					if os.Chmod(dir, 0770) != nil {
						t.Fatal("directory fixture")
					}
					defer func() {
						if os.Chmod(dir, 0750) != nil {
							t.Error("restore directory")
						}
					}()
				}
				if name == "unsafe-socket" {
					if os.Chmod(local.config.Socket, 0666) != nil {
						t.Fatal("socket fixture")
					}
					defer func() {
						if os.Chmod(local.config.Socket, 0660) != nil {
							t.Error("restore socket")
						}
					}()
				}
				startActionChild(t, cmd)
				waitActionChild(t, cmd)
				_ = l.SetDeadline(time.Now().Add(150 * time.Millisecond))
				raw, err := l.AcceptUnix()
				if raw != nil {
					_ = raw.Close()
					t.Fatal("preflight connected")
				}
				var timeout net.Error
				if !errors.As(err, &timeout) || !timeout.Timeout() {
					t.Fatal("preflight absence not observed", err)
				}
				return
			case "negotiation-cancellation":
				l := actionListener(t, local)
				input, err := cmd.StdinPipe()
				if err != nil {
					t.Fatal(err)
				}
				defer input.Close()
				startActionChild(t, cmd)
				raw, err := l.AcceptUnix()
				if err != nil {
					t.Fatal(err)
				}
				defer raw.Close()
				_ = raw.SetDeadline(time.Now().Add(2 * time.Second))
				h, err := readActionNegotiation(raw)
				if err != nil || h.ID != "configured" {
					t.Fatal("actual hello", err)
				}
				if _, err := io.WriteString(input, "cancel\n"); err != nil {
					t.Fatal(err)
				}
				var b [1]byte
				if n, err := raw.Read(b[:]); n != 0 || err != io.EOF {
					t.Fatal("blocked ready not retired", err)
				}
				waitActionChild(t, cmd)
				return
			}
			r := startPropertyListener(t, local)
			if name == "replaced-after-dial" {
				input, err := cmd.StdinPipe()
				if err != nil {
					t.Fatal(err)
				}
				defer input.Close()
				output, err := cmd.StdoutPipe()
				if err != nil {
					t.Fatal(err)
				}
				defer output.Close()
				startActionChild(t, cmd)
				ready := make(chan string, 1)
				go func() { line, _ := bufio.NewReader(output).ReadString('\n'); ready <- line }()
				select {
				case line := <-ready:
					if line != "connected\n" {
						t.Fatal("post-dial signal")
					}
				case <-time.After(2 * time.Second):
					t.Fatal("post-dial signal timeout")
				}
				if os.Remove(local.config.Socket) != nil || os.WriteFile(local.config.Socket, []byte("replacement"), 0600) != nil {
					t.Fatal("replacement fixture")
				}
				if _, err := io.WriteString(input, "continue\n"); err != nil {
					t.Fatal(err)
				}
				waitActionChild(t, cmd)
				r.stop(t)
				if data, err := os.ReadFile(local.config.Socket); err != nil || string(data) != "replacement" {
					t.Fatal("replacement changed", err)
				}
				if os.Remove(local.config.Socket) != nil {
					t.Fatal("replacement cleanup")
				}
				return
			}
			startActionChild(t, cmd)
			waitActionChild(t, cmd)
			r.stop(t)
			if _, err := os.Lstat(local.config.Socket); !os.IsNotExist(err) {
				t.Fatal("owned socket retained", err)
			}
		})
	}
	t.Run("fixtures-teardown", func(t *testing.T) {
		if os.RemoveAll(f.dir) != nil {
			t.Fatal("fixture removal")
		}
		if _, err := os.Lstat(f.dir); !os.IsNotExist(err) {
			t.Fatal("fixture retained", err)
		}
	})
	if t.Failed() {
		return
	}
	fields := make([]string, len(cases))
	for i, n := range cases {
		fields[i] = n + "=pass"
	}
	fmt.Println("CONFIGURED_PROPERTY_RESULT " + strings.Join(fields, " "))
}
func TestNativeConfiguredPropertyChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_CONFIGURED_PROPERTY_CHILD")
	if mode == "" {
		t.Skip("reexec only")
	}
	var c Config
	if json.Unmarshal([]byte(os.Getenv("HORVIUM_HELPER_CONFIG")), &c) != nil {
		t.Fatal("fixture config")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if mode == "cancelled-before-dial" {
		cancel()
	}
	if mode == "original-deadline" {
		var stop context.CancelFunc
		ctx, stop = context.WithTimeout(ctx, 750*time.Millisecond)
		defer stop()
	}
	if mode == "negotiation-cancellation" {
		go func() { _, _ = bufio.NewReader(os.Stdin).ReadString('\n'); cancel() }()
	}
	calls := 0
	var original context.Context
	var deadline time.Time
	check := func(current context.Context) error {
		calls++
		d, ok := current.Deadline()
		if !ok {
			t.Fatal("unbounded dial phase")
		}
		if calls == 1 {
			original = current
			deadline = d
		} else if !d.Equal(deadline) {
			t.Fatal("dial phase renewed original deadline")
		}
		if mode == "original-deadline" && calls == 1 {
			expected, _ := ctx.Deadline()
			if !d.Equal(expected) {
				t.Fatal("shorter parent deadline discarded")
			}
			time.Sleep(10 * time.Millisecond)
		}
		if mode == "revoked-before-dial" || mode == "revoked-after-dial" && calls == 2 {
			return ErrPermission
		}
		if mode == "replaced-after-dial" && calls == 2 {
			fmt.Println("connected")
			ready := make(chan string, 1)
			go func() { line, _ := bufio.NewReader(os.Stdin).ReadString('\n'); ready <- line }()
			select {
			case line := <-ready:
				if line != "continue\n" {
					return ErrPermission
				}
			case <-current.Done():
				return current.Err()
			}
		}
		return current.Err()
	}
	p, err := DialPropertyClient(ctx, c, "configured", strings.Repeat("a", 64), check)
	switch mode {
	case "revoked-before-dial", "revoked-after-dial", "cancelled-before-dial", "wrong-local-uid", "unsafe-directory", "unsafe-socket", "replaced-after-dial", "negotiation-cancellation":
		if p != nil || err == nil {
			t.Fatal("configured denial bypassed")
		}
		switch mode {
		case "revoked-before-dial":
			if !errors.Is(err, ErrPermission) || calls != 1 {
				t.Fatal("before-dial revocation", err, calls)
			}
		case "revoked-after-dial":
			if !errors.Is(err, ErrPermission) || calls != 2 {
				t.Fatal("after-dial revocation", err, calls)
			}
		case "cancelled-before-dial":
			if !errors.Is(err, context.Canceled) || calls != 0 {
				t.Fatal("cancelled dial", err, calls)
			}
		case "wrong-local-uid":
			if !errors.Is(err, ErrPermission) || calls != 0 {
				t.Fatal("actual wrong local UID", err, calls)
			}
		case "unsafe-directory", "unsafe-socket":
			if !errors.Is(err, ErrConfig) || calls != 1 {
				t.Fatal("unsafe namespace dial", err, calls)
			}
		case "replaced-after-dial":
			if !errors.Is(err, ErrConfig) || calls != 2 {
				t.Fatal("post-dial namespace gate", err, calls)
			}
		}
		if original != nil && original.Err() == nil {
			t.Fatal("failed factory retained original dial lifetime")
		}
		return
	}
	if p == nil || err != nil {
		t.Fatal("configured factory", err)
	}
	defer p.Close()
	raw := p.exchange.peer.connection
	copy := *p
	if copy.Close() != nil || original.Err() != nil {
		t.Fatal("copied client retired dial")
	}
	if mode == "submit-denied" {
		j, d, err := p.Submit("100", strings.Repeat("a", 64), strings.Repeat("b", 64))
		var refusal *PropertyRefusalError
		if !errors.As(err, &refusal) || refusal.Code != "denied" || j.ID != "" || d {
			t.Fatal("configured submission enabled", err)
		}
	} else {
		r, err := p.Review("100", propertyClientIntent())
		var refusal *PropertyRefusalError
		if !errors.As(err, &refusal) || refusal.Code != "denied" || r.ID != "" {
			t.Fatal("configured review enabled", err)
		}
	}
	if calls != 8 || !errors.Is(original.Err(), context.Canceled) {
		t.Fatal("phase/lifetime retirement", calls, original.Err())
	}
	sessionClientClosed(t, raw)
}
