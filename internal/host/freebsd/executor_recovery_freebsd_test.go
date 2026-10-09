// SPDX-License-Identifier: BSD-3-Clause
package freebsd

// Authorized disposable research only. No runtime factory/clearance uses these
// observers, cuts or fixture provisioning. The two parents straddle a real reboot.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"github.com/gonz0w/horvium/internal/storage"
)

const recoveryRoot = "/root/work/.tools/executor-recovery-fixture"
const recoveryObserver = "/root/work/.tools/executor-observer"

type recoveryProcess struct {
	BootSec     int64  `json:"boot_sec"`
	BootUsec    int64  `json:"boot_usec"`
	Found       bool   `json:"found"`
	PID         int    `json:"pid"`
	StructBytes int    `json:"struct_bytes"`
	PPID        int    `json:"ppid"`
	PGID        int    `json:"pgid"`
	SID         int    `json:"sid"`
	StartSec    int64  `json:"start_sec"`
	StartUsec   int64  `json:"start_usec"`
	UID         uint32 `json:"uid"`
	RUID        uint32 `json:"ruid"`
	State       int    `json:"state"`
	Zombie      bool   `json:"zombie"`
	Stopped     bool   `json:"stopped"`
	ReaperError int    `json:"reaper_error"`
	Reaper      int    `json:"reaper"`
	Children    uint32 `json:"children"`
	Descendants uint32 `json:"descendants"`
	ReaperFlags uint32 `json:"reaper_flags"`
}
type recoveryRecord struct {
	Mode, Directory, JobID, JobHash, PolicyHash string
	LockInode                                   uint64
	State                                       storage.JobState
	Owner                                       recoveryProcess
}
type recoveryFixture struct {
	Policy                     storage.Policy
	MetadataName, MetadataGUID string
	DataDevices                []string
	MetadataDevice             string
	Boot                       recoveryProcess
	Records                    []recoveryRecord
}

