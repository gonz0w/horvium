// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"encoding/json"
	"github.com/gonz0w/horvium/internal/storage"
	"io"
)

// PropertyRefusalError carries a bounded peer assertion, not proof that work was
// never admitted or a retry instruction. No native error, path or credential is
// transmitted. Unknown/malformed refusals remain protocol errors instead.
type PropertyRefusalError struct{ Code string }

func (e *PropertyRefusalError) Error() string { return "property request refused: " + e.Code }

type propertyReplyEnvelope struct {
	Version int             `json:"version"`
	ID      string          `json:"request_id"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
}
type propertyAdmissionData struct {
	Job       json.RawMessage `json:"job"`
	Duplicate bool            `json:"duplicate"`
}
type propertyRefusalData struct {
	Code string `json:"code"`
}

func propertyReplyLimits() Limits { return Limits{64 << 10, 2, 4096, 16} }
func validPropertyRefusal(code string) bool {
	switch code {
	case "denied", "unavailable", "review-required", "limit-exceeded", "uncertain":
		return true
	}
	return false
}
func writePropertyReply(w io.Writer, id, kind string, data json.RawMessage) error {
	if w == nil || !validID(id) {
		return ErrProtocol
	}
	b, _ := newBudget(propertyReplyLimits())
	return b.write(w, propertyReplyEnvelope{2, id, kind, data})
}
func writePropertyReview(w io.Writer, id string, r storage.Review) error {
	if w == nil || !validID(id) {
		return ErrProtocol
	}
	data, err := storage.EncodePropertyReview(r)
	if err != nil {
		return err
	}
	return writePropertyReply(w, id, "property-reviewed", data)
}
func writePropertySubmission(w io.Writer, id string, j storage.Job, duplicate bool) error {
	if w == nil || !validID(id) {
		return ErrProtocol
	}
	job, err := storage.EncodePropertyJobView(j)
	if err != nil {
		return err
	}
	// The existing validated job view is bounded; the outer framing still checks
	// the complete payload before emitting any bytes. Never truncate history.
	data, err := json.Marshal(propertyAdmissionData{job, duplicate})
	if err != nil {
		return ErrProtocol
	}
	return writePropertyReply(w, id, "property-admitted", data)
}
func writePropertyRefusal(w io.Writer, id, code string) error {
	if w == nil || !validID(id) || !validPropertyRefusal(code) {
		return ErrProtocol
	}
	data, err := json.Marshal(propertyRefusalData{code})
	if err != nil {
		return ErrProtocol
	}
	return writePropertyReply(w, id, "refused", data)
}
func readPropertyReply(r io.Reader, id, kind string) (json.RawMessage, error) {
	if r == nil || !validID(id) {
		return nil, ErrProtocol
	}
	l := propertyReplyLimits()
	b, _ := newBudget(l)
	data, err := b.read(r)
	if err != nil {
		return nil, err
	}
	var e propertyReplyEnvelope
	if err := decode(data, &e, l); err != nil {
		return nil, err
	}
	if e.Version != 2 || e.ID != id {
		return nil, ErrProtocol
	}
	if e.Type == "refused" {
		var d propertyRefusalData
		if err := decode(e.Data, &d, l); err != nil {
			return nil, err
		}
		if !validPropertyRefusal(d.Code) {
			return nil, ErrProtocol
		}
		return nil, &PropertyRefusalError{Code: d.Code}
	}
	if e.Type != kind {
		return nil, ErrProtocol
	}
	return e.Data, nil
}

// ReadPropertyReview consumes one bounded, correlated display reply. Expected
// selection must come from the original request; parsed metadata supplies no
// authentic root peer, original plan/session, live eligibility or launch proof.
func ReadPropertyReview(r io.Reader, id, poolGUID string, intent storage.Intent) (storage.Review, error) {
	if !validPropertyPool(poolGUID) || storage.ValidatePropertyIntent(intent) != nil {
		return storage.Review{}, ErrProtocol
	}
	data, err := readPropertyReply(r, id, "property-reviewed")
	if err != nil {
		return storage.Review{}, err
	}
	review, err := storage.DecodePropertyReview(data)
	if err != nil {
		return storage.Review{}, err
	}
	if review.PoolGUID != poolGUID || review.Intent != intent {
		return storage.Review{}, ErrProtocol
	}
	return review, nil
}

// ReadPropertySubmission checks exact original pool/plan/submission correlation
// and complete lifecycle metadata. Duplicate is an untrusted reply assertion;
// neither it nor any state label proves durability or authorizes native retry.
func ReadPropertySubmission(r io.Reader, id, poolGUID, planID, submissionID string) (storage.Job, bool, error) {
	if !validPropertyPool(poolGUID) || !validApplicationContext(planID) || !validApplicationContext(submissionID) {
		return storage.Job{}, false, ErrProtocol
	}
	data, err := readPropertyReply(r, id, "property-admitted")
	if err != nil {
		return storage.Job{}, false, err
	}
	var d propertyAdmissionData
	if err := decode(data, &d, propertyReplyLimits()); err != nil {
		return storage.Job{}, false, err
	}
	job, err := storage.DecodePropertyJobView(d.Job)
	if err != nil {
		return storage.Job{}, false, err
	}
	if job.Intent.PoolGUID != poolGUID || job.Intent.PlanID != planID || job.Intent.SubmissionID != submissionID {
		return storage.Job{}, false, ErrProtocol
	}
	return job, d.Duplicate, nil
}
