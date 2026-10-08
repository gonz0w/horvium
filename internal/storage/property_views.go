// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"unicode/utf8"
)

// EncodePropertyReview encodes a complete display projection only, never
// session/policy digests, a restorable Plan or an execution capability.
func EncodePropertyReview(r Review) ([]byte, error) {
	// This private carrier is solely the existing bounded consistency check. Its
	// capture time equals creation and its authority digests stay zero; none are
	// encoded, returned or used for provenance, admission or native eligibility.
	i, err := normalizePropertyIntent(PropertyIntent{review: r, capturedAt: r.CreatedAt})
	if err != nil {
		return nil, err
	}
	return marshalPropertyView(propertyReviewToWire(i.review))
}

// DecodePropertyReview returns an owned, consistency-checked display baseline.
// The original root-held plan/session/policy must still be checked independently.
func DecodePropertyReview(data []byte) (Review, error) {
	var w propertyReviewWire
	if err := decodePropertyView(data, &w); err != nil {
		return Review{}, err
	}
	r, err := propertyReviewFromWire(w)
	if err != nil {
		return Review{}, err
	}
	i, err := normalizePropertyIntent(PropertyIntent{review: r, capturedAt: r.CreatedAt})
	if err != nil {
		return Review{}, err
	}
	return i.Review(), nil
}

// EncodePropertyJobView copies bounded lifecycle metadata, without the durable
// record envelope/checksum or complete private intent. It proves no admission.
func EncodePropertyJobView(j Job) ([]byte, error) {
	if err := validateJob(j); err != nil {
		return nil, err
	}
	return marshalPropertyView(jobWire(j))
}

// DecodePropertyJobView checks the complete existing lifecycle rules, supplying
// no durable publication, original execution capability or reservation release.
func DecodePropertyJobView(data []byte) (Job, error) {
	var w jobBody
	if err := decodePropertyView(data, &w); err != nil {
		return Job{}, err
	}
	if len(w.History) > maxJobTransitions {
		return Job{}, ErrLimit
	}
	j := Job{ID: w.ID, Intent: JobIntent{PlanID: w.Intent.PlanID, SubmissionID: w.Intent.SubmissionID, PolicyRevision: w.Intent.PolicyRevision, PoolGUID: w.Intent.PoolGUID, Change: Intent{FilesystemGUID: w.Intent.Change.FilesystemGUID, Property: w.Intent.Change.Property, Value: w.Intent.Change.Value}}, History: make([]JobTransition, len(w.History))}
	for n, e := range w.History {
		j.History[n] = JobTransition{Revision: e.Revision, At: e.At, State: e.State, Reason: e.Reason, ObservationDigest: e.ObservationDigest}
	}
	if err := validateJob(j); err != nil {
		return Job{}, err
	}
	return j, nil
}
func marshalPropertyView(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, ErrStateRecord
	}
	if len(data) > maxStateRecordBytes {
		return nil, ErrLimit
	}
	return data, nil
}
func decodePropertyView(data []byte, out any) error {
	if len(data) > maxStateRecordBytes {
		return ErrLimit
	}
	if !utf8.Valid(data) {
		return ErrStateRecord
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if ownershipJSONValue(d, reflect.TypeOf(out).Elem()) != nil {
		return ErrStateRecord
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrStateRecord
	}
	if json.Unmarshal(data, out) != nil {
		return ErrStateRecord
	}
	return nil
}
