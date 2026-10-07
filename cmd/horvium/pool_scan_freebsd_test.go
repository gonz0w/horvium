// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/host/freebsd"
)

// Only the opted-in disposable mirror fixture may call this helper. Its global
// guest-only debug control freezes actual kernel scanning for stable observation;
// it does not replace ZFS writes/status or manufacture progress. Restore on every
// exit; the outer fixture verifies pool GUID/provider backing before destruction.
func nativePoolScanLifecycle(t *testing.T, ctx context.Context, pool, guid string, raw func() string, original string) {
	t.Helper()
	if os.Geteuid() != 0 || os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATUS_VM") != "1" {
		t.Fatal("requires opted-in disposable guest")
	}
	const control = "vfs.zfs.scan_suspend_progress"
	command := func(c context.Context, path string, args ...string) (string, error) {
		cmd := exec.CommandContext(c, path, args...)
		cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
		out, err := cmd.Output()
		return string(out), err
	}
	must := func(t *testing.T, path string, args ...string) string {
		t.Helper()
		out, err := command(ctx, path, args...)
		if err != nil {
			t.Fatal("owned scan fixture command failed (values withheld)", path)
		}
		return out
	}
	source, err := freebsd.NewPoolHealthSource()
	if err != nil {
		t.Fatal("native health unavailable", err)
	}
	wait := func(t *testing.T, accept func(freebsd.PoolHealthObservation) bool) freebsd.PoolHealthObservation {
		t.Helper()
		c, stop := context.WithTimeout(ctx, 20*time.Second)
		defer stop()
		for c.Err() == nil {
			o, e := source.Collect(c, guid, pool)
			if e == nil && accept(o) {
				return o
			}
			select {
			case <-c.Done():
				t.Fatal("scan observation context expired")
			case <-time.After(100 * time.Millisecond):
			}
		}
		t.Fatal("native scan did not reach required reported state")
		return freebsd.PoolHealthObservation{}
	}
	scrub := func(t *testing.T, flags ...string) {
		t.Helper()
		if strings.TrimSpace(must(t, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)) != guid {
			t.Fatal("fixture pool identity changed")
		}
		args := append([]string{"scrub"}, flags...)
		args = append(args, pool)
		must(t, "/sbin/zpool", args...)
	}
	initial := ""
	controlled := false
	var baseline freebsd.PoolHealthObservation
	t.Cleanup(func() {
		if !controlled {
			return
		}
		c, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_, e := command(c, "/sbin/sysctl", control+"="+initial)
		got, verify := command(c, "/sbin/sysctl", "-n", control)
		if e != nil || verify != nil || strings.TrimSpace(got) != initial {
			t.Error("owned guest scan control restoration failed; discard guest")
		}
	})
	if !t.Run("preflight", func(t *testing.T) {
		baseline = wait(t, func(o freebsd.PoolHealthObservation) bool {
			return o.Scan.State != "running" && o.Scan.State != "paused"
		})
		if baseline.Pool.GUID != guid || baseline.Pool.LoadGUID == "" || baseline.Topology.Health != "ONLINE" || raw() != original {
			t.Fatal("owned baseline not intact")
		}
		// The shared debug switch may only be changed in this isolated guest, with no
		// other active maintenance scan. Never restore an unknown/nonzero setting.
		status := must(t, "/sbin/zpool", "status", "-p")
		if strings.Contains(status, "in progress") || strings.Contains(status, "paused since") {
			t.Fatal("another guest scan is active")
		}
		initial = strings.TrimSpace(must(t, "/sbin/sysctl", "-n", control))
		if initial != "0" {
			t.Fatal("unexpected guest scan control baseline")
		}
		controlled = true // Arm restoration before the write, including uncertain failure.
		must(t, "/sbin/sysctl", control+"=1")
		if strings.TrimSpace(must(t, "/sbin/sysctl", "-n", control)) != "1" {
			t.Fatal("guest scan control was not set")
		}
	}) {
		return
	}
	// A failed preflight must never permit a later fixture write.
	if !controlled {
		t.Fatal("scan fixture preflight failed")
	}
	var running, paused freebsd.PoolHealthObservation
	if !t.Run("running", func(t *testing.T) {
		scrub(t)
		running = wait(t, func(o freebsd.PoolHealthObservation) bool { return o.Scan.Kind == "scrub" && o.Scan.State == "running" })
		if !running.Scan.ProgressReported || running.Scan.ErrorsReported || running.Scan.StartedOn == "" || running.Scan.EndedOn != "" || running.Scan.ProgressHundredths > 10000 {
			t.Fatal("invented running scan facts")
		}
	}) {
		return
	}
	if !t.Run("paused", func(t *testing.T) {
		scrub(t, "-p")
		paused = wait(t, func(o freebsd.PoolHealthObservation) bool { return o.Scan.Kind == "scrub" && o.Scan.State == "paused" })
		if !paused.Scan.ProgressReported || paused.Scan.ErrorsReported || paused.Scan.PausedOn == "" || paused.Scan.StartedOn != running.Scan.StartedOn || paused.Scan.EstimateDisplay != "" || paused.Scan.ScanRateDisplay != "" || paused.Scan.IssueRateDisplay != "" {
			t.Fatal("invented paused scan facts")
		}
	}) {
		return
	}
	if !t.Run("paused-preserved", func(t *testing.T) {
		again, e := source.Collect(ctx, guid, pool)
		if e != nil || !reflect.DeepEqual(again, paused) || raw() != original {
			t.Fatal("paused read changed facts/properties", e)
		}
	}) {
		return
	}
	if !t.Run("resumed", func(t *testing.T) {
		scrub(t)
		resumed := wait(t, func(o freebsd.PoolHealthObservation) bool { return o.Scan.Kind == "scrub" && o.Scan.State == "running" })
		if resumed.Scan.StartedOn != running.Scan.StartedOn || resumed.Scan.PausedOn != "" || !resumed.Scan.ProgressReported || resumed.Scan.ErrorsReported {
			t.Fatal("incorrect resumed scan facts")
		}
	}) {
		return
	}
	if !t.Run("cancelled", func(t *testing.T) {
		scrub(t, "-s")
		cancelled := wait(t, func(o freebsd.PoolHealthObservation) bool {
			return o.Scan.Kind == "scrub" && o.Scan.State == "cancelled"
		})
		if cancelled.Scan.EndedOn == "" || cancelled.Scan.ProgressReported || cancelled.Scan.ErrorsReported || cancelled.Scan.StartedOn != "" {
			t.Fatal("invented cancelled scan outcome")
		}
	}) {
		return
	}
	if !t.Run("control-restored-completed", func(t *testing.T) {
		must(t, "/sbin/sysctl", control+"="+initial)
		if strings.TrimSpace(must(t, "/sbin/sysctl", "-n", control)) != initial {
			t.Fatal("guest scan control not restored")
		}
		scrub(t)
		done := wait(t, func(o freebsd.PoolHealthObservation) bool {
			return o.Scan.Kind == "scrub" && o.Scan.State == "completed"
		})
		if !done.Scan.ErrorsReported || done.Scan.Errors != 0 || done.Scan.ProgressReported || done.Scan.EndedOn == "" {
			t.Fatal("incorrect native completion facts")
		}
	}) {
		return
	}
	if !t.Run("preserved", func(t *testing.T) {
		o, e := source.Collect(ctx, guid, pool)
		if e != nil || o.Pool != baseline.Pool || !reflect.DeepEqual(o.Topology.Nodes, baseline.Topology.Nodes) || o.Topology.Health != "ONLINE" || o.Topology.ReadErrors != 0 || o.Topology.WriteErrors != 0 || o.Topology.ChecksumErrors != 0 || o.Topology.DataErrors != 0 || raw() != original || strings.TrimSpace(must(t, "/sbin/sysctl", "-n", control)) != initial {
			t.Fatal("native scan fixture did not preserve state", e)
		}
	}) {
		return
	}
}
