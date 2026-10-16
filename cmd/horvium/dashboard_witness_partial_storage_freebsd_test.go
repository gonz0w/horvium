//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
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
	Label, DataClass                                                                                           string
	Controller                                                                                                 witnessActor
	AttemptCap, RawCap, ArchiveCap, TmpfsCap                                                                   int
	Fill                                                                                                       *witnessBrokerFill `json:",omitempty"`
	ReleasedBytes, FillerBytesAfterRelease, SourceBytes, ExpectedPrefixBytes                                   int64
	ExpectedPrefixSHA256                                                                                       string
	Boundary                                                                                                   *witnessCaptureBoundary
	StoreBytes                                                                                                 int64
	Attempts                                                                                                   int
	PriorEligible, Failed, PriorDenied, ForwardDenied, ResumeDenied, UnreadRejectedBody, SpaceReleased, Passed bool
	Originals                                                                                                  []witnessPartialStorageOriginal
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
		clean := len(cleanups) == 2 && owner.storageCleanups == 0 && dashboardReaperEmpty() == nil && reflect.DeepEqual(owner.artifacts, observeDashboardParentArtifacts(t))
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
	for _, label := range []string{"normal", "partial-enospc"} {
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
			if label == "normal" {
				return
			}
			fill, filler := witnessFillBrokerMount(t, mount)
			r.Fill = fill
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
			source := bytes.Repeat([]byte("owned-partial-storage/"), witnessFrameCap/4)
			r.SourceBytes = int64(len(source))
			ref, e := store.capture(bytes.NewReader(source), witnessCapacityID(2), "native-small", "capacity-management", "capacity-helper")
			b := store.captureBoundary
			if b != nil {
				r.ExpectedPrefixBytes = b.AcknowledgedBytes
			}
			if r.ExpectedPrefixBytes >= 0 && r.ExpectedPrefixBytes <= int64(len(source)) {
				r.ExpectedPrefixSHA256 = fmt.Sprintf("%x", sha256.Sum256(source[:r.ExpectedPrefixBytes]))
			}
			if e == nil || ref == nil || !store.failed || ref.Complete || ref.Delivered || ref.CaptureFailure != "capture-io" || ref.RawBytes <= 0 || ref.RawBytes >= int64(witnessFrameCap) || b == nil || b.Stage != "copy" || b.Error.Class != "syscall" || b.Error.Errno != uint64(syscall.ENOSPC) || b.AcknowledgedBytes != ref.RawBytes || ref.RawSHA256 != r.ExpectedPrefixSHA256 || store.check(store.sealed[ref.Name]) != nil {
				t.Fatal("actual partial storage prefix was not retained", e, ref, b)
			}
			if os.Remove(filler) != nil {
				t.Fatal("release actual filler")
			}
			probePath := filepath.Join(mount, "released-space-probe")
			r.SpaceReleased = os.WriteFile(probePath, []byte("actual reclaimed space"), 0600) == nil
			if !r.SpaceReleased || os.Remove(probePath) != nil {
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
		})
	}
}
