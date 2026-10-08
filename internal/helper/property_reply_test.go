// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"errors"
	"github.com/gonz0w/horvium/internal/storage"
	"io"
	"reflect"
	"strings"
	"testing"
)

// Synthetic fixture generated from storage.propertyIntentFixture's review wire.
// Contains only tank fixture observations, without session/policy digests.
//
//go:embed testdata/property-review.json
var propertyReviewFixture []byte

func replyReview(t testing.TB) storage.Review {
	t.Helper()
	r, err := storage.DecodePropertyReview(propertyReviewFixture)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func replyJob() storage.Job {
	return storage.Job{ID: strings.Repeat("c", 64), Intent: storage.JobIntent{PlanID: strings.Repeat("a", 64), SubmissionID: strings.Repeat("b", 64), PolicyRevision: 1, PoolGUID: "100", Change: storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "1099511627776"}}, History: []storage.JobTransition{{Revision: 1, At: 1791154800123, State: storage.JobQueued, Reason: "admitted"}}}
}
func reviewReplyPayload() string {
	return `{"version":2,"request_id":"reply-1","type":"property-reviewed","data":` + strings.TrimSpace(string(propertyReviewFixture)) + `}`
}
func submissionReplyPayload(t testing.TB) string {
	t.Helper()
	data, err := storage.EncodePropertyJobView(replyJob())
	if err != nil {
		t.Fatal(err)
	}
	return `{"version":2,"request_id":"reply-1","type":"property-admitted","data":{"job":` + string(data) + `,"duplicate":false}}`
}
func readReplyForTest(r io.Reader, review bool) error {
	if review {
		_, err := ReadPropertyReview(r, "reply-1", "100", storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "1099511627776"})
		return err
	}
	_, _, err := ReadPropertySubmission(r, "reply-1", "100", strings.Repeat("a", 64), strings.Repeat("b", 64))
	return err
}
func TestPropertyRepliesRoundTrip(t *testing.T) {
	want := replyReview(t)
	var b bytes.Buffer
	if err := writePropertyReview(&b, "reply-1", want); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Bytes(), frame(reviewReplyPayload())) {
		t.Fatal("existing complete review wire changed")
	}
	got, err := ReadPropertyReview(&b, "reply-1", want.PoolGUID, want.Intent)
	if err != nil || !reflect.DeepEqual(got, want) || b.Len() != 0 {
		t.Fatal("review reply", err)
	}
	for _, duplicate := range []bool{false, true} {
		b.Reset()
		j := replyJob()
		if err := writePropertySubmission(&b, "reply-1", j, duplicate); err != nil {
			t.Fatal(err)
		}
		got, dup, err := ReadPropertySubmission(&b, "reply-1", j.Intent.PoolGUID, j.Intent.PlanID, j.Intent.SubmissionID)
		if err != nil || !reflect.DeepEqual(got, j) || dup != duplicate {
			t.Fatal("admission reply", err)
		}
	}
}
func TestPropertyRepliesCorrelation(t *testing.T) {
	want := replyReview(t)
	j := replyJob()
	for name, pair := range map[string][2]string{"request": {"other", "100"}, "pool": {"reply-1", "999"}, "invalid-id": {"../fixture", "100"}, "invalid-pool": {"reply-1", "0100"}} {
		t.Run(name, func(t *testing.T) {
			got, err := ReadPropertyReview(bytes.NewReader(frame(reviewReplyPayload())), pair[0], pair[1], want.Intent)
			if err == nil || !reflect.DeepEqual(got, storage.Review{}) {
				t.Fatal("foreign review")
			}
			job, dup, err := ReadPropertySubmission(bytes.NewReader(frame(submissionReplyPayload(t))), pair[0], pair[1], j.Intent.PlanID, j.Intent.SubmissionID)
			if err == nil || dup || !reflect.DeepEqual(job, storage.Job{}) {
				t.Fatal("foreign admission")
			}
		})
	}
	got, err := ReadPropertyReview(bytes.NewReader(frame(reviewReplyPayload())), "reply-1", "100", storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "4096"})
	if err == nil || !reflect.DeepEqual(got, storage.Review{}) {
		t.Fatal("substituted intent")
	}
	for _, pair := range [][2]string{{strings.Repeat("d", 64), j.Intent.SubmissionID}, {j.Intent.PlanID, strings.Repeat("d", 64)}, {"forged", j.Intent.SubmissionID}} {
		got, dup, err := ReadPropertySubmission(bytes.NewReader(frame(submissionReplyPayload(t))), "reply-1", "100", pair[0], pair[1])
		if err == nil || dup || !reflect.DeepEqual(got, storage.Job{}) {
			t.Fatal("substituted identifiers")
		}
	}
}
func TestPropertyRepliesClosedSchema(t *testing.T) {
	for name, change := range map[string]func(string) string{
		"old-version":    func(s string) string { return strings.Replace(s, `"version":2`, `"version":1`, 1) },
		"future-version": func(s string) string { return strings.Replace(s, `"version":2`, `"version":3`, 1) },
		"duplicate":      func(s string) string { return strings.Replace(s, `"version":2`, `"version":2,"version":2`, 1) },
		"case":           func(s string) string { return strings.Replace(s, `"request_id"`, `"Request_ID"`, 1) },
		"missing":        func(s string) string { return strings.Replace(s, `"request_id":"reply-1",`, "", 1) },
		"null":           func(s string) string { return strings.Replace(s, `"request_id":"reply-1"`, `"request_id":null`, 1) },
		"number":         func(s string) string { return strings.Replace(s, `"request_id":"reply-1"`, `"request_id":1`, 1) },
		"privilege": func(s string) string {
			return strings.Replace(s, `"version":2`, `"version":2,"recovery_clear":true`, 1)
		},
		"utf8":     func(s string) string { return strings.Replace(s, "reply-1", "\xff", 1) },
		"trailing": func(s string) string { return s + `{}` },
		"unknown-type": func(s string) string {
			return strings.Replace(strings.Replace(s, "property-reviewed", "executed", 1), "property-admitted", "executed", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			for n, payload := range []string{reviewReplyPayload(), submissionReplyPayload(t)} {
				if readReplyForTest(bytes.NewReader(frame(change(payload))), n == 0) == nil {
					t.Fatal("malformed reply")
				}
			}
		})
	}
	for _, payload := range []string{strings.Replace(reviewReplyPayload(), "property-reviewed", "property-admitted", 1), strings.Replace(submissionReplyPayload(t), "property-admitted", "property-reviewed", 1), strings.Replace(submissionReplyPayload(t), `,"duplicate":false`, "", 1), strings.Replace(submissionReplyPayload(t), `"duplicate":false`, `"duplicate":null`, 1), strings.Replace(submissionReplyPayload(t), `"duplicate":false`, `"duplicate":"false"`, 1), strings.Replace(submissionReplyPayload(t), `"duplicate":false`, `"duplicate":false,"launch":true`, 1)} {
		if readReplyForTest(bytes.NewReader(frame(payload)), false) == nil {
			t.Fatal("mixed/malformed admission")
		}
	}
}
func TestPropertyRepliesRefusals(t *testing.T) {
	for _, code := range []string{"denied", "unavailable", "review-required", "limit-exceeded", "uncertain"} {
		t.Run(code, func(t *testing.T) {
			var b bytes.Buffer
			if err := writePropertyRefusal(&b, "reply-1", code); err != nil {
				t.Fatal(err)
			}
			data := bytes.Clone(b.Bytes())
			for _, review := range []bool{true, false} {
				err := readReplyForTest(bytes.NewReader(data), review)
				var refused *PropertyRefusalError
				if !errors.As(err, &refused) || refused.Code != code {
					t.Fatal("lost refusal", err)
				}
			}
		})
	}
	for _, code := range []string{"", "success", "retry", "/fixture/error", strings.Repeat("x", 4097)} {
		var b bytes.Buffer
		if writePropertyRefusal(&b, "reply-1", code) == nil || b.Len() != 0 {
			t.Fatal("unbounded/unknown refusal")
		}
	}
	for _, payload := range []string{`{"version":2,"request_id":"reply-1","type":"refused","data":{"code":"denied","message":"/fixture/private"}}`, `{"version":2,"request_id":"other","type":"refused","data":{"code":"denied"}}`, `{"version":2,"request_id":"reply-1","type":"refused","data":{"code":"success"}}`} {
		err := readReplyForTest(bytes.NewReader(frame(payload)), true)
		var refused *PropertyRefusalError
		if err == nil || errors.As(err, &refused) {
			t.Fatal("malformed refusal classified")
		}
	}
}
func TestPropertyRepliesBoundsAndOwnership(t *testing.T) {
	for n, payload := range []string{reviewReplyPayload(), submissionReplyPayload(t)} {
		data := frame(payload)
		for _, cut := range []int{0, 1, 3, 4, 5, len(data) / 2, len(data) - 1} {
			if readReplyForTest(bytes.NewReader(data[:cut]), n == 0) == nil {
				t.Fatal("cut accepted", cut)
			}
		}
		tail := []byte("next-frame")
		r := bytes.NewReader(append(data, tail...))
		if err := readReplyForTest(r, n == 0); err != nil {
			t.Fatal(err)
		}
		rest, _ := io.ReadAll(r)
		if !bytes.Equal(rest, tail) {
			t.Fatal("consumed later bytes")
		}
	}
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, 65537)
	for _, review := range []bool{true, false} {
		if err := readReplyForTest(bytes.NewReader(header), review); !errors.Is(err, ErrLimit) {
			t.Fatal("allocation bound", err)
		}
		if readReplyForTest(nil, review) == nil {
			t.Fatal("nil reader")
		}
	}
	if writePropertyReview(nil, "reply-1", replyReview(t)) == nil || writePropertySubmission(nil, "reply-1", replyJob(), false) == nil || writePropertyRefusal(nil, "reply-1", "denied") == nil {
		t.Fatal("nil writer")
	}
	for _, write := range []func(io.Writer) error{func(w io.Writer) error { return writePropertyReview(w, "reply-1", storage.Review{}) }, func(w io.Writer) error { return writePropertySubmission(w, "reply-1", storage.Job{}, false) }, func(w io.Writer) error { return writePropertyReview(w, "../fixture", replyReview(t)) }} {
		var b bytes.Buffer
		if write(&b) == nil || b.Len() != 0 {
			t.Fatal("invalid reply emitted")
		}
	}
	if err := writePropertySubmission(shortActionWriter{}, "reply-1", replyJob(), false); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
}
func TestPropertyRepliesNoRetryOrOversizeOutput(t *testing.T) {
	var b bytes.Buffer
	if err := writePropertyReply(&b, "reply-1", "property-reviewed", bytes.Repeat([]byte(" "), 65537)); !errors.Is(err, ErrLimit) || b.Len() != 0 {
		t.Fatal("oversize frame emitted", err)
	}
	for _, write := range []func(io.Writer) error{
		func(w io.Writer) error { return writePropertyReview(w, "reply-1", replyReview(t)) },
		func(w io.Writer) error { return writePropertySubmission(w, "reply-1", replyJob(), false) },
		func(w io.Writer) error { return writePropertyRefusal(w, "reply-1", "uncertain") },
	} {
		w := &propertyFailWriter{}
		if err := write(w); !errors.Is(err, io.ErrClosedPipe) || w.calls != 1 {
			t.Fatal("failed write retried", err, w.calls)
		}
	}
	r := bytes.NewReader(frame(reviewReplyPayload()))
	if _, err := ReadPropertyReview(r, "reply-1", "100", storage.Intent{FilesystemGUID: "101", Property: "mountpoint", Value: "/fixture"}); err == nil || r.Len() != len(frame(reviewReplyPayload())) {
		t.Fatal("invalid expected selection consumed input")
	}
}

