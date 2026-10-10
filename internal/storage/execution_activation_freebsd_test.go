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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Native mount/identity and real file durability, with synthetic physical-use
// and boot transfers. Fixed kernel/GEOM producers are separate host gates; this
// component does not claim their assembled commissioning or runtime authority.
func activationSystemFixture(t *testing.T, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error)) PoolSystemUse {
	t.Helper()
	u := propertySystemFixture(t, "/dev/gpt/rootfs / ufs rw 1 1\n", emptySwap)
	disk := u.system.Mounts[0].Disks
	for _, path := range []string{journal, ledger} {
		file, e := os.Open(path)
		if e != nil {
			t.Fatal(e)
		}
		var fs unix.Statfs_t
		if unix.Fstatfs(int(file.Fd()), &fs) != nil {
			t.Fatal("owned activation mount")
		}
		id, e := read(context.Background(), int(file.Fd()))
		_ = file.Close()
		if e != nil {
			t.Fatal(e)
		}
		source, target, kind := unix.ByteSliceToString(fs.Mntfromname[:]), unix.ByteSliceToString(fs.Mntonname[:]), unix.ByteSliceToString(fs.Fstypename[:])
		guid := ""
		if kind == "zfs" {
			guid = strconv.FormatUint(id.Pool, 10)
		}
		duplicate := false
		for _, m := range u.system.Mounts {
			if m.Source == source && m.Target == target {
				duplicate = true
			}
		}
		if !duplicate {
			u.system.Mounts = append(u.system.Mounts, MountUse{Source: source, Target: target, Type: kind, PoolGUID: guid, Disks: disk})
		}
		if kind == "zfs" {
			found := false
			for _, p := range u.pools {
				if p.GUID == guid {
					found = true
				}
			}
			if !found {
				u.pools = append(u.pools, ImportedPoolUse{Name: strings.Split(source, "/")[0], GUID: guid, Health: "ONLINE", Disks: disk})
			}
		}
	}
	return u
}
func activationBoot(context.Context) ([16]byte, error) { return [16]byte{1, 2, 3, 4}, nil }
func TestNativeFreshJournalActivationUFS(t *testing.T) { nativeFreshJournalActivation(t, "ufs") }
func TestNativeFreshJournalActivationZFS(t *testing.T) { nativeFreshJournalActivation(t, "zfs") }
func nativeFreshJournalActivation(t *testing.T, kind string) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("owned state VM required")
	}
	f := newRestoreFixture(t, kind)
	ctx := context.Background()
	use := activationSystemFixture(t, f.journal, f.ledger, f.read)
	system := func(context.Context) (PoolSystemUse, error) { return use, nil }
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("fresh activation", name)
		}
		markers = append(markers, name+"=pass")
	}
	fresh := func(name string) *freshExecutionDomains {
		p, e := provisionFreshExecutionDomains(ctx, filepath.Join(f.journal, name), filepath.Join(f.ledger, name), nil)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { _ = p.close() })
		return p
	}
	p := fresh("positive")
	var reads, systems, boots int
	read := func(c context.Context, fd int) (ExecutionBackingObservation, error) { reads++; return f.read(c, fd) }
	sys := func(c context.Context) (PoolSystemUse, error) { systems++; return system(c) }
	boot := func(c context.Context) ([16]byte, error) { boots++; return activationBoot(c) }
	a, e := p.activate(ctx, read, sys, boot, rand.Reader, nil)
	check("initial_activation", e == nil && a != nil)
	check("original_receipt", a.original == a && a.fresh == p && p.pair.native.journal.acknowledged && p.pair.native.journal.ack == a.journal)
	check("fresh_distinct_identities", validJournalIdentity(a.journal.Binding.Installation) && validJournalIdentity(a.journal.Owner.ID) && a.journal.Binding.Installation != a.journal.Owner.ID)
	raw, _ := activationBoot(ctx)
	check("raw_boot_bound", a.journal.Owner.Boot == hex.EncodeToString(raw[:]))
	check("initial_active_record", checkInitialExecutionJournal(a.journal) == nil && a.journal.Owner.State == "active")
	check("repeated_native_system_boot", reads >= 8 && systems >= 4 && boots >= 3)
	data, e := os.ReadFile(filepath.Join(f.journal, "positive", executionJournalName))
	decoded, de := decodeExecutionJournal(data)
	check("canonical_durable_bytes", e == nil && de == nil && decoded == a.journal && bytes.Equal(data, p.pair.native.journal.ackBytes))
	var st unix.Stat_t
	check("private_journal_file", unix.Stat(filepath.Join(f.journal, "positive", executionJournalName), &st) == nil && st.Mode&07777 == 0600 && st.Nlink == 1 && st.Uid == 0)
	c, ce := newExecutionCensus(a.journal.Binding, []string{stateWriterName}, defaultStateCatalogLimits)
	if ce != nil {
		t.Fatal(ce)
	}
	empty, ce := c.finish()
	check("complete_empty_census", ce == nil && empty == a.journal.Census)
	check("activation_not_repeatable", func() bool {
		q, e := p.activate(ctx, f.read, system, activationBoot, rand.Reader, nil)
		return q == nil && errors.Is(e, ErrDenied)
	}())
	got, e := a.observe(ctx)
	check("original_observation", e == nil && got.journal == a.journal && got.systemObserved)
	got, e = a.observe(ctx)
	check("repeated_observation", e == nil && got.journal == a.journal)
	copied := &freshExecutionActivation{original: a, fresh: p, journal: a.journal}
	got, e = copied.observe(ctx)
	check("copied_observation_denied", e != nil && got == (ExecutionDomainsObservation{}))
	check("copied_close_denied", errors.Is(copied.close(), ErrDenied))
	copiedFresh := &freshExecutionDomains{original: p, pair: p.pair, anchors: p.anchors}
	q, qe := copiedFresh.activate(ctx, read, sys, boot, rand.Reader, nil)
	check("copied_fresh_denied", q == nil && qe != nil)
	serialized, we := json.Marshal(a)
	wire := string(serialized)
	check("serialized_receipt_inert", we == nil && wire == "{}")
	got, e = (*freshExecutionActivation)(nil).observe(ctx)
	check("nil_activation_denied", e != nil && got == (ExecutionDomainsObservation{}))
	got, e = a.observe(nil)
	check("nil_observation_context_denied", e != nil && got == (ExecutionDomainsObservation{}))
	cancelledObserve, cancelObserve := context.WithCancel(ctx)
	cancelObserve()
	got, e = a.observe(cancelledObserve)
	check("cancelled_observation_no_retirement", errors.Is(e, context.Canceled) && got == (ExecutionDomainsObservation{}) && !p.blocked)
	second, e := activateFreshExecutionJournal(ctx, filepath.Join(f.journal, "second"), filepath.Join(f.ledger, "second"), f.read, system, activationBoot)
	check("fixed_entropy_composition", e == nil && second != nil && second.journal.Binding.Installation != a.journal.Binding.Installation && second.journal.Owner.ID != a.journal.Owner.ID)
	check("second_close", second.close() == nil)
	beforeJ, beforeL := freshNamespaceSnapshot(t, filepath.Join(f.journal, "positive")), freshNamespaceSnapshot(t, filepath.Join(f.ledger, "positive"))
	check("close_releases_only", a.close() == nil && reflect.DeepEqual(beforeJ, freshNamespaceSnapshot(t, filepath.Join(f.journal, "positive"))) && reflect.DeepEqual(beforeL, freshNamespaceSnapshot(t, filepath.Join(f.ledger, "positive"))))
	check("repeat_close", a.close() == nil)
	got, e = a.observe(ctx)
	check("closed_observation_denied", e != nil && got == (ExecutionDomainsObservation{}))
	reopened, e := OpenExecutionDomainLeases(ctx, filepath.Join(f.journal, "positive"), filepath.Join(f.ledger, "positive"))
	if e != nil {
		t.Fatal(e)
	}
	_, e = reopened.Observe(ctx, a.journal.Binding.Installation, f.read)
	check("reopen_observation_only", e == nil && !reopened.native.journal.acknowledged && !reopened.native.journal.initializable)
	check("reopen_initializer_denied", errors.Is(reopened.native.journal.initialize(a.journal, nil), ErrDenied))
	check("reopen_close", reopened.Close() == nil)
	existing, qe := provisionFreshExecutionDomains(ctx, filepath.Join(f.journal, "positive"), filepath.Join(f.ledger, "positive"), nil)
	check("existing_active_no_freshness", existing == nil && qe != nil && reflect.DeepEqual(beforeJ, freshNamespaceSnapshot(t, filepath.Join(f.journal, "positive"))))
	for _, name := range []string{"nil-context", "cancelled", "nil-reader", "nil-system", "nil-boot", "nil-entropy"} {
		p := fresh(name)
		c := ctx
		r, s, b, en := f.read, system, activationBoot, io.Reader(rand.Reader)
		switch name {
		case "nil-context":
			c = nil
		case "cancelled":
			var cancel context.CancelFunc
			c, cancel = context.WithCancel(ctx)
			cancel()
		case "nil-reader":
			r = nil
		case "nil-system":
			s = nil
		case "nil-boot":
			b = nil
		case "nil-entropy":
			en = nil
		}
		a, e := p.activate(c, r, s, b, en, nil)
		check(name+"_denied_without_effect", a == nil && e != nil && !freshPathExists(filepath.Join(f.journal, name, executionJournalName)) && p.inspect(ctx) == nil)
	}
	for _, name := range []string{"entropy-short", "identity-collision", "boot-error", "boot-drift", "native-error", "system-error", "system-missing", "late-ledger", "prebound"} {
		p := fresh(name)
		r, s, b, en := f.read, system, activationBoot, io.Reader(rand.Reader)
		n := 0
		switch name {
		case "entropy-short":
			en = bytes.NewReader([]byte{1})
		case "identity-collision":
			en = bytes.NewReader(make([]byte, 32))
		case "boot-error":
			b = func(context.Context) ([16]byte, error) { return [16]byte{}, ErrUnavailable }
		case "boot-drift":
			b = func(c context.Context) ([16]byte, error) {
				n++
				v, _ := activationBoot(c)
				if n > 1 {
					v[0]++
				}
				return v, nil
			}
		case "native-error":
			r = func(context.Context, int) (ExecutionBackingObservation, error) {
				return ExecutionBackingObservation{}, ErrUnavailable
			}
		case "system-error":
			s = func(context.Context) (PoolSystemUse, error) { return PoolSystemUse{}, ErrUnavailable }
		case "system-missing":
			s = func(context.Context) (PoolSystemUse, error) { return PoolSystemUse{}, nil }
		case "late-ledger":
			if os.WriteFile(filepath.Join(f.ledger, name, "unknown"), []byte("preserve"), 0600) != nil {
				t.Fatal("owned late entry")
			}
		case "prebound":
			if _, e := p.pair.Observe(ctx, strings.Repeat("1", 32), f.read); e != nil {
				t.Fatal(e)
			}
		}
		a, e := p.activate(ctx, r, s, b, en, nil)
		check(name+"_denied", a == nil && e != nil && !freshPathExists(filepath.Join(f.journal, name, executionJournalName)))
		check(name+"_retired", p.blocked && p.pair.blocked && p.pair.native.journal.failed && p.pair.native.ledger.publicationBlocked)
	}
	for _, point := range []string{"write", "syncFile", "closeFile", "verify", "rename", "syncDirectory", "after-sync-drift"} {
		name := "publication-" + point
		p := fresh(name)
		wrap := func(tx ownershipPublication) ownershipPublication {
			bad := func() error { return syscall.EIO }
			switch point {
			case "write":
				tx.write = bad
			case "syncFile":
				tx.syncFile = bad
			case "closeFile":
				old := tx.closeFile
				tx.closeFile = func() error { _ = old(); return syscall.EIO }
			case "verify":
				tx.verify = bad
			case "rename":
				tx.rename = bad
			case "syncDirectory":
				tx.syncDirectory = bad
			case "after-sync-drift":
				old := tx.syncDirectory
				tx.syncDirectory = func() error {
					if e := old(); e != nil {
						return e
					}
					return os.WriteFile(filepath.Join(f.journal, name, executionJournalName), []byte("owned-drift"), 0600)
				}
			}
			return tx
		}
		a, e := p.activate(ctx, f.read, system, activationBoot, rand.Reader, wrap)
		check(name+"_denied", a == nil && e != nil)
		check(name+"_retired", p.blocked && p.pair.blocked && p.pair.native.journal.failed && p.pair.native.ledger.publicationBlocked)
		bj, bl := freshNamespaceSnapshot(t, filepath.Join(f.journal, name)), freshNamespaceSnapshot(t, filepath.Join(f.ledger, name))
		a, e = p.activate(ctx, f.read, system, activationBoot, rand.Reader, nil)
		check(name+"_retry_preserves_history", a == nil && e != nil && reflect.DeepEqual(bj, freshNamespaceSnapshot(t, filepath.Join(f.journal, name))) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, filepath.Join(f.ledger, name))))
	}
	for _, name := range []string{"post-boot-error", "post-system-error", "post-cancel", "post-directory-touch", "post-lock-touch", "post-journal-touch", "post-ledger-touch", "journal-drift", "ledger-drift", "boot-changed"} {
		p := fresh(name)
		b, s := activationBoot, system
		n := 0
		c := ctx
		cancel := func() {}
		if name == "post-boot-error" {
			b = func(c context.Context) ([16]byte, error) {
				n++
				if n == 3 {
					return [16]byte{}, ErrUnavailable
				}
				return activationBoot(c)
			}
		}
		if name == "post-system-error" {
			s = func(c context.Context) (PoolSystemUse, error) {
				n++
				if n == 3 {
					return PoolSystemUse{}, ErrUnavailable
				}
				return system(c)
			}
		}
		if name == "post-cancel" {
			c, cancel = context.WithCancel(ctx)
			s = func(c context.Context) (PoolSystemUse, error) {
				n++
				if n == 3 {
					cancel()
				}
				return system(c)
			}
		}
		if strings.HasPrefix(name, "post-") && strings.HasSuffix(name, "-touch") {
			b = func(c context.Context) ([16]byte, error) {
				n++
				if n == 3 {
					path := filepath.Join(f.journal, name)
					switch name {
					case "post-lock-touch":
						path = filepath.Join(path, stateWriterName)
					case "post-journal-touch":
						path = filepath.Join(path, executionJournalName)
					case "post-ledger-touch":
						path = filepath.Join(f.ledger, name)
					}
					if os.Chtimes(path, time.Unix(1, 0), time.Unix(1, 0)) != nil {
						t.Fatal("owned late touch")
					}
				}
				return activationBoot(c)
			}
		}
		a, e := p.activate(c, f.read, s, func(c context.Context) ([16]byte, error) { return b(c) }, rand.Reader, nil)
		cancel()
		if strings.HasPrefix(name, "post-") {
			check(name+"_denied_after_ack", a == nil && e != nil && p.pair.native.journal.acknowledged)
		} else {
			if e != nil {
				t.Fatal(e)
			}
			switch name {
			case "journal-drift":
				if os.WriteFile(filepath.Join(f.journal, name, executionJournalName), []byte("owned-drift"), 0600) != nil {
					t.Fatal("drift")
				}
			case "ledger-drift":
				if os.WriteFile(filepath.Join(f.ledger, name, "unknown"), []byte("preserve"), 0600) != nil {
					t.Fatal("drift")
				}
			case "boot-changed":
				b = func(context.Context) ([16]byte, error) { return [16]byte{9}, nil }
			}
			bj, bl := freshNamespaceSnapshot(t, filepath.Join(f.journal, name)), freshNamespaceSnapshot(t, filepath.Join(f.ledger, name))
			got, e := a.observe(ctx)
			check(name+"_denied_preserves_history", e != nil && got == (ExecutionDomainsObservation{}) && reflect.DeepEqual(bj, freshNamespaceSnapshot(t, filepath.Join(f.journal, name))) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, filepath.Join(f.ledger, name))))
		}
		check(name+"_retired", p.blocked && p.pair.blocked && p.pair.native.journal.failed && p.pair.native.ledger.publicationBlocked)
	}
	if kind == "zfs" {
		pool := filepath.Base(filepath.Dir(f.journal))
		for _, role := range []string{"journal", "ledger"} {
			name := "ancestor-" + role
			base, other := f.journal, f.ledger
			parentRole := "ledger"
			if role == "ledger" {
				base, other = f.ledger, f.journal
				parentRole = "journal"
			}
			parent := filepath.Join(base, name+"-parent")
			dataset := pool + "/" + parentRole + "/" + name
			restoreNativeCommand(t, "/sbin/zfs", "create", "-o", "mountpoint="+parent, dataset)
			if os.Chmod(parent, 0700) != nil {
				t.Fatal("owned ancestor mode")
			}
			read := func(c context.Context, fd int) (ExecutionBackingObservation, error) {
				var fs unix.Statfs_t
				if c.Err() != nil {
					return ExecutionBackingObservation{}, c.Err()
				}
				if unix.Fstatfs(fd, &fs) != nil {
					return ExecutionBackingObservation{}, ErrDrift
				}
				if unix.ByteSliceToString(fs.Mntfromname[:]) != dataset {
					return f.read(c, fd)
				}
				guid, e := strconv.ParseUint(restoreNativeCommand(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value", "guid", dataset), 10, 64)
				if e != nil {
					return ExecutionBackingObservation{}, ErrUnavailable
				}
				file, e := os.Open(other)
				if e != nil {
					return ExecutionBackingObservation{}, e
				}
				id, e := f.read(c, int(file.Fd()))
				_ = file.Close()
				if e != nil {
					return ExecutionBackingObservation{}, e
				}
				return ExecutionBackingObservation{Kind: "zfs", Pool: id.Pool, Dataset: guid}, nil
			}
			j, l := filepath.Join(parent, "new"), filepath.Join(other, name)
			if role == "ledger" {
				j, l = l, j
			}
			p, e := provisionFreshExecutionDomains(ctx, j, l, nil)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { _ = p.close() })
			use := activationSystemFixture(t, j, l, read)
			a, e := p.activate(ctx, read, func(context.Context) (PoolSystemUse, error) { return use, nil }, activationBoot, rand.Reader, nil)
			check(name+"_denied", a == nil && errors.Is(e, ErrDenied) && !freshPathExists(filepath.Join(j, executionJournalName)))
			check(name+"_retired", p.blocked && p.pair.native.journal.failed && p.pair.native.ledger.publicationBlocked)
			if p.close() != nil {
				t.Fatal("owned ancestor close")
			}
		}
	}
	nativeInitialRetirement(t, f, check)
	t.Log("FRESH_ACTIVATION_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(markers, " "))
}

// The child stops only after the actual syscall boundary or original receipt.
// Reexec is a disposable fixture; it is not a helper/startup implementation.
func TestFreshActivationCutChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_ACTIVATION_CUT")
	if mode == "" {
		t.Skip("owned activation reexec only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("owned root VM required")
	}
	j, l := os.Getenv("HORVIUM_ACTIVATION_JOURNAL"), os.Getenv("HORVIUM_ACTIVATION_LEDGER")
	p, e := provisionFreshExecutionDomains(context.Background(), j, l, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer p.close()
	read := func(c context.Context, fd int) (ExecutionBackingObservation, error) {
		var fs unix.Statfs_t
		if c.Err() != nil {
			return ExecutionBackingObservation{}, c.Err()
		}
		if unix.Fstatfs(fd, &fs) != nil || unix.ByteSliceToString(fs.Fstypename[:]) != "ufs" {
			return ExecutionBackingObservation{}, ErrDrift
		}
		return ExecutionBackingObservation{Kind: "ufs", UFS: [2]uint32{uint32(fs.Fsid.Val[0]), uint32(fs.Fsid.Val[1])}}, nil
	}
	use := activationSystemFixture(t, j, l, read)
	system := func(context.Context) (PoolSystemUse, error) { return use, nil }
	barrier := func() {
		fmt.Println("ACTIVATION_CUT_REACHED " + mode)
		_, _ = io.Copy(io.Discard, os.Stdin)
		t.Fatal("unexpected cut continuation")
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
	a, e := p.activate(context.Background(), read, system, activationBoot, rand.Reader, wrap)
	if e != nil {
		t.Fatal(e)
	}
	if mode == "acknowledged" && a != nil {
		barrier()
	}
	t.Fatal("missing activation cut")
}
func TestNativeFreshJournalActivationCuts(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("owned state VM required")
	}
	f := newRestoreFixture(t, "ufs")
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("activation cut", name)
		}
		markers = append(markers, name+"=pass")
	}
	for _, mode := range []string{"file-sync", "directory-sync", "acknowledged"} {
		j, l := filepath.Join(f.journal, mode), filepath.Join(f.ledger, mode)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestFreshActivationCutChild$")
		cmd.Env = append(os.Environ(), "HORVIUM_ACTIVATION_CUT="+mode, "HORVIUM_ACTIVATION_JOURNAL="+j, "HORVIUM_ACTIVATION_LEDGER="+l)
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
			t.Fatal("owned cut child")
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
				if scan.Text() == "ACTIVATION_CUT_REACHED "+mode {
					reached = true
					break
				}
			}
			check(mode+"_barrier", reached)
			_ = cmd.Process.Kill()
			e = cmd.Wait()
			var ee *exec.ExitError
			check(mode+"_killed_reaped", errors.As(e, &ee) && ee.ProcessState.Sys().(syscall.WaitStatus).Signal() == syscall.SIGKILL)
			bj, bl := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
			p, e := provisionFreshExecutionDomains(context.Background(), j, l, nil)
			check(mode+"_no_recommission", p == nil && e != nil)
			check(mode+"_history_preserved", reflect.DeepEqual(bj, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, l)))
			if mode == "file-sync" {
				check(mode+"_no_active_record", !freshPathExists(filepath.Join(j, executionJournalName)))
			} else {
				p, e := OpenExecutionDomainLeases(context.Background(), j, l)
				if e != nil {
					t.Fatal(e)
				}
				record, e := p.native.journal.read(nil)
				check(mode+"_active_without_receipt", e == nil && record.Owner.State == "active" && !p.native.journal.acknowledged && !p.native.journal.initializable)
				check(mode+"_no_reconstructed_initializer", errors.Is(p.native.journal.initialize(record, nil), ErrDenied))
				check(mode+"_close", p.Close() == nil)
			}
		}()
	}
	nativeInitialRetirementCuts(t, f, check)
	t.Log("FRESH_ACTIVATION_CUTS_RESULT " + strings.Join(markers, " "))
}
