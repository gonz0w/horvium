//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Compiled in the disposable guest against its actual base headers. No guessed
// xucred/kinfo offsets, cgo, installed component or product ABI change.
const witnessABIProbe = `// SPDX-License-Identifier: BSD-3-Clause
#include <sys/types.h>
#include <sys/ucred.h>
#include <sys/user.h>
#include <sys/un.h>
#include <sys/sysctl.h>
#include <stddef.h>
#include <stdio.h>
#include <unistd.h>
int main(void) {
 unsigned int endian=1;
 long page=sysconf(_SC_PAGESIZE);
 if (page <= 0) return 1;
 printf("{\"cred_size\":%zu,\"cred_version\":%u,\"version_offset\":%zu,\"uid_offset\":%zu,\"pid_offset\":%zu,\"proc_size\":%zu,\"proc_pid\":%zu,\"proc_uid\":%zu,\"proc_sec\":%zu,\"proc_usec\":%zu,\"little_endian\":%s,\"pid_size\":%zu,\"uid_size\":%zu,\"time_size\":%zu,\"proc_ppid\":%zu,\"proc_rss\":%zu,\"proc_runtime\":%zu,\"rss_size\":%zu,\"runtime_size\":%zu,\"page_bytes\":%ld,\"proc_mib\":[%d,%d,%d],\"int_size\":%zu,\"size_t_size\":%zu}\n",
 sizeof(struct xucred), XUCRED_VERSION, offsetof(struct xucred, cr_version), offsetof(struct xucred, cr_uid), offsetof(struct xucred, cr_pid), sizeof(struct kinfo_proc), offsetof(struct kinfo_proc, ki_pid), offsetof(struct kinfo_proc, ki_uid), offsetof(struct kinfo_proc, ki_start.tv_sec), offsetof(struct kinfo_proc, ki_start.tv_usec), *(unsigned char*)&endian ? "true":"false", sizeof(pid_t), sizeof(uid_t), sizeof(time_t), offsetof(struct kinfo_proc, ki_ppid), offsetof(struct kinfo_proc, ki_rssize), offsetof(struct kinfo_proc, ki_runtime), sizeof(((struct kinfo_proc*)0)->ki_rssize), sizeof(((struct kinfo_proc*)0)->ki_runtime), page, CTL_KERN, KERN_PROC, KERN_PROC_PROC, sizeof(int), sizeof(size_t));
 return 0;
}
`

type witnessABI struct {
	ProcMIB       [3]int32 `json:"proc_mib"`
	IntSize       int      `json:"int_size"`
	SizeTSize     int      `json:"size_t_size"`
	ProcPPID      int      `json:"proc_ppid"`
	ProcRSS       int      `json:"proc_rss"`
	ProcRuntime   int      `json:"proc_runtime"`
	RSSSize       int      `json:"rss_size"`
	RuntimeSize   int      `json:"runtime_size"`
	PageBytes     uint64   `json:"page_bytes"`
	CredSize      int      `json:"cred_size"`
	CredVersion   uint32   `json:"cred_version"`
	VersionOffset int      `json:"version_offset"`
	UIDOffset     int      `json:"uid_offset"`
	PIDOffset     int      `json:"pid_offset"`
	ProcSize      int      `json:"proc_size"`
	ProcPID       int      `json:"proc_pid"`
	ProcUID       int      `json:"proc_uid"`
	ProcSec       int      `json:"proc_sec"`
	ProcUsec      int      `json:"proc_usec"`
	LittleEndian  bool     `json:"little_endian"`
	PIDSize       int      `json:"pid_size"`
	UIDSize       int      `json:"uid_size"`
	TimeSize      int      `json:"time_size"`
}