type propertyFailWriter struct{ calls int }

func (w *propertyFailWriter) Write([]byte) (int, error) { w.calls++; return 0, io.ErrClosedPipe }

func FuzzPropertyReplies(f *testing.F) {
	f.Add(uint8(0), frame(reviewReplyPayload()))
	f.Add(uint8(1), frame(submissionReplyPayload(f)))
	f.Add(uint8(0), frame(`{"version":2,"request_id":"reply-1","type":"refused","data":{"code":"denied"}}`))
	f.Fuzz(func(t *testing.T, kind uint8, data []byte) {
		if len(data) > 65540 {
			return
		}
		var b bytes.Buffer
		if kind%2 == 0 {
			got, err := ReadPropertyReview(bytes.NewReader(data), "reply-1", "100", storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "1099511627776"})
			if err != nil {
				if !reflect.DeepEqual(got, storage.Review{}) {
					t.Fatal("partial review")
				}
				return
			}
			if err := writePropertyReview(&b, "reply-1", got); err != nil {
				t.Fatal(err)
			}
		} else {
			got, dup, err := ReadPropertySubmission(bytes.NewReader(data), "reply-1", "100", strings.Repeat("a", 64), strings.Repeat("b", 64))
			if err != nil {
				if dup || !reflect.DeepEqual(got, storage.Job{}) {
					t.Fatal("partial admission")
				}
				return
			}
			if err := writePropertySubmission(&b, "reply-1", got, dup); err != nil {
				t.Fatal(err)
			}
		}
		if err := readReplyForTest(&b, kind%2 == 0); err != nil {
			t.Fatal("round trip", err)
		}
	})
}
