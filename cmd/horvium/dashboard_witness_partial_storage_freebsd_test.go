//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type witnessPartialStorageOriginal struct {
	Name, SHA256 string
	Object       witnessFileObject
	Mode         uint32
	Ref          *dashboardWitnessRef
}
type witnessPartialStorageReceipt struct {
	Label, DataClass                                                                                                                      string
	Controller                                                                                                                            witnessActor
	AttemptCap, RawCap, ArchiveCap, TmpfsCap                                                                                              int
	Fill                                                                                                                                  *witnessBrokerFill `json:",omitempty"`
	FileLimitBefore, FileLimitDuring, FileLimitAfter                                                                                      *unix.Rlimit       `json:",omitempty"`
	ReleasedBytes, FillerBytesAfterRelease, SourceBytes, ExpectedPrefixBytes                                                              int64
	ExpectedPrefixSHA256                                                                                                                  string
	Boundary                                                                                                                              *witnessCaptureBoundary
	StoreBytes                                                                                                                            int64
	Attempts                                                                                                                              int
	PriorEligible, Failed, PriorDenied, ForwardDenied, ResumeDenied, UnreadRejectedBody, AllocationAfterFailure, ResourceRestored, Passed bool
	Originals                                                                                                                             []witnessPartialStorageOriginal
}

// Synthetic custody originals only: actual FreeBSD file writes/ENOSPC, without
// claiming a producer, broker, manager publication or the resource workload.
func TestNativeDashboardWitnessPartialStorage(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("disposable VM opt-in")
	}
	if os.Getuid() != 0 || witnessCapacityTmpfs != 335544320 || witnessArchiveCap != 268435456 || witnessRawCap != 67508864 || witnessAttemptCap != 32 {
		t.Fatal("root fixture owner / unchanged accepted limits required")
	}
	owner := newDashboardGuestOwner(t)
	cleanups := map[string]bool{}
	t.Cleanup(func() {
		clean := len(cleanups) == 3 && owner.storageCleanups == 0 && dashboardReaperEmpty() == nil && reflect.DeepEqual(owner.artifacts, observeDashboardParentArtifacts(t))
		for _, ok := range cleanups {
			clean = clean && ok
		}
		if witnessWritePrivateIndex("../../.tools", "witness-partial-storage-cleanup.json", struct {
			Verified bool
			Mounts   map[string]bool
		}{clean, cleanups}) != nil {
			t.Error("cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("partial storage cleanup uncertain")
		}
	})
	probeDir := "../../.tools/witness-probe"
	if os.Mkdir(probeDir, 0700) != nil {
		t.Fatal("exclusive probe")
	}
	probe := witnessNativeABI(t, probeDir)
	if witnessWritePrivateIndex("../../.tools", "witness-probe-original.json", probe) != nil {
		t.Fatal("probe receipt")
	}
	actor, e := witnessObserveActor(os.Getpid(), probe.ABI)
	if e != nil {
		t.Fatal("original controller actor", e)
	}
	export := "../../.tools/witness-partial-storage"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive export")
	}
	for _, label := range []string{"normal", "enospc-zero", "file-limit-partial"} {
		t.Run(label, func(t *testing.T) {
			out := filepath.Join(export, label)
			if os.Mkdir(out, 0700) != nil {
				t.Fatal("exclusive case")
			}
			r := witnessPartialStorageReceipt{Label: label, DataClass: "synthetic-custody-only", Controller: actor, AttemptCap: witnessAttemptCap, RawCap: witnessRawCap, ArchiveCap: witnessArchiveCap, TmpfsCap: witnessCapacityTmpfs}
			mount := filepath.Join(witnessPrivateDirectory(t), "original-mount")
			if os.Mkdir(mount, 0700) != nil {
				t.Fatal("exclusive mount directory")
			}
			under, e := os.Lstat(mount)
			if e != nil {
				t.Fatal(e)
			}
			readerFixtureCommand(t, "/sbin/mount", "-t", "tmpfs", "-o", "size=335544320,mode=0700", "tmpfs", mount)
			var original unix.Statfs_t
			if unix.Statfs(mount, &original) != nil || string(bytes.TrimRight(original.Fstypename[:], "\x00")) != "tmpfs" {
				t.Fatal("original mount identity")
			}
			t.Cleanup(func() {
				var current unix.Statfs_t
				ok := unix.Statfs(mount, &current) == nil && current.Fsid == original.Fsid && string(bytes.TrimRight(current.Fstypename[:], "\x00")) == "tmpfs"
				if ok {
					ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					ok = exec.CommandContext(ctx, "/sbin/umount", mount).Run() == nil
					cancel()
				}
				if ok {
					after, e := os.Lstat(mount)
					ok = e == nil && os.SameFile(under, after) && os.Remove(mount) == nil
				}
				cleanups[label] = ok
				if witnessWritePrivateIndex(out, "cleanup.json", struct{ OriginalMount, UnderlyingDirectory, Removed bool }{ok, ok, ok}) != nil {
					t.Error("mount cleanup receipt")
				}
				if !ok {
					t.Error("original owned mount cleanup")
				}
			})
			dir := filepath.Join(mount, "originals")
			if os.Mkdir(dir, 0700) != nil {
				t.Fatal("exclusive custody directory")
			}
			if label == "file-limit-partial" {
				witnessPartialStorageChild(t, witnessPartialStorageChildConfig{Directory: dir, Export: out, ABI: probe.ABI})
			} else {
				witnessPartialStorageExercise(t, &r, dir, out)
			}

		})
	}
}