func (a witnessABI) usageLayout() witnessKernelLayout {
	return witnessKernelLayout{Size: a.ProcSize, PID: a.ProcPID, Parent: a.ProcPPID, UID: a.ProcUID, Sec: a.ProcSec, Usec: a.ProcUsec, RSS: a.ProcRSS, Runtime: a.ProcRuntime, PageBytes: a.PageBytes}
}
func (a witnessABI) valid() bool {
	if runtime.GOARCH != "amd64" || !a.LittleEndian || a.CredVersion != 0 || a.CredSize < 12 || a.CredSize > 256 || a.ProcSize < 32 || a.ProcSize > 4096 || a.PIDSize != 4 || a.UIDSize != 4 || a.TimeSize != 8 || a.IntSize != 4 || a.SizeTSize != 8 || a.ProcMIB[0] <= 0 || a.ProcMIB[1] <= 0 || a.ProcMIB[2] <= 0 || a.RSSSize != 8 || a.RuntimeSize != 8 || !a.usageLayout().valid() {
		return false
	}
	for _, p := range []int{a.VersionOffset, a.UIDOffset, a.PIDOffset} {
		if p < 0 || p > a.CredSize-4 {
			return false
		}
	}
	for _, p := range []int{a.ProcPID, a.ProcUID} {
		if p < 0 || p > a.ProcSize-4 {
			return false
		}
	}
	for _, p := range []int{a.ProcSec, a.ProcUsec} {
		if p < 0 || p > a.ProcSize-8 {
			return false
		}
	}
	return a.VersionOffset != a.UIDOffset && a.VersionOffset != a.PIDOffset && a.UIDOffset != a.PIDOffset
}
func witnessNativePeer(c *net.UnixConn, a witnessABI) (uint32, int, error) {
	if c == nil || !a.valid() {
		return 0, 0, errDashboardWitness
	}
	raw, e := c.SyscallConn()
	if e != nil {
		return 0, 0, errDashboardWitness
	}
	b := make([]byte, a.CredSize)
	size := uint32(len(b))
	var errno syscall.Errno
	e = raw.Control(func(fd uintptr) {
		_, _, errno = unix.Syscall6(unix.SYS_GETSOCKOPT, fd, unix.SOL_LOCAL, unix.LOCAL_PEERCRED, uintptr(unsafe.Pointer(&b[0])), uintptr(unsafe.Pointer(&size)), 0)
	})
	if e != nil || errno != 0 || int(size) != len(b) || binary.LittleEndian.Uint32(b[a.VersionOffset:]) != a.CredVersion {
		return 0, 0, errDashboardWitness
	}
	uid := binary.LittleEndian.Uint32(b[a.UIDOffset:])
	pid := int(int32(binary.LittleEndian.Uint32(b[a.PIDOffset:])))
	if pid <= 1 {
		return 0, 0, errDashboardWitness
	}
	return uid, pid, nil
}

type witnessActor struct {
	PID        int    `json:"pid"`
	UID        uint32 `json:"uid"`
	Sec        int64  `json:"sec"`
	Usec       int64  `json:"usec"`
	Executable string `json:"executable"`
	SHA256     string `json:"sha256"`
}

func witnessObserveActor(pid int, a witnessABI) (witnessActor, error) {
	if pid <= 1 || !a.valid() {
		return witnessActor{}, errDashboardWitness
	}
	b, e := unix.SysctlRaw("kern.proc.pid", pid)
	path, pe := unix.SysctlRaw("kern.proc.pathname", pid)
	if e != nil || pe != nil || len(b) != a.ProcSize || int(int32(binary.LittleEndian.Uint32(b[a.ProcPID:]))) != pid {
		return witnessActor{}, errDashboardWitness
	}
	name := string(bytes.TrimRight(path, "\x00"))
	if !filepath.IsAbs(name) {
		return witnessActor{}, errDashboardWitness
	}
	f, e := os.Open(name)
	if e != nil {
		return witnessActor{}, errDashboardWitness
	}
	defer f.Close()
	i, e := f.Stat()
	if e != nil || !i.Mode().IsRegular() || i.Size() < 1 || i.Size() > 128<<20 {
		return witnessActor{}, errDashboardWitness
	}
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(f, i.Size()+1))
	after, ae := f.Stat()
	if e != nil || ae != nil || n != i.Size() || !dashboardArtifactUnchanged(i, after) {
		return witnessActor{}, errDashboardWitness
	}
	v := witnessActor{PID: pid, UID: binary.LittleEndian.Uint32(b[a.ProcUID:]), Sec: int64(binary.LittleEndian.Uint64(b[a.ProcSec:])), Usec: int64(binary.LittleEndian.Uint64(b[a.ProcUsec:])), Executable: name, SHA256: fmt.Sprintf("%x", h.Sum(nil))}
	if v.Sec <= 0 || v.Usec < 0 || v.Usec >= 1000000 {
		return witnessActor{}, errDashboardWitness
	}
	return v, nil
}
func (v witnessActor) check(a witnessABI) error {
	now, e := witnessObserveActor(v.PID, a)
	if e != nil || now != v {
		return errDashboardWitness
	}
	return nil
}
func (v witnessActor) incarnation() string {
	return fmt.Sprintf("%d:%d:%d:%s", v.PID, v.Sec, v.Usec, v.SHA256)
}

type witnessFileObject struct {
	Device     uint64 `json:"device"`
	Inode      uint64 `json:"inode"`
	UID        uint32 `json:"uid"`
	GID        uint32 `json:"gid"`
	Links      uint64 `json:"links"`
	Size       int64  `json:"size"`
	ModifiedNS int64  `json:"modified_ns"`
}

