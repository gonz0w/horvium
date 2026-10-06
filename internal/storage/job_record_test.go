// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func jobFixture() Job {
	return Job{ID: strings.Repeat("a", 64), Intent: JobIntent{PlanID: strings.Repeat("b", 64), SubmissionID: strings.Repeat("c", 64), PolicyRevision: 1, PoolGUID: "100", Change: Intent{FilesystemGUID: "102", Property: Compression, Value: "lz4"}}, History: []JobTransition{{Revision: 1, At: 1000, State: JobQueued, Reason: "admitted", ObservationDigest: ""}}}
}

const jobGoldenContent = `{"version":1,"kind":"job","job":{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","intent":{"plan_id":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","submission_id":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","policy_revision":1,"pool_guid":"100","change":{"filesystem_guid":"102","property":"compression","value":"lz4"}},"history":[{"revision":1,"at":1000,"state":"queued","reason":"admitted","observation_digest":""}]}}`

func uncheckedJob(content string) []byte {
	h := sha256.Sum256([]byte("horvium:job:v1\n" + content))
	return []byte(strings.TrimSuffix(content, "}") + `,"checksum":"` + hex.EncodeToString(h[:]) + `"}`)
}

func TestJobRecordCanonicalAndIndependent(t *testing.T) {
	j := jobFixture()
	before, _ := json.Marshal(j)
	b, err := EncodeJobRecord(j)
	if err != nil || !bytes.Equal(b, uncheckedJob(jobGoldenContent)) {
		t.Fatal("canonical record", err)
	}
	after, _ := json.Marshal(j)
	if !bytes.Equal(before, after) {
		t.Fatal("encode changed input")
	}
	got, err := DecodeJobRecord(b)
	if err != nil || !reflect.DeepEqual(j, got) {
		t.Fatal("round trip", err)
	}
	got.History[0].Reason = "changed"
	b[0] = 'x'
	again, err := DecodeJobRecord(uncheckedJob(jobGoldenContent))
	if err != nil || again.History[0].Reason != "admitted" || j.History[0].Reason != "admitted" {
		t.Fatal("aliasing")
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, uncheckedJob(jobGoldenContent), "", "  ") != nil {
		t.Fatal("indent fixture")
	}
	if _, err := DecodeJobRecord(pretty.Bytes()); err != nil {
		t.Fatal("valid presentation", err)
	}
	exact := append(uncheckedJob(jobGoldenContent), bytes.Repeat([]byte(" "), maxStateRecordBytes-len(uncheckedJob(jobGoldenContent)))...)
	if _, err := DecodeJobRecord(exact); err != nil {
		t.Fatal("exact byte ceiling", err)
	}
}

func TestJobLifecycleAndRestart(t *testing.T) {
	j := jobFixture()
	obs := strings.Repeat("d", 64)
	applying, err := TransitionJob(j, JobApplying, 1001, "launch-intent", "")
	if err != nil || len(j.History) != 1 {
		t.Fatal("applying transition", err)
	}
	observing, err := TransitionJob(applying, JobObserving, 1002, "native-observation", obs)
	if err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []struct {
		state  JobState
		reason string
	}{{JobSucceeded, "verified"}, {JobFailed, "verified-failure"}} {
		finished, err := TransitionJob(observing, outcome.state, 1003, outcome.reason, obs)
		if err != nil || !finished.Terminal() {
			t.Fatal("verified outcome", err)
		}
		encoded, encodeErr := EncodeJobRecord(finished)
		decoded, decodeErr := DecodeJobRecord(encoded)
		if encodeErr != nil || decodeErr != nil || !reflect.DeepEqual(finished, decoded) {
			t.Fatal("verified outcome not retained", encodeErr, decodeErr)
		}
		if _, err := TransitionJob(finished, JobApplying, 1004, "launch-intent", ""); err == nil {
			t.Fatal("terminal replay")
		}
		state, reason, err := jobRestartProjection(finished)
		if err != nil || state != outcome.state || reason != outcome.reason {
			t.Fatal("terminal recovery projection")
		}
	}
	cancelled, err := TransitionJob(j, JobCancelled, 1001, "operator-cancelled", "")
	if err != nil || !cancelled.Terminal() {
		t.Fatal("known unlaunched cancellation", err)
	}
	state, reason, err := jobRestartProjection(cancelled)
	if err != nil || state != JobCancelled || reason != "operator-cancelled" {
		t.Fatal("cancelled projection changed")
	}
	for _, tc := range []struct {
		job    Job
		reason string
	}{{j, "review-expired"}, {applying, "executor-unverified"}, {observing, "executor-unverified"}} {
		state, reason, err := jobRestartProjection(tc.job)
		if err != nil || state != JobReconcileRequired || reason != tc.reason || tc.job.Terminal() {
			t.Fatal("restart enables replay", err)
		}
	}
	uncertain, err := TransitionJob(observing, JobReconcileRequired, 1003, "ambiguous-effect", obs)
	if err != nil || uncertain.Terminal() {
		t.Fatal("uncertain job became evictable", err)
	}
	state, reason, err = jobRestartProjection(uncertain)
	if err != nil || state != JobReconcileRequired || reason != "ambiguous-effect" {
		t.Fatal("uncertainty erased")
	}
	if _, err := TransitionJob(uncertain, JobSucceeded, 1004, "verified", obs); err == nil {
		t.Fatal("unqualified reconciliation enabled")
	}
	for _, candidate := range []Job{j, applying, observing, cancelled, uncertain} {
		encoded, err := EncodeJobRecord(candidate)
		decoded, readErr := DecodeJobRecord(encoded)
		if err != nil || readErr != nil || !reflect.DeepEqual(candidate, decoded) {
			t.Fatal("lifecycle data lost in codec", err, readErr)
		}
	}
	applying.History[0].Reason = "changed"
	if j.History[0].Reason != "admitted" {
		t.Fatal("transition aliased history")
	}
}

