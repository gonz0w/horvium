//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type witnessMetadataOriginal struct {
	Class, ID string
	Status    int
	Body      []byte
}
type witnessIdleWindow struct {
	StartedAt, FinishedAt         time.Time
	ElapsedNS                     int64
	RequestsBefore, RequestsAfter int64
	ControlsBefore, ControlsAfter int
}
type witnessTmpfsSpace struct {
	Blocks, Bfree uint64
	Bsize         int64
}

func witnessLifecycleSpace(t *testing.T, path string) witnessTmpfsSpace {
	t.Helper()
	var fs unix.Statfs_t
	if unix.Statfs(path, &fs) != nil || fs.Bsize <= 0 {
		t.Fatal("original tmpfs space")
	}
	return witnessTmpfsSpace{fs.Blocks, fs.Bfree, int64(fs.Bsize)}
}

type witnessLifecycleCase struct {
	Mode                                             string
	Owner, Observer, Management, Helper              witnessActor
	IDs                                              []string
	Confirmations                                    [][]witnessConfirmationReply
	MetadataFiles                                    []string
	Refreshes                                        []witnessRefreshReply
	HealthyControls                                  int
	ExpiredID, CancelledID                           string
	ExpiredDenied, CancelledDenied, IdleDenied       bool
	ExpiredControls, CancelledControls, IdleControls int
	CachedExpiredOriginalPreserved                   bool
	Idle                                             *witnessIdleWindow
	Sampler                                          *witnessSamplerOwnerReceipt
	SpaceBefore, SpaceAfter                          witnessTmpfsSpace
	BeforeIdle, AfterIdle, Final                     witnessNativeReceipt
	Reaped, HelperUnchanged, ConfigurationUnchanged  bool
	Originals, Indexes                               []witnessInterruptedOriginal
}

