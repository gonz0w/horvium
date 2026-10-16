//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

type witnessExpiryCase struct {
	TimeoutStatus, UnavailableBody                           []byte
	Label, Mode, Publication                                 string
	Owner, Observer, Management, Helper                      witnessActor
	Event                                                    witnessFaultEvent
	Before, Boundary, Final                                  witnessNativeReceipt
	ReleaseNS                                                int64
	ManagerReapedBeforeRelease                               bool
	ManagerRawWait, ManagerExitCode                          int
	ManagerWaitElapsedNS                                     int64
	ManagerAliveAfterExpiry, TimeoutObserved, OriginalDenied bool
	TimeoutStatusSHA256                                      string
	UnavailableStatus                                        int
	Reaped, HelperUnchanged, ConfigurationUnchanged          bool
	Originals, Indexes                                       []witnessInterruptedOriginal
}

// Negative controls keep the original product artifacts, deadlines and caps.
// Waiting until this original deadline is deliberate fault injection, never a
// retry, pacing adjustment or a successful resource/latency observation.
func TestNativeDashboardWitnessExpiryAndShutdown(t *testing.T) {
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
		if witnessWritePrivateIndex("../../.tools", "witness-expiry-cleanup.json", struct{ Verified bool }{clean}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("expiry/shutdown cleanup uncertain")
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
	export := "../../.tools/witness-expiry-cases"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive case export")
	}
	for _, action := range []string{"normal", "shutdown", "expiry"} {
		for _, mode := range []string{"native-small", "projection-10000"} {
			label := action + "-" + mode
			t.Run(label, func(t *testing.T) {
				stage := ""
				if action != "normal" {
					stage = "sealed"
				}
				p := startWitnessPairWithFault(t, owner, probe, mode, stage, label)
				o := p.observer
				r := witnessExpiryCase{Label: label, Mode: mode, Owner: o.config.Parent, Helper: o.config.Helper}
				caseRecorded := false
				t.Cleanup(func() {
					if !caseRecorded {
						if witnessWritePrivateIndex(export, label+"-observations.json", r) != nil {
							t.Error("partial original observations")
						}
					}
					if !o.closed {
						diagnostic, _ := o.control(witnessNativeCommand{Operation: "readback"})
						if witnessWritePrivateIndex(export, label+"-failure.json", diagnostic) != nil {
							t.Error("failure diagnostic")
						}
					}
				})
				var e error
				r.Observer, e = witnessObserveActor(o.process.cmd.Process.Pid, probe.ABI)
				if e != nil {
					t.Fatal(e)
				}
				r.Management, e = witnessObserveActor(p.pair.management.cmd.Process.Pid, probe.ABI)
				if e != nil {
					t.Fatal(e)
				}
				if action == "normal" {
					count := nativeCount
					if mode == "projection-10000" {
						count = 10000
					}
					r.Publication = p.pair.session.ready(t, count, "")
					snapshot, ref, e := o.original(r.Publication)
					if e != nil || !snapshot.Complete() || !ref.Delivered {
						t.Fatal("normal original publication", e)
					}
					r.Final = o.stop(t)
				} else {
					_ = o.faultChannel.SetReadDeadline(time.Now().Add(45 * time.Second))
					if witnessControlRead(o.faultChannel, &r.Event) != nil || r.Event.Stage != "sealed" || !dashboardGenerationPattern.MatchString(r.Event.ID) || r.Event.StartedNS <= 0 || r.Event.DeadlineNS-r.Event.StartedNS != int64(45*time.Second) {
						t.Fatal("original sealed connection window missing")
					}
					r.Before, e = o.control(witnessNativeCommand{Operation: "readback"})
					if e != nil || !r.Before.Active || r.Before.Failed || len(r.Before.Originals) != 1 || r.Before.ConnectorStarted != 1 || r.Before.Connectors != 1 || r.Before.ConnectorActive != 0 {
						t.Fatal("sealed original/connector boundary")
					}
					ref := r.Before.Originals[0].Ref
					if ref.ID != r.Event.ID || !ref.Complete || ref.Delivered || ref.Forward != nil {
						t.Fatal("source was not complete and undelivered")
					}
					if action == "shutdown" {
						before := time.Now()
						p.pair.management.stop(t, syscall.SIGTERM, true)
						r.ManagerWaitElapsedNS = time.Since(before).Nanoseconds()
						wait, ok := p.pair.management.cmd.ProcessState.Sys().(syscall.WaitStatus)
						r.ManagerRawWait = int(wait)
						r.ManagerExitCode = p.pair.management.cmd.ProcessState.ExitCode()
						r.ManagerReapedBeforeRelease = ok && wait.Exited() && wait.ExitStatus() == 0 && !witnessLifetime(r.Management, probe.ABI)
						if !r.ManagerReapedBeforeRelease {
							t.Fatal("original manager did not exit cleanly after SIGTERM")
						}
					} else {
						// Observe the existing absolute connection deadline, not a newly expired
						// socket, changed timeout, relative restart or repeated collection.
						remaining := time.Until(time.Unix(0, r.Event.DeadlineNS))
						if remaining <= 0 || remaining > 45*time.Second {
							t.Fatal("original deadline already lost before negative control")
						}
						timer := time.NewTimer(remaining)
						<-timer.C
					}
					r.ReleaseNS = time.Now().UnixNano()
					if witnessControlWrite(o.faultChannel, witnessNativeCommand{Operation: "release"}) != nil {
						t.Fatal("sealed original release")
					}
					performanceWait(t, func() bool {
						r.Boundary, _ = o.control(witnessNativeCommand{Operation: "readback"})
						return r.Boundary.Failed && !r.Boundary.Active
					})
					if r.Boundary.Failure != "producer-forward" || !r.Boundary.StoreFailed || len(r.Boundary.Originals) != 1 {
						t.Fatal("negative control not original forwarding failure", r.Boundary.Failure)
					}
					ref = r.Boundary.Originals[0].Ref
					b := r.Boundary.ForwardBoundary
					if !ref.Complete || ref.Delivered || ref.ForwardSHA256 != "" || ref.Forward == nil || ref.Forward.Stage != "copy" || ref.Forward.Bytes != 0 || ref.Forward.WriteHalfClosed || b == nil || b.StartedNS != r.Event.StartedNS || b.DeadlineNS != r.Event.DeadlineNS || b.ID != r.Event.ID {
						t.Fatal("negative control lost original delivery/deadline facts")
					}
					if action == "expiry" {
						if r.ReleaseNS < r.Event.DeadlineNS || !b.DeadlineReached || ref.Forward.Error.Class != "deadline" {
							t.Fatal("actual unchanged connection deadline not observed", b, ref.Forward)
						}
						r.ManagerAliveAfterExpiry = r.Management.check(probe.ABI) == nil
						performanceWait(t, func() bool {
							response, e := p.pair.session.http.Get(p.pair.session.base + "/status")
							if e != nil {
								t.Fatal(e)
							}
							body, re := io.ReadAll(io.LimitReader(response.Body, 65537))
							response.Body.Close()
							if re != nil || response.StatusCode != 200 {
								t.Fatal("timeout status unavailable")
							}
							if bytes.Contains(body, []byte("Collecting an observation")) {
								return false
							}
							r.TimeoutObserved = bytes.Contains(body, []byte("Last collection: timeout."))
							r.TimeoutStatus = append([]byte(nil), body...)
							r.TimeoutStatusSHA256 = fmt.Sprintf("%x", sha256.Sum256(body))
							return true
						})
						response, e := p.pair.session.http.Get(p.pair.session.base + "/api/overview")
						if e != nil {
							t.Fatal(e)
						}
						body, re := io.ReadAll(io.LimitReader(response.Body, 65537))
						r.UnavailableBody = append([]byte(nil), body...)
						response.Body.Close()
						r.UnavailableStatus = response.StatusCode
						if !r.ManagerAliveAfterExpiry || !r.TimeoutObserved || re != nil || r.UnavailableStatus != 503 {
							t.Fatalf("timeout qualification failed: manager_alive=%t timeout_observed=%t unavailable_status=%d response_read_failed=%t", r.ManagerAliveAfterExpiry, r.TimeoutObserved, r.UnavailableStatus, re != nil)
						}
					} else if r.ReleaseNS >= r.Event.DeadlineNS || b.DeadlineReached || b.ContextDeadlineExceeded || b.ContextCancelled || ref.Forward.Error.Class != "syscall" || ref.Forward.Error.Errno != uint64(syscall.EPIPE) {
						t.Fatal("orderly manager shutdown mislabeled as observer timeout/cancellation")
					}
					_, _, e = o.original(r.Event.ID)
					r.OriginalDenied = e != nil
					if !r.OriginalDenied {
						t.Fatal("undelivered original eligible")
					}
					r.Final, e = o.control(witnessNativeCommand{Operation: "stop"})
					if e == nil || !r.Final.Failed || !r.Final.FrontRemoved {
						t.Fatal("failed final receipt")
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
				b := r.Final.ForwardBoundary
				if b == nil || b.Management != r.Management.incarnation() || b.Helper != r.Helper.incarnation() || b.DeadlineNS-b.StartedNS != int64(45*time.Second) || b.CapturedElapsedNS <= 0 || b.ForwardStartedElapsedNS < b.CapturedElapsedNS || b.FinishedElapsedNS < b.ForwardStartedElapsedNS {
					t.Fatal("original actor/window/timing missing")
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
					t.Fatal("case export")
				}
				r.Originals, r.Indexes = witnessControllerExport(t, p, out)
				if witnessWritePrivateIndex(out, "receipt.json", r) != nil {
					t.Fatal("case receipt")
				}
				caseRecorded = true
				if !r.Reaped || !r.HelperUnchanged || !r.ConfigurationUnchanged {
					t.Fatal("teardown/actor/config uncertainty")
				}
				if !p.pair.management.waited {
					p.pair.management.stop(t, syscall.SIGTERM, true)
				}
				p.pair.helper.stop(t, syscall.SIGTERM, true)
				t.Logf("NATIVE_WITNESS_EXPIRY label=%s originals=%d published=%t deadline_reached=%t resource_acceptance=false", label, len(r.Originals), r.Publication != "", b.DeadlineReached)
			})
		}
	}
}
