//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

const witnessCapacityTmpfs = 320 << 20

type witnessCapacityZero struct{}

func (witnessCapacityZero) Read(p []byte) (int, error) { clear(p); return len(p), nil }

type witnessCapacityCount struct {
	source io.Reader
	reads  atomic.Int64
	bytes  atomic.Int64
}

func (r *witnessCapacityCount) Read(p []byte) (int, error) {
	r.reads.Add(1)
	n, e := r.source.Read(p)
	r.bytes.Add(int64(n))
	return n, e
}
func witnessCapacityID(n int) string { return fmt.Sprintf("%s-%d", witnessTestID[:32], n) }

type witnessCapacityOriginal struct {
	Name, SourceKind, SHA256 string
	Object                   witnessFileObject
	Mode                     uint32
	Ref                      *dashboardWitnessRef
}
type witnessCapacityReceipt struct {
	Label, DataClass                                                                 string
	Controller                                                                       witnessActor
	AttemptCap, RawCap, ArchiveCap, TmpfsCap                                         int
	Attempts                                                                         int
	StoredBytes                                                                      int64
	Passed, Failed, PriorEligible, BoundaryEligible, ResumeDenied, OldOriginalDenied bool
	UnreadRejectedBody                                                               bool
	Failure                                                                          string
	ENOSPC, SpaceReleased                                                            bool
	FillBytes                                                                        int64
	FillSHA256                                                                       string
	FillObject                                                                       *witnessFileObject
	Originals                                                                        []witnessCapacityOriginal
}

func witnessCapacityEligible(t *testing.T, s *dashboardWitnessStore, n int) *dashboardWitnessRef {
	t.Helper()
	id := witnessCapacityID(n)
	raw := witnessCapacityGolden(id)
	ref, e := s.capture(bytes.NewReader(raw), id, "native-small", "capacity-management", "capacity-helper")
	if e != nil || ref == nil || !ref.Complete {
		t.Fatal("real complete original", n, e)
	}
	sender, peer := witnessTestPair(t)
	if s.forward(ref, sender) != nil {
		t.Fatal("actual byte/half-close delivery")
	}
	received, e := io.ReadAll(peer)
	sender.Close()
	peer.Close()
	if e != nil || !bytes.Equal(received, raw) || ref.RawSHA256 != ref.ForwardSHA256 {
		t.Fatal("original actual endpoint bytes/EOF")
	}
	if snap, e := s.original(ref); e != nil || !snap.Complete() {
		t.Fatal("original eligibility", e)
	}
	return ref
}
func witnessCapacityResume(t *testing.T, s *dashboardWitnessStore, first *dashboardWitnessRef, r *witnessCapacityReceipt) {
	t.Helper()
	probe := &witnessCapacityCount{source: bytes.NewReader(witnessCapacityGolden(strings.Repeat("f", 32) + "-1"))}
	ref, e := s.capture(probe, strings.Repeat("f", 32)+"-1", "projection-10000", "new-management", "new-helper")
	r.ResumeDenied = e != nil && ref == nil
	r.UnreadRejectedBody = probe.reads.Load() == 0
	_, e = s.original(first)
	r.OldOriginalDenied = e != nil
	if !s.failed || !r.ResumeDenied || !r.UnreadRejectedBody || !r.OldOriginalDenied {
		t.Fatal("failed candidate resumed or read rejected body")
	}
}

