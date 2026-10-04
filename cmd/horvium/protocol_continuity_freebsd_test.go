// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// These file daemons are test-owned, started explicitly only in a disposable
// guest. No Horvium code starts, stops, reloads or configures them.
func continuityCommand(t *testing.T, path string, args ...string) {
	t.Helper()
	ctx, stop := context.WithTimeout(context.Background(), 20*time.Second)
	defer stop()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin:/usr/local/bin:/usr/local/sbin"}
	if cmd.Run() != nil {
		t.Fatal("disposable protocol fixture command failed; details withheld")
	}
}
func continuityPort(t *testing.T) string {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal("fixture port")
	}
	_, port, _ := net.SplitHostPort(l.Addr().String())
	_ = l.Close()
	return port
}
func continuityReady(t *testing.T, port string, p *nativeProcess) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		c, e := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), time.Second)
		if e == nil {
			_ = c.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	if f, ok := p.cmd.Stderr.(*os.File); ok {
		b, _ := os.ReadFile(f.Name())
		for _, fixed := range []string{"illegal option", "NFS server is not available", "already running", "getting export list", "getting mount list", "here we go", "bindresvport_sa", "usage:", "Cannot get port number", "Cannot register", "cannot register", "failed"} {
			if bytes.Contains(b, []byte(fixed)) {
				t.Log("DAEMON_DIAGNOSTIC " + fixed)
			}
		}
	}
	t.Fatal("owned fixture listener did not become ready; raw daemon output withheld")
}

// Samba may signal its process group during shutdown. Isolate each owned
// fixture daemon from the SSH runner/test controller before it executes.
func startContinuityDaemon(t *testing.T, binary string, args []string) *nativeProcess {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin:/usr/local/bin:/usr/local/sbin"}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	log, err := os.CreateTemp(t.TempDir(), "daemon-log-")
	if err != nil {
		t.Fatal("owned daemon diagnostic file")
	}
	t.Cleanup(func() { _ = log.Close() })
	cmd.Stdout, cmd.Stderr = log, log
	p := &nativeProcess{cmd: cmd, done: make(chan error, 1)}
	if cmd.Start() != nil {
		t.Fatal("isolated fixture daemon start")
	}
	go func() { p.done <- cmd.Wait() }()
	t.Cleanup(func() {
		if !p.waited {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			select {
			case <-p.done:
			case <-time.After(5 * time.Second):
				t.Error("isolated fixture daemon cleanup timed out")
			}
			p.waited = true
		}
	})
	group, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil || group != cmd.Process.Pid {
		t.Fatal("fixture daemon is not in its own process group")
	}
	return p
}

func stopContinuityDaemon(t *testing.T, p *nativeProcess, signal syscall.Signal) {
	t.Helper()
	p.stop(t, signal, false)
	status, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
	// Base rpcbind intentionally exits 2 after handling a termination signal.
	// Admit only that exact owned executable/requested SIGTERM contract.
	rpcTermination := p.cmd.Path == "/usr/sbin/rpcbind" && signal == syscall.SIGTERM && status.Exited() && status.ExitStatus() == 2
	if !ok || !(rpcTermination || status.Exited() && status.ExitStatus() == 0 || status.Signaled() && status.Signal() == signal) {
		t.Fatal("fixture daemon did not exit with the requested signal or success")
	}
	// Daemon shutdown can also signal workers. Any remaining member is still
	// inside the separate, verified test-owned process group.
	_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
	nativeWait(t, func() bool { return syscall.Kill(-p.cmd.Process.Pid, 0) == syscall.ESRCH })
}

