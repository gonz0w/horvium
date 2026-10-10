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
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A reexec of this exact test binary calls the real metadata transaction. The
// fixture seam only blocks at named boundaries; it replaces no filesystem call.
// This is not the product helper/executor, a native ZFS write or a power cut.
func TestPropertyPublicationCutProcess(t *testing.T) {
	phase := os.Getenv("HORVIUM_PUBLICATION_CUT")
	if phase == "" {
		t.Skip("owned reexec only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("disposable root state VM required")
	}
	data, err := os.ReadFile(os.Getenv("HORVIUM_CUT_INPUT"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := DecodePropertyJob(data)
	if err != nil {
		t.Fatal(err)
	}
	s, err := OpenStateDirectory(os.Getenv("HORVIUM_CUT_DIRECTORY"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	barrier := func() {
		fmt.Fprintln(os.Stdout, "PUBLICATION_CUT_REACHED "+phase)
		// Parent keeps this pipe open, then kills and reaps this actual process.
		_, _ = io.Copy(io.Discard, os.Stdin)
		t.Fatal("publication cut unexpectedly resumed")
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
				if err := original(); err != nil {
					return err
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
			t.Fatal("unknown publication cut")
		}
		return tx
	}
	got, duplicate, err := s.publishPropertySubmissionBounded(p, defaultStateCatalogLimits, wrap)
	if err != nil || duplicate || !reflect.DeepEqual(got, p) {
		t.Fatal("publication did not return actual new success", err)
	}
	if phase != "after_return" {
		t.Fatal("unreached cut")
	}
	barrier()
}

// Compare object metadata and bytes, excluding access time and mutable physical
// allocation accounting. ZFS updates allocated blocks during transaction sync.
func publicationCutSnapshotStat(st syscall.Stat_t) syscall.Stat_t {
	st.Atimespec = syscall.Timespec{}
	st.Blocks = 0
	return st
}

func TestNativePropertyPublicationCuts(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	anchor, err := nativeStateAnchor(t, "horvium-publication-cuts-")
	if err != nil {
		t.Fatal(err)
	}
	p := nativePropertyJobFixture(t)
	data := reservationBytes(t, p)
	input := filepath.Join(anchor, "candidate")
	foreign := filepath.Join(anchor, "foreign")
	if os.WriteFile(input, data, 0600) != nil || os.WriteFile(foreign, []byte("foreign-cut-sentinel"), 0600) != nil {
		t.Fatal("owned input/sentinel")
	}
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("publication cut assertion", name)
		}
		results = append(results, name+"=pass")
	}
	// An allocated-block update alone must not impersonate a changed object.
	// Every other native field remains part of the exact metadata comparison.
	t.Run("snapshot_metadata_contract", func(t *testing.T) {
		original := syscall.Stat_t{}
		accounting := original
		accounting.Atimespec.Sec = 1
		accounting.Blocks = 8
		if !reflect.DeepEqual(publicationCutSnapshotStat(original), publicationCutSnapshotStat(accounting)) {
			t.Fatal("access/allocation accounting mistaken for object mutation")
		}
		var mutate func(reflect.Value)
		mutate = func(v reflect.Value) {
			switch v.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				v.SetInt(v.Int() + 1)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				v.SetUint(v.Uint() + 1)
			case reflect.Array:
				mutate(v.Index(0))
			case reflect.Struct:
				mutate(v.Field(0))
			default:
				t.Fatal("unsupported native stat field")
			}
		}
		value := reflect.ValueOf(original)
		for i := 0; i < value.NumField(); i++ {
			name := value.Type().Field(i).Name
			if name == "Atimespec" || name == "Blocks" {
				continue
			}
			t.Run(name, func(t *testing.T) {
				changed := original
				mutate(reflect.ValueOf(&changed).Elem().Field(i))
				if reflect.DeepEqual(publicationCutSnapshotStat(original), publicationCutSnapshotStat(changed)) {
					t.Fatal("object metadata mutation was ignored")
				}
			})
		}
	})
	check("snapshot_metadata_contract", !t.Failed())
	// Snapshot independently opened regular objects, including the stable lock
	// and any partial temporary. Ignore read access and physical block accounting.
	type object struct {
		stat syscall.Stat_t
		data []byte
	}
	snapshot := func(path string) map[string]object {
		t.Helper()
		entries, err := os.ReadDir(path)
		if err != nil {
			t.Fatal(err)
		}
		got := make(map[string]object, len(entries))
		for _, entry := range entries {
			name := filepath.Join(path, entry.Name())
			info, err := os.Lstat(name)
			if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
				t.Fatal("unsafe owned object", err)
			}
			st := *info.Sys().(*syscall.Stat_t)
			if st.Uid != 0 || st.Nlink != 1 {
				t.Fatal("unsafe owned identity")
			}
			value, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			st = publicationCutSnapshotStat(st)
			got[entry.Name()] = object{st, value}
		}
		return got
	}
	preserved := func(phase string, before, after map[string]object) bool {
		t.Helper()
		if !reflect.DeepEqual(before, after) {
			if len(before) != len(after) {
				t.Log("publication cut namespace count drift", phase)
			}
			for name, prior := range before {
				next, ok := after[name]
				role := "record"
				if name == "writer.lock" {
					role = "lock"
				}
				if !ok {
					t.Log("publication cut missing", phase, role)
					continue
				}
				a, b := reflect.ValueOf(prior.stat), reflect.ValueOf(next.stat)
				for i := 0; i < a.NumField(); i++ {
					if !reflect.DeepEqual(a.Field(i).Interface(), b.Field(i).Interface()) {
						t.Log("publication cut stat drift", phase, role, a.Type().Field(i).Name)
					}
				}
				if !bytes.Equal(prior.data, next.data) {
					t.Log("publication cut bytes drift", phase, role)
				}
			}
		}
		return reflect.DeepEqual(before, after)
	}
	for _, phase := range []string{"before_publish", "staged_before_write", "after_write", "after_file_sync", "before_rename", "after_rename", "after_directory_sync", "after_return"} {
		func() {
			path := filepath.Join(anchor, phase)
			if os.Mkdir(path, 0700) != nil {
				t.Fatal("owned cut directory")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPropertyPublicationCutProcess$")
			cmd.Env = append(os.Environ(), "HORVIUM_PUBLICATION_CUT="+phase, "HORVIUM_CUT_INPUT="+input, "HORVIUM_CUT_DIRECTORY="+path)
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			defer stdin.Close()
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			cmd.Stderr = io.Discard
			if cmd.Start() != nil {
				t.Fatal("owned child start")
			}
			waited := false
			defer func() {
				if !waited {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			line, err := bufio.NewReader(io.LimitReader(stdout, 256)).ReadString('\n')
			check(phase+"_barrier", err == nil && line == "PUBLICATION_CUT_REACHED "+phase+"\n")
			before := snapshot(path)
			other, err := OpenStateDirectory(path)
			if other != nil {
				_ = other.Close()
			}
			check(phase+"_writer_denial", other == nil && errors.Is(err, ErrStateLocked))
			if cmd.Process.Kill() != nil {
				t.Fatal("owned child kill")
			}
			waitErr := cmd.Wait()
			waited = true
			var exit *exec.ExitError
			check(phase+"_sigkill_reaped", errors.As(waitErr, &exit) && exit.Sys().(syscall.WaitStatus).Signal() == syscall.SIGKILL)
			s, err := OpenStateDirectory(path)
			check(phase+"_lock_reacquired", err == nil && s != nil)
			defer s.Close()
			after := snapshot(path)

			check(phase+"_objects_preserved", preserved(phase+"_objects", before, after))
			view, inspectErr := s.InspectJobReservations()
			published := phase == "after_rename" || phase == "after_directory_sync" || phase == "after_return"
			switch {
			case published:
				target := "job-" + p.job.ID + ".json"
				check(phase+"_restart_safe", inspectErr == nil && len(after) == 2 && bytes.Equal(after[target].data, data) && func() bool {
					page, e := view.RestartPage("", 64)
					return e == nil && len(page) == 1 && page[0].Last == p.job.History[0] && page[0].State == JobReconcileRequired && page[0].Reason == "review-expired" && len(view.Pools()) == 1
				}())
				got, duplicate, e := s.PublishPropertySubmission(p)
				check(phase+"_retry_safe", preserved(phase+"_retry", after, snapshot(path)) && e == nil && duplicate && reflect.DeepEqual(got, p))
			case phase == "before_publish":
				page, e := view.RestartPage("", 64)
				check(phase+"_restart_safe", inspectErr == nil && e == nil && page != nil && len(page) == 0 && len(after) == 1)
				got, duplicate, e := view.CheckPropertySubmission(p)
				check(phase+"_retry_safe", preserved(phase+"_retry", after, snapshot(path)) && e == nil && !duplicate && reflect.DeepEqual(got, PropertyJob{}))
			default:
				temporary := ""
				for name := range after {
					if strings.HasPrefix(name, ".job-") {
						temporary = name
					}
				}
				want := data
				if phase == "staged_before_write" {
					want = nil
				}
				check(phase+"_restart_safe", errors.Is(inspectErr, ErrStateUncertain) && reflect.DeepEqual(view, JobReservations{}) && len(after) == 2 && temporary != "" && bytes.Equal(after[temporary].data, want))
				got, duplicate, e := s.PublishPropertySubmission(p)
				check(phase+"_retry_safe", errors.Is(e, ErrStateUncertain) && !duplicate && reflect.DeepEqual(got, PropertyJob{}) && preserved(phase+"_retry", after, snapshot(path)))
			}
			check(phase+"_lock_close", s.Close() == nil)
		}()
	}
	current, err := os.ReadFile(foreign)
	check("foreign_preservation", err == nil && slices.Equal(current, []byte("foreign-cut-sentinel")))
	if os.RemoveAll(anchor) != nil {
		t.Fatal("owned fixture cleanup")
	}
	_, err = os.Lstat(anchor)
	check("fixture_teardown", os.IsNotExist(err))
	t.Log("PROPERTY_PUBLICATION_CUTS_RESULT " + strings.Join(results, " "))
}