func witnessObject(i os.FileInfo) (witnessFileObject, error) {
	if i == nil {
		return witnessFileObject{}, errDashboardWitness
	}
	s, ok := i.Sys().(*syscall.Stat_t)
	if !ok {
		return witnessFileObject{}, errDashboardWitness
	}
	return witnessFileObject{uint64(s.Dev), s.Ino, s.Uid, s.Gid, uint64(s.Nlink), i.Size(), i.ModTime().UnixNano()}, nil
}
func witnessRootSocket(path string, gid uint32) (os.FileInfo, error) {
	i, e := os.Lstat(path)
	if e != nil {
		return nil, errDashboardWitness
	}
	s, ok := i.Sys().(*syscall.Stat_t)
	if !ok || s.Uid != 0 || s.Gid != gid || i.Mode()&os.ModeSocket == 0 || i.Mode().Perm() != 0660 {
		return nil, errDashboardWitness
	}
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		d, e := os.Lstat(dir)
		if e != nil || !d.IsDir() {
			return nil, errDashboardWitness
		}
		s, ok := d.Sys().(*syscall.Stat_t)
		if !ok || s.Uid != 0 || d.Mode().Perm()&0022 != 0 && d.Mode()&os.ModeSticky == 0 {
			return nil, errDashboardWitness
		}
		if dir == "/" {
			break
		}
	}
	return i, nil
}

func witnessSocketPair() (a, b *net.UnixConn, err error) {
	pair, e := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if e != nil {
		return nil, nil, e
	}
	convert := func(fd int) (*net.UnixConn, error) {
		f := os.NewFile(uintptr(fd), "owned-witness-channel")
		c, e := net.FileConn(f)
		f.Close()
		if e != nil {
			return nil, e
		}
		u, ok := c.(*net.UnixConn)
		if !ok {
			c.Close()
			return nil, errDashboardWitness
		}
		return u, nil
	}
	a, e = convert(pair[0])
	if e != nil {
		unix.Close(pair[1])
		return nil, nil, e
	}
	b, e = convert(pair[1])
	if e != nil {
		a.Close()
		return nil, nil, e
	}
	return a, b, nil
}

type witnessConnectorConfig struct {
	DeadlineNS      int64 `json:"deadline_unix_ns"`
	ABI             witnessABI
	ParentPID       int
	Fixture, SHA256 string
	Fault           string `json:"fault,omitempty"`
}

func TestNativeDashboardWitnessConnector(t *testing.T) {
	path := os.Getenv("HORVIUM_WITNESS_CONNECTOR")
	if path == "" {
		t.Skip("owned reexec connector only")
	}
	uid, e := strconv.ParseUint(os.Getenv("HORVIUM_WITNESS_UID"), 10, 32)
	if e != nil || os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" || os.Geteuid() == 0 || uint32(os.Geteuid()) != uint32(uid) {
		t.Fatal("connector identity")
	}
	channelFile := os.NewFile(3, "private-connector-parent")
	channel, e := net.FileConn(channelFile)
	channelFile.Close()
	if e != nil {
		t.Fatal("connector channel")
	}
	defer channel.Close()
	u := channel.(*net.UnixConn)
	_ = u.SetDeadline(time.Now().Add(10 * time.Second))
	var cfg witnessConnectorConfig
	if witnessControlRead(u, &cfg) != nil || !cfg.ABI.valid() || cfg.ParentPID != os.Getppid() || !witnessAncillaryFaultValid(cfg.Fault) {
		t.Fatal("connector original bootstrap")
	}
	parentUID, parentPID, pe := witnessNativePeer(u, cfg.ABI)
	deadline, de := witnessConnectorDeadline(time.Now(), cfg.DeadlineNS)
	if pe != nil || parentUID != 0 || parentPID != cfg.ParentPID || de != nil || u.SetDeadline(deadline) != nil {
		t.Fatal("connector original collection deadline/parent")
	}
	actor, ae := witnessObserveActor(os.Getpid(), cfg.ABI)
	if ae != nil || actor.UID != uint32(uid) || actor.Executable != cfg.Fixture || actor.SHA256 != cfg.SHA256 {
		t.Fatal("connector original artifact/lifetime")
	}
	if witnessControlWrite(u, actor) != nil {
		t.Fatal("connector original identity receipt")
	}
	var bound [1]byte
	if _, e := io.ReadFull(u, bound[:]); e != nil || bound[0] != 'B' {
		t.Fatal("connector original identity confirmation")
	}
	if cfg.Fault == "eof" {
		return
	}
	if cfg.Fault == "missing-rights" {
		if _, e := u.Write([]byte{'F'}); e != nil {
			t.Fatal("fault byte")
		}
		return
	}
	c, e := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if e != nil {
		t.Fatal("connector connect")
	}
	defer c.Close()
	f, e := c.File()
	if e != nil {
		t.Fatal("connector descriptor")
	}
	defer f.Close()
	tag := byte('F')
	fds := []int{int(f.Fd())}
	if cfg.Fault == "wrong-tag" {
		tag = 'X'
	}
	if cfg.Fault == "extra-rights" {
		fds = append(fds, int(f.Fd()))
	}
	if cfg.Fault == "truncated-rights" {
		for len(fds) < 64 {
			fds = append(fds, int(f.Fd()))
		}
	}
	if cfg.Fault == "non-socket" {
		r, w, e := os.Pipe()
		if e != nil {
			t.Fatal("fault pipe")
		}
		defer r.Close()
		defer w.Close()
		fds = []int{int(r.Fd())}
	}
	if cfg.Fault == "wrong-endpoint" {
		a, b, e := witnessSocketPair()
		if e != nil {
			t.Fatal("fault endpoint")
		}
		defer a.Close()
		defer b.Close()
		other, e := a.File()
		if e != nil {
			t.Fatal("fault endpoint file")
		}
		defer other.Close()
		fds = []int{int(other.Fd())}
	}
	rights := unix.UnixRights(fds...)
	if n, on, e := u.WriteMsgUnix([]byte{tag}, rights, nil); e != nil || n != 1 || on != len(rights) {
		t.Fatal("connector handoff")
	}
	if cfg.Fault == "replay-before-ack" {
		if _, _, e := u.WriteMsgUnix([]byte{'F'}, unix.UnixRights(int(f.Fd())), nil); e != nil {
			t.Fatal("fault replay")
		}
	}
	if cfg.Fault == "extra-byte" {
		if _, e := u.Write([]byte{'R'}); e != nil {
			t.Fatal("fault extra byte")
		}
	}

	var ack [1]byte
	if _, e := io.ReadFull(u, ack[:]); e != nil || ack[0] != 'A' {
		t.Fatal("connector acknowledgment")
	}
	if cfg.Fault == "replay-after-ack" {
		if _, _, e := u.WriteMsgUnix([]byte{'F'}, unix.UnixRights(int(f.Fd())), nil); e != nil {
			t.Fatal("fault replay after ack")
		}
	}
}

