//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type witnessBrokerFill struct {
	Bytes, ReportedWriteBytes               int64
	SHA256                                  string
	Object                                  witnessFileObject
	ENOSPC, SmallProbeENOSPC, SpaceReleased bool
}
type witnessBrokerSpaceCase struct {
	Label, Mode, Publication                        string
	Owner, Observer, Management, Helper             witnessActor
	Before, Boundary, Followup, Final               witnessNativeReceipt
	Fill                                            *witnessBrokerFill `json:",omitempty"`
	RefreshStatus                                   int
	OriginalDenied, LastCollectionFailed            bool
	Reaped, HelperUnchanged, ConfigurationUnchanged bool
	Originals, Indexes                              []witnessInterruptedOriginal
}

// All filling and readback occur before management starts, outside its unchanged
// collection deadline. The archive counters/caps and 320 MiB mount are untouched.
func witnessFillBrokerMount(t *testing.T, mount string) (*witnessBrokerFill, string) {
	t.Helper()
	path := filepath.Join(mount, "owned-broker-filler")
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		f.Close()
		if _, e := os.Lstat(path); e == nil {
			if os.Remove(path) != nil {
				t.Error("filler cleanup")
			}
		}
	})
	r := &witnessBrokerFill{}
	buffer := make([]byte, witnessFrameCap)
	for size := len(buffer); size >= 1; size /= 2 {
		for r.Bytes <= witnessCapacityTmpfs {
			n, e := f.Write(buffer[:size])
			r.ReportedWriteBytes += int64(n)
			info, se := f.Stat()
			if se != nil {
				t.Fatal(se)
			}
			r.Bytes = info.Size()
			if e != nil {
				if !errors.Is(e, syscall.ENOSPC) {
					t.Fatal("unexpected fill error", e)
				}
				r.ENOSPC = true
				if _, e = f.Seek(0, io.SeekEnd); e != nil {
					t.Fatal(e)
				}
				break
			}
			if n != size {
				t.Fatal("short fill without ENOSPC")
			}
		}
	}
	if !r.ENOSPC || r.Bytes <= 0 || r.Bytes > witnessCapacityTmpfs {
		t.Fatal("physical cap not reached")
	}
	probePath := filepath.Join(mount, "owned-broker-space-probe")
	probe, pe := os.OpenFile(probePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if pe != nil {
		r.SmallProbeENOSPC = errors.Is(pe, syscall.ENOSPC)
	} else {
		n, we := probe.Write([]byte{0})
		probe.Close()
		r.SmallProbeENOSPC = n == 0 && errors.Is(we, syscall.ENOSPC)
		if os.Remove(probePath) != nil {
			t.Fatal("probe cleanup")
		}
	}
	if !r.SmallProbeENOSPC {
		t.Fatal("new one-byte allocation still possible")
	}
	before, e := f.Stat()
	object, oe := witnessObject(before)
	if e != nil || oe != nil || object.UID != 0 || object.Links != 1 || object.Size != r.Bytes {
		t.Fatal("filler object custody")
	}
	r.Object = object
	if f.Close() != nil {
		t.Fatal("filler close")
	}
	source, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	h := sha256.New()
	n, re := io.CopyBuffer(h, source, buffer)
	source.Close()
	after, ae := os.Lstat(path)
	if re != nil || ae != nil || !os.SameFile(before, after) || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) || n != r.Bytes {
		t.Fatal("filler original byte readback")
	}
	r.SHA256 = fmt.Sprintf("%x", h.Sum(nil))
	return r, path
}