func independentSMB(t *testing.T, dir, root string, content []byte) func() {
	t.Helper()
	for _, leaf := range []string{"lock", "state", "cache", "private", "pid", "log"} {
		if os.Mkdir(filepath.Join(dir, leaf), 0700) != nil {
			t.Fatal("SMB fixture directory")
		}
	}
	port := continuityPort(t)
	conf := []byte(fmt.Sprintf("[global]\nserver role = standalone server\ninterfaces = 127.0.0.1\nbind interfaces only = yes\nsmb ports = %s\ndisable netbios = yes\nserver min protocol = SMB2_02\nmap to guest = Bad User\nguest account = nobody\nload printers = no\nprinting = bsd\nprintcap name = /dev/null\nlock directory = %s/lock\nstate directory = %s/state\ncache directory = %s/cache\nprivate dir = %s/private\npid directory = %s/pid\nlog file = %s/log/log.smbd\nlog level = 0\n[fixture]\npath = %s\nguest ok = yes\nguest only = yes\nread only = yes\nbrowseable = no\n", port, dir, dir, dir, dir, dir, dir, root))
	path := filepath.Join(dir, "smb.conf")
	if os.WriteFile(path, conf, 0600) != nil {
		t.Fatal("SMB fixture config")
	}
	continuityCommand(t, "/usr/local/bin/testparm", "-s", path)
	daemon := startContinuityDaemon(t, "/usr/local/sbin/smbd", []string{"--foreground", "--no-process-group", "--configfile=" + path})
	continuityReady(t, port, daemon)
	t.Cleanup(func() {
		if !daemon.waited {
			stopContinuityDaemon(t, daemon, syscall.SIGTERM)
		}
	})
	return func() {
		t.Helper()
		if daemon.cmd.Process.Signal(syscall.Signal(0)) != nil {
			t.Fatal("independent SMB daemon stopped")
		}
		output := filepath.Join(dir, "received")
		_ = os.Remove(output)
		// The SMB command language contains only fixed/generated fixture paths,
		// never an operator path, credential or inventory name.
		continuityCommand(t, "/usr/local/bin/smbclient", "//127.0.0.1/fixture", "-I", "127.0.0.1", "-p", port, "-N", "-U", "horvium-fixture-guest", "-m", "SMB3", "-s", path, "-c", "get fixture "+output)
		got, e := os.ReadFile(output)
		if e != nil || !bytes.Equal(got, content) {
			t.Fatal("independent SMB content mismatch")
		}
		got, e = os.ReadFile(path)
		if e != nil || !bytes.Equal(got, conf) {
			t.Fatal("SMB configuration changed")
		}
	}
}
func independentNFS(t *testing.T, dir, root string, content []byte) func() {
	t.Helper()
	// Refuse a guest already hosting these daemons. Kernel NFS exports and the
	// standard mountd PID state are guest-global, so fixtures must be exclusive.
	for _, name := range []string{"rpcbind", "mountd", "nfsd"} {
		cmd := exec.Command("/usr/bin/pgrep", "-x", name)
		err := cmd.Run()
		exit, ok := err.(*exec.ExitError)
		if !ok || exit.ExitCode() != 1 {
			t.Fatal("NFS fixture requires unused disposable daemon state")
		}
	}
	l, e := net.Listen("tcp", "127.0.0.1:2049")
	if e != nil {
		t.Fatal("NFS fixture port already occupied")
	}
	_ = l.Close()
	exports := []byte(root + " -ro -mapall=nobody 127.0.0.1\n")
	path := filepath.Join(dir, "exports")
	if os.WriteFile(path, exports, 0600) != nil {
		t.Fatal("NFS fixture exports")
	}
	// mountd automatically adds 127.0.0.1 and ::1 to explicit bind hosts.
	// Avoid duplicate fixed-port binds by providing a separate owned loopback
	// alias. Never bind a fixture export daemon to the guest network interface.
	interfaces := readerFixtureCommand(t, "/sbin/ifconfig", "lo0")
	if bytes.Contains(interfaces, []byte("inet 127.0.0.2 ")) {
		t.Fatal("owned loopback alias already exists")
	}
	continuityCommand(t, "/sbin/ifconfig", "lo0", "inet", "127.0.0.2/32", "alias")
	t.Cleanup(func() { continuityCommand(t, "/sbin/ifconfig", "lo0", "inet", "127.0.0.2", "-alias") })
	rpc := startContinuityDaemon(t, "/usr/sbin/rpcbind", []string{"-d", "-h", "127.0.0.1"})
	t.Log("NFS_PHASE rpcbind_start")
	continuityReady(t, "111", rpc)
	t.Log("NFS_PHASE rpcbind_ready")
	port := continuityPort(t)
	mountd := startContinuityDaemon(t, "/usr/sbin/mountd", []string{"-d", "-h", "127.0.0.2", "-p", port, path})
	t.Log("NFS_PHASE mountd_start")
	continuityReady(t, port, mountd)
	t.Log("NFS_PHASE mountd_ready")
	nfsd := startContinuityDaemon(t, "/usr/sbin/nfsd", []string{"--debug", "-t", "-h", "127.0.0.1", "--minthreads", "2", "--maxthreads", "2"})
	t.Log("NFS_PHASE nfsd_start")
	continuityReady(t, "2049", nfsd)
	t.Log("NFS_PHASE nfsd_ready")
	mount := filepath.Join(dir, "client")
	if os.Mkdir(mount, 0700) != nil {
		t.Fatal("NFS client mountpoint")
	}
	mounted := false
	t.Cleanup(func() {
		if mounted {
			continuityCommand(t, "/sbin/umount", mount)
		}
		if !nfsd.waited {
			stopContinuityDaemon(t, nfsd, syscall.SIGUSR1)
		}
		if !mountd.waited {
			stopContinuityDaemon(t, mountd, syscall.SIGTERM)
		}
		if !rpc.waited {
			stopContinuityDaemon(t, rpc, syscall.SIGTERM)
		}
	})
	checkpoint := 0
	return func() {
		t.Helper()
		for _, daemon := range []*nativeProcess{rpc, mountd, nfsd} {
			if daemon.cmd.Process.Signal(syscall.Signal(0)) != nil {
				t.Fatal("independent NFS daemon stopped")
			}
		}
		// Fresh name + fresh client mount forces another server read instead of
		// accepting a cached client page as continuity evidence.
		checkpoint++
		name := "fixture-" + strconv.Itoa(checkpoint)
		if os.WriteFile(filepath.Join(root, name), content, 0644) != nil {
			t.Fatal("fresh NFS fixture file")
		}
		continuityCommand(t, "/sbin/mount_nfs", "-o", "nfsv3,tcp,ro,soft,timeo=10,retrans=1", "-R", "1", "127.0.0.1:"+root, mount)
		mounted = true
		got, e := os.ReadFile(filepath.Join(mount, name))
		if e != nil || !bytes.Equal(got, content) {
			t.Fatal("independent NFS content mismatch")
		}
		if os.WriteFile(filepath.Join(mount, "forbidden"), content, 0600) == nil {
			t.Fatal("read-only NFS fixture accepted write")
		}
		continuityCommand(t, "/sbin/umount", mount)
		mounted = false
		got, e = os.ReadFile(path)
		if e != nil || !bytes.Equal(got, exports) {
			t.Fatal("NFS exports changed")
		}
	}
}
func TestNativeIndependentSMBNFSContinuity(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_CONTINUITY_VM") != "1" {
		t.Skip("requires explicit disposable protocol guest")
	}
	if os.Geteuid() != 0 {
		t.Fatal("protocol fixture requires root")
	}
	for _, protocol := range []string{"SMB", "NFS"} {
		t.Run(protocol, func(t *testing.T) {
			// Preserve operator-facing service configuration in the guest, including
			// absent files. Fixture daemons may create their own ephemeral PID/kernel
			// state, but never edit these files or rc.d enablement.
			before := map[string][]byte{}
			for _, path := range []string{"/etc/rc.conf", "/etc/exports", "/usr/local/etc/smb4.conf"} {
				b, e := os.ReadFile(path)
				if e != nil && !os.IsNotExist(e) {
					t.Fatal("guest service config")
				}
				before[path] = b
			}
			owned := nativeStorageFixture(t)
			dir, e := os.MkdirTemp("/tmp", "horvium-protocol-")
			if e != nil {
				t.Fatal("protocol fixture directory")
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			if os.Chmod(dir, 0755) != nil {
				t.Fatal("protocol fixture traversal")
			}
			root := filepath.Join(dir, "export")
			continuityCommand(t, "/sbin/zfs", "create", "-o", "mountpoint="+root, owned.Pool+"/continuity")
			if os.Chmod(root, 0755) != nil {
				t.Fatal("fixture export mode")
			}
			content := []byte("Horvium disposable independent protocol fixture\n")
			if os.WriteFile(filepath.Join(root, "fixture"), content, 0644) != nil {
				t.Fatal("fixture content")
			}
			// Unmount the owned export before the general fixture destroys its pool.
			t.Cleanup(func() { continuityCommand(t, "/sbin/zfs", "unmount", owned.Pool+"/continuity") })
			var read func()
			if protocol == "SMB" {
				read = independentSMB(t, dir, root, content)
			} else {
				read = independentNFS(t, dir, root, content)
			}
			c, path, token := nativeRuntimeConfig(t)
			config, _ := os.ReadFile(path)
			digestPath := filepath.Join(c.StateDirectory, "token.sha256")
			digest, _ := os.ReadFile(digestPath)
			built, _ := filepath.Abs("../../.tools/horvium")
			binary := filepath.Join(filepath.Dir(path), "horvium")
			copyPerformanceExecutable(t, built, binary)
			startHelper := func() *nativeProcess {
				p := startNativeProcess(t, binary, []string{"helper", "-config", path}, 0, 0)
				nativeWait(t, func() bool { i, e := os.Stat(c.HelperSocket); return e == nil && i.Mode().Perm() == 0660 })
				return p
			}
			startManagement := func() *nativeProcess {
				return startNativeProcess(t, binary, []string{"serve", "-config", path}, c.ManagementUID, c.SocketGID)
			}
			helper := startHelper()
			management := startManagement()
			session := newPerformanceSession(t, c, token)
			count := len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
			session.ready(t, count, "")
			read()
			socket, e := os.Lstat(c.HelperSocket)
			if e != nil {
				t.Fatal("owned continuity socket")
			}
			helper.stop(t, syscall.SIGKILL, false)
			read()
			session.refresh(t)
			nativeWait(t, func() bool { return session.overview(t).Datasets.Stale })
			read()
			current, e := os.Lstat(c.HelperSocket)
			if e != nil || !os.SameFile(socket, current) || os.Remove(c.HelperSocket) != nil {
				t.Fatal("owned stale continuity socket")
			}
			helper = startHelper()
			session.refresh(t)
			nativeWait(t, func() bool { return !session.overview(t).Datasets.Stale })
			read()
			management.stop(t, syscall.SIGKILL, false)
			read()
			management = startManagement()
			nativeWait(t, func() bool {
				r, e := session.http.Get(session.base + "/api/overview")
				if e != nil {
					return false
				}
				defer r.Body.Close()
				return r.StatusCode == http.StatusUnauthorized
			})
			session = newPerformanceSession(t, c, token)
			session.ready(t, count, "")
			read()
			management.stop(t, syscall.SIGTERM, true)
			helper.stop(t, syscall.SIGTERM, true)
			read()
			configAfter, _ := os.ReadFile(path)
			digestAfter, _ := os.ReadFile(digestPath)
			if !bytes.Equal(config, configAfter) || !bytes.Equal(digest, digestAfter) {
				t.Fatal("management outage changed boot state")
			}
			for path, b := range before {
				after, e := os.ReadFile(path)
				if e != nil && !os.IsNotExist(e) || !bytes.Equal(b, after) {
					t.Fatal("independent system service configuration changed")
				}
			}
			t.Log("FILE_CONTINUITY protocol=" + protocol + " reads=7 helper_outage=pass management_restart=pass session_invalidation=pass config=unchanged")
		})
	}
}