type witnessNativeConfig struct {
	FaultStage        string       `json:"fault_stage,omitempty"`
	RunLabel          string       `json:"run_label,omitempty"`
	ABI               witnessABI   `json:"abi"`
	Parent            witnessActor `json:"parent"`
	Front             string       `json:"front"`
	Back              string       `json:"back"`
	Directory         string       `json:"directory"`
	Mode              string       `json:"mode"`
	ManagementUID     uint32       `json:"management_uid"`
	SocketGID         uint32       `json:"socket_gid"`
	Fixture           string       `json:"fixture"`
	FixtureSHA256     string       `json:"fixture_sha256"`
	Product           string       `json:"product"`
	ProductSHA256     string       `json:"product_sha256"`
	Helper            witnessActor `json:"helper"`
	HelperBinary      string       `json:"helper_binary"`
	HelperSHA256      string       `json:"helper_sha256"`
	PriorArchiveBytes int64        `json:"prior_archive_bytes"`
}
type witnessNativeCommand struct {
	Operation string       `json:"operation"`
	Actor     witnessActor `json:"actor"`
}
type witnessNativeOriginal struct {
	Ref    dashboardWitnessRef `json:"ref"`
	Object witnessFileObject   `json:"object"`
}
type witnessNativeReceipt struct {
	ConnectorEOFChecks   int                              `json:"connector_eof_checks,omitempty"`
	ConnectorEOFRejected int                              `json:"connector_eof_rejected,omitempty"`
	ConnectorBoundary    *witnessConnectorBoundaryReceipt `json:"connector_boundary,omitempty"`
	OK                   bool                             `json:"ok"`
	Failed               bool                             `json:"failed"`
	Failure              string                           `json:"failure"`
	StoreFailed          bool                             `json:"store_failed"`
	Denied               int                              `json:"denied"`
	Originals            []witnessNativeOriginal          `json:"originals"`
	ObserverHigh         uint64                           `json:"observer_high_rss"`
	ObserverCPUNS        int64                            `json:"observer_cpu_ns"`
	ConnectorStarted     int                              `json:"connector_started"`
	ConnectorActive      int                              `json:"connector_active"`
	ConnectorMaxRSS      uint64                           `json:"connector_max_rss"`
	ConnectorCPUNS       int64                            `json:"connector_cpu_ns"`
	Connectors           int                              `json:"connectors"`
	FrontRemoved         bool                             `json:"front_removed"`
	Active               bool                             `json:"active"`
}
type witnessNativeObserver struct {
	connectorEOFChecks, connectorEOFRejected int
	connectorBoundary                        *witnessConnectorBoundaryReceipt
	fault                                    *witnessNativeFaultGate
	mu                                       sync.Mutex
	cfg                                      witnessNativeConfig
	store                                    *dashboardWitnessStore
	listener                                 *net.UnixListener
	front                                    os.FileInfo
	back                                     os.FileInfo
	management                               witnessActor
	registered                               chan struct{}
	active                                   bool
	stopping                                 bool
	allowDeath                               bool
	denied                                   int
	connections                              map[*net.UnixConn]bool
	children                                 sync.WaitGroup
	connectorMax                             uint64
	connectorCPU                             time.Duration
	connectors                               int
	connectorStarted, connectorActive        int
	failed                                   bool
	failure                                  string
	ctx                                      context.Context
}