type witnessPartialStorageChildConfig struct {
	Directory, Export string
	ABI               witnessABI
}

func witnessPartialStorageExercise(t *testing.T, r *witnessPartialStorageReceipt, dir, out string) {
	t.Helper()
	store, e := newDashboardWitnessStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer store.close()
	defer func() {
		r.Failed, r.StoreBytes, r.Attempts, r.Boundary = store.failed, store.bytes, store.counts["native-small"], store.captureBoundary
		entries, e := os.ReadDir(dir)
		if e != nil {
			t.Error("original census")
			return
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			before, e := os.Lstat(path)
			object, oe := witnessObject(before)
			if e != nil || oe != nil || !before.Mode().IsRegular() || object.UID != 0 || object.Links != 1 || object.Size > witnessRawCap {
				t.Error("original custody")
				return
			}
			data, e := os.ReadFile(path)
			after, ae := os.Lstat(path)
			if e != nil || ae != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || int64(len(data)) != object.Size {
				t.Error("original readback")
				return
			}
			item := witnessPartialStorageOriginal{Name: entry.Name(), SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Object: object, Mode: uint32(before.Mode().Perm())}
			if v := store.sealed[entry.Name()]; v != nil {
				copy := v.ref
				item.Ref = &copy
			}
			f, e := os.OpenFile(filepath.Join(out, entry.Name()), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				t.Error("exclusive raw export")
				return
			}
			n, we := f.Write(data)
			se := f.Sync()
			me := f.Chmod(0400)
			ce := f.Close()
			if we != nil || se != nil || me != nil || ce != nil || n != len(data) {
				t.Error("raw export")
				return
			}
			r.Originals = append(r.Originals, item)
		}
		r.Passed = !t.Failed()
		if witnessWritePrivateIndex(out, "receipt.json", r) != nil {
			t.Error("original case receipt")
		}
	}()
	first := witnessCapacityEligible(t, store, 1)
	r.PriorEligible = true
	if r.Label == "normal" {
		return
	}
	filler := ""
	var prior unix.Rlimit
	if r.Label == "enospc-zero" {
		fill, path := witnessFillBrokerMount(t, filepath.Dir(dir))
		r.Fill = fill
		filler = path
		// Free a fixed portion of the actual full-size mount, rather than
		// lowering any store/mount bound or injecting an ENOSPC return.
		r.ReleasedBytes = witnessFrameCap / 2
		r.FillerBytesAfterRelease = fill.Bytes - r.ReleasedBytes
		if os.Truncate(filler, r.FillerBytesAfterRelease) != nil {
			t.Fatal("actual filler truncation")
		}
		info, e := os.Lstat(filler)
		if e != nil || info.Size() != r.FillerBytesAfterRelease {
			t.Fatal("actual released bytes")
		}
	} else {
		// Only this disposable child changes its process file-size limit. The store,
		// archive and full-size tmpfs caps remain unchanged.
		if unix.Getrlimit(unix.RLIMIT_FSIZE, &prior) != nil {
			t.Fatal("original child file limit")
		}
		bounded := prior
		bounded.Cur = 8193
		before, during := prior, bounded
		r.FileLimitBefore, r.FileLimitDuring = &before, &during
		signal.Ignore(syscall.SIGXFSZ)
		if unix.Setrlimit(unix.RLIMIT_FSIZE, &bounded) != nil {
			t.Fatal("child file limit")
		}
		defer func() {
			if unix.Setrlimit(unix.RLIMIT_FSIZE, &prior) != nil {
				t.Error("restore child file limit")
			}
		}()
	}

	source := bytes.Repeat([]byte("owned-partial-storage/"), witnessFrameCap/4)
	r.SourceBytes = int64(len(source))
	ref, e := store.capture(bytes.NewReader(source), witnessCapacityID(2), "native-small", "capacity-management", "capacity-helper")
	if r.Label == "file-limit-partial" && unix.Setrlimit(unix.RLIMIT_FSIZE, &prior) != nil {
		t.Fatal("restore limit before export/readback")
	}
	if r.Label == "file-limit-partial" {
		var after unix.Rlimit
		if unix.Getrlimit(unix.RLIMIT_FSIZE, &after) != nil {
			t.Fatal("restored limit readback")
		}
		r.FileLimitAfter = &after
		r.ResourceRestored = after == prior
	}
	b := store.captureBoundary
	if b != nil {
		r.ExpectedPrefixBytes = b.AcknowledgedBytes
	}
	if r.ExpectedPrefixBytes >= 0 && r.ExpectedPrefixBytes <= int64(len(source)) {
		r.ExpectedPrefixSHA256 = fmt.Sprintf("%x", sha256.Sum256(source[:r.ExpectedPrefixBytes]))
	}
	if e == nil || ref == nil || !store.failed || ref.Complete || ref.Delivered || ref.CaptureFailure != "capture-io" || b == nil || b.Stage != "copy" || b.Error.Class != "syscall" || b.AcknowledgedBytes != ref.RawBytes || ref.RawSHA256 != r.ExpectedPrefixSHA256 || store.check(store.sealed[ref.Name]) != nil {
		t.Fatal("actual partial storage prefix was not retained", e, ref, b)
	}
	if r.Label == "enospc-zero" {
		if ref.RawBytes != 0 || b.Error.Errno != uint64(syscall.ENOSPC) {
			t.Fatal("zero-byte physical ENOSPC changed", ref, b)
		}
	} else if ref.RawBytes != 8193 || b.Error.Errno != uint64(syscall.EFBIG) {
		t.Fatal("actual partial file-limit write missing", ref, b)
	}
	if filler != "" && os.Remove(filler) != nil {
		t.Fatal("release actual filler")
	}
	if r.Label == "enospc-zero" {
		r.ResourceRestored = true
	}
	probePath := filepath.Join(filepath.Dir(dir), "released-space-probe")
	r.AllocationAfterFailure = os.WriteFile(probePath, []byte("actual reclaimed space"), 0600) == nil
	if !r.ResourceRestored || !r.AllocationAfterFailure || os.Remove(probePath) != nil {
		t.Fatal("actual space reclamation")
	}
	_, e = store.original(first)
	r.PriorDenied = e != nil
	r.ForwardDenied = store.forward(ref, &bytes.Buffer{}) != nil
	before := *b
	rejected := &witnessCapacityCount{source: bytes.NewReader(witnessCapacityGolden(witnessCapacityID(3)))}
	next, e := store.capture(rejected, witnessCapacityID(3), "native-small", "capacity-management", "capacity-helper")
	r.ResumeDenied, r.UnreadRejectedBody = e != nil && next == nil, rejected.reads.Load() == 0
	if !r.PriorDenied || !r.ForwardDenied || !r.ResumeDenied || !r.UnreadRejectedBody || !reflect.DeepEqual(before, *store.captureBoundary) || len(store.refs) != 2 || store.counts["native-small"] != 2 {
		t.Fatal("reclaimed-space candidate resumed / original boundary overwritten")
	}
}

