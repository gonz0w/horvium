// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

var restartFaults = []string{"entropy-short", "old-identity", "installation-identity", "boot-before", "system-before", "ledger-before", "journal-before", "write", "syncFile", "closeFile", "verify", "rename", "syncDirectory", "after-sync-drift", "boot-after", "system-after", "journal-after", "ledger-after", "cancel-after", "lost-ack", "after-sync-directory", "after-sync-ledger", "after-sync-lock"}

func restartStorageGuard(t *testing.T) {
	t.Helper()
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" || os.Getenv("HORVIUM_RESTART_VM") != "1" {
		t.Skip("owned restart fixture only")
	}
	if os.Geteuid() != 0 {
		t.Fatal("root required")
	}
}
func restartSeed(t *testing.T, f restoreFixture, name string) (string, string, []byte) {
	t.Helper()
	ctx := context.Background()
	use := activationSystemFixture(t, f.journal, f.ledger, f.read)
	sys := func(context.Context) (PoolSystemUse, error) { return use, nil }
	j, l := filepath.Join(f.journal, name), filepath.Join(f.ledger, name)
	a, e := activateFreshExecutionJournal(ctx, j, l, f.read, sys, activationBoot)
	if e != nil {
		t.Fatal("seed", e)
	}
	if a.finishIdle(ctx, nil) != nil || a.close() != nil {
		t.Fatal("closed seed")
	}
	b, e := os.ReadFile(filepath.Join(j, executionJournalName))
	if e != nil {
		t.Fatal(e)
	}
	return j, l, b
}
func TestNativeClosedStartupPublicationUFS(t *testing.T) { nativeClosedStartupPublication(t, "ufs") }
func TestNativeClosedStartupPublicationZFS(t *testing.T) { nativeClosedStartupPublication(t, "zfs") }
func nativeClosedStartupPublication(t *testing.T, kind string) {
	restartStorageGuard(t)
	f := newRestoreFixture(t, kind)
	ctx := context.Background()
	use := activationSystemFixture(t, f.journal, f.ledger, f.read)
	sys := func(context.Context) (PoolSystemUse, error) { return use, nil }
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("restart", name)
		}
		for _, old := range seen {
			if old == name+"=pass" {
				t.Fatal("duplicate")
			}
		}
		seen = append(seen, name+"=pass")
	}
	j, l, before := restartSeed(t, f, "positive")
	r, n, e := reviewExecutionStartup(ctx, j, l, f.read, sys, activationBoot)
	check("original_closed_review", e == nil && r != nil && n != nil && !n.pair.native.journal.acknowledged)
	defer func() {
		if r != nil {
			_ = r.Close()
		}
	}()
	old, _ := decodeExecutionJournal(before)
	ledgerBefore := freshNamespaceSnapshot(t, l)
	var phases []string
	r.activate = func(c context.Context) (*ExecutionRestartActivation, error) {
		return n.activate(c, rand.Reader, func(tx ownershipPublication) ownershipPublication {
			sf, sd := tx.syncFile, tx.syncDirectory
			tx.syncFile = func() error {
				e := sf()
				if e == nil {
					phases = append(phases, "file")
				}
				return e
			}
			tx.syncDirectory = func() error {
				e := sd()
				if e == nil {
					phases = append(phases, "directory")
				}
				return e
			}
			return tx
		})
	}
	a, e := r.Activate(ctx)
	if a != nil {
		defer a.Close()
	}
	check("original_active_receipt", e == nil && a != nil && n.pair.native.journal.acknowledged)
	check("real_sync_order", reflect.DeepEqual(phases, []string{"file", "directory"}))
	data, e := os.ReadFile(filepath.Join(j, executionJournalName))
	record, de := decodeExecutionJournal(data)
	check("exact_new_incarnation", e == nil && de == nil && record.Generation == old.Generation+1 && record.Binding == old.Binding && record.Census == old.Census && record.Owner.State == "active" && record.Owner.ID != old.Owner.ID && record.Owner.ID != old.Binding.Installation && record.Owner.Boot == old.Owner.Boot && !record.Disarmed && !record.Metadata.Pending && !record.Native.Pending)
	check("ledger_unchanged", reflect.DeepEqual(ledgerBefore, freshNamespaceSnapshot(t, l)))
	check("old_review_consumed", r.Close() != nil && func() bool { _, e := r.Activate(ctx); return e != nil }() && func() bool { _, e := r.Observe(ctx); return e != nil }())
	check("both_leases_retained", restartLocksHeld(t, j, l))
	_, e = a.Observe(ctx)
	check("original_observation", e == nil)
	_, e = a.Observe(ctx)
	check("repeated_observation", e == nil)
	copy := new(ExecutionRestartActivation)
	reflect.ValueOf(copy).Elem().Set(reflect.ValueOf(a).Elem())
	_, e = copy.Observe(ctx)
	check("copied_denied", e != nil && copy.Close() != nil)
	wire, e := json.Marshal(a)
	var decoded ExecutionRestartActivation
	de = json.Unmarshal(wire, &decoded)
	_, ce := decoded.Observe(ctx)
	check("decoded_denied", e == nil && de == nil && string(wire) == "{}" && ce != nil && decoded.Close() != nil)
	cancelled, stop := context.WithCancel(ctx)
	stop()
	_, e = a.Observe(cancelled)
	check("cancelled_no_effect", e != nil)
	_, e = a.Observe(ctx)
	check("cancelled_original_survives", e == nil)
	check("release_once", a.Close() == nil && a.Close() == nil)
	check("released_leases", restartLocksReleased(t, j, l))
	p, e := OpenExecutionDomainLeases(ctx, j, l)
	check("reopen_observation_only", e == nil && p != nil && !p.native.journal.acknowledged)
	if p != nil {
		check("ordinary_successor_denied", p.native.journal.publish(record, nil) != nil)
		check("ordinary_reopen_release", p.Close() == nil)
	}
	q, e := ReviewExecutionStartup(ctx, j, l, f.read, sys, activationBoot)
	check("active_no_takeover", e != nil && q == nil)
	after, e := os.ReadFile(filepath.Join(j, executionJournalName))
	check("active_preserved", e == nil && bytes.Equal(after, data))
	for _, mode := range restartFaults {
		j, l, b := restartSeed(t, f, "failure-"+mode)
		r, n, e := reviewExecutionStartup(ctx, j, l, f.read, sys, activationBoot)
		if e != nil {
			t.Fatal(e)
		}
		defer func() { _ = r.Close() }()
		entropy := io.Reader(rand.Reader)
		record, _ := decodeExecutionJournal(b)
		if mode == "entropy-short" {
			entropy = strings.NewReader("short")
		}
		if mode == "old-identity" || mode == "installation-identity" {
			identity := record.Owner.ID
			if mode == "installation-identity" {
				identity = record.Binding.Installation
			}
			raw, e := hex.DecodeString(identity)
			if e != nil {
				t.Fatal(e)
			}
			entropy = bytes.NewReader(raw)
		}
		c, cancel := context.WithCancel(ctx)
		bootCount, sysCount := 0, 0
		n.boot = func(c context.Context) ([16]byte, error) {
			bootCount++
			v, e := activationBoot(c)
			if (mode == "boot-before" && bootCount == 1) || (mode == "boot-after" && n.pair.native.journal.acknowledged) {
				v[0] ^= 1
			}
			return v, e
		}
		n.system = func(c context.Context) (PoolSystemUse, error) {
			sysCount++
			if (mode == "system-before" && sysCount == 1) || (mode == "system-after" && n.pair.native.journal.acknowledged) {
				return PoolSystemUse{}, ErrUnavailable
			}
			if n.pair.native.journal.acknowledged {
				switch mode {
				case "journal-after":
					_ = os.Chmod(j, 0700)
				case "ledger-after":
					_ = os.Chmod(l, 0700)
				case "cancel-after":
					cancel()
				}
			}
			return use, nil
		}
		if mode == "ledger-before" {
			if os.Chmod(l, 0700) != nil {
				t.Fatal("touch")
			}
		}
		if mode == "journal-before" {
			if os.Chmod(filepath.Join(j, executionJournalName), 0600) != nil {
				t.Fatal("touch")
			}
		}
		wrap := func(tx ownershipPublication) ownershipPublication {
			switch mode {
			case "write":
				tx.write = func() error { return ErrDenied }
			case "syncFile":
				tx.syncFile = func() error { return ErrDenied }
			case "closeFile":
				old := tx.closeFile
				tx.closeFile = func() error { _ = old(); return ErrDenied }
			case "verify":
				tx.verify = func() error { return ErrDenied }
			case "rename":
				tx.rename = func() error { return ErrDenied }
			case "syncDirectory":
				tx.syncDirectory = func() error { return ErrDenied }
			case "lost-ack", "after-sync-directory", "after-sync-ledger", "after-sync-lock":
				old := tx.syncDirectory
				tx.syncDirectory = func() error {
					if e := old(); e != nil {
						return e
					}
					switch mode {
					case "lost-ack":
						return ErrDenied
					case "after-sync-directory":
						return os.Chmod(j, 0700)
					case "after-sync-ledger":
						return os.Chmod(l, 0700)
					default:
						return os.Chmod(filepath.Join(j, stateWriterName), 0600)
					}
				}
			case "after-sync-drift":
				old := tx.syncDirectory
				tx.syncDirectory = func() error {
					if e := old(); e != nil {
						return e
					}
					return os.Chmod(filepath.Join(j, executionJournalName), 0600)
				}
			}
			return tx
		}
		r.activate = func(c context.Context) (*ExecutionRestartActivation, error) { return n.activate(c, entropy, wrap) }
		got, e := r.Activate(c)
		check(mode+"_denied", e != nil && got == nil)
		check(mode+"_retired", n.pair.blocked && n.pair.native.journal.failed && n.pair.native.ledger.publicationBlocked)
		bj, bl := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
		_, e = r.Activate(ctx)
		check(mode+"_retry_preserves", e != nil && reflect.DeepEqual(bj, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, l)))
		check(mode+"_release", r.Close() == nil && restartLocksReleased(t, j, l))
		cancel()
	}

	for _, mode := range []string{"nonempty", "no-close-capacity"} {
		j, l, b := restartSeed(t, f, "candidate-"+mode)
		record, e := decodeExecutionJournal(b)
		if e != nil {
			t.Fatal(e)
		}
		if mode == "nonempty" {
			state, e := openStateDirectory(l, false)
			if e != nil {
				t.Fatal(e)
			}
			if state.PublishOwnership(journaledPolicyFixture(t), 0) != nil {
				_ = state.Close()
				t.Fatal("controlled ownership fixture")
			}
			_, census, e := state.inspectExecutionCensus(record.Binding, defaultStateCatalogLimits, nil)
			_ = state.Close()
			if e != nil {
				t.Fatal(e)
			}
			record.Census = census
		} else {
			record.Generation = math.MaxUint64 - 1
		}
		data, e := encodeExecutionJournal(record)
		if e != nil || os.WriteFile(filepath.Join(j, executionJournalName), data, 0600) != nil {
			t.Fatal("controlled closed fixture")
		}
		r, n, e := reviewExecutionStartup(ctx, j, l, f.read, sys, activationBoot)
		check(mode+"_review_metadata_only", e == nil && r != nil)
		if r != nil {
			defer r.Close()
		}
		bj, bl := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
		a, e := r.Activate(ctx)
		check(mode+"_activation_denied", e != nil && a == nil)
		check(mode+"_retired", n.pair.blocked && n.pair.native.journal.failed && n.pair.native.ledger.publicationBlocked)
		check(mode+"_history_preserved", reflect.DeepEqual(bj, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, l)))
		check(mode+"_release", r.Close() == nil && restartLocksReleased(t, j, l))
	}
	t.Log("CLOSED_STARTUP_PUBLICATION_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(seen, " "))
}
func restartLocksHeld(t *testing.T, j, l string) bool {
	t.Helper()
	held := true
	for _, path := range []string{j, l} {
		s, e := openStateDirectory(path, false)
		if s != nil {
			_ = s.Close()
		}
		held = held && s == nil && errors.Is(e, ErrStateLocked)
	}
	return held
}

