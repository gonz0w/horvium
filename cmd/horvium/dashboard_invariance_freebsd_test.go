//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/host/freebsd"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
	"golang.org/x/sys/unix"
)

// Research only: fixed original expectations still reject changed observations.
// Scratch effects are limited to the byte-verified disposable guest work tree.
func TestNativeDashboardValueInvarianceDiagnosis(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("requires disposable pinned performance guest")
	}
	if os.Geteuid() != 0 {
		t.Fatal("diagnostic requires disposable guest root")
	}
	owner := newDashboardGuestOwner(t)
	var scratchDir string
	t.Cleanup(func() {
		clean := true
		if scratchDir == "" {
			clean = false
		} else if _, err := os.Lstat(scratchDir); !os.IsNotExist(err) {
			clean = false
		}
		for mode, dir := range owner.directories {
			if _, err := os.Lstat(dir); !os.IsNotExist(err) {
				clean = false
			}
			if _, err := os.Lstat(owner.sockets[mode]); !os.IsNotExist(err) {
				clean = false
			}
		}
		if len(owner.directories) != 1 || owner.storageCleanups != 1 || dashboardReaperEmpty() != nil || !reflect.DeepEqual(owner.artifacts, observeDashboardParentArtifacts(t)) {
			clean = false
		}
		if len(owner.files) != 1 {
			clean = false
		}
		for _, file := range owner.files {
			if file.check() != nil {
				clean = false
			}
		}
		t.Logf("DASHBOARD_INVARIANCE_CLEANUP verified=%t qualified=false native_acceptance=false", clean)
		if !clean {
			t.Error("invariance diagnostic cleanup incomplete")
		}
	})
	var err error
	scratchDir, err = os.MkdirTemp("../../.tools", "dashboard-invariance-")
	if err != nil {
		t.Fatal("owned scratch directory")
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(scratchDir); err != nil {
			t.Error("owned scratch removal failed")
		}
	})
	var fs unix.Statfs_t
	if unix.Statfs(scratchDir, &fs) != nil || string(bytes.TrimRight(fs.Fstypename[:], "\x00")) != "zfs" {
		t.Fatal("diagnostic work tree needs disposable ZFS backing")
	}
	scratchDatasetName := string(bytes.TrimRight(fs.Mntfromname[:], "\x00"))
	scratchPoolName := strings.Split(scratchDatasetName, "/")[0]
	expected := nativeStorageFixtureObserved(t, owner.storage)
	for i := 0; i < 98; i++ {
		readerFixtureCommand(t, "/sbin/zfs", "create", "-o", "mountpoint=none", expected.Pool+fmt.Sprintf("/item-%03d", i))
	}
	readerFixtureCommand(t, "/sbin/zpool", "sync", expected.Pool)
	ownedTSV := readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-p", "-t", "filesystem", "-o", "name,guid,used,available,referenced", "-r", expected.Pool)
	owned, err := inventory.ParseDatasets(ownedTSV, inventory.DefaultLimits())
	if err != nil || len(owned) != 100 {
		t.Fatal("independent 100-dataset witness")
	}
	count := len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
	if count > 500 {
		t.Fatal("diagnostic inventory exceeds bounded page")
	}
	c, path, token := nativeRuntimeConfig(t)
	c.IntervalSeconds, c.TimeoutSeconds = 30, 45
	encoded, _ := json.Marshal(c)
	if os.WriteFile(path, encoded, 0644) != nil {
		t.Fatal("diagnostic config")
	}
	owner.pair(t, "native-invariance", path, c)
	binary := filepath.Join(filepath.Dir(path), "horvium")
	built, _ := filepath.Abs("../../.tools/horvium")
	owner.copy(t, "management", built, binary)
	helper := startNativeProcess(t, binary, []string{"helper", "-config", path}, 0, 0)
	performanceWait(t, func() bool { info, e := os.Stat(c.HelperSocket); return e == nil && info.Mode().Perm() == 0660 })
	management := startNativeProcess(t, binary, []string{"serve", "-config", path}, c.ManagementUID, c.SocketGID)
	session := newPerformanceSession(t, c, token)
	session.ready(t, count, "")
	source, err := freebsd.New(c.InventoryLimits())
	if err != nil {
		t.Fatal("independent collector")
	}
	collect := func() inventory.Snapshot {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		s := source.Collect(ctx)
		if !s.Complete() {
			t.Fatal("independent snapshot incomplete")
		}
		return s
	}
	original := collect()
	// Repeat the independent witness at the declared original boundary.
	ownedTSV = readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-p", "-t", "filesystem", "-o", "name,guid,used,available,referenced", "-r", expected.Pool)
	prepared, err := dashboardPrepareNativeInputs("native-small", original, ownedTSV)
	if err != nil {
		t.Fatal("independent original preparation")
	}
	var scratchOriginal inventory.Dataset
	for _, d := range original.Datasets.Data {
		if d.Name == scratchDatasetName {
			scratchOriginal = d
		}
	}
	poolFound := false
	for _, p := range original.Pools.Data {
		if p.Name == scratchPoolName && p.GUID != expected.PoolGUID {
			poolFound = true
		}
	}
	if scratchOriginal.GUID == "" || !poolFound {
		t.Fatal("scratch backing is not independently identified other guest storage")
	}
	generation := session.overview(t).Generation
	scratchFile := filepath.Join(scratchDir, "probe.bin")
	data := make([]byte, 4<<20)
	var state uint64 = 0x123456789abcdef
	for i := range data {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		data[i] = byte(state)
	}
	digest := sha256.Sum256(data)
	replace := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		next, e := dashboardReplaceGeneration(ctx, session.http, session.base, session.csrf, count, generation)
		if e != nil {
			t.Fatal("diagnostic replacement failed")
		}
		generation = next
	}
	for _, stage := range []string{"baseline", "receipt-written", "scratch-written", "scratch-removed"} {
		switch stage {
		case "receipt-written":
			owner.original(t, prepared)
			readerFixtureCommand(t, "/sbin/zpool", "sync", scratchPoolName)
			replace()
		case "scratch-written":
			f, e := os.OpenFile(scratchFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				t.Fatal("owned scratch create")
			}
			n, we := f.Write(data)
			se := f.Sync()
			ce := f.Close()
			if we != nil || n != len(data) || se != nil || ce != nil {
				t.Fatal("bounded scratch write")
			}
			observed, e := os.ReadFile(scratchFile)
			if e != nil || len(observed) != len(data) || sha256.Sum256(observed) != digest {
				t.Fatal("scratch byte readback")
			}
			readerFixtureCommand(t, "/sbin/zpool", "sync", scratchPoolName)
			replace()
		case "scratch-removed":
			if os.Remove(scratchFile) != nil {
				t.Fatal("owned scratch remove")
			}
			readerFixtureCommand(t, "/sbin/zpool", "sync", scratchPoolName)
			replace()
		}
		fresh := collect()
		cached := fresh // The other sections are irrelevant to this diagnostic comparison.
		read := func(route string, value any) {
			response, e := session.http.Get(session.base + route + "?limit=500&generation=" + url.QueryEscape(generation))
			if e != nil {
				t.Fatal("diagnostic cache transport")
			}
			b, e := dashboardReadBody(response.Body)
			if e != nil || response.StatusCode != 200 || json.Unmarshal(b, value) != nil {
				t.Fatal("diagnostic cache page")
			}
		}
		var pools observation.Page[inventory.Pool]
		var datasets observation.Page[inventory.Dataset]
		read("/api/pools", &pools)
		read("/api/datasets", &datasets)
		if pools.Generation != generation || datasets.Generation != generation || pools.More || datasets.More || pools.Total != len(pools.Rows) || datasets.Total != len(datasets.Rows) {
			t.Fatal("incomplete same-generation diagnostic cache pages")
		}
		cached.Pools.Data, cached.Datasets.Data = pools.Rows, datasets.Rows
		nativeChanges, e := dashboardValueChanges(original, fresh, expected.PoolGUID, owned)
		if e != nil {
			t.Fatal("independent native identities changed")
		}
		cacheChanges, e := dashboardValueChanges(original, cached, expected.PoolGUID, owned)
		if e != nil {
			t.Fatal("original cache identities changed")
		}
		nativeCache, e := dashboardValueChanges(fresh, cached, expected.PoolGUID, owned)
		if e != nil {
			t.Fatal("native/cache identities changed")
		}
		var target inventory.Dataset
		for _, d := range fresh.Datasets.Data {
			if d.GUID == scratchOriginal.GUID {
				target = d
			}
		}
		if target.GUID == "" {
			t.Fatal("original scratch dataset disappeared")
		}
		var samples []dashboardWireSample
		for _, request := range prepared.plan[:50] {
			start := time.Now()
			response, transportErr := session.http.Get(session.base + request.Path)
			var b []byte
			var bodyErr error
			status := 0
			if transportErr == nil {
				status = response.StatusCode
				b, bodyErr = dashboardReadBody(response.Body)
			}
			elapsed := time.Since(start)
			outcome, label := dashboardClassifyResponse(request, "initial", status, b, transportErr, bodyErr, prepared.predicate)
			samples = append(samples, dashboardRawWire([]dashboardSample{{Index: request.Index, Case: request.Case, Outcome: outcome, Predicate: label, Status: status, BodyBytes: len(b), Duration: elapsed}})...)
		}
		record := struct {
			Stage                                                string `json:"stage"`
			Qualified                                            bool   `json:"qualified"`
			NativeAcceptance                                     bool   `json:"native_acceptance"`
			OriginalSHA256                                       string `json:"original_sha256"`
			Pools, Datasets, OwnedDatasets, DeclaredScratchBytes int
			NativeChanges, CacheChanges, NativeCacheDifferences  map[string]map[string]int
			ScratchUsedGreater, ScratchReferencedGreater         bool
			HTML                                                 []dashboardWireSample
		}{stage, false, false, prepared.fixture.InventorySHA256, len(original.Pools.Data), len(original.Datasets.Data), len(owned), len(data), nativeChanges, cacheChanges, nativeCache, target.Used > scratchOriginal.Used, target.Referenced > scratchOriginal.Referenced, samples}
		b, e := json.Marshal(record)
		if e != nil {
			t.Fatal("diagnostic encoding")
		}
		t.Log("DASHBOARD_INVARIANCE " + string(b))
		if stage == "scratch-written" && (!record.ScratchUsedGreater || !record.ScratchReferencedGreater) {
			t.Fatal("controlled write did not independently change identified scratch dataset")
		}
	}
	management.stop(t, syscall.SIGTERM, true)
	helper.stop(t, syscall.SIGTERM, true)
}