func TestJobLifecycleDenials(t *testing.T) {
	j := jobFixture()
	a, _ := TransitionJob(j, JobApplying, 1001, "launch-intent", "")
	o, _ := TransitionJob(a, JobObserving, 1002, "native-observation", strings.Repeat("d", 64))
	for _, tc := range []struct {
		job            Job
		state          JobState
		at             uint64
		reason, digest string
	}{
		{j, JobSucceeded, 1001, "verified", strings.Repeat("d", 64)},
		{j, JobObserving, 1001, "native-observation", strings.Repeat("d", 64)},
		{a, JobCancelled, 1002, "operator-cancelled", ""},
		{a, JobFailed, 1002, "verified-failure", strings.Repeat("d", 64)},
		{a, JobSucceeded, 1002, "verified", strings.Repeat("d", 64)},
		{o, JobApplying, 1003, "launch-intent", ""},
		{o, JobObserving, 1003, "native-observation", strings.Repeat("d", 64)},
		{o, JobSucceeded, 1003, "verified", ""},
		{o, JobSucceeded, 1003, "storage-unavailable", strings.Repeat("d", 64)},
		{o, JobReconcileRequired, 1003, "ambiguous-effect", ""},
		{o, JobReconcileRequired, 1003, "ambiguous-effect", strings.Repeat("e", 64)},
		{j, JobApplying, 999, "launch-intent", ""},
		{j, JobApplying, 1001, "unknown", ""},
		{j, JobApplying, 1001, "launch-intent", strings.Repeat("d", 64)},
		{j, JobState("running"), 1001, "launch-intent", ""},
	} {
		got, err := TransitionJob(tc.job, tc.state, tc.at, tc.reason, tc.digest)
		if err == nil || !reflect.DeepEqual(got, Job{}) {
			t.Fatal("invalid transition produced a job", tc.state, err)
		}
	}
}

func TestJobRecordRequiredFieldsAndTypes(t *testing.T) {
	// Independently mutate every required field at every object level. Keeping
	// other fields intact distinguishes shape denial from a checksum-only test.
	paths := [][]string{{}, {"job"}, {"job", "intent"}, {"job", "intent", "change"}, {"job", "history", "0"}}
	for _, path := range paths {
		var pristine map[string]any
		if json.Unmarshal(uncheckedJob(jobGoldenContent), &pristine) != nil {
			t.Fatal("fixture decode")
		}
		object := jobTestObject(pristine, path)
		for key := range object {
			for _, mutation := range []string{"missing", "null", "wrong type", "case", "duplicate"} {
				var root map[string]any
				_ = json.Unmarshal(uncheckedJob(jobGoldenContent), &root)
				obj := jobTestObject(root, path)
				value := obj[key]
				switch mutation {
				case "missing":
					delete(obj, key)
				case "null":
					obj[key] = nil
				case "wrong type":
					obj[key] = true
				case "case":
					delete(obj, key)
					obj[strings.ToUpper(key)] = value
				}
				data, _ := json.Marshal(root)
				if mutation == "duplicate" {
					needle, _ := json.Marshal(key)
					data = bytes.Replace(data, append(needle, ':'), append(append(append([]byte(nil), needle...), []byte(`:null,`)...), append(needle, ':')...), 1)
				}
				t.Run(strings.Join(path, "/")+"/"+key+"/"+mutation, func(t *testing.T) {
					if got, err := DecodeJobRecord(data); err == nil || !reflect.DeepEqual(got, Job{}) {
						t.Fatal("invalid shape accepted", err)
					}
				})
			}
		}
	}
}

