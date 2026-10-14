// SPDX-License-Identifier: BSD-3-Clause
//go:build freebsd

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

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
