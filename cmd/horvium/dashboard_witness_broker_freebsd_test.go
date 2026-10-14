//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
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
#include <stddef.h>
#include <stdio.h>
int main(void) {
 unsigned int endian=1;
 printf("{\"cred_size\":%zu,\"cred_version\":%u,\"version_offset\":%zu,\"uid_offset\":%zu,\"pid_offset\":%zu,\"proc_size\":%zu,\"proc_pid\":%zu,\"proc_uid\":%zu,\"proc_sec\":%zu,\"proc_usec\":%zu,\"little_endian\":%s,\"pid_size\":%zu,\"uid_size\":%zu,\"time_size\":%zu}\n",
 sizeof(struct xucred), XUCRED_VERSION, offsetof(struct xucred, cr_version), offsetof(struct xucred, cr_uid), offsetof(struct xucred, cr_pid), sizeof(struct kinfo_proc), offsetof(struct kinfo_proc, ki_pid), offsetof(struct kinfo_proc, ki_uid), offsetof(struct kinfo_proc, ki_start.tv_sec), offsetof(struct kinfo_proc, ki_start.tv_usec), *(unsigned char*)&endian ? "true":"false", sizeof(pid_t), sizeof(uid_t), sizeof(time_t));
 return 0;
}
`

type witnessABI struct {
	CredSize      int    `json:"cred_size"`
	CredVersion   uint32 `json:"cred_version"`
	VersionOffset int    `json:"version_offset"`
	UIDOffset     int    `json:"uid_offset"`
	PIDOffset     int    `json:"pid_offset"`
	ProcSize      int    `json:"proc_size"`
	ProcPID       int    `json:"proc_pid"`
	ProcUID       int    `json:"proc_uid"`
	ProcSec       int    `json:"proc_sec"`
	ProcUsec      int    `json:"proc_usec"`
	LittleEndian  bool   `json:"little_endian"`
	PIDSize       int    `json:"pid_size"`
	UIDSize       int    `json:"uid_size"`
	TimeSize      int    `json:"time_size"`
}

func (a witnessABI) valid() bool {
	if runtime.GOARCH != "amd64" || !a.LittleEndian || a.CredVersion != 0 || a.CredSize < 12 || a.CredSize > 256 || a.ProcSize < 32 || a.ProcSize > 4096 || a.PIDSize != 4 || a.UIDSize != 4 || a.TimeSize != 8 {
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

// Private fixture IPC is one bounded framed control at a time. The channel is
// inherited from an owned root parent; it is never an externally bound listener.
func witnessControlWrite(w io.Writer, v any) error {
	b, e := json.Marshal(v)
	if e != nil || len(b) == 0 || len(b) > witnessFrameCap {
		return errDashboardWitness
	}
	var h [4]byte
	binary.BigEndian.PutUint32(h[:], uint32(len(b)))
	if n, e := w.Write(h[:]); e != nil || n != 4 {
		return errDashboardWitness
	}
	if n, e := w.Write(b); e != nil || n != len(b) {
		return errDashboardWitness
	}
	return nil
}
func witnessControlRead(r io.Reader, v any) error {
	var budget witnessWireBudget
	b, e := budget.frame(r)
	if e != nil {
		return e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if _, e := witnessJSONValue(d, 0); e != nil {
		return e
	}
	if _, e := d.Token(); e != io.EOF {
		return errDashboardWitness
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return errDashboardWitness
	}
	return nil
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
	u := channel.(*net.UnixConn)
	_ = u.SetDeadline(time.Now().Add(10 * time.Second))
	if n, on, e := u.WriteMsgUnix([]byte{'F'}, unix.UnixRights(int(f.Fd())), nil); e != nil || n != 1 || on != len(unix.UnixRights(int(f.Fd()))) {
		t.Fatal("connector handoff")
	}
	var ack [1]byte
	if _, e := io.ReadFull(u, ack[:]); e != nil || ack[0] != 'A' {
		t.Fatal("connector acknowledgment")
	}
}

type witnessNativeConfig struct {
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
	OK              bool                    `json:"ok"`
	Failed          bool                    `json:"failed"`
	Denied          int                     `json:"denied"`
	Originals       []witnessNativeOriginal `json:"originals"`
	ConnectorMaxRSS uint64                  `json:"connector_max_rss"`
	ConnectorCPUNS  int64                   `json:"connector_cpu_ns"`
	Connectors      int                     `json:"connectors"`
	FrontRemoved    bool                    `json:"front_removed"`
	Active          bool                    `json:"active"`
}
type witnessNativeObserver struct {
	mu           sync.Mutex
	cfg          witnessNativeConfig
	store        *dashboardWitnessStore
	listener     *net.UnixListener
	front        os.FileInfo
	back         os.FileInfo
	management   witnessActor
	registered   chan struct{}
	active       bool
	stopping     bool
	allowDeath   bool
	denied       int
	connections  map[*net.UnixConn]bool
	children     sync.WaitGroup
	connectorMax uint64
	connectorCPU time.Duration
	connectors   int
	failed       bool
	ctx          context.Context
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
	fail := func() { o.mu.Lock(); o.failed = true; o.mu.Unlock() }
	request, e := io.ReadAll(io.LimitReader(c, witnessFrameCap+5))
	if e != nil || len(request) > witnessFrameCap+4 {
		fail()
		return
	}
	id, e := dashboardWitnessRequest(bytes.NewReader(request))
	if e != nil {
		fail()
		return
	}
	socket, e := witnessRootSocket(o.cfg.Back, o.cfg.SocketGID)
	if e != nil || !os.SameFile(socket, back) || helper.check(o.cfg.ABI) != nil {
		if !allowDeath {
			fail()
		}
		return
	}
	parent, child, e := witnessSocketPair()
	if e != nil {
		fail()
		return
	}
	defer parent.Close()
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
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: o.cfg.ManagementUID, Gid: o.cfg.SocketGID, Groups: []uint32{o.cfg.SocketGID}}}
	if cmd.Start() != nil {
		file.Close()
		fail()
		return
	}
	file.Close()
	waited := false
	defer func() {
		if !waited {
			cancel()
			_ = cmd.Wait()
		}
	}()
	_ = parent.SetDeadline(time.Now().Add(10 * time.Second))
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
	uid, pid, e = witnessNativePeer(actual, o.cfg.ABI)
	if e != nil || uid != helper.UID || pid != helper.PID || helper.check(o.cfg.ABI) != nil {
		fail()
		return
	}
	if n, e := parent.Write([]byte{'A'}); e != nil || n != 1 {
		fail()
		return
	}
	e = cmd.Wait()
	waited = true
	if e != nil {
		fail()
		return
	}
	u, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage)
	if !ok || u.Maxrss <= 0 {
		fail()
		return
	}
	o.mu.Lock()
	o.connectorMax = max(o.connectorMax, uint64(u.Maxrss)*1024)
	o.connectorCPU += time.Duration(u.Utime.Sec+u.Stime.Sec)*time.Second + time.Duration(u.Utime.Usec+u.Stime.Usec)*time.Microsecond
	o.connectors++
	o.mu.Unlock()
	_ = actual.SetDeadline(deadline)
	if n, e := actual.Write(request); e != nil || n != len(request) || actual.CloseWrite() != nil {
		fail()
		return
	}
	ref, e := o.store.capture(actual, id, o.cfg.Mode, management.incarnation(), helper.incarnation())
	if e != nil {
		fail()
		return
	}
	if ref.Complete && helper.check(o.cfg.ABI) != nil {
		fail()
		return
	}
	if o.store.forward(ref, c) != nil {
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
func (o *witnessNativeObserver) receipt() witnessNativeReceipt {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.store.mu.Lock()
	defer o.store.mu.Unlock()
	r := witnessNativeReceipt{OK: true, Failed: o.failed || o.store.failed, Denied: o.denied, ConnectorMaxRSS: o.connectorMax, ConnectorCPUNS: int64(o.connectorCPU), Connectors: o.connectors, Active: o.active}
	for _, ref := range o.store.refs {
		v := o.store.sealed[ref.Name]
		object, e := witnessObject(v.info)
		if e != nil || o.store.check(v) != nil {
			r.Failed = true
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
	if e != nil || !cfg.ABI.valid() || cfg.Parent.check(cfg.ABI) != nil || cfg.Parent.UID != 0 || cfg.ManagementUID == 0 || cfg.Front == cfg.Back || !filepath.IsAbs(cfg.Front) || !filepath.IsAbs(cfg.Back) || cfg.Helper.UID != 0 || cfg.Helper.Executable != cfg.HelperBinary || cfg.Helper.SHA256 != cfg.HelperSHA256 {
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
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		for {
			c, e := listener.AcceptUnix()
			if e != nil {
				return
			}
			o.mu.Lock()
			if o.active || o.stopping {
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
			break
		}
		switch command.Operation {
		case "management":
			a := command.Actor
			o.mu.Lock()
			if o.management.PID != 0 || a.UID != cfg.ManagementUID || a.Executable != cfg.Product || a.SHA256 != cfg.ProductSHA256 || a.check(cfg.ABI) != nil {
				o.failed = true
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
			} else {
				o.cfg.Helper = a
				o.back = s
				o.allowDeath = false
			}
			o.mu.Unlock()
		case "readback":
		case "native-death":
			o.mu.Lock()
			o.allowDeath = true
			o.mu.Unlock()
		case "stop":
			stop = true
		default:
			o.mu.Lock()
			o.failed = true
			o.mu.Unlock()
		}
		if stop {
			break
		}
		if witnessControlWrite(control, o.receipt()) != nil {
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
	}
	if e := witnessControlWrite(control, r); e != nil && ctx.Err() == nil {
		t.Error("observer final cleanup receipt")
	}
	cancel()
}
