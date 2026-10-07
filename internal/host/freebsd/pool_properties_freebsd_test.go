// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// This test-only setup declares every device/pool itself in a disposable guest.
// Product collection uses the actual guarded fixed native reader; no fake
// eligibility, alternate command, native mutation or state publisher is installed.
func TestNativePoolPropertyCollection(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PROPERTY_VM") != "1" {
		t.Skip("requires disposable property guest opt-ins")
	}
	if os.Geteuid() != 0 {
		t.Fatal("requires disposable guest root")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := func(ctx context.Context, path string, args ...string) (string, error) {
		out, err := execute(ctx, path, args, 1<<20)
		return string(out.data), err
	}
	must := func(path string, args ...string) string {
		out, err := command(ctx, path, args...)
		if err != nil {
			t.Fatalf("fixture command failed (values withheld): %s: %v", path, err)
		}
		return out
	}
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal(err)
	}
	pool := "horvium_property_" + hex.EncodeToString(nonce[:])
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "foreign-sentinel")
	if err := os.WriteFile(sentinel, []byte("independent fixture sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	type md struct{ unit, path string }
	var devices []md
	created := false
	guid := ""
	passed := map[string]bool{}
	check := func(name string, ok bool) {
		if passed[name] || !ok {
			t.Fatal("native property collection assertion", name)
		}
		passed[name] = true
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if created {
			actual, err := command(cleanup, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)
			if err != nil || guid == "" || strings.TrimSpace(actual) != guid {
				t.Error("owned pool identity changed; discard guest")
				return
			}
			if _, err = command(cleanup, "/sbin/zpool", "destroy", pool); err != nil {
				t.Error("owned pool teardown failed")
				return
			}
			if _, err = command(cleanup, "/sbin/zpool", "list", "-H", pool); err == nil {
				t.Error("owned pool remains")
				return
			}
		}
		for _, d := range devices {
			actual, err := command(cleanup, "/sbin/mdconfig", "-l", "-n", "-f", d.path)
			if err != nil || strings.TrimSpace(actual) != strings.TrimPrefix(d.unit, "md") {
				t.Error("owned provider/backing changed; discard guest")
				return
			}
			if _, err = command(cleanup, "/sbin/mdconfig", "-d", "-u", strings.TrimPrefix(d.unit, "md")); err != nil {
				t.Error("owned provider detach failed")
				return
			}
			if _, err = command(cleanup, "/sbin/mdconfig", "-l", "-n", "-f", d.path); err == nil {
				t.Error("owned provider remains")
				return
			}
		}
		if t.Failed() {
			return
		}
		passed["teardown"] = true
		cases := []string{"complete", "pool_identity", "dataset_identity", "property_values", "inheritance", "snapshot_owner", "features", "topology", "eligibility_denied", "wrong_guid_denied", "rename_fresh", "readonly_denied", "volume_denied", "preservation", "backing_preserved", "teardown", "policy_bound", "policy_missing_denied", "policy_change_denied", "policy_scope_denied", "policy_rename_denied", "policy_closed_denied"}
		cases = append(cases, "system_root_backing", "system_pool_backing", "safety_bound", "safety_system_mount_denied", "safety_fresh_restored", "system_swap_observed", "safety_swap_disjoint", "system_mount_preserved", "system_fixture_teardown", "unsupported_md_denied")
		if len(passed) != len(cases) {
			t.Error("incomplete native assertions")
			return
		}
		fields := make([]string, 0, len(cases))
		for _, name := range cases {
			if !passed[name] {
				t.Error("missing native assertion", name)
				return
			}
			fields = append(fields, name+"=pass")
		}
		t.Log("POOL_PROPERTY_COLLECTION_RESULT " + strings.Join(fields, " "))
	})
	nativePropertySystemFixture(t, ctx, check)
	for n := 0; n < 2; n++ {
		path := filepath.Join(dir, "disk-"+strconv.Itoa(n)+".img")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if err = f.Truncate(256 << 20); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
		unit := strings.TrimSpace(must("/sbin/mdconfig", "-a", "-t", "vnode", "-f", path))
		if !regexp.MustCompile(`^md[0-9]+$`).MatchString(unit) {
			t.Fatal("unexpected provider; discard guest")
		}
		devices = append(devices, md{unit, path})
	}
	if _, err := command(ctx, "/sbin/zpool", "list", "-H", pool); err == nil {
		t.Fatal("fixture namespace exists")
	}
	must("/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", pool, "mirror", "/dev/"+devices[0].unit, "/dev/"+devices[1].unit)
	created = true
	guid = strings.TrimSpace(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool))
	must("/sbin/zfs", "create", "-o", "compression=lz4", "-o", "quota=1048576", pool+"/data")
	must("/sbin/zfs", "snapshot", pool+"/data@baseline")
	childGUID := strings.TrimSpace(must("/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", pool+"/data"))
	rootGUID := strings.TrimSpace(must("/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", pool))
	before := must("/sbin/zfs", "list", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,guid,quota,refquota,compression,mountpoint,canmount,readonly,mounted,encryption", pool)
	source, err := NewPoolPropertySource()
	if err != nil {
		t.Fatal(err)
	}
	o, err := source.Collect(ctx, guid, pool)
	if err != nil {
		t.Fatal("actual collector failed (values withheld)", err)
	}
	check("complete", o.Complete && !o.Eligible && len(o.Filesystems) == 2 && len(o.States) == 2 && len(o.Identities) == 3)
	independent := strings.Fields(must("/sbin/zpool", "list", "-H", "-p", "-o", "guid,load_guid", pool))
	check("pool_identity", len(independent) == 2 && o.Pool.GUID == independent[0] && o.Pool.LoadGUID == independent[1])
	check("dataset_identity", o.Filesystems[0].GUID == rootGUID && o.Filesystems[1].GUID == childGUID && o.Filesystems[1].ParentGUID == rootGUID)
	check("property_values", o.Filesystems[1].Properties.Quota.Effective == "1048576" && o.Filesystems[1].Properties.Compression.Effective == "lz4")
	check("inheritance", o.States[1].Mountpoint.Effective == "none" && o.States[1].Mountpoint.Source == storage.Inherited && o.States[1].Mountpoint.SourceGUID == rootGUID)
	check("snapshot_owner", o.Identities[2].Type == storage.SnapshotType && o.Identities[2].OwnerGUID == childGUID)
	check("features", len(o.Features.Values) == 39 || len(o.Features.Values) == 47)
	check("topology", matchesCollectedMirror(o.Topology, []string{"/dev/" + devices[0].unit, "/dev/" + devices[1].unit}))
	p := storage.Policy{Revision: 1, PoolGUID: guid, PoolName: pool, Scopes: []storage.Scope{{GUID: childGUID, Name: pool + "/data", ParentGUID: rootGUID, Properties: []storage.Property{storage.RefQuota}}}}
	_, err = storage.Build(time.Now(), "fixture-session", p, o, storage.Intent{FilesystemGUID: childGUID, Property: storage.RefQuota, Value: "1"})
	check("eligibility_denied", errors.Is(err, storage.ErrUnavailable))
	// Test-only metadata provisioning: these bytes are not product enrollment or
	// a publication/durability test. The actual source reads through the root lease.
	// The state lease refuses writable ancestors, including sticky /tmp.
	// Provision this test-owned private directory under the guest's trusted root.
	statePath, err := os.MkdirTemp("/root", "horvium-property-policy-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(statePath); err != nil {
			t.Error("owned policy fixture cleanup", err)
		}
		if _, err := os.Lstat(statePath); !errors.Is(err, os.ErrNotExist) {
			t.Error("owned policy fixture remains")
		}
	})
	state, err := storage.OpenStateDirectory(statePath)
	if err != nil {
		t.Fatal("fixture state lease", err)
	}
	t.Cleanup(func() {
		if err := state.Close(); err != nil {
			t.Error("state lease cleanup", err)
		}
	})
	bound, err := NewPropertyPolicySource(state)
	if err != nil {
		t.Fatal(err)
	}
	zero := func(p storage.Policy, o storage.Observation, err error) bool {
		return err != nil && reflect.DeepEqual(p, storage.Policy{}) && reflect.DeepEqual(o, storage.Observation{})
	}
	missingP, missingO, missingErr := bound.Collect(ctx, guid)
	check("policy_missing_denied", zero(missingP, missingO, missingErr))
	writePolicy := func(policy storage.Policy) {
		data, err := storage.EncodeOwnershipRecord(policy)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(statePath, "ownership-"+guid+".json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writePolicy(p)
	boundP, boundO, boundErr := bound.Collect(ctx, guid)
	_, planErr := storage.Build(time.Now(), "fixture-session", boundP, boundO, storage.Intent{FilesystemGUID: childGUID, Property: storage.RefQuota, Value: "1"})
	check("policy_bound", boundErr == nil && reflect.DeepEqual(boundP, p) && reflect.DeepEqual(boundO, o) && errors.Is(planErr, storage.ErrUnavailable))
	systemSource, err := NewSystemUseSource()
	if err != nil {
		t.Fatal(err)
	}
	unsupportedUse, unsupportedErr := systemSource.Collect(ctx)
	check("unsupported_md_denied", errors.Is(unsupportedErr, storage.ErrUnavailable) && reflect.DeepEqual(unsupportedUse, storage.PoolSystemUse{}))
	// Introduce an actual fixture-owned policy change between the native passes
	// and the second root read, using the package-private collection seam.
	reads := 0
	changing := &PropertyPolicySource{pool: source, ownership: func(g string) (storage.Policy, error) {
		reads++
		if reads == 2 {
			changed := p
			changed.Revision++
			writePolicy(changed)
		}
		return state.ReadOwnership(g)
	}}
	badP, badO, badErr := changing.Collect(ctx, guid)
	check("policy_change_denied", zero(badP, badO, badErr) && errors.Is(badErr, storage.ErrDrift))
	badPolicy := p
	badPolicy.Scopes = append([]storage.Scope(nil), p.Scopes...)
	badPolicy.Scopes[0].GUID = "1"
	writePolicy(badPolicy)
	badP, badO, badErr = bound.Collect(ctx, guid)
	check("policy_scope_denied", zero(badP, badO, badErr) && errors.Is(badErr, storage.ErrDenied))
	writePolicy(p)
	denied, e := source.Collect(ctx, "1", pool)
	check("wrong_guid_denied", errors.Is(e, storage.ErrDrift) && reflect.DeepEqual(denied, storage.Observation{}))
	after := must("/sbin/zfs", "list", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,guid,quota,refquota,compression,mountpoint,canmount,readonly,mounted,encryption", pool)
	sentinelBytes, e := os.ReadFile(sentinel)
	check("preservation", after == before && e == nil && string(sentinelBytes) == "independent fixture sentinel")
	must("/sbin/zfs", "rename", pool+"/data", pool+"/renamed")
	renamed, e := source.Collect(ctx, guid, pool)
	check("rename_fresh", e == nil && renamed.Filesystems[1].GUID == childGUID && renamed.Filesystems[1].Name == pool+"/renamed" && o.Filesystems[1].Name == pool+"/data")
	badP, badO, badErr = bound.Collect(ctx, guid)
	check("policy_rename_denied", zero(badP, badO, badErr) && errors.Is(badErr, storage.ErrDenied))
	if err = state.Close(); err != nil {
		t.Fatal(err)
	}
	badP, badO, badErr = bound.Collect(ctx, guid)
	check("policy_closed_denied", zero(badP, badO, badErr) && errors.Is(badErr, storage.ErrStateDirectory))
	must("/sbin/zfs", "set", "readonly=on", pool+"/renamed")
	denied, e = source.Collect(ctx, guid, pool)
	check("readonly_denied", e != nil && reflect.DeepEqual(denied, storage.Observation{}))
	must("/sbin/zfs", "set", "readonly=off", pool+"/renamed")
	must("/sbin/zfs", "create", "-V", "1m", pool+"/volume")
	denied, e = source.Collect(ctx, guid, pool)
	check("volume_denied", e != nil && reflect.DeepEqual(denied, storage.Observation{}))
	for _, d := range devices {
		actual := strings.TrimSpace(must("/sbin/mdconfig", "-l", "-n", "-f", d.path))
		if actual != strings.TrimPrefix(d.unit, "md") {
			t.Fatal("owned backing drift")
		}
	}
	check("backing_preserved", true)
}
