// SPDX-License-Identifier: BSD-3-Clause
//go:build freebsd

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
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// These are faulted custody fixtures on the already-owned original tmpfs, not
// native producer witnesses or a complete process-cancellation matrix.
func dashboardNativeWitnessCustodyFaults(t *testing.T, mount string) {
	var fs unix.Statfs_t
	if os.Geteuid() != 0 || unix.Statfs(mount, &fs) != nil || string(bytes.TrimRight(fs.Fstypename[:], "\x00")) != "tmpfs" {
		t.Fatal("original fault tmpfs identity")
	}
	target := filepath.Join("../../.tools", "witness-custody-faults")
	if os.Mkdir(target, 0700) != nil {
		t.Fatal("exclusive fault archive")
	}
	for _, name := range []string{"cancelled-read", "failed-read", "archive-limit", "half-close"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(mount, "fault-"+name)
			if os.Mkdir(dir, 0700) != nil {
				t.Fatal("exclusive fault directory")
			}
			store, e := newDashboardWitnessStore(dir)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() {
				store.close()
				entries, e := os.ReadDir(dir)
				if e != nil {
					t.Error("fault directory readback")
					return
				}
				for _, entry := range entries {
					if entry.IsDir() || os.Remove(filepath.Join(dir, entry.Name())) != nil {
						t.Error("fault original cleanup")
					}
				}
				if os.Remove(dir) != nil {
					t.Error("fault directory cleanup")
				}
			})
			raw := witnessFrames(witnessGolden())
			prefix := raw[:80]
			var source io.Reader = bytes.NewReader(raw)
			switch name {
			case "cancelled-read":
				source = &witnessTerminalReader{bytes.NewReader(prefix), context.Canceled}
			case "failed-read":
				source = &witnessTerminalReader{bytes.NewReader(prefix), io.ErrUnexpectedEOF}
			case "archive-limit":
				store.bytes = witnessArchiveCap - int64(len(prefix))
			}
			ref, e := store.capture(source, witnessTestID, "native-small", "fault-management", "fault-helper")
			if name == "half-close" {
				if e != nil || ref == nil || !ref.Complete {
					t.Fatal("fault capture setup", e)
				}
				e = store.forward(ref, &witnessHalfCloseFailure{})
			}
			if e == nil || ref == nil || ref.Delivered || !store.failed || len(store.refs) != 1 {
				t.Fatal("fault original lost or eligible", e)
			}
			if name != "half-close" && (ref.Complete || ref.CaptureFailure == "") {
				t.Fatal("capture fault became complete")
			}
			if store.check(store.sealed[ref.Name]) != nil {
				t.Fatal("fault original custody")
			}
			info, e := os.Lstat(filepath.Join(dir, ref.Name))
			if e != nil {
				t.Fatal(e)
			}
			object, e := witnessObject(info)
			if e != nil || object.UID != 0 || object.Links != 1 {
				t.Fatal("root fault object")
			}
			retained, e := os.ReadFile(filepath.Join(dir, ref.Name))
			if e != nil || int64(len(retained)) != ref.RawBytes || fmt.Sprintf("%x", sha256.Sum256(retained)) != ref.RawSHA256 {
				t.Fatal("fault original readback")
			}
			if _, e := store.original(ref); e == nil {
				t.Fatal("fault publication accepted")
			}
			if _, e := store.capture(bytes.NewReader(raw), witnessTestID[:32]+"-2", "native-small", "fault-management", "new-helper"); e == nil {
				t.Fatal("fault candidate resumed")
			}
			// Export actual captured bytes and original object receipts; fixed labels
			// identify injected fixtures without publishing their wire inventory.
			export, e := os.OpenFile(filepath.Join(target, name+".wire"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				t.Fatal("exclusive fault export")
			}
			n, we := export.Write(retained)
			se := export.Sync()
			ce := export.Close()
			if we != nil || se != nil || ce != nil || n != len(retained) || witnessWritePrivateIndex(target, name+".json", witnessNativeOriginal{*ref, object}) != nil {
				t.Fatal("fault archive preservation")
			}
		})
	}
	t.Log("NATIVE_DASHBOARD_WITNESS_CUSTODY faults=4 retained=4 accepted=0 process_cancellation_complete=false")
}

