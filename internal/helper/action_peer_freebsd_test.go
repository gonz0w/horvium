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

func actionChild(t *testing.T, f nativeFixture, mode string, uid uint32) *exec.Cmd {
	t.Helper()
	data, err := json.Marshal(f.config)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(f.binary, "-test.run=^TestNativeActionPeerChild$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_ACTION_PEER_CHILD=" + mode, "HORVIUM_HELPER_CONFIG=" + string(data)}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: f.gid, Groups: []uint32{f.gid}}}
	return cmd
}
func actionListener(t *testing.T, f nativeFixture) *net.UnixListener {
	t.Helper()
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: f.config.Socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	if os.Chown(f.config.Socket, 0, int(f.gid)) != nil || os.Chmod(f.config.Socket, 0660) != nil || safeSocket(f.config) != nil {
		t.Fatal("unsafe fixture socket")
	}
	if err := l.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	return l
}
func startActionChild(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
}
func waitActionChild(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal("native peer child failed", err)
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatal("native peer child deadline")
	}
}
func TestNativeActionPeer(t *testing.T) {
	if os.Getenv("HORVIUM_ACTION_PEER_VM") != "1" {
		t.Skip("requires disposable action-peer opt-in")
	}
	f := fixture(t)
	cases := []string{"allowed", "asserted-context", "wrong-uid", "root-denied", "forged-uid", "unknown-operation", "old-version", "truncated", "idle-deadline", "cancelled-before", "local-uid-denied", "live-cancellation", "live-deadline", "disconnect", "extra-frame", "closed", "nonroot-helper", "fixtures-teardown"}
	for _, name := range cases[:len(cases)-2] {
		t.Run(name, func(t *testing.T) {
			l := actionListener(t, f)
			uid, mode := f.uid, name
			if name == "wrong-uid" {
				uid = f.denied
				mode = "denied"
			}
			if name == "root-denied" {
				uid = 0
				mode = "denied"
			}
			if name == "cancelled-before" {
				mode = "denied"
			}
			if name == "idle-deadline" {
				mode = "denied"
			}
			hold := name == "allowed" || name == "asserted-context" || name == "live-cancellation" || name == "live-deadline" || name == "disconnect" || name == "extra-frame" || name == "closed"
			if hold && name != "asserted-context" {
				mode = "hold"
			}
			cmd := actionChild(t, f, mode, uid)
			var input io.WriteCloser
			var output io.ReadCloser
			var err error
			if hold {
				input, err = cmd.StdinPipe()
				if err != nil {
					t.Fatal(err)
				}
				defer input.Close()
				output, err = cmd.StdoutPipe()
				if err != nil {
					t.Fatal(err)
				}
				defer output.Close()
			}
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
				timeout = 200 * time.Millisecond
			}
			if name == "live-deadline" {
				timeout = time.Second
			}
			if name == "cancelled-before" {
				cancel()
			}
			started := time.Now()
			peer, err := acceptActionPeer(ctx, conn, f.uid, timeout)
			if !hold {
				if peer != nil || err == nil {
					t.Fatal("denied native peer negotiated")
				}
				if (name == "wrong-uid" || name == "root-denied" || name == "idle-deadline") && time.Since(started) > 2*time.Second {
					t.Fatal("denial waited past bounded deadline")
				}
				waitActionChild(t, cmd)
				return
			}
			if err != nil || peer == nil {
				t.Fatal("actual authorized UID failed negotiation", err)
			}
			defer peer.Close()
			if line, err := bufio.NewReader(output).ReadString('\n'); err != nil || line != "ready\n" {
				t.Fatal("child handshake incomplete", err)
			}
			scope := strings.Repeat("a", 64)
			if name == "asserted-context" {
				scope = strings.Repeat("f", 64)
			}
			identity, err := peer.identity()
			if err != nil || identity != (actionIdentity{f.uid, "native-action", scope}) {
				t.Fatal("kernel UID/context binding lost", err)
			}
			if name == "live-cancellation" {
				cancel()
				if peer.checkLive() == nil {
					t.Fatal("cancelled peer live")
				}
			}
			if name == "live-deadline" {
				select {
				case <-time.After(1200 * time.Millisecond):
				case <-ctx.Done():
				}
				if peer.checkLive() == nil {
					t.Fatal("expired peer live")
				}
			}
			if name == "closed" {
				if peer.Close() != nil || peer.Close() != nil || peer.checkLive() == nil {
					t.Fatal("closed peer live")
				}
			}
			command := "close\n"
			if name == "extra-frame" {
				command = "extra\n"
			}
			if _, err := io.WriteString(input, command); err != nil {
				t.Fatal(err)
			}
			waitActionChild(t, cmd)
			if name == "disconnect" || name == "extra-frame" {
				err := peer.checkLive()
				if err == nil {
					t.Fatal("disconnected or pipelined peer live")
				}
				if name == "extra-frame" && !errors.Is(err, ErrProtocol) {
					t.Fatal("unparsed extra bytes were not identified", err)
				}
			}
			if err := peer.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("nonroot-helper", func(t *testing.T) {
		fakeDir := filepath.Join(f.dir, "fake")
		if os.Mkdir(fakeDir, 0700) != nil || os.Chown(fakeDir, int(f.uid), int(f.gid)) != nil {
			t.Fatal("fake listener directory")
		}
		fake := f
		fake.config.Socket = filepath.Join(fakeDir, "s")
		cmd := child(t, fake, "nonroot-server", f.uid)
		startActionChild(t, cmd)
		deadline := time.Now().Add(3 * time.Second)
		for {
			if _, err := os.Lstat(fake.config.Socket); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("fake listener absent")
			}
			time.Sleep(10 * time.Millisecond)
		}
		if os.Rename(fake.config.Socket, f.config.Socket) != nil || os.Chown(f.config.Socket, 0, int(f.gid)) != nil || os.Chmod(f.config.Socket, 0660) != nil {
			t.Fatal("adversarial socket setup")
		}
		client := actionChild(t, f, "nonroot-helper", f.uid)
		startActionChild(t, client)
		waitActionChild(t, client)
		waitActionChild(t, cmd)
		if err := os.Remove(f.config.Socket); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	})
	t.Run("fixtures-teardown", func(t *testing.T) {
		if _, err := os.Lstat(f.config.Socket); !os.IsNotExist(err) {
			t.Fatal("owned socket retained")
		}
		if err := os.RemoveAll(f.dir); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(f.dir); !os.IsNotExist(err) {
			t.Fatal("owned fixture retained")
		}
	})
	if t.Failed() {
		return
	}
	fields := make([]string, len(cases))
	for i, name := range cases {
		fields[i] = name + "=pass"
	}
	fmt.Println("ACTION_PEER_RESULT " + strings.Join(fields, " "))
}

func TestNativeActionPeerChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_ACTION_PEER_CHILD")
	if mode == "" {
		t.Skip("reexec fixture only")
	}
	var c Config
	if json.Unmarshal([]byte(os.Getenv("HORVIUM_HELPER_CONFIG")), &c) != nil {
		t.Fatal("child configuration")
	}
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: c.Socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if mode == "hold" || mode == "asserted-context" || mode == "local-uid-denied" || mode == "nonroot-helper" {
		uid := c.ManagementUID
		scope := strings.Repeat("a", 64)
		if mode == "asserted-context" {
			scope = strings.Repeat("f", 64)
		}
		if mode == "local-uid-denied" {
			uid++
		}
		peer, err := connectActionPeer(context.Background(), conn, uid, "native-action", scope, 3*time.Second)
		if mode == "local-uid-denied" || mode == "nonroot-helper" {
			if err == nil || peer != nil {
				t.Fatal("client identity bypass")
			}
			return
		}
		if err != nil || peer == nil {
			t.Fatal("root peer handshake", err)
		}
		defer peer.Close()
		if identity, err := peer.identity(); err != nil || identity != (actionIdentity{c.ManagementUID, "native-action", scope}) {
			t.Fatal("client binding", err)
		}
		fmt.Println("ready")
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if command == "extra\n" {
			if _, err := conn.Write(frame(`{"version":2,"operation":"destroy"}`)); err != nil {
				t.Fatal(err)
			}
		} else if command != "close\n" {
			t.Fatal("unknown fixture command")
		}
		return
	}
	if mode == "truncated" {
		_, _ = conn.Write([]byte{0, 0, 0, 64, '{'})
		return
	}
	if mode != "denied" {
		payload := `{"version":2,"request_id":"native-action","operation":"negotiate","session_context":"` + strings.Repeat("a", 64) + `"}`
		if mode == "forged-uid" {
			payload = strings.Replace(payload, `{"version"`, `{"uid":0,"version"`, 1)
		}
		if mode == "unknown-operation" {
			payload = strings.Replace(payload, "negotiate", "destroy", 1)
		}
		if mode == "old-version" {
			payload = strings.Replace(payload, `"version":2`, `"version":1`, 1)
		}
		if _, err := conn.Write(frame(payload)); err != nil {
			t.Fatal(err)
		}
	}
	var b [1]byte
	if n, err := conn.Read(b[:]); n != 0 || err != io.EOF {
		t.Fatal("denied child received bytes or stalled", err)
	}
}
