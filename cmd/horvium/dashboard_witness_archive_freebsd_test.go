//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"io"
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

type witnessArchivePrefillOriginal struct {
	Copy witnessInterruptedOriginal
	Ref  dashboardWitnessRef
}
type witnessArchivePrefillReceipt struct {
	DataClass                                                 string
	Actor                                                     witnessActor
	PriorBytes, CapturedBytes, AggregateBytes, RemainingBytes int64
	Originals                                                 []witnessArchivePrefillOriginal
	Exported, MountRemoved                                    bool
}
type witnessArchivePrefill struct {
	finished         bool
	receipt          witnessArchivePrefillReceipt
	store            *dashboardWitnessStore
	mount, directory string
	under            os.FileInfo
	fs               unix.Statfs_t
}

// Real captures on a separate full-size owned tmpfs. No raw file is exported to
// guest ZFS before the final producer collection finishes. These fixed zero
// streams are synthetic pressure fixtures, never producer/publication evidence.
func witnessPrepareArchiveBudget(t *testing.T, abi witnessABI, prior int64) *witnessArchivePrefill {
	t.Helper()
	p := &witnessArchivePrefill{receipt: witnessArchivePrefillReceipt{DataClass: "synthetic-zero-stream-pressure", PriorBytes: prior}}
	var e error
	p.receipt.Actor, e = witnessObserveActor(os.Getpid(), abi)
	if e != nil || prior <= 0 || prior >= witnessArchiveCap-80 {
		t.Fatal("actual preceding archive bytes required", e)
	}
	p.mount = filepath.Join(witnessPrivateDirectory(t), "archive-prefill-mount")
	if os.Mkdir(p.mount, 0700) != nil {
		t.Fatal("exclusive prefill mount")
	}
	p.under, e = os.Lstat(p.mount)
	if e != nil {
		t.Fatal(e)
	}
	readerFixtureCommand(t, "/sbin/mount", "-t", "tmpfs", "-o", "size=335544320,mode=0700", "tmpfs", p.mount)
	if unix.Statfs(p.mount, &p.fs) != nil || string(bytes.TrimRight(p.fs.Fstypename[:], "\x00")) != "tmpfs" {
		t.Fatal("original prefill tmpfs")
	}
	t.Cleanup(func() {
		if !p.finished {
			p.finish(t)
		}
	})
	p.directory = filepath.Join(p.mount, "originals")
	if os.Mkdir(p.directory, 0700) != nil {
		t.Fatal("exclusive prefill originals")
	}
	p.store, e = newDashboardWitnessStore(p.directory)
	if e != nil {
		t.Fatal(e)
	}
	target := int64(witnessArchiveCap) - prior - 80
	for p.store.bytes < target {
		n := min(int64(witnessRawCap), target-p.store.bytes)
		ref, e := p.store.capture(io.LimitReader(witnessCapacityZero{}, n), witnessCapacityID(len(p.store.refs)+1), "native-small", "archive-prefill-owner", "synthetic-zero-stream")
		if e != nil || ref == nil || ref.RawBytes != n || ref.Complete || ref.Delivered || ref.CaptureFailure != "" || p.store.failed || p.store.check(p.store.sealed[ref.Name]) != nil {
			t.Fatal("actual bounded prefill capture", e)
		}
	}
	p.receipt.CapturedBytes = p.store.bytes
	p.receipt.AggregateBytes, e = witnessArchiveSum(prior, p.store.refs)
	p.receipt.RemainingBytes = witnessArchiveCap - p.receipt.AggregateBytes
	if e != nil || p.receipt.RemainingBytes != 80 || p.store.bytes != target {
		t.Fatal("actual archive sum / remaining bytes", e)
	}
	return p
}

func (p *witnessArchivePrefill) finish(t *testing.T) {
	t.Helper()
	if p.finished {
		return
	}
	p.finished = true
	out := "../../.tools/witness-archive-prefill"
	created := false
	defer func() {
		if p.store != nil && p.store.close() != nil {
			t.Error("prefill descriptors")
		}

		var current unix.Statfs_t
		ok := unix.Statfs(p.mount, &current) == nil && current.Fsid == p.fs.Fsid && string(bytes.TrimRight(current.Fstypename[:], "\x00")) == "tmpfs"
		if ok {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			ok = exec.CommandContext(ctx, "/sbin/umount", p.mount).Run() == nil
			cancel()
		}
		if ok {
			after, e := os.Lstat(p.mount)
			ok = e == nil && os.SameFile(p.under, after) && os.Remove(p.mount) == nil
		}
		p.receipt.MountRemoved = ok
		if created && witnessWritePrivateIndex(out, "receipt.json", p.receipt) != nil {
			t.Error("prefill receipt")
		}
		if !ok || !p.receipt.Exported {
			t.Error("prefill export/cleanup uncertain")
		}
	}()
	if os.Mkdir(out, 0700) != nil {
		t.Error("exclusive prefill export")
		return
	}
	created = true
	if p.store != nil {
		for _, ref := range p.store.refs {
			if p.store.check(p.store.sealed[ref.Name]) != nil {
				t.Error("prefill custody changed")
				break
			}
			copied := witnessControllerCopy(t, filepath.Join(p.directory, ref.Name), out, witnessRawCap)
			if copied.SHA256 != ref.RawSHA256 || copied.Object.Size != ref.RawBytes || copied.Mode != 0400 {
				t.Error("prefill export mismatch")
				break
			}
			p.receipt.Originals = append(p.receipt.Originals, witnessArchivePrefillOriginal{copied, *ref})
		}
		p.receipt.Exported = len(p.receipt.Originals) == len(p.store.refs)
	}
}