func (o *witnessNativeObserver) connection(c *net.UnixConn) {
	defer o.children.Done()
	defer c.Close()
	defer func() { o.mu.Lock(); delete(o.connections, c); o.active = false; o.mu.Unlock() }()
	deadline := time.Now().Add(45 * time.Second)
	_ = c.SetDeadline(deadline)
	select {
	case <-o.registered:
	case <-o.ctx.Done():
		return
	}
	o.mu.Lock()
	management, helper, back, allowDeath := o.management, o.cfg.Helper, o.back, o.allowDeath
	o.mu.Unlock()
	uid, pid, e := witnessNativePeer(c, o.cfg.ABI)
	if e != nil || uid != management.UID || pid != management.PID || management.check(o.cfg.ABI) != nil {
		o.mu.Lock()
		o.denied++
		o.mu.Unlock()
		return
	}
	stage := "request-read"
	fail := func() {
		o.mu.Lock()
		o.failed = true
		if o.failure == "" {
			o.failure = stage
		}
		o.mu.Unlock()
	}
	request, e := io.ReadAll(io.LimitReader(c, witnessFrameCap+5))
	if e != nil || len(request) > witnessFrameCap+4 {
		fail()
		return
	}
	stage = "request-grammar"
	id, e := dashboardWitnessRequest(bytes.NewReader(request))
	if e != nil {
		fail()
		return
	}
	stage = "actual-socket"
	socket, e := witnessRootSocket(o.cfg.Back, o.cfg.SocketGID)
	if e != nil || !os.SameFile(socket, back) || helper.check(o.cfg.ABI) != nil {
		if !allowDeath {
			fail()
		}
		return
	}
	stage = "connector-channel"
	parent, child, e := witnessSocketPair()
	if e != nil {
		fail()
		return
	}
	defer parent.Close()
	stage = "connector-file"
	file, e := child.File()
	child.Close()
	if e != nil {
		fail()
		return
	}
	ctx, cancel := context.WithDeadline(o.ctx, deadline)
	defer cancel()
	cmd := exec.CommandContext(ctx, o.cfg.Fixture, "-test.run=^TestNativeDashboardWitnessConnector$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "HORVIUM_WITNESS_CONNECTOR=" + o.cfg.Back, "HORVIUM_WITNESS_UID=" + strconv.FormatUint(uint64(o.cfg.ManagementUID), 10), "LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	cmd.ExtraFiles = []*os.File{file}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: o.cfg.ManagementUID, Gid: o.cfg.SocketGID, Groups: []uint32{o.cfg.SocketGID}}}
	stage = "connector-start"
	if cmd.Start() != nil {
		file.Close()
		fail()
		return
	}
	file.Close()
	o.mu.Lock()
	o.connectorStarted++
	o.connectorActive++
	o.mu.Unlock()
	waited := false
	defer func() {
		if !waited {
			cancel()
			waitError := cmd.Wait()
			o.accountConnector(cmd, stage, waitError, ctx.Err())
		}
	}()
	_ = parent.SetDeadline(deadline)
	stage = "connector-bootstrap"
	if witnessControlWrite(parent, witnessConnectorConfig{DeadlineNS: deadline.UnixNano(), ABI: o.cfg.ABI, ParentPID: os.Getpid(), Fixture: o.cfg.Fixture, SHA256: o.cfg.FixtureSHA256, Fault: witnessConfiguredAncillary(o.cfg)}) != nil {
		fail()
		return
	}
	stage = "connector-identity"
	var connector witnessActor
	if witnessControlRead(parent, &connector) != nil {
		fail()
		return
	}
	original, e := witnessObserveActor(cmd.Process.Pid, o.cfg.ABI)
	if e != nil || connector != original || original.UID != o.cfg.ManagementUID || original.Executable != o.cfg.Fixture || original.SHA256 != o.cfg.FixtureSHA256 {
		fail()
		return
	}
	stage = "interruption-connect"
	if o.interruptConnector("connect", id, original, parent, cmd, ctx, cancel, &waited) {
		fail()
		return
	}
	stage = "connector-identity-confirm"
	if n, e := parent.Write([]byte{'B'}); e != nil || n != 1 {
		fail()
		return
	}
	stage = "connector-descriptor"
	actual, e := witnessPassedDescriptor(parent)
	if e != nil {
		fail()
		return
	}
	defer actual.Close()
	o.mu.Lock()
	o.connections[actual] = true
	o.mu.Unlock()
	defer func() { o.mu.Lock(); delete(o.connections, actual); o.mu.Unlock() }()
	stage = "helper-peer"
	uid, pid, e = witnessNativePeer(actual, o.cfg.ABI)
	if e != nil || uid != helper.UID || pid != helper.PID || helper.check(o.cfg.ABI) != nil {
		fail()
		return
	}
	stage = "interruption-handoff"
	if o.interruptConnector("handoff", id, original, parent, cmd, ctx, cancel, &waited) {
		fail()
		return
	}
	stage = "connector-ack"
	if n, e := parent.Write([]byte{'A'}); e != nil || n != 1 {
		fail()
		return
	}
	stage = "connector-wait"
	e = cmd.Wait()
	waited = true
	accounted := o.accountConnector(cmd, stage, e, ctx.Err()) == nil
	o.recordConnectorBoundary(original, cmd, accounted, false, e, ctx.Err())
	if !accounted || e != nil {
		fail()
		return
	}
	stage = "connector-channel-eof"
	_ = parent.SetReadDeadline(deadline)
	eofError := witnessDescriptorEOF(parent)
	o.mu.Lock()
	o.connectorEOFChecks++
	if eofError != nil {
		o.connectorEOFRejected++
	}
	o.mu.Unlock()
	if eofError != nil {
		fail()
		return
	}
	_ = actual.SetDeadline(deadline)
	stage = "producer-request"
	if n, e := actual.Write(request); e != nil || n != len(request) || actual.CloseWrite() != nil {
		fail()
		return
	}
	stage = "producer-capture"
	var producer io.Reader = actual
	if o.fault != nil && o.fault.stage == "capture" {
		producer = &witnessInterruptedReader{source: actual, reach: func() error { return o.fault.reach("capture", id) }}
	}
	ref, e := o.store.capture(producer, id, o.cfg.Mode, management.incarnation(), helper.incarnation())
	if e != nil {
		fail()
		return
	}
	if o.fault.reach("sealed", id) != nil {
		stage = "interruption-sealed"
		fail()
		return
	}
	stage = "producer-lifetime"
	if ref.Complete && helper.check(o.cfg.ABI) != nil {
		fail()
		return
	}
	stage = "producer-forward"
	var delivery io.Writer = c
	if o.fault != nil && o.fault.stage == "forward" {
		delivery = &witnessInterruptedWriter{target: c, reach: func() error { return o.fault.reach("forward", id) }}
	}
	if o.store.forward(ref, delivery) != nil {
		o.store.mu.Lock()
		v := o.store.sealed[ref.Name]
		if v != nil {
			v.ref.Delivered = false
			v.ref.ForwardSHA256 = ""
		}
		o.store.mu.Unlock()
		fail()
		return
	}
	stage = "original-index"
	o.store.mu.Lock()
	sealed := o.store.sealed[ref.Name]
	object, e := witnessObject(sealed.info)
	record := witnessNativeOriginal{sealed.ref, object}
	o.store.mu.Unlock()
	if e != nil || witnessWritePrivateIndex(o.cfg.Directory, ref.Name+".json", record) != nil {
		fail()
		return
	}
	// Index is published only after exact bytes and write-half EOF. Originals are
	// not publication records; independent metadata confirmation remains required.
}

