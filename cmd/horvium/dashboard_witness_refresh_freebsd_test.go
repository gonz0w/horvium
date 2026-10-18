//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

type witnessRefreshCase struct {
	Label, Mode, InitialPublication, LaterPublication string
	Owner, Observer, Management, Helper               witnessActor
	Event                                             witnessFaultEvent
	Before, Held, Final                               witnessNativeReceipt
	Replies                                           []witnessRefreshReply
	LaterReply                                        witnessRefreshReply
	AdditionalOriginalsAfterRelease                   int
	Reaped, HelperUnchanged, ConfigurationUnchanged   bool
	Originals, Indexes                                []witnessInterruptedOriginal
}

func TestNativeDashboardWitnessRefreshOverlap(t *testing.T) {
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
		if witnessWritePrivateIndex("../../.tools", "witness-refresh-cleanup.json", struct{ Verified bool }{clean}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("refresh overlap cleanup uncertain")
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
	export := "../../.tools/witness-refresh-cases"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive case export")
	}

	for _, mode := range []string{"native-small", "projection-10000"} {
		t.Run(mode, func(t *testing.T) {
			label := "refresh-" + mode
			p := startWitnessPairWithFault(t, owner, probe, mode, "sealed", label)
			o := p.observer
			r := witnessRefreshCase{Label: label, Mode: mode, Owner: o.config.Parent, Helper: o.config.Helper}
			// Preserve partial observations, including successful HTTP replies, on failure.
			t.Cleanup(func() {
				if t.Failed() {
					_ = witnessWritePrivateIndex(export, label+"-partial.json", r)
					if !o.closed {
						d, _ := o.control(witnessNativeCommand{Operation: "readback"})
						_ = witnessWritePrivateIndex(export, label+"-failure.json", d)
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
			_ = o.faultChannel.SetReadDeadline(time.Now().Add(45 * time.Second))
			if witnessControlRead(o.faultChannel, &r.Event) != nil || r.Event.Stage != "sealed" || !dashboardGenerationPattern.MatchString(r.Event.ID) {
				t.Fatal("actual sealed boundary")
			}
			r.Before, e = o.control(witnessNativeCommand{Operation: "readback"})
			if e != nil || !r.Before.Active || r.Before.Failed || r.Before.StoreFailed || len(r.Before.Originals) != 1 || r.Before.ConnectorStarted != 1 || r.Before.Connectors != 1 || r.Before.ConnectorActive != 0 {
				t.Fatal("original collection not held")
			}
			first := r.Before.Originals[0].Ref
			if first.ID != r.Event.ID || !first.Complete || first.Delivered || first.Forward != nil {
				t.Fatal("sealed original boundary")
			}
			session := p.pair.session
			r.Replies, e = witnessRefreshOverlap(session.http, session.base, session.csrf)
			if e != nil {
				t.Fatal("actual refresh dispatch", e)
			}
			for _, reply := range r.Replies {
				if !reply.busy() || reply.StartedNS < r.Event.StartedNS || reply.FinishedNS >= r.Event.DeadlineNS {
					t.Fatal("authenticated overlap did not finish busy within original deadline")
				}
			}
			r.Held, e = o.control(witnessNativeCommand{Operation: "readback"})
			if e != nil || !r.Held.Active || r.Held.Failed || r.Held.StoreFailed || r.Held.Denied != r.Before.Denied || r.Held.ConnectorStarted != 1 || r.Held.Connectors != 1 || r.Held.ConnectorActive != 0 || !reflect.DeepEqual(r.Before.Originals, r.Held.Originals) {
				t.Fatal("busy refresh changed original or started broker work")
			}
			if witnessControlWrite(o.faultChannel, witnessNativeCommand{Operation: "release"}) != nil {
				t.Fatal("release original once")
			}
			count := nativeCount
			if mode == "projection-10000" {
				count = 10000
			}
			r.InitialPublication = session.ready(t, count, "")
			snapshot, ref, e := o.original(r.InitialPublication)
			if e != nil || !snapshot.Complete() || ref.ID != r.Event.ID || !ref.Delivered {
				t.Fatal("initial publication original", e)
			}
			r.LaterReply, e = witnessRefreshOnce(session.http, session.base, session.csrf, 4)
			if e != nil || !(r.LaterReply.accepted() || r.LaterReply.busy()) {
				t.Fatal("one later authenticated refresh", e)
			}
			r.LaterPublication = session.ready(t, count, r.InitialPublication)
			snapshot, ref, e = o.original(r.LaterPublication)
			if e != nil || !snapshot.Complete() || ref.ID == r.InitialPublication || !ref.Delivered {
				t.Fatal("later publication original", e)
			}
			snapshot, ref, e = o.original(r.InitialPublication)
			if e != nil || !snapshot.Complete() || ref.ID != r.Event.ID || !ref.Delivered {
				t.Fatal("retained initial original", e)
			}
			r.Final = o.stop(t)
			o.faultChannel.Close()
			if r.Final.Failed || r.Final.StoreFailed || r.Final.Active || !r.Final.FrontRemoved || len(r.Final.Originals) < 2 || r.Final.Connectors != len(r.Final.Originals) || r.Final.ConnectorStarted != r.Final.Connectors || r.Final.ConnectorActive != 0 || r.Final.Denied != r.Before.Denied {
				t.Fatal("final collection/teardown uncertainty")
			}
			r.AdditionalOriginalsAfterRelease = len(r.Final.Originals) - 2
			for _, original := range r.Final.Originals {
				ref := original.Ref
				if !ref.Complete || !ref.Delivered || ref.CaptureFailure != "" || ref.RawSHA256 != ref.ForwardSHA256 || ref.Forward == nil || ref.Forward.Stage != "delivered" || ref.Forward.Bytes != ref.RawBytes || !ref.Forward.WriteHalfClosed {
					t.Fatal("original full delivery")
				}
			}
			b := r.Final.ForwardBoundary
			if b == nil || b.DeadlineNS-b.StartedNS != int64(45*time.Second) || b.DeadlineReached || b.ContextDeadlineExceeded || b.ContextCancelled {
				t.Fatal("original deadline changed")
			}
			r.Reaped = !witnessLifetime(r.Observer, probe.ABI)
			_, e = witnessRootSocket(o.config.Back, o.config.SocketGID)
			r.HelperUnchanged = e == nil && o.config.Helper.check(probe.ABI) == nil
			config, ce := os.ReadFile(p.pair.path)
			back, be := os.ReadFile(p.backPath)
			token, te := os.ReadFile(filepath.Join(p.pair.config.StateDirectory, "token.sha256"))
			r.ConfigurationUnchanged = ce == nil && be == nil && te == nil && bytes.Equal(config, p.configBytes) && bytes.Equal(back, p.backBytes) && bytes.Equal(token, p.digest)
			out := filepath.Join(export, mode)
			if os.Mkdir(out, 0700) != nil {
				t.Fatal("exclusive case export")
			}
			r.Originals, r.Indexes = witnessControllerExport(t, p, out)
			if witnessWritePrivateIndex(out, "receipt.json", r) != nil {
				t.Fatal("case receipt")
			}
			if !r.Reaped || !r.HelperUnchanged || !r.ConfigurationUnchanged {
				t.Fatal("actor/configuration preservation")
			}
			p.pair.management.stop(t, syscall.SIGTERM, true)
			p.pair.helper.stop(t, syscall.SIGTERM, true)
			t.Logf("NATIVE_WITNESS_REFRESH mode=%s busy=3 publications=2 originals=%d additional_after_release=%d resource_acceptance=false", mode, len(r.Final.Originals), r.AdditionalOriginalsAfterRelease)
		})
	}
}
