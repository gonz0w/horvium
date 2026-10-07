// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/storage"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Only the opted-in native parent supplies its already-qualified owned lease,
// independent full use view and private fixture records. No extra disk is used.
func nativePropertyStateFixture(t *testing.T, ctx context.Context, check func(string, bool), state *storage.StateDirectory, path string, p storage.Policy, u storage.PoolSystemUse, expected, legacy []byte, fs string, softupdates bool) {
	t.Helper()
	unsupported := fs == "ufs" && softupdates
	err := state.CheckPropertyState(p, u)
	check("state_backing_gate", err == nil && !unsupported || unsupported && errors.Is(err, storage.ErrStateDirectory))
	t.Logf("PROPERTY_STATE_BACKING filesystem=%s softupdates=%t check_denied=%t", fs, softupdates, err != nil)
	source, err := NewPropertyStateSource(state)
	if err != nil {
		t.Fatal(err)
	}
	got, observation, err := source.Collect(ctx, p.PoolGUID)
	check("state_composed_gate", !unsupported && err == nil && got.PoolGUID == p.PoolGUID && !observation.Eligible || unsupported && errors.Is(err, storage.ErrStateDirectory) && reflect.DeepEqual(got, storage.Policy{}) && reflect.DeepEqual(observation, storage.Observation{}))
	changed := p
	changed.Revision++
	check("state_policy_drift_denied", errors.Is(state.CheckPropertyState(changed, u), storage.ErrDrift))
	record := filepath.Join(path, "ownership-"+p.PoolGUID+".json")
	if err = os.Remove(record); err != nil {
		t.Fatal(err)
	}
	check("state_missing_policy_denied", errors.Is(state.CheckPropertyState(p, u), storage.ErrDenied))
	if err = os.WriteFile(record, expected, 0600); err != nil {
		t.Fatal(err)
	}
	unknown := filepath.Join(path, "foreign-fixture")
	marker := []byte("owned fixture preservation")
	if err = os.WriteFile(unknown, marker, 0600); err != nil {
		t.Fatal(err)
	}
	err = state.CheckPropertyState(p, u)
	after, readErr := os.ReadFile(unknown)
	check("state_unknown_denied", errors.Is(err, storage.ErrStateRecord) && readErr == nil && reflect.DeepEqual(after, marker))
	if err = os.Remove(unknown); err != nil {
		t.Fatal(err)
	}
	job := storage.Job{ID: strings.Repeat("a", 64), Intent: storage.JobIntent{PlanID: strings.Repeat("b", 64), SubmissionID: strings.Repeat("c", 64), PolicyRevision: p.Revision, PoolGUID: p.PoolGUID, Change: storage.Intent{FilesystemGUID: p.Scopes[0].GUID, Property: storage.Quota, Value: "1"}}, History: []storage.JobTransition{{Revision: 1, At: 1000, State: storage.JobQueued, Reason: "admitted"}}}
	jobPath := filepath.Join(path, "job-"+job.ID+".json")
	writeJob := func(j storage.Job) []byte {
		b, e := storage.EncodeJobRecord(j)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(jobPath, b, 0600); e != nil {
			t.Fatal(e)
		}
		return b
	}
	queued := writeJob(job)
	err = state.CheckPropertyState(p, u)
	after, readErr = os.ReadFile(jobPath)
	check("state_job_hold_denied", errors.Is(err, storage.ErrDenied) && readErr == nil && reflect.DeepEqual(after, queued))
	cancelled, err := storage.TransitionJob(job, storage.JobCancelled, 1001, "operator-cancelled", "")
	if err != nil {
		t.Fatal(err)
	}
	terminal := writeJob(cancelled)
	err = state.CheckPropertyState(p, u)
	after, readErr = os.ReadFile(jobPath)
	check("state_terminal_hold_denied", errors.Is(err, storage.ErrDenied) && readErr == nil && reflect.DeepEqual(after, terminal))
	if err = os.Remove(jobPath); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(path, ".job-"+strings.Repeat("d", 32)+".tmp")
	if err = os.WriteFile(orphan, marker, 0600); err != nil {
		t.Fatal(err)
	}
	err = state.CheckPropertyState(p, u)
	after, readErr = os.ReadFile(orphan)
	check("state_orphan_denied", errors.Is(err, storage.ErrStateUncertain) && readErr == nil && reflect.DeepEqual(after, marker))
	if err = os.Remove(orphan); err != nil {
		t.Fatal(err)
	}
	err = state.CheckPropertyState(p, u)
	got, observation, composeErr := source.Collect(ctx, p.PoolGUID)
	check("state_uncertainty_latched", errors.Is(err, storage.ErrStateUncertain) && errors.Is(composeErr, storage.ErrStateUncertain) && reflect.DeepEqual(got, storage.Policy{}) && reflect.DeepEqual(observation, storage.Observation{}))
	after, err = os.ReadFile(record)
	afterLegacy, legacyErr := os.ReadFile(filepath.Join(path, "ownership-999.json"))
	check("state_preserved", err == nil && legacyErr == nil && reflect.DeepEqual(after, expected) && reflect.DeepEqual(afterLegacy, legacy))
}