func witnessPartialStorageChild(t *testing.T, cfg witnessPartialStorageChildConfig) {
	t.Helper()
	read, write, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	defer read.Close()
	defer write.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	command := exec.CommandContext(ctx, exe, "-test.v", "-test.count=1", "-test.run=^TestNativeDashboardWitnessPartialFileLimitChild$")
	command.Env = append(os.Environ(), "HORVIUM_WITNESS_FILE_LIMIT_CHILD=1")
	command.ExtraFiles = []*os.File{read}
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	started := time.Now()
	if command.Start() != nil {
		t.Fatal("child launch")
	}
	read.Close()
	sendError := witnessControlWrite(write, cfg)
	write.Close()
	waitError := command.Wait()
	elapsed := time.Since(started)
	wait := command.ProcessState.Sys().(syscall.WaitStatus)
	record := struct {
		PID, RawWait, ExitCode           int
		ElapsedNS                        int64
		Deadline, SendFailed, WaitFailed bool
		Output                           []byte
	}{command.Process.Pid, int(wait), command.ProcessState.ExitCode(), elapsed.Nanoseconds(), ctx.Err() != nil, sendError != nil, waitError != nil, output.Bytes()}
	if witnessWritePrivateIndex(cfg.Export, "child-wait.json", record) != nil {
		t.Fatal("child wait receipt")
	}
	if waitError != nil || sendError != nil || ctx.Err() != nil || wait.Signaled() || wait.ExitStatus() != 0 || witnessPIDAliveForPartialStorage(command.Process.Pid) {
		t.Fatal("actual child wait", waitError, output.String())
	}
	f, e := os.Open(filepath.Join(cfg.Export, "receipt.json"))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	var r witnessPartialStorageReceipt
	if witnessControlRead(f, &r) != nil || r.Controller.PID != command.Process.Pid || r.Controller.SHA256 == "" || !r.Passed {
		t.Fatal("child actor/original receipt")
	}
}

