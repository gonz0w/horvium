// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

// TestNativeDiscovery uses the actual product reader, not a mock or research probe.
// It is opt-in and must run solely in a fresh disposable FreeBSD QEMU guest.
func TestNativeDiscovery(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("requires explicit disposable QEMU opt-in")
	}
	if os.Geteuid() != 0 {
		t.Fatal("native fixture setup requires root inside the disposable guest")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := func(path string, args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, path, args...)
		cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
		out, err := cmd.Output()
		return string(out), err
	}
	must := func(path string, args ...string) string {
		out, err := command(path, args...)
		if err != nil {
			t.Fatalf("fixture command %s failed (output withheld): %v", path, err)
		}
		return out
	}
	var token [8]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatal(err)
	}
	pool := "horvium_m1_" + hex.EncodeToString(token[:])
	file := filepath.Join(t.TempDir(), "disk.img")
	f, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(512 << 20); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	unit := strings.TrimSpace(must("/sbin/mdconfig", "-a", "-t", "vnode", "-f", file))
	if !regexp.MustCompile(`^md[0-9]+$`).MatchString(unit) {
		t.Fatal("unexpected generated md unit; discard guest")
	}
	created := false
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		clean := func(path string, args ...string) error { return exec.CommandContext(cleanupCtx, path, args...).Run() }
		if created {
			if err := clean("/sbin/zpool", "destroy", pool); err != nil {
				t.Error("owned pool cleanup failed; discard guest")
				return
			}
			if clean("/sbin/zpool", "list", "-H", pool) == nil {
				t.Error("fixture pool still exists")
			}
		}
		if err := clean("/sbin/mdconfig", "-d", "-u", strings.TrimPrefix(unit, "md")); err != nil {
			t.Error("owned md cleanup failed; discard guest")
			return
		}
		if clean("/sbin/mdconfig", "-l", "-n", "-f", file) == nil {
			t.Error("fixture md backing file still attached")
		}
	})
	if backing := strings.TrimSpace(must("/sbin/mdconfig", "-l", "-n", "-f", file)); backing != strings.TrimPrefix(unit, "md") {
		t.Fatal("md backing-file mismatch")
	}
	// Never force creation, mount data, or accept an operator's disk/pool arguments.
	must("/sbin/zpool", "create", "-m", "none", pool, "/dev/"+unit)
	created = true
	must("/sbin/zfs", "create", "-o", "mountpoint=none", pool+"/data")
	properties := must("/sbin/zfs", "get", "-H", "-p", "-o", "property,value", "mountpoint,readonly,compression", pool+"/data")
	datasetGUID := strings.TrimSpace(must("/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", pool+"/data"))
	poolValues := strings.Fields(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid,size", pool))
	if len(poolValues) != 2 {
		t.Fatal("independent pool properties have unexpected shape")
	}
	poolSize, err := strconv.ParseUint(poolValues[1], 10, 64)
	if err != nil {
		t.Fatal("independent size invalid")
	}
	source, err := New(inventory.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	observed := source.Collect(ctx)
	if !observed.Complete() {
		if observed.GEOM.Status != inventory.OK {
			out, err := readNative(ctx, geom, inventory.DefaultLimits().MaxBytes)
			if err == nil {
				_, err = inventory.ParseGEOM(out.data, inventory.DefaultLimits())
			}
			t.Logf("GEOM shape diagnostic (values withheld): %v", err)
		}
		t.Fatalf("native collection incomplete (inventory withheld): host=%s/%v geom=%s/%v pools=%s/%v datasets=%s/%v", observed.Host.Status, observed.Host.Problems, observed.GEOM.Status, observed.GEOM.Problems, observed.Pools.Status, observed.Pools.Problems, observed.Datasets.Status, observed.Datasets.Problems)
	}
	if observed.Host.Data.PhysicalMemory == nil || *observed.Host.Data.PhysicalMemory == 0 {
		t.Fatal("native memory missing")
	}
	foundProvider, foundPool, foundDataset := false, false, false
	for _, p := range observed.GEOM.Data.Providers {
		if p.Name == unit {
			foundProvider = true
			if p.Kind != inventory.Memory || p.MediaSize != 512<<20 || p.SectorSize == 0 || p.Identity != inventory.IdentityNotApplicable {
				t.Fatal("fixture provider geometry/classification incorrect")
			}
		}
	}
	for _, p := range observed.Pools.Data {
		if p.Name == pool {
			foundPool = true
			if p.GUID != poolValues[0] || p.Size != poolSize || p.Health != "ONLINE" {
				t.Fatal("independent pool values disagree")
			}
		}
	}
	for _, d := range observed.Datasets.Data {
		if d.Name == pool+"/data" {
			foundDataset = true
			if d.GUID != datasetGUID {
				t.Fatal("independent dataset GUID disagrees")
			}
		}
	}
	if !foundProvider || !foundPool || !foundDataset {
		t.Fatal("test-owned resources missing from actual native inventory")
	}
	if after := must("/sbin/zfs", "get", "-H", "-p", "-o", "property,value", "mountpoint,readonly,compression", pool+"/data"); after != properties {
		t.Fatal("discovery changed fixture properties")
	}
	must("/sbin/zfs", "rename", pool+"/data", pool+"/renamed")
	renamed := source.Collect(ctx)
	if !renamed.Complete() {
		t.Fatal("post-rename native inventory incomplete")
	}
	foundRenamed := false
	for _, d := range renamed.Datasets.Data {
		if d.Name == pool+"/data" {
			t.Fatal("removed name remains in new observation")
		}
		if d.Name == pool+"/renamed" {
			foundRenamed = true
			if d.GUID != datasetGUID {
				t.Fatal("GUID changed through rename")
			}
		}
	}
	if !foundRenamed {
		t.Fatal("external rename not observed")
	}
	t.Logf("native contract passed: architecture=%s installed=%s running=%s userland=%s zfs=%s module=%s", observed.Host.Data.Architecture, observed.Host.Data.InstalledKernel, observed.Host.Data.RunningKernel, observed.Host.Data.Userland, observed.Host.Data.ZFSUserland, observed.Host.Data.ZFSKernel)
}
