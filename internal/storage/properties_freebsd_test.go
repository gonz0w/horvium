// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

type fixtureCapture struct{ bytes.Buffer }

func (b *fixtureCapture) Write(p []byte) (int, error) {
	if len(p) > 1<<20-b.Len() {
		return 0, ErrLimit
	}
	return b.Buffer.Write(p)
}

// TestNativePropertyObservations only creates and destroys its own declared
// fixtures in a fresh opt-in QEMU guest. No product mutation or eligibility gate
// is implemented by these privileged TEST setup commands.
func TestNativePropertyObservations(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("requires explicit disposable QEMU opt-in")
	}
	if os.Geteuid() != 0 {
		t.Fatal("requires root in the disposable guest")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := func(ctx context.Context, path string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, path, args...)
		cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
		var out fixtureCapture
		cmd.Stdout = &out
		cmd.Stderr = io.Discard
		err := cmd.Run()
		return out.Bytes(), err
	}
	must := func(path string, args ...string) []byte {
		out, err := command(ctx, path, args...)
		if err != nil {
			t.Fatalf("fixture command %s failed (values withheld): %v", path, err)
		}
		return out
	}
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal(err)
	}
	pool := "horvium_m2_" + hex.EncodeToString(nonce[:])
	dir := t.TempDir()
	backing := filepath.Join(dir, "owned.img")
	file, err := os.OpenFile(backing, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(512 << 20); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	unit := strings.TrimSpace(string(must("/sbin/mdconfig", "-a", "-t", "vnode", "-f", backing)))
	if !regexp.MustCompile(`^md[0-9]+$`).MatchString(unit) {
		t.Fatal("invalid fixture md unit; discard guest")
	}
	declaredPool := false
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if declaredPool {
			if _, err := command(cleanupCtx, "/sbin/zpool", "list", "-H", pool); err == nil {
				if _, err := command(cleanupCtx, "/sbin/zpool", "destroy", pool); err != nil {
					t.Error("owned pool cleanup failed; discard guest")
					return
				}
			}
			if _, err := command(cleanupCtx, "/sbin/zpool", "list", "-H", pool); err == nil {
				t.Error("owned pool remains")
				return
			}
		}
		if _, err := command(cleanupCtx, "/sbin/mdconfig", "-d", "-u", strings.TrimPrefix(unit, "md")); err != nil {
			t.Error("owned md detach failed")
			return
		}
		if _, err := command(cleanupCtx, "/sbin/mdconfig", "-l", "-n", "-f", backing); err == nil {
			t.Error("owned md remains attached")
			return
		}
		if !t.Failed() {
			t.Log("PROPERTY_OBSERVATION_RESULT default=pass local=pass inherited=pass received=pass shadowed=pass no_limit=pass rename=pass name_reuse=pass foreign_preservation=pass eligibility_denied=pass teardown=pass")
		}
	})
	if got := strings.TrimSpace(string(must("/sbin/mdconfig", "-l", "-n", "-f", backing))); got != strings.TrimPrefix(unit, "md") {
		t.Fatal("fixture backing correspondence failed")
	}
	if _, err := command(ctx, "/sbin/zpool", "list", "-H", pool); err == nil {
		t.Fatal("fixture namespace unexpectedly exists")
	}
	declaredPool = true
	must("/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", pool, "/dev/"+unit)
	must("/sbin/zfs", "create", "-o", "mountpoint=none", pool+"/data")
	projection := func() []byte {
		return must("/sbin/zfs", "get", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,property,value,received,source", "guid,quota,refquota,compression", pool)
	}
	parse := func() []Filesystem {
		files, err := ParseProperties(projection(), pool)
		if err != nil {
			t.Fatal("actual native projection denied (values withheld):", err)
		}
		return files
	}
	find := func(files []Filesystem, suffix string) Filesystem {
		for _, f := range files {
			if f.Name == pool+suffix {
				return f
			}
		}
		t.Fatal("owned filesystem absent")
		return Filesystem{}
	}
	initial := parse()
	if find(initial, "").Properties.Compression.Source != Default || find(initial, "/data").Properties.Compression.Source != Default {
		t.Fatal("default source mismatch")
	}
	must("/sbin/zfs", "set", "compression=lz4", pool)
	must("/sbin/zfs", "set", "quota=4194304", "refquota=2097152", pool+"/data")
	must("/sbin/zfs", "create", "-o", "mountpoint=none", "-o", "compression=off", pool+"/foreign")
	must("/sbin/zfs", "create", "-o", "mountpoint=none", "-o", "compression=lz4", "-o", "quota=8388608", "-o", "refquota=4194304", pool+"/source")
	must("/sbin/zfs", "snapshot", pool+"/source@seed")
	stream, err := os.OpenFile(filepath.Join(dir, "owned.stream"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	send := exec.CommandContext(ctx, "/sbin/zfs", "send", "-p", pool+"/source@seed")
	send.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	send.Stdout = stream
	send.Stderr = io.Discard
	if err := send.Run(); err != nil {
		t.Fatal("fixture send failed (values withheld)")
	}
	if _, err := stream.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	receive := exec.CommandContext(ctx, "/sbin/zfs", "receive", "-u", pool+"/received")
	receive.Env = send.Env
	receive.Stdin = stream
	receive.Stderr = io.Discard
	if err := receive.Run(); err != nil {
		t.Fatal("fixture receive failed (values withheld)")
	}
	received := find(parse(), "/received")
	if v := received.Properties.Compression; v.Source != Received || v.ReceivedValue != "lz4" || v.Effective != "lz4" {
		t.Fatal("received compression mismatch")
	}
	if v := received.Properties.Quota; v.Source != Received || v.ReceivedValue != "8388608" || v.Effective != "8388608" {
		t.Fatal("received quota mismatch")
	}
	must("/sbin/zfs", "set", "compression=off", pool+"/received")
	shadowed := find(parse(), "/received").Properties.Compression
	if shadowed.Source != Local || shadowed.Effective != "off" || shadowed.ReceivedValue != "lz4" {
		t.Fatal("shadowed received value lost")
	}
	must("/sbin/zfs", "inherit", "-S", "compression", pool+"/received")
	if find(parse(), "/received").Properties.Compression.Source != Received {
		t.Fatal("received source restoration mismatch")
	}
	must("/sbin/zfs", "set", "quota=none", pool+"/data")
	files := parse()
	data := find(files, "/data")
	root := find(files, "")
	if data.Properties.Quota.Effective != "none" || data.Properties.RefQuota.Effective != "2097152" || data.Properties.Compression.Source != Inherited || data.Properties.Compression.SourceGUID != root.GUID {
		t.Fatal("literal limit or inherited GUID mismatch")
	}
	// Independent, different native projection correlates GUIDs, not product data.
	independent := must("/sbin/zfs", "list", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,guid", pool)
	nativeGUIDs := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(independent)), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) != 2 {
			t.Fatal("independent GUID shape invalid")
		}
		nativeGUIDs[fields[0]] = fields[1]
	}
	for _, f := range files {
		if nativeGUIDs[f.Name] != f.GUID {
			t.Fatal("independent GUID disagreement")
		}
	}
	if len(nativeGUIDs) != len(files) {
		t.Fatal("filesystem count mismatch")
	}
	before := must("/sbin/zfs", "get", "-H", "-p", "-r", "-o", "name,property,value,received,source", "guid,quota,refquota,compression,mountpoint,canmount,readonly", pool)
	mounts := must("/sbin/zfs", "list", "-H", "-r", "-o", "name,mounted,mountpoint", pool)
	_ = parse()
	if after := must("/sbin/zfs", "get", "-H", "-p", "-r", "-o", "name,property,value,received,source", "guid,quota,refquota,compression,mountpoint,canmount,readonly", pool); !bytes.Equal(before, after) {
		t.Fatal("read-only projection changed native properties or foreign sentinel")
	}
	if after := must("/sbin/zfs", "list", "-H", "-r", "-o", "name,mounted,mountpoint", pool); !bytes.Equal(mounts, after) {
		t.Fatal("read-only projection changed mounts")
	}
	// Parsing provides no authority, even when every projected property is valid.
	if _, err := Build(time.Now(), "fixture-session", Policy{Revision: 1, PoolGUID: "1", PoolName: pool}, Observation{Complete: true, PoolGUID: "1", PoolName: pool, Filesystems: files}, Intent{FilesystemGUID: root.GUID, Property: Quota, Value: "none"}); err != ErrUnavailable {
		t.Fatal("parser enabled native eligibility", err)
	}
	must("/sbin/zfs", "rename", pool+"/data", pool+"/renamed")
	if find(parse(), "/renamed").GUID != data.GUID {
		t.Fatal("rename changed filesystem identity")
	}
	must("/sbin/zfs", "create", "-o", "mountpoint=none", pool+"/data")
	if find(parse(), "/data").GUID == data.GUID {
		t.Fatal("reused name confused GUID identity")
	}
}
