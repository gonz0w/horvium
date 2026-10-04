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
	"os/user"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/host/freebsd"
	"github.com/gonz0w/horvium/internal/inventory"
)

func nativeOptIn(t *testing.T) {
	t.Helper()
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("requires explicit disposable QEMU opt-in")
	}
	if os.Geteuid() != 0 {
		t.Fatal("native fixture parent requires root")
	}
}
func account(t *testing.T, name string) (uint32, uint32) {
	t.Helper()
	u, err := user.Lookup(name)
	if err != nil {
		t.Fatal("fixture account unavailable")
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	return uint32(uid), uint32(gid)
}

type nativeFixture struct {
	dir, binary      string
	config           Config
	uid, denied, gid uint32
}

func fixture(t *testing.T) nativeFixture {
	t.Helper()
	nativeOptIn(t)
	uid, gid := account(t, "nobody")
	denied, _ := account(t, "daemon")
	if uid == 0 || denied == 0 || denied == uid || gid == 0 {
		t.Fatal("unsafe fixture identities")
	}
	dir, err := os.MkdirTemp("/tmp", "horvium-helper-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error("socket fixture teardown failed")
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
	// Go's build temp directory is not traversable by the dropped UID. Copy
	// only this test executable into a root-owned, read/execute-only fixture.
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	binary := filepath.Join(dir, "test")
	dst, err := os.OpenFile(binary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		t.Fatal("test executable copy failed")
	}
	return nativeFixture{dir: dir, binary: binary, uid: uid, denied: denied, gid: gid, config: Config{Socket: filepath.Join(run, "s"), ManagementUID: uid, SocketGID: gid, Limits: DefaultLimits(), Timeout: 60 * time.Second, MaxClients: 2}}
}
func child(t *testing.T, f nativeFixture, mode string, uid uint32) *exec.Cmd {
	t.Helper()
	data, err := json.Marshal(f.config)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(f.binary, "-test.run=^TestNativeHelperChild$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_HELPER_CHILD=" + mode, "HORVIUM_HELPER_CONFIG=" + string(data), "HORVIUM_HELPER_UID=" + strconv.FormatUint(uint64(uid), 10)}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: f.gid, Groups: []uint32{f.gid}}}
	return cmd
}
func runChild(t *testing.T, f nativeFixture, mode string, uid uint32) {
	t.Helper()
	cmd := child(t, f, mode, uid)
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("dropped-UID %s test failed: %v", mode, err)
		}
	case <-time.After(75 * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
		t.Fatal("child exceeded bounded deadline")
	}
}
func startServer(t *testing.T, f nativeFixture, source Collector) (*Server, context.CancelFunc, <-chan error) {
	t.Helper()
	s, err := NewServer(f.config, source)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("helper shutdown: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("helper shutdown timed out")
		}
		if _, err := os.Lstat(f.config.Socket); !os.IsNotExist(err) {
			t.Error("owned socket not removed")
		}
	})
	deadline := time.Now().Add(3 * time.Second)
	for {
		if safeSocket(f.config) == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("helper socket not ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	return s, cancel, done
}
func TestNativeHelperKernelAuthorization(t *testing.T) {
	f := fixture(t)
	var calls atomic.Int32
	startServer(t, f, collectFunc(func(context.Context) inventory.Snapshot { calls.Add(1); return observation() }))
	runChild(t, f, "allowed", f.uid)
	if calls.Load() != 1 {
		t.Fatal("allowed UID did not collect")
	}
	runChild(t, f, "denied", f.denied)
	runChild(t, f, "forged", f.uid)
	if calls.Load() != 1 {
		t.Fatal("denied UID or forged request collected")
	}
	// Root is not an implicit management exemption.
	runChild(t, f, "denied", 0)
	if calls.Load() != 1 {
		t.Fatal("root request bypassed exact configured UID")
	}
}
func TestNativeHelperActualReader(t *testing.T) {
	f := fixture(t)
	source, err := freebsd.New(inventory.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	startServer(t, f, source)
	runChild(t, f, "native", f.uid)
}
func TestNativeHelperUnsafePathsAndOwnership(t *testing.T) {
	f := fixture(t)
	source := collectFunc(func(context.Context) inventory.Snapshot { return observation() })
	run := filepath.Dir(f.config.Socket)
	for _, mode := range []os.FileMode{0770, 0777, 0700, 0750 | os.ModeSetgid} {
		if err := os.Chmod(run, mode); err != nil {
			t.Fatal(err)
		}
		if _, err := NewServer(f.config, source); !errors.Is(err, ErrConfig) {
			t.Fatal("unsafe directory permissions accepted")
		}
	}
	_ = os.Chmod(run, 0750)
	_ = os.Chown(run, int(f.uid), int(f.gid))
	if _, err := NewServer(f.config, source); !errors.Is(err, ErrConfig) {
		t.Fatal("service-owned directory accepted")
	}
	_ = os.Chown(run, 0, int(f.gid))
	alias := filepath.Join(f.dir, "alias")
	if err := os.Symlink(run, alias); err != nil {
		t.Fatal(err)
	}
	aliasConfig := f.config
	aliasConfig.Socket = filepath.Join(alias, "s")
	if _, err := NewServer(aliasConfig, source); !errors.Is(err, ErrConfig) {
		t.Fatal("symlink directory accepted")
	}
	ancestor := filepath.Join(f.dir, "insecure")
	if err := os.Mkdir(ancestor, 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ancestor, 0777); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(ancestor, "run")
	if err := os.Mkdir(nested, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(nested, 0, int(f.gid)); err != nil {
		t.Fatal(err)
	}
	ancestorConfig := f.config
	ancestorConfig.Socket = filepath.Join(nested, "s")
	if _, err := NewServer(ancestorConfig, source); !errors.Is(err, ErrConfig) {
		t.Fatal("writable ancestor accepted")
	}
	// Refuse existing regular files, symlinks and sockets without removing them.
	if err := os.WriteFile(f.config.Socket, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := NewServer(f.config, source)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Serve(context.Background()); !errors.Is(err, ErrConfig) {
		t.Fatal("existing file adopted")
	}
	if data, err := os.ReadFile(f.config.Socket); err != nil || string(data) != "fixture" {
		t.Fatal("existing file changed")
	}
	_ = os.Remove(f.config.Socket)
	if err := os.Symlink(filepath.Join(f.dir, "absent"), f.config.Socket); err != nil {
		t.Fatal(err)
	}
	if err := s.Serve(context.Background()); !errors.Is(err, ErrConfig) {
		t.Fatal("symlink adopted")
	}
	if info, err := os.Lstat(f.config.Socket); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink removed")
	}
	_ = os.Remove(f.config.Socket)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: f.config.Socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := s.Serve(context.Background()); !errors.Is(err, ErrConfig) {
		t.Fatal("existing socket adopted")
	}
	if _, err := os.Lstat(f.config.Socket); err != nil {
		t.Fatal("existing socket removed")
	}
	if err := os.Chown(f.config.Socket, 0, int(f.gid)); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []os.FileMode{0666, 0600, 0660 | os.ModeSetgid} {
		if err := os.Chmod(f.config.Socket, mode); err != nil {
			t.Fatal(err)
		}
		if err := safeSocket(f.config); !errors.Is(err, ErrConfig) {
			t.Fatal("unsafe socket mode accepted")
		}
	}
	_ = os.Chmod(f.config.Socket, 0660)
	_ = os.Chown(f.config.Socket, int(f.uid), int(f.gid))
	if err := safeSocket(f.config); !errors.Is(err, ErrConfig) {
		t.Fatal("unprivileged socket ownership accepted")
	}
}
func TestNativeClientRejectsNonRootPeer(t *testing.T) {
	f := fixture(t)
	// An unprivileged listener cannot normally occupy the root-owned run
	// directory. For this adversarial fixture root explicitly moves its socket
	// there and repairs file ownership, leaving the kernel listener UID intact.
	fakeDir := filepath.Join(f.dir, "fake")
	if err := os.Mkdir(fakeDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(fakeDir, int(f.uid), int(f.gid)); err != nil {
		t.Fatal(err)
	}
	fakeSocket := filepath.Join(fakeDir, "s")
	fake := f
	fake.config.Socket = fakeSocket
	cmd := child(t, fake, "nonroot-server", f.uid)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Lstat(fakeSocket); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("nonroot fixture listener absent")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := os.Rename(fakeSocket, f.config.Socket); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(f.config.Socket, 0, int(f.gid)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(f.config.Socket, 0660); err != nil {
		t.Fatal(err)
	}
	runChild(t, f, "nonroot-denied", f.uid)
}

// Reexec is mandatory: syscall.Credential drops identity before connect/listen.
// No setuid call is made inside a multithreaded Go process, and no host user is
// added. Child output is intentionally withheld from exported test artifacts.
func TestNativeHelperChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_HELPER_CHILD")
	if mode == "" {
		t.Skip("reexec fixture only")
	}
	uid, err := strconv.ParseUint(os.Getenv("HORVIUM_HELPER_UID"), 10, 32)
	if err != nil || uint32(os.Getuid()) != uint32(uid) {
		t.Fatal("child identity not dropped")
	}
	var c Config
	if err := json.Unmarshal([]byte(os.Getenv("HORVIUM_HELPER_CONFIG")), &c); err != nil {
		t.Fatal(err)
	}
	if mode == "nonroot-server" {
		l, err := net.ListenUnix("unix", &net.UnixAddr{Name: c.Socket, Net: "unix"})
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
		u, err := l.AcceptUnix()
		if err != nil {
			t.Fatal(err)
		}
		defer u.Close()
		_ = u.SetDeadline(time.Now().Add(3 * time.Second))
		var b [1]byte
		if n, _ := u.Read(b[:]); n != 0 {
			t.Fatal("client transmitted before authenticating helper")
		}
		return
	}
	if mode == "idle" {
		conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: c.Socket, Net: "unix"})
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if serverUID, err := peerUID(conn); err != nil || serverUID != 0 {
			t.Fatal("idle fixture peer is not root")
		}
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		fmt.Fprintln(os.Stdout, "ready")
		var b [1]byte
		if n, err := conn.Read(b[:]); n != 0 || err != io.EOF {
			t.Fatal("idle peer did not expire cleanly")
		}
		return
	}
	if mode == "allowed" || mode == "native" || mode == "nonroot-denied" || mode == "busy" {
		client, err := NewClient(c)
		if err != nil {
			t.Fatal(err)
		}
		s, err := client.Collect(context.Background(), "native-test")
		if mode == "busy" {
			if err == nil {
				t.Fatal("admission cap bypassed")
			}
			return
		}
		if mode == "nonroot-denied" {
			if !errors.Is(err, ErrPermission) {
				t.Fatal("nonroot helper peer accepted")
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if mode == "native" {
			if !s.Complete() || s.Host.Data.Architecture == "" || s.Host.Data.PhysicalMemory == nil {
				t.Fatal("actual native inventory unavailable")
			}
		} else if s.Host.Status != inventory.Partial {
			t.Fatal("section failure erased")
		}
		return
	}
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: c.Socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if serverUID, err := peerUID(conn); err != nil || serverUID != 0 {
		t.Fatal("test listener is not root")
	}
	// A denied UID sends a valid request: rejection must come from kernel
	// identity, independently of the separate forged-field parser test.
	payload := `{"version":1,"request_id":"r","operation":"inventory"}`
	if mode == "forged" {
		payload = `{"version":1,"request_id":"r","operation":"inventory","uid":` + strconv.FormatUint(uint64(c.ManagementUID), 10) + `}`
	}
	_, _ = conn.Write(frame(payload))
	_ = conn.CloseWrite()
	var b [1]byte
	if n, _ := conn.Read(b[:]); n != 0 {
		t.Fatal("denied request received inventory")
	}
}

func TestNativeAdmittedClientsBounded(t *testing.T) {
	f := fixture(t)
	f.config.MaxClients = 1
	f.config.Timeout = 3 * time.Second
	var calls atomic.Int32
	s, err := NewServer(f.config, collectFunc(func(context.Context) inventory.Snapshot { calls.Add(1); return observation() }))
	if err != nil {
		t.Fatal(err)
	}
	seen := make(chan struct{}, 1)
	s.peer = func(c *net.UnixConn) (uint32, error) {
		uid, err := peerUID(c)
		select {
		case seen <- struct{}{}:
		default:
		}
		return uid, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Error("admission fixture did not stop")
		}
		if _, err := os.Lstat(f.config.Socket); !os.IsNotExist(err) {
			t.Error("admission fixture socket leaked")
		}
	})
	deadline := time.Now().Add(3 * time.Second)
	for {
		if safeSocket(f.config) == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("admission fixture not ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	idle := child(t, f, "idle", f.uid)
	stdout, err := idle.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := idle.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if idle.Process != nil {
			_ = idle.Process.Kill()
		}
		_ = idle.Wait()
	})
	ready := make(chan bool, 1)
	go func() {
		line, err := bufio.NewReader(stdout).ReadString('\n')
		ready <- err == nil && line == "ready\n"
	}()
	select {
	case ok := <-ready:
		if !ok {
			t.Fatal("idle child failed before connect")
		}
	case <-time.After(time.Second):
		t.Fatal("idle child did not connect")
	}
	select {
	case <-seen:
	case <-time.After(time.Second):
		t.Fatal("idle client was not admitted")
	}
	runChild(t, f, "busy", f.uid)
	if calls.Load() != 0 {
		t.Fatal("excess client invoked collection")
	}
	// The admitted idle client expires; a subsequent authorized request succeeds.
	if err := idle.Wait(); err != nil {
		t.Fatal("idle client timeout test failed")
	}
	runChild(t, f, "allowed", f.uid)
	if calls.Load() != 1 {
		t.Fatal("admission slot did not recover after deadline")
	}
}

func TestNativeShutdownPreservesReplacement(t *testing.T) {
	f := fixture(t)
	s, err := NewServer(f.config, collectFunc(func(context.Context) inventory.Snapshot { return observation() }))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx) }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if safeSocket(f.config) == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("helper not ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Only the root test parent can replace an entry in the protected directory.
	if err := os.Remove(f.config.Socket); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.config.Socket, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown timed out")
	}
	if data, err := os.ReadFile(f.config.Socket); err != nil || string(data) != "replacement" {
		t.Fatal("shutdown removed another entry")
	}
}
