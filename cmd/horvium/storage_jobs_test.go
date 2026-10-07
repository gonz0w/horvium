// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/storage"
)

// Generated solely from accepted storage.propertyJobFixture/EncodePropertyJob;
// synthetic GUIDs, providers and session digest, never operator metadata.
//
//go:embed testdata/status-property-job.json
var statusPropertyFixture []byte

func statusJobFixture(t testing.TB, n int, terminal bool) storage.Job {
	t.Helper()
	j := storage.Job{ID: fmt.Sprintf("%064x", n), Intent: storage.JobIntent{PlanID: fmt.Sprintf("%064x", n+100), SubmissionID: fmt.Sprintf("%064x", n+1000), PolicyRevision: 1, PoolGUID: "100", Change: storage.Intent{FilesystemGUID: "102", Property: storage.Compression, Value: "lz4"}}, History: []storage.JobTransition{{Revision: 1, At: 1000, State: storage.JobQueued, Reason: "admitted"}}}
	if terminal {
		var err error
		j, err = storage.TransitionJob(j, storage.JobCancelled, 1001, "operator-cancelled", "")
		if err != nil {
			t.Fatal(err)
		}
	}
	return j
}

func statusViewFixture(t testing.TB, count int) storage.JobReservations {
	t.Helper()
	data := make([][]byte, 0, count)
	for i := count; i > 0; i-- {
		b, err := storage.EncodeJobRecord(statusJobFixture(t, i, i > 64))
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, b)
	}
	v, err := storage.BuildJobReservations(data)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestStorageJobStatusCompleteSortedAndConservative(t *testing.T) {
	v := statusViewFixture(t, 65)
	holds := v.Pools()
	var out bytes.Buffer
	if err := writeStorageJobStatus(context.Background(), v, &out); err != nil {
		t.Fatal(err)
	}
	var result storageJobStatus
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Version != 1 || result.Kind != "storage-job-status" || result.Source != "retained-metadata" || result.RecoveryVerified || !result.HoldsRetained || len(result.Jobs) != 65 {
		t.Fatal("partial or authoritative status")
	}
	for i, j := range result.Jobs {
		if j.ID != fmt.Sprintf("%064x", i+1) || j.SubmissionID != fmt.Sprintf("%064x", i+1001) || j.PoolGUID != "100" {
			t.Fatal("lost selection/order")
		}
		if i < 64 && (j.RestartState != storage.JobReconcileRequired || j.RestartReason != "review-expired" || j.LastDurable.State != storage.JobQueued || j.LastDurable.Reason != "admitted") {
			t.Fatal("queued restart authorizes work")
		}
		if i == 64 && (j.RestartState != storage.JobCancelled || j.LastDurable.State != storage.JobCancelled) {
			t.Fatal("terminal history lost")
		}
	}
	if fmt.Sprint(holds) != fmt.Sprint(v.Pools()) {
		t.Fatal("status released holds")
	}
	for _, field := range []string{"plan_id", "change", "session", "policy_digest", "policy_revision", "history", "filesystem_guid"} {
		if strings.Contains(out.String(), `"`+field+`"`) {
			t.Fatal("full/private intent exposed", field)
		}
	}
}

func TestStorageJobStatusEmptyIsComplete(t *testing.T) {
	var out bytes.Buffer
	if err := writeStorageJobStatus(context.Background(), statusViewFixture(t, 0), &out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"jobs":[]`)) {
		t.Fatal("empty list is ambiguous", out.String())
	}
}

func TestStorageJobStatusDenialHasNoOutput(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, mode := range []string{"nil-context", "cancelled", "zero-view"} {
		var out bytes.Buffer
		ctx := context.Background()
		v := statusViewFixture(t, 1)
		switch mode {
		case "nil-context":
			ctx = nil
		case "cancelled":
			ctx = cancelled
		case "zero-view":
			v = storage.JobReservations{}
		}
		if err := writeStorageJobStatus(ctx, v, &out); err == nil || out.Len() != 0 {
			t.Fatal("denial exposed output", mode, err)
		}
	}
}

type statusBrokenWriter struct{}

func (statusBrokenWriter) Write([]byte) (int, error) { return 0, errors.New("private output error") }
func TestStorageJobStatusOutputErrorIsFixed(t *testing.T) {
	err := writeStorageJobStatus(context.Background(), statusViewFixture(t, 1), statusBrokenWriter{})
	if err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("raw output error exposed", err)
	}
}

type statusForbiddenReader struct{ t *testing.T }

func (r statusForbiddenReader) Read([]byte) (int, error) {
	r.t.Fatal("job status read authentication input")
	return 0, io.EOF
}
func TestStorageJobStatusCommandDeniesInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{"storage-jobs"}, {"storage-jobs", "-state-dir", "relative"}, {"storage-jobs", "-state-dir", "/"}, {"storage-jobs", "-state-dir", "/root/../state"}, {"storage-jobs", "-state-dir", "/root/private-secret", "extra"}, {"storage-jobs", "-state-dir", "/root/private-secret", "-repair"}} {
		var out, errout bytes.Buffer
		if run(context.Background(), args, statusForbiddenReader{t}, &out, &errout) == 0 || out.Len() != 0 || strings.Contains(errout.String(), "private-secret") {
			t.Fatal("invalid diagnostic accepted/leaked", args)
		}
	}
}

func TestStorageJobStatusMixedRecordsHidePropertyBaseline(t *testing.T) {
	data, err := os.ReadFile("testdata/status-property-job.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := storage.DecodePropertyJob(data)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := storage.EncodeJobRecord(statusJobFixture(t, 1, false))
	if err != nil {
		t.Fatal(err)
	}
	v, err := storage.BuildJobReservations([][]byte{data, plain})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := writeStorageJobStatus(context.Background(), v, &out); err != nil {
		t.Fatal(err)
	}
	var result storageJobStatus
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Jobs) != 2 || result.Jobs[1].ID != p.Job().ID || result.Jobs[1].RestartReason != "review-expired" {
		t.Fatal("property job omitted")
	}
	for _, private := range []string{"tank/data", "session_digest", "policy_digest", "property-intent", "fixture-session-secret", "/dev/ada"} {
		if strings.Contains(out.String(), private) {
			t.Fatal("private baseline exposed", private)
		}
	}
}

func TestStorageJobStatusMaximumAcceptedCatalogIsComplete(t *testing.T) {
	var out bytes.Buffer
	if err := writeStorageJobStatus(context.Background(), statusViewFixture(t, 1064), &out); err != nil {
		t.Fatal(err)
	}
	var result storageJobStatus
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Jobs) != 1064 || result.Jobs[1063].ID != fmt.Sprintf("%064x", 1064) || out.Len() > 1<<20 {
		t.Fatal("accepted store omitted or exceeded output bound")
	}
}

type statusShortWriter struct{}

func (statusShortWriter) Write(b []byte) (int, error) { return len(b) - 1, nil }
func TestStorageJobStatusShortOutputFails(t *testing.T) {
	if err := writeStorageJobStatus(context.Background(), statusViewFixture(t, 1), statusShortWriter{}); err == nil {
		t.Fatal("partial delivery reported success")
	}
}
