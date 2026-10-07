// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/host/freebsd"
	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
)

// Only the existing opt-in disposable parent calls this fixture. Every native
// write below provisions/changes its own pool, files and providers. No operator
// storage, native publisher, executable admission or recovery is involved.
func nativeStorageAssessmentCommand(t *testing.T, ctx context.Context, binary string) {
	t.Helper()
	path, err := os.MkdirTemp("/root", "storage-assessment-")
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(path, "ledger")
	if os.Mkdir(statePath, 0700) != nil {
		t.Fatal("owned ledger")
	}
	var nonce [8]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		t.Fatal(err)
	}
	pool := "horvium_assessment_" + hex.EncodeToString(nonce[:])
	guid := ""
	created := false
	cleaned := false
	type provider struct{ unit, path string }
	var devices []provider
	command := func(c context.Context, name string, args ...string) (string, error) {
		cmd := exec.CommandContext(c, name, args...)
		cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
		out, e := cmd.Output()
		return string(out), e
	}
	must := func(name string, args ...string) string {
		t.Helper()
		out, e := command(ctx, name, args...)
		if e != nil {
			t.Fatal("owned fixture command failed (values withheld)", name)
		}
		return out
	}
	cleanup := func() bool {
		if cleaned {
			return true
		}
		c, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if created {
			got, e := command(c, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)
			if e != nil || guid == "" || strings.TrimSpace(got) != guid {
				return false
			}
			if _, e = command(c, "/sbin/zpool", "destroy", pool); e != nil {
				return false
			}
			created = false
			if _, e = command(c, "/sbin/zpool", "list", "-H", pool); e == nil {
				return false
			}
		}
		for _, d := range devices {
			got, e := command(c, "/sbin/mdconfig", "-l", "-n", "-f", d.path)
			if e != nil || strings.TrimSpace(got) != strings.TrimPrefix(d.unit, "md") {
				return false
			}
			if _, e = command(c, "/sbin/mdconfig", "-d", "-u", strings.TrimPrefix(d.unit, "md")); e != nil {
				return false
			}
			if _, e = command(c, "/sbin/mdconfig", "-l", "-n", "-f", d.path); e == nil {
				return false
			}
		}
		devices = nil
		if os.RemoveAll(path) != nil {
			return false
		}
		_, e := os.Lstat(path)
		cleaned = errors.Is(e, os.ErrNotExist)
		return cleaned
	}
	t.Cleanup(func() {
		if !cleanup() {
			t.Error("owned assessment teardown failed; discard guest")
		}
	})
	for _, name := range []string{"first.img", "second.img"} {
		disk := filepath.Join(path, name)
		f, e := os.OpenFile(disk, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			t.Fatal(e)
		}
		if e = f.Truncate(256 << 20); e != nil {
			f.Close()
			t.Fatal(e)
		}
		if f.Close() != nil {
			t.Fatal("owned disk close")
		}
		unit := strings.TrimSpace(must("/sbin/mdconfig", "-a", "-t", "vnode", "-f", disk))
		if !regexp.MustCompile(`^md[0-9]+$`).MatchString(unit) {
			t.Fatal("unexpected owned provider; discard guest")
		}
		devices = append(devices, provider{unit, disk})
	}
	if _, e := command(ctx, "/sbin/zpool", "list", "-H", pool); e == nil {
		t.Fatal("fixture pool name already exists")
	}
	must("/sbin/zpool", "create", "-m", "none", "-o", "cachefile=none", pool, "mirror", "/dev/"+devices[0].unit, "/dev/"+devices[1].unit)
	created = true
	guid = strings.TrimSpace(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool))
	target := pool + "/data"
	must("/sbin/zfs", "create", "-o", "quota=1048576", "-o", "compression=lz4", target)
	raw := func() string {
		return must("/sbin/zfs", "get", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,property,value,received,source", "guid,quota,refquota,compression,mountpoint,canmount,readonly,mounted,encryption,encryptionroot,keystatus", pool)
	}
	originalNative := raw()
	source, err := freebsd.NewPoolPropertySource()
	if err != nil {
		t.Fatal(err)
	}
	observed, err := source.Collect(ctx, guid, pool)
	if err != nil || !observed.Complete || observed.Eligible {
		t.Fatal("actual fixture collection", err)
	}
	p := storage.Policy{Revision: 1, PoolGUID: guid, PoolName: pool}
	targetGUID := ""
	for _, f := range observed.Filesystems {
		p.Scopes = append(p.Scopes, storage.Scope{GUID: f.GUID, Name: f.Name, ParentGUID: f.ParentGUID, Properties: []storage.Property{storage.Quota, storage.RefQuota, storage.Compression}})
		if f.Name == target {
			targetGUID = f.GUID
		}
	}
	records := make(map[string][]byte)
	write := func(name string, data []byte) {
		t.Helper()
		if os.WriteFile(filepath.Join(statePath, name), data, 0600) != nil {
			t.Fatal("owned record setup")
		}
		records[name] = append([]byte(nil), data...)
	}
	policyBytes, err := storage.EncodeOwnershipRecord(p)
	if err != nil {
		t.Fatal(err)
	}
	write("ownership-"+guid+".json", policyBytes)
	capture := observed
	capture.Eligible = true // Test-only immutable capture; native product observations stay false.
	jobFor := func(id string, property storage.Property, value string, terminal bool) storage.PropertyJob {
		t.Helper()
		at := time.Now()
		plan, e := storage.Build(at, "command-fixture", p, capture, storage.Intent{FilesystemGUID: targetGUID, Property: property, Value: value})
		if e != nil {
			t.Fatal(e)
		}
		intent, e := plan.CapturePropertyIntent(at, "command-fixture", p, capture)
		if e != nil {
			t.Fatal(e)
		}
		r := intent.Review()
		j := storage.Job{ID: id, Intent: storage.JobIntent{PlanID: r.ID, SubmissionID: id, PolicyRevision: p.Revision, PoolGUID: guid, Change: r.Intent}, History: []storage.JobTransition{{Revision: 1, At: uint64(at.UnixMilli()), State: storage.JobQueued, Reason: "admitted"}}}
		bound, e := storage.BindPropertyJob(j, intent)
		if e != nil {
			t.Fatal(e)
		}
		state, reason := storage.JobApplying, "launch-intent"
		if terminal {
			state, reason = storage.JobCancelled, "operator-cancelled"
		}
		bound, e = storage.TransitionPropertyJob(bound, state, uint64(at.Add(time.Millisecond).UnixMilli()), reason, "")
		if e != nil {
			t.Fatal(e)
		}
		return bound
	}
	id := strings.Repeat("a", 64)
	job := jobFor(id, storage.Quota, "2097152", false)
	jobBytes, err := storage.EncodePropertyJob(job)
	if err != nil {
		t.Fatal(err)
	}
	write("job-"+id+".json", jobBytes)
	bothID := strings.Repeat("b", 64)
	both := jobFor(bothID, storage.Compression, "lz4", true)
	bothBytes, err := storage.EncodePropertyJob(both)
	if err != nil {
		t.Fatal(err)
	}
	write("job-"+bothID+".json", bothBytes)
	argsFor := func(jobID string) string {
		args, _ := json.Marshal([]string{"storage-assess", "-state-dir", statePath, "-job-id", jobID})
		return string(args)
	}
	process := func(jobID string, uid uint32) *exec.Cmd {
		cmd := exec.CommandContext(ctx, binary, "-test.run=^TestStorageJobStatusCommandProcess$")
		cmd.Env = append(os.Environ(), "HORVIUM_STATUS_COMMAND_PROCESS=1", "HORVIUM_STATUS_COMMAND_ARGS="+argsFor(jobID))
		if uid != 0 {
			cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: uid, Groups: []uint32{}}}
		}
		return cmd
	}
	invoke := func(jobID string, uid uint32) ([]byte, []byte, error) {
		cmd := process(jobID, uid)
		var out, errout bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errout
		e := cmd.Run()
		return out.Bytes(), errout.Bytes(), e
	}
	success := func(t *testing.T, jobID string, baseline, effect bool) []byte {
		t.Helper()
		out, errout, e := invoke(jobID, 0)
		var got storagePropertyAssessmentStatus
		if e != nil || len(errout) != 0 || json.Unmarshal(out, &got) != nil || got.Job.ID != jobID || got.Job.PoolGUID != guid || got.BaselineMatches != baseline || got.ExpectedEffectMatches != effect || got.RecoveryVerified || got.ExecutorVerified || !got.HoldsRetained {
			t.Fatal("actual command facts", e)
		}
		if jobID == id && (got.Job.LastDurable.State != storage.JobApplying || got.Job.RestartState != storage.JobReconcileRequired || got.Job.RestartReason != "executor-unverified") {
			t.Fatal("historical/restart labels changed")
		}
		return out
	}
	denied := func(t *testing.T, jobID string, uid uint32) {
		t.Helper()
		out, errout, e := invoke(jobID, uid)
		if e == nil || len(out) != 0 || string(errout) != errStorageAssessment.Error()+"\n" {
			t.Fatal("native command denial exposed details", e)
		}
	}
	var first []byte
	t.Run("baseline", func(t *testing.T) { first = success(t, id, true, false) })
	lockBefore, err := os.Stat(filepath.Join(statePath, "writer.lock"))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("expected", func(t *testing.T) {
		must("/sbin/zfs", "set", "quota=2097152", target)
		success(t, id, false, true)
		must("/sbin/zfs", "set", "quota=1048576", target)
	})
	t.Run("both-terminal", func(t *testing.T) {
		out := success(t, bothID, true, true)
		var got storagePropertyAssessmentStatus
		_ = json.Unmarshal(out, &got)
		if got.Job.LastDurable.State != storage.JobCancelled || got.Job.RestartState != storage.JobCancelled {
			t.Fatal("terminal metadata relabeled")
		}
	})
	t.Run("neither", func(t *testing.T) {
		must("/sbin/zfs", "set", "refquota=1048576", target)
		success(t, id, false, false)
		must("/sbin/zfs", "inherit", "-S", "refquota", target)
	})
	t.Run("unknown-job", func(t *testing.T) { denied(t, strings.Repeat("f", 64), 0) })
	t.Run("legacy-job", func(t *testing.T) {
		data, e := storage.EncodeJobRecord(job.Job())
		if e != nil {
			t.Fatal(e)
		}
		if os.WriteFile(filepath.Join(statePath, "job-"+id+".json"), data, 0600) != nil {
			t.Fatal("legacy setup")
		}
		denied(t, id, 0)
		write("job-"+id+".json", jobBytes)
	})
	t.Run("unsafe-record", func(t *testing.T) {
		name := filepath.Join(statePath, "job-"+id+".json")
		if os.Chmod(name, 0644) != nil {
			t.Fatal("unsafe setup")
		}
		denied(t, id, 0)
		if os.Chmod(name, 0600) != nil {
			t.Fatal("mode restoration")
		}
	})
	t.Run("orphan", func(t *testing.T) {
		name := filepath.Join(statePath, ".job-"+strings.Repeat("d", 32)+".tmp")
		if os.WriteFile(name, []byte("private orphan fixture"), 0600) != nil {
			t.Fatal("orphan setup")
		}
		denied(t, id, 0)
		if os.Remove(name) != nil {
			t.Fatal("orphan removal")
		}
	})
	t.Run("active-writer", func(t *testing.T) {
		held, e := storage.OpenStateDirectory(statePath)
		if e != nil {
			t.Fatal(e)
		}
		defer held.Close()
		denied(t, id, 0)
	})
	t.Run("wrong-uid", func(t *testing.T) { denied(t, id, 65534) })
	t.Run("native-missing", func(t *testing.T) {
		wrong := p
		wrong.PoolName = "absent_" + hex.EncodeToString(nonce[:])
		data, e := storage.EncodeOwnershipRecord(wrong)
		if e != nil {
			t.Fatal(e)
		}
		if os.WriteFile(filepath.Join(statePath, "ownership-"+guid+".json"), data, 0600) != nil {
			t.Fatal("missing pool setup")
		}
		denied(t, id, 0)
		write("ownership-"+guid+".json", policyBytes)
	})
	t.Run("private-baseline-hidden", func(t *testing.T) {
		for _, s := range []string{pool, statePath, "session_digest", "policy_digest", "filesystem_guid", "property-intent", "quota"} {
			if bytes.Contains(first, []byte(s)) {
				t.Fatal("private baseline disclosed")
			}
		}
	})
	t.Run("signal-cancelled", func(t *testing.T) {
		reader, writer, e := os.Pipe()
		if e != nil {
			t.Fatal(e)
		}
		defer reader.Close()
		defer writer.Close()
		fd := int(writer.Fd())
		if unix.SetNonblock(fd, true) != nil {
			t.Fatal("pipe setup")
		}
		for {
			_, e = unix.Write(fd, make([]byte, 4096))
			if errors.Is(e, unix.EAGAIN) {
				break
			}
			if e != nil {
				t.Fatal("fill output pipe")
			}
		}
		if unix.SetNonblock(fd, false) != nil {
			t.Fatal("pipe setup")
		}
		cmd := process(id, 0)
		cmd.Stdout = writer
		var errout bytes.Buffer
		cmd.Stderr = &errout
		if cmd.Start() != nil {
			t.Fatal("start cancelled command")
		}
		finished := make(chan error, 1)
		go func() { finished <- cmd.Wait() }()
		select {
		case <-finished:
			t.Fatal("command exited before signal")
		case <-time.After(500 * time.Millisecond):
		}
		if cmd.Process.Signal(syscall.SIGTERM) != nil {
			_ = cmd.Process.Kill()
			<-finished
			t.Fatal("signal command")
		}
		select {
		case e = <-finished:
			if e == nil || cmd.ProcessState.ExitCode() != 1 || ctx.Err() != nil {
				t.Fatal("cancelled command survived or succeeded")
			}
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-finished
			t.Fatal("signal did not stop command")
		}
	})
	t.Run("preserved-and-released", func(t *testing.T) {
		if out := success(t, id, true, false); !bytes.Equal(out, first) {
			t.Fatal("repeat result drift")
		}
		state, e := storage.OpenStateDirectory(statePath)
		if e != nil {
			t.Fatal("lease not released", e)
		}
		view, e := state.InspectJobReservations()
		if e != nil || len(view.Pools()) != 1 || len(view.Pools()[0].Jobs) != 2 {
			state.Close()
			t.Fatal("holds changed", e)
		}
		if state.Close() != nil {
			t.Fatal("lease close")
		}
		for name, data := range records {
			got, e := os.ReadFile(filepath.Join(statePath, name))
			if e != nil || !bytes.Equal(got, data) {
				t.Fatal("record changed")
			}
		}
		lockAfter, e := os.Stat(filepath.Join(statePath, "writer.lock"))
		if e != nil || !os.SameFile(lockBefore, lockAfter) || raw() != originalNative {
			t.Fatal("lock/native projection changed")
		}
	})
	t.Run("fixture-teardown", func(t *testing.T) {
		if !cleanup() {
			t.Fatal("owned pool/provider/ledger remains")
		}
	})
}
