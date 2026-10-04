// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

func readerFixtureCommand(t *testing.T, path string, args ...string) []byte {
	t.Helper()
	ctx, stop := context.WithTimeout(context.Background(), 20*time.Second)
	defer stop()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	b, err := cmd.Output()
	if err != nil {
		t.Fatal("owned reader fixture command failed; output withheld")
	}
	return b
}

type readerProbe struct{ guard, command int }

// Read process rows only in the disposable guest. Never print process names,
// arguments, IDs or host inventory; signal only a verified helper child subtree.
func freezeReaderProbe(t *testing.T, p *performancePair, pool string) readerProbe {
	// Only the post-measurement crash fixture grows: long owned names make the
	// actual native dataset projection exceed the pipe capacity while its Go
	// reader is briefly paused. No product command/source is substituted.
	for i := 0; i < 400; i++ {
		readerFixtureCommand(t, "/sbin/zfs", "create", "-o", "mountpoint=none", pool+fmt.Sprintf("/crash-%03d-", i)+strings.Repeat("x", 200))
	}
	count := len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
	p.session.refresh(t)
	p.session.ready(t, count, "")
	t.Helper()
	owner := p.helper.cmd.Process.Pid
	defer func() { _ = syscall.Kill(owner, syscall.SIGCONT) }()
	kq, err := syscall.Kqueue()
	if err != nil {
		t.Fatal("owned probe event queue")
	}
	defer syscall.Close(kq)
	syscall.CloseOnExec(kq)
	change := syscall.Kevent_t{Ident: uint64(owner), Filter: syscall.EVFILT_PROC, Flags: syscall.EV_ADD | syscall.EV_CLEAR, Fflags: syscall.NOTE_FORK | syscall.NOTE_TRACK | syscall.NOTE_EXEC | syscall.NOTE_EXIT}
	if _, err := syscall.Kevent(kq, []syscall.Kevent_t{change}, nil, nil); err != nil {
		t.Fatal("owned helper process tracking")
	}
	// Observe kernel fork/exec events concurrently with the refresh request.
	// Sampling after a response can miss an entire fast native collection.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	done := make(chan struct{})
	failed := make(chan struct{}, 1)
	go func() {
		defer close(done)
		for ctx.Err() == nil {
			request, _ := http.NewRequestWithContext(ctx, "POST", p.session.base+"/api/refresh", strings.NewReader(url.Values{"csrf": {p.session.csrf}}.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response, err := p.session.http.Do(request)
			if err != nil {
				if ctx.Err() == nil {
					failed <- struct{}{}
				}
				return
			}
			response.Body.Close()
			if response.StatusCode != 202 && response.StatusCode != 409 {
				failed <- struct{}{}
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(20 * time.Millisecond):
			}
		}
	}()
	defer func() { cancel(); <-done }()
	parents := map[int]int{}
	known := map[string]bool{"zfs": true, "zpool": true, "sysctl": true, "uname": true, "freebsd-version": true, "sh": true}
	var events [32]syscall.Kevent_t
	children, executions, candidates := 0, 0, 0
	diagnosed := false
	for ctx.Err() == nil {
		select {
		case <-failed:
			t.Fatal("owned crash refresh failed")
		default:
		}
		timeout := syscall.Timespec{Nsec: 50_000_000}
		n, err := syscall.Kevent(kq, nil, events[:], &timeout)
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			t.Fatal("owned probe event read")
		}
		for _, event := range events[:n] {
			if event.Flags&syscall.EV_ERROR != 0 || event.Fflags&syscall.NOTE_TRACKERR != 0 {
				t.Fatal("owned probe tracking incomplete")
			}
			pid := int(event.Ident)
			if event.Fflags&syscall.NOTE_EXIT != 0 {
				delete(parents, pid)
				_ = syscall.Kill(owner, syscall.SIGCONT)
				continue
			}
			if event.Fflags&syscall.NOTE_CHILD != 0 {
				parents[pid] = int(event.Data)
				children++
			}
			if event.Fflags&syscall.NOTE_EXEC == 0 {
				continue
			}
			executions++
			if parents[pid] == owner {
				_ = syscall.Kill(owner, syscall.SIGSTOP)
				continue
			}
			guard := parents[pid]
			if pid <= 1 || guard <= 1 || parents[guard] != owner {
				continue
			}
			candidates++
			if syscall.Kill(pid, syscall.SIGSTOP) != nil {
				continue
			}
			state, _ := exec.Command("/bin/ps", "-p", strconv.Itoa(pid), "-o", "ppid=,stat=,ucomm=").Output()
			fields := strings.Fields(string(state))
			guardState, _ := exec.Command("/bin/ps", "-p", strconv.Itoa(guard), "-o", "ppid=,ucomm=").Output()
			guardFields := strings.Fields(string(guardState))
			if !diagnosed && len(fields) >= 3 && len(guardFields) >= 2 {
				t.Logf("CAPTURE_SHAPE command_fields=%d parent_matches=%t stopped_state=%t fixed_native_name=%t guard_fields=%d helper_parent_matches=%t fixed_guard_name=%t", len(fields), fields[0] == strconv.Itoa(guard), strings.HasPrefix(fields[1], "T"), known[fields[2]], len(guardFields), guardFields[0] == strconv.Itoa(owner), guardFields[1] == "timeout")
				diagnosed = true
			}
			if len(fields) == 3 && fields[0] == strconv.Itoa(guard) && strings.HasPrefix(fields[1], "T") && known[fields[2]] && len(guardFields) == 2 && guardFields[0] == strconv.Itoa(owner) && guardFields[1] == "timeout" {
				t.Cleanup(func() {
					b, err := exec.Command("/bin/ps", "-p", strconv.Itoa(pid), "-o", "ppid=").Output()
					if err == nil && strings.TrimSpace(string(b)) == strconv.Itoa(guard) {
						_ = syscall.Kill(pid, syscall.SIGCONT)
					}
				})
				return readerProbe{guard, pid}
			}
			_ = syscall.Kill(pid, syscall.SIGCONT)
			_ = syscall.Kill(owner, syscall.SIGCONT)
		}
	}
	t.Fatalf("could not capture actual native command: child_events=%d exec_events=%d guarded_exec_events=%d; no crash qualification", children, executions, candidates)
	return readerProbe{}
}

func readerProcessStopped(t *testing.T, pid int) {
	t.Helper()
	performanceWait(t, func() bool {
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			return true
		}
		ctx, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		b, err := exec.CommandContext(ctx, "/bin/ps", "-p", strconv.Itoa(pid), "-o", "stat=").Output()
		return err == nil && strings.HasPrefix(strings.TrimSpace(string(b)), "Z")
	})
}

