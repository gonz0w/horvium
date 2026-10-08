// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func sessionPropertyChild(t *testing.T, f nativeFixture, mode string) *exec.Cmd {
	t.Helper()
	data, err := json.Marshal(f.config)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(f.binary, "-test.run=^TestNativeSessionPropertyChild$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_SESSION_PROPERTY_CHILD=" + mode, "HORVIUM_HELPER_CONFIG=" + string(data)}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: f.uid, Gid: f.gid, Groups: []uint32{f.gid}}}
	return cmd
}
func TestNativeSessionPropertyClient(t *testing.T) {
	if os.Getenv("HORVIUM_ACTION_PEER_VM") != "1" {
		t.Skip("requires disposable peer opt-in")
	}
	f := fixture(t)
	cases := []string{"review", "submit", "duplicate", "refused", "before-hello", "after-hello", "after-ready", "before-request", "after-request", "after-reply", "wrong-scope", "invalid-selection", "request-deadline", "fixtures-teardown"}
	for _, name := range cases[:len(cases)-1] {
		t.Run(name, func(t *testing.T) {
			l := actionListener(t, f)
			cmd := sessionPropertyChild(t, f, name)
			startActionChild(t, cmd)
			raw, err := l.AcceptUnix()
			if err != nil {
				t.Fatal(err)
			}
			defer raw.Close()
			_ = raw.SetDeadline(time.Now().Add(3 * time.Second))
			if name == "before-hello" || name == "wrong-scope" {
				var b [1]byte
				if n, err := raw.Read(b[:]); n != 0 || err != io.EOF {
					t.Fatal("pre-negotiation guard leaked bytes", err)
				}
				waitActionChild(t, cmd)
				return
			}
			if name == "after-hello" {
				hello, err := readActionNegotiation(raw)
				if err != nil || hello.ID != "session-property" {
					t.Fatal("hello", err)
				}
				var b [1]byte
				if n, err := raw.Read(b[:]); n != 0 || err != io.EOF {
					t.Fatal("revoked hello waited/retried", err)
				}
				waitActionChild(t, cmd)
				return
			}
			p := connectionAdmission(t, context.Background(), 1, 3*time.Second)
			lease, err := p.admit(raw)
			if err != nil {
				t.Fatal(err)
			}
			x, err := lease.accept(f.uid)
			if name == "after-ready" {
				if x != nil {
					if req, actor, e := x.receive(); req != (propertyRequest{}) || actor != (actionIdentity{}) || e == nil {
						t.Fatal("revoked ready sent request")
					}
					_ = x.Close()
				} else if err == nil {
					t.Fatal("missing root error")
				}
				admittedEmpty(t, p, lease)
				waitActionChild(t, cmd)
				return
			}
			if err != nil || x == nil {
				t.Fatal("native admitted negotiation", err)
			}
			defer x.Close()
			req, actor, err := x.receive()
			if name == "before-request" || name == "invalid-selection" {
				if err == nil || req != (propertyRequest{}) || actor != (actionIdentity{}) {
					t.Fatal("invalid/revoked selection dispatched")
				}
				admittedEmpty(t, p, lease)
				waitActionChild(t, cmd)
				return
			}
			if name == "after-request" {
				// A sent request can race the client's retirement; neither parsed metadata
				// nor an observed disconnect authorizes a root operation in this fixture.
				if err == nil && req.ID != "session-property" {
					t.Fatal("correlation lost")
				}
				_ = x.Close()
				admittedEmpty(t, p, lease)
				waitActionChild(t, cmd)
				return
			}
			if err != nil || actor != (actionIdentity{f.uid, "session-property", strings.Repeat("a", 64)}) || req.PoolGUID != "100" {
				t.Fatal("actual peer/request binding lost", err)
			}
			if name == "request-deadline" {
				var b [1]byte
				if n, e := raw.Read(b[:]); n != 0 || e != io.EOF {
					t.Fatal("client did not retire missing reply", e)
				}
				_ = x.Close()
				admittedEmpty(t, p, lease)
				waitActionChild(t, cmd)
				return
			}
			switch name {
			case "submit", "duplicate":
				err = x.replySubmission(replyJob(), name == "duplicate")
			case "refused":
				err = x.refuse("uncertain")
			default:
				err = x.replyReview(replyReview(t))
			}
			if err != nil {
				t.Fatal("synthetic reply", err)
			}
			admittedEmpty(t, p, lease)
			waitActionChild(t, cmd)
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
	fmt.Println("SESSION_PROPERTY_RESULT " + strings.Join(fields, " "))
}
func TestNativeSessionPropertyChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_SESSION_PROPERTY_CHILD")
	if mode == "" {
		t.Skip("reexec only")
	}
	var config Config
	if json.Unmarshal([]byte(os.Getenv("HORVIUM_HELPER_CONFIG")), &config) != nil {
		t.Fatal("fixture")
	}
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: config.Socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	calls, failAt := 0, 0
	switch mode {
	case "before-hello", "wrong-scope":
		failAt = 1
	case "after-hello":
		failAt = 2
	case "after-ready":
		failAt = 3
	case "before-request":
		failAt = 4
	case "after-request":
		failAt = 5
	case "after-reply":
		failAt = 6
	}
	check := func(ctx context.Context) error {
		calls++
		if ctx == nil || ctx.Err() != nil {
			return ErrPermission
		}
		if calls == failAt {
			return ErrPermission
		}
		return nil
	}
	timeout := 3 * time.Second
	if mode == "request-deadline" {
		timeout = 300 * time.Millisecond
	}
	x, err := ConnectPropertyClient(context.Background(), conn, config.ManagementUID, "session-property", strings.Repeat("a", 64), timeout, check)
	if failAt > 0 && failAt <= 3 {
		if x != nil || err == nil || calls != failAt {
			t.Fatal("factory guard cut accepted", calls, err)
		}
		sessionClientClosed(t, conn)
		return
	}
	if err != nil || x == nil {
		t.Fatal("native guarded factory", err)
	}
	defer x.Close()
	copy := *x
	if r, e := copy.Review("100", propertyClientIntent()); r.ID != "" || e == nil {
		t.Fatal("copied client used original")
	}
	if copy.Close() != nil {
		t.Fatal("copied close")
	}
	if mode == "submit" || mode == "duplicate" {
		j, d, e := x.Submit("100", strings.Repeat("a", 64), strings.Repeat("b", 64))
		sessionClientClosed(t, conn)
		if e != nil || j.ID != strings.Repeat("c", 64) || d != (mode == "duplicate") || calls != 6 {
			t.Fatal("guarded submission", calls, e)
		}
		if j, d, e := x.Submit("100", strings.Repeat("a", 64), strings.Repeat("b", 64)); j.ID != "" || d || e == nil {
			t.Fatal("terminal client reused")
		}
		return
	}
	intent := propertyClientIntent()
	if mode == "invalid-selection" {
		intent.Value = "0"
	}
	r, err := x.Review("100", intent)
	sessionClientClosed(t, conn)
	if failAt > 3 {
		if err == nil || r.ID != "" || calls != failAt {
			t.Fatal("phase revocation accepted", calls, err)
		}
		return
	}
	if mode == "request-deadline" || mode == "invalid-selection" {
		if err == nil || r.ID != "" {
			t.Fatal("deadline/invalid selection accepted")
		}
		return
	}
	if mode == "refused" {
		var refusal *PropertyRefusalError
		if !errors.As(err, &refusal) || refusal.Code != "uncertain" || r.ID != "" || calls != 6 {
			t.Fatal("guarded refusal", calls, err)
		}
		return
	}
	if err != nil || r.ID != strings.Repeat("a", 64) || calls != 6 {
		t.Fatal("guarded review", calls, err)
	}
	if r, e := x.Review("100", intent); e == nil || r.ID != "" {
		t.Fatal("terminal client reused")
	}
}

func sessionClientClosed(t *testing.T, conn *net.UnixConn) {
	t.Helper()
	// EOF alone can leave the local descriptor open. This check must precede
	// any deferred explicit close or attempted second operation in the fixture.
	if err := conn.SetDeadline(time.Now()); !errors.Is(err, net.ErrClosed) {
		t.Fatal("terminal method retained its local socket", err)
	}
}
