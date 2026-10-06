// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestJobSuccessor(t *testing.T) {
	queued := jobFixture()
	applying, _ := TransitionJob(queued, JobApplying, 1001, "launch-intent", "")
	observing, _ := TransitionJob(applying, JobObserving, 1002, "native-observation", strings.Repeat("d", 64))
	if err := ValidateJobSuccessor(queued, applying); err != nil {
		t.Fatal("valid successor", err)
	}
	if err := ValidateJobSuccessor(applying, observing); err != nil {
		t.Fatal("valid observation", err)
	}
	before := append([]JobTransition(nil), applying.History...)
	for _, mutate := range []func(*Job){
		func(j *Job) { j.ID = strings.Repeat("e", 64) },
		func(j *Job) { j.Intent.PlanID = strings.Repeat("e", 64) },
		func(j *Job) { j.Intent.SubmissionID = strings.Repeat("e", 64) },
		func(j *Job) { j.Intent.PolicyRevision++ },
		func(j *Job) { j.Intent.PoolGUID = "200" },
		func(j *Job) { j.Intent.Change.FilesystemGUID = "103" },
		func(j *Job) { j.Intent.Change.Property = Quota; j.Intent.Change.Value = "none" },
		func(j *Job) { j.Intent.Change.Value = "off" },
		func(j *Job) { j.History[0].At-- },
		func(j *Job) { j.History[1].At-- },
	} {
		changed := observing
		changed.History = append([]JobTransition(nil), observing.History...)
		mutate(&changed)
		if _, err := EncodeJobRecord(changed); err != nil {
			t.Fatal("refusal control is not independently valid", err)
		}
		if err := ValidateJobSuccessor(applying, changed); !errors.Is(err, ErrDrift) {
			t.Fatal("identity/intent/history rewrite accepted", err)
		}
	}
	if !reflect.DeepEqual(applying.History, before) {
		t.Fatal("validation mutated history")
	}
	for _, pair := range [][2]Job{{queued, queued}, {queued, observing}, {applying, queued}, {Job{}, applying}, {queued, Job{}}} {
		if err := ValidateJobSuccessor(pair[0], pair[1]); err == nil {
			t.Fatal("missing/stale/skipped/invalid predecessor accepted")
		}
	}
	finished, _ := TransitionJob(observing, JobSucceeded, 1003, "verified", strings.Repeat("d", 64))
	if ValidateJobSuccessor(observing, finished) != nil || ValidateJobSuccessor(finished, applying) == nil {
		t.Fatal("terminal append/replay contract")
	}
}