func witnessCapacityExport(t *testing.T, s *dashboardWitnessStore, out string, r *witnessCapacityReceipt) {
	t.Helper()
	r.Failed = s.failed
	r.StoredBytes = s.bytes
	r.Attempts = s.counts["native-small"]
	r.Passed = !t.Failed()
	entries, e := os.ReadDir(s.directory)
	if e != nil {
		t.Error("original census")
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "original-") || !strings.HasSuffix(entry.Name(), ".wire") {
			t.Error("unknown original entry")
			return
		}
		path := filepath.Join(s.directory, entry.Name())
		before, e := os.Lstat(path)
		object, oe := witnessObject(before)
		if e != nil || oe != nil || !before.Mode().IsRegular() || object.UID != 0 || object.Links != 1 || before.Size() > witnessRawCap {
			t.Error("original custody")
			return
		}
		f, e := os.Open(path)
		if e != nil {
			t.Error(e)
			return
		}
		dest, e := os.OpenFile(filepath.Join(out, entry.Name()), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			f.Close()
			t.Error(e)
			return
		}
		h := sha256.New()
		n, we := io.CopyBuffer(io.MultiWriter(dest, h), io.LimitReader(f, before.Size()+1), make([]byte, witnessFrameCap))
		se := dest.Sync()
		me := dest.Chmod(0400)
		ce := dest.Close()
		f.Close()
		after, ae := os.Lstat(path)
		if we != nil || se != nil || me != nil || ce != nil || ae != nil || n != before.Size() || !os.SameFile(before, after) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
			t.Error("original changed/export failed")
			return
		}
		kind := "literal-golden"
		if (r.Label == "archive-cap" || r.Label == "raw-cap") && entry.Name() != "original-001.wire" {
			kind = "zero-stream"
		}
		original := witnessCapacityOriginal{Name: entry.Name(), SourceKind: kind, SHA256: fmt.Sprintf("%x", h.Sum(nil)), Object: object, Mode: uint32(before.Mode().Perm())}
		if v := s.sealed[entry.Name()]; v != nil {
			copy := v.ref
			original.Ref = &copy
			if copy.RawSHA256 != original.SHA256 || copy.RawBytes != n {
				t.Error("indexed original mismatch")
				return
			}
		}
		r.Originals = append(r.Originals, original)
	}
	r.Passed = r.Passed && !t.Failed()
	if witnessWritePrivateIndex(out, "receipt.json", r) != nil {
		t.Error("exclusive capacity receipt")
	}
}

