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
	"slices"
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
func (s *performanceSession) load(t *testing.T, count int) ([]time.Duration, error) {
	t.Helper()
	latencies := make([]time.Duration, 1000)
	failures := make(chan error, 10)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for worker := 0; worker < 10; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := worker * 100; i < (worker+1)*100; i++ {
				offset := (i * 100) % count
				started := time.Now()
				r, e := s.http.Get(s.base + fmt.Sprintf("/storage/datasets?sort=name&offset=%d&limit=100", offset))
				if e != nil {
					failures <- fmt.Errorf("fixture page transport failure")
					return
				}
				b, e := io.ReadAll(io.LimitReader(r.Body, 1<<20))
				r.Body.Close()
				latencies[i] = time.Since(started)
				expected := fmt.Sprintf("vault/dataset-%06d", offset)
				if e != nil || r.StatusCode != 200 || !bytes.Contains(b, []byte(expected)) {
					failures <- fmt.Errorf("fixture page contract failure")
					return
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	close(failures)
	for e := range failures {
		return nil, e
	}
	return latencies, nil
}
func performancePause(t *testing.T, d time.Duration, label string) {
	t.Helper()
	started := time.Now()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return
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
func (p *performancePair) stop(t *testing.T) (uint64, uint64, int, time.Duration) {
	t.Helper()
	sampled, samples, gap := p.monitor.finish(t)
	p.management.stop(t, syscall.SIGTERM, true)
	p.helper.stop(t, syscall.SIGTERM, true)
	peak := peakProcessRSS(t, p.management) + peakProcessRSS(t, p.helper)
	if _, e := os.Lstat(p.config.HelperSocket); !os.IsNotExist(e) {
		t.Fatal("performance helper socket leaked")
	}
	return sampled, peak, samples, gap
}
func TestNativePerformanceBudgets(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("explicit disposable performance VM opt-in required")
	}
	if os.Getuid() != 0 {
		t.Fatal("performance fixture requires root")
	}
	for _, count := range []int{100, 10000} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			p := newPerformancePair(t, count, false)
			performancePause(t, 60*time.Second, "warmup-"+strconv.Itoa(count))
			idle := float64(0)
			if count == 100 {
				before, e := sampleUsage(p.monitor.pids)
				if e != nil {
					t.Fatal(e)
				}
				started := time.Now()
				performancePause(t, 300*time.Second, "idle")
				after, e := sampleUsage(p.monitor.pids)
				if e != nil {
					t.Fatal(e)
				}
				idle, e = idlePercent(before.CPU, after.CPU, time.Since(started))
				if e != nil {
					t.Fatal(e)
				}
			}
			latencies, e := p.session.load(t, count)
			if e != nil {
				t.Fatal(e)
			}
			// Replace generations while the same ten-reader workload is active.
			done := make(chan error, 1)
			go func() { _, e := p.session.load(t, count); done <- e }()
			previous := p.session.overview(t).Generation
			for iteration := 0; iteration < 3; iteration++ {
				p.session.refresh(t)
				previous = p.session.ready(t, count, previous)
			}
			if e := <-done; e != nil {
				t.Fatal(e)
			}
			sampled, peak, samples, gap := p.stop(t)
			slices.Sort(latencies)
			p50, p95, p99 := latencies[499], latencies[949], latencies[989]
			record := map[string]any{"datasets": count, "clients": 10, "session_count": 1, "requests": 1000, "replacement_requests": 1000, "replacements": 3, "warmup_seconds": 60, "idle_seconds": 0, "idle_cpu_percent_one_core": idle, "sampled_combined_peak_rss_bytes": sampled, "conservative_combined_high_water_rss_bytes": peak, "rss_samples": samples, "max_sample_gap_ms": float64(gap.Microseconds()) / 1000, "p50_ms": float64(p50.Microseconds()) / 1000, "p95_ms": float64(p95.Microseconds()) / 1000, "p99_ms": float64(p99.Microseconds()) / 1000}
			if count == 100 {
				record["idle_seconds"] = 300
			}
			encoded, _ := json.Marshal(record)
			t.Log("PERFORMANCE_RESULT " + string(encoded))
			budget := uint64(256 << 20)
			if count == 100 {
				budget = 128 << 20
				if idle >= 1 {
					t.Errorf("small idle CPU exceeds 1%%: %.4f", idle)
				}
			}
			if peak > budget {
				t.Errorf("combined process high-water sum exceeds budget: %d > %d", peak, budget)
			}
			if count == 10000 && p95 >= 250*time.Millisecond {
				t.Errorf("large p95 exceeds 250 ms: %v", p95)
			}
		})
	}
	t.Run("limits", func(t *testing.T) {
		p := newPerformancePair(t, 100, true)
		previous := p.session.overview(t).Generation
		writePerformanceControl(t, p.control, performanceControl{Datasets: 99990})
		p.session.refresh(t)
		previous = p.session.ready(t, 99990, previous)
		// Model records: 99,990 datasets + class/four geoms/four providers/pool = 100,000.
		for _, control := range []performanceControl{{Datasets: 99991}, {Datasets: 20000, Padding: 4000}} {
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
		}
		writePerformanceControl(t, p.control, performanceControl{Datasets: 100})
		p.session.refresh(t)
		p.session.ready(t, 100, previous)
		sampled, peak, samples, gap := p.stop(t)
		t.Logf("PERFORMANCE_LIMITS accepted_model_records=100000 record_exhaustion=pass byte_exhaustion=pass recovery=pass sampled_rss_bytes=%d high_water_sum_bytes=%d samples=%d max_gap_ms=%.1f", sampled, peak, samples, float64(gap.Microseconds())/1000)
	})
}
