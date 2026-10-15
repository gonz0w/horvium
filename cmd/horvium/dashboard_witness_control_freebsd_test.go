//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

type witnessControlCaseReceipt struct {
	TypedControlRecorded                                                                               bool
	Label                                                                                              string
	InjectedSHA256                                                                                     string
	InjectedBytes                                                                                      int
	Observer, Controller                                                                               witnessActor
	Final                                                                                              witnessNativeReceipt
	OriginalID                                                                                         string
	OriginalBefore, OldOriginalDenied, FollowupDenied, Reaped, HelperUnchanged, ConfigurationUnchanged bool
}

func TestNativeDashboardWitnessControlChannel(t *testing.T) {
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
		if witnessWritePrivateIndex("../../.tools", "witness-control-cleanup.json", struct{ Verified bool }{clean}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("control channel cleanup uncertain")
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
	_ = nativeStorageFixtureObserved(t, owner.storage)
	count := len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
	export := "../../.tools/witness-control-cases"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive case export")
	}
	cases := witnessControlAdverse()
	cases["unknown-operation"] = witnessControlFrame([]byte(`{"operation":"untrusted"}`))
	labels := []string{"normal-boundary"}
	var adverse []string
	for label := range cases {
		adverse = append(adverse, label)
	}
	sort.Strings(adverse)
	labels = append(labels, adverse...)
	for _, label := range labels {
		t.Run(label, func(t *testing.T) {
			p := startWitnessPairWithFault(t, owner, probe, "native-small", "", label)
			o := p.observer
			p.pair.session.ready(t, count, "")
			id := p.pair.session.overview(t).Generation
			snapshot, ref, e := o.original(id)
			if e != nil || !snapshot.Complete() || !ref.Complete || !ref.Delivered {
				t.Fatal("pre-fault actual published original", e)
			}
			actor, e := witnessObserveActor(o.process.cmd.Process.Pid, probe.ABI)
			if e != nil {
				t.Fatal("original observer")
			}
			r := witnessControlCaseReceipt{Label: label, Observer: actor, Controller: o.config.Parent, OriginalID: id, OriginalBefore: true}
			_ = o.channel.SetDeadline(time.Now().Add(45 * time.Second))
			if label == "normal-boundary" {
				payload := []byte(`{"operation":"readback"}`)
				payload = append(payload, bytes.Repeat([]byte(" "), witnessFrameCap-len(payload))...)
				raw := witnessControlFrame(payload)
				r.InjectedBytes = len(raw)
				r.InjectedSHA256 = fmt.Sprintf("%x", sha256.Sum256(raw))
				if n, e := o.channel.Write(raw); e != nil || n != len(raw) {
					t.Fatal("actual exact-cap write", e)
				}
				var readback witnessNativeReceipt
				if witnessControlRead(o.channel, &readback) != nil || !readback.OK || readback.Failed {
					t.Fatal("exact-cap readback")
				}
				if _, _, e := o.original(id); e != nil {
					t.Fatal("normal boundary invalidated cached original")
				}
				r.Final = o.stop(t)
			} else {
				raw := cases[label]
				r.InjectedBytes = len(raw)
				r.InjectedSHA256 = fmt.Sprintf("%x", sha256.Sum256(raw))
				if label == "unknown-operation" {
					var sent bytes.Buffer
					if _, e := o.controlRecorded(witnessNativeCommand{Operation: "untrusted"}, &sent); e == nil {
						t.Error("unknown operation accepted")
					}
					raw = sent.Bytes()
					r.InjectedBytes = len(raw)
					r.InjectedSHA256 = fmt.Sprintf("%x", sha256.Sum256(raw))
					r.TypedControlRecorded = true
					if witnessWritePrivateIndex(export, label+".control", struct{ Bytes []byte }{raw}) != nil {
						t.Fatal("exclusive actual typed control bytes")
					}
					if witnessControlWrite(o.channel, witnessNativeCommand{Operation: "stop"}) != nil {
						t.Fatal("failed owner stop")
					}
				} else {
					if n, e := o.channel.Write(raw); e != nil || n != len(raw) {
						t.Fatal("actual malformed write", e)
					}
					if o.channel.CloseWrite() != nil {
						t.Fatal("original control write EOF")
					}
				}
				if witnessControlRead(o.channel, &r.Final) != nil {
					t.Fatal("original terminal receipt")
				}
				// Force an actual post-failure control attempt before checking the cached
				// original. Keep its root open so close-induced failure cannot fake denial.
				_, e = o.control(witnessNativeCommand{Operation: "readback"})
				r.FollowupDenied = e != nil
				_, _, e = o.original(id)
				r.OldOriginalDenied = e != nil
				select {
				case e := <-o.process.done:
					o.process.waited = true
					if e != nil {
						t.Error("observer original exit", e)
					}
				case <-time.After(10 * time.Second):
					t.Fatal("observer stop timeout")
				}
				o.channel.Close()
				o.root.Close()
				o.closed = true
				o.final = r.Final
				if !r.Final.OK || !r.Final.Failed || !r.Final.FrontRemoved || !r.OldOriginalDenied || !r.FollowupDenied {
					t.Error("control uncertainty did not permanently deny observer and cached original")
				}
				want := "control-read"
				if label == "unknown-operation" {
					want = "control-operation"
				}
				if r.Final.Failure != want {
					t.Error("wrong terminal control failure", r.Final.Failure)
				}
			}
			r.Reaped = !witnessLifetime(actor, probe.ABI) && r.Final.ConnectorStarted == r.Final.Connectors && r.Final.ConnectorActive == 0 && !r.Final.Active
			_, e = witnessRootSocket(o.config.Back, o.config.SocketGID)
			r.HelperUnchanged = e == nil && o.config.Helper.check(probe.ABI) == nil
			cfg, ce := os.ReadFile(p.pair.path)
			back, be := os.ReadFile(p.backPath)
			token, te := os.ReadFile(filepath.Join(p.pair.config.StateDirectory, "token.sha256"))
			r.ConfigurationUnchanged = ce == nil && be == nil && te == nil && bytes.Equal(cfg, p.configBytes) && bytes.Equal(back, p.backBytes) && bytes.Equal(token, p.digest)
			if _, e := os.Lstat(o.config.Front); !os.IsNotExist(e) {
				t.Error("front socket remains")
			}
			if !r.Reaped || !r.HelperUnchanged || !r.ConfigurationUnchanged {
				t.Error("original reaping/helper/config custody")
			}
			if witnessWritePrivateIndex(export, label+".json", r) != nil {
				t.Fatal("exclusive original case receipt")
			}
			t.Logf("NATIVE_WITNESS_CONTROL label=%s refused=%t original_preserved=true resource_acceptance=false", label, r.Final.Failed)
		})
	}
}
