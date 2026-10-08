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
	"strings"
	"syscall"
	"testing"
	"time"
)

func admittedIdleChild(t *testing.T, f nativeFixture, uid uint32) *exec.Cmd {
	t.Helper()
	data, err := json.Marshal(f.config)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(f.binary, "-test.run=^TestNativeAdmittedPropertyChild$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_ADMITTED_PROPERTY_CHILD=1", "HORVIUM_HELPER_CONFIG=" + string(data)}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: f.gid, Groups: []uint32{f.gid}}}
	return cmd
}
func admittedRaw(t *testing.T, f nativeFixture, l *net.UnixListener, p *propertyConnections, uid uint32) (*propertyConnection, *exec.Cmd) {
	t.Helper()
	cmd := admittedIdleChild(t, f, uid)
	startActionChild(t, cmd)
	raw, err := l.AcceptUnix()
	if err != nil {
		t.Fatal(err)
	}
	c, err := p.admit(raw)
	if err != nil {
		t.Fatal("native connection admission", err)
	}
	return c, cmd
}
func admittedEmpty(t *testing.T, p *propertyConnections, c *propertyConnection) {
	t.Helper()
	connectionDone(t, c)
	p.mu.Lock()
	n := len(p.owned)
	p.mu.Unlock()
	if n != 0 {
		t.Fatal("exchange retained capacity")
	}
}
func admittedRoundTrip(t *testing.T, f nativeFixture, l *net.UnixListener, p *propertyConnections, mode string) {
	t.Helper()
	cmd := propertyChild(t, f, mode, f.uid)
	startActionChild(t, cmd)
	raw, err := l.AcceptUnix()
	if err != nil {
		t.Fatal(err)
	}
	c, err := p.admit(raw)
	if err != nil {
		t.Fatal("admit", err)
	}
	a, err := c.accept(f.uid)
	if err != nil || a == nil {
		t.Fatal("admitted original negotiation", err)
	}
	defer a.Close()
	if a.lease != c || a.exchange.peer.connection != raw {
		t.Fatal("original replaced")
	}
	original, _ := c.ctx.Deadline()
	actual, _ := a.exchange.peer.lifetime.Deadline()
	if !actual.Equal(original) {
		t.Fatal("negotiation renewed admission deadline")
	}
	copy := *a
	if req, actor, err := copy.receive(); req != (propertyRequest{}) || actor != (actionIdentity{}) || err == nil {
		t.Fatal("copied composed owner accepted")
	}
	if copy.Close() != nil {
		t.Fatal("copied close")
	}
	req, actor, err := a.receive()
	if mode == "malformed-request" {
		if err == nil || req != (propertyRequest{}) || actor != (actionIdentity{}) {
			t.Fatal("malformed projection")
		}
	} else {
		if err != nil || actor != (actionIdentity{f.uid, "native-property", strings.Repeat("a", 64)}) || req.PoolGUID != "100" {
			t.Fatal("original actual UID/request lost", err)
		}
		switch mode {
		case "submit":
			err = a.replySubmission(replyJob(), false)
		case "refused":
			err = a.refuse("uncertain")
		default:
			err = a.replyReview(replyReview(t))
		}
		if err != nil {
			t.Fatal("admitted terminal reply", err)
		}
	}
	admittedEmpty(t, p, c)
	if a.refuse("denied") == nil {
		t.Fatal("retired writer reused")
	}
	waitActionChild(t, cmd)
}
func admittedNegotiating(t *testing.T, c *propertyConnection, uid uint32) <-chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		a, err := c.accept(uid)
		if a != nil {
			_ = a.Close()
			t.Error("interrupted negotiation returned an owner")
		}
		done <- err
	}()
	deadline := time.Now().Add(time.Second)
	for {
		c.owner.mu.Lock()
		started := c.bindingAttempted
		c.owner.mu.Unlock()
		if started {
			return done
		}
		select {
		case err := <-done:
			t.Fatal("negotiation returned before interruption", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("negotiation never claimed lease")
		}
		time.Sleep(time.Millisecond)
	}
}
func admittedInterrupted(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("interrupted exchange succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("exchange I/O not interrupted")
	}
}
func TestNativeAdmittedPropertyExchange(t *testing.T) {
	if os.Getenv("HORVIUM_ACTION_PEER_VM") != "1" {
		t.Skip("requires disposable peer opt-in")
	}
	f := fixture(t)
	cases := []string{"review", "submit", "refused", "wrong-uid", "root-denied", "terminal-slot-reuse", "malformed-slot-reuse", "capacity-recovery", "negotiation-deadline", "phase-deadline", "earlier-parent-deadline", "shutdown-negotiation", "shutdown-request", "overlap-negotiation", "duplicate-lease", "fixtures-teardown"}
	for _, name := range cases[:len(cases)-1] {
		t.Run(name, func(t *testing.T) {
			l := actionListener(t, f)
			ctx := context.Background()
			timeout := 3 * time.Second
			if name == "negotiation-deadline" || name == "phase-deadline" {
				timeout = 300 * time.Millisecond
			}
			if name == "earlier-parent-deadline" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 300*time.Millisecond)
				defer cancel()
			}
			p := connectionAdmission(t, ctx, 1, timeout)
			switch name {
			case "review", "submit", "refused":
				admittedRoundTrip(t, f, l, p, name)
			case "terminal-slot-reuse":
				admittedRoundTrip(t, f, l, p, "review")
				admittedRoundTrip(t, f, l, p, "submit")
			case "malformed-slot-reuse":
				admittedRoundTrip(t, f, l, p, "malformed-request")
				admittedRoundTrip(t, f, l, p, "review")
			case "capacity-recovery":
				c, first := admittedRaw(t, f, l, p, f.uid)
				second := admittedIdleChild(t, f, f.uid)
				startActionChild(t, second)
				raw, err := l.AcceptUnix()
				if err != nil {
					t.Fatal(err)
				}
				if overflow, err := p.admit(raw); overflow != nil || !errors.Is(err, ErrLimit) {
					t.Fatal("capacity bypassed", err)
				}
				waitActionChild(t, second)
				_ = c.Close()
				admittedEmpty(t, p, c)
				waitActionChild(t, first)
				admittedRoundTrip(t, f, l, p, "review")
			case "wrong-uid", "root-denied":
				uid := f.denied
				if name == "root-denied" {
					uid = 0
				}
				c, cmd := admittedRaw(t, f, l, p, uid)
				if a, err := c.accept(f.uid); a != nil || !errors.Is(err, ErrPermission) {
					t.Fatal("kernel UID denial lost", err)
				}
				admittedEmpty(t, p, c)
				waitActionChild(t, cmd)
				admittedRoundTrip(t, f, l, p, "review")
			case "negotiation-deadline", "earlier-parent-deadline":
				c, cmd := admittedRaw(t, f, l, p, f.uid)
				deadline, _ := c.ctx.Deadline()
				if parent, ok := ctx.Deadline(); ok && !parent.Equal(deadline) {
					t.Fatal("earlier parent deadline lost")
				}
				if a, err := c.accept(f.uid); a != nil || err == nil {
					t.Fatal("idle negotiation succeeded")
				}
				admittedEmpty(t, p, c)
				if time.Now().Before(deadline) || c.ctx.Err() == nil {
					t.Fatal("admission deadline not enforced", c.ctx.Err())
				}
				waitActionChild(t, cmd)
			case "shutdown-negotiation", "overlap-negotiation", "duplicate-lease":
				c, cmd := admittedRaw(t, f, l, p, f.uid)
				if name == "duplicate-lease" {
					if dup, err := p.admit(c.connection); dup != nil || !errors.Is(err, ErrProtocol) {
						t.Fatal("duplicate original admitted", err)
					}
				} else {
					done := admittedNegotiating(t, c, f.uid)
					if name == "shutdown-negotiation" {
						if p.Close() != nil {
							t.Fatal("shutdown")
						}
					} else {
						if a, err := c.accept(f.uid); a != nil || !errors.Is(err, ErrProtocol) {
							t.Fatal("overlapping negotiation accepted", err)
						}
					}
					admittedInterrupted(t, done)
				}
				admittedEmpty(t, p, c)
				waitActionChild(t, cmd)
			case "phase-deadline", "shutdown-request":
				cmd := propertyChild(t, f, "idle-deadline", f.uid)
				var ready io.ReadCloser
				if name == "shutdown-request" {
					var err error
					ready, err = cmd.StdoutPipe()
					if err != nil {
						t.Fatal("child readiness pipe")
					}
					defer ready.Close()
				}
				startActionChild(t, cmd)
				raw, err := l.AcceptUnix()
				if err != nil {
					t.Fatal(err)
				}
				c, err := p.admit(raw)
				if err != nil {
					t.Fatal(err)
				}
				original, _ := c.ctx.Deadline()
				if name == "phase-deadline" {
					time.Sleep(75 * time.Millisecond)
				}
				a, err := c.accept(f.uid)
				if err != nil || a == nil {
					t.Fatal("phase negotiation", err)
				}
				defer a.Close()
				actual, _ := a.exchange.peer.lifetime.Deadline()
				if !actual.Equal(original) {
					t.Fatal("phase renewed deadline")
				}
				if name == "shutdown-request" {
					// Root ready write does not prove that the child completed
					// its own final kernel/quiet-peer check. Observe that phase
					// before intentionally shutting down its request read.
					completed := make(chan string, 1)
					go func() { line, _ := bufio.NewReader(ready).ReadString('\n'); completed <- line }()
					select {
					case line := <-completed:
						if line != "ready\n" {
							t.Fatal("child did not complete negotiation")
						}
					case <-time.After(time.Second):
						t.Fatal("child negotiation witness deadline")
					}
					done := make(chan error, 1)
					go func() {
						req, actor, err := a.receive()
						if req != (propertyRequest{}) || actor != (actionIdentity{}) {
							t.Error("shutdown request projected metadata")
						}
						done <- err
					}()
					deadline := time.Now().Add(time.Second)
					for {
						if !a.exchange.mu.TryLock() {
							break
						}
						a.exchange.mu.Unlock()
						if time.Now().After(deadline) {
							t.Fatal("receive never entered")
						}
						time.Sleep(time.Millisecond)
					}
					if p.Close() != nil {
						t.Fatal("shutdown")
					}
					admittedInterrupted(t, done)
				} else {
					if req, actor, err := a.receive(); req != (propertyRequest{}) || actor != (actionIdentity{}) || err == nil {
						t.Fatal("phase outlived admission")
					}
					if time.Now().Before(original) || c.ctx.Err() == nil {
						t.Fatal("phase deadline lost", c.ctx.Err())
					}
				}
				admittedEmpty(t, p, c)
				waitActionChild(t, cmd)
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
	for i, name := range cases {
		fields[i] = name + "=pass"
	}
	fmt.Println("ADMITTED_PROPERTY_RESULT " + strings.Join(fields, " "))
}
func TestNativeAdmittedPropertyChild(t *testing.T) {
	if os.Getenv("HORVIUM_ADMITTED_PROPERTY_CHILD") != "1" {
		t.Skip("reexec fixture only")
	}
	var c Config
	if json.Unmarshal([]byte(os.Getenv("HORVIUM_HELPER_CONFIG")), &c) != nil {
		t.Fatal("fixture")
	}
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: c.Socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	var b [1]byte
	if n, err := conn.Read(b[:]); n != 0 || err != io.EOF {
		t.Fatal("denied/idle admission emitted bytes or stayed open", err)
	}
}
