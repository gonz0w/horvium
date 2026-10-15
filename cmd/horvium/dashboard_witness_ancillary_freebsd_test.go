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
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func witnessAncillaryFaults() []string {
	return []string{"eof", "missing-rights", "wrong-tag", "extra-rights", "truncated-rights", "non-socket", "wrong-endpoint", "replay-before-ack", "replay-after-ack", "extra-byte"}
}
func witnessAncillaryFaultValid(fault string) bool {
	if fault == "" {
		return true
	}
	for _, valid := range witnessAncillaryFaults() {
		if fault == valid {
			return true
		}
	}
	return false
}
func witnessConfiguredAncillary(cfg witnessNativeConfig) string {
	if cfg.FaultStage == "ancillary" {
		return strings.TrimPrefix(cfg.RunLabel, "ancillary-")
	}
	return ""
}

// Only this descriptor-only scope lowers its own soft limit, then restores it
// before any product or observer is launched. Received descriptors must fit the
// census, including the delivered prefix of an actual MSG_CTRUNC message.
func witnessNativeDescriptorCensus(t *testing.T) {
	var original unix.Rlimit
	if unix.Getrlimit(unix.RLIMIT_NOFILE, &original) != nil || original.Cur < 128 {
		t.Fatal("descriptor census limit unavailable")
	}
	bounded := original
	bounded.Cur = 128
	if unix.Setrlimit(unix.RLIMIT_NOFILE, &bounded) != nil {
		t.Fatal("descriptor census bound")
	}
	defer func() {
		if unix.Setrlimit(unix.RLIMIT_NOFILE, &original) != nil {
			t.Error("descriptor limit restore")
		}
	}()
	census := func() map[int]int {
		out := map[int]int{}
		for fd := 0; fd < 128; fd++ {
			flags, e := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
			if e == nil {
				out[fd] = flags
			} else if e != unix.EBADF {
				t.Fatal("descriptor census uncertain", e)
			}
		}
		return out
	}
	t.Run("warm-poller", func(t *testing.T) { witnessTestPair(t) })
	before := census()
	observations := map[string]map[int]int{}
	for _, tc := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"ownership-cloexec-gc", TestDashboardWitnessDescriptorOwnershipSurvivesGC},
		{"wrong-extra-truncated-nonsocket-closed", TestDashboardWitnessDescriptorRejectsExtraAndWrongTag},
		{"sender-exit-half-close", TestDashboardWitnessTransferredEndpointSurvivesSenderClosure},
		{"channel-replay-eof", TestDashboardWitnessDescriptorChannelMustEnd},
	} {
		t.Run(tc.name, tc.run)
		after := census()
		observations[tc.name] = after
		if !reflect.DeepEqual(before, after) {
			t.Fatal("received descriptor leak or unrelated descriptor changed", tc.name)
		}
		t.Logf("NATIVE_WITNESS_DESCRIPTOR_CENSUS case=%s bound=128 descriptors_unchanged=true resource_acceptance=false", tc.name)
	}
	if witnessWritePrivateIndex("../../.tools", "witness-ancillary-descriptors.json", struct {
		Bound  int
		Before map[int]int
		After  map[string]map[int]int
	}{128, before, observations}) != nil {
		t.Fatal("original descriptor census")
	}
}

type witnessAncillaryCaseReceipt struct {
	Label                                            string
	Observer, Controller                             witnessActor
	ABI                                              witnessABI
	Final                                            witnessNativeReceipt
	Refused, HelperUnchanged, ConfigurationUnchanged bool
}

