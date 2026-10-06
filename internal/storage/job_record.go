// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"reflect"
	"unicode/utf8"
)

// Explicit wire fields isolate the metadata format from planning Go structs.
type jobChange struct {
	FilesystemGUID string   `json:"filesystem_guid"`
	Property       Property `json:"property"`
	Value          string   `json:"value"`
}
type jobIntent struct {
	PlanID         string    `json:"plan_id"`
	SubmissionID   string    `json:"submission_id"`
	PolicyRevision uint64    `json:"policy_revision"`
	PoolGUID       string    `json:"pool_guid"`
	Change         jobChange `json:"change"`
}
type jobTransition struct {
	Revision          uint64   `json:"revision"`
	At                uint64   `json:"at"`
	State             JobState `json:"state"`
	Reason            string   `json:"reason"`
	ObservationDigest string   `json:"observation_digest"`
}
type jobBody struct {
	ID      string          `json:"id"`
	Intent  jobIntent       `json:"intent"`
	History []jobTransition `json:"history"`
}
type jobContent struct {
	Version uint64  `json:"version"`
	Kind    string  `json:"kind"`
	Job     jobBody `json:"job"`
}
type jobRecord struct {
	Version  uint64  `json:"version"`
	Kind     string  `json:"kind"`
	Job      jobBody `json:"job"`
	Checksum string  `json:"checksum"`
}

// EncodeJobRecord validates/copies bounded lifecycle metadata only. It neither
// stores a job nor captures the full reviewed native baseline needed by future
// admission/recovery. No checksum, identifier or state label grants execution.
func EncodeJobRecord(j Job) ([]byte, error) {
	if err := validateJob(j); err != nil {
		return nil, err
	}
	w := jobWire(j)
	b, err := json.Marshal(jobRecord{Version: 1, Kind: "job", Job: w, Checksum: jobChecksum(w)})
	if err != nil {
		return nil, ErrStateRecord
	}
	if len(b) > maxStateRecordBytes {
		return nil, ErrLimit
	}
	return b, nil
}

// DecodeJobRecord returns independent validated metadata or the zero Job. All
// fields are required and nonnullable; unsupported/damaged state is never reset.
func DecodeJobRecord(data []byte) (Job, error) {
	if len(data) > maxStateRecordBytes {
		return Job{}, ErrLimit
	}
	if !utf8.Valid(data) {
		return Job{}, ErrStateRecord
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if ownershipJSONValue(d, reflect.TypeFor[jobRecord]()) != nil {
		return Job{}, ErrStateRecord
	}
	if _, err := d.Token(); err != io.EOF {
		return Job{}, ErrStateRecord
	}
	var r jobRecord
	if json.Unmarshal(data, &r) != nil || r.Version != 1 || r.Kind != "job" {
		return Job{}, ErrStateRecord
	}
	if len(r.Job.History) > maxJobTransitions {
		return Job{}, ErrLimit
	}
	j := Job{ID: r.Job.ID, Intent: JobIntent{PlanID: r.Job.Intent.PlanID, SubmissionID: r.Job.Intent.SubmissionID, PolicyRevision: r.Job.Intent.PolicyRevision, PoolGUID: r.Job.Intent.PoolGUID, Change: Intent{FilesystemGUID: r.Job.Intent.Change.FilesystemGUID, Property: r.Job.Intent.Change.Property, Value: r.Job.Intent.Change.Value}}, History: make([]JobTransition, len(r.Job.History))}
	for i, event := range r.Job.History {
		j.History[i] = JobTransition{Revision: event.Revision, At: event.At, State: event.State, Reason: event.Reason, ObservationDigest: event.ObservationDigest}
	}
	if err := validateJob(j); err != nil {
		return Job{}, err
	}
	if len(r.Checksum) != 64 || r.Checksum != jobChecksum(jobWire(j)) {
		return Job{}, ErrStateRecord
	}
	return j, nil
}

func jobWire(j Job) jobBody {
	w := jobBody{ID: j.ID, Intent: jobIntent{PlanID: j.Intent.PlanID, SubmissionID: j.Intent.SubmissionID, PolicyRevision: j.Intent.PolicyRevision, PoolGUID: j.Intent.PoolGUID, Change: jobChange{FilesystemGUID: j.Intent.Change.FilesystemGUID, Property: j.Intent.Change.Property, Value: j.Intent.Change.Value}}, History: make([]jobTransition, len(j.History))}
	for i, event := range j.History {
		w.History[i] = jobTransition{Revision: event.Revision, At: event.At, State: event.State, Reason: event.Reason, ObservationDigest: event.ObservationDigest}
	}
	return w
}

func jobChecksum(j jobBody) string {
	b, _ := json.Marshal(jobContent{Version: 1, Kind: "job", Job: j})
	h := sha256.Sum256(append([]byte("horvium:job:v1\n"), b...))
	return hex.EncodeToString(h[:])
}
