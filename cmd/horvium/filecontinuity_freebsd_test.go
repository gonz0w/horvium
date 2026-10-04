//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Standalone, loopback-only base sshd in a disposable guest. No system sshd
// configuration, authorized_keys, user, rc.d state or service is changed.
func independentSFTP(t *testing.T) func() {
	t.Helper()
	dir, e := os.MkdirTemp("/root", "horvium-sftp-")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	key := filepath.Join(dir, "client")
	hostkey := filepath.Join(dir, "host")
	for _, path := range []string{key, hostkey} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		e := exec.CommandContext(ctx, "/usr/bin/ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", path).Run()
		cancel()
		if e != nil {
			t.Fatal("disposable SFTP key generation")
		}
	}
	root := filepath.Join(dir, "root")
	if os.Mkdir(root, 0755) != nil {
		t.Fatal("SFTP chroot")
	}
	content := []byte("Horvium disposable independent file-service fixture\n")
	if os.WriteFile(filepath.Join(root, "fixture"), content, 0600) != nil {
		t.Fatal("SFTP fixture")
	}
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	_, port, _ := net.SplitHostPort(l.Addr().String())
	_ = l.Close()
	pub, e := os.ReadFile(hostkey + ".pub")
	if e != nil {
		t.Fatal("host public key")
	}
	fields := strings.Fields(string(pub))
	if len(fields) < 2 {
		t.Fatal("public key schema")
	}
	known := filepath.Join(dir, "known_hosts")
	if os.WriteFile(known, []byte("[127.0.0.1]:"+port+" "+fields[0]+" "+fields[1]+"\n"), 0600) != nil {
		t.Fatal("fixture host trust")
	}
	conf := []byte(fmt.Sprintf("ListenAddress 127.0.0.1\nPort %s\nHostKey %s\nPidFile %s\nAuthorizedKeysFile %s\nAllowUsers root\nPermitRootLogin prohibit-password\nPasswordAuthentication no\nKbdInteractiveAuthentication no\nUsePAM no\nStrictModes yes\nAllowTcpForwarding no\nAllowAgentForwarding no\nX11Forwarding no\nPermitTunnel no\nMaxStartups 2\nChrootDirectory %s\nForceCommand internal-sftp -R\nSubsystem sftp internal-sftp\nLogLevel QUIET\n", port, hostkey, filepath.Join(dir, "pid"), key+".pub", root))
	path := filepath.Join(dir, "sshd_config")
	if os.WriteFile(path, conf, 0600) != nil {
		t.Fatal("fixture daemon config")
	}
	daemon := startNativeProcess(t, "/usr/sbin/sshd", []string{"-D", "-e", "-f", path}, 0, 0)
	nativeWait(t, func() bool {
		c, e := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), time.Second)
		if e != nil {
			return false
		}
		_ = c.Close()
		return true
	})
	t.Cleanup(func() {
		if !daemon.waited {
			daemon.stop(t, syscall.SIGTERM, true)
		}
	})
	return func() {
		t.Helper()
		if daemon.cmd.Process.Signal(syscall.Signal(0)) != nil {
			t.Fatal("independent daemon stopped")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		output := filepath.Join(dir, "received")
		cmd := exec.CommandContext(ctx, "/usr/bin/sftp", "-q", "-F", "/dev/null", "-b", "-", "-P", port, "-i", key, "-o", "IdentitiesOnly=yes", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes", "-o", "UserKnownHostsFile="+known, "-o", "GlobalKnownHostsFile=/dev/null", "-o", "ConnectTimeout=5", "root@127.0.0.1")
		cmd.Stdin = strings.NewReader("get /fixture " + output + "\n")
		if cmd.Run() != nil {
			t.Fatal("independent SFTP transfer failed; details withheld")
		}
		if got, e := os.ReadFile(output); e != nil || !bytes.Equal(got, content) {
			t.Fatal("independent file contents changed")
		}
		if got, e := os.ReadFile(path); e != nil || !bytes.Equal(got, conf) {
			t.Fatal("independent daemon config changed")
		}
	}
}
