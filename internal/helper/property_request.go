// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"encoding/json"
	"io"
	"reflect"
	"strconv"

	"github.com/gonz0w/horvium/internal/storage"
)

// Parsed assertions only: none of these fields attests a peer, session, policy,
// original plan or permission. Runtime Server/Client do not consume this codec.
type propertyRequest struct {
	ID, Operation, PoolGUID string
	Intent                  storage.Intent
	PlanID, SubmissionID    string
}
type propertyEnvelope struct {
	Version   int             `json:"version"`
	ID        string          `json:"request_id"`
	Operation string          `json:"operation"`
	Data      json.RawMessage `json:"data"`
}
type propertyReviewData struct {
	PoolGUID       string           `json:"pool_guid"`
	FilesystemGUID string           `json:"filesystem_guid"`
	Property       storage.Property `json:"property"`
	Value          string           `json:"value"`
}
type propertySubmissionData struct {
	PoolGUID     string `json:"pool_guid"`
	PlanID       string `json:"plan_id"`
	SubmissionID string `json:"submission_id"`
}

func propertyRequestLimits() Limits { return Limits{1024, 2, 64, 3} }
func validPropertyPool(guid string) bool {
	if len(guid) < 1 || len(guid) > 20 {
		return false
	}
	n, err := strconv.ParseUint(guid, 10, 64)
	return err == nil && n > 0 && strconv.FormatUint(n, 10) == guid
}
func validPropertyReview(d propertyReviewData) bool {
	return validPropertyPool(d.PoolGUID) && storage.ValidatePropertyIntent(storage.Intent{FilesystemGUID: d.FilesystemGUID, Property: d.Property, Value: d.Value}) == nil
}
func validPropertySubmission(d propertySubmissionData) bool {
	return validPropertyPool(d.PoolGUID) && validApplicationContext(d.PlanID) && validApplicationContext(d.SubmissionID)
}
func writePropertyRequest(w io.Writer, id, operation string, data any) error {
	if w == nil || !validID(id) {
		return ErrProtocol
	}
	l := propertyRequestLimits()
	// Callers validate the typed selection; bound it before JSON allocation too.
	if _, err := valueBound(reflect.ValueOf(data), l, 0, l.MaxBytes); err != nil {
		return err
	}
	b, _ := newBudget(l)
	raw, err := json.Marshal(data)
	if err != nil {
		return ErrProtocol
	}
	return b.write(w, propertyEnvelope{2, id, operation, raw})
}

// WritePropertyReview encodes an allowlisted literal intent, without creating a
// plan or authorizing a native read/write. The owner must independently check the
// original application session and kernel peer before any exchange.
func WritePropertyReview(w io.Writer, id, poolGUID string, intent storage.Intent) error {
	d := propertyReviewData{poolGUID, intent.FilesystemGUID, intent.Property, intent.Value}
	if !validPropertyReview(d) {
		return ErrProtocol
	}
	return writePropertyRequest(w, id, "property-review", d)
}

// WritePropertySubmission names a root-held plan and an idempotency identifier.
// It carries no replacement intent, policy, execution proof or client-chosen job.
func WritePropertySubmission(w io.Writer, id, poolGUID, planID, submissionID string) error {
	d := propertySubmissionData{poolGUID, planID, submissionID}
	if !validPropertySubmission(d) {
		return ErrProtocol
	}
	return writePropertyRequest(w, id, "property-submit", d)
}

// Reads exactly one bounded frame. id must come from the original negotiated
// connection, never the incoming request. Later bytes remain for its owner; a
// successful decode supplies no socket-liveness or transaction-lifetime proof.
func readPropertyRequest(r io.Reader, id string) (propertyRequest, error) {
	if r == nil || !validID(id) {
		return propertyRequest{}, ErrProtocol
	}
	l := propertyRequestLimits()
	b, _ := newBudget(l)
	data, err := b.read(r)
	if err != nil {
		return propertyRequest{}, err
	}
	var e propertyEnvelope
	if err := decode(data, &e, l); err != nil {
		return propertyRequest{}, err
	}
	if e.Version != 2 || e.ID != id {
		return propertyRequest{}, ErrProtocol
	}
	switch e.Operation {
	case "property-review":
		var d propertyReviewData
		if err := decode(e.Data, &d, l); err != nil {
			return propertyRequest{}, err
		}
		if !validPropertyReview(d) {
			return propertyRequest{}, ErrProtocol
		}
		return propertyRequest{ID: e.ID, Operation: e.Operation, PoolGUID: d.PoolGUID, Intent: storage.Intent{FilesystemGUID: d.FilesystemGUID, Property: d.Property, Value: d.Value}}, nil
	case "property-submit":
		var d propertySubmissionData
		if err := decode(e.Data, &d, l); err != nil {
			return propertyRequest{}, err
		}
		if !validPropertySubmission(d) {
			return propertyRequest{}, ErrProtocol
		}
		return propertyRequest{ID: e.ID, Operation: e.Operation, PoolGUID: d.PoolGUID, PlanID: d.PlanID, SubmissionID: d.SubmissionID}, nil
	}
	return propertyRequest{}, ErrProtocol
}
