//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"errors"
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

type witnessOverlapPeer struct {
	PID, Exit int
	UID       uint32
	ElapsedNS int64
}
type witnessOverlapCase struct {
	Label, Mode, Publication                        string
	Owner, Observer, Management, Helper             witnessActor
	Before, Rejected, Final                         witnessNativeReceipt
	Peers                                           []witnessOverlapPeer
	ListenerClosedBeforeRelease                     bool
	ClosureProbes                                   int
	Reaped, HelperUnchanged, ConfigurationUnchanged bool
	Originals, Indexes                              []witnessInterruptedOriginal
}

// These are actual broker/caller controls on the unchanged product artifacts.
// An unrelated same-UID peer is not an overlapping authenticated manager.
func TestNativeDashboardWitnessOverlapAndDrain(t *testing.T) {
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
		if witnessWritePrivateIndex("../../.tools", "witness-overlap-cleanup.json", struct{ Verified bool }{clean}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("overlap/drain cleanup uncertain")
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
	nativeCount := len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
	export := "../../.tools/witness-overlap-cases"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive case export")
	}
	for _, action := range []string{"overlap", "drain"} {
		for _, mode := range []string{"native-small", "projection-10000"} {
			label := action + "-" + mode
			t.Run(label, func(t *testing.T) {
				p := startWitnessPairWithFault(t, owner, probe, mode, "sealed", label)
				o := p.observer
				t.Cleanup(func() {
					if !o.closed {
						diagnostic, _ := o.control(witnessNativeCommand{Operation: "readback"})
						if witnessWritePrivateIndex(export, label+"-failure.json", diagnostic) != nil {
							t.Error("failure diagnostic")
						}
					}
				})
				r := witnessOverlapCase{Label: label, Mode: mode, Owner: o.config.Parent, Helper: o.config.Helper}
				var e error
				r.Observer, e = witnessObserveActor(o.process.cmd.Process.Pid, probe.ABI)
				if e != nil {
					t.Fatal(e)
				}
				r.Management, e = witnessObserveActor(p.pair.management.cmd.Process.Pid, probe.ABI)
				if e != nil {
					t.Fatal(e)
				}
				_ = o.faultChannel.SetReadDeadline(time.Now().Add(45 * time.Second))
				var event witnessFaultEvent
				if witnessControlRead(o.faultChannel, &event) != nil || event.Stage != "sealed" || !dashboardGenerationPattern.MatchString(event.ID) {
					t.Fatal("actual sealed boundary")
				}
				r.Before, e = o.control(witnessNativeCommand{Operation: "readback"})
				if e != nil || !r.Before.Active || r.Before.Failed || len(r.Before.Originals) != 1 || r.Before.ConnectorStarted != 1 || r.Before.Connectors != 1 || r.Before.ConnectorActive != 0 {
					t.Fatal("original call not held after connector reaping")
				}
				first := r.Before.Originals[0].Ref
				if first.ID != event.ID || !first.Complete || first.Delivered || first.Forward != nil {
					t.Fatal("sealed source boundary")
				}
				for range 2 {
					cmd := exec.Command(o.config.Fixture, "-test.run=^TestNativeDashboardWitnessImpostor$", "-test.count=1")
					cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "HORVIUM_WITNESS_IMPOSTOR=" + p.pair.config.HelperSocket}
					cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: p.pair.config.ManagementUID, Gid: p.pair.config.SocketGID, Groups: []uint32{p.pair.config.SocketGID}}}
					started := time.Now()
					output, e := cmd.CombinedOutput()
					if e != nil {
						t.Fatal("overlapping same-UID peer was not immediately denied", e, string(output))
					}
					r.Peers = append(r.Peers, witnessOverlapPeer{PID: cmd.Process.Pid, Exit: cmd.ProcessState.ExitCode(), UID: p.pair.config.ManagementUID, ElapsedNS: time.Since(started).Nanoseconds()})
				}
				r.Rejected, e = o.control(witnessNativeCommand{Operation: "readback"})
				if e != nil || !r.Rejected.Active || r.Rejected.Failed || r.Rejected.Denied != r.Before.Denied+2 || r.Rejected.ConnectorStarted != 1 || r.Rejected.Connectors != 1 || r.Rejected.ConnectorActive != 0 || !reflect.DeepEqual(r.Before.Originals, r.Rejected.Originals) {
					t.Fatal("overlap queued work, changed custody or latched failure")
				}
				if action == "drain" {
					// A single stop request; probes observe admission closure, never request work.
					// Keep the original 45-second collection deadline and all native caps.
					if witnessControlWrite(o.channel, witnessNativeCommand{Operation: "stop"}) != nil {
						t.Fatal("original stop request")
					}
					deadline := time.Now().Add(10 * time.Second)
					for time.Now().Before(deadline) {
						c, e := net.DialTimeout("unix", o.config.Front, time.Second)
						r.ClosureProbes++
						if e != nil {
							if !os.IsNotExist(e) && !isWitnessConnectionRefused(e) {
								t.Fatal("unproven listener closure", e)
							}
							r.ListenerClosedBeforeRelease = true
							break
						}
						c.Close()
						time.Sleep(10 * time.Millisecond)
					}
					if !r.ListenerClosedBeforeRelease || !witnessLifetime(r.Observer, probe.ABI) {
						t.Fatal("observer did not remain alive draining original call")
					}
				}
				if witnessControlWrite(o.faultChannel, witnessNativeCommand{Operation: "release"}) != nil {
					t.Fatal("release sealed original")
				}
				if action == "overlap" {
					count := nativeCount
					if mode == "projection-10000" {
						count = 10000
					}
					r.Publication = p.pair.session.ready(t, count, "")
					snapshot, ref, e := o.original(r.Publication)
					if e != nil || !snapshot.Complete() || ref.ID != event.ID || !ref.Delivered {
						t.Fatal("accepted original did not publish after overlap", e)
					}
					r.Final = o.stop(t)
				} else {
					_ = o.channel.SetReadDeadline(time.Now().Add(45 * time.Second))
					if witnessControlRead(o.channel, &r.Final) != nil || !r.Final.OK || r.Final.Failed || !r.Final.FrontRemoved {
						t.Fatal("graceful drain final receipt")
					}
					select {
					case e := <-o.process.done:
						o.process.waited = true
						if e != nil {
							t.Fatal("drain wait", e)
						}
					case <-time.After(10 * time.Second):
						t.Fatal("drain wait timeout")
					}
					o.channel.Close()
					o.root.Close()
					o.closed = true
					o.final = r.Final
					// Delivery does not confirm publication; no metadata lookup after stop.
				}
				o.faultChannel.Close()
				if len(r.Final.Originals) != 1 || r.Final.ConnectorStarted != 1 || r.Final.Connectors != 1 || r.Final.ConnectorActive != 0 || r.Final.Active || r.Final.Failed {
					t.Fatal("terminal work or failure changed")
				}
				ref := r.Final.Originals[0].Ref
				b := r.Final.ForwardBoundary
				if ref.ID != event.ID || !ref.Complete || !ref.Delivered || ref.RawSHA256 != ref.ForwardSHA256 || ref.Forward == nil || ref.Forward.Stage != "delivered" || ref.Forward.Bytes != ref.RawBytes || !ref.Forward.WriteHalfClosed || b == nil || b.ID != event.ID || b.DeadlineNS-b.StartedNS != int64(45*time.Second) || b.DeadlineReached || b.ContextDeadlineExceeded || b.ContextCancelled {
					t.Fatal("drain changed original bytes/deadline/delivery")
				}
				r.Reaped = !witnessLifetime(r.Observer, probe.ABI)
				_, e = witnessRootSocket(o.config.Back, o.config.SocketGID)
				r.HelperUnchanged = e == nil && o.config.Helper.check(probe.ABI) == nil
				config, ce := os.ReadFile(p.pair.path)
				back, be := os.ReadFile(p.backPath)
				token, te := os.ReadFile(filepath.Join(p.pair.config.StateDirectory, "token.sha256"))
				r.ConfigurationUnchanged = ce == nil && be == nil && te == nil && bytes.Equal(config, p.configBytes) && bytes.Equal(back, p.backBytes) && bytes.Equal(token, p.digest)
				out := filepath.Join(export, label)
				if os.Mkdir(out, 0700) != nil {
					t.Fatal("case export")
				}
				r.Originals, r.Indexes = witnessControllerExport(t, p, out)
				if witnessWritePrivateIndex(out, "receipt.json", r) != nil {
					t.Fatal("case receipt")
				}
				if !r.Reaped || !r.HelperUnchanged || !r.ConfigurationUnchanged {
					t.Fatal("teardown/actor/config uncertainty")
				}
				p.pair.management.stop(t, syscall.SIGTERM, true)
				p.pair.helper.stop(t, syscall.SIGTERM, true)
				t.Logf("NATIVE_WITNESS_OVERLAP label=%s denied=%d delivered=true publication_confirmed=%t resource_acceptance=false", label, r.Rejected.Denied-r.Before.Denied, r.Publication != "")
			})
		}
	}
}
func isWitnessConnectionRefused(e error) bool { return errors.Is(e, syscall.ECONNREFUSED) }