type witnessConnectorWaitDiagnostic struct {
	Stage                                   string
	WaitFailed, DeadlineExceeded, Cancelled bool
}

func (o *witnessNativeObserver) accountConnector(cmd *exec.Cmd, stage string, waitError, contextError error) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.connectorActive--
	var u *syscall.Rusage
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
		u, _ = cmd.ProcessState.SysUsage().(*syscall.Rusage)
	}
	diagnostic := witnessConnectorWaitDiagnostic{stage, waitError != nil, errors.Is(contextError, context.DeadlineExceeded), errors.Is(contextError, context.Canceled)}
	if u == nil || u.Maxrss <= 0 || uint64(u.Maxrss) > math.MaxUint64/1024 {
		_ = witnessWriteDiagnosticJSON("witness-rejected-connector-usage.json", struct {
			Usage        *syscall.Rusage
			StatePresent bool
			ExitCode     int
			Context      witnessConnectorWaitDiagnostic
		}{u, cmd.ProcessState != nil, exitCode, diagnostic})
		o.failed = true
		if o.failure == "" {
			o.failure = "connector-usage"
		}
		return errDashboardWitness
	}
	cpu, e := witnessNativeCPU(u)
	if e != nil || cpu > math.MaxInt64-int64(o.connectorCPU) {
		_ = witnessWriteDiagnosticJSON("witness-rejected-connector-usage.json", struct {
			Usage              *syscall.Rusage
			CPU                int64
			PriorCPU           int64
			ConversionRejected bool
			Context            witnessConnectorWaitDiagnostic
		}{u, cpu, int64(o.connectorCPU), e != nil, diagnostic})
		o.failed = true
		if o.failure == "" {
			o.failure = "connector-usage"
		}
		return errDashboardWitness
	}
	o.connectorMax = max(o.connectorMax, uint64(u.Maxrss)*1024)
	o.connectorCPU += time.Duration(cpu)
	o.connectors++
	return nil
}
func (o *witnessNativeObserver) receipt() witnessNativeReceipt {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.store.mu.Lock()
	defer o.store.mu.Unlock()
	var usage syscall.Rusage
	ue := syscall.Getrusage(syscall.RUSAGE_SELF, &usage)
	cpu, ce := witnessNativeCPU(&usage)
	if ue != nil || ce != nil || usage.Maxrss <= 0 || uint64(usage.Maxrss) > math.MaxUint64/1024 {
		o.failed = true
		if o.failure == "" {
			o.failure = "observer-usage"
		}
	}
	r := witnessNativeReceipt{ConnectorEOFChecks: o.connectorEOFChecks, ConnectorEOFRejected: o.connectorEOFRejected, ConnectorBoundary: o.connectorBoundary, ObserverHigh: uint64(usage.Maxrss) * 1024, ObserverCPUNS: cpu, ConnectorStarted: o.connectorStarted, ConnectorActive: o.connectorActive, OK: true, Failed: o.failed || o.store.failed, Failure: o.failure, StoreFailed: o.store.failed, Denied: o.denied, ConnectorMaxRSS: o.connectorMax, ConnectorCPUNS: int64(o.connectorCPU), Connectors: o.connectors, Active: o.active}
	for _, ref := range o.store.refs {
		v := o.store.sealed[ref.Name]
		object, e := witnessObject(v.info)
		if e != nil || o.store.check(v) != nil {
			r.Failed = true
			o.failed = true
			if r.Failure == "" {
				if e != nil {
					r.Failure = "receipt-object"
				} else {
					r.Failure = "receipt-custody"
				}
			}
			if o.failure == "" {
				o.failure = r.Failure
			}
		}
		r.Originals = append(r.Originals, witnessNativeOriginal{v.ref, object})
	}
	return r
}
func (o *witnessNativeObserver) stop(graceful bool) error {
	o.mu.Lock()
	o.stopping = true
	if !graceful {
		for c := range o.connections {
			_ = c.Close()
		}
	}
	o.mu.Unlock()
	_ = o.listener.Close()
	o.children.Wait()
	i, e := os.Lstat(o.cfg.Front)
	if e != nil || !os.SameFile(i, o.front) || os.Remove(o.cfg.Front) != nil {
		return errDashboardWitness
	}
	return nil
}
func TestNativeDashboardWitnessObserver(t *testing.T) {
	path := os.Getenv("HORVIUM_WITNESS_OBSERVER")
	if path == "" {
		t.Skip("owned reexec observer only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("observer authorization")
	}
	i, e := os.Lstat(path)
	if e != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 {
		t.Fatal("observer config")
	}
	object, e := witnessObject(i)
	if e != nil || object.UID != 0 || object.Links != 1 {
		t.Fatal("observer config custody")
	}
	f, e := os.Open(path)
	if e != nil {
		t.Fatal("observer config open")
	}
	var cfg witnessNativeConfig
	e = witnessControlRead(f, &cfg)
	f.Close()
	if e != nil || !witnessFaultStageValid(cfg.FaultStage) || cfg.FaultStage != "" && (cfg.RunLabel == "" || !witnessFaultLabelValid(cfg.RunLabel)) || !cfg.ABI.valid() || cfg.Parent.check(cfg.ABI) != nil || cfg.Parent.UID != 0 || cfg.ManagementUID == 0 || cfg.Front == cfg.Back || !filepath.IsAbs(cfg.Front) || !filepath.IsAbs(cfg.Back) || cfg.Helper.UID != 0 || cfg.Helper.Executable != cfg.HelperBinary || cfg.Helper.SHA256 != cfg.HelperSHA256 {
		t.Fatal("observer configuration")
	}
	exe, _ := os.Executable()
	self, e := witnessObserveActor(os.Getpid(), cfg.ABI)
	if e != nil || exe != cfg.Fixture || self.SHA256 != cfg.FixtureSHA256 {
		t.Fatal("observer artifact")
	}
	channelFile := os.NewFile(3, "owned-observer-parent")
	connection, e := net.FileConn(channelFile)
	channelFile.Close()
	if e != nil {
		t.Fatal("observer inherited channel")
	}
	control := connection.(*net.UnixConn)
	defer control.Close()
	uid, pid, e := witnessNativePeer(control, cfg.ABI)
	if e != nil || uid != 0 || pid != cfg.Parent.PID {
		t.Fatal("observer parent peer")
	}
	store, e := newDashboardWitnessStore(cfg.Directory)
	if e != nil {
		t.Fatal("observer archive")
	}
	defer store.close()
	if cfg.PriorArchiveBytes < 0 || cfg.PriorArchiveBytes > witnessArchiveCap {
		t.Fatal("original aggregate archive budget")
	}
	store.bytes = cfg.PriorArchiveBytes
	var fs unix.Statfs_t
	if unix.Statfs(cfg.Directory, &fs) != nil || string(bytes.TrimRight(fs.Fstypename[:], "\x00")) != "tmpfs" {
		t.Fatal("observer original storage")
	}
	archive, e := os.Stat(cfg.Directory)
	ar, ae := witnessObject(archive)
	if e != nil || ae != nil || ar.UID != 0 {
		t.Fatal("observer root archive")
	}
	back, e := witnessRootSocket(cfg.Back, cfg.SocketGID)
	if e != nil || cfg.Helper.check(cfg.ABI) != nil {
		t.Fatal("observer actual helper")
	}
	if _, e := os.Lstat(cfg.Front); !os.IsNotExist(e) {
		t.Fatal("observer refuses adopted front")
	}
	listener, e := net.ListenUnix("unix", &net.UnixAddr{Name: cfg.Front, Net: "unix"})
	if e != nil {
		t.Fatal("observer listener")
	}
	listener.SetUnlinkOnClose(false)
	if os.Chown(cfg.Front, 0, int(cfg.SocketGID)) != nil || os.Chmod(cfg.Front, 0660) != nil {
		listener.Close()
		t.Fatal("observer front ownership")
	}
	front, e := witnessRootSocket(cfg.Front, cfg.SocketGID)
	if e != nil {
		listener.Close()
		t.Fatal("observer front identity")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	workCtx, stopWork := context.WithCancel(ctx)
	defer stopWork()
	o := &witnessNativeObserver{cfg: cfg, store: store, listener: listener, front: front, back: back, registered: make(chan struct{}), connections: map[*net.UnixConn]bool{}, ctx: workCtx}
	fault, fe := witnessOpenFaultGate(cfg, workCtx)
	if fe != nil {
		t.Fatal("original interruption gate")
	}
	if fault != nil {
		defer fault.channel.Close()
	}
	o.fault = fault
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		for {
			c, e := listener.AcceptUnix()
			if e != nil {
				return
			}
			o.mu.Lock()
			// Evidence remains readable after failure; fresh data connections may not
			// start another connector or capture in the failed candidate.
			if o.active || o.stopping || o.failed {
				o.denied++
				o.mu.Unlock()
				c.Close()
				continue
			}
			o.active = true
			o.connections[c] = true
			o.children.Add(1)
			o.mu.Unlock()
			go o.connection(c)
		}
	}()
	go func() { <-ctx.Done(); control.Close(); listener.Close() }()
	if witnessControlWrite(control, witnessNativeReceipt{OK: true}) != nil {
		cancel()
	}
	stop := false
	for !stop {
		var command witnessNativeCommand
		if witnessControlRead(control, &command) != nil {
			o.mu.Lock()
			o.failed = true
			if o.failure == "" {
				o.failure = "control-read"
			}
			o.mu.Unlock()
			break
		}
		switch command.Operation {
		case "management":
			a := command.Actor
			o.mu.Lock()
			if o.management.PID != 0 || a.UID != cfg.ManagementUID || a.Executable != cfg.Product || a.SHA256 != cfg.ProductSHA256 || a.check(cfg.ABI) != nil {
				o.failed = true
				if o.failure == "" {
					o.failure = "management-registration"
				}
			} else {
				o.management = a
				close(o.registered)
			}
			o.mu.Unlock()
		case "helper":
			a := command.Actor
			s, e := witnessRootSocket(cfg.Back, cfg.SocketGID)
			o.mu.Lock()
			if o.active || o.cfg.Helper.check(cfg.ABI) == nil || e != nil || a.UID != 0 || a.Executable != cfg.HelperBinary || a.SHA256 != cfg.HelperSHA256 || a.check(cfg.ABI) != nil || os.SameFile(s, o.back) {
				o.failed = true
				if o.failure == "" {
					o.failure = "helper-registration"
				}
			} else {
				o.cfg.Helper = a
				o.back = s
				o.allowDeath = false
			}
			o.mu.Unlock()
		case "readback":
			o.store.mu.Lock()
			metadataID := ""
			if len(o.store.refs) > 0 {
				metadataID = o.store.refs[0].ID
			}
			o.store.mu.Unlock()
			if o.fault.reach("metadata", metadataID) != nil {
				o.mu.Lock()
				o.failed = true
				o.failure = "interruption-metadata"
				o.mu.Unlock()
			}
		case "native-death":
			o.mu.Lock()
			o.allowDeath = true
			o.mu.Unlock()
		case "stop":
			stop = true
		default:
			o.mu.Lock()
			o.failed = true
			if o.failure == "" {
				o.failure = "control-operation"
			}
			o.mu.Unlock()
		}
		if stop {
			break
		}
		if witnessControlWrite(control, o.receipt()) != nil {
			o.mu.Lock()
			o.failed = true
			if o.failure == "" {
				o.failure = "control-write"
			}
			o.mu.Unlock()
			break
		}
	}
	// Normal owner completion stops admission and drains the one existing call
	// under its original deadline while management is alive. Signal/parent loss
	// cancels/ closes immediately and retains any failure. No workload is retried.
	if !stop {
		stopWork()
	}
	cleanup := o.stop(stop)
	stopWork()
	if cleanup != nil {
		t.Error("observer owned socket cleanup")
	}
	<-acceptDone
	r := o.receipt()
	r.FrontRemoved = cleanup == nil
	if cleanup != nil {
		r.Failed = true
		if r.Failure == "" {
			r.Failure = "front-cleanup"
		}
	}
	if _, _, controllerCase := witnessControllerLabel(cfg.RunLabel); controllerCase {
		if witnessWritePrivateIndex("../../.tools", "witness-controller-final-"+cfg.RunLabel+".json", r) != nil {
			t.Error("exclusive controller-loss final receipt")
		}
	}
	if e := witnessControlWrite(control, r); e != nil && ctx.Err() == nil {
		t.Error("observer final cleanup receipt")
	}
	cancel()
}
