//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type witnessNativeFaultGate struct {
	startedNS, deadlineNS int64
	stage                 string
	channel               *net.UnixConn
	ctx                   context.Context
	once                  sync.Once
	err                   error
}

func witnessFaultStageValid(stage string) bool {
	return stage == "" || stage == "capture" || stage == "sealed" || stage == "forward" || stage == "metadata" || stage == "connect" || stage == "handoff" || stage == "ancillary"
}

type witnessFaultEvent struct {
	StartedNS, DeadlineNS int64 `json:",omitempty"`
	Stage, ID             string
	Connector             *witnessActor `json:",omitempty"`
}

// Only the sealed negative control carries the original connection window.
// Normal/timed paths have no fault gate and acquire no new metadata operation.
func (g *witnessNativeFaultGate) reachSealed(id string, started, deadline time.Time) error {
	if g == nil || g.stage != "sealed" {
		return nil
	}
	g.startedNS, g.deadlineNS = started.UnixNano(), deadline.UnixNano()
	return g.reach("sealed", id)
}

func (g *witnessNativeFaultGate) reach(stage, id string) error { return g.reachActor(stage, id, nil) }
func (g *witnessNativeFaultGate) reachActor(stage, id string, actor *witnessActor) error {
	if g == nil || g.stage != stage {
		return nil
	}
	g.once.Do(func() {
		_ = g.channel.SetDeadline(time.Now().Add(45 * time.Second))
		if witnessControlWrite(g.channel, witnessFaultEvent{Stage: stage, ID: id, Connector: actor, StartedNS: g.startedNS, DeadlineNS: g.deadlineNS}) != nil {
			g.err = errDashboardWitness
			return
		}
		done := make(chan error, 1)
		go func() {
			var ack witnessNativeCommand
			e := witnessControlRead(g.channel, &ack)
			if e != nil || ack.Operation != "release" || ack.Actor != (witnessActor{}) {
				done <- errDashboardWitness
			} else {
				done <- nil
			}
		}()
		select {
		case g.err = <-done:
		case <-g.ctx.Done():
			g.channel.Close()
			<-done
			g.err = errDashboardWitness
		}
	})
	return g.err
}
func witnessOpenFaultGate(cfg witnessNativeConfig, ctx context.Context) (*witnessNativeFaultGate, error) {
	if cfg.FaultStage == "" || cfg.FaultStage == "ancillary" {
		return nil, nil
	}
	f := os.NewFile(4, "owned-observer-interruption")
	c, e := net.FileConn(f)
	f.Close()
	if e != nil {
		return nil, errDashboardWitness
	}
	u := c.(*net.UnixConn)
	uid, pid, e := witnessNativePeer(u, cfg.ABI)
	if e != nil || uid != 0 || pid != cfg.Parent.PID {
		u.Close()
		return nil, errDashboardWitness
	}
	return &witnessNativeFaultGate{stage: cfg.FaultStage, channel: u, ctx: ctx}, nil
}

func witnessFaultLabelValid(label string) bool {
	if label == "expiry-native-small" || label == "expiry-projection-10000" || label == "shutdown-native-small" || label == "shutdown-projection-10000" || label == "forward-manager-exit" || label == "overlap-native-small" || label == "overlap-projection-10000" || label == "drain-native-small" || label == "drain-projection-10000" {
		return true
	}
	if strings.HasPrefix(label, "controller-") {
		_, _, ok := witnessControllerLabel(label)
		return ok
	}
	if strings.HasPrefix(label, "ancillary-") {
		return witnessAncillaryFaultValid(strings.TrimPrefix(label, "ancillary-")) && label != "ancillary-"
	}
	for _, stage := range []string{"capture", "sealed", "forward", "metadata", "connect", "handoff"} {
		for _, action := range []string{"sigterm", "sigkill", "channel-eof", "release"} {
			if label == stage+"-"+action && (action != "release" || stage == "connect" || stage == "handoff") {
				return true
			}
		}
	}
	return false
}
func witnessRunLabel(cfg witnessNativeConfig) string {
	if cfg.RunLabel != "" {
		return cfg.RunLabel
	}
	return cfg.Mode
}