func jobTestObject(root map[string]any, path []string) map[string]any {
	var current any = root
	for _, key := range path {
		if key == "0" {
			current = current.([]any)[0]
		} else {
			current = current.(map[string]any)[key]
		}
	}
	return current.(map[string]any)
}

func TestJobMetadataLimitsAndPrecision(t *testing.T) {
	j := jobFixture()
	j.Intent.PolicyRevision = ^uint64(0)
	b, err := EncodeJobRecord(j)
	got, e := DecodeJobRecord(b)
	if err != nil || e != nil || got.Intent.PolicyRevision != j.Intent.PolicyRevision {
		t.Fatal("uint64 policy revision lost")
	}
	for _, at := range []uint64{0, 253402300800000, ^uint64(0)} {
		bad := jobFixture()
		bad.History[0].At = at
		if b, err := EncodeJobRecord(bad); err == nil || b != nil {
			t.Fatal("invalid civil timestamp")
		}
	}
	for _, change := range []Intent{{"102", Quota, "none"}, {"102", Quota, "18446744073709551615"}, {"102", RefQuota, "1024"}, {"102", Compression, "off"}} {
		valid := jobFixture()
		valid.Intent.Change = change
		if _, err := EncodeJobRecord(valid); err != nil {
			t.Fatal("accepted property subset refused", err)
		}
	}
	invalid := jobFixture()
	invalid.History[0].State = JobSucceeded
	if invalid.Terminal() {
		t.Fatal("invalid success history became terminal")
	}
	if state, reason, err := jobRestartProjection(invalid); err == nil || state != "" || reason != "" {
		t.Fatal("invalid restart projection")
	}
	if _, err := TransitionJob(invalid, JobApplying, 1001, "launch-intent", ""); err == nil {
		t.Fatal("invalid input transition")
	}
	applying, _ := TransitionJob(jobFixture(), JobApplying, 1001, "launch-intent", "")
	for _, change := range []func(*Job){
		func(j *Job) { j.History[1].Revision = ^uint64(0) },
		func(j *Job) { j.History[1].Revision = 1 },
		func(j *Job) { j.History[1].At = 999 },
	} {
		bad := applying
		bad.History = append([]JobTransition(nil), applying.History...)
		change(&bad)
		if b, err := EncodeJobRecord(bad); err == nil || b != nil {
			t.Fatal("invalid history sequence encoded")
		}
	}
	for _, mutate := range []func(*Job){
		func(j *Job) { j.History = nil },
		func(j *Job) { j.ID = "" },
		func(j *Job) { j.Intent.PlanID = strings.Repeat("0", 63) },
		func(j *Job) { j.Intent.SubmissionID = "../job" },
		func(j *Job) { j.Intent.Change.FilesystemGUID = "0" },
		func(j *Job) { j.History[0].Reason = "claimed" },
		func(j *Job) { j.History[0].ObservationDigest = strings.Repeat("d", 64) },
	} {
		bad := jobFixture()
		mutate(&bad)
		if b, err := EncodeJobRecord(bad); err == nil || b != nil {
			t.Fatal("invalid metadata encoded")
		}
	}
	for _, field := range []string{"id", "plan", "submission", "pool", "filesystem", "property", "value", "reason", "observation"} {
		bad := jobFixture()
		large := strings.Repeat("x", maxFieldBytes+1)
		switch field {
		case "id":
			bad.ID = large
		case "plan":
			bad.Intent.PlanID = large
		case "submission":
			bad.Intent.SubmissionID = large
		case "pool":
			bad.Intent.PoolGUID = large
		case "filesystem":
			bad.Intent.Change.FilesystemGUID = large
		case "property":
			bad.Intent.Change.Property = Property(large)
		case "value":
			bad.Intent.Change.Value = large
		case "reason":
			bad.History[0].Reason = large
		case "observation":
			bad.History[0].ObservationDigest = large
		}
		if b, err := EncodeJobRecord(bad); !errors.Is(err, ErrLimit) || b != nil {
			t.Fatal("field budget bypass", field, err)
		}
	}
}

