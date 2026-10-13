//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/helper"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

// The cap fixture can serialize 100,000 records; keep its wait above the
// configured 45-second collection deadline, without unbounded retries.
func performanceWait(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("performance fixture condition timed out")
}

type performanceControl struct{ Datasets, Padding int }
type controlledPerformanceSource struct{ path string }

func (s controlledPerformanceSource) Collect(ctx context.Context) inventory.Snapshot {
	b, e := os.ReadFile(s.path)
	var c performanceControl
	if e != nil || json.Unmarshal(b, &c) != nil || c.Datasets < 0 || c.Datasets > 100000 || c.Padding < 0 || c.Padding > 4000 {
		return inventory.Snapshot{}
	}
	return performanceSnapshot(c.Datasets, c.Padding)
}
func TestNativePerformanceHelper(t *testing.T) {
	path := os.Getenv("HORVIUM_PERF_HELPER")
	if path == "" {
		t.Skip("reexec performance fixture only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" || os.Getuid() != 0 {
		t.Fatal("performance fixture authorization")
	}
	c, e := admin.LoadConfig(path)
	if e != nil {
		t.Fatal("fixture config")
	}
	server, e := helper.NewServer(c.HelperConfig(), controlledPerformanceSource{filepath.Join(filepath.Dir(path), "control.json")})
	if e != nil {
		t.Fatal("fixture helper")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if e = server.Serve(ctx); e != nil && !errors.Is(e, context.Canceled) {
		t.Fatal("fixture helper failed")
	}
}

type usageMonitor struct {
	pids     [2]int
	mu       sync.Mutex
	max      uint64
	count    int
	maxGap   time.Duration
	previous time.Time
	err      error
	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
}

func sampleUsage(pids [2]int) (processUsage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/ps", "-p", strconv.Itoa(pids[0])+","+strconv.Itoa(pids[1]), "-o", "pid=", "-o", "rss=", "-o", "time=")
	cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	b, e := cmd.Output()
	if e != nil {
		return processUsage{}, fmt.Errorf("fixture process sample unavailable")
	}
	return parseUsage(b, pids)
}
func sampleDashboardCPU(pids [2]int) (dashboardCPUCounters, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/ps", "-p", strconv.Itoa(pids[0])+","+strconv.Itoa(pids[1]), "-o", "pid=", "-o", "usertime=", "-o", "systime=")
	cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	data, err := cmd.Output()
	if err != nil {
		return dashboardCPUCounters{}, errors.New("paired CPU observation unavailable")
	}
	return dashboardParseCPU(data, pids)
}
func observeDashboardIdle(t *testing.T, pids [2]int, label string) dashboardIdleObservation {
	t.Helper()
	started := time.Now() // envelope includes the two bounded ps observations
	before, err := sampleDashboardCPU(pids)
	if err != nil {
		t.Fatal(err)
	}
	performancePause(t, 300*time.Second, label)
	after, err := sampleDashboardCPU(pids)
	if err != nil {
		t.Fatal(err)
	}
	observation, err := dashboardObserveIdle(before, after, time.Since(started))
	if err != nil {
		t.Fatal(err)
	}
	return observation
}

func newUsageMonitor(pids [2]int) *usageMonitor {
	m := &usageMonitor{pids: pids, stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(m.done)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-m.stop:
				return
			case <-ticker.C:
				s, e := sampleUsage(pids)
				now := time.Now()
				m.mu.Lock()
				if e != nil {
					m.err = e
				} else {
					m.count++
					m.max = max(m.max, s.RSSBytes)
					if !m.previous.IsZero() {
						m.maxGap = max(m.maxGap, now.Sub(m.previous))
					}
					m.previous = now
				}
				m.mu.Unlock()
			}
		}
	}()
	return m
}
func (m *usageMonitor) close() {
	m.stopOnce.Do(func() { close(m.stop) })
	<-m.done
}
func (m *usageMonitor) finish(t *testing.T) (uint64, int, time.Duration) {
	t.Helper()
	m.close()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil || m.count < 1 {
		t.Fatal("incomplete RSS sampling")
	}
	return m.max, m.count, m.maxGap
}

type performanceSession struct {
	http       *http.Client
	base, csrf string
}

func newPerformanceSession(t *testing.T, c admin.Config, token string) *performanceSession {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	transport := &http.Transport{Proxy: nil, MaxConnsPerHost: 10, MaxIdleConnsPerHost: 10, ResponseHeaderTimeout: 5 * time.Second, MaxResponseHeaderBytes: 16 << 10}
	client := &http.Client{Transport: transport, Jar: jar, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	t.Cleanup(transport.CloseIdleConnections)
	s := &performanceSession{http: client, base: "http://" + c.Listen}
	performanceWait(t, func() bool {
		r, e := client.Get(s.base + "/auth/challenge")
		if e != nil {
			return false
		}
		r.Body.Close()
		return r.StatusCode == 200
	})
	r, e := client.Get(s.base + "/auth/challenge")
	if e != nil {
		t.Fatal("fixture challenge")
	}
	var challenge admin.Challenge
	e = json.NewDecoder(r.Body).Decode(&challenge)
	r.Body.Close()
	if e != nil {
		t.Fatal("fixture challenge schema")
	}
	r, e = client.PostForm(s.base+"/auth/login", url.Values{"csrf": {challenge.CSRF}, "token": {token}})
	if e != nil {
		t.Fatal("fixture login")
	}
	var session admin.Session
	e = json.NewDecoder(r.Body).Decode(&session)
	r.Body.Close()
	if e != nil || r.StatusCode != 200 {
		t.Fatal("fixture login denied")
	}
	s.csrf = session.CSRF
	return s
}
func (s *performanceSession) overview(t *testing.T) observation.Overview {
	t.Helper()
	r, e := s.http.Get(s.base + "/api/overview")
	if e != nil {
		t.Fatal("fixture overview unreachable")
	}
	defer r.Body.Close()
	var o observation.Overview
	if r.StatusCode != 200 || json.NewDecoder(io.LimitReader(r.Body, 65536)).Decode(&o) != nil {
		t.Fatal("fixture overview unavailable")
	}
	return o
}
func (s *performanceSession) refresh(t *testing.T) {
	t.Helper()
	performanceWait(t, func() bool {
		r, e := s.http.PostForm(s.base+"/api/refresh", url.Values{"csrf": {s.csrf}})
		if e != nil {
			t.Fatal("fixture refresh unreachable")
		}
		r.Body.Close()
		if r.StatusCode == 409 {
			return false
		}
		if r.StatusCode != 202 {
			t.Fatal("fixture refresh denied")
		}
		return true
	})
}
func (s *performanceSession) ready(t *testing.T, count int, previous string) string {
	t.Helper()
	var generation string
	performanceWait(t, func() bool {
		r, e := s.http.Get(s.base + "/api/overview")
		if e != nil {
			return false
		}
		defer r.Body.Close()
		if r.StatusCode == 503 {
			return false
		}
		var o observation.Overview
		if r.StatusCode != 200 || json.NewDecoder(io.LimitReader(r.Body, 65536)).Decode(&o) != nil {
			t.Fatal("fixture overview schema")
		}
		generation = o.Generation
		return generation != "" && generation != previous && o.Complete && o.DatasetCount == count && !o.Datasets.Stale
	})
	return generation
}

// Publish sanitized partial fixture observations even on a failed phase. These
// cannot qualify the full resource contract or replace independently bound host records.
func measureDashboardFixture(t *testing.T, p *performancePair, prepared *dashboardPreparedFixture) dashboardMeasuredWorkload {
	t.Helper()
	previous := p.session.overview(t).Generation // controller read before timed phases
	result, err := dashboardMeasureWorkload(func(phase string) (*dashboardPhaseReceipt, []dashboardSample, error) {
		return prepared.load(p.session.http, p.session.base, phase)
	}, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		generation, err := dashboardReplaceGeneration(ctx, p.session.http, p.session.base, p.session.csrf, prepared.count, previous)
		if err == nil {
			previous = generation
		}
		return err
	})
	if err != nil {
		record := struct {
			Mode           string                    `json:"mode"`
			Fixture        dashboardFixtureReceipt   `json:"fixture"`
			Phases         dashboardMeasuredWorkload `json:"phases"`
			InitialRaw     []dashboardWireSample     `json:"initial_raw"`
			ReplacementRaw []dashboardWireSample     `json:"replacement_raw"`
			WorkloadPassed bool                      `json:"workload_passed"`
		}{prepared.mode, prepared.fixture, result, dashboardRawWire(result.InitialRaw), dashboardRawWire(result.ReplacementRaw), err == nil}
		encoded, e := json.Marshal(record)
		if e != nil {
			t.Fatal("dashboard workload observation encoding")
		}
		t.Log("DASHBOARD_WORKLOAD " + string(encoded))
		t.Fatal(err)
	}
	return result
}

func performancePause(t *testing.T, d time.Duration, label string) time.Duration {
	t.Helper()
	started := time.Now()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return time.Since(started)
		case <-ticker.C:
			t.Logf("performance phase=%s elapsed_seconds=%.0f", label, time.Since(started).Seconds())
		}
	}
}
func copyPerformanceExecutable(t *testing.T, from, to string) {
	t.Helper()
	input, e := os.Open(from)
	if e != nil {
		t.Fatal("fixture executable")
	}
	defer input.Close()
	output, e := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if e != nil {
		t.Fatal("fixture executable copy")
	}
	_, e = io.Copy(output, input)
	ce := output.Close()
	if e != nil || ce != nil {
		t.Fatal("fixture executable copy failed")
	}
}
func peakProcessRSS(t *testing.T, p *nativeProcess) uint64 {
	t.Helper()
	usage, ok := p.cmd.ProcessState.SysUsage().(*syscall.Rusage)
	if !ok || usage.Maxrss <= 0 {
		t.Fatal("process high-water RSS unavailable")
	}
	return uint64(usage.Maxrss) * 1024
}

