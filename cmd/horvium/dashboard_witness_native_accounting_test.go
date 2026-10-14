//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"encoding/binary"
	"golang.org/x/sys/unix"
	"math"
	"os"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

type witnessNativeUsageSample struct {
	AtNS                                              int64 `json:"at_ns"`
	ReadNS                                            int64 `json:"read_ns"`
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
func witnessNativeSample(roles []witnessActor, abi witnessABI, scratch []byte) (out witnessNativeUsageSample, err error) {
	started := time.Now()
	var originals [][]byte
	var readErrno syscall.Errno
	defer func() {
		out.ReadNS = int64(time.Since(started))
		if err == nil && out.ReadNS > int64(2*time.Second) {
			err = witnessFailure("usage-kernel-deadline")
		}
		if err != nil {
			_ = witnessWriteDiagnosticJSON("witness-rejected-sample-"+strconv.Itoa(len(roles))+".json", struct {
				Rows            string
				Roles           []witnessActor
				ABI             witnessABI
				KernelOriginals [][]byte
				ElapsedNS       int64
				Errno           uint64
			}{"", roles, abi, originals, out.ReadNS, uint64(readErrno)})
		}
	}()
	if !abi.valid() || len(scratch) != witnessFrameCap || len(roles) != 2 && len(roles) != 3 {
		return out, witnessFailure("usage-roles")
	}
	layout := abi.usageLayout()
	size := uintptr(len(scratch))
	_, _, readErrno = unix.Syscall6(unix.SYS___SYSCTL, uintptr(unsafe.Pointer(&abi.ProcMIB[0])), 3, uintptr(unsafe.Pointer(&scratch[0])), uintptr(unsafe.Pointer(&size)), 0, 0)
	if readErrno != 0 || size > uintptr(len(scratch)) {
		return out, witnessFailure("usage-kernel-read")
	}
	processes, e := witnessDecodeKernelCensus(scratch[:size], layout)
	if e != nil {
		return out, witnessFailure("usage-kernel-read")
	}
	ownedPID := map[int]bool{}
	for _, role := range roles {
		ownedPID[role.PID] = true
	}
	// Only fixture roles and observer children are retained as original binary
	// rejection inputs. The bounded whole census is never archived as inventory.
	for offset := 0; offset < int(size); offset += layout.Size {
		raw := scratch[offset : offset+layout.Size]
		pid := int(int32(binary.LittleEndian.Uint32(raw[layout.PID:])))
		if ownedPID[pid] || pid == os.Getpid() || len(roles) == 3 && processes[pid].Parent == roles[2].PID {
			originals = append(originals, append([]byte(nil), raw...))
		}
	}
	seen := map[int]bool{}
	for i, a := range roles {
		if a.PID <= 1 || seen[a.PID] || a.PID == os.Getpid() {
			return out, witnessFailure("usage-duplicate")
		}
		seen[a.PID] = true
		r, ok := processes[a.PID]
		if !ok || r.UID != a.UID || r.Sec != a.Sec || r.Usec != a.Usec || r.RSS == 0 {
			return out, witnessFailure("usage-identity")
		}
		if i < 2 {
			if r.RSS > math.MaxUint64-out.PairRSS || r.CPU > math.MaxInt64-out.PairCPU {
				return out, witnessFailure("usage-sum")
			}
			out.PairRSS += r.RSS
			out.PairCPU += r.CPU
		} else {
			out.ObserverRSS = r.RSS
			out.ObserverCPU = r.CPU
		}
	}
	if len(roles) == 3 {
		for _, child := range processes {
			if child.Parent != roles[2].PID {
				continue
			}
			if child.PID <= 1 || (child.UID != roles[0].UID && child.UID != roles[2].UID) || seen[child.PID] || out.ConnectorCount != 0 {
				return out, witnessFailure("usage-descendants")
			}
			out.ConnectorCount = 1
			out.ConnectorRSS = child.RSS
		}
	}
	controller, ok := processes[os.Getpid()]
	if !ok || controller.UID != uint32(os.Getuid()) || controller.RSS == 0 {
		return out, witnessFailure("usage-controller")
	}
	out.ControllerRSS = controller.RSS
	out.ControllerCPU = controller.CPU
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
	scratch := make([]byte, witnessFrameCap)
	first, e := witnessNativeSample(roles, abi, scratch)
	if e != nil {
		t.Fatal("original initial instrumentation sample rejected", e)
	}
	m := &witnessNativeUsage{stop: make(chan struct{}), done: make(chan struct{}), started: time.Now(), roles: roles, abi: abi}
	first.AtNS = 1
	m.samples = append(m.samples, first)
	go func() {
		defer close(m.done)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-m.stop:
				return
			case <-ticker.C:
				s, e := witnessNativeSample(roles, abi, scratch)
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
