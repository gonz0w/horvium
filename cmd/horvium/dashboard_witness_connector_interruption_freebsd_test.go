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
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

type witnessConnectorBoundaryReceipt struct {
	witnessConnectorWaitRecord
	Actor                                              witnessActor
	Stage, Action                                      string
	Accounted, ChannelClosed, WaitFailed, ContextEnded bool
}

// Only explicitly configured disposable cases pause a live connector. Normal
// runs still exit the connector before the first producer request/capture.
func (o *witnessNativeObserver) interruptConnector(stage, id string, actor witnessActor, channel *net.UnixConn, cmd *exec.Cmd, ctx context.Context, cancel context.CancelFunc, waited *bool) bool {
	if o.fault == nil || o.fault.stage != stage {
		return false
	}
	e := o.fault.reachActor(stage, id, &actor)
	if e == nil && strings.HasSuffix(o.cfg.RunLabel, "-release") {
		return false
	}
	closed := false
	if e != nil {
		channel.Close()
		closed = true
		cancel()
	} else if strings.HasSuffix(o.cfg.RunLabel, "-channel-eof") {
		channel.Close()
		closed = true
	}
	waitError := cmd.Wait()
	*waited = true
	accounted := o.accountConnector(cmd, "interruption-"+stage, waitError, ctx.Err()) == nil
	o.recordConnectorBoundary(actor, cmd, accounted, closed, waitError, ctx.Err())
	return true
}
func (o *witnessNativeObserver) recordConnectorBoundary(actor witnessActor, cmd *exec.Cmd, accounted, closed bool, waitError, contextError error) {
	if o.fault == nil || (o.fault.stage != "connect" && o.fault.stage != "handoff") {
		return
	}
	r := &witnessConnectorBoundaryReceipt{Actor: actor, Stage: o.fault.stage, Action: strings.TrimPrefix(o.cfg.RunLabel, o.fault.stage+"-"), Accounted: accounted, ChannelClosed: closed, WaitFailed: waitError != nil, ContextEnded: contextError != nil}
	if cmd.ProcessState != nil {
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			signal := 0
			if status.Signaled() {
				signal = int(status.Signal())
			}
			r.witnessConnectorWaitRecord = witnessConnectorWaitRepresentation(true, status.Exited(), status.ExitStatus(), signal, uint32(status))
		}
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	if o.connectorBoundary == nil {
		o.connectorBoundary = r
		if witnessWritePrivateIndex("../../.tools", "witness-connector-wait-"+o.cfg.RunLabel+".json", r) != nil {
			o.failed = true
			if o.failure == "" {
				o.failure = "connector-wait-custody"
			}
		}
	}
}

type witnessConnectorCaseReceipt struct {
	Label                                                              string
	Event                                                              witnessFaultEvent
	Observer, Controller                                               witnessActor
	ABI                                                                witnessABI
	Final                                                              witnessNativeReceipt
	Refused, Released, Reaped, HelperUnchanged, ConfigurationUnchanged bool
}

