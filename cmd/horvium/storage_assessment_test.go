// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

func assessmentStatusFixture(t testing.TB) storage.PropertyJobAssessment {
	t.Helper()
	job, err := storage.DecodePropertyJob(statusPropertyFixture)
	if err != nil {
		t.Fatal(err)
	}
	j := job.Job()
	j, err = storage.TransitionJob(j, storage.JobApplying, j.History[0].At+1, "launch-intent", "")
	if err != nil {
		t.Fatal(err)
	}
	return storage.PropertyJobAssessment{Restart: storage.JobRestartSummary{ID: j.ID, SubmissionID: j.Intent.SubmissionID, PoolGUID: j.Intent.PoolGUID, Last: j.History[len(j.History)-1], State: storage.JobReconcileRequired, Reason: "executor-unverified"}}
}

func TestStoragePropertyAssessmentFactsAndHistory(t *testing.T) {
	for _, facts := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
		a := assessmentStatusFixture(t)
		a.BaselineMatches, a.PostconditionMatches = facts[0], facts[1]
		original := a
		var out bytes.Buffer
		if err := writeStoragePropertyAssessment(context.Background(), a.Restart.ID, a, &out); err != nil {
			t.Fatal(err)
		}
		var got storagePropertyAssessmentStatus
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Version != 1 || got.Kind != "storage-property-assessment" || got.Source != "retained-metadata-and-native-properties" || got.RecoveryVerified || got.ExecutorVerified || !got.HoldsRetained || got.BaselineMatches != facts[0] || got.ExpectedEffectMatches != facts[1] || got.Job.ID != a.Restart.ID || got.Job.LastDurable.State != storage.JobApplying || got.Job.RestartState != storage.JobReconcileRequired || got.Job.RestartReason != "executor-unverified" || !reflect.DeepEqual(a, original) {
			t.Fatal("facts changed durable/recovery presentation", got)
		}
		last := a.Restart.Last
		expected := storageJobStatusRow{ID: a.Restart.ID, SubmissionID: a.Restart.SubmissionID, PoolGUID: a.Restart.PoolGUID, LastDurable: storageJobStatusEvent{Revision: last.Revision, At: last.At, State: last.State, Reason: last.Reason, ObservationDigest: last.ObservationDigest}, RestartState: a.Restart.State, RestartReason: a.Restart.Reason}
		if got.Job != expected {
			t.Fatal("durable event or identifier changed")
		}
		for _, field := range []string{"session_digest", "policy_digest", "filesystem_guid", "property-intent", "tank/data", "review", "context", "quota"} {
			if bytes.Contains(out.Bytes(), []byte(field)) {
				t.Fatal("private baseline disclosed", field)
			}
		}
	}
}

func TestStoragePropertyAssessmentDenialsBeforeOutput(t *testing.T) {
	for _, kind := range []string{"zero", "wrong-id", "invalid-id", "cancelled", "nil-context", "oversized"} {
		t.Run(kind, func(t *testing.T) {
			a := assessmentStatusFixture(t)
			id := a.Restart.ID
			ctx := context.Background()
			switch kind {
			case "zero":
				a = storage.PropertyJobAssessment{}
			case "wrong-id":
				id = strings.Repeat("f", 64)
			case "invalid-id":
				id = "private-invalid-selection"
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "nil-context":
				ctx = nil
			case "oversized":
				a.Restart.Last.Reason = strings.Repeat("a", 8192)
			}
			var out bytes.Buffer
			if err := writeStoragePropertyAssessment(ctx, id, a, &out); err == nil || out.Len() != 0 {
				t.Fatal("denial exposed output", err)
			}
		})
	}
	a := assessmentStatusFixture(t)
	if writeStoragePropertyAssessment(context.Background(), a.Restart.ID, a, nil) == nil {
		t.Fatal("nil output accepted")
	}
}

