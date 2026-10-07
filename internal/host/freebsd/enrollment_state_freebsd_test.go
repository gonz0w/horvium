// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
)

// Use only a new owned private directory on the parent fixture's root backing.
// It is independent of the earlier fixture's intentionally latched uncertainty.
func nativeEnrollmentStateFixture(t *testing.T, ctx context.Context, check func(string, bool), previous storage.Policy, o storage.Observation, disks []inventory.DiskObservation, u storage.PoolSystemUse, parent string) {
	t.Helper()
	path, err := os.MkdirTemp(parent, "enrollment-state-")
	if err != nil {
		t.Fatal(err)
	}
	var state *storage.StateDirectory
	cleanup := func() bool {
		if state != nil {
			if state.Close() != nil {
				return false
			}
			state = nil
		}
		if os.RemoveAll(path) != nil {
			return false
		}
		_, e := os.Lstat(path)
		return errors.Is(e, os.ErrNotExist)
	}
	t.Cleanup(func() {
		if !cleanup() {
			t.Error("enrollment state fixture cleanup")
		}
	})
	state, err = storage.OpenStateDirectory(path)
	if err != nil {
		t.Fatal(err)
	}
	var fs unix.Statfs_t
	if err = unix.Statfs(path, &fs); err != nil {
		t.Fatal(err)
	}
	unsupported := unix.ByteSliceToString(fs.Fstypename[:]) == "ufs" && fs.Flags&unix.MNT_SOFTDEP != 0
	at := time.Now().UTC()
	build := func(p storage.Policy) storage.PropertyEnrollmentPlan {
		plan, e := storage.BuildPropertyEnrollment(at, "root-fixture", p, o, disks, u)
		if e != nil {
			t.Fatal(e)
		}
		return plan
	}
	first := previous
	first.Revision = 1
	firstPlan := build(first)
	absent, err := state.BindPropertyEnrollment(at, "root-fixture", firstPlan, storage.Policy{}, o, disks, u)
	check("enrollment_state_absent", !unsupported && err == nil && absent.Validate(at, "root-fixture", o, disks, u) == nil || unsupported && errors.Is(err, storage.ErrStateDirectory) && reflect.DeepEqual(absent, storage.PropertyEnrollmentState{}))
	write := func(name string, p storage.Policy) []byte {
		data, e := storage.EncodeOwnershipRecord(p)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(path, name), data, 0600); e != nil {
			t.Fatal(e)
		}
		return data
	}
	record := "ownership-" + previous.PoolGUID + ".json"
	expected := write(record, previous)
	legacy := storage.Policy{Revision: 1, PoolGUID: "999", PoolName: "legacy_fixture"}
	legacyBytes := write("ownership-999.json", legacy)
	next := previous
	next.Revision++
	plan := build(next)
	bound, err := state.BindPropertyEnrollment(at, "root-fixture", plan, previous, o, disks, u)
	immutable := false
	if err == nil {
		beforeReview, beforePrevious := bound.Review(), bound.Predecessor()
		changedReview, changedPrevious := bound.Review(), bound.Predecessor()
		changedReview.Policy.Scopes[0].Properties[0] = storage.RefQuota
		changedReview.Policy.Members[0].Serial = "changed"
		changedPrevious.Scopes[0].Properties[0] = storage.RefQuota
		immutable = reflect.DeepEqual(beforeReview, bound.Review()) && reflect.DeepEqual(beforePrevious, bound.Predecessor())
	}
	check("enrollment_state_bound", !unsupported && err == nil && immutable && bound.Validate(at, "root-fixture", o, disks, u) == nil || unsupported && errors.Is(err, storage.ErrStateDirectory) && reflect.DeepEqual(bound, storage.PropertyEnrollmentState{}))
	t.Logf("ENROLLMENT_STATE_BACKING filesystem=%s softupdates=%t binding_denied=%t", unix.ByteSliceToString(fs.Fstypename[:]), fs.Flags&unix.MNT_SOFTDEP != 0, err != nil)
	source, err := NewPropertyEnrollmentSource(state)
	if err != nil {
		t.Fatal(err)
	}
	freshBound, err := source.Review(ctx, "root-fixture", next, previous)
	check("enrollment_source_review", !unsupported && err == nil && !freshBound.Review().Observation.Eligible || unsupported && errors.Is(err, storage.ErrStateDirectory) && reflect.DeepEqual(freshBound, storage.PropertyEnrollmentState{}))
	err = source.Validate(ctx, "root-fixture", freshBound)
	check("enrollment_source_revalidate", !unsupported && err == nil || unsupported && errors.Is(err, storage.ErrInvalid))
	badPrevious := plan.Review().Policy
	badPrevious.Revision = previous.Revision
	badPrevious.Scopes[0].Properties = []storage.Property{storage.RefQuota}
	sourceZero, sourceErr := source.Review(ctx, "root-fixture", next, badPrevious)
	check("enrollment_source_policy_denied", errors.Is(sourceErr, storage.ErrDrift) && reflect.DeepEqual(sourceZero, storage.PropertyEnrollmentState{}))
	badMember := plan.Review().Policy
	badMember.Members[0].Serial = "changed"
	sourceZero, sourceErr = source.Review(ctx, "root-fixture", badMember, previous)
	check("enrollment_source_member_denied", sourceErr != nil && reflect.DeepEqual(sourceZero, storage.PropertyEnrollmentState{}))
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	sourceZero, sourceErr = source.Review(cancelled, "root-fixture", next, previous)
	check("enrollment_source_cancelled", errors.Is(sourceErr, context.Canceled) && reflect.DeepEqual(sourceZero, storage.PropertyEnrollmentState{}))
	sourceZero, sourceErr = source.Review(ctx, "invalid root context", next, previous)
	check("enrollment_source_context_denied", errors.Is(sourceErr, storage.ErrInvalid) && reflect.DeepEqual(sourceZero, storage.PropertyEnrollmentState{}))
	changed := plan.Review().Policy
	changed.Revision = previous.Revision
	changed.Scopes[0].Properties = []storage.Property{storage.RefQuota}
	zero, err := state.BindPropertyEnrollment(at, "root-fixture", plan, changed, o, disks, u)
	check("enrollment_state_predecessor_denied", errors.Is(err, storage.ErrDrift) && reflect.DeepEqual(zero, storage.PropertyEnrollmentState{}))
	write(record, changed)
	zero, err = state.BindPropertyEnrollment(at, "root-fixture", plan, previous, o, disks, u)
	check("enrollment_state_record_drift", errors.Is(err, storage.ErrDrift) && reflect.DeepEqual(zero, storage.PropertyEnrollmentState{}))
	write(record, previous)
	legacy.Revision++
	write("ownership-999.json", legacy)
	if unsupported {
		zero, err = state.BindPropertyEnrollment(at, "root-fixture", plan, previous, o, disks, u)
	} else {
		err = bound.Validate(at, "root-fixture", o, disks, u)
	}
	sourceErr = source.Validate(ctx, "root-fixture", freshBound)
	check("enrollment_state_unrelated_drift", (!unsupported && errors.Is(err, storage.ErrDrift) || unsupported && errors.Is(err, storage.ErrStateDirectory) && reflect.DeepEqual(zero, storage.PropertyEnrollmentState{})) && (!unsupported && errors.Is(sourceErr, storage.ErrDrift) || unsupported && errors.Is(sourceErr, storage.ErrInvalid)))
	legacy.Revision--
	write("ownership-999.json", legacy)
	marker := []byte("owned enrollment fixture marker")
	foreign := filepath.Join(path, "foreign-fixture")
	if err = os.WriteFile(foreign, marker, 0600); err != nil {
		t.Fatal(err)
	}
	zero, err = state.BindPropertyEnrollment(at, "root-fixture", plan, previous, o, disks, u)
	got, readErr := os.ReadFile(foreign)
	check("enrollment_state_unknown_denied", errors.Is(err, storage.ErrStateRecord) && reflect.DeepEqual(zero, storage.PropertyEnrollmentState{}) && readErr == nil && reflect.DeepEqual(got, marker))
	if err = os.Remove(foreign); err != nil {
		t.Fatal(err)
	}
	job := storage.Job{ID: strings.Repeat("a", 64), Intent: storage.JobIntent{PlanID: strings.Repeat("b", 64), SubmissionID: strings.Repeat("c", 64), PolicyRevision: previous.Revision, PoolGUID: previous.PoolGUID, Change: storage.Intent{FilesystemGUID: previous.Scopes[0].GUID, Property: storage.Quota, Value: "1"}}, History: []storage.JobTransition{{Revision: 1, At: 1000, State: storage.JobQueued, Reason: "admitted"}}}
	jobPath := filepath.Join(path, "job-"+job.ID+".json")
	held := true
	for _, terminal := range []bool{false, true} {
		if terminal {
			job, err = storage.TransitionJob(job, storage.JobCancelled, 1001, "operator-cancelled", "")
			if err != nil {
				t.Fatal(err)
			}
		}
		data, e := storage.EncodeJobRecord(job)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(jobPath, data, 0600); e != nil {
			t.Fatal(e)
		}
		zero, e = state.BindPropertyEnrollment(at, "root-fixture", plan, previous, o, disks, u)
		got, readErr = os.ReadFile(jobPath)
		held = held && errors.Is(e, storage.ErrDenied) && reflect.DeepEqual(zero, storage.PropertyEnrollmentState{}) && readErr == nil && reflect.DeepEqual(got, data)
	}
	check("enrollment_state_job_denied", held)
	if err = os.Remove(jobPath); err != nil {
		t.Fatal(err)
	}
	_, err = state.BindPropertyEnrollment(at.Add(5*time.Minute), "root-fixture", plan, previous, o, disks, u)
	check("enrollment_state_expired_denied", errors.Is(err, storage.ErrExpired))
	_, err = state.BindPropertyEnrollment(at, "other-fixture", plan, previous, o, disks, u)
	check("enrollment_state_context_denied", errors.Is(err, storage.ErrDenied))
	orphan := filepath.Join(path, ".ownership-"+strings.Repeat("d", 32)+".tmp")
	if err = os.WriteFile(orphan, marker, 0600); err != nil {
		t.Fatal(err)
	}
	zero, err = state.BindPropertyEnrollment(at, "root-fixture", plan, previous, o, disks, u)
	got, readErr = os.ReadFile(orphan)
	if e := os.Remove(orphan); e != nil {
		t.Fatal(e)
	}
	_, latched := state.BindPropertyEnrollment(at, "root-fixture", plan, previous, o, disks, u)
	check("enrollment_state_orphan_denied", errors.Is(err, storage.ErrStateUncertain) && errors.Is(latched, storage.ErrStateUncertain) && reflect.DeepEqual(zero, storage.PropertyEnrollmentState{}) && readErr == nil && reflect.DeepEqual(got, marker))
	sourceZero, sourceErr = source.Review(ctx, "root-fixture", next, previous)
	check("enrollment_source_store_uncertain", errors.Is(sourceErr, storage.ErrStateUncertain) && reflect.DeepEqual(sourceZero, storage.PropertyEnrollmentState{}))
	got, readErr = os.ReadFile(filepath.Join(path, record))
	legacyGot, e := os.ReadFile(filepath.Join(path, "ownership-999.json"))
	check("enrollment_state_preserved", readErr == nil && e == nil && reflect.DeepEqual(got, expected) && reflect.DeepEqual(legacyGot, legacyBytes) && !o.Eligible)
	check("enrollment_source_preserved", readErr == nil && e == nil && reflect.DeepEqual(got, expected) && reflect.DeepEqual(legacyGot, legacyBytes) && (!unsupported && !freshBound.Review().Observation.Eligible || unsupported && reflect.DeepEqual(freshBound, storage.PropertyEnrollmentState{})))
	check("enrollment_state_fixture_teardown", cleanup())
}
