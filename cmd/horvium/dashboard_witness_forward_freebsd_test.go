//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

type witnessForwardCase struct {
	Label, Mode, Publication                                          string
	Owner, Observer, Management, Helper                               witnessActor
	ManagerKilledBeforeRelease, ManagerReaped, ManagerAliveAtDelivery bool
	ManagerRawWait                                                    int
	ManagerWaitElapsedNS                                              int64
	Final                                                             witnessNativeReceipt
	Reaped, HelperUnchanged, ConfigurationUnchanged                   bool
	Originals, Indexes                                                []witnessInterruptedOriginal
}

// Focused diagnostics, not the failed full attempt-cap qualification. Real
// manager/helper artifacts, original deadline and all normal limits are retained.
func TestNativeDashboardWitnessForwardDiagnostics(t *testing.T) {
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
		if witnessWritePrivateIndex("../../.tools", "witness-forward-cleanup.json", struct{ Verified bool }{clean}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("forward diagnostic cleanup uncertain")
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
	export := "../../.tools/witness-forward-cases"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive forward cases")
	}
	for _, label := range []string{"native-small", "projection-10000", "forward-manager-exit"} {
		t.Run(label, func(t *testing.T) {
			mode, stage, count := label, "", nativeCount
			if label == "projection-10000" {
				count = 10000
			}
			if label == "forward-manager-exit" {
				mode, stage = "native-small", "sealed"
			}
			p := startWitnessPairWithFault(t, owner, probe, mode, stage, label)
			o := p.observer
			t.Cleanup(func() {
				if !o.closed {
					diagnostic, _ := o.control(witnessNativeCommand{Operation: "readback"})
					if witnessWritePrivateIndex(export, label+"-failure.json", diagnostic) != nil {
						t.Error("original failure diagnostic")
					}
				}
			})
			r := witnessForwardCase{Label: label, Mode: mode, Owner: o.config.Parent, Helper: o.config.Helper}
			var e error
			r.Observer, e = witnessObserveActor(o.process.cmd.Process.Pid, probe.ABI)
			if e != nil {
				t.Fatal(e)
			}
			r.Management, e = witnessObserveActor(p.pair.management.cmd.Process.Pid, probe.ABI)
			if e != nil {
				t.Fatal(e)
			}
			if stage == "" {
				p.pair.session.ready(t, count, "")
				r.Publication = p.pair.session.overview(t).Generation
				snapshot, ref, e := o.original(r.Publication)
				if e != nil || !snapshot.Complete() || !ref.Complete || !ref.Delivered || ref.Forward == nil || ref.Forward.Stage != "delivered" || ref.Forward.Bytes != ref.RawBytes || ref.Forward.SHA256 != ref.RawSHA256 || !ref.Forward.WriteHalfClosed || ref.Forward.Error.Class != "" {
					t.Fatal("normal original/forward/publication", e)
				}
				r.ManagerAliveAtDelivery = r.Management.check(probe.ABI) == nil
				r.Final = o.stop(t)
			} else {
				_ = o.faultChannel.SetReadDeadline(time.Now().Add(45 * time.Second))
				var event witnessFaultEvent
				if witnessControlRead(o.faultChannel, &event) != nil || event.Stage != "sealed" || !dashboardGenerationPattern.MatchString(event.ID) {
					t.Fatal("original sealed boundary")
				}
				before := time.Now()
				p.pair.management.stop(t, syscall.SIGKILL, false)
				r.ManagerWaitElapsedNS = time.Since(before).Nanoseconds()
				wait, ok := p.pair.management.cmd.ProcessState.Sys().(syscall.WaitStatus)
				r.ManagerRawWait = int(wait)
				r.ManagerReaped = ok && wait.Signaled() && wait.Signal() == syscall.SIGKILL && !witnessLifetime(r.Management, probe.ABI)
				r.ManagerKilledBeforeRelease = r.ManagerReaped
				if !r.ManagerReaped || witnessControlWrite(o.faultChannel, witnessNativeCommand{Operation: "release"}) != nil {
					t.Fatal("original manager wait/release")
				}
				performanceWait(t, func() bool {
					r.Final, _ = o.control(witnessNativeCommand{Operation: "readback"})
					return r.Final.Failed && !r.Final.Active
				})
				if r.Final.Failure != "producer-forward" || !r.Final.StoreFailed || len(r.Final.Originals) != 1 {
					t.Fatal("peer loss was not forwarding failure", r.Final.Failure)
				}
				ref := r.Final.Originals[0].Ref
				if !ref.Complete || ref.Delivered || ref.ForwardSHA256 != "" || ref.Forward == nil || ref.Forward.Stage != "copy" || ref.Forward.Error.Class != "syscall" || ref.Forward.Error.Errno != uint64(syscall.EPIPE) {
					t.Fatal("actual broken-pipe diagnostic missing", ref.Forward)
				}
				if _, _, e := o.original(event.ID); e == nil {
					t.Fatal("undelivered original eligible")
				}
				r.Final, e = o.control(witnessNativeCommand{Operation: "stop"})
				if e == nil || !r.Final.Failed || !r.Final.FrontRemoved {
					t.Fatal("failed owner terminal receipt")
				}
				select {
				case e := <-o.process.done:
					o.process.waited = true
					if e != nil {
						t.Fatal("observer wait", e)
					}
				case <-time.After(10 * time.Second):
					t.Fatal("observer wait timeout")
				}
				o.channel.Close()
				o.root.Close()
				o.faultChannel.Close()
				o.closed = true
				o.final = r.Final
			}
			boundary := r.Final.ForwardBoundary
			if boundary == nil || boundary.Management != r.Management.incarnation() || boundary.Helper != r.Helper.incarnation() || boundary.DeadlineNS-boundary.StartedNS != int64(45*time.Second) || boundary.CapturedElapsedNS <= 0 || boundary.ForwardStartedElapsedNS < boundary.CapturedElapsedNS || boundary.FinishedElapsedNS < boundary.ForwardStartedElapsedNS || boundary.Forward == nil {
				t.Fatal("original connection timing/actor diagnostic missing")
			}
			if stage != "" && (boundary.DeadlineReached || boundary.ContextDeadlineExceeded || boundary.ContextCancelled) {
				t.Fatal("manager death mislabeled as observer deadline/cancellation")
			}
			r.Reaped = !witnessLifetime(r.Observer, probe.ABI) && !r.Final.Active && r.Final.ConnectorActive == 0 && r.Final.ConnectorStarted == r.Final.Connectors
			_, e = witnessRootSocket(o.config.Back, o.config.SocketGID)
			r.HelperUnchanged = e == nil && o.config.Helper.check(probe.ABI) == nil
			config, ce := os.ReadFile(p.pair.path)
			back, be := os.ReadFile(p.backPath)
			token, te := os.ReadFile(filepath.Join(p.pair.config.StateDirectory, "token.sha256"))
			r.ConfigurationUnchanged = ce == nil && be == nil && te == nil && bytes.Equal(config, p.configBytes) && bytes.Equal(back, p.backBytes) && bytes.Equal(token, p.digest)
			out := filepath.Join(export, label)
			if os.Mkdir(out, 0700) != nil {
				t.Fatal("exclusive case export")
			}
			r.Originals, r.Indexes = witnessControllerExport(t, p, out)
			if witnessWritePrivateIndex(out, "receipt.json", r) != nil {
				t.Fatal("original forward receipt")
			}
			if !r.Reaped || !r.HelperUnchanged || !r.ConfigurationUnchanged || stage == "" && !r.ManagerAliveAtDelivery {
				t.Fatal("forward cleanup/configuration/actor uncertainty")
			}
			if !p.pair.management.waited {
				p.pair.management.stop(t, syscall.SIGTERM, true)
			}
			p.pair.helper.stop(t, syscall.SIGTERM, true)
			t.Logf("NATIVE_WITNESS_FORWARD label=%s originals=%d peer_loss=%t resource_acceptance=false", label, len(r.Originals), r.ManagerKilledBeforeRelease)
		})
	}
	t.Run("actual-expired-socket", func(t *testing.T) {
		store, e := newDashboardWitnessStore(witnessPrivateDirectory(t))
		if e != nil {
			t.Fatal(e)
		}
		defer store.close()
		raw := witnessFrames(witnessGolden())
		ref, e := store.capture(bytes.NewReader(raw), witnessTestID, "native-small", "diagnostic-management", "diagnostic-helper")
		if e != nil {
			t.Fatal(e)
		}
		left, right, e := witnessSocketPair()
		if e != nil {
			t.Fatal(e)
		}
		defer left.Close()
		defer right.Close()
		if left.SetWriteDeadline(time.Now().Add(-time.Second)) != nil {
			t.Fatal("injected expired socket")
		}
		if store.forward(ref, left) == nil || !store.failed || ref.Delivered || ref.Forward == nil || ref.Forward.Bytes != 0 || ref.Forward.Error.Class != "deadline" {
			t.Fatal("actual expired socket misclassified")
		}
		if witnessWritePrivateIndex(export, "expired-socket.json", ref) != nil {
			t.Fatal("deadline fixture receipt")
		}
		t.Log(fmt.Sprintf("NATIVE_WITNESS_FORWARD expired_socket=true accepted_bytes=%d resource_acceptance=false", ref.Forward.Bytes))
	})
}
