// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestNativeProbeOwnerProcess(t *testing.T) {
	args := flag.Args()
	if len(args) != 2 {
		return
	}
	dir, mode := args[0], args[1]
	_, _ = executeNative(context.Background(), os.Args[0], []string{"-test.run=^TestProbeTreeProcess$", "--", mode, filepath.Join(dir, "child"), filepath.Join(dir, "probe")}, 128)
	os.Exit(0)
}

func TestNativeProbeOwnerDeath(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("requires disposable FreeBSD QEMU opt-in")
	}
	for _, mode := range []string{"cancel", "escaped-child"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			owner := exec.Command(os.Args[0], "-test.run=^TestNativeProbeOwnerProcess$", "--", dir, mode)
			owner.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if owner.Start() != nil {
				t.Fatal("owned helper fixture start")
			}
			done := make(chan error, 1)
			go func() { done <- owner.Wait() }()
			waited := false
			t.Cleanup(func() {
				if !waited {
					_ = owner.Process.Kill()
					select {
					case <-done:
					case <-time.After(3 * time.Second):
						t.Error("owned helper fixture reap")
					}
				}
			})
			probe := fixturePID(t, filepath.Join(dir, "probe"))
			child := fixturePID(t, filepath.Join(dir, "child"))
			group, err := syscall.Getpgid(probe)
			if err != nil || group <= 1 || group == syscall.Getpgrp() {
				t.Fatal("probe isolation")
			}
			childGroup, err := syscall.Getpgid(child)
			if err != nil || (mode == "escaped-child" && childGroup == group) || (mode == "cancel" && childGroup != group) {
				t.Fatal("descendant fixture group")
			}
			// Only the still-owned helper handle may be signalled. Saved IDs
			// remain read-only observations after its death; fixtures self-expire.
			if os.WriteFile(filepath.Join(dir, "probe.release"), nil, 0600) != nil {
				t.Fatal("release owned probe")
			}
			if owner.Process.Kill() != nil {
				t.Fatal("kill helper during active probe")
			}
			select {
			case <-done:
				waited = true
			case <-time.After(3 * time.Second):
				t.Fatal("helper death did not complete")
			}
			assertProbeStopped(t, probe)
			assertProbeStopped(t, child)
			if group != probe {
				assertProbeStopped(t, group)
			}
		})
	}
}

func TestNativeGuardedProbeBounds(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("requires disposable FreeBSD QEMU opt-in")
	}
	for _, tc := range []struct {
		mode string
		want error
	}{
		{"ok", nil}, {"overflow", ErrLimit}, {"stderr", ErrLimit}, {"exit", ErrExit}, {"hang", context.DeadlineExceeded},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			duration := 5 * time.Second
			if tc.mode == "hang" {
				duration = 300 * time.Millisecond
			}
			ctx, stop := context.WithTimeout(context.Background(), duration)
			defer stop()
			out, err := executeNative(ctx, os.Args[0], []string{"-test.run=^TestCommandProcess$", "--", tc.mode}, 128)
			if !errors.Is(err, tc.want) || out.consumed > 128 {
				t.Fatalf("guarded probe class/bound: want %v, got %v", tc.want, err)
			}
			if tc.want != nil && len(out.data) != 0 {
				t.Fatal("guarded failure published output")
			}
			if tc.want == nil && string(out.data) != "safe\n" {
				t.Fatal("guard lost successful native output")
			}
		})
	}
	out, err := executeNative(context.Background(), "/nonexistent/horvium-probe", nil, 512)
	if !errors.Is(err, ErrUnavailable) || len(out.data) != 0 || out.consumed > 512 {
		t.Fatal("guarded missing executable class/output")
	}
}
