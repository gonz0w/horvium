//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bufio"
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A stopped, owned idle fixture allows an independent ps comparison without
// assuming that a running process's RSS or CPU stays constant between reads.
// This runs before the measurement windows and never pauses a product process.
func witnessNativeKernelComparison(t *testing.T, abi witnessABI) {
	t.Helper()
	path, e := os.Executable()
	if e != nil {
		t.Fatal("original controller artifact")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "-test.run=^TestDashboardWitnessCommandProbe$")
	cmd.Env = []string{"HORVIUM_WITNESS_COMMAND_PROBE=kernel-idle", "LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	pipe, e := cmd.StdoutPipe()
	if e != nil || cmd.Start() != nil {
		t.Fatal("owned idle control start")
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	ready, e := bufio.NewReader(pipe).ReadString('\n')
	if e != nil || ready != "ready\n" {
		t.Fatal("owned idle control readiness")
	}
	if cmd.Process.Signal(syscall.SIGSTOP) != nil {
		t.Fatal("owned idle control stop")
	}
	performanceWait(t, func() bool {
		c, stop := context.WithTimeout(context.Background(), 2*time.Second)
		defer stop()
		command := exec.CommandContext(c, "/bin/ps", "-p", strconv.Itoa(cmd.Process.Pid), "-o", "stat=")
		b, _, e := witnessRunSampleCommand(c, command)
		return e == nil && strings.HasPrefix(strings.TrimSpace(string(b)), "T")
	})
	b, e := unix.SysctlRaw("kern.proc.pid", cmd.Process.Pid)
	if e != nil {
		t.Fatal("actual kernel comparison input")
	}
	p, e := witnessDecodeKernelProcess(b, abi.usageLayout())
	if e != nil || p.PID != cmd.Process.Pid || p.UID != uint32(os.Getuid()) || p.Parent != os.Getpid() || p.RSS == 0 {
		t.Fatal("actual kernel comparison identity")
	}
	c, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	command := exec.CommandContext(c, "/bin/ps", "-p", strconv.Itoa(cmd.Process.Pid), "-o", "pid=", "-o", "ppid=", "-o", "rss=", "-o", "time=")
	raw, _, e := witnessRunSampleCommand(c, command)
	fields := strings.Fields(string(raw))
	if e != nil || len(fields) != 4 {
		t.Fatal("independent kernel comparison command")
	}
	pid, pe := strconv.Atoi(fields[0])
	parent, pa := strconv.Atoi(fields[1])
	rss, re := strconv.ParseUint(fields[2], 10, 64)
	cpu, ce := cpuTime(fields[3])
	delta := p.CPU - int64(cpu)
	if delta < 0 {
		delta = -delta
	}
	if pe != nil || pa != nil || re != nil || ce != nil || pid != p.PID || parent != p.Parent || rss != p.RSS/1024 || delta > int64(10*time.Millisecond) {
		t.Fatal("observed native field units disagree with independent ps")
	}
	if witnessWriteDiagnosticJSON("witness-kernel-comparison.json", struct {
		PIDMatched, ParentMatched, UIDMatched, RSSMatched bool
		CPUPrecisionNS                                    int64
		Layout                                            witnessKernelLayout
	}{true, true, true, true, int64(10 * time.Millisecond), abi.usageLayout()}) != nil {
		t.Fatal("original kernel comparison receipt")
	}
	t.Log(fmt.Sprintf("NATIVE_WITNESS_KERNEL_COMPARISON identity=true rss_exact=true cpu_precision_ns=%d actual_header_layout=true resource_acceptance=false", int64(10*time.Millisecond)))
}
