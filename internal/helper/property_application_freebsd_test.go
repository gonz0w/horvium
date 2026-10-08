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

func applicationPropertyChild(t *testing.T, f nativeFixture, mode string) *exec.Cmd {
	t.Helper()
	data, err := json.Marshal(f.config)
	if err != nil {
		t.Fatal("fixture configuration")
	}
	cmd := exec.Command(f.binary, "-test.run=^TestNativeApplicationPropertyChild$", "-test.count=1", "-test.timeout=10s")
	cmd.Env = []string{"HORVIUM_APPLICATION_PROPERTY_CHILD=" + mode, "HORVIUM_HELPER_CONFIG=" + string(data)}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: f.uid, Gid: f.gid, Groups: []uint32{f.gid}}}
	return cmd
}
func TestNativeApplicationPropertyClient(t *testing.T) {
	if os.Getenv("HORVIUM_ACTION_PEER_VM") != "1" {
		t.Skip("requires disposable peer opt-in")
	}
	f := fixture(t)
	f.config.Timeout = 3 * time.Second
	cases := []string{"review-denied", "submit-denied", "logout-before-dial", "close-before-dial", "decoded-authority", "invalid-selection", "cancelled-before-dial", "logout-after-hello", "logout-after-request", "close-after-request", "fixtures-teardown"}
	for _, name := range cases[:len(cases)-1] {
		t.Run(name, func(t *testing.T) {
			cmd := applicationPropertyChild(t, f, name)
			if name == "review-denied" || name == "submit-denied" {
				r := startPropertyListener(t, f)
				startActionChild(t, cmd)
				waitActionChild(t, cmd)
				r.stop(t)
				if _, err := os.Lstat(f.config.Socket); !os.IsNotExist(err) {
					t.Fatal("listener socket retained")
				}
				return
			}
			l := actionListener(t, f)
			if !strings.Contains(name, "after-") {
				startActionChild(t, cmd)
				waitActionChild(t, cmd)
				if err := l.SetDeadline(time.Now().Add(150 * time.Millisecond)); err != nil {
					t.Fatal("backlog deadline")
				}
				raw, err := l.AcceptUnix()
				if raw != nil {
					_ = raw.Close()
					t.Fatal("revoked/invalid application dialed")
				}
				var timeout net.Error
				if !errors.As(err, &timeout) || !timeout.Timeout() {
					t.Fatal("empty accept backlog not observed")
				}
				return
			}
			input, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal("revoke input")
			}
			defer input.Close()
			output, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal("revoke output")
			}
			defer output.Close()
			startActionChild(t, cmd)
			raw, err := l.AcceptUnix()
			if err != nil {
				t.Fatal("application connection")
			}
			defer raw.Close()
			if err = raw.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
				t.Fatal("connection deadline")
			}
			if uid, err := peerUID(raw); err != nil || uid != f.uid {
				t.Fatal("actual application peer UID")
			}
			revoke := func() {
				if _, err := io.WriteString(input, "revoke\n"); err != nil {
					t.Fatal("revoke signal")
				}
				done := make(chan string, 1)
				go func() { line, _ := bufio.NewReader(output).ReadString('\n'); done <- line }()
				select {
				case line := <-done:
					if line != "revoked\n" {
						t.Fatal("original store revocation acknowledgement")
					}
				case <-time.After(2 * time.Second):
					t.Fatal("revocation acknowledgement deadline")
				}
			}
			if name == "logout-after-hello" {
				hello, err := readActionNegotiation(raw)
				if err != nil || hello.ID != "application" {
					t.Fatal("application hello")
				}
				revoke()
				if err := writeActionReady(raw, hello); err != nil {
					t.Fatal("fixture ready")
				}
				var b [1]byte
				if n, err := raw.Read(b[:]); n != 0 || err != io.EOF {
					t.Fatal("revoked application sent request")
				}
				waitActionChild(t, cmd)
				return
			}
			admission := connectionAdmission(t, context.Background(), 1, 3*time.Second)
			lease, err := admission.admit(raw)
			if err != nil {
				t.Fatal("original admission")
			}
			peer, err := lease.accept(f.uid)
			if err != nil {
				t.Fatal("kernel negotiation")
			}
			defer peer.Close()
			req, actor, err := peer.receive()
			if err != nil || req.ID != "application" || req.PoolGUID != "100" || actor.uid != f.uid || actor.requestID != req.ID || !validApplicationContext(actor.applicationContext) {
				t.Fatal("typed application request")
			}
			// Context is untrusted correlation at root; no backend or success reply.
			revoke()
			if err := peer.refuse("denied"); err != nil {
				t.Fatal("root refusal")
			}
			admittedEmpty(t, admission, lease)
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
	fmt.Println("APPLICATION_PROPERTY_RESULT " + strings.Join(fields, " "))
}
