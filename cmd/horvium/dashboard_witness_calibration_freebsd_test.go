//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"encoding/json"
	"github.com/gonz0w/horvium/internal/observation"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

type witnessCalibrationGeneration struct {
	ID                    string `json:"generation"`
	ObservedNS            int64  `json:"observed_ns"`
	StartedAt, FinishedAt time.Time
}
type witnessCalibrationPath struct {
	Path                                                    string                         `json:"path"`
	WindowNS                                                int64                          `json:"window_ns"`
	Reads                                                   int                            `json:"metadata_reads"`
	Generations                                             []witnessCalibrationGeneration `json:"generations"`
	DurationsNS                                             []int64                        `json:"complete_response_ns"`
	Usage                                                   []witnessNativeUsageSample     `json:"usage"`
	ManagementHigh, HelperHigh, ObserverHigh, ConnectorHigh uint64
	ConnectorCPU                                            int64
	ResourceAcceptance                                      bool `json:"resource_acceptance"`
}

func witnessCalibrationWindow(t *testing.T, p *performancePair, path string) witnessCalibrationPath {
	t.Helper()
	r := witnessCalibrationPath{Path: path}
	started := time.Now()
	deadline := started.Add(65 * time.Second)
	seen := map[string]bool{}
	for time.Now().Before(deadline) {
		began := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		request, e := http.NewRequestWithContext(ctx, "GET", p.session.base+"/api/overview", nil)
		if e != nil {
			cancel()
			t.Fatal("calibration request")
		}
		response, e := p.session.http.Do(request)
		if e != nil {
			cancel()
			t.Fatal("calibration transport")
		}
		body, e := dashboardReadBody(response.Body)
		elapsed := time.Since(began)
		cancel()
		var overview observation.Overview
		metadata, me := witnessPublicationBytes(body)
		if e != nil || response.StatusCode != 200 || me != nil || !metadata.Complete || json.Unmarshal(body, &overview) != nil {
			t.Fatal("calibration publication metadata")
		}
		r.Reads++
		r.DurationsNS = append(r.DurationsNS, int64(elapsed))
		if !seen[overview.Generation] {
			seen[overview.Generation] = true
			if !dashboardGenerationPattern.MatchString(overview.Generation) {
				t.Fatal("calibration generation grammar")
			}
			r.Generations = append(r.Generations, witnessCalibrationGeneration{overview.Generation, int64(time.Since(started)), overview.StartedAt, overview.FinishedAt})
		}
		remaining := time.Until(deadline)
		pause := time.Second - time.Since(began)
		if pause > remaining {
			pause = remaining
		}
		if pause > 0 {
			time.Sleep(pause)
		}
	}
	r.WindowNS = int64(time.Since(started))
	if len(r.Generations) < 3 {
		t.Fatal("fewer than three observed calibration generations")
	}
	return r
}
func witnessDirectCalibration(t *testing.T, owner *dashboardGuestOwner, probe witnessProbeReceipt, count int) witnessCalibrationPath {
	t.Helper()
	c, path, token := nativeRuntimeConfig(t)
	c.IntervalSeconds, c.TimeoutSeconds = 30, 45
	b, _ := json.Marshal(c)
	if os.WriteFile(path, b, 0644) != nil {
		t.Fatal("direct original config")
	}
	owner.pair(t, "witness-direct-calibration", path, c)
	binary := filepath.Join(filepath.Dir(path), "horvium")
	built, _ := filepath.Abs("../../.tools/horvium")
	owner.copy(t, "management", built, binary)
	helper := startNativeProcess(t, binary, []string{"helper", "-config", path}, 0, 0)
	performanceWait(t, func() bool { _, e := witnessRootSocket(c.HelperSocket, c.SocketGID); return e == nil })
	management := startNativeProcess(t, binary, []string{"serve", "-config", path}, c.ManagementUID, c.SocketGID)
	p := &performancePair{config: c, path: path, helper: helper, management: management}
	p.session = newPerformanceSession(t, c, token)
	p.session.ready(t, count, "")
	usage := newWitnessNativeUsage(t, p, nil, probe.ABI)
	r := witnessCalibrationWindow(t, p, "direct")
	r.Usage = usage.finish(t)
	management.stop(t, syscall.SIGTERM, true)
	helper.stop(t, syscall.SIGTERM, true)
	r.ManagementHigh = peakProcessRSS(t, management)
	r.HelperHigh = peakProcessRSS(t, helper)
	if _, e := os.Lstat(c.HelperSocket); !os.IsNotExist(e) {
		t.Fatal("direct socket remains")
	}
	return r
}
func witnessWriteDiagnosticJSON(name string, value any) error {
	b, e := json.Marshal(value)
	if e != nil || len(b) > dashboardBodyLimit {
		return errDashboardWitness
	}
	f, e := os.OpenFile(filepath.Join("../../.tools", name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return errDashboardWitness
	}
	n, we := f.Write(b)
	se := f.Sync()
	me := f.Chmod(0400)
	ce := f.Close()
	if we != nil || se != nil || me != nil || ce != nil || n != len(b) {
		return errDashboardWitness
	}
	return nil
}
func witnessCalibrationPublish(t *testing.T, direct, broker witnessCalibrationPath) {
	t.Helper()
	if direct.Path != "direct" || broker.Path != "broker" || direct.WindowNS < 65e9 || broker.WindowNS < 65e9 || len(direct.Generations) < 3 || len(broker.Generations) < 3 {
		t.Fatal("calibration incomplete")
	}
	if witnessWriteDiagnosticJSON("witness-calibration.json", struct {
		Version            int
		Direct, Broker     witnessCalibrationPath
		CadenceAuthority   string
		NativeAcceptance   bool
		ResourceAcceptance bool
	}{1, direct, broker, "observed publication metadata; not exhaustive producer collection evidence", false, false}) != nil {
		t.Fatal("original calibration receipt")
	}
	t.Log("NATIVE_WITNESS_CALIBRATION direct_generations=" + strconv.Itoa(len(direct.Generations)) + " broker_generations=" + strconv.Itoa(len(broker.Generations)) + " window_seconds=65 native_acceptance=false resource_acceptance=false")
}
