// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestNativeFreshExecutionDomainsUFS(t *testing.T) { nativeFreshExecutionDomains(t, "ufs") }
func TestNativeFreshExecutionDomainsZFS(t *testing.T) { nativeFreshExecutionDomains(t, "zfs") }
func nativeFreshExecutionDomains(t *testing.T, kind string) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("owned state VM required")
	}
	f := newRestoreFixture(t, kind)
	ctx := context.Background()
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("fresh namespace assertion", name)
		}
		markers = append(markers, name+"=pass")
	}
	paths := func(name string) (string, string) {
		return filepath.Join(f.journal, name), filepath.Join(f.ledger, name)
	}
	jp, lp := paths("positive")
	var trace []string
	p, e := provisionFreshExecutionDomains(ctx, jp, lp, func(role, stage string) error { trace = append(trace, role+":"+stage); return nil })
	check("exclusive_creation", e == nil && p != nil)
	t.Cleanup(func() { _ = p.close() })
	check("original_provenance", p.original == p && p.pair.origin == p.pair)
	check("journal_before_ledger", reflect.DeepEqual(trace, []string{"journal:mkdir", "journal:open", "journal:lock", "journal:sync-lock", "journal:sync-directory", "journal:sync-parent", "journal:after-parent-sync", "journal:verify", "ledger:mkdir", "ledger:open", "ledger:lock", "ledger:sync-lock", "ledger:sync-directory", "ledger:sync-parent", "ledger:after-parent-sync", "ledger:verify"}))
	check("repeat_fresh_inspection", p.inspect(ctx) == nil && p.inspect(ctx) == nil)
	for i, role := range []string{"journal", "ledger"} {
		s := p.pair.native.journal.lease
		if i == 1 {
			s = p.pair.native.ledger
		}
		var ds, ls unix.Stat_t
		check(role+"_private_objects", unix.Fstat(int(s.directory.Fd()), &ds) == nil && ds.Mode&07777 == 0700 && unix.Fstat(int(s.writer.Fd()), &ls) == nil && validStateLock(ls))
		flag, err := unix.FcntlInt(s.writer.Fd(), unix.F_GETFD, 0)
		check(role+"_cloexec", err == nil && flag&unix.FD_CLOEXEC != 0)
		q, err := openStateDirectory([]string{jp, lp}[i], false)
		if q != nil {
			_ = q.Close()
		}
		check(role+"_original_lock", q == nil && errors.Is(err, ErrStateLocked))
	}
	observed, e := p.pair.Observe(ctx, strings.Repeat("1", 32), f.read)
	check("native_binding_observed", e == nil && observed != (ExecutionDomainsObservation{}))
	check("no_journal_or_receipt", !observed.journalPresent && !p.pair.native.journal.acknowledged && !p.pair.native.journal.initializable)
	policy, e := DecodeOwnershipRecord(ownershipGolden())
	if e != nil {
		t.Fatal(e)
	}
	check("metadata_writer_denied", errors.Is(p.pair.publishJournaledOwnership(ctx, policy, 0, f.read, nil), ErrDenied))
	copied := &freshExecutionDomains{original: p, pair: p.pair, anchors: p.anchors}
	check("copied_inspection_denied", errors.Is(copied.inspect(ctx), ErrDenied))
	check("copied_close_denied", errors.Is(copied.close(), ErrDenied) && p.inspect(ctx) == nil)
	check("nil_inspection_denied", errors.Is((*freshExecutionDomains)(nil).inspect(ctx), ErrDenied))
	check("nil_context_denied", errors.Is(p.inspect(nil), ErrDenied))
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	check("cancelled_inspection_denied", errors.Is(p.inspect(cancelled), context.Canceled) && p.inspect(ctx) == nil)

	j, l := paths("context")
	q, err := provisionFreshExecutionDomains(nil, j, l, nil)
	check("nil_creation_context_denied", q == nil && err != nil)
	q, err = provisionFreshExecutionDomains(cancelled, j, l, nil)
	check("cancelled_creation_denied", q == nil && errors.Is(err, context.Canceled) && !freshPathExists(j) && !freshPathExists(l))
	for _, change := range []string{"directory_collision", "lock_collision"} {
		j, l := paths(change)
		q, e := provisionFreshExecutionDomains(ctx, j, l, func(role, stage string) error {
			if role == "journal" && change == "directory_collision" && stage == "open" {
				if os.Rename(j, j+"-original") != nil || os.Mkdir(j, 0700) != nil {
					t.Fatal("owned creation directory collision")
				}
			}
			if role == "journal" && change == "lock_collision" && stage == "lock" {
				if os.WriteFile(filepath.Join(j, stateWriterName), []byte("sentinel"), 0600) != nil {
					t.Fatal("owned creation lock collision")
				}
			}
			return nil
		})
		check(change+"_denied", q == nil && e != nil)
		check(change+"_ledger_absent", !freshPathExists(l))
		if change == "lock_collision" {
			data, e := os.ReadFile(filepath.Join(j, stateWriterName))
			check("lock_collision_preserved", e == nil && string(data) == "sentinel")
		}
	}
	link := filepath.Join(f.journal, "parent-link")
	if os.Symlink(f.ledger, link) != nil {
		t.Fatal("owned symlink parent")
	}
	q, err = provisionFreshExecutionDomains(ctx, filepath.Join(link, "journal"), filepath.Join(f.ledger, "link-ledger"), nil)
	check("symlink_parent_denied", q == nil && err != nil && !freshPathExists(filepath.Join(f.ledger, "journal")) && !freshPathExists(filepath.Join(f.ledger, "link-ledger")))
	for _, tc := range []struct{ name, j, l string }{
		{"relative", "relative", lp + "x"}, {"unclean", jp + "/../x", lp + "x"}, {"same", jp + "x", jp + "x"}, {"nested", jp + "x", jp + "x/ledger"}, {"reverse_nested", lp + "x/journal", lp + "x"}, {"root", "/", lp + "x"}, {"control", jp + "\n", lp + "x"},
	} {
		q, e := provisionFreshExecutionDomains(ctx, tc.j, tc.l, nil)
		check(tc.name+"_path_denied", q == nil && e != nil)
	}
	for _, which := range []string{"journal", "ledger", "both"} {
		j, l := paths("existing-" + which)
		for i, path := range []string{j, l} {
			if which == "both" || which == []string{"journal", "ledger"}[i] {
				if os.Mkdir(path, 0700) != nil {
					t.Fatal("owned existing directory")
				}
			}
		}
		beforeJ, beforeL := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
		q, e := provisionFreshExecutionDomains(ctx, j, l, nil)
		check("existing_"+which+"_denied", q == nil && e != nil)
		check("existing_"+which+"_preserved", reflect.DeepEqual(beforeJ, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(beforeL, freshNamespaceSnapshot(t, l)))
	}
	for _, role := range []string{"journal", "ledger"} {
		for _, stage := range []string{"mkdir", "open", "lock", "sync-lock", "sync-directory", "sync-parent", "after-parent-sync", "verify"} {
			j, l := paths("failure-" + role + "-" + stage)
			q, e := provisionFreshExecutionDomains(ctx, j, l, func(r, s string) error {
				if r == role && s == stage {
					return ErrDenied
				}
				return nil
			})
			check(role+"_"+stage+"_failure", q == nil && e != nil)
			beforeJ, beforeL := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
			if role != "journal" || stage != "mkdir" {
				q, e = provisionFreshExecutionDomains(ctx, j, l, nil)
				check(role+"_"+stage+"_retry_preserved", q == nil && e != nil && reflect.DeepEqual(beforeJ, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(beforeL, freshNamespaceSnapshot(t, l)))
			}
		}
	}
	for _, change := range []string{"same_inode", "lock_replaced", "directory_replaced", "late_record"} {
		j, l := paths("drift-" + change)
		q, e := provisionFreshExecutionDomains(ctx, j, l, nil)
		if e != nil {
			t.Fatal(e)
		}
		lock := filepath.Join(l, stateWriterName)
		switch change {
		case "same_inode":
			if os.WriteFile(lock, []byte("x"), 0600) != nil || os.Truncate(lock, 0) != nil {
				t.Fatal("owned same inode change")
			}
		case "lock_replaced":
			if os.Remove(lock) != nil || os.WriteFile(lock, nil, 0600) != nil {
				t.Fatal("owned lock replacement")
			}
		case "directory_replaced":
			if os.Rename(l, l+"-original") != nil || os.Mkdir(l, 0700) != nil {
				t.Fatal("owned directory replacement")
			}
		case "late_record":
			if os.WriteFile(filepath.Join(j, "foreign"), []byte("sentinel"), 0600) != nil {
				t.Fatal("owned late entry")
			}
		}
		beforeJ, beforeL := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
		check(change+"_denied", q.inspect(ctx) != nil && q.blocked)
		check(change+"_retired", q.inspect(ctx) != nil && reflect.DeepEqual(beforeJ, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(beforeL, freshNamespaceSnapshot(t, l)))
		check(change+"_close", q.close() == nil)
	}
	for _, role := range []string{"journal", "ledger"} {
		j, l := paths("unsafe-" + role)
		parent := filepath.Dir(j)
		if role == "ledger" {
			parent = filepath.Dir(l)
		}
		if os.Chmod(parent, 0770) != nil {
			t.Fatal("owned unsafe parent")
		}
		q, e := provisionFreshExecutionDomains(ctx, j, l, nil)
		check(role+"_unsafe_parent_denied", q == nil && e != nil)
		check(role+"_unsafe_no_creation", !freshPathExists(j) && !freshPathExists(l))
		if os.Chmod(parent, 0700) != nil {
			t.Fatal("owned parent restore")
		}
	}
	check("original_close", p.close() == nil)
	check("closed_inspection_denied", p.inspect(ctx) != nil)
	check("repeat_close", p.close() == nil)
	reopened, e := OpenExecutionDomainLeases(ctx, jp, lp)
	check("public_reopen_observation", e == nil && reopened != nil && !reopened.native.journal.acknowledged && !reopened.native.journal.initializable)
	check("public_reopen_close", reopened.Close() == nil)
	t.Log("FRESH_EXECUTION_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
}
func freshPathExists(path string) bool { _, e := os.Lstat(path); return !os.IsNotExist(e) }

func TestFreshExecutionCutChild(t *testing.T) {
	j, l, mode := os.Getenv("HORVIUM_FRESH_JOURNAL"), os.Getenv("HORVIUM_FRESH_LEDGER"), os.Getenv("HORVIUM_FRESH_CUT")
	if mode == "" {
		t.Skip("owned fresh reexec only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("owned root VM")
	}
	if mode == "uid" {
		if unix.Setresuid(1, 1, 1) != nil {
			t.Fatal("kernel UID drop")
		}
		q, e := provisionFreshExecutionDomains(context.Background(), j, l, nil)
		if q != nil || e == nil {
			t.Fatal("nonroot provisioned")
		}
		fmt.Println("FRESH_CUT_REACHED uid")
		return
	}
	_, e := provisionFreshExecutionDomains(context.Background(), j, l, func(role, stage string) error {
		if role == mode && stage == "after-parent-sync" {
			fmt.Println("FRESH_CUT_REACHED " + mode)
			_, _ = io.Copy(io.Discard, os.Stdin)
			return ErrDenied
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	t.Fatal("unexpected provisioning completion")
}
func TestNativeFreshExecutionCuts(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("owned state VM required")
	}
	f := newRestoreFixture(t, "ufs")
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("fresh cut", name)
		}
		markers = append(markers, name+"=pass")
	}
	for _, mode := range []string{"journal", "ledger", "uid"} {
		j, l := filepath.Join(f.journal, mode), filepath.Join(f.ledger, mode)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestFreshExecutionCutChild$")
		cmd.Env = append(os.Environ(), "HORVIUM_FRESH_JOURNAL="+j, "HORVIUM_FRESH_LEDGER="+l, "HORVIUM_FRESH_CUT="+mode)
		in, e := cmd.StdinPipe()
		if e != nil {
			t.Fatal(e)
		}
		out, e := cmd.StdoutPipe()
		if e != nil {
			t.Fatal(e)
		}
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if cmd.Start() != nil {
			t.Fatal("owned fresh child")
		}
		func() {
			defer cancel()
			defer in.Close()
			defer func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			scan := bufio.NewScanner(out)
			reached := false
			for scan.Scan() {
				if scan.Text() == "FRESH_CUT_REACHED "+mode {
					reached = true
					break
				}
			}
			check(mode+"_barrier", reached)
			if mode != "uid" {
				_ = cmd.Process.Kill()
				e = cmd.Wait()
				var ee *exec.ExitError
				check(mode+"_killed_reaped", errors.As(e, &ee) && ee.ProcessState.Sys().(syscall.WaitStatus).Signal() == syscall.SIGKILL)
			} else {
				check("uid_child_reaped", cmd.Wait() == nil)
			}
			beforeJ, beforeL := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
			if mode != "uid" {
				q, e := provisionFreshExecutionDomains(context.Background(), j, l, nil)
				check(mode+"_no_resume", q == nil && e != nil)
				check(mode+"_partial_preserved", reflect.DeepEqual(beforeJ, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(beforeL, freshNamespaceSnapshot(t, l)))
			} else {
				check("uid_no_creation", !freshPathExists(j) && !freshPathExists(l))
			}
		}()
	}
	t.Log("FRESH_EXECUTION_CUTS_RESULT " + strings.Join(markers, " "))
}

func freshNamespaceSnapshot(t *testing.T, path string) map[string]restoreObjectSnapshot {
	t.Helper()
	if _, e := os.Lstat(path); os.IsNotExist(e) {
		return nil
	}
	return restoreJournalSnapshot(t, path)
}