func TestNativeDashboardWitnessConnectorInterruption(t *testing.T) {
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
			t.Error("connector interruption cleanup uncertain")
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
	export := "../../.tools/witness-connector-interruptions"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive case export")
	}
	t.Run("normal-control", func(t *testing.T) {
		p := startWitnessPair(t, owner, probe, "native-small")
		if p.observer.faultChannel != nil {
			t.Fatal("uninjected control has fault channel")
		}
		p.pair.session.ready(t, count, "")
		generation := p.pair.session.overview(t).Generation
		snapshot, ref, e := p.observer.original(generation)
		if e != nil || !snapshot.Complete() || !ref.Complete || !ref.Delivered || ref.RawSHA256 != ref.ForwardSHA256 {
			t.Fatal("uninjected original confirmation", e)
		}
		final := p.observer.stop(t)
		if final.ConnectorBoundary != nil || final.ConnectorStarted != final.Connectors || final.ConnectorActive != 0 {
			t.Fatal("uninjected connector accounting")
		}
		if witnessWritePrivateIndex("../../.tools", "witness-connector-normal.json", final) != nil {
			t.Fatal("exclusive normal receipt")
		}
		t.Log("NATIVE_WITNESS_CONNECTOR_NORMAL original_confirmed=true no_fault_channel=true all_connectors_reaped=true resource_acceptance=false")
	})
	for _, stage := range []string{"connect", "handoff"} {
		for _, action := range []string{"release", "sigterm", "sigkill", "channel-eof"} {
			label := stage + "-" + action
			t.Run(label, func(t *testing.T) {
				p := startWitnessPairWithFault(t, owner, probe, "native-small", stage, label)
				o := p.observer
				observer, e := witnessObserveActor(o.process.cmd.Process.Pid, probe.ABI)
				if e != nil {
					t.Fatal("original observer")
				}
				var event witnessFaultEvent
				_ = o.faultChannel.SetReadDeadline(time.Now().Add(45 * time.Second))
				if witnessControlRead(o.faultChannel, &event) != nil || event.Stage != stage || !dashboardGenerationPattern.MatchString(event.ID) || event.Connector == nil {
					t.Fatal("actual connector boundary not reached")
				}
				actor := *event.Connector
				if actor.PID == observer.PID || actor.UID != o.config.ManagementUID || actor.Executable != o.config.Fixture || actor.SHA256 != o.config.FixtureSHA256 || !witnessLifetime(actor, probe.ABI) {
					t.Fatal("original credential-dropped connector binding")
				}
				switch action {
				case "sigterm", "sigkill":
					signal := syscall.SIGTERM
					if action == "sigkill" {
						signal = syscall.SIGKILL
					}
					if syscall.Kill(actor.PID, signal) != nil {
						t.Fatal("signal original connector")
					}
				}
				if witnessControlWrite(o.faultChannel, witnessNativeCommand{Operation: "release"}) != nil {
					t.Fatal("release original observer boundary")
				}
				released := action == "release"
				var final witnessNativeReceipt
				if released {
					p.pair.session.ready(t, count, "")
					generation := p.pair.session.overview(t).Generation
					snapshot, ref, e := o.original(generation)
					if e != nil || !snapshot.Complete() || !ref.Complete || !ref.Delivered || ref.RawSHA256 != ref.ForwardSHA256 {
						t.Fatal("released actual original not confirmed", e)
					}
					final = o.stop(t)
				} else {
					performanceWait(t, func() bool {
						final, e = o.control(witnessNativeCommand{Operation: "readback"})
						return e != nil && final.Failed && final.ConnectorActive == 0 && final.ConnectorBoundary != nil
					})
					if _, _, e := o.original(event.ID); e == nil {
						t.Fatal("interrupted connector accepted original")
					}
					_ = o.channel.SetDeadline(time.Now().Add(10 * time.Second))
					if witnessControlWrite(o.channel, witnessNativeCommand{Operation: "stop"}) != nil || witnessControlRead(o.channel, &final) != nil || !final.Failed || !final.FrontRemoved {
						t.Fatal("failed observer final receipt")
					}
					select {
					case e := <-o.process.done:
						o.process.waited = true
						if e != nil {
							t.Fatal("failed owner stop")
						}
					case <-time.After(10 * time.Second):
						t.Fatal("failed owner stop timeout")
					}
					o.channel.Close()
					o.root.Close()
					o.closed = true
					o.final = final
					entries, e := os.ReadDir(p.directory)
					if e != nil || len(entries) != 0 || len(final.Originals) != 0 {
						t.Fatal("pre-capture interruption created/accepted an original")
					}
				}
				o.faultChannel.Close()
				original := witnessConnectorCaseReceipt{Label: label, Event: event, Observer: observer, Controller: o.config.Parent, ABI: probe.ABI, Final: final}
				if witnessWritePrivateIndex("../../.tools", "witness-connector-original-"+label+".json", original) != nil {
					t.Fatal("exclusive original final context")
				}
				r := final.ConnectorBoundary
				if r == nil || r.Actor != actor || r.Stage != stage || r.Action != action || !r.Accounted || !r.Present || r.ContextEnded || r.WaitFailed != !released || !witnessConnectorWaitMatches(action, r.Exited, r.ExitCode, r.Signal) || r.ChannelClosed != (action == "channel-eof") {
					t.Fatal("original connector wait outcome mismatch")
				}
				if witnessLifetime(actor, probe.ABI) || witnessLifetime(observer, probe.ABI) || final.ConnectorStarted != final.Connectors || final.ConnectorActive != 0 || final.ConnectorMaxRSS == 0 || final.ConnectorCPUNS <= 0 {
					t.Fatal("connector original reap/accounting incomplete")
				}
				if _, e := os.Lstat(o.config.Front); !os.IsNotExist(e) {
					t.Fatal("front socket remains")
				}
				if _, e := witnessRootSocket(o.config.Back, o.config.SocketGID); e != nil || o.config.Helper.check(probe.ABI) != nil {
					t.Fatal("actual helper changed")
				}
				cfg, e := os.ReadFile(p.pair.path)
				back, be := os.ReadFile(p.backPath)
				token, te := os.ReadFile(filepath.Join(p.pair.config.StateDirectory, "token.sha256"))
				if e != nil || be != nil || te != nil || !bytes.Equal(cfg, p.configBytes) || !bytes.Equal(back, p.backBytes) || !bytes.Equal(token, p.digest) {
					t.Fatal("configuration/token changed")
				}
				receipt := witnessConnectorCaseReceipt{label, event, observer, o.config.Parent, probe.ABI, final, !released, released, true, true, true}
				if witnessWritePrivateIndex(export, label+".json", receipt) != nil {
					t.Fatal("original connector case receipt")
				}
				t.Logf("NATIVE_WITNESS_CONNECTOR_INTERRUPTION stage=%s action=%s refused=%t released=%t original_wait=true all_connectors_reaped=true resource_acceptance=false", stage, action, !released, released)
			})
		}
	}
}