type witnessArchiveCase struct {
	FailureStatus, UnavailableBody []byte
	UnavailableStatus              int
	witnessBrokerSpaceCase
	PriorBytes, ConfiguredPriorBytes, FinalAggregateBytes int64
}

func TestNativeDashboardWitnessArchiveBudget(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("disposable VM opt-in")
	}
	if os.Getuid() != 0 || witnessArchiveCap != 268435456 || witnessRawCap != 67508864 || witnessCapacityTmpfs != 335544320 || witnessAttemptCap != 32 {
		t.Fatal("root fixture / unchanged bounds")
	}
	owner := newDashboardGuestOwner(t)
	var pressure *witnessArchivePrefill
	t.Cleanup(func() {
		clean := owner.storageCleanups == 1 && dashboardReaperEmpty() == nil && reflect.DeepEqual(owner.artifacts, observeDashboardParentArtifacts(t)) && pressure != nil && pressure.receipt.MountRemoved && pressure.receipt.Exported
		for key, dir := range owner.directories {
			if _, e := os.Lstat(dir); !os.IsNotExist(e) {
				clean = false
			}
			if _, e := os.Lstat(owner.sockets[key]); !os.IsNotExist(e) {
				clean = false
			}
		}
		if witnessWritePrivateIndex("../../.tools", "witness-archive-cleanup.json", struct{ Verified bool }{clean}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("archive cleanup uncertain")
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
	export := "../../.tools/witness-archive-cases"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive case export")
	}
	for _, label := range []string{"normal", "archive-limit"} {
		t.Run(label, func(t *testing.T) {
			r := witnessArchiveCase{witnessBrokerSpaceCase: witnessBrokerSpaceCase{Label: label, Mode: "native-small"}, PriorBytes: owner.witnessArchiveBytes}
			recorded := false
			t.Cleanup(func() {
				if !recorded {
					if witnessWritePrivateIndex(export, label+"-observations.json", r) != nil {
						t.Error("failure observations")
					}
				}
			})
			start := func(t *testing.T, cfg witnessNativeConfig, path string) *witnessFixtureObserver {
				if label == "archive-limit" {
					pressure = witnessPrepareArchiveBudget(t, probe.ABI, owner.witnessArchiveBytes)
					cfg.PriorArchiveBytes = pressure.receipt.AggregateBytes
					owner.witnessArchiveBytes = cfg.PriorArchiveBytes
				}
				r.ConfiguredPriorBytes = cfg.PriorArchiveBytes
				o := startWitnessObserver(t, cfg, path)
				var e error
				r.Before, e = o.control(witnessNativeCommand{Operation: "readback"})
				if e != nil || r.Before.Active || len(r.Before.Originals) != 0 || r.Before.ConnectorStarted != 0 {
					t.Fatal("management started before budget setup")
				}
				return o
			}
			p := startWitnessPairUsingObserver(t, owner, probe, "native-small", "", label, start)
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
				if r.Boundary.Failure != "producer-capture" || !r.Boundary.StoreFailed || r.Boundary.ConnectorStarted != 1 || r.Boundary.Connectors != 1 || r.Boundary.ConnectorActive != 0 || len(r.Boundary.Originals) != 1 || b == nil || b.Stage != "copy" || b.AcknowledgedBytes != 80 || b.Error.Class != "" || b.Error.Errno != 0 || b.Management != r.Management.incarnation() || b.Helper != r.Helper.incarnation() {
					t.Fatal("actual archive boundary", r.Boundary.Failure, b)
				}
				ref := r.Boundary.Originals[0].Ref
				if ref.RawBytes != 80 || ref.CaptureFailure != "archive-limit" || ref.Complete || ref.Delivered || ref.Forward != nil || ref.ID != b.ID {
					t.Fatal("archive prefix eligible or wrong cause")
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
			if !r.Reaped || !r.HelperUnchanged || !r.ConfigurationUnchanged || label == "archive-limit" && r.FinalAggregateBytes != witnessArchiveCap {
				t.Fatal("aggregate/teardown/actor uncertainty")
			}
			p.pair.management.stop(t, syscall.SIGTERM, true)
			p.pair.helper.stop(t, syscall.SIGTERM, true)
			if label == "archive-limit" {
				pressure.finish(t)
			}
		})
	}
}