type performancePair struct {
	config             admin.Config
	path, control      string
	helper, management *nativeProcess
	session            *performanceSession
	monitor            *usageMonitor
}

func newPerformancePair(t *testing.T, count int, modelBoundary bool) *performancePair {
	t.Helper()
	c, path, token := nativeRuntimeConfig(t)
	c.IntervalSeconds = 30
	c.TimeoutSeconds = 45
	c.MaxTransportRecords = 200000
	if modelBoundary {
		// Keep wire bytes above the model budget to isolate model rejection.
		c.MaxJSONBytes = 128 << 20
	}
	b, _ := json.Marshal(c)
	if os.WriteFile(path, b, 0644) != nil {
		t.Fatal("fixture config rewrite")
	}
	dir := filepath.Dir(path)
	control := filepath.Join(dir, "control.json")
	writePerformanceControl(t, control, performanceControl{Datasets: count})
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	helperBinary := filepath.Join(dir, "helper-test")
	copyPerformanceExecutable(t, exe, helperBinary)
	built, e := filepath.Abs("../../.tools/horvium")
	if e != nil {
		t.Fatal(e)
	}
	binary := filepath.Join(dir, "horvium")
	copyPerformanceExecutable(t, built, binary)
	cmd := exec.Command(helperBinary, "-test.run=^TestNativePerformanceHelper$", "-test.count=1", "-test.timeout=20m")
	cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "HORVIUM_PERF_HELPER=" + path, "LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	helperProcess := &nativeProcess{cmd: cmd, done: make(chan error, 1)}
	if cmd.Start() != nil {
		t.Fatal("fixture helper process")
	}
	go func() { helperProcess.done <- cmd.Wait() }()
	t.Cleanup(func() {
		if !helperProcess.waited {
			helperProcess.stop(t, syscall.SIGTERM, true)
		}
	})
	performanceWait(t, func() bool { info, e := os.Stat(c.HelperSocket); return e == nil && info.Mode().Perm() == 0660 })
	management := startNativeProcess(t, binary, []string{"serve", "-config", path}, c.ManagementUID, c.SocketGID)
	monitor := newUsageMonitor([2]int{helperProcess.cmd.Process.Pid, management.cmd.Process.Pid})
	t.Cleanup(monitor.close)
	p := &performancePair{c, path, control, helperProcess, management, nil, monitor}
	p.session = newPerformanceSession(t, c, token)
	p.session.ready(t, count, "")
	return p
}
func writePerformanceControl(t *testing.T, path string, c performanceControl) {
	t.Helper()
	b, _ := json.Marshal(c)
	temporary := path + ".next"
	if os.WriteFile(temporary, b, 0600) != nil || os.Rename(temporary, path) != nil {
		t.Fatal("fixture control publication")
	}
}
func (p *performancePair) stopResources(t *testing.T) dashboardResourceMeasurements {
	t.Helper()
	sampled, samples, gap := p.monitor.finish(t)
	p.management.stop(t, syscall.SIGTERM, true)
	p.helper.stop(t, syscall.SIGTERM, true)
	managementHigh, helperHigh := peakProcessRSS(t, p.management), peakProcessRSS(t, p.helper)
	if _, err := os.Lstat(p.config.HelperSocket); !os.IsNotExist(err) {
		t.Fatal("performance helper socket leaked")
	}
	return dashboardResourceMeasurements{SampledRSS: sampled, ManagementHigh: managementHigh, HelperHigh: helperHigh, Samples: samples, MaxGap: gap}
}
func (p *performancePair) stop(t *testing.T) (uint64, uint64, int, time.Duration) {
	t.Helper()
	measured := p.stopResources(t)
	return measured.SampledRSS, measured.ManagementHigh + measured.HelperHigh, measured.Samples, measured.MaxGap
}
func publishDashboardResource(t *testing.T, prepared *dashboardPreparedFixture, work dashboardMeasuredWorkload, measured dashboardResourceMeasurements, native bool) {
	t.Helper()
	record, err := dashboardBuildResource(prepared, work, measured)
	if err != nil {
		failed := struct {
			Mode         string                        `json:"mode"`
			Fixture      dashboardFixtureReceipt       `json:"fixture"`
			Measurements dashboardResourceMeasurements `json:"measurements"`
			Phases       dashboardMeasuredWorkload     `json:"phases"`
		}{prepared.mode, prepared.fixture, measured, work}
		if data, e := json.Marshal(failed); e == nil {
			t.Log("DASHBOARD_RESOURCE_FAILURE " + string(data))
		}
		t.Fatal(err)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal("dashboard resource observation encoding")
	}
	t.Log("DASHBOARD_RESOURCE " + string(encoded))
	legacy, err := json.Marshal(dashboardLegacyResource(record, native))
	if err != nil {
		t.Fatal("legacy resource observation encoding")
	}
	marker := "PERFORMANCE_RESULT "
	if native {
		marker = "NATIVE_READER_RESULT "
	}
	t.Log(marker + string(legacy))
}
func nativeDashboardControlLedger(t *testing.T) *dashboardControlLedger {
	t.Helper()
	ledger, err := newDashboardControlLedger(t.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if t.Failed() || t.Skipped() {
			return
		}
		record, err := ledger.finish()
		if err != nil {
			t.Error("dashboard parent controls incomplete")
			return
		}
		data, err := json.Marshal(record)
		if err != nil {
			t.Error("dashboard parent control encoding")
			return
		}
		t.Log("NATIVE_DASHBOARD_CONTROLS " + string(data))
	})
	return ledger
}
func TestNativePerformanceBudgets(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("explicit disposable performance VM opt-in required")
	}
	if os.Getuid() != 0 {
		t.Fatal("performance fixture requires root")
	}
	ledger := nativeDashboardControlLedger(t)
	for _, count := range []int{100, 10000} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			p := newPerformancePair(t, count, false)
			warmup := performancePause(t, 60*time.Second, "warmup-"+strconv.Itoa(count))
			var idle dashboardIdleObservation
			if count == 100 {
				idle = observeDashboardIdle(t, p.monitor.pids, "idle")
			}
			prepared, e := newDashboardPreparedFixture("synthetic-"+strconv.Itoa(count), performanceSnapshot(count, 0), nil)
			if e != nil {
				t.Fatal(e)
			}
			if err := ledger.declare(prepared); err != nil {
				t.Fatal(err)
			}
			work := measureDashboardFixture(t, p, prepared)
			measured := p.stopResources(t)
			measured.Warmup, measured.Idle = warmup, idle
			if err := ledger.completeMode(prepared, work, measured); err != nil {
				t.Fatal(err)
			}
			publishDashboardResource(t, prepared, work, measured, false)
		})
	}
	t.Run("limits", func(t *testing.T) {
		boundary := new(dashboardBoundaryControl)
		p := newPerformancePair(t, 100, true)
		previous := p.session.overview(t).Generation
		writePerformanceControl(t, p.control, performanceControl{Datasets: 99990})
		p.session.refresh(t)
		previous = p.session.ready(t, 99990, previous)
		if err := boundary.step("accepted-model", dashboardDeclaredModelRecords(99990, 0)); err != nil {
			t.Fatal(err)
		}
		// Model records: 99,990 datasets + class/four geoms/four providers/pool = 100,000.
		for i, control := range []performanceControl{{Datasets: 99991}, {Datasets: 20000, Padding: 4000}} {
			writePerformanceControl(t, p.control, control)
			p.session.refresh(t)
			performanceWait(t, func() bool {
				r, e := p.session.http.Get(p.session.base + "/status")
				if e != nil {
					t.Fatal("limit status unreachable")
				}
				b, e := io.ReadAll(io.LimitReader(r.Body, 65536))
				r.Body.Close()
				if e != nil || r.StatusCode != 200 {
					t.Fatal("limit status unavailable")
				}
				// Refresh marks collecting before returning 202. Wait for this
				// attempt to finish, rather than matching an earlier stale flag.
				if bytes.Contains(b, []byte("Collecting")) {
					return false
				}
				if !bytes.Contains(b, []byte("limit")) {
					return false
				}
				o := p.session.overview(t)
				return o.Generation == previous && o.Datasets.Stale
			})
			event := []string{"record-exhaustion", "byte-exhaustion"}[i]
			if err := boundary.step(event, dashboardDeclaredModelRecords(control.Datasets, control.Padding)); err != nil {
				t.Fatal(err)
			}
		}
		writePerformanceControl(t, p.control, performanceControl{Datasets: 100})
		p.session.refresh(t)
		p.session.ready(t, 100, previous)
		if err := boundary.step("recovery", dashboardDeclaredModelRecords(100, 0)); err != nil {
			t.Fatal(err)
		}
		measured := p.stopResources(t)
		observed, err := boundary.finish(measured)
		if err != nil {
			t.Fatal(err)
		}
		if err := ledger.boundary(observed); err != nil {
			t.Fatal(err)
		}
		sampled, peak, samples, gap := measured.SampledRSS, measured.ManagementHigh+measured.HelperHigh, measured.Samples, measured.MaxGap
		t.Logf("PERFORMANCE_LIMITS accepted_model_records=100000 record_exhaustion=pass byte_exhaustion=pass recovery=pass sampled_rss_bytes=%d high_water_sum_bytes=%d samples=%d max_gap_ms=%.1f", sampled, peak, samples, float64(gap.Microseconds())/1000)
	})
}