func TestNativeDashboardWitnessMetadataLifecycle(t *testing.T) {
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
		if witnessWritePrivateIndex("../../.tools", "witness-lifecycle-cleanup.json", struct{ Verified bool }{clean}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("metadata lifecycle cleanup uncertain")
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
	export := "../../.tools/witness-lifecycle-cases"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive case export")
	}

	for _, mode := range []string{"native-small", "projection-10000"} {
		t.Run(mode, func(t *testing.T) {
			out := filepath.Join(export, mode)
			if os.Mkdir(out, 0700) != nil {
				t.Fatal("exclusive lifecycle case")
			}
			r := witnessLifecycleCase{Mode: mode}
			t.Cleanup(func() {
				if t.Failed() {
					if witnessWritePrivateIndex(out, "partial.json", r) != nil {
						t.Error("partial lifecycle receipt")
					}
				}
			})
			p := startWitnessPair(t, owner, probe, mode)
			o := p.observer
			s := p.pair.session
			counter := &witnessRequestCounter{transport: s.http.Transport}
			s.http.Transport = counter
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
			var metadataMu sync.Mutex
			expiredHTTPStatus := 0
			lookup := func(class string) witnessMetadataReader {
				return witnessHTTPMetadataObserve(s.http, s.base, func(id string, status int, body []byte) error {
					metadataMu.Lock()
					defer metadataMu.Unlock()
					name := fmt.Sprintf("metadata-%02d.json", len(r.MetadataFiles)+1)
					if class == "expired" {
						expiredHTTPStatus = status
					}
					if e := witnessWritePrivateIndex(out, name, witnessMetadataOriginal{class, id, status, body}); e != nil {
						return e
					}
					r.MetadataFiles = append(r.MetadataFiles, name)
					return nil
				})
			}
			healthy := newDashboardWitnessRegistry(o.original, lookup("healthy"))
			count := nativeCount
			if mode == "projection-10000" {
				count = 10000
			}
			confirm := func(id string) {
				replies, e := witnessConfirmConcurrent(context.Background(), healthy, id)
				r.IDs = append(r.IDs, id)
				r.Confirmations = append(r.Confirmations, replies)
				if e != nil {
					t.Fatal("real coalesced publication", e)
				}
				for _, reply := range replies {
					if reply.ID != id || reply.Denied {
						t.Fatal("confirmation lineage")
					}
				}
				if healthy.controls() != len(r.IDs) {
					t.Fatal("more than one metadata operation per new ID")
				}
				// Repeated callers recheck immutable originals, without another HTTP lookup.
				if _, _, e := healthy.confirm(context.Background(), id); e != nil || healthy.controls() != len(r.IDs) {
					t.Fatal("confirmed original changed", e)
				}
			}
			previous := ""
			for i := 0; i < 3; i++ {
				if i > 0 {
					reply, e := witnessRefreshOnce(s.http, s.base, s.csrf, i)
					r.Refreshes = append(r.Refreshes, reply)
					if e != nil || !(reply.accepted() || reply.busy()) {
						t.Fatal("one-shot replacement", e)
					}
				}
				id := s.ready(t, count, previous)
				confirm(id)
				previous = id
			}
			r.ExpiredID = r.IDs[0]
			expired := newDashboardWitnessRegistry(o.original, lookup("expired"))
			_, _, first := expired.confirm(context.Background(), r.ExpiredID)
			_, _, second := expired.confirm(context.Background(), previous)
			r.ExpiredControls = expired.controls()
			r.ExpiredDenied = first != nil && second != nil && first.Error() == second.Error() && r.ExpiredControls == 1 && expiredHTTPStatus == 409
			if !r.ExpiredDenied {
				t.Fatal("real expired confirmation resumed", first, second)
			}
			// A previously confirmed immutable original remains usable for that historical
			// sample; this is not a new claim that the evicted API generation is retained.
			_, ref, e := healthy.confirm(context.Background(), r.ExpiredID)
			r.CachedExpiredOriginalPreserved = e == nil && ref.ID == r.ExpiredID && healthy.controls() == len(r.IDs)
			if !r.CachedExpiredOriginalPreserved {
				t.Fatal("confirmed historical original changed")
			}
			r.CancelledID = s.overview(t).Generation
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			actual := lookup("cancelled")
			cancellationHTTPComplete := false
			cancelled := newDashboardWitnessRegistry(o.original, func(ctx context.Context, id string) (witnessPublication, error) {
				m, e := actual(ctx, id)
				cancellationHTTPComplete = e == nil
				cancel()
				return m, e
			})
			_, _, first = cancelled.confirm(ctx, r.CancelledID)
			_, _, second = cancelled.confirm(context.Background(), r.CancelledID)
			r.CancelledControls = cancelled.controls()
			r.CancelledDenied = first != nil && second != nil && first.Error() == second.Error() && r.CancelledControls == 1 && cancellationHTTPComplete
			if !r.CancelledDenied {
				t.Fatal("cancelled confirmation resumed", first, second)
			}
			idleDenied := newDashboardWitnessRegistry(o.original, lookup("idle-denied"))
			idleDenied.setIdle(true)
			_, _, first = idleDenied.confirm(context.Background(), previous)
			idleDenied.setIdle(false)
			_, _, second = idleDenied.confirm(context.Background(), previous)
			r.IdleControls = idleDenied.controls()
			r.IdleDenied = first != nil && second != nil && r.IdleControls == 0
			if !r.IdleDenied {
				t.Fatal("idle guard performed metadata or recovered")
			}
			if mode == "native-small" {
				r.BeforeIdle, e = o.control(witnessNativeCommand{Operation: "readback"})
				if e != nil {
					t.Fatal(e)
				}
				r.SpaceBefore = witnessLifecycleSpace(t, p.mount)
				usage := newWitnessNativeUsage(t, p.pair, o, probe.ABI)
				healthy.setIdle(true)
				window := &witnessIdleWindow{StartedAt: time.Now().UTC(), RequestsBefore: counter.requests.Load(), ControlsBefore: healthy.controls()}
				r.Idle = window
				started := time.Now()
				time.Sleep(300 * time.Second)
				window.ElapsedNS = int64(time.Since(started))
				window.FinishedAt = time.Now().UTC()
				window.RequestsAfter = counter.requests.Load()
				window.ControlsAfter = healthy.controls()
				healthy.setIdle(false)
				samples := usage.finish(t)
				r.Sampler = &usage.receipt
				if window.ElapsedNS < int64(300*time.Second) || window.RequestsBefore != window.RequestsAfter || window.ControlsBefore != window.ControlsAfter {
					t.Fatal("request/control in original idle window")
				}
				r.AfterIdle, e = o.control(witnessNativeCommand{Operation: "readback"})
				if e != nil {
					t.Fatal(e)
				}
				r.SpaceAfter = witnessLifecycleSpace(t, p.mount)
				if len(r.AfterIdle.Originals) <= len(r.BeforeIdle.Originals) {
					t.Fatal("no passive background collections")
				}
				// End the request-free window before any overview/control confirmation.
				id := s.ready(t, count, previous)
				confirm(id)
				t.Logf("NATIVE_WITNESS_IDLE seconds=%.3f requests=0 controls=0 source_originals_before=%d after=%d samples=%d missing=%d late=%d resource_acceptance=false", float64(window.ElapsedNS)/1e9, len(r.BeforeIdle.Originals), len(r.AfterIdle.Originals), len(samples), usage.receipt.Coverage.Missing, usage.receipt.Coverage.Late)
			}
			r.HealthyControls = healthy.controls()
			r.Final = o.stop(t)
			if r.Final.Failed || r.Final.StoreFailed || r.Final.Active || !r.Final.FrontRemoved || r.Final.ConnectorActive != 0 || r.Final.ConnectorStarted != r.Final.Connectors || r.Final.Connectors != len(r.Final.Originals) {
				t.Fatal("terminal broker uncertainty")
			}
			for _, v := range r.Final.Originals {
				ref := v.Ref
				if !ref.Complete || !ref.Delivered || ref.RawSHA256 != ref.ForwardSHA256 || ref.CaptureFailure != "" || ref.Forward == nil || !ref.Forward.WriteHalfClosed || ref.Forward.Bytes != ref.RawBytes {
					t.Fatal("original full delivery")
				}
			}
			b := r.Final.ForwardBoundary
			if b == nil || b.DeadlineNS-b.StartedNS != int64(45*time.Second) || b.DeadlineReached || b.ContextDeadlineExceeded || b.ContextCancelled {
				t.Fatal("original forwarding deadline")
			}
			r.Reaped = !witnessLifetime(r.Observer, probe.ABI)
			_, e = witnessRootSocket(o.config.Back, o.config.SocketGID)
			r.HelperUnchanged = e == nil && o.config.Helper.check(probe.ABI) == nil
			config, ce := os.ReadFile(p.pair.path)
			back, be := os.ReadFile(p.backPath)
			token, te := os.ReadFile(filepath.Join(p.pair.config.StateDirectory, "token.sha256"))
			r.ConfigurationUnchanged = ce == nil && be == nil && te == nil && bytes.Equal(config, p.configBytes) && bytes.Equal(back, p.backBytes) && bytes.Equal(token, p.digest)
			r.Originals, r.Indexes = witnessControllerExport(t, p, out)
			if !r.Reaped || !r.HelperUnchanged || !r.ConfigurationUnchanged {
				t.Fatal("actor/configuration preservation")
			}
			if witnessWritePrivateIndex(out, "receipt.json", r) != nil {
				t.Fatal("case receipt")
			}
			p.pair.management.stop(t, syscall.SIGTERM, true)
			p.pair.helper.stop(t, syscall.SIGTERM, true)
			t.Logf("NATIVE_WITNESS_LIFECYCLE mode=%s confirmed_ids=%d healthy_controls=%d expired_controls=%d cancelled_controls=%d idle_controls=%d originals=%d resource_acceptance=false", mode, len(r.IDs), r.HealthyControls, r.ExpiredControls, r.CancelledControls, r.IdleControls, len(r.Final.Originals))
		})
	}
}
