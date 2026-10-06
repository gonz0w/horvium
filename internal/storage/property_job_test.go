// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func propertyJobFixture(t testing.TB) PropertyJob {
	t.Helper()
	i, _, _, _ := propertyIntentFixture(t)
	j := jobFixture()
	j.Intent.PlanID = i.review.ID
	j.Intent.PolicyRevision = i.review.PolicyRevision
	j.Intent.PoolGUID = i.review.PoolGUID
	j.Intent.Change = i.review.Intent
	j.History[0].At = uint64(i.capturedAt.Add(time.Millisecond).UnixMilli())
	result, err := BindPropertyJob(j, i)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestPropertyJobRoundTripAndIsolation(t *testing.T) {
	p := propertyJobFixture(t)
	data, err := EncodePropertyJob(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodePropertyJob(data)
	if err != nil || !reflect.DeepEqual(got, p) {
		t.Fatal("lost job/baseline", err)
	}
	if !bytes.Equal(data, propertyJobResign(data)) {
		t.Fatal("wrong checksum domain")
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, data, "", "  ") != nil {
		t.Fatal("fixture indent")
	}
	presented, err := DecodePropertyJob(pretty.Bytes())
	if err != nil || !reflect.DeepEqual(presented, p) {
		t.Fatal("presentation changed metadata", err)
	}
	reordered := p.Intent()
	// Reverse independently owned baseline row order.
	sort.Slice(reordered.review.Context, func(a, b int) bool { return reordered.review.Context[a].Name > reordered.review.Context[b].Name })
	sort.Slice(reordered.review.States, func(a, b int) bool { return reordered.review.States[a].Name > reordered.review.States[b].Name })
	canonical, err := BindPropertyJob(p.Job(), reordered)
	again, e := EncodePropertyJob(canonical)
	if err != nil || e != nil || !bytes.Equal(again, data) {
		t.Fatal("row order changed canonical binding", err, e)
	}
	if bytes.Contains(data, []byte("fixture-session-secret")) {
		t.Fatal("raw session retained")
	}
	j, i := got.Job(), got.Intent()
	j.History[0].Reason = "changed"
	i.review.Context[0].Name = "changed"
	i.review.Topology.Nodes[0].Address = "changed"
	if !reflect.DeepEqual(got, p) {
		t.Fatal("accessor aliases private state")
	}
	original, baseline := p.Job(), p.Intent()
	bound, err := BindPropertyJob(original, baseline)
	if err != nil {
		t.Fatal(err)
	}
	original.History[0].Reason = "changed"
	baseline.review.States[0].Name = "changed"
	if !reflect.DeepEqual(bound, p) {
		t.Fatal("binding aliases input")
	}
	for name, decode := range map[string]func([]byte) error{
		"job":       func(b []byte) error { _, e := DecodeJobRecord(b); return e },
		"intent":    func(b []byte) error { _, e := DecodePropertyIntent(b); return e },
		"ownership": func(b []byte) error { _, e := DecodeOwnershipRecord(b); return e },
	} {
		t.Run(name, func(t *testing.T) {
			if decode(data) == nil {
				t.Fatal("new kind accepted by old decoder")
			}
		})
	}
	old, _ := EncodeJobRecord(p.Job())
	if result, e := DecodePropertyJob(old); e == nil || !reflect.DeepEqual(result, PropertyJob{}) {
		t.Fatal("old job fabricated baseline")
	}
	standalone, _ := EncodePropertyIntent(p.Intent())
	if result, e := DecodePropertyJob(standalone); e == nil || !reflect.DeepEqual(result, PropertyJob{}) {
		t.Fatal("standalone review became job")
	}
	damaged := bytes.Replace(data, []byte(`"checksum":"`), []byte(`"checksum":"0`), 1)
	if result, e := DecodePropertyJob(damaged); e == nil || !reflect.DeepEqual(result, PropertyJob{}) {
		t.Fatal("damaged checksum survived")
	}
}

func TestPropertyJobBindingDenials(t *testing.T) {
	p := propertyJobFixture(t)
	for _, tc := range []struct {
		name   string
		change func(*Job, *PropertyIntent)
	}{
		{"plan", func(j *Job, _ *PropertyIntent) { j.Intent.PlanID = strings.Repeat("d", 64) }},
		{"policy", func(j *Job, _ *PropertyIntent) { j.Intent.PolicyRevision++ }},
		{"pool", func(j *Job, _ *PropertyIntent) { j.Intent.PoolGUID = "999" }},
		{"target", func(j *Job, _ *PropertyIntent) { j.Intent.Change.FilesystemGUID = "102" }},
		{"property", func(j *Job, _ *PropertyIntent) { j.Intent.Change.Property = RefQuota }},
		{"value", func(j *Job, _ *PropertyIntent) { j.Intent.Change.Value = "none" }},
		{"before nanosecond capture", func(j *Job, i *PropertyIntent) { j.History[0].At = uint64(i.capturedAt.UnixMilli()) }},
		{"expiry", func(j *Job, i *PropertyIntent) {
			j.History[0].At = uint64(i.review.ExpiresAt.Add(time.Millisecond).UnixMilli())
		}},
		{"zero baseline", func(_ *Job, i *PropertyIntent) { *i = PropertyIntent{} }},
		{"invalid lifecycle", func(j *Job, _ *PropertyIntent) { j.History[0].State = JobSucceeded }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			j, i := p.Job(), p.Intent()
			tc.change(&j, &i)
			got, e := BindPropertyJob(j, i)
			if e == nil || !reflect.DeepEqual(got, PropertyJob{}) {
				t.Fatal("unbound metadata survived", e)
			}
		})
	}
	// Millisecond events may bind only once they reach the nanosecond capture.
	j, i := p.Job(), p.Intent()
	j.History[0].At = uint64(i.capturedAt.Truncate(time.Millisecond).Add(time.Millisecond).UnixMilli())
	if _, e := BindPropertyJob(j, i); e != nil {
		t.Fatal("ceil capture refused", e)
	}
	i.review.CreatedAt = i.review.CreatedAt.Truncate(time.Millisecond)
	i.review.ExpiresAt = i.review.CreatedAt.Add(5 * time.Minute)
	j.History[0].At = uint64(i.review.ExpiresAt.UnixMilli())
	if _, e := BindPropertyJob(j, i); e == nil {
		t.Fatal("exact expiry bound")
	}
}