func TestStoragePropertyAssessmentOutputFailureAndCancellation(t *testing.T) {
	a := assessmentStatusFixture(t)
	for _, out := range []io.Writer{statusBrokenWriter{}, statusShortWriter{}} {
		if err := writeStoragePropertyAssessment(context.Background(), a.Restart.ID, a, out); err == nil || err.Error() != "output unavailable" {
			t.Fatal("output failure", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := writeStoragePropertyAssessment(ctx, a.Restart.ID, a, assessmentCancellingWriter{cancel}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled delivery reported success", err)
	}
}

type assessmentCancellingWriter struct{ cancel context.CancelFunc }

func (w assessmentCancellingWriter) Write(p []byte) (int, error) { w.cancel(); return len(p), nil }

func TestStoragePropertyAssessmentArgumentsAndProtectedInput(t *testing.T) {
	for _, args := range [][]string{
		{"storage-assess"}, {"storage-assess", "-state-dir", "/"},
		{"storage-assess", "-state-dir", "relative", "-job-id", strings.Repeat("a", 64)},
		{"storage-assess", "-state-dir", "/root/a/../b", "-job-id", strings.Repeat("a", 64)},
		{"storage-assess", "-state-dir", "/root/private-ledger", "-job-id", "private-selection"},
		{"storage-assess", "-state-dir", "/root/private-ledger", "-job-id", strings.Repeat("a", 64), "extra"},
		{"storage-assess", "-native-pool", "private-native-name"},
	} {
		var out, errout bytes.Buffer
		if code := run(context.Background(), args, statusForbiddenReader{t}, &out, &errout); code != 1 || out.Len() != 0 || errout.String() != errInput.Error()+"\n" {
			t.Fatal("argument denial", code, out.String(), errout.String())
		}
	}
	if err := runStoragePropertyAssessment(context.Background(), "/root/private-ledger", "invalid", io.Discard); !errors.Is(err, storage.ErrInvalid) {
		t.Fatal("selection not rejected before opening lease", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runStoragePropertyAssessment(ctx, "/root/private-ledger", strings.Repeat("a", 64), io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled native access", err)
	}
}

func TestStoragePropertyAssessmentLifetime(t *testing.T) {
	ctx, stop := storageJobCommandContext(context.Background(), []string{"storage-assess"}, func(int) { t.Error("unexpected watchdog exit") })
	defer stop()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > 145*time.Second || time.Until(deadline) < 144*time.Second {
		t.Fatal("assessment lifetime missing/wrong")
	}
	stop()
}

func TestStoragePropertyAssessmentCancellationExitAndDisarm(t *testing.T) {
	for _, finished := range []bool{false, true} {
		parent, cancel := context.WithCancel(context.Background())
		exited := make(chan int, 1)
		_, stop := storageJobCommandContext(parent, []string{"storage-assess"}, func(code int) { exited <- code })
		if finished {
			stop()
		}
		cancel()
		wait := time.Second
		if finished {
			wait = 50 * time.Millisecond
		}
		select {
		case code := <-exited:
			if finished || code != 1 {
				t.Fatal("wrong watchdog exit", finished, code)
			}
		case <-time.After(wait):
			if !finished {
				t.Fatal("cancelled command survived")
			}
		}
		stop()
	}
}

func assessmentCommandJobFixture(t testing.TB, at time.Time, p storage.Policy, capture storage.Observation, targetGUID, id string, property storage.Property, value string, terminal bool) storage.PropertyJob {
	t.Helper()
	plan, e := storage.Build(at, "command-fixture", p, capture, storage.Intent{FilesystemGUID: targetGUID, Property: property, Value: value})
	if e != nil {
		t.Fatal(e)
	}
	intent, e := plan.CapturePropertyIntent(at, "command-fixture", p, capture)
	if e != nil {
		t.Fatal(e)
	}
	r := intent.Review()
	j := storage.Job{ID: id, Intent: storage.JobIntent{PlanID: r.ID, SubmissionID: id, PolicyRevision: p.Revision, PoolGUID: r.PoolGUID, Change: r.Intent}, History: []storage.JobTransition{{Revision: 1, At: uint64(at.Add(time.Millisecond).UnixMilli()), State: storage.JobQueued, Reason: "admitted"}}}
	bound, e := storage.BindPropertyJob(j, intent)
	if e != nil {
		t.Fatal(e)
	}
	state, reason := storage.JobApplying, "launch-intent"
	if terminal {
		state, reason = storage.JobCancelled, "operator-cancelled"
	}
	bound, e = storage.TransitionPropertyJob(bound, state, uint64(at.Add(2*time.Millisecond).UnixMilli()), reason, "")
	if e != nil {
		t.Fatal(e)
	}
	return bound
}

func TestStoragePropertyAssessmentFixtureAdmissionAfterFractionalCapture(t *testing.T) {
	retained, err := storage.DecodePropertyJob(statusPropertyFixture)
	if err != nil {
		t.Fatal(err)
	}
	r := retained.Intent().Review()
	p := storage.Policy{Revision: r.PolicyRevision, PoolGUID: r.PoolGUID, PoolName: r.PoolName}
	o := storage.Observation{Complete: true, Eligible: true, PoolGUID: r.PoolGUID, PoolName: r.PoolName, Filesystems: r.Context, States: r.States, Pool: r.Pool, Features: r.Features, Topology: r.Topology}
	for _, f := range r.Context {
		o.Identities = append(o.Identities, storage.DatasetIdentity{GUID: f.GUID, Name: f.Name, Type: storage.FilesystemType})
		p.Scopes = append(p.Scopes, storage.Scope{GUID: f.GUID, Name: f.Name, ParentGUID: f.ParentGUID, Properties: []storage.Property{storage.Quota, storage.RefQuota, storage.Compression}})
	}
	o.Identities = append(o.Identities, r.Versions...)
	at := time.Unix(1791400000, 123456789).UTC()
	for _, terminal := range []bool{false, true} {
		job := assessmentCommandJobFixture(t, at, p, o, r.Intent.FilesystemGUID, strings.Repeat("a", 64), r.Intent.Property, r.Intent.Value, terminal)
		queued := time.UnixMilli(int64(job.Job().History[0].At))
		if queued.Before(at) || !queued.Before(at.Add(5*time.Minute)) {
			t.Fatal("fixture admission precedes captured review")
		}
		if _, err := storage.EncodePropertyJob(job); err != nil {
			t.Fatal(err)
		}
	}
}

// Preserve a valid root policy namespace while selecting an absent native name.
func assessmentMissingPoolFixture(p storage.Policy, name string) storage.Policy {
	result := p
	result.PoolName = name
	result.Scopes = append([]storage.Scope(nil), p.Scopes...)
	for n := range result.Scopes {
		result.Scopes[n].Name = name + strings.TrimPrefix(result.Scopes[n].Name, p.PoolName)
	}
	return result
}

func TestStoragePropertyAssessmentMissingPoolPolicyFixture(t *testing.T) {
	job, err := storage.DecodePropertyJob(statusPropertyFixture)
	if err != nil {
		t.Fatal(err)
	}
	r := job.Intent().Review()
	p := storage.Policy{Revision: r.PolicyRevision, PoolGUID: r.PoolGUID, PoolName: r.PoolName}
	for _, f := range r.Context {
		p.Scopes = append(p.Scopes, storage.Scope{GUID: f.GUID, Name: f.Name, ParentGUID: f.ParentGUID, Properties: []storage.Property{r.Intent.Property}})
	}
	before, err := storage.EncodeOwnershipRecord(p)
	if err != nil {
		t.Fatal(err)
	}
	missing := assessmentMissingPoolFixture(p, "absent_fixture")
	if _, err := storage.EncodeOwnershipRecord(missing); err != nil {
		t.Fatal("missing-native fixture contains invalid policy", err)
	}
	after, err := storage.EncodeOwnershipRecord(p)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("missing-native setup mutated policy")
	}
}