func TestNativeDashboardWitnessCapacity(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("disposable VM opt-in")
	}
	if os.Getuid() != 0 {
		t.Fatal("root fixture owner required")
	}
	if witnessAttemptCap != 32 || witnessFrameCap != 65536 || witnessRawCap != 67508864 || witnessArchiveCap != 268435456 || witnessCapacityTmpfs != 335544320 {
		t.Fatal("accepted limits changed")
	}
	export := "../../.tools/witness-capacity"
	if os.Mkdir(export, 0700) != nil {
		t.Fatal("exclusive capacity export")
	}
	owner := newDashboardGuestOwner(t)
	cleanups := map[string]bool{}
	t.Cleanup(func() {
		clean := len(cleanups) == 5 && owner.storageCleanups == 0 && dashboardReaperEmpty() == nil && reflect.DeepEqual(owner.artifacts, observeDashboardParentArtifacts(t))
		for _, ok := range cleanups {
			clean = clean && ok
		}
		if witnessWritePrivateIndex("../../.tools", "witness-capacity-cleanup.json", struct {
			Mounts                                          map[string]bool
			NoZFSOrMD, ArtifactsUnchanged, DescendantsEmpty bool
		}{cleanups, owner.storageCleanups == 0, reflect.DeepEqual(owner.artifacts, observeDashboardParentArtifacts(t)), dashboardReaperEmpty() == nil}) != nil {
			t.Error("original cleanup receipt")
		}
		t.Logf("DASHBOARD_WITNESS_CLEANUP verified=%t resource_acceptance=false", clean)
		if !clean {
			t.Error("capacity cleanup uncertain")
		}
	})
	probeDir := "../../.tools/witness-probe"
	if os.Mkdir(probeDir, 0700) != nil {
		t.Fatal("exclusive probe")
	}
	probe := witnessNativeABI(t, probeDir)
	if witnessWritePrivateIndex("../../.tools", "witness-probe-original.json", probe) != nil {
		t.Fatal("probe original")
	}
	actor, e := witnessObserveActor(os.Getpid(), probe.ABI)
	if e != nil {
		t.Fatal("original fixture actor")
	}
	for _, label := range []string{"attempt-cap", "single-active", "raw-cap", "archive-cap", "physical-tmpfs"} {
		t.Run(label, func(t *testing.T) {
			out := filepath.Join(export, label)
			if os.Mkdir(out, 0700) != nil {
				t.Fatal("exclusive case export")
			}
			if witnessWritePrivateIndex(out, "started.json", struct {
				Label, DataClass string
				Controller       witnessActor
			}{label, "synthetic-custody-only", actor}) != nil {
				t.Fatal("original start")
			}
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
				t.Fatal("original physical mount")
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
					t.Error("case cleanup receipt")
				}
				if !ok {
					t.Error("owned mount cleanup")
				}
			})
			dir := filepath.Join(mount, "originals")
			if os.Mkdir(dir, 0700) != nil {
				t.Fatal("exclusive original namespace")
			}
			store, e := newDashboardWitnessStore(dir)
			if e != nil {
				t.Fatal(e)
			}
			defer store.close()
			receipt := &witnessCapacityReceipt{Label: label, DataClass: "synthetic-custody-only", Controller: actor, AttemptCap: witnessAttemptCap, RawCap: witnessRawCap, ArchiveCap: witnessArchiveCap, TmpfsCap: witnessCapacityTmpfs}
			defer witnessCapacityExport(t, store, out, receipt)
			var first *dashboardWitnessRef
			switch label {
			case "attempt-cap":
				for n := 1; n <= 32; n++ {
					ref := witnessCapacityEligible(t, store, n)
					if n == 1 {
						first = ref
					}
				}
				receipt.PriorEligible = true
				receipt.BoundaryEligible = true
				rejected := &witnessCapacityCount{source: bytes.NewReader(witnessCapacityGolden(witnessCapacityID(33)))}
				ref, e := store.capture(rejected, witnessCapacityID(33), "native-small", "capacity-management", "capacity-helper")
				if e == nil || ref != nil || rejected.reads.Load() != 0 || len(store.refs) != 32 || store.counts["native-small"] != 32 {
					t.Fatal("attempt 33 did not reject before reading/writing")
				}
				receipt.Failure = "attempt-limit"
			case "single-active":
				entered := make(chan struct{})
				release := make(chan struct{})
				var once sync.Once
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				var result *dashboardWitnessRef
				var captureError error
				done := make(chan struct{})
				source := &witnessInterruptedReader{source: bytes.NewReader(witnessCapacityGolden(witnessCapacityID(1))), reach: func() error {
					close(entered)
					select {
					case <-release:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				}}
				go func() {
					result, captureError = store.capture(source, witnessCapacityID(1), "native-small", "capacity-management", "capacity-helper")
					close(done)
				}()
				defer func() { once.Do(func() { close(release) }); <-done }()
				select {
				case <-entered:
				case <-ctx.Done():
					t.Fatal("real first capture never blocked")
				}
				second := &witnessCapacityCount{source: bytes.NewReader(witnessCapacityGolden(witnessCapacityID(2)))}
				secondDone := make(chan error, 1)
				go func() {
					_, e := store.capture(second, witnessCapacityID(2), "native-small", "capacity-management", "capacity-helper")
					secondDone <- e
				}()
				select {
				case e := <-secondDone:
					if e == nil || second.reads.Load() != 0 {
						t.Fatal("concurrent capture accepted or consumed")
					}
				case <-time.After(2 * time.Second):
					once.Do(func() { close(release) })
					<-secondDone
					t.Fatal("concurrent capture queued")
				}
				once.Do(func() { close(release) })
				<-done
				if captureError != nil || result == nil || !result.Complete || !store.failed || len(store.refs) != 1 || store.counts["native-small"] != 1 {
					t.Fatal("original overlap/latch result")
				}
				first = result
				receipt.Failure = "single-active"
			case "raw-cap":
				first = witnessCapacityEligible(t, store, 1)
				receipt.PriorEligible = true
				source := &witnessCapacityCount{source: io.LimitReader(witnessCapacityZero{}, int64(witnessRawCap)+1)}
				ref, e := store.capture(source, witnessCapacityID(2), "native-small", "capacity-management", "capacity-helper")
				if e == nil || ref == nil || ref.RawBytes != witnessRawCap || ref.CaptureFailure != "raw-limit" || ref.Complete || ref.Delivered || source.bytes.Load() != int64(witnessRawCap)+1 {
					t.Fatal("real raw-cap prefix/one-byte overrun not retained")
				}
				receipt.Failure = ref.CaptureFailure
			case "archive-cap":
				first = witnessCapacityEligible(t, store, 1)
				receipt.PriorEligible = true
				n := 2
				for store.bytes < witnessArchiveCap {
					remaining := min(int64(64<<20), int64(witnessArchiveCap)-store.bytes)
					ref, e := store.capture(io.LimitReader(witnessCapacityZero{}, remaining), witnessCapacityID(n), "native-small", "capacity-management", "capacity-helper")
					if e != nil || ref == nil || ref.RawBytes != remaining || ref.Complete || ref.CaptureFailure != "" || store.failed {
						t.Fatal("actual archive fill before boundary", e)
					}
					n++
				}
				if store.bytes != witnessArchiveCap {
					t.Fatal("archive boundary not exactly reached")
				}
				_, e := store.original(first)
				receipt.BoundaryEligible = e == nil
				if e != nil {
					t.Fatal("exact boundary invalidated previous complete original")
				}
				ref, e := store.capture(io.LimitReader(witnessCapacityZero{}, 1), witnessCapacityID(n), "native-small", "capacity-management", "capacity-helper")
				if e == nil || ref == nil || ref.RawBytes != 0 || ref.CaptureFailure != "archive-limit" || ref.Complete || ref.Delivered || store.bytes != witnessArchiveCap {
					t.Fatal("actual archive overflow not denied")
				}
				receipt.Failure = ref.CaptureFailure
			case "physical-tmpfs":
				first = witnessCapacityEligible(t, store, 1)
				receipt.PriorEligible = true
				fillPath := filepath.Join(mount, "owned-filler")
				fill, e := os.OpenFile(fillPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if e != nil {
					t.Fatal(e)
				}
				defer fill.Close()
				buffer := make([]byte, witnessFrameCap)
				var writeError error
				for receipt.FillBytes <= witnessCapacityTmpfs {
					n, e := fill.Write(buffer)
					receipt.FillBytes += int64(n)
					if e != nil {
						writeError = e
						break
					}
				}
				receipt.ENOSPC = errors.Is(writeError, syscall.ENOSPC)
				if !receipt.ENOSPC || receipt.FillBytes <= 0 || receipt.FillBytes > witnessCapacityTmpfs {
					t.Fatal("full-size physical ENOSPC not observed")
				}
				info, e := fill.Stat()
				object, oe := witnessObject(info)
				if e != nil || oe != nil || object.UID != 0 || object.Links != 1 || info.Size() != receipt.FillBytes {
					t.Fatal("original filler custody")
				}
				receipt.FillObject = &object
				h := sha256.New()
				if _, e := fill.Seek(0, io.SeekStart); e != nil {
					t.Fatal(e)
				}
				// Reopen the original write-only object and hash actual bytes before removal.
				opened, e := os.Open(fillPath)
				if e != nil {
					t.Fatal(e)
				}
				observed, e := io.CopyBuffer(h, opened, buffer)
				opened.Close()
				after, ae := os.Lstat(fillPath)
				if e != nil || ae != nil || !os.SameFile(info, after) || after.Size() != info.Size() || observed != receipt.FillBytes {
					t.Fatal("original filler readback")
				}
				receipt.FillSHA256 = fmt.Sprintf("%x", h.Sum(nil))
				_, e = store.capture(bytes.NewReader(witnessCapacityGolden(witnessCapacityID(2))), witnessCapacityID(2), "native-small", "capacity-management", "capacity-helper")
				if e == nil || !store.failed || store.bytes >= witnessArchiveCap {
					t.Fatal("physical I/O uncertainty accepted or archive limit substituted")
				}
				fill.Close()
				if os.Remove(fillPath) != nil {
					t.Fatal("original filler removal")
				}
				var freed unix.Statfs_t
				receipt.SpaceReleased = unix.Statfs(mount, &freed) == nil && freed.Fsid == original.Fsid && freed.Bavail > 0
				if !receipt.SpaceReleased {
					t.Fatal("original physical space not released")
				}
				receipt.Failure = "physical-enospc"
			}
			witnessCapacityResume(t, store, first, receipt)
			t.Logf("NATIVE_WITNESS_CAPACITY case=%s failed=true resume_denied=true actual_counters=true resource_acceptance=false", label)
		})
	}
}
