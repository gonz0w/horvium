// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
)

func freshPropertySubmission(t testing.TB, number int, pool string) PropertyJob {
	v := nativePropertyJobFixture(t)
	v.job.ID = fmt.Sprintf("%064x", number*3+1)
	v.job.Intent.PlanID = fmt.Sprintf("%064x", number*3+2)
	v.job.Intent.SubmissionID = fmt.Sprintf("%064x", number*3+3)
	v.intent.review.ID = v.job.Intent.PlanID
	v.job.Intent.PoolGUID = pool
	v.intent.review.PoolGUID = pool
	v.intent.review.Pool.GUID = pool
	v.intent.review.Features.PoolGUID = pool
	v.intent.review.Topology.PoolGUID = pool
	return v
}

func TestNativePropertySubmission(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable state VM required")
	}
	if os.Geteuid() != 0 {
		t.Fatal("owned root required")
	}
	anchor, err := os.MkdirTemp("/root", "horvium-submission-")
	if err != nil {
		t.Fatal(err)
	}
	var leases []*StateDirectory
	defer func() {
		for _, s := range leases {
			_ = s.Close()
		}
		_ = os.RemoveAll(anchor)
	}()
	var results []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("submission assertion", name)
		}
		results = append(results, name+"=pass")
	}
	lease := func(name string) (*StateDirectory, string) {
		t.Helper()
		path := filepath.Join(anchor, name)
		if os.Mkdir(path, 0700) != nil {
			t.Fatal("owned directory")
		}
		s, err := OpenStateDirectory(path)
		if err != nil {
			t.Fatal(err)
		}
		leases = append(leases, s)
		return s, path
	}
	write := func(path string, data []byte) {
		t.Helper()
		if os.WriteFile(path, data, 0600) != nil {
			t.Fatal("owned write")
		}
	}
	noTemps := func(path string) bool {
		entries, err := os.ReadDir(path)
		if err != nil {
			return false
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") {
				return false
			}
		}
		return true
	}
	p := nativePropertyJobFixture(t)
	fresh := func(number int, pool string) PropertyJob { return freshPropertySubmission(t, number, pool) }
	zero := func(got PropertyJob, duplicate bool, err error) bool {
		return err != nil && !duplicate && reflect.DeepEqual(got, PropertyJob{})
	}
	s, path := lease("normal")
	got, duplicate, err := s.PublishPropertySubmission(p)
	check("create", err == nil && !duplicate && reflect.DeepEqual(got, p))
	target := filepath.Join(path, "job-"+p.job.ID+".json")
	before, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	st := before.Sys().(*syscall.Stat_t)
	check("private_file", before.Mode().IsRegular() && st.Uid == 0 && st.Mode&07777 == 0600 && st.Nlink == 1)
	data, err := os.ReadFile(target)
	check("canonical_bytes", err == nil && bytes.Equal(data, reservationBytes(t, p)))
	retry := PropertyJob{job: p.Job(), intent: p.Intent()}
	retry.job.ID = strings.Repeat("d", 64)
	got, duplicate, err = s.PublishPropertySubmission(retry)
	after, statErr := os.Stat(target)
	entries, listErr := os.ReadDir(path)
	retryBytes, readErr := os.ReadFile(target)
	check("duplicate_no_write", err == nil && duplicate && reflect.DeepEqual(got, p) && statErr == nil && os.SameFile(before, after) && before.ModTime() == after.ModTime() && before.Size() == after.Size() && readErr == nil && bytes.Equal(data, retryBytes) && listErr == nil && len(entries) == 2)
	got.job.History[0].Reason = "changed"
	got.intent.review.Context[0].Name = "changed"
	got, duplicate, err = s.PublishPropertySubmission(retry)
	check("independent_result", err == nil && duplicate && reflect.DeepEqual(got, p))
	applying, err := TransitionPropertyJob(p, JobApplying, p.job.History[0].At+1, "launch-intent", "")
	if err != nil || s.PublishPropertyJob(applying, 1) != nil {
		t.Fatal("lifecycle fixture", err)
	}
	got, duplicate, err = s.PublishPropertySubmission(retry)
	check("current_history", err == nil && duplicate && reflect.DeepEqual(got, applying))
	check("lease_close", s.Close() == nil)
	s, err = OpenStateDirectory(path)
	if err != nil {
		t.Fatal(err)
	}
	leases = append(leases, s)
	got, duplicate, err = s.PublishPropertySubmission(retry)
	check("reopen_metadata_retry", err == nil && duplicate && reflect.DeepEqual(got, applying))
	for _, kind := range []string{"invalid", "not_queued", "submission_reuse", "plan_reuse", "pool_hold"} {
		v := fresh(10, p.job.Intent.PoolGUID)
		switch kind {
		case "invalid":
			v = PropertyJob{}
		case "not_queued":
			v = applying
		case "submission_reuse":
			v.job.Intent.SubmissionID = p.job.Intent.SubmissionID
		case "plan_reuse":
			v.job.Intent.PlanID = p.job.Intent.PlanID
			v.intent.review.ID = v.job.Intent.PlanID
		}
		got, duplicate, err := s.PublishPropertySubmission(v)
		check(kind+"_denial", zero(got, duplicate, err) && noTemps(path))
	}
	other := fresh(11, "777")
	got, duplicate, err = s.PublishPropertySubmission(other)
	check("unrelated_pool", err == nil && !duplicate && reflect.DeepEqual(got, other))
	legacyS, legacyPath := lease("legacy")
	j := reservationJob(20, true)
	j.Intent.PoolGUID = p.job.Intent.PoolGUID
	write(filepath.Join(legacyPath, "job-"+j.ID+".json"), reservationBytes(t, j))
	got, duplicate, err = legacyS.PublishPropertySubmission(p)
	check("terminal_pool_hold", zero(got, duplicate, err) && errors.Is(err, ErrDenied))
	j.Intent = p.job.Intent
	write(filepath.Join(legacyPath, "job-"+j.ID+".json"), reservationBytes(t, j))
	got, duplicate, err = legacyS.PublishPropertySubmission(p)
	check("legacy_retry_denial", zero(got, duplicate, err) && errors.Is(err, ErrDenied))
	// All goroutines use one real writer lease; exactly one new publication.
	for _, kind := range []string{"same_submission", "same_pool", "last_slot", "distinct_pools"} {
		fixture, folder := lease("race-" + kind)
		count := 8
		var wg sync.WaitGroup
		var mu sync.Mutex
		created, duplicates, denied := 0, 0, 0
		var firstID string
		for n := 0; n < count; n++ {
			v := fresh(n+100, "100")
			if kind == "same_submission" {
				v = PropertyJob{job: p.Job(), intent: p.Intent()}
				v.job.ID = fmt.Sprintf("%064x", n+100)
			}
			if kind == "last_slot" || kind == "distinct_pools" {
				v = fresh(n+100, fmt.Sprint(n+1000))
			}
			wg.Add(1)
			go func(v PropertyJob) {
				defer wg.Done()
				limits := defaultStateCatalogLimits
				if kind == "last_slot" {
					limits.nonterminal = 1
				}
				g, d, e := fixture.publishPropertySubmissionBounded(v, limits, nil)
				mu.Lock()
				defer mu.Unlock()
				if e != nil {
					if !zero(g, d, e) || kind == "same_submission" || kind == "distinct_pools" || kind == "same_pool" && !errors.Is(e, ErrDenied) || kind == "last_slot" && !errors.Is(e, ErrLimit) {
						t.Error("race denial", kind, e)
					}
					denied++
					return
				}
				if d {
					duplicates++
				} else {
					created++
				}
				if kind == "same_submission" {
					if firstID != "" && firstID != g.job.ID {
						t.Error("multiple duplicate jobs")
					}
					firstID = g.job.ID
				}
			}(v)
		}
		wg.Wait()
		catalog, err := fixture.InspectState()
		want := 1
		if kind == "distinct_pools" {
			want = count
		}
		check("serialized_"+kind, err == nil && created == want && len(catalog.Jobs) == want && created+duplicates+denied == count && (kind != "same_submission" || duplicates == count-1) && noTemps(folder))
	}
	for _, kind := range []string{"exact", "owner_bytes", "nonterminal", "invalid"} {
		fixture, folder := lease("bound-" + kind)
		owner := ownershipGolden()
		write(filepath.Join(folder, "ownership-100.json"), owner)
		limits := stateCatalogLimits{1, 1, 1, uint64(len(owner) + len(reservationBytes(t, p)))}
		if kind == "owner_bytes" {
			limits.bytes--
		}
		if kind == "nonterminal" {
			j := reservationJob(30, false)
			write(filepath.Join(folder, "job-"+j.ID+".json"), reservationBytes(t, j))
			limits.bytes = maxOwnershipCatalogBytes
		}
		if kind == "invalid" {
			limits = stateCatalogLimits{}
		}
		got, duplicate, err := fixture.publishPropertySubmissionBounded(p, limits, nil)
		if kind == "exact" {
			check("exact_capacity", err == nil && !duplicate && reflect.DeepEqual(got, p))
			got, duplicate, err = fixture.publishPropertySubmissionBounded(retry, limits, nil)
			check("full_store_retry", err == nil && duplicate && reflect.DeepEqual(got, p))
		} else {
			check(kind+"_capacity_denial", zero(got, duplicate, err) && errors.Is(err, ErrLimit) && noTemps(folder))
		}
	}
	for _, phase := range []string{"write", "syncFile", "closeFile", "verify", "rename", "syncDirectory", "cleanup"} {
		fixture, folder := lease("phase-" + phase)
		wrap := func(tx ownershipPublication) ownershipPublication {
			fail := func() error { return errors.New("owned failure") }
			switch phase {
			case "write":
				tx.write = fail
			case "syncFile":
				tx.syncFile = fail
			case "closeFile":
				tx.closeFile = fail
			case "verify":
				tx.verify = fail
			case "rename":
				tx.rename = fail
			case "syncDirectory":
				tx.syncDirectory = fail
			case "cleanup":
				close := tx.closeFile
				tx.verify = fail
				tx.cleanup = func() error { _ = close(); return fail() }
			}
			return tx
		}
		got, duplicate, err := fixture.publishPropertySubmissionBounded(p, defaultStateCatalogLimits, wrap)
		uncertain := phase == "syncDirectory" || phase == "cleanup"
		want := ErrStateDirectory
		if uncertain {
			want = ErrStateUncertain
		}
		check(phase+"_failure", zero(got, duplicate, err) && errors.Is(err, want) && (uncertain || noTemps(folder)))
		if uncertain {
			got, duplicate, err = fixture.PublishPropertySubmission(retry)
			check(phase+"_blocks_retry", zero(got, duplicate, err) && errors.Is(err, ErrStateUncertain))
		}
		if phase == "syncDirectory" {
			read, err := fixture.ReadPropertyJob(p.job.ID)
			check("uncertain_visible_read", err == nil && reflect.DeepEqual(read, p))
		}
	}
	for _, kind := range []string{"pool", "plan", "submission", "unknown", "orphan", "bytes"} {
		fixture, folder := lease("late-" + kind)
		foreign := fresh(50, "777")
		name := "job-" + foreign.job.ID + ".json"
		data := reservationBytes(t, foreign)
		limits := defaultStateCatalogLimits
		switch kind {
		case "pool":
			foreign = fresh(50, p.job.Intent.PoolGUID)
			data = reservationBytes(t, foreign)
		case "plan":
			foreign.job.Intent.PlanID = p.job.Intent.PlanID
			foreign.intent.review.ID = foreign.job.Intent.PlanID
			data = reservationBytes(t, foreign)
		case "submission":
			foreign.job.Intent.SubmissionID = p.job.Intent.SubmissionID
			data = reservationBytes(t, foreign)
		case "unknown":
			name = "unknown"
		case "orphan":
			name = ".job-" + strings.Repeat("1", 32) + ".tmp"
		case "bytes":
			limits.bytes = uint64(len(reservationBytes(t, p)) + len(data) - 1)
		}
		wrap := func(tx ownershipPublication) ownershipPublication {
			original := tx.syncFile
			tx.syncFile = func() error {
				if err := original(); err != nil {
					return err
				}
				write(filepath.Join(folder, name), data)
				return nil
			}
			return tx
		}
		got, duplicate, err := fixture.publishPropertySubmissionBounded(p, limits, wrap)
		_, missing := os.Stat(filepath.Join(folder, "job-"+p.job.ID+".json"))
		preserved, readErr := os.ReadFile(filepath.Join(folder, name))
		check("late_"+kind+"_denial", zero(got, duplicate, err) && os.IsNotExist(missing) && readErr == nil && bytes.Equal(preserved, data) && (kind == "orphan" || noTemps(folder)))
	}
	for _, fixture := range []*StateDirectory{nil, {}, s} {
		if fixture == s {
			if s.Close() != nil {
				t.Fatal("close")
			}
		}
		got, duplicate, err := fixture.PublishPropertySubmission(p)
		if !zero(got, duplicate, err) || !errors.Is(err, ErrStateDirectory) {
			t.Fatal("inert/closed submission", err)
		}
	}
	check("closed_inert_denial", true)
	for _, handle := range leases {
		if handle.Close() != nil {
			t.Fatal("lease teardown")
		}
	}
	check("fixture_teardown", os.RemoveAll(anchor) == nil)
	check("fixture_absent", func() bool { _, err := os.Stat(anchor); return os.IsNotExist(err) }())
	t.Log("PROPERTY_SUBMISSION_RESULT " + strings.Join(results, " "))
}
