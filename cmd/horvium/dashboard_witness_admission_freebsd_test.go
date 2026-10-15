//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

type witnessAdmissionPublication struct {
	ID       string
	Complete bool
	Datasets int
}
type witnessAdmissionReceipt struct {
	Label, Mode                                               string
	Publications                                              []witnessAdmissionPublication
	Before, Capacity, Boundary, Followup, Final               witnessNativeReceipt `json:"-"`
	Snapshots                                                 map[string]string
	RefreshStatuses                                           []int
	BoundaryRefreshStatus, FollowupRefreshStatus              int
	Owner, Observer, Management, Helper                       witnessActor
	Control                                                   []byte
	OldOriginalDenied, FollowupRejected, LastCollectionFailed bool
	Reaped, HelperUnchanged, ConfigurationUnchanged           bool
	Originals, Indexes                                        []witnessInterruptedOriginal
}

// A single explicit operator request, never a workload retry or a synthetic source.
func witnessAdmissionRefresh(t *testing.T, s *performanceSession, allowBusy bool) int {
	t.Helper()
	r, e := s.http.PostForm(s.base+"/api/refresh", url.Values{"csrf": {s.csrf}})
	if e != nil {
		t.Fatal("actual manager refresh", e)
	}
	_, e = io.Copy(io.Discard, io.LimitReader(r.Body, 65537))
	r.Body.Close()
	if e != nil || r.StatusCode != http.StatusAccepted && !(allowBusy && r.StatusCode == http.StatusConflict) {
		t.Fatal("single refresh not accepted", r.StatusCode, e)
	}
	return r.StatusCode
}

