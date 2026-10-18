//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

type witnessRawCase struct {
	Source                         *witnessRawSourceReceipt
	FailureStatus, UnavailableBody []byte
	UnavailableStatus              int
	witnessBrokerSpaceCase
	PriorBytes, ConfiguredPriorBytes, FinalAggregateBytes int64
}

func TestNativeDashboardWitnessRawBudget(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("disposable VM opt-in")
	}
	if os.Getuid() != 0 || witnessArchiveCap != 268435456 || witnessRawCap != 67508864 || witnessCapacityTmpfs != 335544320 || witnessAttemptCap != 32 {
		t.Fatal("root fixture / unchanged bounds")
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
		if witnessWritePrivateIndex("../../.tools", "witness-raw-cleanup.json", struct{ Verified bool }{clean}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("raw cleanup uncertain")
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
	// Control frames intentionally exclude multiline compiler prose. Preserve
	// that full original separately and send only its actual ABI as bootstrap.
	if witnessWritePrivateIndex("../../.tools", "witness-raw-abi.json", probe.ABI) != nil {
		t.Fatal("exclusive source ABI bootstrap")
	}
	_ = nativeStorageFixtureObserved(t, owner.storage)
	count := len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
	export := "../../.tools/witness-raw-cases"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive case export")
	}
	for _, label := range []string{"normal", "raw-limit"} {
		t.Run(label, func(t *testing.T) {
			r := witnessRawCase{witnessBrokerSpaceCase: witnessBrokerSpaceCase{Label: label, Mode: "native-small"}, PriorBytes: owner.witnessArchiveBytes}
			recorded := false
			t.Cleanup(func() {
				if !recorded {
					if witnessWritePrivateIndex(export, label+"-observations.json", r) != nil {
						t.Error("failure observations")
					}
				}
			})
			start := func(t *testing.T, cfg witnessNativeConfig, path string) *witnessFixtureObserver {
				r.ConfiguredPriorBytes = cfg.PriorArchiveBytes
				o := startWitnessObserver(t, cfg, path)
				var e error
				r.Before, e = o.control(witnessNativeCommand{Operation: "readback"})
				if e != nil || r.Before.Active || len(r.Before.Originals) != 0 || r.Before.ConnectorStarted != 0 {
					t.Fatal("management started before budget setup")
				}
				return o
			}
			var helperStart witnessFixtureHelperStart
			if label == "raw-limit" {
				helperStart = startWitnessRawProducer
			}
			p := startWitnessPairUsingHelper(t, owner, probe, "native-small", "", label, start, helperStart)
			o := p.observer
			t.Cleanup(func() {
				if !o.closed {
					diagnostic, _ := o.control(witnessNativeCommand{Operation: "readback"})
					if witnessWritePrivateIndex(export, label+"-failure.json", diagnostic) != nil {
						t.Error("failure readback")
					}
				}
			})
			r.Owner, r.Helper = o.config.Parent, o.config.Helper
			var e error
			r.Observer, e = witnessObserveActor(o.process.cmd.Process.Pid, probe.ABI)
			if e != nil {
				t.Fatal(e)
			}
			r.Management, e = witnessObserveActor(p.pair.management.cmd.Process.Pid, probe.ABI)
			if e != nil {
				t.Fatal(e)
			}
			if label == "normal" {
				r.Publication = p.pair.session.ready(t, count, "")
				snapshot, ref, e := o.original(r.Publication)
				if e != nil || !snapshot.Complete() || !ref.Delivered {
					t.Fatal("normal publication", e)
				}
				r.Final = o.stop(t)
				r.Boundary, r.Followup = r.Final, r.Final
			} else {
				performanceWait(t, func() bool {
					r.Boundary, _ = o.control(witnessNativeCommand{Operation: "readback"})
					return r.Boundary.Failed && !r.Boundary.Active
				})
				b := r.Boundary.CaptureBoundary
				if r.Boundary.Failure != "producer-capture" || !r.Boundary.StoreFailed || r.Boundary.ConnectorStarted != 1 || r.Boundary.Connectors != 1 || r.Boundary.ConnectorActive != 0 || len(r.Boundary.Originals) != 1 || b == nil || b.Stage != "copy" || b.AcknowledgedBytes != witnessRawCap || b.Error.Class != "" || b.Error.Errno != 0 || b.Management != r.Management.incarnation() || b.Helper != r.Helper.incarnation() {
					t.Fatal("actual raw boundary", r.Boundary.Failure, b)
				}
				ref := r.Boundary.Originals[0].Ref
				if ref.RawBytes != witnessRawCap || ref.CaptureFailure != "raw-limit" || ref.Complete || ref.Delivered || ref.Forward != nil || ref.ID != b.ID {
					t.Fatal("raw prefix eligible or wrong cause")
				}

				var source witnessRawSourceReceipt
				performanceWait(t, func() bool {
					f, e := os.Open("../../.tools/witness-raw-source.json")
					if e != nil {
						return false
					}
					defer f.Close()
					return witnessControlRead(f, &source) == nil && witnessEOF(f) == nil
				})
				r.Source = &source
				connector := r.Boundary.ConnectorBoundary
				if source.DataClass != "synthetic-oversized-producer-control" || source.Actor != r.Helper || source.RequestID != b.ID || source.PeerUID != p.pair.config.ManagementUID || connector == nil || source.PeerPID != connector.Actor.PID || !connector.Accounted || !connector.Present || connector.ContextEnded || connector.ChannelClosed || connector.WaitFailed || !connector.Exited || connector.ExitCode != 0 || connector.Signal != 0 || source.SentBytes != witnessRawCap+1 || source.WriteError.Class != "" || source.FinishedNS >= source.DeadlineNS || r.ConfiguredPriorBytes+witnessRawCap >= witnessArchiveCap {
					t.Fatal("actual oversized source / original connector / unconstrained archive", source, connector)
				}
				_, _, e = o.original(b.ID)
				r.OriginalDenied = e != nil
				performanceWait(t, func() bool { return witnessBrokerCollectionIdle(t, p.pair.session) })
				r.RefreshStatus = witnessAdmissionRefresh(t, p.pair.session, true)
				performanceWait(t, func() bool {
					r.Followup, _ = o.control(witnessNativeCommand{Operation: "readback"})
					return r.Followup.Denied > r.Boundary.Denied && !r.Followup.Active && witnessBrokerCollectionIdle(t, p.pair.session)
				})
				response, se := p.pair.session.http.Get(p.pair.session.base + "/status")
				if se != nil {
					t.Fatal(se)
				}
				r.FailureStatus, se = io.ReadAll(io.LimitReader(response.Body, 65537))
				response.Body.Close()
				r.LastCollectionFailed = se == nil && response.StatusCode == 200 && bytes.Contains(r.FailureStatus, []byte("Last collection:")) && bytes.Contains(r.FailureStatus, []byte("Cached data is stale."))
				unavailable, ue := p.pair.session.http.Get(p.pair.session.base + "/api/overview")
				if ue != nil {
					t.Fatal(ue)
				}
				r.UnavailableBody, ue = io.ReadAll(io.LimitReader(unavailable.Body, 65537))
				unavailable.Body.Close()
				r.UnavailableStatus = unavailable.StatusCode
				if ue != nil || r.UnavailableStatus != 503 {
					t.Fatal("failed initial capture published", ue)
				}
				if !r.OriginalDenied || !r.LastCollectionFailed || !reflect.DeepEqual(r.Boundary.Originals, r.Followup.Originals) || !reflect.DeepEqual(r.Boundary.CaptureBoundary, r.Followup.CaptureBoundary) || r.Followup.ConnectorStarted != 1 || r.Followup.Connectors != 1 || r.Followup.ConnectorActive != 0 {
					t.Fatal("failed budget resumed or first failure changed")
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
				o.closed = true
				o.final = r.Final
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
			r.FinalAggregateBytes = owner.witnessArchiveBytes
			if witnessWritePrivateIndex(out, "receipt.json", r) != nil {
				t.Fatal("case receipt")
			}
			recorded = true
			if !r.Reaped || !r.HelperUnchanged || !r.ConfigurationUnchanged || label == "raw-limit" && r.FinalAggregateBytes != r.PriorBytes+witnessRawCap {
				t.Fatal("raw/teardown/actor uncertainty")
			}
			p.pair.management.stop(t, syscall.SIGTERM, true)
			p.pair.helper.stop(t, syscall.SIGTERM, true)
		})
	}
}