func TestNativeDashboardWitnessBrokerENOSPC(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("disposable VM opt-in")
	}
	if os.Getuid() != 0 {
		t.Fatal("root owner required")
	}
	if witnessCapacityTmpfs != 335544320 || witnessArchiveCap != 268435456 || witnessAttemptCap != 32 || witnessRawCap != 67508864 {
		t.Fatal("accepted native caps changed")
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
		if witnessWritePrivateIndex("../../.tools", "witness-enospc-cleanup.json", struct{ Verified bool }{clean}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("broker ENOSPC cleanup uncertain")
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
	export := "../../.tools/witness-enospc-cases"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive case export")
	}
	for _, action := range []string{"normal", "enospc"} {
		for _, mode := range []string{"native-small", "projection-10000"} {
			label := action + "-" + mode
			t.Run(label, func(t *testing.T) {
				r := witnessBrokerSpaceCase{Label: label, Mode: mode}
				fillPath := ""
				start := func(t *testing.T, cfg witnessNativeConfig, path string) *witnessFixtureObserver {
					o := startWitnessObserver(t, cfg, path)
					var e error
					r.Before, e = o.control(witnessNativeCommand{Operation: "readback"})
					if e != nil || r.Before.Active || len(r.Before.Originals) != 0 || r.Before.ConnectorStarted != 0 {
						t.Fatal("management started before physical setup")
					}
					if action == "enospc" {
						r.Fill, fillPath = witnessFillBrokerMount(t, filepath.Dir(cfg.Directory))
					}
					return o
				}
				p := startWitnessPairUsingObserver(t, owner, probe, mode, "", label, start)
				o := p.observer
				t.Cleanup(func() {
					if !o.closed {
						diagnostic, _ := o.control(witnessNativeCommand{Operation: "readback"})
						if witnessWritePrivateIndex(export, label+"-failure.json", diagnostic) != nil {
							t.Error("failure diagnostic")
						}
					}
				})
				r.Owner = o.config.Parent
				r.Helper = o.config.Helper
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
					r.Boundary = r.Final
					r.Followup = r.Final
				} else {
					performanceWait(t, func() bool {
						r.Boundary, _ = o.control(witnessNativeCommand{Operation: "readback"})
						return r.Boundary.Failed && !r.Boundary.Active
					})
					b := r.Boundary.CaptureBoundary
					if r.Boundary.Failure != "producer-capture" || !r.Boundary.StoreFailed || r.Boundary.ConnectorStarted != 1 || r.Boundary.Connectors != 1 || r.Boundary.ConnectorActive != 0 || b == nil || b.Error.Class != "syscall" || b.Error.Errno != uint64(syscall.ENOSPC) || b.ID == "" || b.Management != r.Management.incarnation() || b.Helper != r.Helper.incarnation() {
						t.Fatal("actual capture ENOSPC/actor facts missing", r.Boundary.Failure, b)
					}
					for _, v := range r.Boundary.Originals {
						if v.Ref.Complete || v.Ref.Delivered || v.Ref.Forward != nil {
							t.Fatal("ENOSPC original accepted")
						}
					}
					_, _, e = o.original(b.ID)
					r.OriginalDenied = e != nil
					if os.Remove(fillPath) != nil {
						t.Fatal("filler removal")
					}
					var freed unix.Statfs_t
					r.Fill.SpaceReleased = unix.Statfs(p.mount, &freed) == nil && freed.Fsid == p.mountFS.Fsid && freed.Bavail > 0
					recovery := filepath.Join(p.mount, "owned-after-enospc")
					if os.WriteFile(recovery, []byte("space released"), 0600) != nil || os.Remove(recovery) != nil || !r.Fill.SpaceReleased {
						t.Fatal("physical allocation did not recover")
					}
					performanceWait(t, func() bool { return witnessBrokerCollectionIdle(t, p.pair.session) })
					r.RefreshStatus = witnessAdmissionRefresh(t, p.pair.session, true)
					performanceWait(t, func() bool {
						r.Followup, _ = o.control(witnessNativeCommand{Operation: "readback"})
						return !r.Followup.Active && r.Followup.Denied > r.Boundary.Denied
					})
					performanceWait(t, func() bool { return witnessBrokerCollectionIdle(t, p.pair.session) })
					response, e := p.pair.session.http.Get(p.pair.session.base + "/status")
					if e != nil {
						t.Fatal(e)
					}
					body, re := io.ReadAll(io.LimitReader(response.Body, 65537))
					response.Body.Close()
					r.LastCollectionFailed = re == nil && response.StatusCode == 200 && bytes.Contains(body, []byte("Last collection:")) && bytes.Contains(body, []byte("Cached data is stale."))
					if !r.OriginalDenied || !r.LastCollectionFailed || !reflect.DeepEqual(r.Boundary.Originals, r.Followup.Originals) || !reflect.DeepEqual(r.Boundary.CaptureBoundary, r.Followup.CaptureBoundary) || r.Followup.ConnectorStarted != 1 || r.Followup.Connectors != 1 || r.Followup.ConnectorActive != 0 {
						t.Fatal("freed space resumed failed candidate or erased evidence")
					}
					unavailable, ue := p.pair.session.http.Get(p.pair.session.base + "/api/overview")
					if ue != nil {
						t.Fatal(ue)
					}
					_, ue = io.Copy(io.Discard, io.LimitReader(unavailable.Body, 65537))
					unavailable.Body.Close()
					if ue != nil || unavailable.StatusCode != 503 {
						t.Fatal("failed initial capture published", unavailable.StatusCode, ue)
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
				if entries, e := os.ReadDir(p.directory); e != nil {
					t.Fatal(e)
				} else if len(entries) > 0 {
					r.Originals, r.Indexes = witnessControllerExport(t, p, out)
				} else {
					p.originalsExported = true
				}
				if witnessWritePrivateIndex(out, "receipt.json", r) != nil {
					t.Fatal("case receipt")
				}
				if !r.Reaped || !r.HelperUnchanged || !r.ConfigurationUnchanged {
					t.Fatal("teardown/actor/config uncertainty")
				}
				p.pair.management.stop(t, syscall.SIGTERM, true)
				p.pair.helper.stop(t, syscall.SIGTERM, true)
				t.Logf("NATIVE_WITNESS_BROKER_SPACE label=%s enospc=%t originals=%d publication_confirmed=%t resource_acceptance=false", label, r.Fill != nil, len(r.Originals), r.Publication != "")
			})
		}
	}
}
func witnessBrokerCollectionIdle(t *testing.T, s *performanceSession) bool {
	t.Helper()
	r, e := s.http.Get(s.base + "/status")
	if e != nil {
		t.Fatal(e)
	}
	defer r.Body.Close()
	b, e := io.ReadAll(io.LimitReader(r.Body, 65537))
	return e == nil && r.StatusCode == 200 && !bytes.Contains(b, []byte("Collecting an observation"))
}
