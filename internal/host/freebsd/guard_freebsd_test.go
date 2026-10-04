// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
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
			t.Cleanup(func() {
				_ = syscall.Kill(-group, syscall.SIGKILL)
				_ = syscall.Kill(child, syscall.SIGKILL)
			})
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
		})
	}
}
