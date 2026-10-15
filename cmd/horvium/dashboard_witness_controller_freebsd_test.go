//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type witnessControllerReady struct{ Controller, Observer witnessActor }
type witnessControllerReceipt struct {
	Indexes                                                  []witnessInterruptedOriginal
	ObserverWaitRecorded                                     bool
	Label                                                    string
	Owner, Controller, Observer                              witnessActor
	Event                                                    witnessFaultEvent
	Final                                                    witnessNativeReceipt
	Originals                                                []witnessInterruptedOriginal
	ControllerExited                                         bool
	ControllerExitCode, ControllerSignal                     int
	ObserverExited                                           bool
	ObserverExitCode, ObserverSignal                         int
	ControllerHighRSS, ObserverHighRSS                       uint64
	ControllerCPUNS, ObserverCPUNS                           int64
	Refused, Reaped, HelperUnchanged, ConfigurationUnchanged bool
}

// This actor is the observer's actual parent and sole control-channel owner.
// The separately surviving test owns product processes, storage and the reaper.
func TestNativeDashboardWitnessController(t *testing.T) {
	path := os.Getenv("HORVIUM_WITNESS_CONTROLLER")
	if path == "" {
		t.Skip("owned controller reexecution only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" || os.Getuid() != 0 {
		t.Fatal("controller authorization")
	}
	info, e := os.Lstat(path)
	object, oe := witnessObject(info)
	if e != nil || oe != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || object.UID != 0 || object.Links != 1 {
		t.Fatal("controller bootstrap custody")
	}
	f, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	var cfg witnessNativeConfig
	e = witnessControlRead(f, &cfg)
	eof := witnessEOF(f)
	f.Close()
	if e != nil || eof != nil || cfg.Parent.PID != os.Getppid() || cfg.Parent.UID != 0 || cfg.Parent.check(cfg.ABI) != nil {
		t.Fatal("controller original owner")
	}
	parent := cfg.Parent
	controlFile := os.NewFile(3, "surviving-owner")
	c, e := net.FileConn(controlFile)
	controlFile.Close()
	if e != nil {
		t.Fatal("controller channel")
	}
	control := c.(*net.UnixConn)
	defer control.Close()
	uid, pid, e := witnessNativePeer(control, cfg.ABI)
	if e != nil || uid != 0 || pid != parent.PID {
		t.Fatal("controller owner peer")
	}
	faultFile := os.NewFile(4, "surviving-owner-boundary")
	fc, e := net.FileConn(faultFile)
	faultFile.Close()
	if e != nil {
		t.Fatal("controller boundary channel")
	}
	faults := fc.(*net.UnixConn)
	defer faults.Close()
	uid, pid, e = witnessNativePeer(faults, cfg.ABI)
	if e != nil || uid != 0 || pid != parent.PID {
		t.Fatal("controller boundary peer")
	}
	self, e := witnessObserveActor(os.Getpid(), cfg.ABI)
	if e != nil || self.UID != 0 || self.Executable != cfg.Fixture || self.SHA256 != cfg.FixtureSHA256 {
		t.Fatal("controller original artifact")
	}
	cfg.Parent = self
	o := startWitnessObserver(t, cfg, strings.TrimSuffix(path, ".controller"))
	observer, e := witnessObserveActor(o.process.cmd.Process.Pid, cfg.ABI)
	if e != nil {
		t.Fatal("controller observer actor")
	}
	if witnessControlWrite(control, witnessControllerReady{self, observer}) != nil {
		t.Fatal("controller ready")
	}
	if o.faultChannel != nil {
		go func() {
			var event witnessFaultEvent
			if witnessControlRead(o.faultChannel, &event) == nil {
				_ = faults.SetWriteDeadline(time.Now().Add(45 * time.Second))
				_ = witnessControlWrite(faults, event)
			}
		}()
	}
	for {
		_ = control.SetReadDeadline(time.Now().Add(90 * time.Second))
		var command witnessNativeCommand
		if witnessControlRead(control, &command) != nil {
			t.Error("surviving owner lost")
			return
		}
		if command.Operation == "stop" {
			r := o.stop(t)
			if witnessControlWrite(control, r) != nil {
				t.Error("controller final relay")
			}
			return
		}
		r, _ := o.control(command)
		if witnessControlWrite(control, r) != nil {
			t.Error("controller receipt relay")
			return
		}
	}
}

func witnessControllerParent(a witnessActor, parent int, abi witnessABI) bool {
	raw, e := unix.SysctlRaw("kern.proc.pid", a.PID)
	if e != nil {
		return false
	}
	p, e := witnessDecodeKernelProcess(raw, abi.usageLayout())
	return e == nil && p.PID == a.PID && p.Parent == parent && p.UID == a.UID && p.Sec == a.Sec && p.Usec == a.Usec
}

func startWitnessController(t *testing.T, cfg witnessNativeConfig, path string) (*witnessFixtureObserver, witnessControllerReady) {
	t.Helper()
	f, e := os.OpenFile(path+".controller", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal("exclusive controller bootstrap")
	}
	we := witnessControlWrite(f, cfg)
	se := f.Sync()
	ce := f.Close()
	if we != nil || se != nil || ce != nil {
		t.Fatal("controller bootstrap write")
	}
	parent, child, e := witnessSocketPair()
	if e != nil {
		t.Fatal(e)
	}
	faultParent, faultChild, e := witnessSocketPair()
	if e != nil {
		t.Fatal(e)
	}
	file, e := child.File()
	child.Close()
	if e != nil {
		t.Fatal(e)
	}
	faultFile, e := faultChild.File()
	faultChild.Close()
	if e != nil {
		t.Fatal(e)
	}
	defer file.Close()
	defer faultFile.Close()
	log, e := os.OpenFile(filepath.Join("../../.tools", "witness-controller-"+cfg.RunLabel+".log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { parent.Close(); faultParent.Close(); log.Close() })
	cmd := exec.Command(cfg.Fixture, "-test.run=^TestNativeDashboardWitnessController$", "-test.count=1")
	cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "HORVIUM_WITNESS_CONTROLLER=" + path + ".controller", "LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	cmd.ExtraFiles = []*os.File{file, faultFile}
	cmd.Stdout = log
	cmd.Stderr = log
	p := &nativeProcess{cmd: cmd, done: make(chan error, 1)}
	if cmd.Start() != nil {
		t.Fatal("controller process")
	}
	file.Close()
	faultFile.Close()
	go func() { p.done <- cmd.Wait() }()
	t.Cleanup(func() {
		if !p.waited {
			p.stop(t, syscall.SIGKILL, false)
		}
	})
	_ = parent.SetReadDeadline(time.Now().Add(45 * time.Second))
	var ready witnessControllerReady
	if witnessControlRead(parent, &ready) != nil || ready.Controller.PID != cmd.Process.Pid || ready.Controller.UID != 0 || ready.Observer.UID != 0 || ready.Controller.SHA256 != cfg.FixtureSHA256 || ready.Observer.SHA256 != cfg.FixtureSHA256 || ready.Controller.check(cfg.ABI) != nil || ready.Observer.check(cfg.ABI) != nil || !witnessControllerParent(ready.Controller, os.Getpid(), cfg.ABI) || !witnessControllerParent(ready.Observer, ready.Controller.PID, cfg.ABI) {
		t.Fatal("controller/observer original lineage")
	}
	root, e := os.OpenRoot(cfg.Directory)
	if e != nil {
		t.Fatal(e)
	}
	cfg.Parent = ready.Controller
	o := &witnessFixtureObserver{config: cfg, process: p, channel: parent, faultChannel: faultParent, root: root, originals: map[string]witnessNativeOriginal{}}
	t.Cleanup(func() {
		if o.closed {
			return
		}
		if !p.waited {
			p.stop(t, syscall.SIGKILL, false)
		}
		if witnessLifetime(ready.Observer, cfg.ABI) {
			_, _ = witnessControllerWait(t, ready.Observer.PID)
		}
		// A failed assertion must not call graceful stop on an already-dead proxy.
		// Preserve an available final index before the original pair's fallback export.
		file, e := os.Open(filepath.Join("../../.tools", "witness-controller-final-"+cfg.RunLabel+".json"))
		if e == nil {
			var final witnessNativeReceipt
			if witnessControlRead(file, &final) == nil && witnessEOF(file) == nil {
				o.final = final
			}
			file.Close()
		}
		parent.Close()
		faultParent.Close()
		root.Close()
		o.failed = true
		o.closed = true
	})
	return o, ready
}

func witnessControllerWait(t *testing.T, pid int) (syscall.WaitStatus, syscall.Rusage) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		var status syscall.WaitStatus
		var usage syscall.Rusage
		got, e := syscall.Wait4(pid, &status, syscall.WNOHANG, &usage)
		if e == syscall.EINTR {
			continue
		}
		if e != nil {
			t.Fatal("adopted observer wait", e)
		}
		if got == pid {
			return status, usage
		}
		if got != 0 {
			t.Fatal("unrelated child reaped")
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("original adopted observer remains; discard guest")
	return 0, syscall.Rusage{}
}
func witnessControllerFinal(t *testing.T, label string) witnessNativeReceipt {
	t.Helper()
	path := filepath.Join("../../.tools", "witness-controller-final-"+label+".json")
	f, e := os.Open(path)
	if e != nil {
		t.Fatal("original final receipt unavailable", e)
	}
	defer f.Close()
	info, e := f.Stat()
	obj, oe := witnessObject(info)
	if e != nil || oe != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0400 || obj.UID != 0 || obj.Links != 1 || info.Size() > witnessFrameCap+4 {
		t.Fatal("final receipt custody")
	}
	var r witnessNativeReceipt
	if witnessControlRead(f, &r) != nil || witnessEOF(f) != nil {
		t.Fatal("original final frame")
	}
	after, e := f.Stat()
	if e != nil || !witnessRawUnchanged(info, after) {
		t.Fatal("original final changed")
	}
	return r
}
func witnessControllerCopy(t *testing.T, path, out string, limit int64) witnessInterruptedOriginal {
	t.Helper()
	before, e := os.Lstat(path)
	obj, oe := witnessObject(before)
	if e != nil || oe != nil || !before.Mode().IsRegular() || obj.UID != 0 || obj.Links != 1 || before.Size() > limit {
		t.Fatal("original object custody")
	}
	source, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	dest, e := os.OpenFile(filepath.Join(out, filepath.Base(path)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		source.Close()
		t.Fatal(e)
	}
	h := sha256.New()
	n, we := io.CopyBuffer(io.MultiWriter(dest, h), io.LimitReader(source, before.Size()+1), make([]byte, witnessFrameCap))
	source.Close()
	se := dest.Sync()
	me := dest.Chmod(0400)
	ce := dest.Close()
	after, ae := os.Lstat(path)
	if we != nil || se != nil || me != nil || ce != nil || ae != nil || n != before.Size() || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || before.Mode() != after.Mode() {
		t.Fatal("original bytes changed/export failed")
	}
	return witnessInterruptedOriginal{filepath.Base(path), obj, uint32(before.Mode().Perm()), fmt.Sprintf("%x", h.Sum(nil))}
}
func witnessControllerExport(t *testing.T, p *witnessNativePair, out string) ([]witnessInterruptedOriginal, []witnessInterruptedOriginal) {
	t.Helper()
	entries, e := os.ReadDir(p.directory)
	if e != nil {
		t.Fatal(e)
	}
	var records, indexes []witnessInterruptedOriginal
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "original-") {
			t.Fatal("unknown original entry")
		}
		path := filepath.Join(p.directory, entry.Name())
		if strings.HasSuffix(entry.Name(), ".wire.json") {
			indexInfo, ie := os.Lstat(path)
			indexObject, ioe := witnessObject(indexInfo)
			if ie != nil || ioe != nil || !indexInfo.Mode().IsRegular() || indexInfo.Mode().Perm() != 0400 || indexObject.UID != 0 || indexObject.Links != 1 || indexInfo.Size() > witnessFrameCap+4 {
				t.Fatal("original index custody")
			}
			f, e := os.Open(path)
			if e != nil {
				t.Fatal(e)
			}
			var record witnessNativeOriginal
			re := witnessControlRead(f, &record)
			eof := witnessEOF(f)
			f.Close()
			found := false
			for _, original := range p.observer.final.Originals {
				if reflect.DeepEqual(original, record) {
					found = true
				}
			}
			if re != nil || eof != nil || !found || record.Ref.Name+".json" != entry.Name() {
				t.Fatal("original sidecar differs from final sealed record")
			}
			copied := witnessControllerCopy(t, path, out, witnessFrameCap+4)
			if copied.Mode != 0400 || copied.Object != indexObject {
				t.Fatal("original sidecar not sealed")
			}
			indexes = append(indexes, copied)
		} else if strings.HasSuffix(entry.Name(), ".wire") {
			record := witnessControllerCopy(t, path, out, witnessRawCap)
			records = append(records, record)
			p.owner.witnessArchiveBytes += record.Object.Size
		} else {
			t.Fatal("unknown original entry")
		}
	}
	if len(records) == 0 {
		t.Fatal("original prefix lost")
	}
	p.originalsExported = true
	return records, indexes
}

func TestNativeDashboardWitnessControllerInterruption(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("disposable VM opt-in")
	}
	if os.Getuid() != 0 {
		t.Fatal("root owner required")
	}
	owner := newDashboardGuestOwner(t)
	t.Cleanup(func() {
		clean := owner.storageCleanups == 1 && dashboardReaperEmpty() == nil && reflect.DeepEqual(owner.artifacts, observeDashboardParentArtifacts(t))
		for key, dir := range owner.directories {
			if _, e := os.Lstat(dir); !os.IsNotExist(e) {
				clean = false
			}
			if _, e := os.Lstat(owner.sockets[key]); !os.IsNotExist(e) {
				clean = false
			}
		}
		if witnessWritePrivateIndex("../../.tools", "witness-controller-cleanup.json", struct{ Verified bool }{clean}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("controller owner cleanup uncertain")
		}
	})
	probeDir := "../../.tools/witness-probe"
	if os.Mkdir(probeDir, 0700) != nil {
		t.Fatal("exclusive probe")
	}
	probe := witnessNativeABI(t, probeDir)
	if witnessWritePrivateIndex("../../.tools", "witness-probe-original.json", probe) != nil {
		t.Fatal("probe receipt")
	}
	_ = nativeStorageFixtureObserved(t, owner.storage)
	count := len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
	export := "../../.tools/witness-controller-cases"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive case export")
	}
	labels := []string{"controller-normal"}
	for _, stage := range []string{"capture", "sealed", "forward", "metadata"} {
		for _, action := range []string{"sigterm", "sigkill"} {
			labels = append(labels, "controller-"+stage+"-"+action)
		}
	}
	for _, label := range labels {
		t.Run(label, func(t *testing.T) {
			stage, action, adverse := witnessControllerLabel(label)
			var ready witnessControllerReady
			p := startWitnessPairUsingObserver(t, owner, probe, "native-small", stage, label, func(t *testing.T, cfg witnessNativeConfig, path string) *witnessFixtureObserver {
				var o *witnessFixtureObserver
				o, ready = startWitnessController(t, cfg, path)
				return o
			})
			o := p.observer
			outer, e := witnessObserveActor(os.Getpid(), probe.ABI)
			if e != nil {
				t.Fatal("surviving owner actor")
			}
			r := witnessControllerReceipt{Label: label, Owner: outer, Controller: ready.Controller, Observer: ready.Observer}
			out := filepath.Join(export, label)
			if os.Mkdir(out, 0700) != nil {
				t.Fatal("exclusive case directory")
			}
			if !adverse {
				p.pair.session.ready(t, count, "")
				id := p.pair.session.overview(t).Generation
				snap, ref, e := o.original(id)
				if e != nil || !snap.Complete() || !ref.Complete || !ref.Delivered {
					t.Fatal("normal controller published original", e)
				}
				r.Final = o.stop(t)
				r.Reaped = !witnessLifetime(ready.Controller, probe.ABI) && !witnessLifetime(ready.Observer, probe.ABI)
			} else {
				lookup := make(chan error, 1)
				if stage == "metadata" {
					p.pair.session.ready(t, count, "")
					id := p.pair.session.overview(t).Generation
					go func() { _, _, e := o.original(id); lookup <- e }()
				}
				_ = o.faultChannel.SetReadDeadline(time.Now().Add(45 * time.Second))
				if witnessControlRead(o.faultChannel, &r.Event) != nil || r.Event.Stage != stage || !dashboardGenerationPattern.MatchString(r.Event.ID) {
					t.Fatal("actual controller boundary")
				}
				if ready.Controller.check(probe.ABI) != nil || ready.Observer.check(probe.ABI) != nil || !witnessControllerParent(ready.Observer, ready.Controller.PID, probe.ABI) || !witnessControllerParent(ready.Controller, outer.PID, probe.ABI) {
					t.Fatal("original process lineage before signal")
				}
				signal := syscall.SIGTERM
				if action == "sigkill" {
					signal = syscall.SIGKILL
				}
				o.process.stop(t, signal, false)
				status, ok := o.process.cmd.ProcessState.Sys().(syscall.WaitStatus)
				if !ok || !witnessControllerWaitMatches(action, status.Exited(), status.ExitStatus(), int(status.Signal())) {
					t.Error("actual controller signal wait", status)
				}
				r.ControllerExited = status.Exited()
				r.ControllerExitCode = status.ExitStatus()
				r.ControllerSignal = int(status.Signal())
				usage, ok := o.process.cmd.ProcessState.SysUsage().(*syscall.Rusage)
				if !ok || usage.Maxrss <= 0 {
					t.Fatal("controller wait usage")
				}
				r.ControllerHighRSS = uint64(usage.Maxrss) * 1024
				r.ControllerCPUNS, e = witnessNativeCPU(usage)
				if e != nil {
					t.Fatal(e)
				}
				observerStatus, observerUsage := witnessControllerWait(t, ready.Observer.PID)
				r.ObserverExited = observerStatus.Exited()
				r.ObserverExitCode = observerStatus.ExitStatus()
				if observerStatus.Signaled() {
					r.ObserverSignal = int(observerStatus.Signal())
				}
				r.ObserverWaitRecorded = true
				r.ObserverHighRSS = uint64(observerUsage.Maxrss) * 1024
				r.ObserverCPUNS, e = witnessNativeCPU(&observerUsage)
				if e != nil || observerUsage.Maxrss <= 0 || !observerStatus.Exited() || observerStatus.ExitStatus() != 1 {
					t.Fatal("adopted original observer exit", observerStatus)
				}
				r.Final = witnessControllerFinal(t, label)
				if stage == "metadata" {
					select {
					case e := <-lookup:
						if e == nil {
							t.Error("dead controller metadata accepted")
						}
					case <-time.After(10 * time.Second):
						t.Fatal("metadata cancellation join")
					}
				}
				_, e = o.control(witnessNativeCommand{Operation: "readback"})
				if e == nil {
					t.Error("dead controller control accepted")
				}
				_, _, e = o.original(r.Event.ID)
				r.Refused = e != nil
				r.Reaped = !witnessLifetime(ready.Controller, probe.ABI) && !witnessLifetime(ready.Observer, probe.ABI)
				o.channel.Close()
				o.faultChannel.Close()
				o.root.Close()
				o.closed = true
				o.final = r.Final
				if !r.Refused || !r.Reaped || !r.Final.OK || !r.Final.Failed || !r.Final.FrontRemoved || r.Final.Active || r.Final.ConnectorActive != 0 || r.Final.ConnectorStarted != r.Final.Connectors {
					t.Error("controller death did not deny/reap/clean observer")
				}
			}
			_, e = witnessRootSocket(o.config.Back, o.config.SocketGID)
			r.HelperUnchanged = e == nil && o.config.Helper.check(probe.ABI) == nil
			cfg, ce := os.ReadFile(p.pair.path)
			back, be := os.ReadFile(p.backPath)
			token, te := os.ReadFile(filepath.Join(p.pair.config.StateDirectory, "token.sha256"))
			r.ConfigurationUnchanged = ce == nil && be == nil && te == nil && bytes.Equal(cfg, p.configBytes) && bytes.Equal(back, p.backBytes) && bytes.Equal(token, p.digest)
			if !r.Reaped || !r.HelperUnchanged || !r.ConfigurationUnchanged {
				t.Error("surviving original owner/helper/configuration")
			}
			if _, e := os.Lstat(o.config.Front); !os.IsNotExist(e) {
				t.Error("original front remains")
			}
			if !adverse {
				status, ok := o.process.cmd.ProcessState.Sys().(syscall.WaitStatus)
				if !ok || !status.Exited() || status.ExitStatus() != 0 {
					t.Fatal("normal controller wait")
				}
				r.ControllerExited = true
				r.ControllerExitCode = status.ExitStatus()
				usage, ok := o.process.cmd.ProcessState.SysUsage().(*syscall.Rusage)
				if !ok || usage.Maxrss <= 0 {
					t.Fatal("normal controller wait usage")
				}
				r.ControllerHighRSS = uint64(usage.Maxrss) * 1024
				r.ControllerCPUNS, e = witnessNativeCPU(usage)
				if e != nil {
					t.Fatal(e)
				}
			}
			p.pair.management.stop(t, syscall.SIGTERM, true)
			p.pair.helper.stop(t, syscall.SIGTERM, true)
			r.Originals, r.Indexes = witnessControllerExport(t, p, out)
			if witnessWritePrivateIndex(out, "receipt.json", r) != nil {
				t.Fatal("exclusive controller case receipt")
			}
			t.Logf("NATIVE_WITNESS_CONTROLLER label=%s refused=%t actual_controller_signal=%d observer_adopted=%t owner_alive=true resource_acceptance=false", label, r.Refused, r.ControllerSignal, adverse)
		})
	}
}
