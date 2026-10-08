// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/gonz0w/horvium/internal/storage"
	"io"
	"strings"
	"testing"
)

func reviewPayload() string {
	return `{"version":2,"request_id":"property-1","operation":"property-review","data":{"pool_guid":"123","filesystem_guid":"456","property":"quota","value":"4096"}}`
}
func submissionPayload() string {
	return `{"version":2,"request_id":"property-1","operation":"property-submit","data":{"pool_guid":"123","plan_id":"` + strings.Repeat("a", 64) + `","submission_id":"` + strings.Repeat("b", 64) + `"}}`
}
func TestPropertyRequestRoundTrip(t *testing.T) {
	for _, intent := range []storage.Intent{{FilesystemGUID: "456", Property: storage.Quota, Value: "4096"}, {FilesystemGUID: "456", Property: storage.RefQuota, Value: "none"}, {FilesystemGUID: "18446744073709551615", Property: storage.Quota, Value: "18446744073709551615"}, {FilesystemGUID: "456", Property: storage.Compression, Value: "off"}, {FilesystemGUID: "456", Property: storage.Compression, Value: "lz4"}} {
		t.Run(string(intent.Property)+"-"+intent.Value, func(t *testing.T) {
			var b bytes.Buffer
			if err := WritePropertyReview(&b, "property-1", "123", intent); err != nil {
				t.Fatal(err)
			}
			got, err := readPropertyRequest(&b, "property-1")
			want := propertyRequest{ID: "property-1", Operation: "property-review", PoolGUID: "123", Intent: intent}
			if err != nil || got != want || b.Len() != 0 {
				t.Fatal(got, err)
			}
		})
	}
	var b bytes.Buffer
	if err := WritePropertySubmission(&b, "property-1", "123", strings.Repeat("a", 64), strings.Repeat("b", 64)); err != nil {
		t.Fatal(err)
	}
	got, err := readPropertyRequest(&b, "property-1")
	want := propertyRequest{ID: "property-1", Operation: "property-submit", PoolGUID: "123", PlanID: strings.Repeat("a", 64), SubmissionID: strings.Repeat("b", 64)}
	if err != nil || got != want {
		t.Fatal(got, err)
	}
}
func TestPropertyRequestClosedSchema(t *testing.T) {
	for name, change := range map[string]func(string) string{
		"old-version":    func(s string) string { return strings.Replace(s, `"version":2`, `"version":1`, 1) },
		"future-version": func(s string) string { return strings.Replace(s, `"version":2`, `"version":3`, 1) },
		"duplicate": func(s string) string {
			return strings.Replace(s, `"pool_guid":"123"`, `"pool_guid":"123","pool_guid":"123"`, 1)
		},
		"case":         func(s string) string { return strings.Replace(s, `"pool_guid"`, `"Pool_GUID"`, 1) },
		"missing":      func(s string) string { return strings.Replace(s, `"pool_guid":"123",`, "", 1) },
		"null":         func(s string) string { return strings.Replace(s, `"pool_guid":"123"`, `"pool_guid":null`, 1) },
		"number":       func(s string) string { return strings.Replace(s, `"pool_guid":"123"`, `"pool_guid":123`, 1) },
		"wrong-id":     func(s string) string { return strings.Replace(s, "property-1", "property-2", 1) },
		"invalid-utf8": func(s string) string { return strings.Replace(s, "property-1", "\xff", 1) },
		"trailing":     func(s string) string { return s + `{}` },
		"nested-claim": func(s string) string { return strings.Replace(s, `"pool_guid":"123"`, `"pool_guid":"123","uid":0`, 1) },
		"top-claim": func(s string) string {
			return strings.Replace(s, `"version":2`, `"version":2,"session_context":"asserted"`, 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, payload := range []string{reviewPayload(), submissionPayload()} {
				got, err := readPropertyRequest(bytes.NewReader(frame(change(payload))), "property-1")
				if err == nil || got != (propertyRequest{}) {
					t.Fatal("malformed request published", got, err)
				}
			}
		})
	}
	for _, claim := range []string{`"command":"zfs set"`, `"path":"/fixture"`, `"policy_revision":1`, `"properties":{"quota":"1"}`, `"job_id":"chosen"`, `"recovery_clear":true`} {
		t.Run(claim, func(t *testing.T) {
			for _, payload := range []string{reviewPayload(), submissionPayload()} {
				got, err := readPropertyRequest(bytes.NewReader(frame(strings.Replace(payload, `"pool_guid":"123"`, `"pool_guid":"123",`+claim, 1))), "property-1")
				if err == nil || got != (propertyRequest{}) {
					t.Fatal("claim accepted")
				}
			}
		})
	}
	for _, operation := range []string{"inventory", "negotiate", "set", "destroy", "property-review-extra", ""} {
		t.Run("operation-"+operation, func(t *testing.T) {
			payload := strings.Replace(reviewPayload(), "property-review", operation, 1)
			if _, err := readPropertyRequest(bytes.NewReader(frame(payload)), "property-1"); err == nil {
				t.Fatal("unknown operation")
			}
		})
	}
	// Operation-specific payloads cannot be exchanged or blended.
	for _, payload := range []string{strings.Replace(reviewPayload(), "property-review", "property-submit", 1), strings.Replace(submissionPayload(), "property-submit", "property-review", 1), strings.Replace(submissionPayload(), `"pool_guid":"123"`, `"pool_guid":"123","value":"lz4"`, 1)} {
		if got, err := readPropertyRequest(bytes.NewReader(frame(payload)), "property-1"); err == nil || got != (propertyRequest{}) {
			t.Fatal("mixed operation", got)
		}
	}
}
func TestPropertyRequestEnvelopeDenials(t *testing.T) {
	for name, payload := range map[string]string{
		"duplicate-operation": strings.Replace(reviewPayload(), `"operation":"property-review"`, `"operation":"property-review","operation":"property-submit"`, 1),
		"missing-data":        `{"version":2,"request_id":"property-1","operation":"property-review"}`,
		"null-data":           `{"version":2,"request_id":"property-1","operation":"property-review","data":null}`,
		"array-data":          `{"version":2,"request_id":"property-1","operation":"property-review","data":[]}`,
		"floating-version":    strings.Replace(reviewPayload(), `"version":2`, `"version":2.0`, 1),
		"string-version":      strings.Replace(reviewPayload(), `"version":2`, `"version":"2"`, 1),
		"empty-data":          `{"version":2,"request_id":"property-1","operation":"property-submit","data":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := readPropertyRequest(bytes.NewReader(frame(payload)), "property-1"); err == nil || got != (propertyRequest{}) {
				t.Fatal("malformed envelope", got, err)
			}
		})
	}
}

func TestPropertyRequestInvalidSelections(t *testing.T) {
	for _, guid := range []string{"", "0", "01", "-1", "+1", " 1", "1 ", "1.0", "18446744073709551616", "fixture/pool"} {
		t.Run("guid-"+guid, func(t *testing.T) {
			for _, payload := range []string{reviewPayload(), submissionPayload()} {
				payload = strings.Replace(payload, `"pool_guid":"123"`, `"pool_guid":"`+guid+`"`, 1)
				if _, err := readPropertyRequest(bytes.NewReader(frame(payload)), "property-1"); err == nil {
					t.Fatal("invalid pool")
				}
			}
			var b bytes.Buffer
			if WritePropertyReview(&b, "property-1", guid, storage.Intent{FilesystemGUID: "456", Property: storage.Quota, Value: "1"}) == nil || b.Len() != 0 {
				t.Fatal("invalid pool emitted")
			}
		})
	}
	for _, intent := range []storage.Intent{{FilesystemGUID: "0", Property: storage.Quota, Value: "1"}, {FilesystemGUID: "01", Property: storage.Quota, Value: "1"}, {FilesystemGUID: "456", Property: "mountpoint", Value: "/fixture"}, {FilesystemGUID: "456", Property: storage.Quota, Value: "0"}, {FilesystemGUID: "456", Property: storage.Quota, Value: "1G"}, {FilesystemGUID: "456", Property: storage.RefQuota, Value: "01"}, {FilesystemGUID: "456", Property: storage.Quota, Value: "18446744073709551616"}, {FilesystemGUID: "456", Property: storage.Compression, Value: "gzip"}, {FilesystemGUID: "456", Property: storage.Compression, Value: "inherit"}, {FilesystemGUID: "456", Property: storage.Compression, Value: "none"}} {
		t.Run(string(intent.Property)+"-"+intent.Value+"-"+intent.FilesystemGUID, func(t *testing.T) {
			var b bytes.Buffer
			if WritePropertyReview(&b, "property-1", "123", intent) == nil || b.Len() != 0 {
				t.Fatal("invalid intent emitted")
			}
			payload := strings.Replace(reviewPayload(), `"filesystem_guid":"456"`, `"filesystem_guid":"`+intent.FilesystemGUID+`"`, 1)
			payload = strings.Replace(payload, `"property":"quota"`, `"property":"`+string(intent.Property)+`"`, 1)
			payload = strings.Replace(payload, `"value":"4096"`, `"value":"`+intent.Value+`"`, 1)
			if _, err := readPropertyRequest(bytes.NewReader(frame(payload)), "property-1"); err == nil {
				t.Fatal("invalid intent accepted")
			}
		})
	}
	for _, id := range []string{"", strings.Repeat("A", 64), strings.Repeat("a", 63), strings.Repeat("g", 64), "../fixture"} {
		t.Run("digest-"+id, func(t *testing.T) {
			for _, field := range []string{"plan_id", "submission_id"} {
				payload := submissionPayload()
				old := strings.Repeat("a", 64)
				if field == "submission_id" {
					old = strings.Repeat("b", 64)
				}
				payload = strings.Replace(payload, old, id, 1)
				if _, err := readPropertyRequest(bytes.NewReader(frame(payload)), "property-1"); err == nil {
					t.Fatal("invalid digest accepted")
				}
				var b bytes.Buffer
				plan, submission := strings.Repeat("a", 64), strings.Repeat("b", 64)
				if field == "plan_id" {
					plan = id
				} else {
					submission = id
				}
				if WritePropertySubmission(&b, "property-1", "123", plan, submission) == nil || b.Len() != 0 {
					t.Fatal("invalid digest emitted")
				}
			}
		})
	}
}
func TestPropertyRequestBoundsAndOwnership(t *testing.T) {
	for _, payload := range []string{reviewPayload(), submissionPayload()} {
		data := frame(payload)
		for cut := 0; cut < len(data); cut++ {
			if got, err := readPropertyRequest(bytes.NewReader(data[:cut]), "property-1"); err == nil || got != (propertyRequest{}) {
				t.Fatal("cut accepted", cut)
			}
		}
		tail := []byte("next-frame")
		r := bytes.NewReader(append(data, tail...))
		if _, err := readPropertyRequest(r, "property-1"); err != nil {
			t.Fatal(err)
		}
		rest, _ := io.ReadAll(r)
		if !bytes.Equal(rest, tail) {
			t.Fatal("consumed next frame")
		}
		if _, err := readRequest(bytes.NewReader(data), DefaultLimits()); err == nil {
			t.Fatal("inventory helper accepted property")
		}
	}
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, 1025)
	if _, err := readPropertyRequest(bytes.NewReader(header), "property-1"); !errors.Is(err, ErrLimit) {
		t.Fatal("allocation bound", err)
	}
	for _, payload := range []string{strings.Replace(reviewPayload(), "property-1", strings.Repeat("r", 65), 1), `{"version":2,"request_id":"property-1","operation":"property-review","data":{"a":{"b":{}}}}`} {
		if _, err := readPropertyRequest(bytes.NewReader(frame(payload)), "property-1"); err == nil {
			t.Fatal("field/depth bound")
		}
	}
	if _, err := readPropertyRequest(nil, "property-1"); err == nil {
		t.Fatal("nil reader")
	}
	if _, err := readPropertyRequest(bytes.NewReader(frame(reviewPayload())), "../fixture"); err == nil {
		t.Fatal("invalid expected correlation")
	}
	intent := storage.Intent{FilesystemGUID: "456", Property: storage.Quota, Value: "1"}
	if WritePropertyReview(nil, "property-1", "123", intent) == nil || WritePropertySubmission(nil, "property-1", "123", strings.Repeat("a", 64), strings.Repeat("b", 64)) == nil {
		t.Fatal("nil writer")
	}
	var b bytes.Buffer
	if WritePropertyReview(&b, "../fixture", "123", intent) == nil || b.Len() != 0 {
		t.Fatal("bad request ID emitted")
	}
	if err := WritePropertyReview(shortActionWriter{}, "property-1", "123", intent); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
}
func FuzzPropertyRequest(f *testing.F) {
	f.Add(frame(reviewPayload()))
	f.Add(frame(submissionPayload()))
	f.Add([]byte{0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1028 {
			return
		}
		got, err := readPropertyRequest(bytes.NewReader(data), "property-1")
		if err != nil {
			if got != (propertyRequest{}) {
				t.Fatal("partial request")
			}
			return
		}
		var b bytes.Buffer
		switch got.Operation {
		case "property-review":
			err = WritePropertyReview(&b, got.ID, got.PoolGUID, got.Intent)
		case "property-submit":
			err = WritePropertySubmission(&b, got.ID, got.PoolGUID, got.PlanID, got.SubmissionID)
		default:
			t.Fatal("unknown operation")
		}
		if err != nil {
			t.Fatal(err)
		}
		again, err := readPropertyRequest(&b, "property-1")
		if err != nil || again != got {
			t.Fatal("roundtrip", err)
		}
	})
}