func TestNativeDashboardWitnessAdmission(t *testing.T) {
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
		if witnessWritePrivateIndex("../../.tools", "witness-admission-cleanup.json", struct{ Verified bool }{clean}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("admission cleanup uncertain")
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
	export := "../../.tools/witness-admission-cases"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive case root")
	}
	for _, label := range []string{"control-failure", "native-small", "projection-10000"} {
		t.Run(label, func(t *testing.T) {
			mode, count, attempts := "native-small", nativeCount, 32
			if label == "control-failure" {
				attempts = 1
			}
			if label == "projection-10000" {
				mode, count = "projection-10000", 10000
			}
			p := startWitnessPairWithFault(t, owner, probe, mode, "", "admission-"+label)
			o, s := p.observer, p.pair.session
			// Preserve the actual failure reason even if readiness aborts before
			// the case receipt can be completed. This diagnostic is not acceptance.
			t.Cleanup(func() {
				if !o.closed {
					diagnostic, _ := o.control(witnessNativeCommand{Operation: "readback"})
					if witnessWritePrivateIndex(export, label+"-failure.json", diagnostic) != nil {
						t.Error("bounded failure diagnostic")
					}
				}
			})
			r := witnessAdmissionReceipt{Label: label, Mode: mode, Owner: o.config.Parent, Helper: o.config.Helper}
			var e error
			r.Observer, e = witnessObserveActor(o.process.cmd.Process.Pid, probe.ABI)
			if e != nil {
				t.Fatal(e)
			}
			r.Management, e = witnessObserveActor(p.pair.management.cmd.Process.Pid, probe.ABI)
			if e != nil {
				t.Fatal(e)
			}
			// Authentication readiness precedes the first inventory publication.
			s.ready(t, count, "")
			previous := ""
			for {
				performanceWait(t, func() bool {
					overview := s.overview(t)
					if !overview.Complete || overview.Generation == previous || overview.DatasetCount != count {
						return false
					}
					if !dashboardGenerationPattern.MatchString(overview.Generation) {
						t.Fatal("actual publication ID")
					}
					previous = overview.Generation
					r.Publications = append(r.Publications, witnessAdmissionPublication{previous, overview.Complete, overview.DatasetCount})
					return true
				})
				if len(r.Publications) == 1 {
					if _, _, e := o.original(previous); e != nil {
						t.Fatal("pre-fault original", e)
					}
					r.Before, e = o.control(witnessNativeCommand{Operation: "readback"})
					if e != nil || len(r.Before.Originals) != 1 || r.Before.ConnectorStarted != 1 {
						t.Fatal("pre-fault actual census", e)
					}
				}
				r.Capacity, e = o.control(witnessNativeCommand{Operation: "readback"})
				captured := len(r.Capacity.Originals)
				if captured == 0 || captured > attempts || e != nil && captured < attempts {
					t.Fatal("actual capture census failed before boundary", captured, e)
				}
				last := r.Capacity.Originals[captured-1].Ref
				if captured == attempts && last.ID == previous && last.Complete && last.Delivered {
					break
				}
				// A source already active or delivered ahead of the observed publication
				// belongs to the original operation; join it instead of queuing work.
				if r.Capacity.Active || last.ID != previous {
					continue
				}
				// A single 409 joins an unchanged scheduler collection that won the race.
				// Count actual retained originals; request-ID gaps never invent attempts.
				r.RefreshStatuses = append(r.RefreshStatuses, witnessAdmissionRefresh(t, s, true))
			}
			if label == "control-failure" {
				var sent bytes.Buffer
				r.Boundary, e = o.controlRecorded(witnessNativeCommand{Operation: "untrusted"}, &sent)
				r.Control = append([]byte(nil), sent.Bytes()...)
				if e == nil || !r.Boundary.Failed || r.Boundary.StoreFailed || r.Boundary.Failure != "control-operation" {
					t.Fatal("actual independent control failure")
				}
			} else {
				// Attempt 33 may be the unchanged scheduler's next collection. It
				// consumes capacity just like a manual request and must stay failed.
				if r.Capacity.Failed && !r.Capacity.Active {
					r.Boundary = r.Capacity
				} else {
					r.BoundaryRefreshStatus = witnessAdmissionRefresh(t, s, true)
					performanceWait(t, func() bool {
						r.Boundary, _ = o.control(witnessNativeCommand{Operation: "readback"})
						return r.Boundary.Failed && !r.Boundary.Active
					})
				}
				if !r.Boundary.StoreFailed || r.Boundary.ConnectorStarted != 33 || len(r.Boundary.Originals) != 32 {
					t.Error("actual attempt 33 boundary", r.Boundary.ConnectorStarted, len(r.Boundary.Originals))
				}
			}

			// The broker can finish before the real manager has observed its EOF.
			// Join that original operation before making the next single request.
			performanceWait(t, func() bool {
				response, e := s.http.Get(s.base + "/status")
				if e != nil {
					return false
				}
				defer response.Body.Close()
				body, e := io.ReadAll(io.LimitReader(response.Body, 65537))
				return e == nil && response.StatusCode == 200 && !bytes.Contains(body, []byte("Collecting an observation"))
			})
			// A second real client collection after the latched failure may not create
			// another connector, original or publication. Readback stays available for evidence.
			r.FollowupRefreshStatus = witnessAdmissionRefresh(t, s, true)
			performanceWait(t, func() bool {
				r.Followup, _ = o.control(witnessNativeCommand{Operation: "readback"})
				return !r.Followup.Active && (r.Followup.Denied > r.Boundary.Denied || r.Followup.ConnectorStarted > r.Boundary.ConnectorStarted)
			})
			r.FollowupRejected = r.Followup.Denied > r.Boundary.Denied && r.Followup.ConnectorStarted == r.Boundary.ConnectorStarted && reflect.DeepEqual(r.Followup.Originals, r.Boundary.Originals)
			performanceWait(t, func() bool {
				response, e := s.http.Get(s.base + "/status")
				if e != nil {
					return false
				}
				defer response.Body.Close()
				body, e := io.ReadAll(io.LimitReader(response.Body, 65537))
				if e != nil || response.StatusCode != 200 {
					return false
				}
				if bytes.Contains(body, []byte("Collecting an observation")) {
					return false
				}
				r.LastCollectionFailed = bytes.Contains(body, []byte("Last collection:")) && bytes.Contains(body, []byte("Cached data is stale."))
				return true
			})
			if overview := s.overview(t); overview.Generation != previous || !overview.Complete {
				t.Error("failed collection replaced original publication")
			}
			_, _, e = o.original(r.Publications[0].ID)
			r.OldOriginalDenied = e != nil
			// Failed candidates still report their original terminal receipt and exit cleanly.
			r.Final, e = o.control(witnessNativeCommand{Operation: "stop"})
			if e == nil || !r.Final.Failed || !r.Final.FrontRemoved {
				t.Error("failed terminal receipt")
			}
			select {
			case e := <-o.process.done:
				o.process.waited = true
				if e != nil {
					t.Error("observer foreground exit", e)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("observer wait")
			}
			o.channel.Close()
			o.root.Close()
			o.closed = true
			o.final = r.Final
			r.Reaped = !witnessLifetime(r.Observer, probe.ABI) && !r.Final.Active && r.Final.ConnectorActive == 0 && r.Final.ConnectorStarted == r.Final.Connectors
			_, e = witnessRootSocket(o.config.Back, o.config.SocketGID)
			r.HelperUnchanged = e == nil && o.config.Helper.check(probe.ABI) == nil
			config, ce := os.ReadFile(p.pair.path)
			back, be := os.ReadFile(p.backPath)
			token, te := os.ReadFile(filepath.Join(p.pair.config.StateDirectory, "token.sha256"))
			r.ConfigurationUnchanged = ce == nil && be == nil && te == nil && bytes.Equal(config, p.configBytes) && bytes.Equal(back, p.backBytes) && bytes.Equal(token, p.digest)
			out := filepath.Join(export, label)
			if os.Mkdir(out, 0700) != nil {
				t.Fatal("exclusive case")
			}
			r.Originals, r.Indexes = witnessControllerExport(t, p, out)
			// Each original control receipt retains the unchanged 64 KiB ceiling.
			// The aggregate evidence index references separate immutable frames.
			r.Snapshots = map[string]string{}
			for name, snapshot := range map[string]witnessNativeReceipt{"Before": r.Before, "Capacity": r.Capacity, "Boundary": r.Boundary, "Followup": r.Followup, "Final": r.Final} {
				if witnessWritePrivateIndex(out, name+".json", snapshot) != nil {
					t.Fatal("bounded original snapshot", name)
				}
				f, e := os.Open(filepath.Join(out, name+".json"))
				if e != nil {
					t.Fatal(e)
				}
				b, e := io.ReadAll(io.LimitReader(f, witnessFrameCap+5))
				f.Close()
				if e != nil || len(b) < 5 || len(b) > witnessFrameCap+4 {
					t.Fatal("snapshot bound", name)
				}
				r.Snapshots[name] = fmt.Sprintf("%x", sha256.Sum256(b))
			}
			if witnessWritePrivateIndex(out, "receipt.json", r) != nil {
				t.Fatal("exclusive original receipt")
			}
			if !r.FollowupRejected || !r.LastCollectionFailed || !r.OldOriginalDenied || !r.Reaped || !r.HelperUnchanged || !r.ConfigurationUnchanged {
				t.Fatal("latched failure admitted work or lost custody", r.FollowupRejected, r.LastCollectionFailed, r.OldOriginalDenied, r.Reaped)
			}
			if label == "control-failure" {
				// The original actual helper is still available. Test artifact refusal
				// after the admission assertions, outside their collection boundary.
				artifactPair := *p.pair
				artifactPair.config = p.backConfig
				witnessNativeConnectorArtifactControl(t, &artifactPair, probe.ABI, o.config.Fixture)
			}
			p.pair.management.stop(t, syscall.SIGTERM, true)
			p.pair.helper.stop(t, syscall.SIGTERM, true)
			t.Logf("NATIVE_WITNESS_ADMISSION label=%s publications=%d rejected_after_failure=%t resource_acceptance=false", label, len(r.Publications), r.FollowupRejected)
		})
	}
}
