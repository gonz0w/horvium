// SPDX-License-Identifier: BSD-3-Clause
package observation

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/helper"
	"github.com/gonz0w/horvium/internal/host/freebsd"
	"github.com/gonz0w/horvium/internal/inventory"
)

func TestNativeCollectorHelperRecovery(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("requires explicit disposable QEMU opt-in")
	}
	if os.Geteuid() != 0 {
		t.Fatal("fixture parent requires root")
	}
	u, err := user.Lookup("nobody")
	if err != nil {
		t.Fatal("fixture UID unavailable")
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil || uid == 0 {
		t.Fatal("unsafe fixture UID")
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil || gid == 0 {
		t.Fatal("unsafe fixture group")
	}
	dir, err := os.MkdirTemp("/tmp", "horvium-cache-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error("fixture cleanup failed")
		}
	})
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	run := filepath.Join(dir, "run")
	if err := os.Mkdir(run, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(run, 0, int(gid)); err != nil {
		t.Fatal(err)
	}
	cfg := helper.Config{Socket: filepath.Join(run, "s"), ManagementUID: uint32(uid), SocketGID: uint32(gid), Limits: helper.DefaultLimits(), Timeout: 60 * time.Second, MaxClients: 8}
	src, err := freebsd.New(inventory.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	server, err := helper.NewServer(cfg, src)
	if err != nil {
		t.Fatal(err)
	}
	var stop context.CancelFunc
	var serverDone chan error
	start := func() {
		t.Helper()
		ctx, cancel := context.WithCancel(context.Background())
		stop = cancel
		serverDone = make(chan error, 1)
		go func(done chan error) { done <- server.Serve(ctx) }(serverDone)
		deadline := time.Now().Add(3 * time.Second)
		for {
			info, err := os.Lstat(cfg.Socket)
			if err == nil && info.Mode().Perm() == 0660 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("fixture helper not ready")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	shutdown := func() {
		t.Helper()
		if stop == nil {
			return
		}
		stop()
		select {
		case err := <-serverDone:
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("fixture helper did not stop")
		}
		stop = nil
		if _, err := os.Lstat(cfg.Socket); !os.IsNotExist(err) {
			t.Fatal("fixture socket leaked")
		}
	}
	t.Cleanup(func() {
		if stop != nil {
			stop()
			select {
			case <-serverDone:
			case <-time.After(5 * time.Second):
				t.Error("fixture helper cleanup timed out")
			}
		}
	})
	start()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	from, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer from.Close()
	binary := filepath.Join(dir, "test")
	to, err := os.OpenFile(binary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(to, from)
	closeErr := to.Close()
	if copyErr != nil || closeErr != nil {
		t.Fatal("fixture executable copy failed")
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "-test.run=^TestNativeCollectorChild$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_COLLECTOR_CHILD=1", "HORVIUM_COLLECTOR_CONFIG=" + string(data)}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: []uint32{uint32(gid)}}}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); _ = stdin.Close(); _ = cmd.Wait() })
	lines := bufio.NewScanner(stdout)
	readLine := func(want string) {
		t.Helper()
		ready := make(chan bool, 1)
		go func() { ready <- lines.Scan() && lines.Text() == want }()
		select {
		case ok := <-ready:
			if !ok {
				t.Fatalf("child did not confirm %s", want)
			}
		case <-ctx.Done():
			t.Fatalf("child exceeded deadline at %s", want)
		}
	}
	readLine("ready")
	shutdown()
	if _, err := fmt.Fprintln(stdin, "outage"); err != nil {
		t.Fatal(err)
	}
	readLine("stale")
	start()
	if _, err := fmt.Fprintln(stdin, "recover"); err != nil {
		t.Fatal(err)
	}
	readLine("recovered")
	_ = stdin.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatal("unprivileged collector child failed")
	}
	shutdown()
}

func TestNativeCollectorChild(t *testing.T) {
	if os.Getenv("HORVIUM_COLLECTOR_CHILD") != "1" {
		t.Skip("reexec fixture only")
	}
	var c helper.Config
	if err := json.Unmarshal([]byte(os.Getenv("HORVIUM_COLLECTOR_CONFIG")), &c); err != nil {
		t.Fatal(err)
	}
	if os.Getuid() == 0 || uint32(os.Getuid()) != c.ManagementUID {
		t.Fatal("collector identity not dropped before connect")
	}
	client, err := helper.NewClient(c)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := New(quietConfig(), client)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- manager.Run(ctx) }()
	wait := func(predicate func() bool) {
		t.Helper()
		deadline := time.Now().Add(60 * time.Second)
		for !predicate() {
			if time.Now().After(deadline) {
				t.Fatal("native collector state deadline")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	wait(func() bool { return manager.State().CurrentID != "" && !manager.State().Collecting })
	first := manager.State().CurrentID
	overview, err := manager.Overview(ctx, first)
	if err != nil || !overview.Complete || overview.Host.PhysicalMemory == nil || overview.Host.Architecture != "amd64" {
		t.Fatal("real native collection unavailable")
	}
	if _, err := manager.Datasets(ctx, Query{Generation: first, Limit: 1}); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintln(os.Stdout, "ready")
	input := bufio.NewScanner(os.Stdin)
	if !input.Scan() || input.Text() != "outage" {
		t.Fatal("missing outage command")
	}
	if err := manager.Refresh(); err != nil {
		t.Fatal(err)
	}
	wait(func() bool { return !manager.State().Collecting && manager.State().LastCompleted.ErrorCode != "" })
	state := manager.State()
	if state.CurrentID != first || !state.Datasets.Stale {
		t.Fatal("helper outage erased or refreshed inventory")
	}
	if p, err := manager.Datasets(ctx, Query{Generation: first}); err != nil || !p.Freshness.Stale {
		t.Fatal("cached outage context unavailable")
	}
	fmt.Fprintln(os.Stdout, "stale")
	if !input.Scan() || input.Text() != "recover" {
		t.Fatal("missing recovery command")
	}
	if err := manager.Refresh(); err != nil {
		t.Fatal(err)
	}
	wait(func() bool { return !manager.State().Collecting && manager.State().CurrentID != first })
	if manager.State().LastCompleted.ErrorCode != "" || manager.State().Host.Stale {
		t.Fatal("helper recovery retained failure")
	}
	if _, err := manager.Datasets(ctx, Query{Generation: first}); err != nil {
		t.Fatal("previous generation not retained after recovery")
	}
	fmt.Fprintln(os.Stdout, "recovered")
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if manager.State().CurrentID != "" {
		t.Fatal("collector shutdown retained inventory")
	}
}
