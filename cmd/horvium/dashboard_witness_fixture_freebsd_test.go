//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/inventory"
	"golang.org/x/sys/unix"
)

func witnessWritePrivateIndex(dir, name string, value any) error {
	root, e := os.OpenRoot(dir)
	if e != nil {
		return errDashboardWitness
	}
	defer root.Close()
	f, e := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return errDashboardWitness
	}
	e = witnessControlWrite(f, value)
	se := f.Sync()
	me := f.Chmod(0400)
	ce := f.Close()
	if e != nil || se != nil || me != nil || ce != nil {
		return errDashboardWitness
	}
	return nil
}

type witnessProbeReceipt struct {
	ABI             witnessABI        `json:"abi"`
	SourceSHA256    string            `json:"source_sha256"`
	CompilerSHA256  string            `json:"compiler_sha256"`
	CompilerVersion string            `json:"compiler_version"`
	HeaderSHA256    map[string]string `json:"header_sha256"`
	BinarySHA256    string            `json:"binary_sha256"`
}

func witnessNativeABI(t *testing.T, dir string) witnessProbeReceipt {
	t.Helper()
	source := filepath.Join(dir, "credential-probe.c")
	binary := filepath.Join(dir, "credential-probe")
	if os.WriteFile(source, []byte(witnessABIProbe), 0600) != nil {
		t.Fatal("native ABI source")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/cc", "-Wall", "-Wextra", "-Werror", "-std=c11", "-o", binary, source)
	cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	if cmd.Run() != nil {
		t.Fatal("actual base-header ABI probe compilation")
	}
	cmd = exec.CommandContext(ctx, binary)
	data, e := cmd.Output()
	var abi witnessABI
	if e != nil || len(data) > 4096 || json.Unmarshal(data, &abi) != nil || !abi.valid() {
		t.Fatal("actual base-header ABI probe output")
	}
	digest := func(path string) string {
		t.Helper()
		f, e := os.Open(path)
		if e != nil {
			t.Fatal("probe provenance")
		}
		defer f.Close()
		h := sha256.New()
		n, e := io.Copy(h, io.LimitReader(f, 128<<20))
		if e != nil || n == 0 || n >= 128<<20 {
			t.Fatal("probe provenance bounds")
		}
		return fmt.Sprintf("%x", h.Sum(nil))
	}
	compiler, e := filepath.EvalSymlinks("/usr/bin/cc")
	if e != nil {
		t.Fatal("probe compiler path")
	}
	version := readerFixtureCommand(t, "/usr/bin/cc", "--version")
	if len(version) > 4096 {
		t.Fatal("probe compiler version")
	}
	r := witnessProbeReceipt{ABI: abi, SourceSHA256: digest(source), CompilerSHA256: digest(compiler), CompilerVersion: strings.TrimSpace(string(version)), BinarySHA256: digest(binary), HeaderSHA256: map[string]string{}}
	for _, name := range []string{"sys/types.h", "sys/ucred.h", "sys/user.h", "sys/un.h"} {
		r.HeaderSHA256[name] = digest("/usr/include/" + name)
	}
	return r
}

type witnessFixtureObserver struct {
	mu        sync.Mutex
	config    witnessNativeConfig
	process   *nativeProcess
	channel   *net.UnixConn
	root      *os.Root
	originals map[string]witnessNativeOriginal
	closed    bool
	final     witnessNativeReceipt
}

func startWitnessObserver(t *testing.T, cfg witnessNativeConfig, configPath string) *witnessFixtureObserver {
	t.Helper()
	f, e := os.OpenFile(configPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal("observer config creation")
	}
	e = witnessControlWrite(f, cfg)
	ce := f.Close()
	if e != nil || ce != nil {
		t.Fatal("observer configuration receipt")
	}
	parent, child, e := witnessSocketPair()
	if e != nil {
		t.Fatal("observer channel")
	}
	file, e := child.File()
	child.Close()
	if e != nil {
		parent.Close()
		t.Fatal("observer inherited descriptor")
	}
	cmd := exec.Command(cfg.Fixture, "-test.run=^TestNativeDashboardWitnessObserver$", "-test.count=1")
	log, e := os.OpenFile(filepath.Join("../../.tools", "witness-observer-"+cfg.Mode+".log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal("observer private transcript")
	}
	t.Cleanup(func() { log.Close() })
	cmd.Stdout = log
	cmd.Stderr = log
	cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "HORVIUM_WITNESS_OBSERVER=" + configPath, "LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	cmd.ExtraFiles = []*os.File{file}
	p := &nativeProcess{cmd: cmd, done: make(chan error, 1)}
	if cmd.Start() != nil {
		file.Close()
		parent.Close()
		t.Fatal("observer process")
	}
	file.Close()
	go func() { p.done <- cmd.Wait() }()
	root, e := os.OpenRoot(cfg.Directory)
	if e != nil {
		parent.Close()
		p.stop(t, syscall.SIGTERM, false)
		t.Fatal("observer original directory")
	}
	o := &witnessFixtureObserver{config: cfg, process: p, channel: parent, root: root, originals: map[string]witnessNativeOriginal{}}
	t.Cleanup(func() {
		if !o.closed {
			o.stop(t)
		}
	})
	_ = parent.SetDeadline(time.Now().Add(45 * time.Second))
	var receipt witnessNativeReceipt
	if witnessControlRead(parent, &receipt) != nil || !receipt.OK || receipt.Failed {
		t.Fatal("observer readiness")
	}
	return o
}
func (o *witnessFixtureObserver) control(command witnessNativeCommand) (witnessNativeReceipt, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	var receipt witnessNativeReceipt
	if o.closed {
		return receipt, errDashboardWitness
	}
	_ = o.channel.SetDeadline(time.Now().Add(45 * time.Second))
	if witnessControlWrite(o.channel, command) != nil || witnessControlRead(o.channel, &receipt) != nil || !receipt.OK || receipt.Failed {
		return receipt, fmt.Errorf("invalid witness control: stage=%s store_failed=%t", receipt.Failure, receipt.StoreFailed)
	}
	return receipt, nil
}
func (o *witnessFixtureObserver) original(id string) (inventory.Snapshot, dashboardWitnessRef, error) {
	o.mu.Lock()
	record, exists := o.originals[id]
	o.mu.Unlock()
	if !exists {
		r, e := o.control(witnessNativeCommand{Operation: "readback"})
		if e != nil {
			return inventory.Snapshot{}, dashboardWitnessRef{}, e
		}
		for _, candidate := range r.Originals {
			if candidate.Ref.ID == id {
				if exists {
					return inventory.Snapshot{}, dashboardWitnessRef{}, errDashboardWitness
				}
				record = candidate
				exists = true
			}
		}
		if !exists || !record.Ref.Complete || !record.Ref.Delivered || record.Ref.Management == "" || record.Ref.Helper == "" || record.Object.UID != 0 || record.Object.Links != 1 {
			return inventory.Snapshot{}, dashboardWitnessRef{}, errDashboardWitness
		}
		o.mu.Lock()
		o.originals[id] = record
		o.mu.Unlock()
	}
	ref := record.Ref
	if !strings.HasPrefix(ref.Name, "original-") || strings.ContainsAny(ref.Name, "/\\") || ref.RawBytes <= 0 || ref.RawBytes > witnessRawCap || ref.RawSHA256 != ref.ForwardSHA256 {
		return inventory.Snapshot{}, dashboardWitnessRef{}, errDashboardWitness
	}
	i, e := o.root.Lstat(ref.Name)
	if e != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0400 {
		return inventory.Snapshot{}, dashboardWitnessRef{}, errDashboardWitness
	}
	object, e := witnessObject(i)
	if e != nil || object != record.Object {
		return inventory.Snapshot{}, dashboardWitnessRef{}, errDashboardWitness
	}
	f, e := o.root.Open(ref.Name)
	if e != nil {
		return inventory.Snapshot{}, dashboardWitnessRef{}, errDashboardWitness
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(i, opened) {
		return inventory.Snapshot{}, dashboardWitnessRef{}, errDashboardWitness
	}
	h := sha256.New()
	snapshot, e := dashboardDecodeWitness(io.TeeReader(io.NewSectionReader(f, 0, ref.RawBytes), h), id)
	after, ae := f.Stat()
	if e != nil || ae != nil || !witnessRawUnchanged(opened, after) || fmt.Sprintf("%x", h.Sum(nil)) != ref.RawSHA256 || !snapshot.Complete() {
		return inventory.Snapshot{}, dashboardWitnessRef{}, errDashboardWitness
	}
	return snapshot, ref, nil
}
func (o *witnessFixtureObserver) stop(t *testing.T) witnessNativeReceipt {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return o.final
	}
	_ = o.channel.SetDeadline(time.Now().Add(45 * time.Second))
	var r witnessNativeReceipt
	we := witnessControlWrite(o.channel, witnessNativeCommand{Operation: "stop"})
	re := witnessControlRead(o.channel, &r)
	o.channel.Close()
	select {
	case e := <-o.process.done:
		o.process.waited = true
		if e != nil {
			t.Error("observer foreground exit")
		}
	case <-time.After(10 * time.Second):
		o.process.stop(t, syscall.SIGTERM, false)
		t.Error("observer graceful stop timed out")
	}
	o.root.Close()
	o.closed = true
	o.final = r
	if we != nil || re != nil || !r.OK || r.Failed || !r.FrontRemoved {
		t.Error("observer original teardown receipt")
	}
	if _, e := os.Lstat(o.config.Front); !os.IsNotExist(e) {
		t.Error("observer front remains")
	}
	return r
}

type witnessNativePair struct {
	owner                          *dashboardGuestOwner
	pair                           *performancePair
	observer                       *witnessFixtureObserver
	backPath                       string
	backConfig                     admin.Config
	directory                      string
	mount                          string
	mountInfo                      os.FileInfo
	mountFS                        unix.Statfs_t
	configBytes, backBytes, digest []byte
	originalsExported              bool
}

func startWitnessPair(t *testing.T, owner *dashboardGuestOwner, probe witnessProbeReceipt, mode string) *witnessNativePair {
	t.Helper()
	c, path, token := nativeRuntimeConfig(t)
	c.IntervalSeconds, c.TimeoutSeconds = 30, 45
	back := c
	back.HelperSocket = filepath.Join(filepath.Dir(c.HelperSocket), "actual-helper.sock")
	backPath := filepath.Join(filepath.Dir(path), "actual-helper.json")
	frontBytes, _ := json.Marshal(c)
	backBytes, _ := json.Marshal(back)
	if os.WriteFile(path, frontBytes, 0644) != nil || os.WriteFile(backPath, backBytes, 0644) != nil {
		t.Fatal("witness two existing-schema configs")
	}
	owner.pair(t, mode, path, c)
	dir := filepath.Dir(path)
	binary := filepath.Join(dir, "horvium")
	built, _ := filepath.Abs("../../.tools/horvium")
	owner.copy(t, "management", built, binary)
	fixture := filepath.Join(dir, "dashboard-native.test")
	sourceFixture, _ := filepath.Abs("../../.tools/dashboard-native.test")
	owner.copy(t, "fixture", sourceFixture, fixture)
	helperBinary, helperHash := binary, owner.artifacts.Artifacts["management"]
	var helper *nativeProcess
	if mode == "native-small" {
		helper = startNativeProcess(t, binary, []string{"helper", "-config", backPath}, 0, 0)
	} else {
		helperBinary = filepath.Join(dir, "reader-fixture.test")
		sourceHelper, _ := filepath.Abs("../../.tools/reader-fixture.test")
		owner.copy(t, "helper", sourceHelper, helperBinary)
		helperHash = owner.artifacts.Artifacts["helper"]
		cmd := exec.Command(helperBinary, "-test.run=^TestNativeProjectionHelper$", "--", backPath)
		cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
		helper = &nativeProcess{cmd: cmd, done: make(chan error, 1)}
		if cmd.Start() != nil {
			t.Fatal("witness projection helper")
		}
		go func() { helper.done <- cmd.Wait() }()
		t.Cleanup(func() {
			if !helper.waited {
				helper.stop(t, syscall.SIGTERM, true)
			}
		})
	}
	performanceWait(t, func() bool { _, e := witnessRootSocket(back.HelperSocket, c.SocketGID); return e == nil })
	mount := filepath.Join(dir, "witness-tmpfs")
	if os.Mkdir(mount, 0700) != nil {
		t.Fatal("owned mount directory")
	}
	mountInfo, e := os.Lstat(mount)
	if e != nil {
		t.Fatal("mount directory receipt")
	}
	readerFixtureCommand(t, "/sbin/mount", "-t", "tmpfs", "-o", "size=335544320,mode=0700", "tmpfs", mount)
	var fs unix.Statfs_t
	if unix.Statfs(mount, &fs) != nil || string(bytes.TrimRight(fs.Fstypename[:], "\x00")) != "tmpfs" {
		t.Fatal("original witness tmpfs")
	}
	archive := filepath.Join(mount, "originals")
	if os.Mkdir(archive, 0700) != nil {
		t.Fatal("root archive")
	}
	p := &witnessNativePair{owner: owner, backPath: backPath, backConfig: back, directory: archive, mount: mount, mountInfo: mountInfo, mountFS: fs, configBytes: frontBytes, backBytes: backBytes}
	// Register cleanup immediately after mount; no successful path is required.
	t.Cleanup(func() { p.cleanup(t) })
	parent, e := witnessObserveActor(os.Getpid(), probe.ABI)
	if e != nil {
		t.Fatal("original controller actor")
	}
	helperActor, e := witnessObserveActor(helper.cmd.Process.Pid, probe.ABI)
	if e != nil {
		t.Fatal("original helper actor")
	}
	cfg := witnessNativeConfig{ABI: probe.ABI, Parent: parent, Front: c.HelperSocket, Back: back.HelperSocket, Directory: archive, Mode: mode, ManagementUID: c.ManagementUID, SocketGID: c.SocketGID, Fixture: fixture, FixtureSHA256: owner.artifacts.Artifacts["fixture"], Product: binary, ProductSHA256: owner.artifacts.Artifacts["management"], Helper: helperActor, HelperBinary: helperBinary, HelperSHA256: helperHash, PriorArchiveBytes: owner.witnessArchiveBytes}
	o := startWitnessObserver(t, cfg, filepath.Join(dir, "observer.json"))
	p.observer = o
	management := startNativeProcess(t, binary, []string{"serve", "-config", path}, c.ManagementUID, c.SocketGID)
	actor, e := witnessObserveActor(management.cmd.Process.Pid, probe.ABI)
	if e != nil {
		t.Fatal("original management actor")
	}
	if _, e = o.control(witnessNativeCommand{Operation: "management", Actor: actor}); e != nil {
		t.Fatal("observer original management registration")
	}
	p.pair = &performancePair{config: c, path: path, helper: helper, management: management}
	p.pair.session = newPerformanceSession(t, c, token)
	p.digest, e = os.ReadFile(filepath.Join(c.StateDirectory, "token.sha256"))
	if e != nil {
		t.Fatal("original token digest")
	}
	return p
}
func (p *witnessNativePair) cleanup(t *testing.T) {
	t.Helper()
	if p.pair != nil {
		if p.observer != nil && !p.observer.closed {
			p.observer.stop(t)
		}
		if !p.pair.management.waited {
			p.pair.management.stop(t, syscall.SIGTERM, true)
		}
		if !p.pair.helper.waited {
			p.pair.helper.stop(t, syscall.SIGTERM, true)
		}
	}
	var current unix.Statfs_t
	if unix.Statfs(p.mount, &current) != nil || current.Fsid != p.mountFS.Fsid || string(bytes.TrimRight(current.Fstypename[:], "\x00")) != "tmpfs" {
		t.Error("owned mount identity changed")
		return
	}
	if !p.originalsExported && p.observer != nil {
		p.export(t)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if exec.CommandContext(ctx, "/sbin/umount", p.mount).Run() != nil {
		t.Error("owned witness unmount failed")
		return
	}
	i, e := os.Lstat(p.mount)
	if e != nil || !os.SameFile(i, p.mountInfo) || os.Remove(p.mount) != nil {
		t.Error("owned mount directory cleanup")
	}
}
func (p *witnessNativePair) export(t *testing.T) {
	t.Helper()
	target := filepath.Join("../../.tools", "witness-originals-"+p.observer.config.Mode)
	if os.Mkdir(target, 0700) != nil {
		t.Error("exclusive private witness export")
		return
	}
	root, e := os.OpenRoot(p.directory)
	if e != nil {
		t.Error("original export root")
		return
	}
	defer root.Close()
	for _, record := range p.observer.final.Originals {
		ref := record.Ref
		f, e := root.Open(ref.Name)
		if e != nil {
			t.Error("original export missing")
			return
		}
		out, e := os.OpenFile(filepath.Join(target, ref.Name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			f.Close()
			t.Error("exclusive export file")
			return
		}
		h := sha256.New()
		n, we := io.CopyBuffer(io.MultiWriter(out, h), io.LimitReader(f, ref.RawBytes+1), make([]byte, witnessFrameCap))
		se := out.Sync()
		ce := out.Close()
		f.Close()
		if we != nil || se != nil || ce != nil || n != ref.RawBytes || fmt.Sprintf("%x", h.Sum(nil)) != ref.RawSHA256 || witnessWritePrivateIndex(target, ref.Name+".json", record) != nil {
			t.Error("original export custody")
			return
		}
		p.owner.witnessArchiveBytes += n
	}
	p.originalsExported = true
}

func TestNativeDashboardWitnessShort(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("requires disposable pinned performance guest")
	}
	if os.Geteuid() != 0 {
		t.Fatal("witness short gates require disposable root")
	}
	owner := newDashboardGuestOwner(t)
	// Cleanup is independently observed by the existing original owner after the
	// subtests' observer/helper/mount cleanups; it is not the full resource parent.
	t.Cleanup(func() {
		clean := owner.storageCleanups == 1 && dashboardReaperEmpty() == nil && reflect.DeepEqual(owner.artifacts, observeDashboardParentArtifacts(t))
		for mode, dir := range owner.directories {
			if _, e := os.Lstat(dir); !os.IsNotExist(e) {
				clean = false
			}
			if _, e := os.Lstat(owner.sockets[mode]); !os.IsNotExist(e) {
				clean = false
			}
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("witness short cleanup uncertain")
		}
	})
	probeDir := filepath.Join("../../.tools", "witness-probe")
	if os.Mkdir(probeDir, 0700) != nil {
		t.Fatal("exclusive retained native probe directory")
	}
	probe := witnessNativeABI(t, probeDir)
	if witnessWritePrivateIndex("../../.tools", "witness-probe-original.json", probe) != nil {
		t.Fatal("private extra native probe receipt")
	}
	expected := nativeStorageFixtureObserved(t, owner.storage)
	for i := 0; i < 98; i++ {
		readerFixtureCommand(t, "/sbin/zfs", "create", "-o", "mountpoint=none", expected.Pool+fmt.Sprintf("/item-%03d", i))
	}
	ownedTSV := readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-p", "-t", "filesystem", "-o", "name,guid,used,available,referenced", "-r", expected.Pool)
	owned, e := inventory.ParseDatasets(ownedTSV, inventory.DefaultLimits())
	if e != nil || len(owned) != 100 {
		t.Fatal("independent owned 100 GUIDs")
	}
	count := len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
	for _, mode := range []string{"native-small", "projection-10000"} {
		t.Run(mode, func(t *testing.T) {
			p := startWitnessPair(t, owner, probe, mode)
			n := count
			if mode == "projection-10000" {
				n = 10000
			}
			id := p.pair.session.ready(t, n, "")
			registry := newDashboardWitnessRegistry(p.observer.original, witnessHTTPMetadata(p.pair.session.http, p.pair.session.base))
			s, ref, e := registry.confirm(context.Background(), id)
			if e != nil {
				t.Fatal("actual delivered original / publication", e)
			}
			var inputs []inventory.Dataset
			if mode == "native-small" {
				inputs = owned
			}
			prepared, e := newDashboardPreparedFixture(mode, s, inputs)
			if e != nil {
				t.Fatal("independent native declaration", e)
			}
			predicate := registry.predicate(prepared.plan)
			for _, request := range prepared.plan[:10] {
				response, e := p.pair.session.http.Get(p.pair.session.base + request.Path)
				if e != nil {
					t.Fatal("short HTML transport")
				}
				body, e := dashboardReadBody(response.Body)
				if e != nil || predicate(request, response.StatusCode, body) != nil {
					t.Fatal("actual original exact HTML")
				}
				changed := bytes.Replace(body, []byte(" B</td>"), []byte("9 B</td>"), 1)
				if request.Case != "overview" && (bytes.Equal(body, changed) || predicate(request, response.StatusCode, changed) == nil) {
					t.Fatal("actual numeric corruption accepted")
				}
			}
			p.pair.session.refresh(t)
			next := p.pair.session.ready(t, n, id)
			nextSnapshot, nextRef, e := registry.confirm(context.Background(), next)
			if e != nil || next == id || nextRef.ID != next || nextRef.RawSHA256 == ref.RawSHA256 {
				t.Fatal("explicit new original lineage", e)
			}
			if mode == "native-small" {
				if _, e := dashboardValueChanges(s, nextSnapshot, expected.PoolGUID, owned); e != nil {
					t.Fatal("owned identity census drift")
				}
			}
			if mode == "projection-10000" {
				if _, e := newDashboardPreparedFixture(mode, nextSnapshot, nil); e != nil {
					t.Fatal("projection values changed")
				}
			}
			if mode == "native-small" {
				if _, e := p.observer.control(witnessNativeCommand{Operation: "native-death"}); e != nil {
					t.Fatal("owned native death registration")
				}
				socket, e := os.Lstat(p.backConfig.HelperSocket)
				if e != nil {
					t.Fatal("actual helper socket receipt")
				}
				probe := freezeReaderProbe(t, p.pair)
				before := p.pair.session.overview(t).Generation
				p.pair.helper.stop(t, syscall.SIGKILL, false)
				readerProcessStopped(t, probe.command)
				readerProcessStopped(t, probe.guard)
				performanceWait(t, func() bool { o := p.pair.session.overview(t); return o.Generation == before && o.Datasets.Stale })
				if _, _, e := registry.confirm(context.Background(), before); e != nil {
					t.Fatal("stale retained original lost")
				}
				performanceWait(t, func() bool {
					r, e := p.observer.control(witnessNativeCommand{Operation: "readback"})
					return e == nil && !r.Active
				})
				current, e := os.Lstat(p.backConfig.HelperSocket)
				if e != nil || !os.SameFile(socket, current) || os.Remove(p.backConfig.HelperSocket) != nil {
					t.Fatal("owned actual stale socket")
				}
				if _, e := witnessRootSocket(p.pair.config.HelperSocket, p.pair.config.SocketGID); e != nil {
					t.Fatal("observer front must survive actual helper death")
				}
				helper := startNativeProcess(t, p.observer.config.Product, []string{"helper", "-config", p.backPath}, 0, 0)
				p.pair.helper = helper
				performanceWait(t, func() bool {
					_, e := witnessRootSocket(p.backConfig.HelperSocket, p.pair.config.SocketGID)
					return e == nil
				})
				actor, e := witnessObserveActor(helper.cmd.Process.Pid, p.observer.config.ABI)
				if e != nil {
					t.Fatal("new original helper actor")
				}
				if _, e := p.observer.control(witnessNativeCommand{Operation: "helper", Actor: actor}); e != nil {
					t.Fatal("new helper epoch registration")
				}
				p.pair.session.refresh(t)
				recovered := p.pair.session.ready(t, n, before)
				_, recoveredRef, e := registry.confirm(context.Background(), recovered)
				if e != nil || recoveredRef.Helper == nextRef.Helper || recoveredRef.Management != nextRef.Management {
					t.Fatal("new helper lineage / preserved manager", e)
				}
			}
			// Same UID, unrelated original process must not borrow management PID.
			cmd := exec.Command(p.observer.config.Fixture, "-test.run=^TestNativeDashboardWitnessImpostor$", "-test.count=1")
			cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "HORVIUM_WITNESS_IMPOSTOR=" + p.pair.config.HelperSocket}
			cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: p.pair.config.ManagementUID, Gid: p.pair.config.SocketGID, Groups: []uint32{p.pair.config.SocketGID}}}
			if cmd.Run() != nil {
				t.Fatal("same-UID unrelated peer was not denied")
			}
			r, e := p.observer.control(witnessNativeCommand{Operation: "readback"})
			if e != nil || r.Denied != 1 || r.Connectors < 2 || r.ConnectorMaxRSS == 0 || r.ConnectorCPUNS <= 0 {
				t.Fatal("kernel identity / connector receipts", e)
			}
			p.observer.stop(t)
			p.pair.management.stop(t, syscall.SIGTERM, true)
			p.pair.helper.stop(t, syscall.SIGTERM, true)
			if b, e := os.ReadFile(p.pair.path); e != nil || !bytes.Equal(b, p.configBytes) {
				t.Fatal("front config changed")
			}
			if b, e := os.ReadFile(p.backPath); e != nil || !bytes.Equal(b, p.backBytes) {
				t.Fatal("back config changed")
			}
			if b, e := os.ReadFile(filepath.Join(p.pair.config.StateDirectory, "token.sha256")); e != nil || !bytes.Equal(b, p.digest) {
				t.Fatal("token digest changed")
			}
			if _, e := os.Lstat(p.backConfig.HelperSocket); !os.IsNotExist(e) {
				t.Fatal("actual helper socket remains")
			}
			p.export(t)
			b, _ := json.Marshal(struct {
				Mode               string `json:"mode"`
				Originals          int    `json:"originals"`
				Controls           int    `json:"metadata_controls"`
				ConnectorPeak      uint64 `json:"connector_peak_rss"`
				ConnectorCPUNS     int64  `json:"connector_cpu_ns"`
				NativeAcceptance   bool   `json:"native_acceptance"`
				ResourceAcceptance bool   `json:"resource_acceptance"`
			}{mode, len(r.Originals), registry.controls(), r.ConnectorMaxRSS, r.ConnectorCPUNS, false, false})
			t.Log("NATIVE_DASHBOARD_WITNESS_SHORT " + string(b))
		})
	}

	_ = probe
}
func TestNativeDashboardWitnessImpostor(t *testing.T) {
	path := os.Getenv("HORVIUM_WITNESS_IMPOSTOR")
	if path == "" {
		t.Skip("owned impostor control only")
	}
	if os.Geteuid() == 0 || os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Fatal("impostor fixture scope")
	}
	c, e := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if e != nil {
		t.Fatal("impostor must reach owned listener")
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	request := witnessFrames([]string{`{"version":1,"request_id":"` + witnessTestID + `","operation":"inventory"}`})
	_, _ = c.Write(request)
	_ = c.CloseWrite()
	var b [1]byte
	n, e := c.Read(b[:])
	if n != 0 || e == nil {
		t.Fatal("impostor received accepted reply")
	}
}
