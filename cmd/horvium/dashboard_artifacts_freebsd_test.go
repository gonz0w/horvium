//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func dashboardNativeFact(path string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, path, args...)
	command.WaitDelay = time.Second
	command.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	var output dashboardFactBuffer
	command.Stdout = &output
	command.Stderr = io.Discard
	if e := command.Run(); e != nil {
		return "", errors.New("native fact command incomplete")
	}
	return string(output.data), nil
}

// Separate read-only preflight for a future reviewed host caller. Not selected by
// the resource parents/runner. It creates no storage, helper, service or listener.
func TestNativeDashboardArtifactProvenance(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("explicit disposable FreeBSD opt-in required")
	}
	if os.Getuid() != 0 || runtime.GOARCH != "amd64" {
		t.Fatal("preflight requires owned FreeBSD/amd64 guest controller")
	}
	root, e := os.OpenRoot(".tools")
	if e != nil {
		t.Fatal("missing owned artifact directory")
	}
	defer func() {
		if root.Close() != nil {
			t.Error("artifact root close failed")
		}
	}()
	executable, e := os.Executable()
	if e != nil {
		t.Fatal("controller executable unavailable")
	}
	before, e := dashboardObserveArtifacts(root, executable, buildinfo.Read)
	if e != nil {
		t.Fatal("guest artifact preflight failed; details withheld")
	}
	kernel, ke := dashboardNativeFact("/bin/freebsd-version", "-r")
	userland, ue := dashboardNativeFact("/bin/freebsd-version", "-u")
	zfs, ze := dashboardNativeFact("/sbin/zfs", "version")
	memory, me := dashboardNativeFact("/sbin/sysctl", "-n", "hw.physmem")
	if ke != nil || ue != nil || ze != nil || me != nil {
		t.Fatal("actual native environment incomplete")
	}
	facts, e := dashboardNormalizeGuestFacts(dashboardGuestFacts{kernel, userland, zfs, memory})
	if e != nil {
		t.Fatal("actual native facts invalid; details withheld")
	}
	after, e := dashboardObserveArtifacts(root, executable, buildinfo.Read)
	if e != nil || after.GoVersion != before.GoVersion {
		t.Fatal("guest artifacts changed during native observations")
	}
	for role, hash := range before.Artifacts {
		if after.Artifacts[role] != hash {
			t.Fatal("guest artifacts changed during native observations")
		}
	}
	record := struct {
		dashboardArtifactObservation
		dashboardGuestFactObservation
	}{after, facts}
	data, e := json.Marshal(record)
	if e != nil {
		t.Fatal("cannot frame guest observation")
	}
	t.Logf("DASHBOARD_GUEST_ARTIFACTS %s", data)
}