func TestPropertyJobTransitionsAndSuccessors(t *testing.T) {
	p := propertyJobFixture(t)
	at := uint64(p.Intent().review.ExpiresAt.Add(time.Minute).UnixMilli())
	applying, e := TransitionPropertyJob(p, JobApplying, at, "launch-intent", "")
	if e != nil || ValidatePropertyJobSuccessor(p, applying) != nil {
		t.Fatal("successor", e)
	}
	if !reflect.DeepEqual(applying.Intent(), p.Intent()) || len(p.Job().History) != 1 {
		t.Fatal("changed baseline or predecessor")
	}
	observing, e := TransitionPropertyJob(applying, JobObserving, at, "native-observation", strings.Repeat("e", 64))
	if e != nil {
		t.Fatal(e)
	}
	succeeded, e := TransitionPropertyJob(observing, JobSucceeded, at, "verified", strings.Repeat("f", 64))
	if e != nil || !succeeded.Job().Terminal() {
		t.Fatal("valid lifecycle lost", e)
	}
	if got, e := TransitionPropertyJob(succeeded, JobApplying, at, "launch-intent", ""); e == nil || !reflect.DeepEqual(got, PropertyJob{}) {
		t.Fatal("terminal replay")
	}
	if ValidatePropertyJobSuccessor(p, observing) == nil || ValidatePropertyJobSuccessor(p, p) == nil {
		t.Fatal("skipped or stale successor")
	}
	for _, tc := range []struct {
		name   string
		change func(*PropertyJob)
	}{
		{"capture", func(p *PropertyJob) { p.intent.capturedAt = p.intent.capturedAt.Add(-time.Nanosecond) }},
		{"session binding", func(p *PropertyJob) { p.intent.sessionDigest[0] ^= 1 }},
		{"policy binding", func(p *PropertyJob) { p.intent.policyDigest[0] ^= 1 }},
		{"import identity", func(p *PropertyJob) { p.intent.review.Pool.LoadGUID = "999" }},
		{"old event", func(p *PropertyJob) { p.job.History[0].At++ }},
		{"job id", func(p *PropertyJob) { p.job.ID = strings.Repeat("d", 64) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next, _ := BindPropertyJob(applying.Job(), applying.Intent())
			tc.change(&next)
			if e := ValidatePropertyJobSuccessor(p, next); e == nil {
				t.Fatal("rewritten successor accepted")
			}
		})
	}
	cancelled, e := TransitionPropertyJob(p, JobCancelled, at, "operator-cancelled", "")
	if e != nil || !cancelled.Job().Terminal() {
		t.Fatal("prelaunch cancellation", e)
	}
	uncertain, e := TransitionPropertyJob(p, JobReconcileRequired, at, "review-expired", "")
	if e != nil || uncertain.Job().Terminal() {
		t.Fatal("uncertainty lost", e)
	}
	if _, e := TransitionPropertyJob(applying, JobCancelled, at, "operator-cancelled", ""); e == nil {
		t.Fatal("postlaunch cancellation")
	}
}