func recoveryOptIn(t testing.TB) {
	t.Helper()
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PROPERTY_VM") != "1" || os.Getenv("HORVIUM_RESEARCH_VM") != "1" {
		t.Skip("requires owned recovery research guest")
	}
	if os.Getuid() != 0 || os.Geteuid() != 0 {
		t.Fatal("owned research root only")
	}
}
func recoveryCommand(t testing.TB, path string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := execute(ctx, path, args, 1<<20)
	if err != nil {
		t.Fatal("owned research command failed", path, err)
	}
	return string(out.data)
}
func recoveryJSON(t testing.TB, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}
func recoveryRead(t testing.TB, path string, value any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 1<<20 {
		t.Fatal("bounded owned research record", err)
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		t.Fatal(err)
	}
}
func recoveryInspect(t testing.TB, pid int) recoveryProcess {
	t.Helper()
	var p recoveryProcess
	out := recoveryCommand(t, recoveryObserver, strconv.Itoa(pid))
	if json.Unmarshal([]byte(out), &p) != nil || p.PID != pid || p.BootSec <= 0 || p.BootUsec < 0 || p.BootUsec >= 1000000 ||
		p.Found && (p.StructBytes <= 0 || p.StartSec <= 0 || p.StartUsec < 0 || p.StartUsec >= 1000000) {
		t.Fatal("actual process observer facts")
	}
	return p
}
func recoveryHash(t testing.TB, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func recoverySame(a, b recoveryProcess) bool {
	return a.Found && b.Found && a.PID == b.PID && a.BootSec == b.BootSec && a.BootUsec == b.BootUsec && a.StartSec == b.StartSec && a.StartUsec == b.StartUsec && a.UID == b.UID && a.RUID == b.RUID
}

func recoverySignal(t testing.TB, original recoveryProcess, signal syscall.Signal) {
	t.Helper()
	current := recoveryInspect(t, original.PID)
	if !current.Found || current.Zombie {
		return
	}
	if !recoverySame(original, current) {
		t.Error("owned process identity changed; signal withheld")
		return
	}
	if err := syscall.Kill(original.PID, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
		t.Error("owned process signal", err)
	}
}

func recoveryStop(t testing.TB, original recoveryProcess) {
	t.Helper()
	recoverySignal(t, original, syscall.SIGSTOP)
	until := time.Now().Add(2 * time.Second)
	for time.Now().Before(until) {
		current := recoveryInspect(t, original.PID)
		if !recoverySame(original, current) {
			t.Fatal("original process vanished before stop witness")
		}
		if current.Stopped {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("original process did not reach stopped state")
}
func recoveryWait(t testing.TB, path string, duration time.Duration) {
	t.Helper()
	until := time.Now().Add(duration)
	for time.Now().Before(until) {
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("owned research boundary not reached")
}
func recoveryBlock(t testing.TB) {
	t.Helper()
	// Parent kills this owning process. No worker/product retry or fake effect.
	time.Sleep(90 * time.Second)
	t.Fatal("owned cut was not delivered")
}

// The child holds the real state lease around the unchanged guard and the existing
// escaped descendant fixture. This is containment research, not a ZFS operation.
func TestNativeRecoveryOwnerChild(t *testing.T) {
	a := flag.Args()
	if len(a) != 2 {
		return
	}
	recoveryOptIn(t)
	s, err := storage.OpenStateDirectory(a[0])
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, err = executeNative(context.Background(), os.Args[0], []string{"-test.run=^TestProbeTreeProcess$", "--", "escaped-child", filepath.Join(a[1], "child"), filepath.Join(a[1], "probe")}, 128)
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// Test-only external clearance and wrapper witnesses; all metadata/native methods
// are the actual original coordinator/backend/claim/control implementations.
func TestNativeRecoveryBackendChild(t *testing.T) {
	a := flag.Args()
	if len(a) != 4 {
		return
	}
	recoveryOptIn(t)
	mode, path, guid, witness := a[0], a[1], a[2], a[3]
	s, err := storage.OpenStateDirectory(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	policy, err := s.ReadOwnership(guid)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	b, err := newPropertyBackend(s, guid, func(c context.Context) error { return c.Err() })
	if err != nil {
		t.Fatal(err)
	}
	coordinator := storage.NewPropertyCoordinator(b)
	plan, err := coordinator.Review(ctx, "owned-recovery-spike", storage.Intent{FilesystemGUID: policy.Scopes[0].GUID, Property: storage.Quota, Value: "2097152"})
	if err != nil {
		t.Fatal(err)
	}
	job, duplicate, err := coordinator.Submit(ctx, "owned-recovery-spike", plan.ID, strings.Repeat("9", 64))
	if err != nil || duplicate {
		t.Fatal("actual child admission", err)
	}
	if mode == "queued-cut" {
		recoveryJSON(t, witness, job.Job().ID)
		recoveryBlock(t)
	}
	worker, err := coordinator.ClaimPropertyExecution(ctx, "owned-recovery-spike", job.Job().ID)
	if err != nil {
		t.Fatal(err)
	}
	if mode != "normal-completion" {
		factory := b.source.control
		b.source.control = func(j storage.PropertyJob, live func(context.Context) error) (*propertyControl, error) {
			control, err := factory(j, live)
			if err != nil {
				return nil, err
			}
			native := control.run
			control.run = func(c context.Context, i storage.PropertyIntent) error {
				if mode == "effect-cut" {
					if err := native(c, i); err != nil {
						return err
					}
				}
				if mode != "effect-cut" && mode != "applying-cut" {
					return storage.ErrDenied
				}
				recoveryJSON(t, witness, job.Job().ID)
				recoveryBlock(t)
				return storage.ErrDenied
			}
			return control, nil
		}
	}
	result, err := worker.Run(ctx, job.Job().ID)
	if err != nil || backendState(result) != storage.JobSucceeded {
		t.Fatal("actual child completion", err)
	}
	recoveryJSON(t, witness, job.Job().ID)
}

func recoveryStart(t *testing.T, name string, args ...string) (*exec.Cmd, <-chan error, *bool) {
	t.Helper()
	argv := append([]string{"-test.run=^" + name + "$", "--"}, args...)
	c := exec.Command(os.Args[0], argv...)
	c.Env = os.Environ()
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	log, err := os.OpenFile(filepath.Join(recoveryRoot, "child-"+filepath.Base(args[0])+".log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	c.Stdout, c.Stderr = log, log
	if err := c.Start(); err != nil {
		_ = log.Close()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	waited := new(bool)
	go func() { done <- c.Wait(); _ = log.Close() }()
	t.Cleanup(func() {
		if !*waited {
			_ = c.Process.Kill()
			select {
			case <-done:
				*waited = true
			case <-time.After(3 * time.Second):
				t.Error("owned recovery child reap")
			}
		}
	})
	return c, done, waited
}
func recoveryJoin(t *testing.T, c *exec.Cmd, done <-chan error, waited *bool, killed bool) {
	t.Helper()
	if killed {
		if err := c.Process.Kill(); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case err := <-done:
		*waited = true
		if killed {
			var exit *exec.ExitError
			if !errors.As(err, &exit) || !exit.ProcessState.Sys().(syscall.WaitStatus).Signaled() {
				t.Fatal("actual child was not killed", err)
			}
		} else if err != nil {
			t.Fatal("owned child failed; private log retained", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("owned child exit deadline")
	}
}
func recoveryContainment(t *testing.T, mode, anchor string) {
	t.Helper()
	dir := filepath.Join(anchor, mode)
	if os.Mkdir(dir, 0700) != nil {
		t.Fatal("owned guard lease directory")
	}
	work := filepath.Join(recoveryRoot, mode)
	if os.Mkdir(work, 0700) != nil {
		t.Fatal("owned guard witnesses")
	}
	c, done, waited := recoveryStart(t, "TestNativeRecoveryOwnerChild", dir, work)
	probe := fixturePID(t, filepath.Join(work, "probe"))
	child := fixturePID(t, filepath.Join(work, "child"))
	p := recoveryInspect(t, probe)
	leaf := recoveryInspect(t, child)
	owner := recoveryInspect(t, c.Process.Pid)
	guard := recoveryInspect(t, p.PPID)
	if !p.Found || !leaf.Found || !owner.Found || !guard.Found || guard.PPID != owner.PID ||
		guard.ReaperError != 0 || guard.Reaper != guard.PID || guard.Descendants < 2 || leaf.PGID == p.PGID ||
		p.UID != 0 || p.RUID != 0 || leaf.UID != 0 || leaf.RUID != 0 {
		t.Fatal("actual owned guard/escaped-child lineage")
	}
	t.Cleanup(func() {
		recoverySignal(t, guard, syscall.SIGCONT)
		for _, original := range []recoveryProcess{p, leaf, guard} {
			recoverySignal(t, original, syscall.SIGKILL)
			assertProbeStopped(t, original.PID)
		}
	})
	if other, err := storage.OpenStateDirectory(dir); err == nil {
		_ = other.Close()
		t.Fatal("live owner did not retain singleton")
	}
	if mode == "stopped-guard" {
		recoveryStop(t, guard)
	}
	if mode == "guard-first" {
		recoveryStop(t, owner)
		recoverySignal(t, guard, syscall.SIGKILL)
	}
	recoveryJoin(t, c, done, waited, true)
	s, err := storage.OpenStateDirectory(dir)
	if err != nil {
		t.Fatal("singleton reacquisition after owner death", err)
	}
	defer s.Close()
	a, b := recoveryInspect(t, probe), recoveryInspect(t, child)
	// Observe each original descendant independently. Stopping a process group
	// can affect which members survive owner death; do not invent that outcome.
	for _, pair := range [][2]recoveryProcess{{a, p}, {b, leaf}} {
		if pair[0].Found && !pair[0].Zombie && !recoverySame(pair[0], pair[1]) {
			t.Fatal("process observation cannot attribute original descendant")
		}
	}
	probeLive, childLive := recoverySame(a, p) && !a.Zombie, recoverySame(b, leaf) && !b.Zombie
	if mode == "guard-first" && !probeLive && !childLive {
		t.Fatal("guard-first counterexample did not reach original live descendant")
	}
	if probeLive || childLive {
		t.Log("singleton reacquired while an original descendant remained alive; no clearance")
	}
	if mode == "stopped-guard" {
		recoverySignal(t, guard, syscall.SIGCONT)
	}
	if mode == "guard-first" {
		recoverySignal(t, p, syscall.SIGKILL)
		recoverySignal(t, leaf, syscall.SIGKILL)
	}
	for _, pid := range []int{probe, child, guard.PID} {
		assertProbeStopped(t, pid)
	}
	if !t.Failed() {
		data, err := json.Marshal(struct {
			Mode  string `json:"mode"`
			Lock  bool   `json:"singleton_reacquired"`
			Probe bool   `json:"original_probe_live"`
			Child bool   `json:"original_child_live"`
			Clean bool   `json:"cleanup_verified"`
		}{mode, true, probeLive, childLive, true})
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("EXECUTOR_CONTAINMENT " + string(data))
	}
}

func recoverySetup(t *testing.T) recoveryFixture {
	t.Helper()
	if err := os.Mkdir(recoveryRoot, 0700); err != nil {
		t.Fatal("exclusive owned recovery fixture", err)
	}
	graph, err := inventory.ParseGEOM([]byte(recoveryCommand(t, "/sbin/sysctl", "-b", "kern.geom.confxml")), inventory.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	system, err := NewSystemUseSource()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	u, err := system.Collect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	used := map[string]bool{}
	rootDisks := 0
	for _, m := range u.System().Mounts {
		for _, d := range m.Disks {
			used[d.Name] = true
			if m.Target == "/" {
				rootDisks++
			}
		}
	}
	for _, s := range u.System().Swap {
		for _, d := range s.Disks {
			used[d.Name] = true
		}
	}
	if rootDisks == 0 {
		t.Fatal("independently resolved root backing required")
	}
	var first, second, metadata inventory.Provider
	count := 0
	for _, p := range graph.Providers {
		if p.Kind != inventory.DiskCandidate || p.MediaSize != 64<<20 {
			continue
		}
		count++
		if p.Class != "DISK" || used[p.Name] || !strings.HasPrefix(p.Name, "vtbd") || strings.ContainsAny(p.Name, "/ \\\t\r\n") {
			t.Fatal("foreign/system fixture disk")
		}
		serial := ""
		for _, id := range p.Identifiers {
			if id.Kind == "serial" {
				serial = id.Value
			}
		}
		switch serial {
		case "HVM-UNIQUE-512":
			first = p
		case "HVM-UNIQUE-4K":
			second = p
		case "HVM-DUPLICATE":
			if metadata.ID == "" {
				metadata = p
			}
		}
	}
	if count != 5 || first.ID == "" || second.ID == "" || metadata.ID == "" || first.Identity != inventory.IdentityUnique || second.Identity != inventory.IdentityUnique {
		t.Fatal("declared owned device set")
	}
	dataPool := "horvium_recovery_data"
	metaPool := "horvium_recovery_metadata"
	for _, pool := range []string{dataPool, metaPool} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := execute(ctx, "/sbin/zpool", []string{"list", "-H", pool}, 4096)
		cancel()
		if err == nil {
			t.Fatal("fixture pool already exists")
		}
	}
	for _, p := range []inventory.Provider{first, second, metadata} {
		fields := strings.Fields(recoveryCommand(t, "/usr/sbin/diskinfo", "/dev/"+p.Name))
		if len(fields) < 3 || fields[2] != "67108864" {
			t.Fatal("actual fixture geometry")
		}
	}
	recoveryCommand(t, "/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", "-o", "ashift=12", dataPool, "mirror", "/dev/"+first.Name, "/dev/"+second.Name)
	anchor := filepath.Join(recoveryRoot, "metadata")
	if os.Mkdir(anchor, 0700) != nil {
		t.Fatal("owned metadata mountpoint")
	}
	recoveryCommand(t, "/sbin/zpool", "create", "-m", anchor, "-o", "cachefile=none", "-o", "ashift=12", metaPool, "/dev/"+metadata.Name)
	if os.Chmod(anchor, 0700) != nil {
		t.Fatal("private metadata backing")
	}
	guid := strings.TrimSpace(recoveryCommand(t, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", dataPool))
	poolSource, err := NewPoolPropertySource()
	if err != nil {
		t.Fatal(err)
	}
	o, err := poolSource.Collect(ctx, guid, dataPool)
	if err != nil {
		t.Fatal(err)
	}
	policy := storage.Policy{Revision: 1, PoolGUID: guid, PoolName: dataPool, Scopes: []storage.Scope{{GUID: o.Filesystems[0].GUID, Name: dataPool, Properties: []storage.Property{storage.Quota}}}}
	for _, v := range o.Topology.Nodes {
		if v.Kind != "leaf" {
			continue
		}
		unit := strings.TrimPrefix(v.Address, "/dev/")
		info, err := inventory.ParseDiskObservation([]byte(recoveryCommand(t, "/sbin/sysctl", "-b", "kern.geom.confxml")), []byte(recoveryCommand(t, "/usr/sbin/diskinfo", "/dev/"+unit)), []byte(recoveryCommand(t, "/usr/sbin/diskinfo", "-s", "/dev/"+unit)), inventory.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		var serial, source string
		for _, id := range info.Provider.Identifiers {
			if id.Kind == "serial" {
				serial, source = id.Value, id.Source
			}
		}
		policy.Members = append(policy.Members, storage.PoolMember{VdevGUID: v.GUID, Serial: serial, SerialSource: source, MediaSize: info.Provider.MediaSize, SectorSize: info.Provider.SectorSize, StripeSize: info.StripeSize, StripeOffset: info.StripeOffset})
	}
	if _, err := storage.EncodeOwnershipRecord(policy); err != nil {
		t.Fatal(err)
	}
	// Root dataset remains unmounted; the foreign sentinel is on the boot fixture,
	// outside both explicit mutable pools. Hash it across reboot/teardown.
	if os.WriteFile(filepath.Join(recoveryRoot, "sentinel"), []byte("owned recovery sentinel\n"), 0600) != nil {
		t.Fatal("sentinel")
	}
	return recoveryFixture{Policy: policy, MetadataName: metaPool, MetadataGUID: strings.TrimSpace(recoveryCommand(t, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", metaPool)), DataDevices: []string{first.Name, second.Name}, MetadataDevice: metadata.Name, Boot: recoveryInspect(t, os.Getpid())}
}

func recoveryCut(t *testing.T, f recoveryFixture, mode string) recoveryRecord {
	t.Helper()
	path := filepath.Join(recoveryRoot, "metadata", mode)
	if os.Mkdir(path, 0700) != nil {
		t.Fatal("owned cut directory")
	}
	data, err := storage.EncodeOwnershipRecord(f.Policy)
	if err != nil || os.WriteFile(filepath.Join(path, "ownership-"+f.Policy.PoolGUID+".json"), data, 0600) != nil {
		t.Fatal("fixture-only enrolled policy", err)
	}
	witness := filepath.Join(recoveryRoot, mode+"-witness")
	c, done, waited := recoveryStart(t, "TestNativeRecoveryBackendChild", mode, path, f.Policy.PoolGUID, witness)
	recoveryWait(t, witness, 90*time.Second)
	owner := recoveryInspect(t, c.Process.Pid)
	if mode != "normal-completion" && !owner.Found {
		t.Fatal("original live cut owner")
	}
	recoveryJoin(t, c, done, waited, mode != "normal-completion")
	var id string
	recoveryRead(t, witness, &id)
	s, err := storage.OpenStateDirectory(path)
	if err != nil {
		t.Fatal("real original lock reacquisition", err)
	}
	defer s.Close()
	job, err := s.ReadPropertyJob(id)
	if err != nil {
		t.Fatal(err)
	}
	want := storage.JobApplying
	if mode == "queued-cut" {
		want = storage.JobQueued
	}
	if mode == "normal-completion" {
		want = storage.JobSucceeded
	}
	if backendState(job) != want {
		t.Fatal("actual durable boundary differs", backendState(job))
	}
	view, err := s.InspectJobReservations()
	if err != nil || len(view.Pools()) != 1 || len(view.Pools()[0].Jobs) != 1 {
		t.Fatal("retained pool hold", err)
	}
	if claim, err := s.ClaimPropertyState(id); claim != nil || err == nil {
		t.Fatal("reopened metadata reconstructed original authority")
	}
	if mode == "effect-cut" || mode == "normal-completion" {
		if strings.TrimSpace(recoveryCommand(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value,source", "quota", f.Policy.PoolName)) != "2097152\tlocal" {
			t.Fatal("independent real quota effect")
		}
	} else if strings.TrimSpace(recoveryCommand(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value,source", "quota", f.Policy.PoolName)) != "0\tdefault" {
		t.Fatal("pre-effect cut changed native quota")
	}
	lock, err := os.Stat(filepath.Join(path, "writer.lock"))
	if err != nil {
		t.Fatal(err)
	}
	return recoveryRecord{Mode: mode, Directory: mode, JobID: id, JobHash: recoveryHash(t, filepath.Join(path, "job-"+id+".json")), PolicyHash: recoveryHash(t, filepath.Join(path, "ownership-"+f.Policy.PoolGUID+".json")), LockInode: lock.Sys().(*syscall.Stat_t).Ino, State: want, Owner: owner}
}

func TestNativeExecutorRecoveryBefore(t *testing.T) {
	recoveryOptIn(t)
	var f recoveryFixture
	if !t.Run("fixture-preflight", func(t *testing.T) { f = recoverySetup(t) }) {
		return
	}
	t.Run("process-observer", func(t *testing.T) {
		a := recoveryInspect(t, os.Getpid())
		b := recoveryInspect(t, os.Getpid())
		if !a.Found || a.StartSec != b.StartSec || a.StartUsec != b.StartUsec || a.UID != 0 || a.RUID != 0 {
			t.Fatal("same original process identity")
		}
		if p := recoveryInspect(t, 2147483647); p.Found {
			t.Fatal("missing fixture process appeared")
		}
		t.Logf("native kinfo ABI bytes=%d; boot/start usec observed; no clearance", a.StructBytes)
	})
	for _, mode := range []string{"owner-death", "stopped-guard", "guard-first"} {
		t.Run(mode, func(t *testing.T) { recoveryContainment(t, mode, filepath.Join(recoveryRoot, "metadata")) })
	}
	for _, mode := range []string{"queued-cut", "applying-cut", "normal-completion", "effect-cut"} {
		if !t.Run(mode, func(t *testing.T) { f.Records = append(f.Records, recoveryCut(t, f, mode)) }) {
			return
		}
		if mode == "normal-completion" {
			recoveryCommand(t, "/sbin/zfs", "inherit", "-S", "quota", f.Policy.PoolName)
		}
	}
	t.Run("retained-for-reboot", func(t *testing.T) {
		if len(f.Records) != 4 {
			t.Fatal("missing actual durable cuts")
		}
		recoveryJSON(t, filepath.Join(recoveryRoot, "fixture.json"), f)
		recoveryCommand(t, "/bin/sync")
		t.Log("active property-command cut not exercised; effect-cut witnesses completed control before outcome publication")
	})
	if !t.Failed() {
		fmt.Println("EXECUTOR_RECOVERY_RESULT before=pass")
	}
}

func TestNativeExecutorRecoveryAfter(t *testing.T) {
	recoveryOptIn(t)
	var f recoveryFixture
	recoveryRead(t, filepath.Join(recoveryRoot, "fixture.json"), &f)
	if !t.Run("new-boot", func(t *testing.T) {
		now := recoveryInspect(t, os.Getpid())
		if now.BootSec == f.Boot.BootSec && now.BootUsec == f.Boot.BootUsec {
			t.Fatal("same boot cannot establish reboot")
		}
		for _, r := range f.Records {
			if r.Owner.Found {
				current := recoveryInspect(t, r.Owner.PID)
				if recoverySame(r.Owner, current) {
					t.Fatal("original process survived observed reboot")
				}
				t.Logf("old owner PID exists on new boot=%t; no original execution authority", current.Found)
			}
		}
	}) {
		return
	}
	if !t.Run("exact-fixture-import", func(t *testing.T) {
		// Only trusted fixed fixture names/GUIDs and directly declared virtio disks;
		// no production import or guessing after missing GUID acknowledgement.
		if f.Policy.PoolName != "horvium_recovery_data" || f.MetadataName != "horvium_recovery_metadata" || len(f.DataDevices) != 2 || len(f.Records) != 4 || storage.ValidatePoolSelection(f.Policy.PoolGUID, f.Policy.PoolName) != nil || storage.ValidatePoolSelection(f.MetadataGUID, f.MetadataName) != nil {
			t.Fatal("owned reboot descriptor")
		}
		for _, unit := range append(append([]string{}, f.DataDevices...), f.MetadataDevice) {
			if !strings.HasPrefix(unit, "vtbd") || strings.ContainsAny(unit, "/ \\\t\r\n") {
				t.Fatal("owned reboot device")
			}
			fields := strings.Fields(recoveryCommand(t, "/usr/sbin/diskinfo", "/dev/"+unit))
			if len(fields) < 3 || fields[2] != "67108864" {
				t.Fatal("reboot fixture geometry changed")
			}
		}
		for _, pool := range []struct{ name, guid, unit string }{{f.Policy.PoolName, f.Policy.PoolGUID, f.DataDevices[0]}, {f.MetadataName, f.MetadataGUID, f.MetadataDevice}} {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			out, err := execute(ctx, "/sbin/zpool", []string{"get", "-H", "-p", "-o", "value", "guid", pool.name}, 4096)
			cancel()
			if err != nil {
				recoveryCommand(t, "/sbin/zpool", "import", "-N", "-d", "/dev/"+pool.unit, pool.guid)
			} else if strings.TrimSpace(string(out.data)) != pool.guid {
				t.Fatal("foreign imported fixture pool")
			}
			if strings.TrimSpace(recoveryCommand(t, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool.name)) != pool.guid {
				t.Fatal("fixture import GUID")
			}
		}
		if strings.TrimSpace(recoveryCommand(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value", "mounted", f.MetadataName)) != "yes" {
			recoveryCommand(t, "/sbin/zfs", "mount", f.MetadataName)
		}
	}) {
		return
	}
	t.Run("unchanged-ledger", func(t *testing.T) {
		for _, r := range f.Records {
			path := filepath.Join(recoveryRoot, "metadata", r.Directory)
			if recoveryHash(t, filepath.Join(path, "job-"+r.JobID+".json")) != r.JobHash || recoveryHash(t, filepath.Join(path, "ownership-"+f.Policy.PoolGUID+".json")) != r.PolicyHash {
				t.Fatal("reboot changed actual durable metadata")
			}
			lock, err := os.Stat(filepath.Join(path, "writer.lock"))
			if err != nil || lock.Sys().(*syscall.Stat_t).Ino != r.LockInode {
				t.Fatal("reboot replaced original lock object", err)
			}
		}
	})
	t.Run("holds-and-no-authority", func(t *testing.T) {
		for _, r := range f.Records {
			path := filepath.Join(recoveryRoot, "metadata", r.Directory)
			s, err := storage.OpenStateDirectory(path)
			if err != nil {
				t.Fatal(err)
			}
			func() {
				defer s.Close()
				view, err := s.InspectJobReservations()
				if err != nil || len(view.Pools()) != 1 || len(view.Pools()[0].Jobs) != 1 {
					t.Fatal("reboot lost hold", err)
				}
				page, err := view.RestartPage("", 64)
				if err != nil || len(page) != 1 || page[0].Last.State != r.State {
					t.Fatal("reboot lost durable event", err)
				}
				if r.State != storage.JobSucceeded && page[0].State != storage.JobReconcileRequired {
					t.Fatal("interrupted job lost reconciliation")
				}
				if c, err := s.ClaimPropertyState(r.JobID); c != nil || err == nil {
					t.Fatal("reboot restored volatile authority")
				}
				source, err := NewPropertyStateSource(s)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if _, _, err := source.Collect(ctx, f.Policy.PoolGUID); err == nil {
					t.Fatal("retained hold granted new eligibility")
				}
			}()
		}
	})
	t.Run("independent-properties", func(t *testing.T) {
		if strings.TrimSpace(recoveryCommand(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value,source", "quota", f.Policy.PoolName)) != "2097152\tlocal" {
			t.Fatal("actual effect changed across reboot")
		}
		if strings.TrimSpace(recoveryCommand(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", f.Policy.PoolName)) != f.Policy.Scopes[0].GUID {
			t.Fatal("filesystem GUID changed")
		}
	})
	t.Run("fixture-preservation", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(recoveryRoot, "sentinel"))
		if err != nil || string(data) != "owned recovery sentinel\n" {
			t.Fatal("sentinel changed")
		}
		for _, r := range f.Records {
			if r.Mode != "normal-completion" && r.Owner.PID <= 1 {
				t.Fatal("missing original owner observation")
			}
		}
	})
	t.Run("fixture-teardown", func(t *testing.T) {
		for _, pool := range []struct{ name, guid string }{{f.Policy.PoolName, f.Policy.PoolGUID}, {f.MetadataName, f.MetadataGUID}} {
			if strings.TrimSpace(recoveryCommand(t, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool.name)) != pool.guid {
				t.Fatal("teardown identity drift")
			}
			recoveryCommand(t, "/sbin/zpool", "destroy", pool.name)
		}
		if os.RemoveAll(recoveryRoot) != nil {
			t.Fatal("owned fixture removal")
		}
		if _, err := os.Lstat(recoveryRoot); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("owned fixture remains")
		}
	})
	if !t.Failed() {
		fmt.Println("EXECUTOR_RECOVERY_RESULT after=pass")
	}
}
