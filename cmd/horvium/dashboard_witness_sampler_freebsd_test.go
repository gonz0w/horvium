//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const witnessSamplerInterval = 100 * time.Millisecond

type witnessSamplerConfig struct {
	ABI                 witnessABI
	Controller, Sampler witnessActor
	Roles               []witnessActor
	Object              witnessFileObject
}
type witnessSamplerReceipt struct {
	OK      bool
	Failure string
	EndNS   int64
	Samples int
}
type witnessSamplerOwnerReceipt struct {
	Config               witnessSamplerConfig
	Result               witnessSamplerReceipt
	Coverage             witnessScheduleReport
	OriginalFile, SHA256 string
	Object               witnessFileObject
	Reaped, WaitFailed   bool
	HighRSS              uint64
	CPUNS                int64
}
type witnessNativeUsage struct {
	once    sync.Once
	channel *net.UnixConn
	file    *os.File
	cmd     *exec.Cmd
	done    chan error
	cancel  context.CancelFunc
	config  witnessSamplerConfig
	samples []witnessNativeUsageSample
	err     error
	receipt witnessSamplerOwnerReceipt
}

// An explicit inherited channel and original file bind this private child to
// one disposable controller. EOF, invalid control and wall-clock expiry deny
// completion. There is no listener, service, HTTP endpoint or product change.
func TestNativeDashboardWitnessSampler(t *testing.T) {
	if os.Getenv("HORVIUM_WITNESS_SAMPLER") != "1" {
		t.Skip("owned sampler only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" || os.Getuid() != 0 {
		t.Fatal("sampler disposable ownership")
	}
	inherited := os.NewFile(3, "sampler-parent")
	connection, e := net.FileConn(inherited)
	inherited.Close()
	if e != nil {
		t.Fatal("sampler channel")
	}
	control := connection.(*net.UnixConn)
	defer control.Close()
	output := os.NewFile(4, "sampler-original")
	defer output.Close()
	_ = control.SetDeadline(time.Now().Add(45 * time.Second))
	var cfg witnessSamplerConfig
	if witnessControlRead(control, &cfg) != nil || !cfg.ABI.valid() || cfg.Controller.PID != os.Getppid() || cfg.Sampler.PID != os.Getpid() || cfg.Controller.UID != 0 || cfg.Sampler.UID != 0 || cfg.Controller.SHA256 != cfg.Sampler.SHA256 || cfg.Controller.Executable != cfg.Sampler.Executable || cfg.Controller.check(cfg.ABI) != nil || cfg.Sampler.check(cfg.ABI) != nil {
		t.Fatal("sampler original lifetimes")
	}
	uid, pid, e := witnessNativePeer(control, cfg.ABI)
	info, se := output.Stat()
	object, oe := witnessObject(info)
	if e != nil || uid != 0 || pid != cfg.Controller.PID || se != nil || oe != nil || object != cfg.Object || object.UID != 0 || object.Links != 1 || object.Size != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		t.Fatal("sampler original channel/file")
	}
	_ = control.SetDeadline(time.Now().Add(650 * time.Second))
	stop := make(chan bool, 1)
	go func() {
		var operation witnessNativeCommand
		e := witnessControlRead(control, &operation)
		stop <- e == nil && operation.Operation == "stop" && operation.Actor == (witnessActor{})
	}()
	result := witnessSamplerReceipt{OK: true}
	started := time.Now()
	budget := &witnessSamplerWriter{writer: output}
	scratch := make([]byte, witnessFrameCap)
	sample := func(scheduled int64) bool {
		s, e := witnessNativeSample(cfg.Roles, cfg.Controller, cfg.ABI, scratch)
		s.ScheduledNS, s.AtNS = scheduled, int64(time.Since(started))
		if e != nil || witnessControlWrite(budget, s) != nil {
			result.OK, result.Failure = false, "sample-or-original-write"
			return false
		}
		result.Samples++
		return true
	}
	if !sample(0) || witnessControlWrite(control, witnessSamplerReceipt{OK: result.OK, Failure: result.Failure}) != nil {
		t.Fatal("sampler initial sample")
	}
	next := int64(witnessSamplerInterval)
	timer := time.NewTimer(time.Until(started.Add(time.Duration(next))))
	defer timer.Stop()
	limit := time.NewTimer(650 * time.Second)
	defer limit.Stop()
loop:
	for {
		select {
		case clean := <-stop:
			if !clean {
				result.OK, result.Failure = false, "control-cancelled"
			}
			break loop
		case <-limit.C:
			result.OK, result.Failure = false, "window-deadline"
			break loop
		case <-timer.C:
			if !sample(next) {
				break loop
			}
			// Never synthesize catch-up records for an interval already missed.
			next = (int64(time.Since(started))/int64(witnessSamplerInterval) + 1) * int64(witnessSamplerInterval)
			timer.Reset(time.Until(started.Add(time.Duration(next))))
		}
	}
	result.EndNS = int64(time.Since(started))
	if output.Sync() != nil || output.Chmod(0400) != nil {
		result.OK, result.Failure = false, "original-seal"
	}
	_ = control.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_ = witnessControlWrite(control, result)
	if !result.OK {
		t.Error("sampler refused completion", result.Failure)
	}
}

type witnessSamplerWriter struct {
	writer io.Writer
	bytes  int
}

func (w *witnessSamplerWriter) Write(p []byte) (int, error) {
	if len(p) > witnessSamplerBytes-w.bytes {
		return 0, io.ErrShortBuffer
	}
	n, e := w.writer.Write(p)
	w.bytes += n
	return n, e
}

func newWitnessNativeUsage(t *testing.T, p *performancePair, observer *witnessFixtureObserver, abi witnessABI) *witnessNativeUsage {
	t.Helper()
	pids := []int{p.management.cmd.Process.Pid, p.helper.cmd.Process.Pid}
	if observer != nil {
		pids = append(pids, observer.process.cmd.Process.Pid)
	}
	roles := []witnessActor{}
	for _, pid := range pids {
		actor, e := witnessObserveActor(pid, abi)
		if e != nil {
			t.Fatal("original measurement actor")
		}
		roles = append(roles, actor)
	}
	controller, e := witnessObserveActor(os.Getpid(), abi)
	if e != nil {
		t.Fatal("original controller actor")
	}
	return startWitnessSampler(t, roles, controller, abi)
}
func startWitnessSampler(t *testing.T, roles []witnessActor, controller witnessActor, abi witnessABI) *witnessNativeUsage {
	t.Helper()
	dir := filepath.Join("../../.tools", "witness-samplers")
	if e := os.Mkdir(dir, 0700); e != nil && !os.IsExist(e) {
		t.Fatal("sampler original directory")
	}
	info, e := os.Lstat(dir)
	object, oe := witnessObject(info)
	if e != nil || oe != nil || !info.IsDir() || info.Mode().Perm() != 0700 || object.UID != 0 {
		t.Fatal("sampler directory ownership")
	}
	file, e := os.CreateTemp(dir, "usage-")
	if e != nil {
		t.Fatal("sampler original create")
	}
	info, e = file.Stat()
	object, oe = witnessObject(info)
	if e != nil || oe != nil {
		file.Close()
		t.Fatal("sampler original identity")
	}
	parent, child, e := witnessSocketPair()
	if e != nil {
		file.Close()
		t.Fatal("sampler owned channel")
	}
	inherited, e := child.File()
	child.Close()
	if e != nil {
		parent.Close()
		file.Close()
		t.Fatal("sampler inherited channel")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 650*time.Second)
	cmd := exec.CommandContext(ctx, controller.Executable, "-test.run=^TestNativeDashboardWitnessSampler$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "HORVIUM_WITNESS_SAMPLER=1", "LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	cmd.ExtraFiles = []*os.File{inherited, file}
	m := &witnessNativeUsage{channel: parent, file: file, cmd: cmd, done: make(chan error, 1), cancel: cancel}
	if cmd.Start() != nil {
		inherited.Close()
		parent.Close()
		file.Close()
		cancel()
		t.Fatal("sampler process start")
	}
	inherited.Close()
	go func() { m.done <- cmd.Wait() }()
	t.Cleanup(func() {
		m.close()
		if !m.receipt.Reaped {
			t.Error("sampler not reaped")
		}
	})
	actor, e := witnessObserveActor(cmd.Process.Pid, abi)
	if e != nil {
		t.Fatal("sampler original actor")
	}
	m.config = witnessSamplerConfig{abi, controller, actor, roles, object}
	_ = parent.SetDeadline(time.Now().Add(45 * time.Second))
	var ready witnessSamplerReceipt
	if witnessControlWrite(parent, m.config) != nil || witnessControlRead(parent, &ready) != nil || !ready.OK {
		t.Fatal("sampler original readiness")
	}
	return m
}
func (m *witnessNativeUsage) close() {
	m.once.Do(func() {
		defer m.cancel()
		defer m.file.Close()
		_ = m.channel.SetDeadline(time.Now().Add(5 * time.Second))
		result := witnessSamplerReceipt{}
		if witnessControlWrite(m.channel, witnessNativeCommand{Operation: "stop"}) != nil || witnessControlRead(m.channel, &result) != nil || !result.OK {
			m.err = witnessFailure("sampler-control")
		}
		m.channel.Close()
		var wait error
		select {
		case wait = <-m.done:
		case <-time.After(5 * time.Second):
			_ = m.cmd.Process.Kill()
			m.err = witnessFailure("sampler-wait-timeout")
			select {
			case wait = <-m.done:
			case <-time.After(5 * time.Second):
				return
			}
		}
		m.receipt = witnessSamplerOwnerReceipt{Config: m.config, Result: result, Reaped: m.cmd.ProcessState != nil, WaitFailed: wait != nil, OriginalFile: filepath.Base(m.file.Name())}
		var u *syscall.Rusage
		if m.cmd.ProcessState != nil {
			u, _ = m.cmd.ProcessState.SysUsage().(*syscall.Rusage)
		}
		if u != nil && u.Maxrss > 0 && uint64(u.Maxrss) <= math.MaxUint64/1024 {
			cpu, e := witnessNativeCPU(u)
			if e == nil {
				m.receipt.HighRSS, m.receipt.CPUNS = uint64(u.Maxrss)*1024, cpu
			} else {
				m.err = witnessFailure("sampler-wait-usage")
			}
		} else {
			m.err = witnessFailure("sampler-wait-usage")
		}
		if wait != nil {
			m.err = witnessFailure("sampler-exit")
		}
		if m.err == nil {
			m.readback(result)
		}
		if witnessWritePrivateIndex(filepath.Dir(m.file.Name()), filepath.Base(m.file.Name())+"-receipt.json", m.receipt) != nil {
			m.err = witnessFailure("sampler-receipt")
		}
	})
}
func (m *witnessNativeUsage) readback(result witnessSamplerReceipt) {
	fail := func() { m.err = witnessFailure("sampler-original-readback") }
	before, e := m.file.Stat()
	object, oe := witnessObject(before)
	pathInfo, pe := os.Lstat(m.file.Name())
	if e != nil || oe != nil || pe != nil || !os.SameFile(before, pathInfo) || object.Device != m.config.Object.Device || object.Inode != m.config.Object.Inode || object.UID != 0 || object.Links != 1 || object.Size <= 0 || object.Size > witnessSamplerBytes || before.Mode().Perm() != 0400 {
		fail()
		return
	}
	data, e := witnessSamplerReadOriginal(m.file)
	if e != nil {
		fail()
		return
	}
	after, ae := m.file.Stat()
	afterObject, aoe := witnessObject(after)
	afterPath, ape := os.Lstat(m.file.Name())
	if ae != nil || aoe != nil || ape != nil || afterObject != object || !os.SameFile(before, afterPath) {
		fail()
		return
	}
	r := bytes.NewReader(data)
	var schedule []witnessScheduleSample
	for r.Len() > 0 {
		var s witnessNativeUsageSample
		if len(m.samples) >= 6502 || witnessControlRead(r, &s) != nil || s.ReadNS <= 0 || s.ReadNS > int64(2*time.Second) || s.PairRSS == 0 || s.ControllerRSS == 0 || s.SamplerRSS == 0 {
			fail()
			return
		}
		m.samples = append(m.samples, s)
		schedule = append(schedule, witnessScheduleSample{s.ScheduledNS, s.AtNS})
	}
	coverage, e := witnessScheduleCoverage(schedule, result.EndNS, int64(witnessSamplerInterval))
	if e != nil || result.Samples != len(m.samples) {
		fail()
		return
	}
	m.receipt.Coverage, m.receipt.Object, m.receipt.SHA256 = coverage, object, fmt.Sprintf("%x", sha256.Sum256(data))
}
func (m *witnessNativeUsage) finish(t *testing.T) []witnessNativeUsageSample {
	t.Helper()
	m.close()
	if m.err != nil || !m.receipt.Reaped || m.receipt.WaitFailed || len(m.samples) == 0 {
		t.Fatal("original instrumentation owner unavailable", m.err)
	}
	t.Logf("NATIVE_WITNESS_SAMPLER planned=%d samples=%d missing=%d duplicates=%d late=%d reaped=true sampler_high=%d sampler_cpu_ns=%d resource_acceptance=false", m.receipt.Coverage.Planned, len(m.samples), m.receipt.Coverage.Missing, m.receipt.Coverage.Duplicates, m.receipt.Coverage.Late, m.receipt.HighRSS, m.receipt.CPUNS)
	return append([]witnessNativeUsageSample(nil), m.samples...)
}

// These actual child controls precede measurement: an uncertain window never
// returns samples, but every started sampler is waited and its usage retained.
func witnessNativeSamplerControls(t *testing.T, p *performancePair, abi witnessABI) {
	t.Helper()
	for _, operation := range []string{"parent-channel-eof", "sampler-sigkill"} {
		m := newWitnessNativeUsage(t, p, nil, abi)
		if operation == "parent-channel-eof" {
			_ = m.channel.Close()
		} else if m.cmd.Process.Kill() != nil {
			t.Fatal("sampler owned kill")
		}
		m.close()
		if m.err == nil || !m.receipt.Reaped || m.receipt.HighRSS == 0 || m.receipt.CPUNS <= 0 {
			t.Fatal("uncertain sampler accepted or omitted started owner", operation)
		}
		t.Logf("NATIVE_WITNESS_SAMPLER_CONTROL case=%s denied=true reaped=true usage_retained=true resource_acceptance=false", operation)
	}
}

// An actual unprivileged connector refuses a mismatched artifact binding before
// connecting or transferring a descriptor. Its outside-window cost is retained.
func witnessNativeConnectorArtifactControl(t *testing.T, p *performancePair, abi witnessABI, fixture string) {
	t.Helper()
	actor, e := witnessObserveActor(os.Getpid(), abi)
	if e != nil {
		t.Fatal("connector control original artifact")
	}
	parent, child, e := witnessSocketPair()
	if e != nil {
		t.Fatal("connector control channel")
	}
	defer parent.Close()
	file, e := child.File()
	child.Close()
	if e != nil {
		t.Fatal("connector control inherited channel")
	}
	defer file.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	originalDeadline, _ := ctx.Deadline()
	cmd := exec.CommandContext(ctx, fixture, "-test.run=^TestNativeDashboardWitnessConnector$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "HORVIUM_WITNESS_CONNECTOR=" + p.config.HelperSocket, "HORVIUM_WITNESS_UID=" + fmt.Sprint(p.config.ManagementUID), "LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: p.config.ManagementUID, Gid: p.config.SocketGID, Groups: []uint32{p.config.SocketGID}}}
	cmd.ExtraFiles = []*os.File{file}
	stdout, stderr := witnessCommandOutput{limit: 4096}, witnessCommandOutput{limit: 4096}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	started := time.Now()
	if cmd.Start() != nil {
		t.Fatal("connector control start")
	}
	waited := false
	defer func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	// Drop the parent's duplicate of the child's inherited endpoint. Otherwise
	// a real child exit cannot produce EOF on parent until the deadline fires.
	if file.Close() != nil {
		t.Fatal("connector control inherited duplicate close")
	}
	_ = parent.SetDeadline(originalDeadline)
	if witnessControlWrite(parent, witnessConnectorConfig{DeadlineNS: originalDeadline.UnixNano(), ABI: abi, ParentPID: os.Getpid(), Fixture: fixture, SHA256: strings.Repeat("0", 64)}) != nil {
		t.Fatal("connector control bootstrap")
	}
	var accepted witnessActor
	if witnessControlRead(parent, &accepted) == nil {
		t.Fatal("wrong artifact binding accepted")
	}
	waitError := cmd.Wait()
	waited = true
	var usage *syscall.Rusage
	present, exited, code, signal, status := false, false, -1, 0, uint32(0)
	if cmd.ProcessState != nil {
		usage, _ = cmd.ProcessState.SysUsage().(*syscall.Rusage)
		if w, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			present = true
			exited = w.Exited()
			code = w.ExitStatus()
			status = uint32(w)
			if w.Signaled() {
				signal = int(w.Signal())
			}
		}
	}
	cpu, ce := witnessNativeCPU(usage)
	refusal := bytes.Contains(stdout.data, []byte("connector original artifact/lifetime")) || bytes.Contains(stderr.data, []byte("connector original artifact/lifetime"))
	denied := waitError != nil && ctx.Err() == nil && present && exited && code == 1 && signal == 0 && usage != nil && usage.Maxrss > 0 && ce == nil && cpu > 0 && refusal && !stdout.overflow && !stderr.overflow
	// Preserve the actual outcome and bounded bytes before asserting semantic refusal.
	if witnessWriteDiagnosticJSON("witness-connector-artifact-control.json", struct {
		Schema                                                        int
		Denied, Reaped, StatePresent, Exited, ArtifactRefusalObserved bool
		ExitCode, Signal                                              int
		RawWaitStatus                                                 uint32
		DeadlineExceeded, Cancelled, StdoutOverflow, StderrOverflow   bool
		DeadlineNS, ElapsedNS, CPUNS                                  int64
		Stdout, Stderr                                                []byte
		FixtureSHA256                                                 string
		Usage                                                         *syscall.Rusage
	}{2, denied, cmd.ProcessState != nil, present, exited, refusal, code, signal, status, errors.Is(ctx.Err(), context.DeadlineExceeded), errors.Is(ctx.Err(), context.Canceled), stdout.overflow, stderr.overflow, originalDeadline.UnixNano(), int64(time.Since(started)), cpu, stdout.data, stderr.data, actor.SHA256, usage}) != nil {
		t.Fatal("connector control original receipt")
	}
	if !denied {
		t.Fatal("wrong artifact connector control not denied/accounted")
	}
	t.Log("NATIVE_WITNESS_CONNECTOR_ARTIFACT_CONTROL mismatch=denied descriptor=none reaped=true usage_retained=true resource_acceptance=false")
}
