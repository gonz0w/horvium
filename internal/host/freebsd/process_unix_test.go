//go:build freebsd || darwin || linux || openbsd || netbsd

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

// Only the reexecuted test binary accepts these fixture paths/modes. Product
// readers continue to use their fixed, parameter-free native projections.
func TestProbeTreeProcess(t *testing.T) {
	args := flag.Args()
	if len(args) != 3 {
		return
	}
	mode, ready, parent := args[0], args[1], args[2]
	if mode == "descendant" || mode == "sentinel" {
		if err := os.WriteFile(ready, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			os.Exit(2)
		}
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	if err := os.WriteFile(parent, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		os.Exit(2)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestProbeTreeProcess$", "--", "descendant", ready, parent)
	child.Env = os.Environ()
	if mode == "escaped-child" {
		child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	if mode != "closed-pipes" {
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
	}
	if child.Start() != nil {
		os.Exit(2)
	}
	until := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(until) {
			os.Exit(2)
		}
		time.Sleep(5 * time.Millisecond)
	}
	// Let the supervising test verify group ownership before triggering any
	// immediate exit/output overflow; no scheduler race can hide the descendant.
	for {
		if _, err := os.Stat(parent + ".release"); err == nil {
			break
		}
		if time.Now().After(until) {
			os.Exit(2)
		}
		time.Sleep(5 * time.Millisecond)
	}
	switch mode {
	case "stdout-limit":
		fmt.Print(strings.Repeat("x", 4096))
	case "stderr-limit":
		fmt.Fprint(os.Stderr, strings.Repeat("sensitive", 1024))
	case "parent-exit":
		fmt.Print("unpublishable output")
		os.Exit(0)
	case "parent-failure":
		fmt.Fprint(os.Stderr, "sensitive native failure")
		os.Exit(2)
	case "closed-pipes":
		fmt.Print("safe\n")
		os.Exit(0)
	}
	// Waiting reaps the child when normal cancellation kills the whole group.
	_ = child.Wait()
	os.Exit(0)
}

func fixturePID(t *testing.T, path string) int {
	t.Helper()
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		data, err := os.ReadFile(path)
		if err == nil {
			pid, err := strconv.Atoi(string(data))
			if err == nil && pid > 1 {
				return pid
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("owned probe fixture did not become ready")
	return 0
}

func assertProbeStopped(t *testing.T, pid int) {
	t.Helper()
	until := time.Now().Add(3 * time.Second)
	for time.Now().Before(until) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return
		}
		// A killed orphan may await init's reap. It is no longer executing; do
		// not mistake a zombie for an active probe or require host reaper timing.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		state, err := exec.CommandContext(ctx, "/bin/ps", "-p", strconv.Itoa(pid), "-o", "stat=").Output()
		cancel()
		if err == nil && strings.HasPrefix(strings.TrimSpace(string(state)), "Z") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("owned probe process remains active after bounded execution")
}

func TestProbeProcessGroupCleanup(t *testing.T) {
	// A separate, test-owned group represents unrelated host work. Each probe
	// must leave it running while terminating only its own descendants.
	sentinelReady := filepath.Join(t.TempDir(), "sentinel")
	sentinel := exec.Command(os.Args[0], "-test.run=^TestProbeTreeProcess$", "--", "sentinel", sentinelReady, "unused")
	sentinel.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := sentinel.Start(); err != nil {
		t.Fatal(err)
	}
	sentinelDone := make(chan error, 1)
	sentinelWatch, stopSentinelWatch := context.WithCancel(context.Background())
	sentinelObserved := make(chan error, 1)
	go func() { sentinelObserved <- observeChildExit(sentinelWatch, sentinel.Process.Pid) }()
	t.Cleanup(func() {
		stopSentinelWatch()
		_ = sentinel.Process.Kill() // This test's sentinel has not been waited.
		go func() { sentinelDone <- sentinel.Wait() }()
		select {
		case <-sentinelObserved:
		case <-time.After(time.Second):
			t.Error("sentinel observer did not join")
		}
		select {
		case <-sentinelDone:
		case <-time.After(3 * time.Second):
			t.Error("unrelated fixture process could not be reaped")
		}
	})
	fixturePID(t, sentinelReady)
	for _, tc := range []struct {
		mode string
		want error
	}{
		{"deadline", context.DeadlineExceeded},
		{"cancel", context.Canceled},
		{"collection-cancel", context.Canceled},
		{"stdout-limit", ErrLimit},
		{"stderr-limit", ErrLimit},
		{"parent-exit", ErrExit},
		{"parent-failure", ErrExit},
		{"closed-pipes", nil},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			dir := t.TempDir()
			ready, parent := filepath.Join(dir, "child"), filepath.Join(dir, "parent")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.mode == "deadline" {
				var stop context.CancelFunc
				ctx, stop = context.WithTimeout(ctx, 2*time.Second)
				defer stop()
			}
			type response struct {
				output   readOutput
				err      error
				snapshot inventory.Snapshot
				calls    int
			}
			done := make(chan response, 1)
			go func() {
				if tc.mode == "collection-cancel" {
					calls := 0
					source := testSource(func(ctx context.Context, _ probe, limit int) (readOutput, error) {
						calls++
						return execute(ctx, os.Args[0], []string{"-test.run=^TestProbeTreeProcess$", "--", tc.mode, ready, parent}, limit)
					})
					source.commandTimeout, source.totalTimeout = 10*time.Second, 10*time.Second
					snapshot := source.Collect(ctx)
					done <- response{err: ctx.Err(), snapshot: snapshot, calls: calls}
					return
				}
				out, err := execute(ctx, os.Args[0], []string{"-test.run=^TestProbeTreeProcess$", "--", tc.mode, ready, parent}, 128)
				done <- response{output: out, err: err}
			}()
			// Bind read-only observations to this fixture before releasing it.
			root := fixturePID(t, parent)
			// The executor owns this fresh group through its sole Wait. Every fixture
			// self-expires; saved numeric IDs grant no fallback signal authority.
			child := fixturePID(t, ready)
			group, err := syscall.Getpgid(child)
			if err != nil || group != root {
				t.Fatal("probe child did not inherit the isolated fixture group")
			}
			if err := os.WriteFile(parent+".release", nil, 0600); err != nil {
				t.Fatal(err)
			}
			if tc.mode == "cancel" || tc.mode == "collection-cancel" {
				cancel()
			}
			select {
			case result := <-done:
				if tc.mode == "collection-cancel" {
					s := result.snapshot
					if result.calls != 1 || s.Complete() || s.Host.Status != inventory.Unavailable || s.GEOM.Status != inventory.Unavailable || s.Pools.Status != inventory.Unavailable || s.Datasets.Status != inventory.Unavailable {
						t.Fatal("cancelled native collection continued probes or published success")
					}
					for _, problems := range [][]inventory.Problem{s.Host.Problems, s.GEOM.Problems, s.Pools.Problems, s.Datasets.Problems} {
						if len(problems) == 0 {
							t.Fatal("cancelled section lost its failure state")
						}
						for _, problem := range problems {
							if problem.Code != "cancelled" {
								t.Fatal("cancelled probe has an incorrect safe error class")
							}
						}
					}
				}
				if !errors.Is(result.err, tc.want) {
					t.Fatalf("want %v, got %v", tc.want, result.err)
				}
				if tc.want != nil && len(result.output.data) != 0 {
					t.Fatal("failed probe published native output")
				}
				if tc.want == nil && string(result.output.data) != "safe\n" {
					t.Fatal("successful probe lost its bounded output")
				}
				if result.output.consumed > 128 {
					t.Fatal("probe exceeded its aggregate byte allowance")
				}
			case <-time.After(5 * time.Second):
				cancel()
				t.Fatal("probe did not return within its execution bound")
			}
			assertProbeStopped(t, root)
			assertProbeStopped(t, child)
			select {
			case <-sentinelObserved:
				t.Fatal("probe cleanup stopped unrelated work")
			default:
			}
			// Failed/cancelled work must not poison the next fixed probe.
			out, err := execute(context.Background(), os.Args[0], []string{"-test.run=^TestCommandProcess$", "--", "ok"}, 128)
			if err != nil || string(out.data) != "safe\n" {
				t.Fatal("subsequent healthy probe failed")
			}
		})
	}
}
