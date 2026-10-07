// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
)

// Re-exec the compiled public main/flag/signal path, without building in a guest.
// Only the opted-in native parent enables this helper; it is not a product switch.
func TestStorageJobStatusCommandProcess(t *testing.T) {
	if os.Getenv("HORVIUM_STATUS_COMMAND_PROCESS") != "1" {
		return
	}
	var args []string
	if json.Unmarshal([]byte(os.Getenv("HORVIUM_STATUS_COMMAND_ARGS")), &args) != nil {
		os.Exit(2)
	}
	os.Args = append([]string{os.Args[0]}, args...)
	main()
}

func TestNativeStorageJobStatus(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATUS_VM") != "1" {
		t.Skip("disposable metadata VM opt-ins required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("native parent requires actual root")
	}
	// The selected status suite owns no disk, pool, mount, daemon or listener.
	path, err := os.MkdirTemp("/root", "storage-status-")
	if err != nil {
		t.Fatal(err)
	}
	var temp string
	var held *storage.StateDirectory
	cleanup := func() bool {
		if held != nil {
			if held.Close() != nil {
				return false
			}
			held = nil
		}
		if os.RemoveAll(path) != nil || temp != "" && os.RemoveAll(temp) != nil {
			return false
		}
		_, a := os.Lstat(path)
		gone := errors.Is(a, os.ErrNotExist)
		if temp != "" {
			_, b := os.Lstat(temp)
			gone = gone && errors.Is(b, os.ErrNotExist)
		}
		return gone
	}
	t.Cleanup(func() {
		if !cleanup() {
			t.Error("native status cleanup")
		}
	})
	temp, err = os.MkdirTemp("/tmp", "storage-status-command-")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(temp, 0755); err != nil {
		t.Fatal(err)
	}
	original, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(original)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(temp, "command.test")
	if err = os.WriteFile(binary, data, 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(binary, 0755); err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(copied) != sha256.Sum256(data) {
		t.Fatal("command binary changed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cases := []string{"built_process", "empty", "mixed_complete", "queued_held", "applying_held", "observing_held", "terminal_retained", "private_baseline_hidden", "repeat_preserved", "lease_released", "active_writer_denied", "kernel_uid_denied", "unknown_denied", "newer_denied", "unsafe_file_denied", "orphan_denied", "cancelled_denied", "output_cancelled", "native_unavailable_not_consulted", "lock_preserved", "fixtures_teardown"}
	passed := make(map[string]bool)
	check := func(name string, ok bool) {
		t.Helper()
		if !ok || passed[name] {
			t.Fatal("native status assertion", name)
		}
		passed[name] = true
	}
	command := func(uid uint32) ([]byte, []byte, error) {
		args, _ := json.Marshal([]string{"storage-jobs", "-state-dir", path})
		cmd := exec.CommandContext(ctx, binary, "-test.run=^TestStorageJobStatusCommandProcess$")
		cmd.Env = append(os.Environ(), "HORVIUM_STATUS_COMMAND_PROCESS=1", "HORVIUM_STATUS_COMMAND_ARGS="+string(args))
		if uid != 0 {
			cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: uid, Groups: []uint32{}}}
		}
		var out, errout bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errout
		e := cmd.Run() // Run always waits/reaps the owned command.
		return out.Bytes(), errout.Bytes(), e
	}
	out, errout, e := command(0)
	var result storageJobStatus
	check("built_process", e == nil && len(errout) == 0 && json.Unmarshal(out, &result) == nil)
	check("empty", len(result.Jobs) == 0 && !result.RecoveryVerified && result.HoldsRetained)
	lockBefore, e := os.Lstat(filepath.Join(path, "writer.lock"))
	if e != nil {
		t.Fatal(e)
	}
	records := make(map[string][]byte)
	write := func(name string, b []byte) {
		t.Helper()
		if e := os.WriteFile(filepath.Join(path, name), b, 0600); e != nil {
			t.Fatal(e)
		}
		records[name] = append([]byte(nil), b...)
	}
	for i := 1; i <= 65; i++ {
		j := statusJobFixture(t, i, i >= 64)
		if i == 2 || i == 3 {
			j, e = storage.TransitionJob(j, storage.JobApplying, 1001, "launch-intent", "")
			if e != nil {
				t.Fatal(e)
			}
			if i == 3 {
				j, e = storage.TransitionJob(j, storage.JobObserving, 1002, "native-observation", strings.Repeat("d", 64))
				if e != nil {
					t.Fatal(e)
				}
			}
		}
		b, e := storage.EncodeJobRecord(j)
		if e != nil {
			t.Fatal(e)
		}
		write("job-"+j.ID+".json", b)
	}
	property, e := storage.DecodePropertyJob(statusPropertyFixture)
	if e != nil {
		t.Fatal(e)
	}
	j := property.Job()
	j, e = storage.TransitionJob(j, storage.JobApplying, j.History[0].At+1, "launch-intent", "")
	if e != nil {
		t.Fatal(e)
	}
	property, e = storage.BindPropertyJob(j, property.Intent())
	if e != nil {
		t.Fatal(e)
	}
	b, e := storage.EncodePropertyJob(property)
	if e != nil {
		t.Fatal(e)
	}
	write("job-"+j.ID+".json", b)
	out, errout, e = command(0)
	result = storageJobStatus{}
	check("mixed_complete", e == nil && len(errout) == 0 && json.Unmarshal(out, &result) == nil && len(result.Jobs) == 66)
	check("queued_held", result.Jobs[0].LastDurable.State == storage.JobQueued && result.Jobs[0].RestartReason == "review-expired" && result.Jobs[0].RestartState == storage.JobReconcileRequired && result.HoldsRetained && !result.RecoveryVerified)
	check("applying_held", result.Jobs[1].LastDurable.State == storage.JobApplying && result.Jobs[1].RestartReason == "executor-unverified" && result.Jobs[65].ID == j.ID && result.Jobs[65].RestartReason == "executor-unverified")
	check("observing_held", result.Jobs[2].LastDurable.State == storage.JobObserving && result.Jobs[2].LastDurable.ObservationDigest == strings.Repeat("d", 64) && result.Jobs[2].RestartReason == "executor-unverified")
	check("terminal_retained", result.Jobs[63].RestartState == storage.JobCancelled && result.Jobs[64].RestartState == storage.JobCancelled && result.HoldsRetained)
	private := true
	for _, field := range []string{"session_digest", "policy_digest", "tank/data", "property-intent", "filesystem_guid"} {
		private = private && !bytes.Contains(out, []byte(field))
	}
	check("private_baseline_hidden", private)
	first := append([]byte(nil), out...)
	out, errout, e = command(0)
	preserved := e == nil && len(errout) == 0 && bytes.Equal(out, first)
	for name, b := range records {
		got, e := os.ReadFile(filepath.Join(path, name))
		preserved = preserved && e == nil && bytes.Equal(got, b)
	}
	check("repeat_preserved", preserved)
	querySucceeded := e == nil
	// Fill an owned pipe, retaining its unread end. Public main must exit on
	// SIGTERM even when its stdout cannot deliver the complete status document.
	reader, writer, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
	fd := int(writer.Fd())
	if unix.SetNonblock(fd, true) != nil {
		t.Fatal("pipe nonblocking setup")
	}
	for {
		_, e = unix.Write(fd, make([]byte, 4096))
		if errors.Is(e, unix.EAGAIN) {
			break
		}
		if e != nil {
			t.Fatal("fill owned output pipe")
		}
	}
	if unix.SetNonblock(fd, false) != nil {
		t.Fatal("pipe blocking setup")
	}
	args, _ := json.Marshal([]string{"storage-jobs", "-state-dir", path})
	blocked := exec.CommandContext(ctx, binary, "-test.run=^TestStorageJobStatusCommandProcess$")
	blocked.Env = append(os.Environ(), "HORVIUM_STATUS_COMMAND_PROCESS=1", "HORVIUM_STATUS_COMMAND_ARGS="+string(args))
	blocked.Stdout = writer
	var blockedErrors bytes.Buffer
	blocked.Stderr = &blockedErrors
	if blocked.Start() != nil {
		t.Fatal("start owned blocked-output command")
	}
	finished := make(chan error, 1)
	go func() { finished <- blocked.Wait() }()
	select {
	case <-finished:
		t.Fatal("full output pipe unexpectedly completed")
	case <-time.After(500 * time.Millisecond):
	}
	if blocked.Process.Signal(syscall.SIGTERM) != nil {
		_ = blocked.Process.Kill()
		<-finished
		t.Fatal("signal owned command")
	}
	select {
	case e = <-finished:
		check("output_cancelled", e != nil && blocked.ProcessState.ExitCode() == 1 && ctx.Err() == nil)
	case <-time.After(3 * time.Second):
		_ = blocked.Process.Kill()
		<-finished
		t.Fatal("cancelled stdout command survived")
	}
	_ = reader.Close()
	_ = writer.Close()
	// Independently prove that the stored fixture GUID has no imported native
	// pool. The product query does not need native collection to show the job.
	poolGUIDs, poolErr := exec.CommandContext(ctx, "/sbin/zpool", "list", "-H", "-o", "guid").Output()
	missing := poolErr == nil
	for _, guid := range strings.Fields(string(poolGUIDs)) {
		missing = missing && guid != "100"
	}
	check("native_unavailable_not_consulted", missing && result.Jobs[0].PoolGUID == "100" && querySucceeded)
	held, e = storage.OpenStateDirectory(path)
	check("lease_released", e == nil)
	out, _, e = command(0)
	check("active_writer_denied", e != nil && len(out) == 0)
	if held.Close() != nil {
		t.Fatal("lease close")
	}
	out, _, e = command(65534)
	check("kernel_uid_denied", e != nil && len(out) == 0)
	for _, mode := range []string{"unknown", "newer", "unsafe_file", "orphan"} {
		name := "foreign-private-marker"
		marker := []byte("private malformed fixture output")
		switch mode {
		case "newer":
			name = "ownership-999.json"
			marker = []byte(`{"version":999,"private_marker":"not disclosed"}`)
		case "unsafe_file":
			name = "job-" + statusJobFixture(t, 1, false).ID + ".json"
			marker = records[name]
		case "orphan":
			name = ".ownership-owned-status-orphan"
		}
		if e = os.WriteFile(filepath.Join(path, name), marker, 0600); e != nil {
			t.Fatal(e)
		}
		if mode == "unsafe_file" && os.Chmod(filepath.Join(path, name), 0644) != nil {
			t.Fatal("fixture mode")
		}
		out, errout, e = command(0)
		got, readErr := os.ReadFile(filepath.Join(path, name))
		check(mode+"_denied", e != nil && len(out) == 0 && !bytes.Contains(errout, marker) && readErr == nil && bytes.Equal(got, marker))
		if mode == "unsafe_file" {
			if os.Chmod(filepath.Join(path, name), 0600) != nil {
				t.Fatal("restore mode")
			}
		} else if os.Remove(filepath.Join(path, name)) != nil {
			t.Fatal("remove marker")
		}
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	var noout bytes.Buffer
	check("cancelled_denied", errors.Is(runStorageJobs(cancelled, path, &noout), context.Canceled) && noout.Len() == 0)
	lockAfter, e := os.Lstat(filepath.Join(path, "writer.lock"))
	check("lock_preserved", e == nil && os.SameFile(lockBefore, lockAfter) && lockAfter.Mode().Perm() == 0600)
	var backing unix.Statfs_t
	if unix.Statfs(path, &backing) != nil {
		t.Fatal("metadata backing")
	}
	soft := "off"
	if backing.Flags&unix.MNT_SOFTDEP != 0 {
		soft = "on"
	}
	check("fixtures_teardown", cleanup())
	if len(passed) != len(cases) {
		t.Fatal("missing assertion")
	}
	fields := make([]string, 0, len(cases))
	for _, name := range cases {
		fields = append(fields, name+"=pass")
	}
	t.Log("STORAGE_JOB_STATUS_RESULT " + strings.Join(fields, " "))
	t.Logf("STORAGE_JOB_STATUS_BACKING filesystem=%s softupdates=%s teardown=pass", unix.ByteSliceToString(backing.Fstypename[:]), soft)

}
