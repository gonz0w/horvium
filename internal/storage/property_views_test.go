// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPropertyReviewViewRoundTrip(t *testing.T) {
	i, plan, _, _ := propertyIntentFixture(t)
	want := i.Review()
	data, err := EncodePropertyReview(want)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"fixture-session-secret", "session_digest", "policy_digest", "captured_at", `"checksum":`} {
		if bytes.Contains(data, []byte(secret)) {
			t.Fatal("private metadata crossed view", secret)
		}
	}
	got, err := DecodePropertyReview(data)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("lost full review", err)
	}
	got.Context[0].Name = "changed"
	got.States[0].Mountpoint.Effective = "/changed"
	got.Affected[0].GUID = "999"
	got.Versions[0].OwnerGUID = "999"
	got.Features.Values[0].State = "changed"
	got.Topology.Nodes[0].Address = "changed"
	if !reflect.DeepEqual(i.Review(), want) || !reflect.DeepEqual(plan.Review(), want) {
		t.Fatal("aliased original baseline")
	}
	if _, err := DecodePropertyIntent(data); err == nil {
		t.Fatal("display became captured intent")
	}
	if _, err := DecodePropertyJob(data); err == nil {
		t.Fatal("display became durable job")
	}
	again, err := DecodePropertyReview(data)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := EncodePropertyReview(again)
	if err != nil || !bytes.Equal(canonical, data) {
		t.Fatal("noncanonical round trip", err)
	}
}
func TestPropertyReviewViewConsistency(t *testing.T) {
	for name, mutate := range map[string]func(*Review){"plan-id": func(r *Review) { r.ID = "forged" }, "expiry": func(r *Review) { r.ExpiresAt = r.CreatedAt.Add(time.Minute) }, "revision": func(r *Review) { r.PolicyRevision = 0 }, "affected": func(r *Review) { r.Affected = r.Affected[:1] }, "property-source": func(r *Review) { r.Context[0].Properties.Quota.Source = Temporary }, "version-owner": func(r *Review) { r.Versions[0].OwnerGUID = "999" }, "features": func(r *Review) { r.Features.Values = r.Features.Values[:1] }, "topology": func(r *Review) { r.Topology.Nodes[0].GUID = "0" }} {
		t.Run(name, func(t *testing.T) {
			i, _, _, _ := propertyIntentFixture(t)
			r := i.Review()
			mutate(&r)
			if data, err := EncodePropertyReview(r); err == nil || data != nil {
				t.Fatal("inconsistent view", err)
			}
		})
	}
}
func TestPropertyViewsClosedSchema(t *testing.T) {
	i, _, _, _ := propertyIntentFixture(t)
	review, err := EncodePropertyReview(i.Review())
	if err != nil {
		t.Fatal(err)
	}
	job, err := EncodePropertyJobView(jobFixture())
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func([]byte) []byte{"unknown": func(b []byte) []byte { return bytes.Replace(b, []byte(`{"id":`), []byte(`{"uid":0,"id":`), 1) }, "duplicate": func(b []byte) []byte { return bytes.Replace(b, []byte(`{"id":`), []byte(`{"id":"forged","id":`), 1) }, "case": func(b []byte) []byte { return bytes.Replace(b, []byte(`"id":`), []byte(`"ID":`), 1) }, "null": func(b []byte) []byte {
		return bytes.Replace(b, []byte(`"pool_guid":"100"`), []byte(`"pool_guid":null`), 1)
	}, "missing": func(b []byte) []byte { return bytes.Replace(b, []byte(`"pool_guid":"100",`), nil, 1) }, "trailing": func(b []byte) []byte { return append(b, []byte(`{}`)...) }} {
		t.Run(name, func(t *testing.T) {
			if got, err := DecodePropertyReview(mutate(bytes.Clone(review))); err == nil || !reflect.DeepEqual(got, Review{}) {
				t.Fatal("malformed review", err)
			}
			if got, err := DecodePropertyJobView(mutate(bytes.Clone(job))); err == nil || !reflect.DeepEqual(got, Job{}) {
				t.Fatal("malformed job", err)
			}
		})
	}
	for _, data := range [][]byte{nil, {}, []byte("null"), []byte("[]"), {255}} {
		if _, err := DecodePropertyReview(data); err == nil {
			t.Fatal("invalid JSON")
		}
		if _, err := DecodePropertyJobView(data); err == nil {
			t.Fatal("invalid JSON")
		}
	}
	if _, err := DecodePropertyReview(job); err == nil {
		t.Fatal("job as review")
	}
	if _, err := DecodePropertyJobView(review); err == nil {
		t.Fatal("review as job")
	}
}
func TestPropertyJobViewLifecycle(t *testing.T) {
	j := jobFixture()
	data, err := EncodePropertyJobView(j)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodePropertyJobView(data)
	if err != nil || !reflect.DeepEqual(got, j) {
		t.Fatal("job projection", err)
	}
	got.History[0].Reason = "changed"
	if j.History[0].Reason != "admitted" {
		t.Fatal("alias")
	}
	if _, err := DecodeJobRecord(data); err == nil {
		t.Fatal("view became durable record")
	}
	j.History = append(j.History, JobTransition{Revision: 2, At: j.History[0].At + 1, State: JobSucceeded, Reason: "verified", ObservationDigest: strings.Repeat("c", 64)})
	if data, err := EncodePropertyJobView(j); err == nil || data != nil {
		t.Fatal("impossible lifecycle")
	}
	j = jobFixture()
	j.History = append(j.History, JobTransition{Revision: 2, At: j.History[0].At + 1, State: JobApplying, Reason: "launch-intent"})
	data, err = EncodePropertyJobView(j)
	if err != nil {
		t.Fatal(err)
	}
	got, err = DecodePropertyJobView(data)
	if err != nil || !reflect.DeepEqual(got, j) {
		t.Fatal("lost later history")
	}
}
func TestPropertyViewsBounds(t *testing.T) {
	oversized := bytes.Repeat([]byte(" "), maxStateRecordBytes+1)
	if _, err := DecodePropertyReview(oversized); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	if _, err := DecodePropertyJobView(oversized); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	i, _, _, _ := propertyIntentFixture(t)
	r := i.Review()
	r.Context = make([]Filesystem, maxRecords+1)
	if data, err := EncodePropertyReview(r); !errors.Is(err, ErrLimit) || data != nil {
		t.Fatal("rows unbounded", err)
	}
	r = i.Review()
	r.PoolName = strings.Repeat("a", maxFieldBytes+1)
	if data, err := EncodePropertyReview(r); !errors.Is(err, ErrLimit) || data != nil {
		t.Fatal("strings unbounded", err)
	}
	j := jobFixture()
	j.History = make([]JobTransition, maxJobTransitions+1)
	if data, err := EncodePropertyJobView(j); !errors.Is(err, ErrLimit) || data != nil {
		t.Fatal("history unbounded", err)
	}
}
func TestPropertyViewsDurableCompatibility(t *testing.T) {
	// Golden hashes were independently generated from merged PR 95's synthetic
	// fixtures before extracting the shared review converters. No live metadata.
	for name, want := range map[string]string{
		"intent":       "f0d929adbfe7f1421541caf03422b04f068c1b849a511d76a011cce8bc30b269",
		"property-job": "bcf8a9112fdbfcd7ce22fb50db276755c2e117722f89c1f49be622e9da60365f",
	} {
		t.Run(name, func(t *testing.T) {
			var data []byte
			var err error
			if name == "intent" {
				i, _, _, _ := propertyIntentFixture(t)
				data, err = EncodePropertyIntent(i)
			} else {
				data, err = EncodePropertyJob(propertyJobFixture(t))
			}
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != want {
				t.Fatal("existing durable format changed", err)
			}
		})
	}
}

