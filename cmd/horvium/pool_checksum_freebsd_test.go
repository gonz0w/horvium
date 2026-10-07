// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/host/freebsd"
)

// Test-only real kernel read-buffer corruption, targeted to one GUID in the
// already-owned mirror. No backing file is manually corrupted. The healthy
// replica must preserve content. Clear only the acknowledged handler ID; an
// unknown registration outcome requires discarding the disposable guest.
func nativePoolChecksum(t *testing.T, ctx context.Context, pool, guid, leaf, fixtureDir string, raw func() string, original string) {
	t.Helper()
	if os.Geteuid() != 0 || os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATUS_VM") != "1" {
		t.Fatal("requires opted-in disposable guest")
	}
	const inject = "/usr/bin/zinject"
	const emptyHandlers = "No handlers registered.\nRun 'zinject -h' for usage information.\n"
	command := func(c context.Context, path string, args ...string) (string, error) {
		cmd := exec.CommandContext(c, path, args...)
		cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
		out, e := cmd.Output()
		return string(out), e
	}
	must := func(t *testing.T, path string, args ...string) string {
		t.Helper()
		out, e := command(ctx, path, args...)
		if e != nil {
			t.Fatal("owned checksum command failed (values withheld)", path)
		}
		return out
	}
	verifyPool := func(t *testing.T) {
		t.Helper()
		if strings.TrimSpace(must(t, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)) != guid {
			t.Fatal("owned pool identity changed")
		}
	}
	source, e := freebsd.NewPoolHealthSource()
	if e != nil {
		t.Fatal("native health unavailable", e)
	}
	wait := func(t *testing.T) freebsd.PoolHealthObservation {
		t.Helper()
		c, stop := context.WithTimeout(ctx, 20*time.Second)
		defer stop()
		for c.Err() == nil {
			o, e := source.Collect(c, guid, pool)
			if e == nil && o.Scan.Kind == "scrub" && o.Scan.State == "completed" {
				return o
			}
			select {
			case <-c.Done():
				t.Fatal("checksum scan observation expired")
			case <-time.After(100 * time.Millisecond):
			}
		}
		t.Fatal("checksum scrub did not complete")
		return freebsd.PoolHealthObservation{}
	}
	dataset := pool + "/checksum_probe"
	mount := filepath.Join(fixtureDir, "checksum-mount")
	datasetGUID, handler, leafGUID := "", "", ""
	created, registrationAttempted := false, false
	// Cleanup order matters: registered handlers hold the pool. Remove only this
	// fixture's ID before removing its GUID-verified mounted child filesystem.
	cleanup := func() bool {
		c, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if registrationAttempted {
			if handler != "" {
				_, _ = command(c, inject, "-c", handler)
			}
			out, e := command(c, inject)
			if e != nil || out != emptyHandlers {
				return false
			}
			handler = ""
			registrationAttempted = false
		}
		if created {
			got, e := command(c, "/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", dataset)
			if e != nil || datasetGUID == "" || strings.TrimSpace(got) != datasetGUID {
				return false
			}
			if _, e = command(c, "/sbin/zfs", "destroy", dataset); e != nil {
				return false
			}
			created = false
			if _, e = command(c, "/sbin/zfs", "list", "-H", dataset); e == nil {
				return false
			}
		}
		if e := os.Remove(mount); e != nil && !os.IsNotExist(e) {
			return false
		}
		return true
	}
	t.Cleanup(func() {
		if !cleanup() {
			t.Error("owned checksum cleanup failed; discard guest")
		}
	})
	var baseline freebsd.PoolHealthObservation
	if !t.Run("preflight", func(t *testing.T) {
		verifyPool(t)
		baseline = wait(t)
		if baseline.Pool.GUID != guid || baseline.Pool.LoadGUID == "" || baseline.Topology.Health != "ONLINE" || raw() != original {
			t.Fatal("owned checksum baseline not intact")
		}
		for _, node := range baseline.Topology.Nodes {
			if node.Address == leaf && node.Kind == "leaf" {
				leafGUID = node.GUID
			}
		}
		if leafGUID == "" {
			t.Fatal("selected owned leaf not bound")
		}
		if must(t, inject) != emptyHandlers || strings.TrimSpace(must(t, "/sbin/sysctl", "-n", "vfs.zfs.scan_suspend_progress")) != "0" {
			t.Fatal("foreign fault handler or scan control active")
		}
		digest := strings.TrimSpace(must(t, "/sbin/sha256", "-q", inject))
		if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(digest) {
			t.Fatal("invalid base injector hash")
		}
		t.Log("STORAGE_CHECKSUM_TOOL_SHA256 " + digest)
	}) {
		return
	}
	var content []byte
	if !t.Run("content-fixture", func(t *testing.T) {
		verifyPool(t)
		if _, e := command(ctx, "/sbin/zfs", "list", "-H", dataset); e == nil {
			t.Fatal("checksum dataset already exists")
		}
		if os.Mkdir(mount, 0700) != nil {
			t.Fatal("exclusive checksum mount directory")
		}
		must(t, "/sbin/zfs", "create", "-o", "mountpoint="+mount, "-o", "compression=off", "-o", "quota=1048576", dataset)
		created = true
		datasetGUID = strings.TrimSpace(must(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", dataset))
		if n, e := strconv.ParseUint(datasetGUID, 10, 64); e != nil || n == 0 || strconv.FormatUint(n, 10) != datasetGUID {
			t.Fatal("invalid owned checksum dataset identity")
		}
		content = make([]byte, 32<<10)
		if _, e := rand.Read(content); e != nil {
			t.Fatal("checksum fixture bytes")
		}
		f, e := os.OpenFile(filepath.Join(mount, "sentinel"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			t.Fatal("exclusive content file")
		}
		n, e := f.Write(content)
		syncErr := f.Sync()
		closeErr := f.Close()
		if e != nil || n != len(content) || syncErr != nil || closeErr != nil {
			t.Fatal("durable fixture content")
		}
		must(t, "/sbin/zpool", "sync", pool)
	}) {
		return
	}
	if !t.Run("fault-registered", func(t *testing.T) {
		verifyPool(t)
		o, e := source.Collect(ctx, guid, pool)
		if e != nil {
			t.Fatal("fresh leaf binding unavailable", e)
		}
		found := false
		for _, node := range o.Topology.Nodes {
			if node.GUID == leafGUID && node.Address == leaf && node.Kind == "leaf" && node.Health == "ONLINE" {
				found = true
			}
		}
		if !found || must(t, inject) != emptyHandlers {
			t.Fatal("owned leaf or empty handler baseline changed")
		}
		registrationAttempted = true // Arm cleanup before the possible kernel effect.
		out := must(t, inject, "-q", "-d", leafGUID, "-e", "corrupt", "-T", "read", "-f", "100", pool)
		id, e := strconv.ParseUint(strings.TrimSuffix(out, "\n"), 10, 31)
		if e != nil || id == 0 || strconv.FormatUint(id, 10)+"\n" != out {
			t.Fatal("unknown handler acknowledgement; discard guest")
		}
		handler = strconv.FormatUint(id, 10)
		leafNumber, e := strconv.ParseUint(leafGUID, 10, 64)
		if e != nil {
			t.Fatal("invalid selected leaf GUID")
		}
		matches := 0
		for _, line := range strings.Split(must(t, inject), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[0] == handler && fields[1] == pool && fields[2] == strconv.FormatUint(leafNumber, 16) {
				matches++
			}
		}
		if matches != 1 {
			t.Fatal("native handler not bound to owned pool/leaf")
		}
	}) {
		return
	}
	var findings freebsd.PoolHealthObservation
	if !t.Run("scrub-findings", func(t *testing.T) {
		verifyPool(t)
		must(t, "/sbin/zpool", "scrub", pool)
		findings = wait(t)
		found := false
		for _, node := range findings.Topology.Nodes {
			if node.GUID == leafGUID && node.ChecksumErrors > 0 {
				found = true
			}
		}
		if !found || !findings.Scan.ErrorsReported || findings.Scan.Errors != 0 || findings.Scan.ProgressReported || findings.Topology.DataErrors != 0 || findings.Pool.LoadGUID != baseline.Pool.LoadGUID {
			t.Fatal("checksum findings absent or conflated with completion")
		}
	}) {
		return
	}
	if !t.Run("fault-cleared", func(t *testing.T) {
		must(t, inject, "-c", handler)
		if must(t, inject) != emptyHandlers {
			t.Fatal("owned fault handler remains")
		}
		handler = ""
		registrationAttempted = false
	}) {
		return
	}
	verifyContent := func(t *testing.T) {
		t.Helper()
		got, e := os.ReadFile(filepath.Join(mount, "sentinel"))
		if e != nil || !bytes.Equal(got, content) {
			t.Fatal("mirror did not preserve fixture content")
		}
	}
	if !t.Run("content-preserved", func(t *testing.T) { verifyContent(t) }) {
		return
	}
	if !t.Run("clean-scrub", func(t *testing.T) {
		verifyPool(t)
		must(t, "/sbin/zpool", "clear", pool)
		must(t, "/sbin/zpool", "scrub", pool)
		o := wait(t)
		if o.Topology.Health != "ONLINE" || o.Topology.ReadErrors != 0 || o.Topology.WriteErrors != 0 || o.Topology.ChecksumErrors != 0 || o.Topology.DataErrors != 0 || !o.Scan.ErrorsReported || o.Scan.Errors != 0 {
			t.Fatal("owned clean scan did not restore healthy counters")
		}
		for _, node := range o.Topology.Nodes {
			if node.Health != "ONLINE" || node.ReadErrors != 0 || node.WriteErrors != 0 || node.ChecksumErrors != 0 {
				t.Fatal("owned member errors not cleared")
			}
		}
		verifyContent(t)
	}) {
		return
	}
	t.Run("fixture-restored", func(t *testing.T) {
		if !cleanup() || raw() != original || must(t, inject) != emptyHandlers {
			t.Fatal("owned checksum fixture did not restore native state")
		}
		o, e := source.Collect(ctx, guid, pool)
		if e != nil || o.Pool != baseline.Pool {
			t.Fatal("pool/import identity drift after fixture", e)
		}
	})
}
