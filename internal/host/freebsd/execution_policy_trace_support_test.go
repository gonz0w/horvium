// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
type policyTraceSampler struct {
	stop, done chan struct{}
	once       sync.Once
	samples    []policyTraceSample
	err        error
}

// Test-only observer. Native callers fix runtime.Stack and all three offsets;
// portable tests exercise early return and schedule without changing product time.
func startPolicyTraceSampler(start time.Time, dir string, offsets [3]time.Duration, capture func([]byte) int) (*policyTraceSampler, error) {
	if start.IsZero() || capture == nil || offsets[0] <= 0 || offsets[0] >= offsets[1] || offsets[1] >= offsets[2] || offsets[2] > 110*time.Second {
		return nil, errors.New("invalid fixed diagnostic schedule")
	}
	s := &policyTraceSampler{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(s.done)
		buffer := make([]byte, 64<<10)
		stopped := false
		for i, offset := range offsets {
			row := policyTraceSample{ID: i + 1, Offset: int64(offset), Status: "skipped"}
			if !stopped {
				timer := time.NewTimer(time.Until(start.Add(offset)))
				select {
				case <-s.stop:
					stopped = true
				case <-timer.C:
				}
				timer.Stop()
				select {
				case <-s.stop:
					stopped = true
				default:
				}
			}
			if !stopped {
				row.Begin = time.Since(start).Nanoseconds()
				n := capture(buffer)
				row.End = time.Since(start).Nanoseconds()
				if n <= 0 || n >= len(buffer) {
					s.err = errors.New("empty or truncated snapshot")
					return
				}
				digest, e := policyTraceWrite(dir, i+1, buffer[:n])
				if e != nil {
					s.err = e
					return
				}
				row.Bytes = n
				row.SHA = digest
				row.Status = "captured"
			}
			s.samples = append(s.samples, row)
		}
	}()
	return s, nil
}
func (s *policyTraceSampler) finish(budget time.Duration) ([]policyTraceSample, error) {
	if s == nil || budget <= 0 || budget > 5*time.Second {
		return nil, errors.New("invalid sampler join")
	}
	s.once.Do(func() { close(s.stop) })
	timer := time.NewTimer(budget)
	defer timer.Stop()
	select {
	case <-s.done:
		return s.samples, s.err
	case <-timer.C:
		return nil, errors.New("sampler join incomplete")
	}
}
func policyTraceWrite(dir string, id int, data []byte) (string, error) {
	if id < 1 || id > 3 || len(data) == 0 || len(data) >= 64<<10 {
		return "", errors.New("invalid fixed snapshot")
	}
	info, e := os.Lstat(dir)
	if e != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return "", errors.New("private snapshot directory")
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		return "", e
	}
	defer root.Close()
	f, e := root.OpenFile(fmt.Sprintf("sample%d.stack", id), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return "", e
	}
	n, e := f.Write(data)
	closed := f.Close()
	if e != nil {
		return "", e
	}
	if closed != nil {
		return "", closed
	}
	if n != len(data) {
		return "", errors.New("incomplete snapshot")
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

var policyTraceSymbol = regexp.MustCompile(`^[A-Za-z0-9_./*()[\]{}·-]+$`)
var policyTraceLocation = regexp.MustCompile(`^\s*([A-Za-z0-9_./-]+\.(?:go|s)):([0-9]+)(?: \+0x[0-9a-f]+)?$`)

func policyTraceFrames(data []byte) (string, []policyTraceFrame) {
	if len(data) == 0 || len(data) >= 64<<10 {
		return "unlocated", nil
	}
	var blocks []string
	for _, block := range strings.Split(string(data), "\n\n") {
		if strings.HasPrefix(block, "goroutine ") && strings.Contains(block, "github.com/gonz0w/horvium/internal/host/freebsd.activateNativeClosedExecutionOwner(") {
			blocks = append(blocks, block)
		}
	}
	if len(blocks) == 0 {
		return "unlocated", nil
	}
	if len(blocks) != 1 {
		return "ambiguous", nil
	}
	lines := strings.Split(blocks[0], "\n")
	var frames []policyTraceFrame
	for i := 1; i < len(lines); i += 2 {
		if strings.HasPrefix(lines[i], "created by ") {
			break
		}
		if i+1 >= len(lines) {
			return "unlocated", nil
		}
		argument := strings.LastIndex(lines[i], "(")
		if argument < 1 {
			return "unlocated", nil
		}
		symbol := lines[i][:argument]
		location := policyTraceLocation.FindStringSubmatch(lines[i+1])
		if len(symbol) > 256 || !policyTraceSymbol.MatchString(symbol) || len(location) != 3 || len(location[1]) > 256 {
			return "unlocated", nil
		}
		file := strings.TrimPrefix(location[1], "github.com/gonz0w/horvium/")
		line, e := strconv.Atoi(location[2])
		if e != nil || line < 1 || strings.HasPrefix(file, "/") || filepath.Clean(file) != file || strings.Contains(file, "..") {
			return "unlocated", nil
		}
		frames = append(frames, policyTraceFrame{File: file, Line: line, Symbol: symbol})
		if len(frames) > 64 {
			return "unlocated", nil
		}
	}
	if len(frames) == 0 {
		return "unlocated", nil
	}
	return "located", frames
}

func TestPolicyTraceEarlyReturn(t *testing.T) {
	s, err := startPolicyTraceSampler(time.Now(), policyTraceTestDirectory(t), [3]time.Duration{30 * time.Second, 75 * time.Second, 110 * time.Second}, func([]byte) int { t.Error("capture after early return"); return 1 })
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
	dir := policyTraceTestDirectory(t)
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
	if e := os.Symlink("sample1.stack", filepath.Join(dir, "sample2.stack")); e != nil {
		t.Fatal(e)
	}
	if _, e := policyTraceWrite(dir, 2, data); e == nil {
		t.Fatal("existing symlink cannot redirect snapshot write")
	}
	if e := os.Link(filepath.Join(dir, "sample1.stack"), filepath.Join(dir, "sample3.stack")); e != nil {
		t.Fatal(e)
	}
	if _, e := policyTraceWrite(dir, 3, data); e == nil {
		t.Fatal("existing hard link cannot replace original")
	}
	if e := os.Chmod(dir, 0755); e != nil {
		t.Fatal(e)
	}
	if _, e := policyTraceWrite(dir, 2, data); e == nil {
		t.Fatal("nonprivate directory denied")
	}
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	link := filepath.Join(t.TempDir(), "linked-snapshots")
	if e := os.Symlink(dir, link); e != nil {
		t.Fatal(e)
	}
	if _, e := policyTraceWrite(link, 2, data); e == nil {
		t.Fatal("symlinked directory denied")
	}
	unchanged, e := os.ReadFile(filepath.Join(dir, "sample1.stack"))
	if e != nil || string(unchanged) != string(data) {
		t.Fatal("rejected writes preserve original bytes")
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

func TestPolicyTraceSchedule(t *testing.T) {
	dir := policyTraceTestDirectory(t)
	captured := make(chan struct{})
	calls := 0
	offsets := [3]time.Duration{time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond}
	s, e := startPolicyTraceSampler(time.Now(), dir, offsets, func(buffer []byte) int {
		calls++
		n := copy(buffer, "private bounded snapshot\n")
		if calls == 3 {
			close(captured)
		}
		return n
	})
	if e != nil {
		t.Fatal(e)
	}
	select {
	case <-captured:
	case <-time.After(time.Second):
		t.Fatal("declared samples not captured")
	}
	rows, e := s.finish(time.Second)
	if e != nil || len(rows) != 3 || calls != 3 {
		t.Fatal("one capture per fixed offset", e)
	}
	for i, row := range rows {
		if row.ID != i+1 || row.Offset != int64(offsets[i]) || row.Begin < row.Offset || row.End < row.Begin || row.Status != "captured" || len(row.SHA) != 64 {
			t.Fatal("actual ordered capture", row)
		}
		info, e := os.Lstat(filepath.Join(dir, fmt.Sprintf("sample%d.stack", i+1)))
		if e != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || int(info.Size()) != row.Bytes {
			t.Fatal("bounded private raw file", e)
		}
	}
}

func TestPolicyTraceFailedCaptureAndJoin(t *testing.T) {
	for _, size := range []int{0, 65536, -1} {
		s, e := startPolicyTraceSampler(time.Now(), policyTraceTestDirectory(t), [3]time.Duration{time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond}, func([]byte) int { return size })
		if e != nil {
			t.Fatal(e)
		}
		select {
		case <-s.done:
		case <-time.After(time.Second):
			t.Fatal("bad capture did not finish")
		}
		if _, e = s.finish(time.Second); e == nil {
			t.Fatal("invalid/truncated capture cannot complete")
		}
	}
	entered, release := make(chan struct{}), make(chan struct{})
	s, e := startPolicyTraceSampler(time.Now(), policyTraceTestDirectory(t), [3]time.Duration{time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond}, func(b []byte) int { close(entered); <-release; return copy(b, "private snapshot\n") })
	if e != nil {
		t.Fatal(e)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("capture did not start")
	}
	_, joinError := s.finish(time.Millisecond)
	close(release)
	_, cleanupError := s.finish(time.Second)
	if joinError == nil || cleanupError != nil {
		t.Fatal("incomplete join must fail; portable worker must still stop", joinError, cleanupError)
	}
}

func policyTraceTestDirectory(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "snapshots")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}