func FuzzPropertyViews(f *testing.F) {
	i, _, _, _ := propertyIntentFixture(f)
	review, err := EncodePropertyReview(i.Review())
	if err != nil {
		f.Fatal(err)
	}
	job, err := EncodePropertyJobView(jobFixture())
	if err != nil {
		f.Fatal(err)
	}
	f.Add(uint8(0), review)
	f.Add(uint8(1), job)
	f.Fuzz(func(t *testing.T, kind uint8, data []byte) {
		if len(data) > maxStateRecordBytes+1 {
			return
		}
		if kind%2 == 0 {
			got, err := DecodePropertyReview(data)
			if err != nil {
				if !reflect.DeepEqual(got, Review{}) {
					t.Fatal("partial view")
				}
				return
			}
			encoded, err := EncodePropertyReview(got)
			if err != nil {
				t.Fatal(err)
			}
			again, err := DecodePropertyReview(encoded)
			if err != nil || !reflect.DeepEqual(again, got) {
				t.Fatal("review round trip")
			}
		} else {
			got, err := DecodePropertyJobView(data)
			if err != nil {
				if !reflect.DeepEqual(got, Job{}) {
					t.Fatal("partial job")
				}
				return
			}
			encoded, err := EncodePropertyJobView(got)
			if err != nil {
				t.Fatal(err)
			}
			again, err := DecodePropertyJobView(encoded)
			if err != nil || !reflect.DeepEqual(again, got) {
				t.Fatal("job round trip")
			}
		}
	})
}