func propertyJobResign(data []byte) []byte {
	marker := bytes.LastIndex(data, []byte(`,"checksum":`))
	if marker < 0 {
		panic("missing test checksum")
	}
	content := append(append([]byte(nil), data[:marker]...), '}')
	sum := sha256.Sum256(append([]byte("horvium:property-job:v1\n"), content...))
	return append(append([]byte(nil), data[:marker]...), []byte(`,"checksum":"`+hex.EncodeToString(sum[:])+`"}`)...)
}

func TestPropertyJobSemanticDenials(t *testing.T) {
	original, _ := EncodePropertyJob(propertyJobFixture(t))
	for _, tc := range []struct{ name, old, next string }{
		{"future version", `"version":1`, `"version":2`},
		{"wrong kind", `"kind":"property-job"`, `"kind":"job"`},
		{"plan mismatch", `"plan_id":"` + strings.Repeat("a", 64) + `"`, `"plan_id":"` + strings.Repeat("d", 64) + `"`},
		{"pool mismatch", `"pool_guid":"100"`, `"pool_guid":"999"`},
		{"policy mismatch", `"policy_revision":1`, `"policy_revision":2`},
		{"before capture", `"at":` + jsonNumber(propertyJobFixture(t).Job().History[0].At), `"at":1000`},
		{"invalid lifecycle", `"reason":"admitted"`, `"reason":"verified"`},
		{"missing impact", `"affected":[{"guid":"101","name":"tank"},{"guid":"102","name":"tank/data"}]`, `"affected":[]`},
		{"integer exponent", `"revision":1`, `"revision":1e0`},
		{"integer fraction", `"revision":1`, `"revision":1.0`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutated := bytes.Replace(original, []byte(tc.old), []byte(tc.next), 1)
			if bytes.Equal(mutated, original) {
				t.Fatal("test mutation absent")
			}
			got, e := DecodePropertyJob(propertyJobResign(mutated))
			if e == nil || !reflect.DeepEqual(got, PropertyJob{}) {
				t.Fatal("valid checksum granted inconsistent metadata", e)
			}
		})
	}
}

func jsonNumber(n uint64) string { b, _ := json.Marshal(n); return string(b) }

