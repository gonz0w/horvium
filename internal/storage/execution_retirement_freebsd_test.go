// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
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

func nativeInitialRetirement(t *testing.T, f restoreFixture, check func(string, bool)) {
	t.Helper()
	ctx := context.Background()
	use := activationSystemFixture(t, f.journal, f.ledger, f.read)
	sys := func(context.Context) (PoolSystemUse, error) { return use, nil }
	fresh := func(name string, system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error)) *freshExecutionActivation {
		p, e := provisionFreshExecutionDomains(ctx, filepath.Join(f.journal, name), filepath.Join(f.ledger, name), nil)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { _ = p.close() })
		a, e := p.activate(ctx, f.read, system, boot, rand.Reader, nil)
		if e != nil {
			t.Fatal(e)
		}
		return a
	}
	a := fresh("retirement-positive", sys, activationBoot)
	before := a.journal
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	for _, c := range []struct {
		name string
		ctx  context.Context
	}{{"nil", nil}, {"cancelled", cancelled}} {
		check("retirement_"+c.name+"_denied", a.finishIdle(c.ctx, nil) != nil)
		_, e := a.observe(ctx)
		check("retirement_"+c.name+"_original_survives", e == nil)
	}
	copy := &freshExecutionActivation{original: a, fresh: a.fresh, journal: a.journal, read: a.read, system: a.system, boot: a.boot}
	check("retirement_copy_denied", copy.finishIdle(ctx, nil) != nil)
	wire, e := json.Marshal(a)
	var decoded freshExecutionActivation
	check("retirement_serialized_denied", e == nil && string(wire) == "{}" && json.Unmarshal(wire, &decoded) == nil && decoded.finishIdle(ctx, nil) != nil)
	check("retirement_initial_finish", a.finishIdle(ctx, nil) == nil)
	j, l := a.fresh.pair.native.journal, a.fresh.pair.native.ledger
	actual, e := j.read(nil)
	expected := before
	expected.Generation++
	expected.Owner.State = "closed"
	check("retirement_exact_closed_record", e == nil && actual == expected && j.ack == expected)
	check("retirement_stores_retired", a.fresh.blocked && a.fresh.pair.blocked && j.failed && l.publicationBlocked)
	bj, bl := freshNamespaceSnapshot(t, filepath.Join(f.journal, "retirement-positive")), freshNamespaceSnapshot(t, filepath.Join(f.ledger, "retirement-positive"))
	check("retirement_repeat_denied", a.finishIdle(ctx, nil) != nil)
	_, e = a.observe(ctx)
	check("retirement_observation_denied", e != nil)
	check("retirement_repeat_preserves", reflect.DeepEqual(bj, freshNamespaceSnapshot(t, filepath.Join(f.journal, "retirement-positive"))) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, filepath.Join(f.ledger, "retirement-positive"))))
	for _, role := range []struct{ name, path string }{{"journal", filepath.Join(f.journal, "retirement-positive")}, {"ledger", filepath.Join(f.ledger, "retirement-positive")}} {
		q, e := openStateDirectory(role.path, false)
		if q != nil {
			_ = q.Close()
		}
		check("retirement_"+role.name+"_lease_held", q == nil && errors.Is(e, ErrStateLocked))
	}
	check("retirement_release", a.close() == nil && a.close() == nil)
	p, e := OpenExecutionDomainLeases(ctx, filepath.Join(f.journal, "retirement-positive"), filepath.Join(f.ledger, "retirement-positive"))
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	record, e := p.native.journal.read(nil)
	check("retirement_reopen_metadata_only", e == nil && record == expected && !p.native.journal.acknowledged && !p.native.journal.initializable)
	check("retirement_reopen_initializer_denied", p.native.journal.initialize(before, nil) != nil)
	check("retirement_reopen_close", p.Close() == nil)
	q, e := activateFreshExecutionJournal(ctx, filepath.Join(f.journal, "retirement-positive"), filepath.Join(f.ledger, "retirement-positive"), f.read, sys, activationBoot)
	check("retirement_no_recommission", q == nil && e != nil && reflect.DeepEqual(bj, freshNamespaceSnapshot(t, filepath.Join(f.journal, "retirement-positive"))))
	for _, mode := range []string{"boot-before", "boot-after", "system-before", "system-after", "ledger-entry", "journal-mode", "post-journal-touch", "post-directory-touch", "post-ledger-touch", "post-cancel", "pending-native", "pending-metadata", "disarmed", "write", "syncFile", "closeFile", "verify", "rename", "syncDirectory", "after-sync-drift"} {
		name := "retirement-" + mode
		jp, lp := filepath.Join(f.journal, name), filepath.Join(f.ledger, name)
		finishing := false
		boots, systems := 0, 0
		c, stop := context.WithCancel(ctx)
		boot := func(c context.Context) ([16]byte, error) {
			if finishing {
				boots++
				if mode == "boot-before" || mode == "boot-after" && boots == 3 {
					return [16]byte{}, ErrUnavailable
				}
			}
			return activationBoot(c)
		}
		system := func(c context.Context) (PoolSystemUse, error) {
			if finishing {
				systems++
				if mode == "system-before" || mode == "system-after" && systems == 3 {
					return PoolSystemUse{}, ErrUnavailable
				}
				if systems == 3 {
					path := ""
					switch mode {
					case "post-journal-touch":
						path = filepath.Join(jp, executionJournalName)
					case "post-directory-touch":
						path = jp
					case "post-ledger-touch":
						path = lp
					case "post-cancel":
						stop()
					}
					if path != "" && os.Chtimes(path, time.Unix(1, 0), time.Unix(1, 0)) != nil {
						t.Fatal("owned late touch")
					}
				}
			}
			return use, nil
		}
		a := fresh(name, system, boot)
		switch mode {
		case "ledger-entry":
			if os.WriteFile(filepath.Join(lp, "unknown"), []byte("owned"), 0600) != nil {
				t.Fatal("owned entry")
			}
		case "journal-mode":
			if os.Chmod(filepath.Join(jp, executionJournalName), 0644) != nil {
				t.Fatal("owned mode")
			}
		case "pending-native", "pending-metadata", "disarmed":
			next := a.journal
			next.Generation++
			switch mode {
			case "pending-native":
				next.Native = executionJournalNative{true, strings.Repeat("1", 64), strings.Repeat("2", 64), "1", "2", strings.Repeat("3", 64), next.Owner.ID, next.Owner.Boot, strings.Repeat("4", 32)}
			case "pending-metadata":
				next.Metadata = executionJournalMetadata{true, next.Census, strings.Repeat("5", 64), "ownership-1.json", "", strings.Repeat("6", 64)}
			case "disarmed":
				next.Disarmed = true
			}
			if a.fresh.pair.native.journal.publish(next, nil) != nil {
				t.Fatal("owned successor")
			}
		}
		wrap := func(tx ownershipPublication) ownershipPublication {
			bad := func() error { return syscall.EIO }
			switch mode {
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
					return os.WriteFile(filepath.Join(jp, executionJournalName), []byte("owned-drift"), 0600)
				}
			}
			return tx
		}
		finishing = true
		e := a.finishIdle(c, wrap)
		stop()
		check("retirement_"+mode+"_denied", e != nil)
		check("retirement_"+mode+"_retired", a.fresh.blocked && a.fresh.pair.blocked && a.fresh.pair.native.journal.failed && a.fresh.pair.native.ledger.publicationBlocked)
		bj, bl := freshNamespaceSnapshot(t, jp), freshNamespaceSnapshot(t, lp)
		check("retirement_"+mode+"_retry_preserves", a.finishIdle(ctx, nil) != nil && reflect.DeepEqual(bj, freshNamespaceSnapshot(t, jp)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, lp)))
	}
}