func TestNativeReaderResourceBudgets(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("requires explicitly disposable pinned performance guest")
	}
	if os.Geteuid() != 0 {
		t.Fatal("reader resource fixture requires root")
	}
	for _, mode := range []string{"native-small", "projection-10000"} {
		t.Run(mode, func(t *testing.T) {
			count, owned := 10000, 0
			var expected ownedFixture
			expectedRow := func(offset int) string { return fmt.Sprintf("fixture/dataset-%06d", offset) }
			if mode == "native-small" {
				expected = nativeStorageFixture(t)
				// Root + existing data + 98 fresh, unmounted filesystems = 100 owned.
				for i := 0; i < 98; i++ {
					readerFixtureCommand(t, "/sbin/zfs", "create", "-o", "mountpoint=none", expected.Pool+fmt.Sprintf("/item-%03d", i))
				}
				owned = len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name", "-r", expected.Pool))))
				if owned != 100 {
					t.Fatal("independent owned native filesystem count")
				}
				names := strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name", "-s", "name")))
				count = len(names)
				expectedRow = func(offset int) string { return html.EscapeString(names[offset]) }
			}
			c, path, token := nativeRuntimeConfig(t)
			c.IntervalSeconds, c.TimeoutSeconds = 30, 45
			encoded, _ := json.Marshal(c)
			if os.WriteFile(path, encoded, 0644) != nil {
				t.Fatal("native reader config")
			}
			beforeConfig := append([]byte(nil), encoded...)
			digestPath := filepath.Join(c.StateDirectory, "token.sha256")
			beforeDigest, err := os.ReadFile(digestPath)
			if err != nil {
				t.Fatal("native reader digest")
			}
			dir := filepath.Dir(path)
			built, _ := filepath.Abs("../../.tools/horvium")
			binary := filepath.Join(dir, "horvium")
			copyPerformanceExecutable(t, built, binary)
			var helper *nativeProcess
			if mode == "native-small" {
				helper = startNativeProcess(t, binary, []string{"helper", "-config", path}, 0, 0)
			} else {
				fixture, _ := filepath.Abs("../../.tools/reader-fixture.test")
				fixtureBinary := filepath.Join(dir, "reader-fixture.test")
				copyPerformanceExecutable(t, fixture, fixtureBinary)
				cmd := exec.Command(fixtureBinary, "-test.run=^TestNativeProjectionHelper$", "--", path)
				cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
				helper = &nativeProcess{cmd: cmd, done: make(chan error, 1)}
				if cmd.Start() != nil {
					t.Fatal("projection fixture start")
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
			monitor := newUsageMonitor([2]int{helper.cmd.Process.Pid, management.cmd.Process.Pid})
			t.Cleanup(monitor.close)
			p := &performancePair{config: c, path: path, helper: helper, management: management, monitor: monitor}
			p.session = newPerformanceSession(t, c, token)
			previous := p.session.ready(t, count, "")
			if owned != 0 {
				r, e := p.session.http.Get(p.session.base + "/api/datasets?filter=" + url.QueryEscape(expected.Pool+"/data"))
				if e != nil {
					t.Fatal("owned native query")
				}
				var page observation.Page[inventory.Dataset]
				e = json.NewDecoder(io.LimitReader(r.Body, 65536)).Decode(&page)
				r.Body.Close()
				if e != nil || r.StatusCode != http.StatusOK || len(page.Rows) != 1 || page.Rows[0].GUID != expected.DatasetGUID {
					t.Fatal("independent owned native GUID mismatch")
				}
			}
			performancePause(t, 60*time.Second, "reader-warmup-"+mode)
			idle := float64(0)
			if mode == "native-small" {
				before, e := sampleUsage(monitor.pids)
				if e != nil {
					t.Fatal(e)
				}
				started := time.Now()
				performancePause(t, 300*time.Second, "reader-native-idle")
				after, e := sampleUsage(monitor.pids)
				if e != nil {
					t.Fatal(e)
				}
				idle, e = idlePercent(before.CPU, after.CPU, time.Since(started))
				if e != nil || idle >= 1 {
					t.Fatal("native reader idle CPU budget")
				}
			}
			latencies, err := p.session.loadExpected(t, count, expectedRow)
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { _, e := p.session.loadExpected(t, count, expectedRow); done <- e }()
			for i := 0; i < 3; i++ {
				p.session.refresh(t)
				previous = p.session.ready(t, count, previous)
			}
			if e := <-done; e != nil {
				t.Fatal(e)
			}
			sampled, samples, gap := monitor.finish(t)
			var highWater uint64
			if mode == "native-small" {
				socket, e := os.Lstat(c.HelperSocket)
				if e != nil {
					t.Fatal("owned socket")
				}
				probe := freezeReaderProbe(t, p, expected.Pool)
				before := p.session.overview(t).Generation
				helper.stop(t, syscall.SIGKILL, false)
				readerProcessStopped(t, probe.command)
				readerProcessStopped(t, probe.guard)
				performanceWait(t, func() bool { o := p.session.overview(t); return o.Generation == before && o.Datasets.Stale })
				current, e := os.Lstat(c.HelperSocket)
				if e != nil || !os.SameFile(socket, current) || os.Remove(c.HelperSocket) != nil {
					t.Fatal("owned stale socket")
				}
				originalPeak := peakProcessRSS(t, helper)
				helper = startNativeProcess(t, binary, []string{"helper", "-config", path}, 0, 0)
				performanceWait(t, func() bool { info, e := os.Stat(c.HelperSocket); return e == nil && info.Mode().Perm() == 0660 })
				p.session.refresh(t)
				recoveryCount := len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
				p.session.ready(t, recoveryCount, before)
				helper.stop(t, syscall.SIGTERM, true)
				management.stop(t, syscall.SIGTERM, true)
				peak := max(originalPeak, peakProcessRSS(t, helper)) + peakProcessRSS(t, management)
				highWater = peak
			} else {
				helper.stop(t, syscall.SIGTERM, true)
				management.stop(t, syscall.SIGTERM, true)
				highWater = peakProcessRSS(t, helper) + peakProcessRSS(t, management)
			}
			configAfter, _ := os.ReadFile(path)
			digestAfter, _ := os.ReadFile(digestPath)
			if !bytes.Equal(beforeConfig, configAfter) || !bytes.Equal(beforeDigest, digestAfter) {
				t.Fatal("native reader changed boot state")
			}
			if _, err := os.Lstat(c.HelperSocket); !os.IsNotExist(err) {
				t.Fatal("native helper socket leaked")
			}
			if mode == "native-small" {
				t.Log("NATIVE_READER_CRASH captured_native_command=pass subtree_exit=pass stale_retention=pass recovery=pass boot_state=unchanged")
			}
			slices.Sort(latencies)
			budget := uint64(256 << 20)
			if mode == "native-small" {
				budget = 128 << 20
			}
			if max(sampled, highWater) > budget || mode == "projection-10000" && latencies[949] >= 250*time.Millisecond {
				t.Fatal("native reader resource/latency budget")
			}
			record := map[string]any{"mode": mode, "datasets": count, "owned_datasets": owned, "clients": 10, "requests": 1000, "replacement_requests": 1000, "replacements": 3, "warmup_seconds": 60, "idle_seconds": 0, "idle_cpu_percent_one_core": idle, "sampled_combined_peak_rss_bytes": sampled, "conservative_combined_high_water_rss_bytes": highWater, "budget_peak_rss_bytes": max(sampled, highWater), "rss_samples": samples, "max_sample_gap_ms": float64(gap.Microseconds()) / 1000, "p50_ms": float64(latencies[499].Microseconds()) / 1000, "p95_ms": float64(latencies[949].Microseconds()) / 1000, "p99_ms": float64(latencies[989].Microseconds()) / 1000}
			if mode == "native-small" {
				record["idle_seconds"] = 300
			}
			encoded, _ = json.Marshal(record)
			t.Log("NATIVE_READER_RESULT " + string(encoded))
		})
	}
}

// A short native gate verifies capture/containment before repeating the measured
// warm-up and idle workloads. It uses the same built product and capture code.
func TestNativeBuiltProbeCaptureAndOwnerDeath(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("explicit disposable native opt-in required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("capture fixture requires root")
	}
	owned := nativeStorageFixture(t)
	for i := 0; i < 98; i++ {
		readerFixtureCommand(t, "/sbin/zfs", "create", "-o", "mountpoint=none", owned.Pool+fmt.Sprintf("/item-%03d", i))
	}
	count := len(strings.Fields(string(readerFixtureCommand(t, "/sbin/zfs", "list", "-H", "-t", "filesystem", "-o", "name"))))
	c, path, token := nativeRuntimeConfig(t)
	built, _ := filepath.Abs("../../.tools/horvium")
	binary := filepath.Join(filepath.Dir(path), "horvium")
	copyPerformanceExecutable(t, built, binary)
	helper := startNativeProcess(t, binary, []string{"helper", "-config", path}, 0, 0)
	nativeWait(t, func() bool { info, e := os.Stat(c.HelperSocket); return e == nil && info.Mode().Perm() == 0660 })
	management := startNativeProcess(t, binary, []string{"serve", "-config", path}, c.ManagementUID, c.SocketGID)
	p := &performancePair{config: c, path: path, helper: helper, management: management}
	p.session = newPerformanceSession(t, c, token)
	p.session.ready(t, count, "")
	probe := freezeReaderProbe(t, p, owned.Pool)
	helper.stop(t, syscall.SIGKILL, false)
	readerProcessStopped(t, probe.command)
	readerProcessStopped(t, probe.guard)
	management.stop(t, syscall.SIGTERM, true)
	t.Log("NATIVE_BUILT_CAPTURE command_stopped=pass owner_death_subtree_exit=pass")
}
