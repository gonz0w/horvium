// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gonz0w/horvium/internal/storage"
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

func propertyChild(t *testing.T, f nativeFixture, mode string, uid uint32) *exec.Cmd {
	t.Helper()
	data, err := json.Marshal(f.config)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(f.binary, "-test.run=^TestNativePropertyExchangeChild$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_PROPERTY_EXCHANGE_CHILD=" + mode, "HORVIUM_HELPER_CONFIG=" + string(data)}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: f.gid, Groups: []uint32{f.gid}}}
	return cmd
}
func propertyNativeIntent() storage.Intent {
	return storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "1099511627776"}
}
func TestNativePropertyExchange(t *testing.T) {
	if os.Getenv("HORVIUM_ACTION_PEER_VM") != "1" {
		t.Skip("requires disposable peer opt-in")
	}
	f := fixture(t)
	cases := []string{"review", "submit", "duplicate", "refused", "queued-request", "second-operation", "second-receive", "second-reply", "wrong-uid", "root-denied", "local-uid", "malformed-request", "wrong-request-id", "request-extra", "request-cut", "request-oversize", "idle-deadline", "cancellation", "explicit-close", "reply-extra", "reply-missing-eof", "reply-wrong-selection", "reply-wrong-id", "overlap", "nonroot-helper", "fixtures-teardown"}
	for _, name := range cases[:len(cases)-2] {
		t.Run(name, func(t *testing.T) {
			l := actionListener(t, f)
			uid := f.uid
			mode := name
			if name == "wrong-uid" {
				uid = f.denied
				mode = "denied"
			}
			if name == "root-denied" {
				uid = 0
				mode = "denied"
			}
			cmd := propertyChild(t, f, mode, uid)
			out, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			defer out.Close()
			startActionChild(t, cmd)
			conn, err := l.AcceptUnix()
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			timeout := 3 * time.Second
			if name == "idle-deadline" {
				timeout = 250 * time.Millisecond
			}
			x, err := acceptPropertyPeer(ctx, conn, f.uid, timeout)
			if name == "wrong-uid" || name == "root-denied" || name == "local-uid" {
				if err == nil || x != nil {
					t.Fatal("unauthorized peer negotiated")
				}
				waitActionChild(t, cmd)
				return
			}
			if err != nil || x == nil {
				t.Fatal("actual UID failed property negotiation", err)
			}
			defer x.Close()
			if line, err := bufio.NewReader(out).ReadString('\n'); err != nil || line != "ready\n" {
				t.Fatal("child not ready", err)
			}
			if name == "queued-request" {
				time.Sleep(20 * time.Millisecond)
			}
			if name == "cancellation" {
				cancel()
			}
			if name == "explicit-close" {
				if x.Close() != nil || x.Close() != nil {
					t.Fatal("close not idempotent")
				}
			}
			if name == "overlap" {
				done := make(chan error, 1)
				go func() { _, _, err := x.receive(); done <- err }()
				deadline := time.Now().Add(time.Second)
				for {
					xLocked := x.mu.TryLock()
					if !xLocked {
						break
					}
					x.mu.Unlock()
					if time.Now().After(deadline) {
						t.Fatal("receive never acquired ownership")
					}
					time.Sleep(time.Millisecond)
				}
				if _, _, err := x.receive(); err == nil {
					t.Fatal("overlapping receive accepted")
				}
				select {
				case err := <-done:
					if err == nil {
						t.Fatal("first overlapping receive succeeded")
					}
				case <-time.After(time.Second):
					t.Fatal("overlap did not interrupt read")
				}
				waitActionChild(t, cmd)
				return
			}
			req, actor, err := x.receive()
			denied := name == "malformed-request" || name == "wrong-request-id" || name == "request-extra" || name == "request-cut" || name == "request-oversize" || name == "idle-deadline" || name == "cancellation" || name == "explicit-close"
			if denied {
				if err == nil || req != (propertyRequest{}) || actor != (actionIdentity{}) {
					t.Fatal("invalid request published", err)
				}
				if name == "request-extra" && !errors.Is(err, ErrProtocol) {
					t.Fatal("extra frame not recognized", err)
				}
				waitActionChild(t, cmd)
				return
			}
			if err != nil || actor != (actionIdentity{f.uid, "native-property", strings.Repeat("a", 64)}) || req.ID != "native-property" || req.PoolGUID != "100" {
				t.Fatal("original kernel/request binding lost", err)
			}
			if name == "second-receive" {
				if req2, a, err := x.receive(); err == nil || req2 != (propertyRequest{}) || a != (actionIdentity{}) {
					t.Fatal("second receive accepted")
				}
				waitActionChild(t, cmd)
				return
			}
			if strings.HasPrefix(name, "reply-") {
				r := replyReview(t)
				id := "native-property"
				if name == "reply-wrong-selection" {
					r.Intent.Value = "4096"
				}
				if name == "reply-wrong-id" {
					id = "other"
				}
				if err := writePropertyReview(conn, id, r); err != nil {
					t.Fatal(err)
				}
				if name == "reply-extra" {
					if _, err := conn.Write([]byte("extra")); err != nil {
						t.Fatal(err)
					}
				}
				if name == "reply-missing-eof" {
					var b [1]byte
					if n, err := conn.Read(b[:]); n != 0 || err != io.EOF {
						t.Fatal("client did not bound missing EOF", err)
					}
				} else {
					_ = conn.Close()
				}
				waitActionChild(t, cmd)
				return
			}
			switch name {
			case "submit", "duplicate":
				if req.Operation != "property-submit" || req.PlanID != strings.Repeat("a", 64) || req.SubmissionID != strings.Repeat("b", 64) {
					t.Fatal("wrong typed submission")
				}
				err = x.replySubmission(replyJob(), name == "duplicate")
			case "refused":
				err = x.refuse("uncertain")
			default:
				if req.Operation != "property-review" || req.Intent != propertyNativeIntent() {
					t.Fatal("wrong typed intent")
				}
				err = x.replyReview(replyReview(t))
			}
			if err != nil {
				t.Fatal("terminal reply failed", err)
			}
			if name == "second-reply" && x.refuse("denied") == nil {
				t.Fatal("second reply accepted")
			}
			waitActionChild(t, cmd)
		})
	}
	t.Run("nonroot-helper", func(t *testing.T) {
		dir := filepath.Join(f.dir, "fake")
		if os.Mkdir(dir, 0700) != nil || os.Chown(dir, int(f.uid), int(f.gid)) != nil {
			t.Fatal("fake directory")
		}
		fake := f
		fake.config.Socket = filepath.Join(dir, "s")
		cmd := child(t, fake, "nonroot-server", f.uid)
		startActionChild(t, cmd)
		deadline := time.Now().Add(3 * time.Second)
		for {
			if _, err := os.Lstat(fake.config.Socket); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("fake socket missing")
			}
			time.Sleep(10 * time.Millisecond)
		}
		if os.Rename(fake.config.Socket, f.config.Socket) != nil || os.Chown(f.config.Socket, 0, int(f.gid)) != nil || os.Chmod(f.config.Socket, 0660) != nil {
			t.Fatal("repaired file ownership")
		}
		c := propertyChild(t, f, "nonroot-helper", f.uid)
		startActionChild(t, c)
		waitActionChild(t, c)
		waitActionChild(t, cmd)
		if err := os.Remove(f.config.Socket); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	})
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
	for n, name := range cases {
		fields[n] = name + "=pass"
	}
	fmt.Println("PROPERTY_EXCHANGE_RESULT " + strings.Join(fields, " "))
}
func TestNativePropertyExchangeChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_PROPERTY_EXCHANGE_CHILD")
	if mode == "" {
		t.Skip("reexec fixture only")
	}
	var c Config
	if json.Unmarshal([]byte(os.Getenv("HORVIUM_HELPER_CONFIG")), &c) != nil {
		t.Fatal("config")
	}
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: c.Socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if mode == "denied" {
		var b [1]byte
		if n, err := conn.Read(b[:]); n != 0 || err != io.EOF {
			t.Fatal("UID denial read bytes", err)
		}
		return
	}
	uid := c.ManagementUID
	if mode == "local-uid" {
		uid++
	}
	timeout := 3 * time.Second
	if mode == "reply-missing-eof" {
		timeout = 250 * time.Millisecond
	}
	x, err := connectPropertyPeer(context.Background(), conn, uid, "native-property", strings.Repeat("a", 64), timeout)
	if mode == "local-uid" || mode == "nonroot-helper" {
		if err == nil || x != nil {
			t.Fatal("client identity bypass")
		}
		return
	}
	if err != nil || x == nil {
		t.Fatal("original root negotiation", err)
	}
	defer x.Close()
	fmt.Println("ready")
	if mode == "idle-deadline" || mode == "cancellation" || mode == "explicit-close" || mode == "overlap" {
		var b [1]byte
		if n, err := conn.Read(b[:]); n != 0 || err != io.EOF {
			t.Fatal("idle exchange did not close", err)
		}
		return
	}
	if mode == "malformed-request" || mode == "wrong-request-id" || mode == "request-extra" || mode == "request-cut" || mode == "request-oversize" {
		var b bytes.Buffer
		if WritePropertyReview(&b, "native-property", "100", propertyNativeIntent()) != nil {
			t.Fatal("fixture intent")
		}
		data := b.Bytes()
		if mode == "malformed-request" {
			data = frame(strings.Replace(string(data[4:]), `"version":2`, `"version":2,"uid":0`, 1))
		}
		if mode == "wrong-request-id" {
			data = frame(strings.Replace(string(data[4:]), "native-property", "other", 1))
		}
		if mode == "request-extra" {
			data = append(data, frame(`{"extra":true}`)...)
		}
		if mode == "request-cut" {
			data = data[:5]
		}
		if mode == "request-oversize" {
			data = make([]byte, 4)
			binary.BigEndian.PutUint32(data, 1025)
		}
		if _, err := conn.Write(data); err != nil {
			t.Fatal(err)
		}
		if mode == "request-cut" {
			_ = conn.Close()
			return
		}
		var q [1]byte
		if n, err := conn.Read(q[:]); n != 0 || err != io.EOF {
			t.Fatal("malformed request not closed", err)
		}
		return
	}
	if mode == "submit" || mode == "duplicate" {
		j, dup, err := x.submit("100", strings.Repeat("a", 64), strings.Repeat("b", 64))
		if err != nil || dup != (mode == "duplicate") || j.ID != strings.Repeat("c", 64) {
			t.Fatal("admission metadata lost", err)
		}
		return
	}
	r, err := x.review("100", propertyNativeIntent())
	if mode == "refused" {
		var refused *PropertyRefusalError
		if !errors.As(err, &refused) || refused.Code != "uncertain" {
			t.Fatal("refusal lost", err)
		}
		return
	}
	if strings.HasPrefix(mode, "reply-") || mode == "second-receive" {
		if err == nil || r.ID != "" {
			t.Fatal("bad terminal reply accepted")
		}
		return
	}
	if err != nil || r.ID != strings.Repeat("a", 64) {
		t.Fatal("review metadata lost", err)
	}
	if mode == "second-operation" {
		if r, err := x.review("100", propertyNativeIntent()); err == nil || r.ID != "" {
			t.Fatal("second operation accepted")
		}
	}
}