// The exact child pauses only after the actual declared sync/acknowledgement.
func TestClosedStartupPublicationCutChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_RESTART_CUT")
	if mode == "" {
		t.Skip("owned cut child only")
	}
	restartStorageGuard(t)
	j, l := os.Getenv("HORVIUM_RESTART_JOURNAL"), os.Getenv("HORVIUM_RESTART_LEDGER")
	read := func(c context.Context, fd int) (ExecutionBackingObservation, error) {
		var fs unix.Statfs_t
		if c.Err() != nil || unix.Fstatfs(fd, &fs) != nil || unix.ByteSliceToString(fs.Fstypename[:]) != "ufs" {
			return ExecutionBackingObservation{}, ErrDenied
		}
		return ExecutionBackingObservation{Kind: "ufs", UFS: [2]uint32{uint32(fs.Fsid.Val[0]), uint32(fs.Fsid.Val[1])}}, nil
	}
	use := activationSystemFixture(t, filepath.Dir(j), filepath.Dir(l), read)
	sys := func(context.Context) (PoolSystemUse, error) { return use, nil }
	r, n, e := reviewExecutionStartup(context.Background(), j, l, read, sys, activationBoot)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	barrier := func() {
		fmt.Println("CLOSED_STARTUP_CUT_REACHED " + mode)
		_, _ = io.Copy(io.Discard, os.Stdin)
		t.Fatal("cut continued")
	}
	if mode == "uid" {
		if unix.Setgroups([]int{}) != nil || unix.Setgid(65534) != nil || unix.Setuid(65534) != nil || os.Geteuid() != 65534 {
			t.Fatal("drop")
		}
		for _, s := range []*StateDirectory{n.pair.native.journal.lease, n.pair.native.ledger} {
			var st unix.Stat_t
			if unix.Fstat(int(s.directory.Fd()), &st) != nil || st.Uid != 0 {
				t.Fatal("inherited root")
			}
		}
		if _, e := r.Activate(context.Background()); !errors.Is(e, ErrDenied) {
			t.Fatal("nonroot activation")
		}
		barrier()
	}
	wrap := func(tx ownershipPublication) ownershipPublication {
		if mode == "file-sync" {
			old := tx.syncFile
			tx.syncFile = func() error {
				if e := old(); e != nil {
					return e
				}
				barrier()
				return ErrDenied
			}
		}
		if mode == "directory-sync" {
			old := tx.syncDirectory
			tx.syncDirectory = func() error {
				if e := old(); e != nil {
					return e
				}
				barrier()
				return ErrDenied
			}
		}
		return tx
	}
	r.activate = func(c context.Context) (*ExecutionRestartActivation, error) { return n.activate(c, rand.Reader, wrap) }
	a, e := r.Activate(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	if mode == "acknowledged" {
		barrier()
	}
	t.Fatal("unknown mode")
}
func TestNativeClosedStartupPublicationCuts(t *testing.T) {
	restartStorageGuard(t)
	f := newRestoreFixture(t, "ufs")
	var seen []string
	check := func(n string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("restart cut", n)
		}
		seen = append(seen, n+"=pass")
	}
	for _, mode := range []string{"file-sync", "directory-sync", "acknowledged", "uid"} {
		j, l, b := restartSeed(t, f, "cut-"+mode)
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestClosedStartupPublicationCutChild$", "-test.timeout=45s")
		cmd.Env = append(os.Environ(), "HORVIUM_RESTART_CUT="+mode, "HORVIUM_RESTART_JOURNAL="+j, "HORVIUM_RESTART_LEDGER="+l)
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
			t.Fatal("start")
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
				if scan.Text() == "CLOSED_STARTUP_CUT_REACHED "+mode {
					reached = true
					break
				}
			}
			if !reached {
				t.Log("child stderr", stderr.String())
			}
			check(mode+"_barrier", reached)
			_ = cmd.Process.Kill()
			e = cmd.Wait()
			var ee *exec.ExitError
			check(mode+"_killed_reaped", errors.As(e, &ee) && ee.ProcessState.Sys().(syscall.WaitStatus).Signal() == syscall.SIGKILL)
			data, e := os.ReadFile(filepath.Join(j, executionJournalName))
			record, de := decodeExecutionJournal(data)
			old, _ := decodeExecutionJournal(b)
			expected := "active"
			gen := old.Generation + 1
			if mode == "file-sync" || mode == "uid" {
				expected = "closed"
				gen = old.Generation
			}
			check(mode+"_visible_history", e == nil && de == nil && record.Owner.State == expected && record.Generation == gen && record.Census == old.Census && record.Binding == old.Binding)
			bj, bl := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
			use := activationSystemFixture(t, f.journal, f.ledger, f.read)
			q, er := ReviewExecutionStartup(context.Background(), j, l, f.read, func(context.Context) (PoolSystemUse, error) { return use, nil }, activationBoot)
			if mode == "uid" {
				check(mode+"_closed_review_only", er == nil && q != nil && q.Close() == nil)
			} else {
				check(mode+"_no_startup_takeover", er != nil && q == nil)
			}
			check(mode+"_history_preserved", reflect.DeepEqual(bj, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, l)))
			sj, ej := openStateDirectory(j, false)
			sl, el := openStateDirectory(l, false)
			check(mode+"_leases_released", ej == nil && el == nil && sj.Close() == nil && sl.Close() == nil)
		}()
	}
	t.Log("CLOSED_STARTUP_CUT_RESULT " + strings.Join(seen, " "))
}

func restartLocksReleased(t *testing.T, j, l string) bool {
	t.Helper()
	released := true
	for _, path := range []string{j, l} {
		s, e := openStateDirectory(path, false)
		ok := e == nil && s != nil
		if s != nil {
			ok = s.Close() == nil && ok
		}
		released = released && ok
	}
	return released
}