func TestNativeDashboardWitnessAncillary(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("disposable VM opt-in")
	}
	if os.Getuid() != 0 {
		t.Fatal("root fixture owner required")
	}
	var initial unix.Rlimit
	if unix.Getrlimit(unix.RLIMIT_NOFILE, &initial) != nil {
		t.Fatal("original descriptor limit")
	}
	if !t.Run("descriptor-census", witnessNativeDescriptorCensus) {
		return
	}
	var restored unix.Rlimit
	if unix.Getrlimit(unix.RLIMIT_NOFILE, &restored) != nil || restored != initial {
		t.Fatal("original product descriptor limit not restored")
	}
	if witnessWritePrivateIndex("../../.tools", "witness-ancillary-limits.json", struct{ Original, Restored unix.Rlimit }{initial, restored}) != nil {
		t.Fatal("original descriptor limit receipt")
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
			t.Error("ancillary cleanup uncertain")
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
	export := "../../.tools/witness-ancillary"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive ancillary export")
	}
	t.Run("normal-control", func(t *testing.T) {
		p := startWitnessPair(t, owner, probe, "native-small")
		o := p.observer
		if o.faultChannel != nil {
			t.Fatal("normal channel injected")
		}
		p.pair.session.ready(t, count, "")
		generation := p.pair.session.overview(t).Generation
		snapshot, ref, e := o.original(generation)
		if e != nil || !snapshot.Complete() || !ref.Complete || !ref.Delivered || ref.RawSHA256 != ref.ForwardSHA256 {
			t.Fatal("normal source confirmation", e)
		}
		final := o.stop(t)
		if final.ConnectorStarted != final.Connectors || final.ConnectorActive != 0 || final.ConnectorEOFChecks != final.Connectors || final.ConnectorEOFRejected != 0 {
			t.Fatal("normal original handoff EOF not checked")
		}
		if witnessWritePrivateIndex("../../.tools", "witness-ancillary-normal.json", final) != nil {
			t.Fatal("normal original receipt")
		}
		t.Log("NATIVE_WITNESS_ANCILLARY_NORMAL original_confirmed=true exact_channel_eof=true all_connectors_reaped=true resource_acceptance=false")
	})
	for _, fault := range witnessAncillaryFaults() {
		label := "ancillary-" + fault
		t.Run(label, func(t *testing.T) {
			p := startWitnessPairWithFault(t, owner, probe, "native-small", "ancillary", label)
			o := p.observer
			observer, e := witnessObserveActor(o.process.cmd.Process.Pid, probe.ABI)
			if e != nil {
				t.Fatal("original observer")
			}
			var final witnessNativeReceipt
			performanceWait(t, func() bool {
				final, e = o.control(witnessNativeCommand{Operation: "readback"})
				return e != nil && final.Failed && final.ConnectorStarted > 0 && final.ConnectorActive == 0
			})
			if _, _, e := o.original("unconfirmed"); e == nil {
				t.Fatal("ambiguous channel accepted original")
			}
			_ = o.channel.SetDeadline(time.Now().Add(10 * time.Second))
			if witnessControlWrite(o.channel, witnessNativeCommand{Operation: "stop"}) != nil || witnessControlRead(o.channel, &final) != nil {
				t.Fatal("failed owner original final")
			}
			select {
			case e := <-o.process.done:
				o.process.waited = true
				if e != nil {
					t.Fatal("observer exit")
				}
			case <-time.After(10 * time.Second):
				t.Fatal("observer stop timeout")
			}
			o.channel.Close()
			o.root.Close()
			o.closed = true
			o.final = final
			// Retain original facts before semantic assertions, including failed cases.
			original := witnessAncillaryCaseReceipt{Label: label, Observer: observer, Controller: o.config.Parent, ABI: probe.ABI, Final: final}
			if witnessWritePrivateIndex("../../.tools", "witness-ancillary-original-"+fault+".json", original) != nil {
				t.Fatal("original case context")
			}
			wantStage := "connector-descriptor"
			wantChecks, wantRejected := 0, 0
			if fault == "wrong-endpoint" {
				wantStage = "helper-peer"
			}
			if fault == "replay-before-ack" || fault == "replay-after-ack" || fault == "extra-byte" {
				wantStage = "connector-channel-eof"
				wantChecks = 1
				wantRejected = 1
			}
			if !final.OK || !final.Failed || !final.FrontRemoved || final.Failure != wantStage || len(final.Originals) != 0 || final.ConnectorStarted != final.Connectors || final.ConnectorActive != 0 || final.ConnectorMaxRSS == 0 || final.ConnectorCPUNS <= 0 || final.ConnectorEOFChecks != wantChecks || final.ConnectorEOFRejected != wantRejected {
				t.Fatal("original ancillary rejection/wait mismatch", final.Failure)
			}
			entries, e := os.ReadDir(p.directory)
			if e != nil || len(entries) != 0 {
				t.Fatal("ambiguous channel produced original")
			}
			if witnessLifetime(observer, probe.ABI) {
				t.Fatal("original observer alive")
			}
			if _, e := os.Lstat(o.config.Front); !os.IsNotExist(e) {
				t.Fatal("front remains")
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
			original.Refused = true
			original.HelperUnchanged = true
			original.ConfigurationUnchanged = true
			if witnessWritePrivateIndex(export, label+".json", original) != nil {
				t.Fatal("case receipt")
			}
			t.Logf("NATIVE_WITNESS_ANCILLARY fault=%s refused=true originals=0 all_connectors_reaped=true resource_acceptance=false", fault)
		})
	}
}
