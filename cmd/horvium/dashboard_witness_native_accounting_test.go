//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"encoding/binary"
	"golang.org/x/sys/unix"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type witnessNativeUsageSample struct {
	AtNS                                              int64 `json:"at_ns"`
	PairRSS, ObserverRSS, ConnectorRSS, ControllerRSS uint64
	PairCPU, ObserverCPU, ControllerCPU               int64
	ConnectorCount                                    int
}
type witnessNativeUsage struct {
	mu         sync.Mutex
	stopOnce   sync.Once
	stop, done chan struct{}
	started    time.Time
	samples    []witnessNativeUsageSample
	err        error
	roles      []witnessActor
	abi        witnessABI
}

func witnessLifetime(a witnessActor, abi witnessABI) bool {
	b, e := unix.SysctlRaw("kern.proc.pid", a.PID)
	return e == nil && len(b) == abi.ProcSize && int(int32(binary.LittleEndian.Uint32(b[abi.ProcPID:]))) == a.PID && binary.LittleEndian.Uint32(b[abi.ProcUID:]) == a.UID && int64(binary.LittleEndian.Uint64(b[abi.ProcSec:])) == a.Sec && int64(binary.LittleEndian.Uint64(b[abi.ProcUsec:])) == a.Usec
}
func witnessNativeSample(roles []witnessActor, abi witnessABI) (witnessNativeUsageSample, error) {
	var out witnessNativeUsageSample
	if len(roles) != 2 && len(roles) != 3 {
		return out, errDashboardWitness
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/ps", "-axo", "pid=,ppid=,rss=,time=")
	cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	b, e := cmd.Output()
	if e != nil || len(b) > witnessFrameCap {
		return out, errDashboardWitness
	}
	type row struct {
		parent int
		rss    uint64
		cpu    time.Duration
	}
	processes := map[int]row{}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		f := strings.Fields(line)
		if len(f) != 4 {
			return out, errDashboardWitness
		}
		pid, e := strconv.Atoi(f[0])
		parent, pe := strconv.Atoi(f[1])
		rss, re := strconv.ParseUint(f[2], 10, 64)
		cpu, ce := cpuTime(f[3])
		if e != nil || pe != nil || re != nil || ce != nil || pid < 0 || parent < 0 || rss > math.MaxUint64/1024 {
			return out, errDashboardWitness
		}
		if _, ok := processes[pid]; ok {
			return out, errDashboardWitness
		}
		processes[pid] = row{parent, rss * 1024, cpu}
	}
	for i, a := range roles {
		r, ok := processes[a.PID]
		if !ok || r.rss == 0 || !witnessLifetime(a, abi) {
			return out, errDashboardWitness
		}
		if i < 2 {
			if r.rss > math.MaxUint64-out.PairRSS || int64(r.cpu) > math.MaxInt64-out.PairCPU {
				return out, errDashboardWitness
			}
			out.PairRSS += r.rss
			out.PairCPU += int64(r.cpu)
		} else {
			out.ObserverRSS = r.rss
			out.ObserverCPU = int64(r.cpu)
		}
	}
	if len(roles) == 3 {
		observer := roles[2]
		for _, r := range processes {
			if r.parent == observer.PID {
				out.ConnectorCount++
				out.ConnectorRSS += r.rss
			}
		}
		if out.ConnectorCount > 1 {
			return out, errDashboardWitness
		}
	}
	current, ok := processes[os.Getpid()]
	if !ok || current.rss == 0 {
		return out, errDashboardWitness
	}
	out.ControllerRSS = current.rss
	out.ControllerCPU = int64(current.cpu)
	return out, nil
}
func newWitnessNativeUsage(t *testing.T, p *performancePair, observer *witnessFixtureObserver, abi witnessABI) *witnessNativeUsage {
	t.Helper()
	roles := []witnessActor{}
	pids := []int{p.management.cmd.Process.Pid, p.helper.cmd.Process.Pid}
	if observer != nil {
		pids = append(pids, observer.process.cmd.Process.Pid)
	}
	for _, pid := range pids {
		actor, e := witnessObserveActor(pid, abi)
		if e != nil {
			t.Fatal("original measurement actor")
		}
		roles = append(roles, actor)
	}
	m := &witnessNativeUsage{stop: make(chan struct{}), done: make(chan struct{}), started: time.Now(), roles: roles, abi: abi}
	go func() {
		defer close(m.done)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-m.stop:
				return
			case <-ticker.C:
				s, e := witnessNativeSample(roles, abi)
				s.AtNS = int64(time.Since(m.started))
				m.mu.Lock()
				if e != nil {
					m.err = e
					m.mu.Unlock()
					return
				}
				m.samples = append(m.samples, s)
				m.mu.Unlock()
			}
		}
	}()
	t.Cleanup(m.close)
	return m
}
func (m *witnessNativeUsage) close() { m.stopOnce.Do(func() { close(m.stop); <-m.done }) }
func (m *witnessNativeUsage) finish(t *testing.T) []witnessNativeUsageSample {
	t.Helper()
	m.close()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil || len(m.samples) == 0 {
		t.Fatal("original instrumentation sample unavailable", m.err)
	}
	return append([]witnessNativeUsageSample(nil), m.samples...)
}
func witnessNativeCPU(u *syscall.Rusage) (int64, error) {
	if u.Utime.Sec < 0 || u.Stime.Sec < 0 || u.Utime.Usec < 0 || u.Utime.Usec >= 1000000 || u.Stime.Usec < 0 || u.Stime.Usec >= 1000000 || u.Utime.Sec > math.MaxInt64/1000000000-u.Stime.Sec {
		return 0, errDashboardWitness
	}
	total := (u.Utime.Sec + u.Stime.Sec) * 1000000000
	micros := (u.Utime.Usec + u.Stime.Usec) * 1000
	if total > math.MaxInt64-micros {
		return 0, errDashboardWitness
	}
	return total + micros, nil
}
func witnessUsageDiagnostic(t *testing.T, mode string, samples []witnessNativeUsageSample, receipt witnessNativeReceipt) {
	t.Helper()
	var paired, augmented, controller uint64
	var gap, previous int64
	for _, s := range samples {
		if s.PairRSS > math.MaxUint64-s.ObserverRSS || s.PairRSS+s.ObserverRSS > math.MaxUint64-s.ConnectorRSS {
			t.Fatal("instrumentation sum overflow")
		}
		paired = max(paired, s.PairRSS)
		augmented = max(augmented, s.PairRSS+s.ObserverRSS+s.ConnectorRSS)
		controller = max(controller, s.ControllerRSS)
		if previous != 0 {
			gap = max(gap, s.AtNS-previous)
		}
		previous = s.AtNS
	}
	t.Logf("NATIVE_WITNESS_USAGE mode=%s sampled_pair=%d sampled_augmented=%d observer_high=%d connector_high=%d connector_started=%d connector_reaped=%d connector_active=%d connector_cpu_ns=%d controller_peak=%d samples=%d max_gap_ns=%d nominal_ns=100000000 resource_acceptance=false", mode, paired, augmented, receipt.ObserverHigh, receipt.ConnectorMaxRSS, receipt.ConnectorStarted, receipt.Connectors, receipt.ConnectorActive, receipt.ConnectorCPUNS, controller, len(samples), gap)
	if receipt.ConnectorStarted != receipt.Connectors || receipt.ConnectorActive != 0 || receipt.ObserverHigh == 0 {
		t.Fatal("unaccounted instrumentation owner")
	}
}
