//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/host/freebsd"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

// Research probe only. No resource receipts, warm-up/idle budget claims,
// replacement/crash controls or acceptance markers are emitted by this test.
func TestNativeDashboardFocusedDiagnosis(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("requires explicitly disposable pinned performance guest")
	}
	if os.Geteuid() != 0 {
		t.Fatal("diagnostic fixture requires disposable guest root")
	}
	owner := newDashboardGuestOwner(t)
	t.Cleanup(func() {
		clean := true
		for mode, dir := range owner.directories {
			if _, err := os.Lstat(dir); !os.IsNotExist(err) {
				clean = false
				t.Error("diagnostic directory cleanup uncertain")
			}
			if _, err := os.Lstat(owner.sockets[mode]); !os.IsNotExist(err) {
				clean = false
				t.Error("diagnostic socket cleanup uncertain")
			}
		}
		if owner.storageCleanups != 1 || len(owner.directories) != 3 || dashboardReaperEmpty() != nil || !reflect.DeepEqual(owner.artifacts, observeDashboardParentArtifacts(t)) {
			clean = false
			t.Error("diagnostic storage/descendant/artifact observations incomplete")
		}
		t.Logf("DASHBOARD_DIAGNOSTIC_CLEANUP verified=%t qualified=false native_acceptance=false", clean)
	})
	for _, mode := range []string{"native-reduced", "projection-10000", "synthetic-10000"} {
		t.Run(mode, func(t *testing.T) {
			var pair *performancePair
			var original inventory.Snapshot
			if mode == "synthetic-10000" {
				pair = newPerformancePair(t, 10000, false, owner)
				owner.pair(t, mode, pair.path, pair.config)
				original = performanceSnapshot(10000, 0)
			} else {
				if mode == "native-reduced" {
					nativeStorageFixtureObserved(t, owner.storage) // Two owned filesystems, not the qualifying 100.
				}
				c, path, token := nativeRuntimeConfig(t)
				c.IntervalSeconds, c.TimeoutSeconds = 30, 45
				encoded, _ := json.Marshal(c)
				if os.WriteFile(path, encoded, 0644) != nil {
					t.Fatal("diagnostic config")
				}
				owner.pair(t, mode, path, c)
				dir := filepath.Dir(path)
				built, _ := filepath.Abs("../../.tools/horvium")
				binary := filepath.Join(dir, "horvium")
				owner.copy(t, "management", built, binary)
				var helper *nativeProcess
				if mode == "native-reduced" {
					helper = startNativeProcess(t, binary, []string{"helper", "-config", path}, 0, 0)
				} else {
					fixture, _ := filepath.Abs("../../.tools/reader-fixture.test")
					copied := filepath.Join(dir, "reader-fixture.test")
					owner.copy(t, "helper", fixture, copied)
					cmd := exec.Command(copied, "-test.run=^TestNativeProjectionHelper$", "--", path)
					cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
					helper = &nativeProcess{cmd: cmd, done: make(chan error, 1)}
					if cmd.Start() != nil {
						t.Fatal("diagnostic projection start")
					}
					go func() { helper.done <- cmd.Wait() }()
					t.Cleanup(func() {
						if !helper.waited {
							helper.stop(t, syscall.SIGTERM, true)
						}
					})
				}
				performanceWait(t, func() bool { info, e := os.Stat(c.HelperSocket); return e == nil && info.Mode().Perm() == 0660 })
				management := startNativeProcess(t, binary, []string{"serve", "-config", path}, c.ManagementUID, c.SocketGID)
				pair = &performancePair{config: c, path: path, helper: helper, management: management}
				pair.session = newPerformanceSession(t, c, token)
				count := 10000
				if mode == "native-reduced" {
					count = len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
				}
				// Existing readiness tolerates initial 503 and requires a complete,
				// fresh publication. The strict overview reader is only safe after it.
				pair.session.ready(t, count, "")
				source, err := freebsd.New(c.InventoryLimits())
				if err != nil {
					t.Fatal("diagnostic independent collector")
				}
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				original = source.Collect(ctx)
				cancel()
				if mode == "projection-10000" {
					// Independently declared emitter inputs, never cached response values.
					original.Datasets.Data = make([]inventory.Dataset, 10000)
					for i := range original.Datasets.Data {
						original.Datasets.Data[i] = inventory.Dataset{Name: fmt.Sprintf("fixture/dataset-%06d", i), GUID: fmt.Sprint(i + 100), Used: uint64(i), Available: 1 << 40, Referenced: uint64(i)}
					}
				}
			}
			if !original.Complete() || len(original.Pools.Data) > 500 {
				t.Fatal("diagnostic original incomplete or exceeds single pool page")
			}
			plan, err := dashboardPlan(len(original.Datasets.Data))
			if err != nil {
				t.Fatal(err)
			}
			predicate, err := newDashboardHTMLPredicate(original, plan)
			if err != nil {
				t.Fatal("diagnostic predicate preparation")
			}
			generation := pair.session.overview(t).Generation
			response, err := pair.session.http.Get(pair.session.base + "/api/pools?limit=500&generation=" + url.QueryEscape(generation))
			if err != nil {
				t.Fatal("diagnostic cached pool query")
			}
			body, err := dashboardReadBody(response.Body)
			var page observation.Page[inventory.Pool]
			if err != nil || response.StatusCode != http.StatusOK || json.Unmarshal(body, &page) != nil || page.Generation != generation || page.More || page.Total != len(page.Rows) {
				t.Fatal("diagnostic cached pool page incomplete")
			}
			// Counts only: never emit names, GUIDs, response bodies or credentials.
			differences := map[string]int{}
			cached := map[string]inventory.Pool{}
			for _, row := range page.Rows {
				cached[row.GUID] = row
			}
			for _, row := range original.Pools.Data {
				other, exists := cached[row.GUID]
				if !exists {
					differences["missing_identity"]++
					continue
				}
				if row.Name != other.Name {
					differences["name"]++
				}
				if row.Size != other.Size {
					differences["size"]++
				}
				if row.Allocated != other.Allocated {
					differences["allocated"]++
				}
				if row.Free != other.Free {
					differences["free"]++
				}
				if row.Health != other.Health {
					differences["health"]++
				}
			}
			if len(cached) != len(page.Rows) || len(original.Pools.Data) != len(page.Rows) {
				differences["cardinality"]++
			}
			var samples []dashboardWireSample
			for i := 0; i < 50; i++ {
				request := plan[i]
				started := time.Now()
				r, transportErr := pair.session.http.Get(pair.session.base + request.Path)
				var data []byte
				var bodyErr error
				status := 0
				if transportErr == nil {
					status = r.StatusCode
					data, bodyErr = dashboardReadBody(r.Body)
				}
				elapsed := time.Since(started) // Complete body, excludes predicate parsing.
				outcome, label := dashboardClassifyResponse(request, "initial", status, data, transportErr, bodyErr, predicate)
				samples = append(samples, dashboardRawWire([]dashboardSample{{Index: i, Case: request.Case, Outcome: outcome, Status: status, BodyBytes: len(data), Predicate: label, Duration: elapsed}})...)
			}
			var api []dashboardWireSample
			for i, path := range []string{"/api/overview", "/api/pools", "/api/pools?limit=500", "/api/datasets", "/api/datasets?limit=500"} {
				for j := 0; j < 10; j++ {
					started := time.Now()
					r, err := pair.session.http.Get(pair.session.base + path)
					if err != nil {
						t.Fatal("diagnostic API transport")
					}
					data, err := dashboardReadBody(r.Body)
					elapsed := time.Since(started)
					if err != nil || r.StatusCode != 200 {
						t.Fatal("diagnostic API body/status")
					}
					api = append(api, dashboardRawWire([]dashboardSample{{Index: i*10 + j, Case: dashboardCases[i], Outcome: "success", Status: 200, BodyBytes: len(data), Predicate: "status-body-only", Duration: elapsed}})...)
				}
			}
			encoded, _ := json.Marshal(original)
			record := struct {
				Mode             string `json:"mode"`
				Qualified        bool   `json:"qualified"`
				NativeAcceptance bool   `json:"native_acceptance"`
				OriginalSHA256   string `json:"original_sha256"`
				Pools, Datasets  int
				PoolDifferences  map[string]int `json:"pool_differences"`
				HTML, API        []dashboardWireSample
			}{mode, false, false, fmt.Sprintf("%x", sha256.Sum256(encoded)), len(original.Pools.Data), len(original.Datasets.Data), differences, samples, api}
			data, err := json.Marshal(record)
			if err != nil {
				t.Fatal("diagnostic encoding")
			}
			t.Log("DASHBOARD_DIAGNOSTIC " + string(data))
			pair.management.stop(t, syscall.SIGTERM, true)
			pair.helper.stop(t, syscall.SIGTERM, true)
		})
	}
}
