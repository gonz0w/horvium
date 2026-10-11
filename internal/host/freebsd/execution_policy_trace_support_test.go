// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"errors"
	"testing"
	"time"
)

type policyTraceSample struct {
	ID     int    `json:"id"`
	Offset int64  `json:"offset_ns"`
	Begin  int64  `json:"begin_ns"`
	End    int64  `json:"end_ns"`
	Bytes  int    `json:"bytes"`
	SHA    string `json:"sha256"`
	Status string `json:"status"`
}
type policyTraceFrame struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Symbol string `json:"symbol"`
}
type policyTraceSampler struct{}

// Compilable test-only failing-first stubs; not product behavior.
func startPolicyTraceSampler(time.Time, string, [3]time.Duration, func([]byte) int) (*policyTraceSampler, error) {
	return nil, errors.New("sampler not implemented")
}
func (*policyTraceSampler) finish(time.Duration) ([]policyTraceSample, error) {
	return nil, errors.New("sampler not implemented")
}
func policyTraceFrames([]byte) (string, []policyTraceFrame) { return "unlocated", nil }
func policyTraceWrite(string, int, []byte) (string, error) {
	return "", errors.New("safe snapshots not implemented")
}

func TestPolicyTraceEarlyReturn(t *testing.T) {
	s, err := startPolicyTraceSampler(time.Now(), t.TempDir(), [3]time.Duration{30 * time.Second, 75 * time.Second, 110 * time.Second}, func([]byte) int { t.Error("capture after early return"); return 1 })
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	samples, err := s.finish(time.Second)
	if err != nil || len(samples) != 3 || time.Since(start) > time.Second {
		t.Fatal("early return must stop future timers", err)
	}
	for i, row := range samples {
		if row.ID != i+1 || row.Status != "skipped" || row.Bytes != 0 {
			t.Fatal("honest skipped sample")
		}
	}
}
func TestPolicyTraceSnapshotFiles(t *testing.T) {
	dir := t.TempDir()
	data := []byte("bounded private stack\n")
	if digest, err := policyTraceWrite(dir, 1, data); err != nil || len(digest) != 64 {
		t.Fatal("original bounded snapshot", err)
	}
	if _, err := policyTraceWrite(dir, 1, data); err == nil {
		t.Fatal("duplicate snapshot cannot replace original")
	}
	for _, id := range []int{0, 4} {
		if _, err := policyTraceWrite(dir, id, data); err == nil {
			t.Fatal("fixed names only")
		}
	}
	for _, data := range [][]byte{nil, make([]byte, 64<<10), make([]byte, 64<<10+1)} {
		if _, err := policyTraceWrite(dir, 2, data); err == nil {
			t.Fatal("empty/full/truncated snapshot")
		}
	}
}
func TestPolicyTraceSanitize(t *testing.T) {
	raw := []byte("goroutine 9 [syscall]:\nos.(*Process).wait(0xprivate)\n\tos/exec.go:20 +0x33\ngithub.com/gonz0w/horvium/internal/host/freebsd.activateNativeClosedExecutionOwner(0x123, 0x456)\n\tgithub.com/gonz0w/horvium/internal/host/freebsd/execution_restart_freebsd.go:30 +0x99\n\n")
	status, frames := policyTraceFrames(raw)
	if status != "located" || len(frames) != 2 || frames[0].Symbol != "os.(*Process).wait" || frames[1].File != "internal/host/freebsd/execution_restart_freebsd.go" {
		t.Fatal("source-only frames without raw arguments", status, frames)
	}
	for _, bad := range [][]byte{nil, append(append([]byte(nil), raw...), raw...), []byte("goroutine 1 [running]:\nprivate(0x123)\n\t/private/path:1 +0x3\n")} {
		s, f := policyTraceFrames(bad)
		if s == "located" || len(f) != 0 {
			t.Fatal("missing/ambiguous/unsafe stacks cannot locate factory")
		}
	}
}