func TestPropertyJobRequiredFields(t *testing.T) {
	data, _ := EncodePropertyJob(propertyJobFixture(t))
	for _, path := range propertyIntentObjectPaths(propertyIntentJSON(data)) {
		object := jobTestObject(propertyIntentJSON(data), path)
		keys := make([]string, 0, len(object))
		for k := range object {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		keys = append(keys, "unexpected")
		for _, key := range keys {
			mutations := []string{"missing", "null", "type", "case", "duplicate"}
			if key == "unexpected" {
				mutations = []string{"unknown"}
			}
			for _, mutation := range mutations {
				t.Run(strings.Join(path, "/")+"/"+key+"/"+mutation, func(t *testing.T) {
					root := propertyIntentJSON(data)
					obj := jobTestObject(root, path)
					value := obj[key]
					switch mutation {
					case "missing":
						delete(obj, key)
					case "null":
						obj[key] = nil
					case "type":
						if _, ok := value.(bool); ok {
							obj[key] = "true"
						} else {
							obj[key] = true
						}
					case "case":
						delete(obj, key)
						obj[strings.ToUpper(key)] = value
					case "duplicate":
						obj[key] = "horvium-duplicate-placeholder"
					case "unknown":
						obj[key] = true
					}
					mutated, _ := json.Marshal(root)
					if mutation == "duplicate" {
						v, _ := json.Marshal(value)
						k, _ := json.Marshal(key)
						replacement := append(append(append(append([]byte(nil), v...), ','), k...), ':')
						replacement = append(replacement, v...)
						mutated = bytes.Replace(mutated, []byte(`"horvium-duplicate-placeholder"`), replacement, 1)
					}
					got, e := DecodePropertyJob(mutated)
					if e == nil || !reflect.DeepEqual(got, PropertyJob{}) {
						t.Fatal("invalid shape survived", e)
					}
				})
			}
		}
	}
}

func TestPropertyJobBoundsAndPrecision(t *testing.T) {
	p := propertyJobFixture(t)
	p.job.Intent.PolicyRevision = ^uint64(0)
	p.intent.review.PolicyRevision = ^uint64(0)
	p.job.Intent.Change.Value = "18446744073709551615"
	p.intent.review.Intent.Value = p.job.Intent.Change.Value
	b, e := EncodePropertyJob(p)
	got, e2 := DecodePropertyJob(b)
	if e != nil || e2 != nil || !reflect.DeepEqual(p, got) {
		t.Fatal("uint64 lost", e, e2)
	}
	exact := append(append([]byte(nil), b...), bytes.Repeat([]byte(" "), maxStateRecordBytes-len(b))...)
	if _, e := DecodePropertyJob(exact); e != nil {
		t.Fatal("exact byte bound", e)
	}
	if got, e := DecodePropertyJob(append(exact, ' ')); !errors.Is(e, ErrLimit) || !reflect.DeepEqual(got, PropertyJob{}) {
		t.Fatal("oversize accepted", e)
	}
	for _, bad := range [][]byte{append(append([]byte(nil), b...), []byte(" {}")...), []byte{0xff}, []byte(`{"job":` + strings.Repeat("[", 10000))} {
		if got, e := DecodePropertyJob(bad); e == nil || !reflect.DeepEqual(got, PropertyJob{}) {
			t.Fatal("hostile input survived", e)
		}
	}
	p.job.History = make([]JobTransition, maxJobTransitions+1)
	if b, e := EncodePropertyJob(p); !errors.Is(e, ErrLimit) || b != nil {
		t.Fatal("history bound", e)
	}
	p = propertyJobFixture(t)
	p.intent.review.Context = make([]Filesystem, maxRecords+1)
	if b, e := EncodePropertyJob(p); !errors.Is(e, ErrLimit) || b != nil {
		t.Fatal("baseline bound", e)
	}
	if b, e := EncodePropertyJob(PropertyJob{}); e == nil || b != nil {
		t.Fatal("zero metadata encoded")
	}
}

func FuzzPropertyJob(f *testing.F) {
	b, _ := EncodePropertyJob(propertyJobFixture(f))
	f.Add(b)
	f.Add([]byte(`{"version":1}`))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		before := append([]byte(nil), data...)
		p, e := DecodePropertyJob(data)
		if !bytes.Equal(before, data) {
			t.Fatal("changed caller bytes")
		}
		if e != nil {
			if !reflect.DeepEqual(p, PropertyJob{}) {
				t.Fatal("metadata on error")
			}
			return
		}
		canonical, e := EncodePropertyJob(p)
		if e != nil || len(canonical) > maxStateRecordBytes {
			t.Fatal("unbounded accepted metadata", e)
		}
		again, e := DecodePropertyJob(canonical)
		if e != nil || !reflect.DeepEqual(p, again) {
			t.Fatal("unstable accepted record", e)
		}
	})
}
