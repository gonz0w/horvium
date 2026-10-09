// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Exact-binary reexec and fixed phase names are fixture-only. No product helper,
// ZFS effect, power cut, reboot or restart clearance is exercised here.
func TestJournalPublicationCutProcess(t *testing.T) {
	phase := os.Getenv("HORVIUM_JOURNAL_CUT")
	if phase == "" {
		t.Skip("owned reexec only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("owned root VM required")
	}
	s, e := openExecutionJournalStore(os.Getenv("HORVIUM_JOURNAL_CUT_DIRECTORY"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.close()
	initial := executionJournalFixture()
	if s.initialize(initial, nil) != nil {
		t.Fatal("owned initial journal")
	}
	next := initial
	next.Generation++
	next.Metadata = journalMetadataFixture(initial)
	barrier := func() {
		fmt.Fprintln(os.Stdout, "JOURNAL_CUT_REACHED "+phase)
		_, _ = io.Copy(io.Discard, os.Stdin)
		t.Fatal("owned barrier resumed")
	}
	if phase == "before_publish" {
		barrier()
	}
	wrap := func(tx ownershipPublication) ownershipPublication {
		around := func(original func() error, before bool) func() error {
			return func() error {
				if before {
					barrier()
				}
				if e := original(); e != nil {
					return e
				}
				if !before {
					barrier()
				}
				return nil
			}
		}
		switch phase {
		case "staged_before_write":
			tx.write = around(tx.write, true)
		case "after_write":
			tx.write = around(tx.write, false)
		case "after_file_sync":
			tx.syncFile = around(tx.syncFile, false)
		case "before_rename":
			tx.rename = around(tx.rename, true)
		case "after_rename":
			tx.rename = around(tx.rename, false)
		case "after_directory_sync":
			tx.syncDirectory = around(tx.syncDirectory, false)
		case "after_return":
		default:
			t.Fatal("unknown owned phase")
		}
		return tx
	}
	if s.publish(next, wrap) != nil {
		t.Fatal("owned journal publication failed")
	}
	if phase != "after_return" {
		t.Fatal("owned phase not reached")
	}
	barrier()
}

func TestNativeExecutionJournalCuts(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	anchor, e := nativeStateAnchor(t, "horvium-journal-cuts-")
	if e != nil {
		t.Fatal(e)
	}
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("journal cut assertion", name)
		}
		results = append(results, name+"=pass")
	}
	initial := executionJournalFixture()
	next := initial
	next.Generation++
	next.Metadata = journalMetadataFixture(initial)
	initialBytes, _ := encodeExecutionJournal(initial)
	nextBytes, _ := encodeExecutionJournal(next)
	foreign := filepath.Join(anchor, "foreign")
	if os.WriteFile(foreign, initialBytes, 0600) != nil {
		t.Fatal("owned cut sentinel")
	}
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	for _, phase := range []string{"before_publish", "staged_before_write", "after_write", "after_file_sync", "before_rename", "after_rename", "after_directory_sync", "after_return"} {
		folder := filepath.Join(anchor, phase)
		lockPath := filepath.Join(folder, stateWriterName)
		if os.Mkdir(folder, 0700) != nil || os.WriteFile(lockPath, nil, 0600) != nil {
			t.Fatal("owned cut domain")
		}
		lockInfo, e := os.Stat(lockPath)
		if e != nil {
			t.Fatal(e)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		child := exec.CommandContext(ctx, binary, "-test.run=^TestJournalPublicationCutProcess$")
		child.Env = append(os.Environ(), "HORVIUM_JOURNAL_CUT="+phase, "HORVIUM_JOURNAL_CUT_DIRECTORY="+folder)
		input, e := child.StdinPipe()
		if e != nil {
			cancel()
			t.Fatal(e)
		}
		output, e := child.StdoutPipe()
		if e != nil {
			cancel()
			t.Fatal(e)
		}
		child.Stderr = io.Discard
		if child.Start() != nil {
			cancel()
			t.Fatal("owned cut child start")
		}
		joined := false
		cleanup := func() {
			if !joined {
				_ = child.Process.Kill()
				_ = child.Wait()
				joined = true
			}
			_ = input.Close()
			_ = output.Close()
			cancel()
		}
		defer cleanup()
		scanner := bufio.NewScanner(output)
		scanner.Buffer(make([]byte, 1024), 4096)
		reached := false
		for scanner.Scan() {
			if scanner.Text() == "JOURNAL_CUT_REACHED "+phase {
				reached = true
				break
			}
		}
		check(phase+"_barrier", reached)
		other, e := openExecutionJournalStore(folder)
		check(phase+"_writer_denial", other == nil && e == ErrStateLocked)
		check(phase+"_kill_reaped", child.Process.Kill() == nil && child.Wait() != nil)
		joined = true
		_ = input.Close()
		_ = output.Close()
		cancel()
		lease, e := openStateDirectory(folder, false)
		check(phase+"_lock_reacquired", e == nil && lease != nil)
		after, e := os.Stat(lockPath)
		check(phase+"_lock_preserved", e == nil && os.SameFile(lockInfo, after))
		data, e := os.ReadFile(filepath.Join(folder, executionJournalName))
		published := phase == "after_rename" || phase == "after_directory_sync" || phase == "after_return"
		expected := initialBytes
		if published {
			expected = nextBytes
		}
		check(phase+"_record_preserved", e == nil && bytes.Equal(data, expected))
		entries, e := os.ReadDir(folder)
		orphans := 0
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), executionJournalTemporaryPrefix) {
				orphans++
			}
		}
		expectedOrphans := 0
		if phase == "staged_before_write" || phase == "after_write" || phase == "after_file_sync" || phase == "before_rename" {
			expectedOrphans = 1
		}
		check(phase+"_temporary_state", e == nil && orphans == expectedOrphans)
		check(phase+"_lease_close", lease.Close() == nil)
		fresh, e := openExecutionJournalStore(folder)
		if expectedOrphans == 1 {
			check(phase+"_no_reconstructed_writer", fresh == nil && e == ErrStateUncertain)
		} else {
			if e != nil {
				t.Fatal(e)
			}
			got, e := fresh.read(nil)
			if e != nil || got.Generation != uint64(1+boolToInt(published)) {
				t.Fatal("owned cut readback", e)
			}
			check(phase+"_no_reconstructed_writer", fresh.initialize(initial, nil) == ErrDenied && fresh.publish(next, nil) == ErrDenied)
			if fresh.close() != nil {
				t.Fatal("owned cut reopen close")
			}
		}
		cleanup()
	}
	data, e := os.ReadFile(foreign)
	check("foreign_preservation", e == nil && bytes.Equal(data, initialBytes))
	if os.RemoveAll(anchor) != nil {
		t.Fatal("owned cut teardown")
	}
	_, e = os.Stat(anchor)
	check("fixture_teardown", os.IsNotExist(e))
	t.Log("EXECUTION_JOURNAL_CUTS_RESULT " + strings.Join(results, " "))
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