func TestNativeDashboardWitnessPartialFileLimitChild(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" || os.Getenv("HORVIUM_WITNESS_FILE_LIMIT_CHILD") != "1" {
		t.Skip("owned disposable child only")
	}
	if os.Getuid() != 0 {
		t.Fatal("root fixture child")
	}
	f := os.NewFile(3, "owned-bootstrap")
	if f == nil {
		t.Fatal("child bootstrap")
	}
	defer f.Close()
	var cfg witnessPartialStorageChildConfig
	if witnessControlRead(f, &cfg) != nil || !filepath.IsAbs(cfg.Directory) || !cfg.ABI.valid() {
		t.Fatal("bounded child bootstrap")
	}
	var extra [1]byte
	if n, e := f.Read(extra[:]); n != 0 || e != io.EOF {
		t.Fatal("bootstrap trailing bytes")
	}
	actor, e := witnessObserveActor(os.Getpid(), cfg.ABI)
	if e != nil {
		t.Fatal("actual child actor")
	}
	r := witnessPartialStorageReceipt{Label: "file-limit-partial", DataClass: "synthetic-custody-only", Controller: actor, AttemptCap: witnessAttemptCap, RawCap: witnessRawCap, ArchiveCap: witnessArchiveCap, TmpfsCap: witnessCapacityTmpfs}
	witnessPartialStorageExercise(t, &r, cfg.Directory, cfg.Export)
}

func witnessPIDAliveForPartialStorage(pid int) bool { return unix.Kill(pid, 0) == nil }
