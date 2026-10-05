// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"
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
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PROPERTY_VM") != "1" {
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
	// Installed profile checks are fixture assertions only, not a product file reader.
	profilePath := "/usr/share/zfs/compatibility.d/openzfs-2.2"
	for _, path := range []string{"/", "/usr", "/usr/share", "/usr/share/zfs", "/usr/share/zfs/compatibility.d", profilePath} {
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0022 != 0 {
			t.Fatal("installed profile path unsafe (values withheld)")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 {
			t.Fatal("installed profile ownership unsafe")
		}
		if path == profilePath {
			if !info.Mode().IsRegular() || info.Size() != 584 {
				t.Fatal("installed profile shape unavailable")
			}
		} else if !info.IsDir() {
			t.Fatal("installed profile ancestor unavailable")
		}
	}
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal("installed profile read failed")
	}
	profileNames, err := CommonFeatureProfile(profileData)
	if err != nil {
		t.Fatal("installed common profile differs")
	}
	version := strings.TrimSpace(string(must("/bin/freebsd-version", "-u")))
	format := ""
	if strings.HasPrefix(version, "14.5-RELEASE") {
		format = "freebsd-14.5"
	} else if strings.HasPrefix(version, "15.1-RELEASE") {
		format = "freebsd-15.1"
	} else {
		t.Fatal("unqualified fixture release")
	}
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal(err)
	}
	pool := "horvium_m2_" + hex.EncodeToString(nonce[:])
	dir := t.TempDir()
	type ownedMD struct{ unit, backing string }
	var ownedMDs []ownedMD
	declaredPool := false
	exportedPool := false
	ownedGUID := ""
	importArgs := func(readonly string) []string {
		args := []string{"import", "-N"}
		for _, md := range ownedMDs {
			args = append(args, "-d", "/dev/"+md.unit)
		}
		return append(args, "-o", "cachefile=none", "-o", "readonly="+readonly, ownedGUID)
	}
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if declaredPool {
			if exportedPool {
				if _, err := command(cleanupCtx, "/sbin/zpool", "list", "-H", pool); err != nil {
					if !validGUID(ownedGUID) {
						t.Error("exported fixture identity unavailable; discard guest")
						return
					}
					if _, err := command(cleanupCtx, "/sbin/zpool", importArgs("off")...); err != nil {
						t.Error("owned exported pool cleanup import failed; discard guest")
						return
					}
				}
			}
			if _, err := command(cleanupCtx, "/sbin/zpool", "list", "-H", pool); err == nil {
				nativeGUID, err := command(cleanupCtx, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)
				if err != nil || !validGUID(ownedGUID) || strings.TrimSpace(string(nativeGUID)) != ownedGUID {
					t.Error("cleanup pool GUID differs; discard guest")
					return
				}
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
		for _, md := range ownedMDs {
			actual, err := command(cleanupCtx, "/sbin/mdconfig", "-l", "-n", "-f", md.backing)
			if err != nil || strings.TrimSpace(string(actual)) != strings.TrimPrefix(md.unit, "md") {
				t.Error("owned backing/provider changed; discard guest")
				return
			}
			if _, err := command(cleanupCtx, "/sbin/mdconfig", "-d", "-u", strings.TrimPrefix(md.unit, "md")); err != nil {
				t.Error("owned md detach failed")
				return
			}
			if _, err := command(cleanupCtx, "/sbin/mdconfig", "-l", "-n", "-f", md.backing); err == nil {
				t.Error("owned md remains attached")
				return
			}
		}
		if !t.Failed() {
			t.Log("PROPERTY_OBSERVATION_RESULT default=pass local=pass inherited=pass received=pass shadowed=pass no_limit=pass rename=pass name_reuse=pass foreign_preservation=pass eligibility_denied=pass teardown=pass identity_binding=pass complete_filesystems=pass snapshot_owner=pass bookmark_owner=pass replicated_guid=pass snapshot_rename=pass snapshot_reuse=pass volume_denial=pass stable_enumeration=pass state_binding=pass mount_sources=pass readonly_denial=pass encrypted_loaded_denial=pass encrypted_unloaded_denial=pass encryption_root_binding=pass state_drift=pass state_preservation=pass pool_identity_binding=pass pool_source_metadata=pass pool_bootfs_denial=pass pool_readonly_denial=pass import_incarnation=pass reimport_drift=pass pool_preservation=pass pool_name_reuse=pass feature_binding=pass feature_sources=pass feature_preservation=pass installed_profile=pass common_feature_set=pass feature_drift=pass topology_binding=pass mirror_layout=pass raidz2_layout=pass independent_vdevs=pass degraded_denial=pass unsupported_layout=pass topology_drift=pass topology_preservation=pass vdev_guid_reuse=pass")
		}
	})
	createMD := func() string {
		backing := filepath.Join(dir, "owned-"+strconv.Itoa(len(ownedMDs))+".img")
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
		ownedMDs = append(ownedMDs, ownedMD{unit, backing})
		if got := strings.TrimSpace(string(must("/sbin/mdconfig", "-l", "-n", "-f", backing))); got != strings.TrimPrefix(unit, "md") {
			t.Fatal("fixture backing correspondence failed")
		}
		return unit
	}
	unit := createMD()
	secondUnit := createMD()
	thirdUnit, fourthUnit := createMD(), createMD()
	backing := ownedMDs[0].backing
	if got := strings.TrimSpace(string(must("/sbin/mdconfig", "-l", "-n", "-f", backing))); got != strings.TrimPrefix(unit, "md") {
		t.Fatal("fixture backing correspondence failed")
	}
	if _, err := command(ctx, "/sbin/zpool", "list", "-H", pool); err == nil {
		t.Fatal("fixture namespace unexpectedly exists")
	}
	declaredPool = true
	must("/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", pool, "mirror", "/dev/"+unit, "/dev/"+secondUnit, "mirror", "/dev/"+thirdUnit, "/dev/"+fourthUnit)
	ownedGUID = strings.TrimSpace(string(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)))
	if !validGUID(ownedGUID) {
		t.Fatal("owned pool identity invalid")
	}
	poolState := func() PoolState {
		rows := must("/sbin/zpool", "get", "-H", "-p", "-o", "name,property,value,source", "guid,load_guid,health,readonly,bootfs,altroot,cachefile,compatibility,multihost", pool)
		state, err := ParsePoolState(rows, pool)
		if err != nil {
			t.Fatal("native pool state denied (values withheld):", err)
		}
		return state
	}
	topology := func() PoolTopology {
		before := poolState()
		if before.GUID != ownedGUID {
			t.Fatal("topology pool identity differs")
		}
		capture := func() PoolTopology {
			guids := must("/sbin/zpool", "status", "-g", "-p", pool)
			paths := must("/sbin/zpool", "status", "-L", "-P", "-p", pool)
			x, err := ParsePoolTopology(guids, paths, pool, ownedGUID)
			if err != nil {
				t.Fatal("native topology denied (values withheld):", err)
			}
			return x
		}
		a, b := capture(), capture()
		if err := VerifyTopologyStability(a, b); err != nil {
			t.Fatal("native topology changed during capture", err)
		}
		if after := poolState(); after != before {
			t.Fatal("pool policy/incarnation changed during status capture")
		}
		return a
	}
	independentTopology := func(x PoolTopology) {
		for _, guidView := range []bool{true, false} {
			args := []string{"list", "-H", "-p", "-v", "-o", "name,health"}
			if guidView {
				args = append(args, "-g")
			} else {
				args = append(args, "-L", "-P")
			}
			args = append(args, pool)
			observed := make(map[string]string)
			for _, line := range strings.Split(strings.TrimSpace(string(must("/sbin/zpool", args...))), "\n") {
				fields := strings.Fields(line)
				if len(fields) != 2 || observed[fields[0]] != "" {
					t.Fatal("independent vdev table shape invalid")
				}
				observed[fields[0]] = fields[1]
			}
			if observed[pool] != x.Health || len(observed) != len(x.Nodes)+1 {
				t.Fatal("independent vdev membership differs")
			}
			for _, node := range x.Nodes {
				key := node.Address
				if guidView {
					key = node.GUID
				}
				if observed[key] != node.Health {
					t.Fatal("independent vdev identity/state differs")
				}
			}
		}
	}
	initialTopology := topology()
	if err := VerifyPoolTopology(ownedGUID, pool, initialTopology); err != nil {
		t.Fatal(err)
	}
	independentTopology(initialTopology)
	if len(initialTopology.Nodes) != 6 {
		t.Fatal("owned mirror membership incomplete")
	}
	features := func() PoolFeatures {
		rows := must("/sbin/zpool", "get", "-H", "-p", "-o", "name,property,value,source", "all", pool)
		f, err := ParsePoolFeatures(rows, pool, format)
		if err != nil {
			t.Fatal("native features denied (values withheld):", err)
		}
		if err := VerifyPoolFeatures(ownedGUID, pool, f); err != nil {
			t.Fatal("native feature completeness denied:", err)
		}
		return f
	}
	initialFeatures := features()
	for _, v := range initialFeatures.Values {
		independent := strings.TrimSpace(string(must("/sbin/zpool", "get", "-H", "-p", "-o", "value,source", v.Name, pool)))
		if independent != v.State+"\t"+string(v.Source) {
			t.Fatal("independent feature value/source differs")
		}
	}
	initialPool := poolState()
	if err := VerifyPoolState(ownedGUID, pool, initialPool); err != nil {
		t.Fatal(err)
	}
	independentPool := strings.Split(strings.TrimSpace(string(must("/sbin/zpool", "list", "-H", "-p", "-o", "name,guid,load_guid,health,readonly,bootfs,altroot,cachefile,compatibility,multihost", pool))), "\t")
	if len(independentPool) != 10 || independentPool[0] != pool || independentPool[1] != initialPool.GUID || independentPool[2] != initialPool.LoadGUID {
		t.Fatal("independent pool identity disagreement")
	}
	for n, v := range poolValues(initialPool) {
		if v.Effective != independentPool[n+3] {
			t.Fatal("independent pool value disagreement")
		}
	}
	poolSources := []Source{initialPool.GUIDSource, initialPool.LoadGUIDSource}
	for _, v := range poolValues(initialPool) {
		poolSources = append(poolSources, v.Source)
	}
	for n, property := range poolProperties {
		source := strings.TrimSpace(string(must("/sbin/zpool", "get", "-H", "-p", "-o", "source", property, pool)))
		if source != string(poolSources[n]) {
			t.Fatal("independent pool source disagreement")
		}
	}
	if initialPool.CacheFile.Effective != "none" {
		t.Fatal("fixture pool cache policy changed")
	}

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
	// Independently enumerate all native dataset types around the property read.
	identities := func() []DatasetIdentity {
		rows := must("/sbin/zfs", "list", "-H", "-p", "-r", "-t", "filesystem,volume,snapshot,bookmark", "-o", "name,guid,type", pool)
		ids, err := ParseIdentities(rows, pool)
		if err != nil {
			t.Fatal("native identity table denied (values withheld):", err)
		}
		return ids
	}
	findID := func(ids []DatasetIdentity, suffix string) DatasetIdentity {
		for _, id := range ids {
			if id.Name == pool+suffix {
				return id
			}
		}
		t.Fatal("owned identity absent")
		return DatasetIdentity{}
	}
	must("/sbin/zfs", "bookmark", pool+"/source@seed", pool+"/source#seed")
	first := identities()
	files = parse()
	second := identities()
	if err := VerifyIdentityStability(pool, first, second); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFilesystemIdentities(pool, files, second); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFilesystemIdentities(pool, files[:len(files)-1], second); !errors.Is(err, ErrDrift) {
		t.Fatal("omitted filesystem accepted", err)
	}
	changed := append([]DatasetIdentity(nil), second...)
	for n := range changed {
		if changed[n].Name == pool+"/data" {
			changed[n].GUID = "1"
		}
	}
	if err := VerifyFilesystemIdentities(pool, files, changed); err == nil {
		t.Fatal("reused filesystem GUID accepted")
	}
	source := findID(second, "/source@seed")
	replica := findID(second, "/received@seed")
	bookmark := findID(second, "/source#seed")
	if source.Type != SnapshotType || replica.Type != SnapshotType || bookmark.Type != BookmarkType || source.GUID != replica.GUID || source.GUID != bookmark.GUID || source.OwnerGUID != find(files, "/source").GUID || replica.OwnerGUID != received.GUID || bookmark.OwnerGUID != source.OwnerGUID {
		t.Fatal("native version GUID/owner mismatch")
	}
	for _, id := range second {
		guid := strings.TrimSpace(string(must("/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", id.Name)))
		if guid != id.GUID {
			t.Fatal("independent identity GUID disagreement")
		}
	}
	must("/sbin/zfs", "rename", pool+"/source@seed", pool+"/source@renamed")
	renamed := identities()
	if findID(renamed, "/source@renamed").GUID != source.GUID {
		t.Fatal("snapshot rename changed GUID")
	}
	if err := VerifyIdentityStability(pool, second, renamed); !errors.Is(err, ErrDrift) {
		t.Fatal("native rename drift ignored", err)
	}
	must("/sbin/zfs", "snapshot", pool+"/source@seed")
	if findID(identities(), "/source@seed").GUID == source.GUID {
		t.Fatal("snapshot name reuse confused GUID")
	}
	must("/sbin/zfs", "create", "-s", "-V", "8388608", "-o", "volmode=none", pool+"/volume")
	withVolume := identities()
	if findID(withVolume, "/volume").Type != VolumeType {
		t.Fatal("native volume omitted")
	}
	if err := VerifyFilesystemIdentities(pool, parse(), withVolume); !errors.Is(err, ErrUnavailable) {
		t.Fatal("unrepresented volume impact accepted", err)
	}
	must("/sbin/zfs", "destroy", pool+"/volume")
	if err := VerifyFilesystemIdentities(pool, parse(), identities()); err != nil {
		t.Fatal(err)
	}
	// Mount and encryption observations are independently correlated, never
	// treated as the remaining pool/topology/system eligibility proof.
	states := func() []FilesystemState {
		rows := must("/sbin/zfs", "get", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,property,value,received,source", "guid,mountpoint,canmount,readonly,mounted,encryption,encryptionroot,keystatus", pool)
		result, err := ParseFilesystemStates(rows, pool)
		if err != nil {
			t.Fatal("native state table denied (values withheld):", err)
		}
		return result
	}
	findState := func(rows []FilesystemState, suffix string) FilesystemState {
		for _, s := range rows {
			if s.Name == pool+suffix {
				return s
			}
		}
		t.Fatal("owned filesystem state absent")
		return FilesystemState{}
	}
	baseline := states()
	if err := VerifyFilesystemStates(pool, identities(), baseline); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFilesystemStates(pool, identities(), baseline[:len(baseline)-1]); !errors.Is(err, ErrDrift) {
		t.Fatal("whole state row omitted", err)
	}
	independentState := must("/sbin/zfs", "list", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,guid,mountpoint,canmount,readonly,mounted,encryption,encryptionroot,keystatus", pool)
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(independentState)), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) != 9 {
			t.Fatal("independent state shape invalid")
		}
		s := findState(baseline, strings.TrimPrefix(fields[0], pool))
		if s.GUID != fields[1] {
			t.Fatal("independent state GUID disagreement")
		}
		for n, v := range stateValues(s) {
			if v.Effective != fields[n+2] {
				t.Fatal("independent state value disagreement")
			}
		}
		count++
	}
	if count != len(baseline) {
		t.Fatal("independent state coverage differs")
	}
	foreign := findState(baseline, "/foreign")
	if foreign.Mountpoint.Effective != "none" || foreign.Mountpoint.Source != Local || foreign.Mounted.Effective != "no" || foreign.Encryption.Effective != "off" {
		t.Fatal("native mount/source mismatch")
	}
	must("/sbin/zfs", "set", "readonly=on", pool+"/foreign")
	if err := VerifyFilesystemStates(pool, identities(), states()); !errors.Is(err, ErrUnavailable) {
		t.Fatal("readonly fixture qualified", err)
	}
	must("/sbin/zfs", "inherit", "readonly", pool+"/foreign")
	must("/sbin/zfs", "set", "canmount=noauto", pool+"/data")
	changedState := findState(states(), "/data")
	if changedState.CanMount.Effective != "noauto" || changedState.CanMount.Source != Local || changedState.CanMount.SourceGUID != data.GUID {
		t.Fatal("canmount local source mismatch")
	}
	policy := Policy{Revision: 1, PoolGUID: ownedGUID, PoolName: pool, Scopes: []Scope{{GUID: data.GUID, Name: data.Name, ParentGUID: data.ParentGUID, Properties: []Property{RefQuota}}}}
	observation := Observation{Complete: true, Eligible: true, PoolGUID: ownedGUID, PoolName: pool, Filesystems: parse(), Identities: identities(), States: baseline, Pool: poolState(), Features: features(), Topology: topology()}
	plan, err := Build(time.Now(), "fixture-session", policy, observation, Intent{FilesystemGUID: data.GUID, Property: RefQuota, Value: "none"})
	if err != nil {
		t.Fatal("synthetic policy review failed", err)
	}
	observation.States = states()
	if err := plan.Validate(time.Now(), "fixture-session", policy, observation); !errors.Is(err, ErrDrift) {
		t.Fatal("native state change ignored", err)
	}
	must("/sbin/zfs", "set", "canmount=on", pool+"/data")
	// Fresh random encryption material belongs only to this disposable fixture.
	// Never log it, retain it as evidence or grant any product key operation.
	keyPath := filepath.Join(dir, "owned.key")
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, key[:], 0600); err != nil {
		t.Fatal(err)
	}
	must("/sbin/zfs", "create", "-o", "mountpoint=none", "-o", "encryption=aes-256-gcm", "-o", "keyformat=raw", "-o", "keylocation=file://"+keyPath, pool+"/encrypted")
	must("/sbin/zfs", "create", "-o", "mountpoint=none", pool+"/encrypted/child")
	encrypted := states()
	cryptoRoot := findState(encrypted, "/encrypted")
	cryptoChild := findState(encrypted, "/encrypted/child")
	if cryptoRoot.EncryptionRootGUID != cryptoRoot.GUID || cryptoChild.EncryptionRootGUID != cryptoRoot.GUID || cryptoChild.Encryption.Effective != "aes-256-gcm" || cryptoChild.KeyStatus.Effective != "available" {
		t.Fatal("encryption root/key binding mismatch")
	}
	if err := VerifyFilesystemStates(pool, identities(), encrypted); !errors.Is(err, ErrUnavailable) {
		t.Fatal("loaded encryption fixture qualified", err)
	}
	must("/sbin/zfs", "unload-key", pool+"/encrypted")
	if findState(states(), "/encrypted/child").KeyStatus.Effective != "unavailable" {
		t.Fatal("unloaded key state lost")
	}
	if err := VerifyFilesystemStates(pool, identities(), states()); !errors.Is(err, ErrUnavailable) {
		t.Fatal("unloaded encryption fixture qualified", err)
	}
	must("/sbin/zfs", "destroy", pool+"/encrypted/child")
	must("/sbin/zfs", "destroy", pool+"/encrypted")
	if err := os.Remove(keyPath); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFilesystemStates(pool, identities(), states()); err != nil {
		t.Fatal(err)
	}
	must("/sbin/zpool", "set", "bootfs="+pool+"/data", pool)
	bootPool := poolState()
	if bootPool.BootFS.Effective != pool+"/data" {
		t.Fatal("native boot selector absent")
	}
	if err := VerifyPoolState(ownedGUID, pool, bootPool); !errors.Is(err, ErrUnavailable) {
		t.Fatal("boot-selected fixture qualified", err)
	}
	must("/sbin/zpool", "set", "bootfs=", pool)
	if err := VerifyPoolState(ownedGUID, pool, poolState()); err != nil {
		t.Fatal(err)
	}
	poolBefore := poolState()
	before := must("/sbin/zfs", "get", "-H", "-p", "-r", "-o", "name,property,value,received,source", "guid,quota,refquota,compression,mountpoint,canmount,readonly", pool)
	mounts := must("/sbin/zfs", "list", "-H", "-r", "-o", "name,mounted,mountpoint", pool)
	stateBefore := states()
	stableBefore := identities()
	if err := VerifyFilesystemIdentities(pool, parse(), stableBefore); err != nil {
		t.Fatal(err)
	}
	if err := VerifyIdentityStability(pool, stableBefore, identities()); err != nil {
		t.Fatal(err)
	}
	if after := poolState(); after != poolBefore {
		t.Fatal("pool reader changed policy or incarnation")
	}
	if after := states(); !reflect.DeepEqual(stateBefore, after) {
		t.Fatal("state reader changed native constraints")
	}
	if after := must("/sbin/zfs", "get", "-H", "-p", "-r", "-o", "name,property,value,received,source", "guid,quota,refquota,compression,mountpoint,canmount,readonly", pool); !bytes.Equal(before, after) {
		t.Fatal("read-only projection changed native properties or foreign sentinel")
	}
	if after := must("/sbin/zfs", "list", "-H", "-r", "-o", "name,mounted,mountpoint", pool); !bytes.Equal(mounts, after) {
		t.Fatal("read-only projection changed mounts")
	}
	featureRead := features()
	if !reflect.DeepEqual(featureRead, features()) {
		t.Fatal("feature observations changed native state")
	}
	// Parsing provides no authority, even when every projected property is valid.
	if _, err := Build(time.Now(), "fixture-session", Policy{Revision: 1, PoolGUID: "1", PoolName: pool}, Observation{Complete: true, PoolGUID: "1", PoolName: pool, Filesystems: files, Identities: identities(), States: states(), Pool: poolState(), Features: features(), Topology: topology()}, Intent{FilesystemGUID: root.GUID, Property: Quota, Value: "none"}); err != ErrUnavailable {
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
	// Test-only export/reimport is confined to this known GUID/provider. No force,
	// mounts or product recovery/import capability follows from fixture preparation.
	latestFiles := parse()
	latestRoot := find(latestFiles, "")
	poolObservation := Observation{Complete: true, Eligible: true, PoolGUID: ownedGUID, PoolName: pool, Pool: poolState(), Filesystems: latestFiles, Identities: identities(), States: states(), Features: features(), Topology: topology()}
	poolPolicy := Policy{Revision: 1, PoolGUID: ownedGUID, PoolName: pool, Scopes: []Scope{{GUID: latestRoot.GUID, Name: pool, Properties: []Property{RefQuota}}}}
	poolPlan, err := Build(time.Now(), "fixture-session", poolPolicy, poolObservation, Intent{FilesystemGUID: latestRoot.GUID, Property: RefQuota, Value: "none"})
	if err != nil {
		t.Fatal("synthetic pool review failed", err)
	}
	previousLoad := poolObservation.Pool.LoadGUID
	beforeImportProperties := projection()
	beforeImportStates := states()
	beforeImportIDs := identities()
	importOwned := func(readonly string) {
		must("/sbin/zpool", "export", pool)
		exportedPool = true
		must("/sbin/zpool", importArgs(readonly)...)
		exportedPool = false
	}
	importOwned("off")
	reimported := poolState()
	if reimported.GUID != ownedGUID || reimported.LoadGUID == previousLoad || reimported.CacheFile.Effective != "none" {
		t.Fatal("pool incarnation not bound to import")
	}
	if !bytes.Equal(beforeImportProperties, projection()) || !reflect.DeepEqual(beforeImportStates, states()) {
		t.Fatal("unmounted fixture import changed filesystem policy")
	}
	if err := VerifyIdentityStability(pool, beforeImportIDs, identities()); err != nil {
		t.Fatal(err)
	}
	poolObservation.Pool = reimported
	if err := poolPlan.Validate(time.Now(), "fixture-session", poolPolicy, poolObservation); !errors.Is(err, ErrDrift) {
		t.Fatal("pool reimport drift ignored", err)
	}
	importOwned("on")
	readonlyPool := poolState()
	if readonlyPool.Readonly.Effective != "on" || readonlyPool.GUID != ownedGUID {
		t.Fatal("readonly import projection incorrect")
	}
	if err := VerifyPoolState(ownedGUID, pool, readonlyPool); !errors.Is(err, ErrUnavailable) {
		t.Fatal("readonly pool qualified", err)
	}
	importOwned("off")
	if err := VerifyPoolState(ownedGUID, pool, poolState()); err != nil {
		t.Fatal(err)
	}
	readTopology := topology()
	if err := VerifyTopologyStability(readTopology, topology()); err != nil {
		t.Fatal("status reads changed membership", err)
	}
	attachedUnit := createMD()
	freshFiles := parse()
	freshRoot := find(freshFiles, "")
	nativeObs := Observation{Complete: true, Eligible: true, PoolGUID: ownedGUID, PoolName: pool, Pool: poolState(), Features: features(), Topology: readTopology, Filesystems: freshFiles, Identities: identities(), States: states()}
	nativePolicy := Policy{Revision: 1, PoolGUID: ownedGUID, PoolName: pool, Scopes: []Scope{{GUID: freshRoot.GUID, Name: pool, Properties: []Property{RefQuota}}}}
	nativePlan, err := Build(time.Now(), "fixture-session", nativePolicy, nativeObs, Intent{FilesystemGUID: freshRoot.GUID, Property: RefQuota, Value: "none"})
	if err != nil {
		t.Fatal(err)
	}
	// External-root fixture mutation tests drift; product attach/offline/online is not added.
	must("/sbin/zpool", "attach", pool, "/dev/"+unit, "/dev/"+attachedUnit)
	must("/sbin/zpool", "wait", "-t", "resilver", pool)
	attached := topology()
	if attached.PoolGUID != readTopology.PoolGUID || len(attached.Nodes) != 7 {
		t.Fatal("native same-pool membership change absent")
	}
	if err := VerifyPoolTopology(ownedGUID, pool, attached); !errors.Is(err, ErrUnavailable) {
		t.Fatal("changed width/scan qualified", err)
	}
	if err := VerifyTopologyStability(readTopology, attached); !errors.Is(err, ErrDrift) {
		t.Fatal("native membership drift ignored", err)
	}
	nativeObs.Topology = attached
	nativeObs.Pool = poolState()
	nativeObs.Features = features()
	nativeObs.Filesystems = parse()
	nativeObs.Identities = identities()
	nativeObs.States = states()
	if err := nativePlan.Validate(time.Now(), "fixture-session", nativePolicy, nativeObs); !errors.Is(err, ErrDrift) {
		t.Fatal("native review drift ignored", err)
	}
	must("/sbin/zpool", "offline", pool, "/dev/"+unit)
	degraded := topology()
	offline := false
	for _, node := range degraded.Nodes {
		if node.Address == "/dev/"+unit && node.Health == "OFFLINE" {
			offline = true
		}
	}
	if degraded.Health != "DEGRADED" || !offline {
		t.Fatal("degraded native state lost")
	}
	if err := VerifyPoolTopology(ownedGUID, pool, degraded); !errors.Is(err, ErrUnavailable) {
		t.Fatal("degraded native topology qualified", err)
	}
	must("/sbin/zpool", "online", pool, "/dev/"+unit)
	must("/sbin/zpool", "wait", "-t", "resilver", pool)
	destroyOwned := func() {
		if current := strings.TrimSpace(string(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool))); current != ownedGUID {
			t.Fatal("owned pool changed before fixture destroy")
		}
		must("/sbin/zpool", "destroy", pool)
	}
	destroyOwned()
	must("/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", pool, "raidz2", "/dev/"+unit, "/dev/"+secondUnit, "/dev/"+thirdUnit, "/dev/"+fourthUnit)
	ownedGUID = strings.TrimSpace(string(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)))
	raidz := topology()
	independentTopology(raidz)
	if len(raidz.Nodes) != 5 {
		t.Fatal("owned RAIDZ2 membership incomplete")
	}
	if err := VerifyPoolTopology(ownedGUID, pool, raidz); err != nil {
		t.Fatal("native RAIDZ2 layout denied", err)
	}
	destroyOwned()
	must("/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", pool, "/dev/"+unit, "/dev/"+secondUnit)
	ownedGUID = strings.TrimSpace(string(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)))
	stripe := topology()
	independentTopology(stripe)
	if len(stripe.Nodes) != 2 {
		t.Fatal("owned stripe membership incomplete")
	}
	if err := VerifyPoolTopology(ownedGUID, pool, stripe); !errors.Is(err, ErrUnavailable) {
		t.Fatal("unsupported native stripe qualified", err)
	}
	oldGUID := ownedGUID
	destroyOwned()
	must("/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", "-o", "compatibility="+profilePath, pool, "mirror", "/dev/"+unit, "/dev/"+secondUnit)
	ownedGUID = strings.TrimSpace(string(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)))
	reused := poolState()
	if !validGUID(ownedGUID) || ownedGUID == oldGUID || reused.Name != pool {
		t.Fatal("pool name reuse confused identity")
	}
	if err := VerifyPoolState(oldGUID, pool, reused); !errors.Is(err, ErrDrift) {
		t.Fatal("old pool identity accepted for reused name", err)
	}

	reusedTopology := topology()
	independentTopology(reusedTopology)
	oldNodes := make(map[string]bool)
	for _, node := range initialTopology.Nodes {
		oldNodes[node.GUID] = true
	}
	for _, node := range reusedTopology.Nodes {
		if oldNodes[node.GUID] {
			t.Fatal("recreated vdev retained old GUID")
		}
	}
	common := features()
	allowed := make(map[string]bool)
	for _, name := range profileNames {
		allowed["feature@"+name] = true
	}
	for _, v := range common.Values {
		if allowed[v.Name] {
			if v.State != "enabled" && v.State != "active" {
				t.Fatal("common profile feature missing")
			}
		} else if v.State != "disabled" {
			t.Fatal("feature outside common profile enabled")
		}
	}
	if poolState().Compatibility.Effective != profilePath {
		t.Fatal("fixture compatibility profile not retained")
	}
	// Create only owned versions to activate a known feature and compare the fresh
	// native observations under counterfactual trusted model inputs. No eligibility
	// collector or product mutation is exercised by this fixture setup.
	commonFiles := parse()
	commonRoot := find(commonFiles, "")
	commonObs := Observation{Complete: true, Eligible: true, PoolGUID: ownedGUID, PoolName: pool, Pool: poolState(), Features: common, Topology: topology(), Filesystems: commonFiles, Identities: identities(), States: states()}
	commonPolicy := Policy{Revision: 1, PoolGUID: ownedGUID, PoolName: pool, Scopes: []Scope{{GUID: commonRoot.GUID, Name: pool, Properties: []Property{RefQuota}}}}
	commonPlan, err := Build(time.Now(), "fixture-session", commonPolicy, commonObs, Intent{FilesystemGUID: commonRoot.GUID, Property: RefQuota, Value: "none"})
	if err != nil {
		t.Fatal(err)
	}
	stateOf := func(f PoolFeatures, name string) string {
		for _, v := range f.Values {
			if v.Name == "feature@"+name {
				return v.State
			}
		}
		t.Fatal("fixture feature absent")
		return ""
	}
	if stateOf(common, "bookmarks") != "enabled" {
		t.Fatal("bookmarks fixture baseline unexpected")
	}
	must("/sbin/zfs", "snapshot", pool+"@feature_probe")
	must("/sbin/zfs", "bookmark", pool+"@feature_probe", pool+"#feature_probe")
	commonObs.Features = features()
	commonObs.Identities = identities()
	commonObs.Filesystems = parse()
	commonObs.States = states()
	if stateOf(commonObs.Features, "bookmarks") != "active" {
		t.Fatal("native feature activation not observed")
	}
	if err := commonPlan.Validate(time.Now(), "fixture-session", commonPolicy, commonObs); !errors.Is(err, ErrDrift) {
		t.Fatal("native feature/version drift ignored", err)
	}
	must("/sbin/zfs", "destroy", pool+"#feature_probe")
	must("/sbin/zfs", "destroy", pool+"@feature_probe")

}
