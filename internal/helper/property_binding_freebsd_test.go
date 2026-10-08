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
	"sync"
	"syscall"
	"testing"
	"time"
)

func bindingChild(t *testing.T, f nativeFixture, mode string) *exec.Cmd {
	t.Helper()
	data, err := json.Marshal(f.config)
	if err != nil {
		t.Fatal("fixture config")
	}
	cmd := exec.Command(f.binary, "-test.run=^TestNativePropertyBindingChild$", "-test.count=1", "-test.timeout=10s")
	cmd.Env = []string{"HORVIUM_PROPERTY_BINDING_CHILD=" + mode, "HORVIUM_HELPER_CONFIG=" + string(data)}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: f.uid, Gid: f.gid, Groups: []uint32{f.gid}}}
	return cmd
}
func TestNativePropertyRequestBinding(t *testing.T) {
	if os.Getenv("HORVIUM_ACTION_PEER_VM") != "1" {
		t.Skip("requires disposable peer opt-in")
	}
	f := fixture(t)
	cases := []string{"original-review", "original-submit", "snapshot-isolation", "copied-binding", "reconstructed-binding", "foreign-owner", "disconnect", "extra-frame", "expiry", "shutdown", "explicit-close", "terminal-reply", "concurrent-check", "nil-context", "cancelled-context", "fixtures-teardown"}
	for _, name := range cases[:len(cases)-1] {
		t.Run(name, func(t *testing.T) {
			l := actionListener(t, f)
			cmd := bindingChild(t, f, name)
			input, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal("control pipe")
			}
			defer input.Close()
			output, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal("witness pipe")
			}
			defer output.Close()
			startActionChild(t, cmd)
			raw, err := l.AcceptUnix()
			if err != nil {
				t.Fatal("root accept")
			}
			defer raw.Close()
			timeout := 3 * time.Second
			if name == "expiry" {
				timeout = 500 * time.Millisecond
			}
			p := connectionAdmission(t, context.Background(), 1, timeout)
			lease, err := p.admit(raw)
			if err != nil {
				t.Fatal("root admission")
			}
			a, err := lease.accept(f.uid)
			if err != nil {
				t.Fatal("kernel peer negotiation")
			}
			defer a.Close()
			r, err := receivePropertyBinding(a)
			if err != nil || r == nil {
				t.Fatal("original kernel request binding")
			}
			defer r.Close()
			reader := bufio.NewReader(output)
			witness := func(want string) {
				done := make(chan string, 1)
				go func() { line, _ := reader.ReadString('\n'); done <- line }()
				select {
				case line := <-done:
					if line != want+"\n" {
						t.Fatal("fixture witness")
					}
				case <-time.After(2 * time.Second):
					t.Fatal("fixture witness deadline")
				}
			}
			witness("sent")
			req, actor, err := r.snapshot(context.Background())
			if err != nil || req.ID != "binding" || req.PoolGUID != "100" || actor.uid != f.uid || actor.requestID != req.ID || actor.applicationContext != strings.Repeat("a", 64) {
				t.Fatal("original request/peer binding")
			}
			if name == "original-submit" && (req.Operation != "property-submit" || req.PlanID != strings.Repeat("a", 64) || req.SubmissionID != strings.Repeat("b", 64)) {
				t.Fatal("submission binding")
			}
			if name != "original-submit" && (req.Operation != "property-review" || req.Intent != propertyClientIntent()) {
				t.Fatal("review binding")
			}
			terminal := false
			switch name {
			case "snapshot-isolation":
				expectedReq, expectedActor := req, actor
				req.PoolGUID = "200"
				req.Intent.Value = "2"
				actor.uid = 0
				actor.applicationContext = ""
				got, who, err := r.snapshot(context.Background())
				if err != nil || got != expectedReq || who != expectedActor {
					t.Fatal("snapshot changed original binding")
				}
			case "copied-binding", "reconstructed-binding", "foreign-owner":
				var forged *propertyRequestBinding
				if name == "copied-binding" {
					copy := *r
					forged = &copy
				} else {
					forged = &propertyRequestBinding{peer: a, owner: r.owner, request: req, actor: actor}
					forged.identity = forged
					if name == "foreign-owner" {
						forged.owner = &propertyRequestOwner{peer: a, binding: forged}
					}
				}
				if forged.check(context.Background()) == nil || forged.Close() != nil {
					t.Fatal("forged binding passed/closed")
				}
				if err := r.check(context.Background()); err != nil {
					t.Fatal("forged binding retired original")
				}
			case "disconnect", "extra-frame":
				if _, err := io.WriteString(input, "cut\n"); err != nil {
					t.Fatal("cut signal")
				}
				witness("cut")
				req, actor, err := r.snapshot(context.Background())
				if req != (propertyRequest{}) || actor != (actionIdentity{}) || err == nil {
					t.Fatal("dead/extra peer projected metadata")
				}
				if name == "extra-frame" && !errors.Is(err, ErrProtocol) {
					t.Fatal("extra frame not detected")
				}
				terminal = true
			case "expiry":
				select {
				case <-lease.ctx.Done():
				case <-time.After(2 * time.Second):
					t.Fatal("original deadline not observed")
				}
				if r.check(context.Background()) == nil {
					t.Fatal("expired request binding lived")
				}
				terminal = true
			case "shutdown":
				if p.Close() != nil {
					t.Fatal("admission shutdown")
				}
				if r.check(context.Background()) == nil {
					t.Fatal("shutdown binding lived")
				}
				terminal = true
			case "explicit-close":
				if r.Close() != nil || r.Close() != nil || r.check(context.Background()) == nil {
					t.Fatal("original binding close")
				}
				terminal = true
			case "terminal-reply":
				if a.refuse("denied") != nil {
					t.Fatal("terminal refusal")
				}
				if r.check(context.Background()) == nil {
					t.Fatal("replied binding lived")
				}
				terminal = true
			case "concurrent-check":
				var wg sync.WaitGroup
				results := make(chan error, 32)
				for range 32 {
					wg.Go(func() { results <- r.check(context.Background()) })
				}
				wg.Wait()
				close(results)
				for err := range results {
					if err != nil {
						t.Fatal("original concurrent check")
					}
				}
			case "nil-context", "cancelled-context":
				var ctx context.Context
				expected := ErrConfig
				if name == "cancelled-context" {
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(context.Background())
					cancel()
					expected = context.Canceled
				}
				if req, actor, err := r.snapshot(ctx); req != (propertyRequest{}) || actor != (actionIdentity{}) || !errors.Is(err, expected) {
					t.Fatal("invalid context projected metadata")
				}
				terminal = true
			}
			if !terminal {
				if err := a.refuse("denied"); err != nil {
					t.Fatal("fixed refusal")
				}
			}
			if err := r.Close(); err != nil {
				t.Fatal("binding retirement")
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
	fmt.Println("PROPERTY_BINDING_RESULT " + strings.Join(fields, " "))
}
func TestNativePropertyBindingChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_PROPERTY_BINDING_CHILD")
	if mode == "" {
		t.Skip("reexec only")
	}
	var c Config
	if json.Unmarshal([]byte(os.Getenv("HORVIUM_HELPER_CONFIG")), &c) != nil {
		t.Fatal("fixture config")
	}
	raw, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: c.Socket, Net: "unix"})
	if err != nil {
		t.Fatal("fixture dial")
	}
	defer raw.Close()
	peer, err := connectPropertyPeer(context.Background(), raw, c.ManagementUID, "binding", strings.Repeat("a", 64), 3*time.Second)
	if err != nil {
		t.Fatal("actual child kernel negotiation")
	}
	defer peer.Close()
	if mode == "original-submit" {
		err = WritePropertySubmission(raw, "binding", "100", strings.Repeat("a", 64), strings.Repeat("b", 64))
	} else {
		err = WritePropertyReview(raw, "binding", "100", propertyClientIntent())
	}
	if err != nil {
		t.Fatal("typed request")
	}
	fmt.Println("sent")
	if mode == "disconnect" || mode == "extra-frame" {
		if line, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil || line != "cut\n" {
			t.Fatal("cut signal")
		}
		if mode == "disconnect" {
			if raw.Close() != nil {
				t.Fatal("disconnect")
			}
		} else {
			if WritePropertyReview(raw, "binding", "100", propertyClientIntent()) != nil {
				t.Fatal("second frame")
			}
		}
		fmt.Println("cut")
		return
	}
	if mode == "original-submit" {
		_, _, err = ReadPropertySubmission(raw, "binding", "100", strings.Repeat("a", 64), strings.Repeat("b", 64))
	} else {
		_, err = ReadPropertyReview(raw, "binding", "100", propertyClientIntent())
	}
	switch mode {
	case "original-review", "original-submit", "snapshot-isolation", "copied-binding", "reconstructed-binding", "foreign-owner", "terminal-reply", "concurrent-check":
		var refusal *PropertyRefusalError
		if !errors.As(err, &refusal) || refusal.Code != "denied" {
			t.Fatal("root denial")
		}
	default:
		if err == nil {
			t.Fatal("invalid binding obtained reply")
		}
	}
	var b [1]byte
	if n, err := raw.Read(b[:]); n != 0 || err != io.EOF {
		t.Fatal("root descriptor remained live")
	}
}