// No fault barrier is present in normal or timed qualification runs. These
// explicit disposable cases stop the actual broker at real source/custody I/O.
func TestNativeDashboardWitnessInterruption(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("disposable VM opt-in")
	}
	if os.Getuid() != 0 {
		t.Fatal("root fixture owner required")
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
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("interruption cleanup uncertain")
		}
	})
	probeDir := "../../.tools/witness-probe"
	if os.Mkdir(probeDir, 0700) != nil {
		t.Fatal("exclusive probe")
	}
	probe := witnessNativeABI(t, probeDir)
	if witnessWritePrivateIndex("../../.tools", "witness-probe-original.json", probe) != nil {
		t.Fatal("probe original")
	}
	expected := nativeStorageFixtureObserved(t, owner.storage)
	for i := 0; i < 18; i++ {
		readerFixtureCommand(t, "/sbin/zfs", "create", "-o", "mountpoint=none", expected.Pool+fmt.Sprintf("/item-%03d", i))
	}
	count := len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
	exportRoot := "../../.tools/witness-interruptions"
	if os.Mkdir(exportRoot, 0700) != nil {
		t.Fatal("exclusive interruption export")
	}

	t.Run("normal-control", func(t *testing.T) {
		p := startWitnessPair(t, owner, probe, "native-small")
		if p.observer.faultChannel != nil {
			t.Fatal("normal broker acquired fault channel")
		}
		p.pair.session.ready(t, count, "")
		generation := p.pair.session.overview(t).Generation
		snapshot, ref, e := p.observer.original(generation)
		if e != nil || !snapshot.Complete() || !ref.Complete || !ref.Delivered || ref.RawSHA256 != ref.ForwardSHA256 {
			t.Fatal("normal original delivery/confirmation changed", e)
		}
		final := p.observer.stop(t)
		if final.ConnectorStarted != final.Connectors || final.ConnectorActive != 0 {
			t.Fatal("normal connector owner omitted")
		}
		if witnessWritePrivateIndex("../../.tools", "witness-interruption-normal.json", final) != nil {
			t.Fatal("normal control original")
		}
		t.Log("NATIVE_WITNESS_INTERRUPTION_NORMAL original_confirmed=true no_fault_channel=true all_connectors_reaped=true resource_acceptance=false")
	})
	for _, stage := range []string{"capture", "sealed", "forward", "metadata"} {
		for _, action := range []string{"sigterm", "sigkill", "channel-eof"} {
			label := stage + "-" + action
			t.Run(label, func(t *testing.T) {
				p := startWitnessPairWithFault(t, owner, probe, "native-small", stage, label)
				o := p.observer
				actor, e := witnessObserveActor(o.process.cmd.Process.Pid, probe.ABI)
				if e != nil {
					t.Fatal("original observer actor")
				}
				front, e := witnessRootSocket(o.config.Front, o.config.SocketGID)
				if e != nil {
					t.Fatal("original front socket")
				}
				lookup := make(chan error, 1)
				if stage == "metadata" {
					p.pair.session.ready(t, count, "")
					generation := p.pair.session.overview(t).Generation
					go func() { _, _, e := o.original(generation); lookup <- e }()
				}
				_ = o.faultChannel.SetReadDeadline(time.Now().Add(45 * time.Second))
				var event witnessFaultEvent
				if witnessControlRead(o.faultChannel, &event) != nil || event.Stage != stage || !dashboardGenerationPattern.MatchString(event.ID) {
					t.Fatal("actual interruption boundary not reached")
				}
				var final witnessNativeReceipt
				switch action {
				case "channel-eof":
					o.faultChannel.Close()
					if stage == "metadata" {
						select {
						case e := <-lookup:
							if e == nil {
								t.Fatal("uncertain metadata accepted")
							}
						case <-time.After(10 * time.Second):
							t.Fatal("metadata cancellation timeout")
						}
					}
					performanceWait(t, func() bool {
						final, e = o.control(witnessNativeCommand{Operation: "readback"})
						return e != nil && final.Failed
					})
					_ = o.channel.SetDeadline(time.Now().Add(10 * time.Second))
					if witnessControlWrite(o.channel, witnessNativeCommand{Operation: "stop"}) != nil || witnessControlRead(o.channel, &final) != nil || !final.Failed {
						t.Fatal("refused owner final receipt")
					}
					select {
					case <-o.process.done:
						o.process.waited = true
					case <-time.After(10 * time.Second):
						t.Fatal("fault observer stop timeout")
					}
				default:
					signal := syscall.SIGTERM
					if action == "sigkill" {
						signal = syscall.SIGKILL
					}
					o.process.stop(t, signal, false)
					if stage == "metadata" {
						select {
						case e := <-lookup:
							if e == nil {
								t.Fatal("dead owner metadata accepted")
							}
						case <-time.After(10 * time.Second):
							t.Fatal("dead metadata lookup timeout")
						}
					}
				}
				o.channel.Close()
				o.root.Close()
				o.closed = true
				o.final = final
				o.faultChannel.Close()
				if witnessLifetime(actor, probe.ABI) {
					t.Fatal("original observer remains alive")
				}
				if _, _, e := o.original(event.ID); e == nil {
					t.Fatal("dead observer original accepted")
				}
				usage, _ := o.process.cmd.ProcessState.SysUsage().(*syscall.Rusage)
				if usage == nil || usage.Maxrss <= 0 {
					t.Fatal("observer original wait usage unavailable")
				}
				cpu, ce := witnessNativeCPU(usage)
				if ce != nil || cpu <= 0 {
					t.Fatal("observer original wait CPU")
				}
				current, fe := os.Lstat(o.config.Front)
				if fe == nil {
					if action != "sigkill" || !os.SameFile(front, current) || os.Remove(o.config.Front) != nil {
						t.Fatal("front cleanup uncertainty")
					}
				} else if !os.IsNotExist(fe) {
					t.Fatal("front observation")
				}
				if _, e := witnessRootSocket(o.config.Back, o.config.SocketGID); e != nil || o.config.Helper.check(probe.ABI) != nil {
					t.Fatal("fault killed/replaced actual helper")
				}
				cfg, e := os.ReadFile(p.pair.path)
				back, be := os.ReadFile(p.backPath)
				token, te := os.ReadFile(filepath.Join(p.pair.config.StateDirectory, "token.sha256"))
				if e != nil || be != nil || te != nil || !bytes.Equal(cfg, p.configBytes) || !bytes.Equal(back, p.backBytes) || !bytes.Equal(token, p.digest) {
					t.Fatal("fault changed configuration/token")
				}
				dir := filepath.Join(exportRoot, label)
				if os.Mkdir(dir, 0700) != nil {
					t.Fatal("exclusive case export")
				}
				records := []witnessInterruptedOriginal{}
				entries, e := os.ReadDir(p.directory)
				if e != nil {
					t.Fatal("original custody census")
				}
				for _, entry := range entries {
					if entry.IsDir() || !strings.HasPrefix(entry.Name(), "original-") {
						t.Fatal("unknown interrupted original")
					}
					name := filepath.Join(p.directory, entry.Name())
					info, e := os.Lstat(name)
					object, oe := witnessObject(info)
					if e != nil || oe != nil || object.UID != 0 || object.Links != 1 || !info.Mode().IsRegular() || info.Size() > witnessRawCap {
						t.Fatal("interrupted original custody")
					}
					raw, e := os.ReadFile(name)
					after, ae := os.Lstat(name)
					if e != nil || ae != nil || !os.SameFile(info, after) || info.Size() != int64(len(raw)) || !info.ModTime().Equal(after.ModTime()) {
						t.Fatal("interrupted original changed")
					}
					out, oe := os.OpenFile(filepath.Join(dir, entry.Name()), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
					if oe != nil {
						t.Fatal("exclusive original export")
					}
					n, we := out.Write(raw)
					se := out.Sync()
					me := out.Chmod(0400)
					ce := out.Close()
					if n != len(raw) || we != nil || se != nil || me != nil || ce != nil {
						t.Fatal("sealed original export")
					}

					records = append(records, witnessInterruptedOriginal{entry.Name(), object, uint32(info.Mode().Perm()), fmt.Sprintf("%x", sha256.Sum256(raw))})
				}
				if len(records) == 0 {
					t.Fatal("interrupted original prefix lost")
				}
				receipt := witnessInterruptionReceipt{label, event, actor, o.config.Parent, probe.ABI, final, records, true, true, uint64(usage.Maxrss) * 1024, cpu, true, true}
				if witnessWritePrivateIndex(dir, "receipt.json", receipt) != nil {
					t.Fatal("original interruption receipt")
				}
				p.originalsExported = true
				t.Logf("NATIVE_WITNESS_INTERRUPTION stage=%s action=%s refused=true observer_reaped=true helper_unchanged=true originals=%d resource_acceptance=false", stage, action, len(records))
			})
		}
	}
}

type witnessInterruptedOriginal struct {
	Name   string
	Object witnessFileObject
	Mode   uint32
	SHA256 string
}
type witnessInterruptionReceipt struct {
	Label                                   string
	Event                                   witnessFaultEvent
	Observer, Controller                    witnessActor
	ABI                                     witnessABI
	Final                                   witnessNativeReceipt
	Originals                               []witnessInterruptedOriginal
	Refused, Reaped                         bool
	HighRSS                                 uint64
	CPUNS                                   int64
	HelperUnchanged, ConfigurationUnchanged bool
}
