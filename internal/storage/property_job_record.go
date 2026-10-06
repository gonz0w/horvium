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

type propertyJobContent struct {
	Version uint64             `json:"version"`
	Kind    string             `json:"kind"`
	Job     jobBody            `json:"job"`
	Intent  propertyIntentWire `json:"intent"`
}
type propertyJobRecord struct {
	Version  uint64             `json:"version"`
	Kind     string             `json:"kind"`
	Job      jobBody            `json:"job"`
	Intent   propertyIntentWire `json:"intent"`
	Checksum string             `json:"checksum"`
}

// EncodePropertyJob checks a single bounded unit of lifecycle and baseline
// metadata. No native reader/publisher accepts this distinct record kind yet.
func EncodePropertyJob(p PropertyJob) ([]byte, error) {
	normalized, err := normalizePropertyJob(p)
	if err != nil {
		return nil, err
	}
	return encodePropertyJob(normalized)
}

func encodePropertyJob(p PropertyJob) ([]byte, error) {
	j, i := jobWire(p.job), propertyIntentToWire(p.intent)
	b, err := json.Marshal(propertyJobRecord{Version: 1, Kind: "property-job", Job: j, Intent: i, Checksum: propertyJobChecksum(j, i)})
	if err != nil {
		return nil, ErrStateRecord
	}
	if len(b) > maxStateRecordBytes {
		return nil, ErrLimit
	}
	return b, nil
}

// DecodePropertyJob returns independent metadata or zero on any denial. Strict
// shape and checksums establish consistency, never authentication or storage proof.
func DecodePropertyJob(data []byte) (PropertyJob, error) {
	if len(data) > maxStateRecordBytes {
		return PropertyJob{}, ErrLimit
	}
	if !utf8.Valid(data) {
		return PropertyJob{}, ErrStateRecord
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if ownershipJSONValue(d, reflect.TypeFor[propertyJobRecord]()) != nil {
		return PropertyJob{}, ErrStateRecord
	}
	if _, err := d.Token(); err != io.EOF {
		return PropertyJob{}, ErrStateRecord
	}
	var w propertyJobRecord
	if json.Unmarshal(data, &w) != nil || w.Version != 1 || w.Kind != "property-job" {
		return PropertyJob{}, ErrStateRecord
	}
	if len(w.Job.History) > maxJobTransitions {
		return PropertyJob{}, ErrLimit
	}
	j := Job{ID: w.Job.ID, Intent: JobIntent{PlanID: w.Job.Intent.PlanID, SubmissionID: w.Job.Intent.SubmissionID, PolicyRevision: w.Job.Intent.PolicyRevision, PoolGUID: w.Job.Intent.PoolGUID, Change: Intent{w.Job.Intent.Change.FilesystemGUID, w.Job.Intent.Change.Property, w.Job.Intent.Change.Value}}, History: make([]JobTransition, len(w.Job.History))}
	for n, event := range w.Job.History {
		j.History[n] = JobTransition{event.Revision, event.At, event.State, event.Reason, event.ObservationDigest}
	}
	i, err := propertyIntentFromWire(w.Intent)
	if err != nil {
		return PropertyJob{}, err
	}
	p, err := BindPropertyJob(j, i)
	if err != nil {
		return PropertyJob{}, err
	}
	if !validJobDigest(w.Checksum) || w.Checksum != propertyJobChecksum(jobWire(p.job), propertyIntentToWire(p.intent)) {
		return PropertyJob{}, ErrStateRecord
	}
	return p, nil
}

func propertyJobChecksum(j jobBody, i propertyIntentWire) string {
	b, _ := json.Marshal(propertyJobContent{Version: 1, Kind: "property-job", Job: j, Intent: i})
	sum := sha256.Sum256(append([]byte("horvium:property-job:v1\n"), b...))
	return hex.EncodeToString(sum[:])
}
