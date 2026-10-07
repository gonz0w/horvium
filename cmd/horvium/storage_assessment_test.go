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