func TestJobRecordStrictDenials(t *testing.T) {
	good := string(uncheckedJob(jobGoldenContent))
	for name, data := range map[string]string{
		"empty": "", "trailing": good + "{}", "null": "null", "array": "[" + good + "]", "truncated": good[:len(good)-1],
		"duplicate":         strings.Replace(good, `"version":1`, `"version":1,"version":1`, 1),
		"escaped duplicate": strings.Replace(good, `"version":1`, `"version":1,"versi\u006fn":1`, 1),
		"case":              strings.Replace(good, `"version"`, `"Version"`, 1),
		"unknown":           strings.Replace(good, `"revision":1`, `"unknown":true,"revision":1`, 1),
		"missing":           strings.Replace(good, `"observation_digest":""`, `"removed":""`, 1),
		"null history":      strings.Replace(good, `"history":[`, `"history":null,"removed":[`, 1),
		"version":           strings.Replace(good, `"version":1`, `"version":2`, 1),
		"kind":              strings.Replace(good, `"kind":"job"`, `"kind":"ownership"`, 1),
		"number string":     strings.Replace(good, `"at":1000`, `"at":"1000"`, 1),
		"number exponent":   strings.Replace(good, `"at":1000`, `"at":1e3`, 1),
		"number fraction":   strings.Replace(good, `"at":1000`, `"at":1000.0`, 1),
		"number negative":   strings.Replace(good, `"at":1000`, `"at":-1`, 1),
		"overflow":          strings.Replace(good, `"at":1000`, `"at":18446744073709551616`, 1),
		"damage":            strings.Replace(good, `"lz4"`, `"off"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := DecodeJobRecord([]byte(data)); err == nil || !reflect.DeepEqual(got, Job{}) {
				t.Fatal("invalid record accepted", err)
			}
		})
	}
	for _, data := range [][]byte{append([]byte(good), 0xff), bytes.Repeat([]byte("x"), maxStateRecordBytes+1), []byte(`{"job":` + strings.Repeat("[", 10000) + strings.Repeat("]", 10000) + `}`)} {
		if got, err := DecodeJobRecord(data); err == nil || !reflect.DeepEqual(got, Job{}) {
			t.Fatal("unbounded/malformed input accepted")
		}
	}
}

func TestJobChecksumCannotGrantLifecycle(t *testing.T) {
	for _, content := range []string{
		strings.Replace(jobGoldenContent, `"state":"queued"`, `"state":"succeeded"`, 1),
		strings.Replace(jobGoldenContent, `"revision":1`, `"revision":0`, 1),
		strings.Replace(jobGoldenContent, `"at":1000`, `"at":0`, 1),
		strings.Replace(jobGoldenContent, `"policy_revision":1`, `"policy_revision":0`, 1),
		strings.Replace(jobGoldenContent, `"pool_guid":"100"`, `"pool_guid":"0100"`, 1),
		strings.Replace(jobGoldenContent, `"property":"compression"`, `"property":"mountpoint"`, 1),
		strings.Replace(jobGoldenContent, `"value":"lz4"`, `"value":"lz4;reboot"`, 1),
		strings.Replace(jobGoldenContent, strings.Repeat("a", 64), strings.Repeat("A", 64), 1),
	} {
		if got, err := DecodeJobRecord(uncheckedJob(content)); err == nil || !reflect.DeepEqual(got, Job{}) {
			t.Fatal("checksum granted invalid lifecycle")
		}
	}
	bad := jobFixture()
	bad.History = make([]JobTransition, maxJobTransitions+1)
	if b, err := EncodeJobRecord(bad); !errors.Is(err, ErrLimit) || b != nil {
		t.Fatal("history limit not enforced")
	}
	var wire map[string]any
	_ = json.Unmarshal(uncheckedJob(jobGoldenContent), &wire)
	body := wire["job"].(map[string]any)
	event := body["history"].([]any)[0]
	rows := make([]any, maxJobTransitions+1)
	for i := range rows {
		rows[i] = event
	}
	body["history"] = rows
	oversizedHistory, _ := json.Marshal(wire)
	if got, err := DecodeJobRecord(oversizedHistory); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(got, Job{}) {
		t.Fatal("decoded history limit not enforced", err)
	}
	bad = jobFixture()
	bad.Intent.Change.Value = strings.Repeat("x", maxFieldBytes+1)
	if b, err := EncodeJobRecord(bad); !errors.Is(err, ErrLimit) || b != nil {
		t.Fatal("field budget not enforced")
	}
}

func FuzzJobRecord(f *testing.F) {
	for _, b := range [][]byte{uncheckedJob(jobGoldenContent), nil, []byte(`{"version":2}`)} {
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > maxStateRecordBytes+1 {
			return
		}
		before := append([]byte(nil), b...)
		j, err := DecodeJobRecord(b)
		if !bytes.Equal(before, b) {
			t.Fatal("input mutated")
		}
		if err != nil {
			if !reflect.DeepEqual(j, Job{}) {
				t.Fatal("partial job on denial")
			}
			return
		}
		encoded, err := EncodeJobRecord(j)
		if err != nil || len(encoded) > maxStateRecordBytes {
			t.Fatal("accepted job cannot encode", err)
		}
		again, err := DecodeJobRecord(encoded)
		if err != nil || !reflect.DeepEqual(j, again) {
			t.Fatal("accepted job cannot round trip", err)
		}
	})
}