// Use a separate nested mount so physical exhaustion cannot disturb the live
// observer's originals. The normal witness/archive caps are never increased.
func dashboardNativeWitnessAdditionalFaults(t *testing.T, mount string) {
	t.Helper()
	t.Run("receipt-latches-custody-failure", func(t *testing.T) {
		dir := filepath.Join(mount, "receipt-mutation")
		if os.Mkdir(dir, 0700) != nil {
			t.Fatal("exclusive mutation fixture")
		}
		store, e := newDashboardWitnessStore(dir)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() {
			store.close()
			entries, _ := os.ReadDir(dir)
			for _, entry := range entries {
				if os.Remove(filepath.Join(dir, entry.Name())) != nil {
					t.Error("mutation original cleanup")
				}
			}
			if os.Remove(dir) != nil {
				t.Error("mutation directory cleanup")
			}
		})
		raw := witnessFrames(witnessGolden())
		ref, e := store.capture(bytes.NewReader(raw), witnessTestID, "native-small", "fault-management", "fault-helper")
		if e != nil {
			t.Fatal(e)
		}
		if store.forward(ref, &bytes.Buffer{}) != nil {
			t.Fatal("fault original delivery")
		}
		sealed := store.sealed[ref.Name]
		o := &witnessNativeObserver{store: store}
		if r := o.receipt(); r.Failed {
			t.Fatal("healthy custody rejected")
		}
		path := filepath.Join(dir, ref.Name)
		changed := bytes.Clone(raw)
		changed[len(changed)-1] ^= 1
		if os.Chmod(path, 0600) != nil || os.WriteFile(path, changed, 0600) != nil || os.Chmod(path, 0400) != nil {
			t.Fatal("same object mutation")
		}
		r := o.receipt()
		if !r.Failed || r.Failure != "receipt-custody" {
			t.Fatal("native mutation escaped receipt")
		}
		if os.Chmod(path, 0600) != nil || os.WriteFile(path, raw, 0600) != nil || os.Chmod(path, 0400) != nil || os.Chtimes(path, sealed.info.ModTime(), sealed.info.ModTime()) != nil || store.check(sealed) != nil {
			t.Fatal("restore original bytes/object timestamp")
		}
		if r := o.receipt(); !r.Failed || r.Failure != "receipt-custody" {
			t.Fatal("restoration cleared observer rejection")
		}
	})
	t.Run("physical-tmpfs-exhaustion", func(t *testing.T) {
		nested := filepath.Join(mount, "physical-fault")
		if os.Mkdir(nested, 0700) != nil {
			t.Fatal("exclusive physical mount")
		}
		before, e := os.Stat(nested)
		if e != nil {
			t.Fatal(e)
		}
		readerFixtureCommand(t, "/sbin/mount", "-t", "tmpfs", "-o", "size=1048576,mode=0700", "tmpfs", nested)
		var fs unix.Statfs_t
		if unix.Statfs(nested, &fs) != nil || string(bytes.TrimRight(fs.Fstypename[:], "\x00")) != "tmpfs" {
			t.Fatal("physical fault mount identity")
		}
		t.Cleanup(func() {
			var current unix.Statfs_t
			if unix.Statfs(nested, &current) != nil || current.Fsid != fs.Fsid {
				t.Error("physical original mount lost")
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if exec.CommandContext(ctx, "/sbin/umount", nested).Run() != nil {
				t.Error("physical fault unmount")
				return
			}
			after, e := os.Stat(nested)
			if e != nil || !os.SameFile(before, after) || os.Remove(nested) != nil {
				t.Error("physical mount directory cleanup")
			}
		})
		dir := filepath.Join(nested, "originals")
		if os.Mkdir(dir, 0700) != nil {
			t.Fatal("exclusive physical archive")
		}
		store, e := newDashboardWitnessStore(dir)
		if e != nil {
			t.Fatal(e)
		}
		defer store.close()
		fill, e := os.OpenFile(filepath.Join(nested, "bounded-fill"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			t.Fatal(e)
		}
		var writeErr error
		var total int
		buffer := make([]byte, witnessFrameCap)
		for total <= 2<<20 {
			n, e := fill.Write(buffer)
			total += n
			if e != nil {
				writeErr = e
				break
			}
		}
		fill.Close()
		if !errors.Is(writeErr, syscall.ENOSPC) || total > 2<<20 {
			t.Fatal("physical ENOSPC not independently observed")
		}
		ref, e := store.capture(bytes.NewReader(witnessFrames(witnessGolden())), witnessTestID, "native-small", "fault-management", "fault-helper")
		if e == nil || !store.failed || ref != nil && (ref.Complete || ref.Delivered) {
			t.Fatal("physical I/O uncertainty accepted")
		}
		if _, e := store.capture(bytes.NewReader(nil), witnessTestID[:32]+"-2", "native-small", "fault-management", "new-helper"); e == nil {
			t.Fatal("physical failed candidate resumed")
		}
		if witnessWritePrivateIndex("../../.tools", "witness-physical-fault.json", struct {
			ENOSPC          bool
			FillBytes       int
			ArchiveCap      int
			TmpfsCap        int
			Eligible        bool
			OriginalIndexed bool
		}{true, total, witnessArchiveCap, 1 << 20, false, ref != nil}) != nil {
			t.Fatal("physical fault receipt")
		}
	})
	t.Log("NATIVE_WITNESS_ADDITIONAL_FAULTS same_object_mutation=denied restoration=still_denied physical_tmpfs_enospc=denied normal_caps_unchanged=true process_cancellation_complete=false")
}