func TestInitialRetirementCutChild(t *testing.T) {
	mode := os.Getenv("HORVIUM_RETIREMENT_CUT")
	if mode == "" {
		t.Skip("owned retirement child only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("owned root fixture required")
	}
	ctx := context.Background()
	j, l := os.Getenv("HORVIUM_ACTIVATION_JOURNAL"), os.Getenv("HORVIUM_ACTIVATION_LEDGER")
	read := func(c context.Context, fd int) (ExecutionBackingObservation, error) {
		var fs unix.Statfs_t
		if c.Err() != nil || unix.Fstatfs(fd, &fs) != nil || unix.ByteSliceToString(fs.Fstypename[:]) != "ufs" {
			return ExecutionBackingObservation{}, ErrDenied
		}
		return ExecutionBackingObservation{Kind: "ufs", UFS: [2]uint32{uint32(fs.Fsid.Val[0]), uint32(fs.Fsid.Val[1])}}, nil
	}
	use := activationSystemFixture(t, filepath.Dir(j), filepath.Dir(l), read)
	a, e := activateFreshExecutionJournal(ctx, j, l, read, func(context.Context) (PoolSystemUse, error) { return use, nil }, activationBoot)
	if e != nil {
		t.Fatal(e)
	}
	defer a.close()
	barrier := func() {
		fmt.Println("RETIREMENT_CUT_REACHED " + mode)
		_, _ = io.Copy(io.Discard, os.Stdin)
		t.Fatal("cut continued")
	}
	if mode == "uid" {
		if unix.Setgroups([]int{}) != nil || unix.Setgid(65534) != nil || unix.Setuid(65534) != nil || os.Geteuid() != 65534 {
			t.Fatal("actual uid drop")
		}
		for _, s := range []*StateDirectory{a.fresh.pair.native.journal.lease, a.fresh.pair.native.ledger} {
			var st unix.Stat_t
			if unix.Fstat(int(s.directory.Fd()), &st) != nil || st.Uid != 0 {
				t.Fatal("original root descriptor")
			}
		}
		if !errors.Is(a.finishIdle(ctx, nil), ErrDenied) {
			t.Fatal("nonroot original finished")
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
	if e := a.finishIdle(ctx, wrap); e != nil {
		t.Fatal(e)
	}
	if mode == "acknowledged" {
		barrier()
	}
	t.Fatal("retirement cut missing")
}
func nativeInitialRetirementCuts(t *testing.T, f restoreFixture, check func(string, bool)) {
	t.Helper()
	for _, mode := range []string{"file-sync", "directory-sync", "acknowledged", "uid"} {
		j, l := filepath.Join(f.journal, "retirement-"+mode), filepath.Join(f.ledger, "retirement-"+mode)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.run=^TestInitialRetirementCutChild$")
		cmd.Env = append(os.Environ(), "HORVIUM_RETIREMENT_CUT="+mode, "HORVIUM_ACTIVATION_JOURNAL="+j, "HORVIUM_ACTIVATION_LEDGER="+l)
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
		if e := cmd.Start(); e != nil {
			t.Fatal(e)
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
				if scan.Text() == "RETIREMENT_CUT_REACHED "+mode {
					reached = true
					break
				}
			}
			check("retirement_"+mode+"_barrier", reached)
			_ = cmd.Process.Kill()
			e = cmd.Wait()
			var ee *exec.ExitError
			check("retirement_"+mode+"_killed_reaped", errors.As(e, &ee) && ee.ProcessState.Sys().(syscall.WaitStatus).Signal() == syscall.SIGKILL)
			bj, bl := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)

			data, e := os.ReadFile(filepath.Join(j, executionJournalName))
			record, de := decodeExecutionJournal(data)
			state := "closed"
			generation := uint64(2)
			if mode == "file-sync" || mode == "uid" {
				state = "active"
				generation = 1
			}
			check("retirement_"+mode+"_visible_history", e == nil && de == nil && record.Owner.State == state && record.Generation == generation)
			p, e := OpenExecutionDomainLeases(context.Background(), j, l)
			if mode == "file-sync" {
				if p != nil {
					_ = p.Close()
				}
				check("retirement_"+mode+"_no_receipt", p == nil && e != nil)
				// The retained staged close prevents a complete namespace view.
				sj, je := openStateDirectory(j, false)
				sl, le := openStateDirectory(l, false)
				if sj != nil {
					defer sj.Close()
				}
				if sl != nil {
					defer sl.Close()
				}
				check("retirement_"+mode+"_release", je == nil && le == nil && sj.Close() == nil && sl.Close() == nil)
			} else {
				if e != nil {
					t.Fatal(e)
				}
				defer p.Close()
				check("retirement_"+mode+"_no_receipt", !p.native.journal.acknowledged && p.native.journal.initialize(record, nil) != nil)
				check("retirement_"+mode+"_release", p.Close() == nil)
			}
			q, e := activateFreshExecutionJournal(context.Background(), j, l, f.read, func(context.Context) (PoolSystemUse, error) { return PoolSystemUse{}, ErrUnavailable }, activationBoot)
			check("retirement_"+mode+"_no_recommission", q == nil && e != nil)
			check("retirement_"+mode+"_preservation", reflect.DeepEqual(bj, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, l)))
		}()
	}
}
