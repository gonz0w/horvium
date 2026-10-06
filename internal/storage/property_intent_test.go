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

func propertyIntentFixture(t testing.TB) (PropertyIntent, Plan, Policy, Observation) {
	t.Helper()
	p, o, change := fixture()
	o.Identities = append(o.Identities, DatasetIdentity{GUID: "200", Name: "tank/data@snap", Type: SnapshotType, OwnerGUID: "102"})
	at := instant.Add(123456789 * time.Nanosecond)
	plan, err := Build(at, "fixture-session-secret", p, o, change)
	if err != nil {
		t.Fatal(err)
	}
	plan.review.ID = strings.Repeat("a", 64)
	captured, err := plan.CapturePropertyIntent(at.Add(time.Second), "fixture-session-secret", p, o)
	if err != nil {
		t.Fatal(err)
	}
	return captured, plan, p, o
}

func TestPropertyIntentCaptureBindingsAndPrecision(t *testing.T) {
	captured, plan, p, o := propertyIntentFixture(t)
	if !reflect.DeepEqual(captured.Review(), plan.Review()) {
		t.Fatal("review baseline lost")
	}
	data, err := EncodePropertyIntent(captured)
	if err != nil || bytes.Contains(data, []byte("fixture-session-secret")) {
		t.Fatal("raw session leaked", err)
	}
	var wire map[string]json.RawMessage
	if json.Unmarshal(data, &wire) != nil || string(wire["version"]) != "1" || string(wire["kind"]) != `"property-intent"` {
		t.Fatal("wrong envelope")
	}
	marker := bytes.LastIndex(data, []byte(`,"checksum":`))
	if marker < 0 {
		t.Fatal("missing damage checksum")
	}
	content := append(append([]byte(nil), data[:marker]...), '}')
	digest := sha256.Sum256(append([]byte("horvium:property-intent:v1\n"), content...))
	if string(wire["checksum"]) != `"`+hex.EncodeToString(digest[:])+`"` {
		t.Fatal("wrong checksum domain/canonical bytes")
	}
	got, err := DecodePropertyIntent(data)
	if err != nil || !reflect.DeepEqual(got, captured) || got.CompareBaseline(p, o) != nil {
		t.Fatal("precision or baseline lost", err)
	}
	view := got.Review()
	view.Context[0].Name = "changed"
	view.States[0].Mountpoint.Effective = "/changed"
	view.Affected[0].GUID = "999"
	view.Versions[0].OwnerGUID = "999"
	view.Features.Values[0].State = "changed"
	view.Topology.Nodes[0].Address = "changed"
	if !reflect.DeepEqual(captured.Review(), got.Review()) {
		t.Fatal("review accessor aliases immutable baseline")
	}
	again, err := EncodePropertyIntent(got)
	if err != nil || !bytes.Equal(data, again) {
		t.Fatal("noncanonical round trip")
	}
	if _, err := DecodeJobRecord(data); err == nil {
		t.Fatal("intent became lifecycle job")
	}
	jobData, _ := EncodeJobRecord(jobFixture())
	if _, err := DecodePropertyIntent(jobData); err == nil {
		t.Fatal("job became reviewed baseline")
	}
}

func TestPropertyIntentCaptureRefusals(t *testing.T) {
	_, plan, p, o := propertyIntentFixture(t)
	for _, tc := range []struct {
		name    string
		at      time.Time
		session string
		mutate  func(*Policy, *Observation)
		want    error
	}{
		{"before review", plan.review.CreatedAt.Add(-time.Nanosecond), "fixture-session-secret", nil, ErrExpired},
		{"exact expiry", plan.review.ExpiresAt, "fixture-session-secret", nil, ErrExpired},
		{"another session", plan.review.CreatedAt, "other-session", nil, ErrDenied},
		{"policy revision", plan.review.CreatedAt, "fixture-session-secret", func(p *Policy, _ *Observation) { p.Revision++ }, ErrDrift},
		{"incomplete observation", plan.review.CreatedAt, "fixture-session-secret", func(_ *Policy, o *Observation) { o.Complete = false }, ErrDrift},
		{"unqualified observation", plan.review.CreatedAt, "fixture-session-secret", func(_ *Policy, o *Observation) { o.Eligible = false }, ErrDrift},
		{"property drift", plan.review.CreatedAt, "fixture-session-secret", func(_ *Policy, o *Observation) { o.Filesystems[0].Properties.Quota = Value{"123", Local, "101", ""} }, ErrDrift},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pp, oo := p, o
			oo.Filesystems = append([]Filesystem(nil), o.Filesystems...)
			if tc.mutate != nil {
				tc.mutate(&pp, &oo)
			}
			got, err := plan.CapturePropertyIntent(tc.at, tc.session, pp, oo)
			if !errors.Is(err, tc.want) || !reflect.DeepEqual(got, PropertyIntent{}) {
				t.Fatal("invalid capture retained metadata", err)
			}
		})
	}
	if got, err := (Plan{}).CapturePropertyIntent(instant, "session", p, o); err == nil || !reflect.DeepEqual(got, PropertyIntent{}) {
		t.Fatal("zero plan captured")
	}
}

func TestPropertyIntentBaselineDrift(t *testing.T) {
	captured, _, p, o := propertyIntentFixture(t)
	for _, tc := range []struct {
		name   string
		mutate func(*Policy, *Observation)
	}{
		{"policy", func(p *Policy, _ *Observation) { p.Revision++ }},
		{"import incarnation", func(_ *Policy, o *Observation) { o.Pool.LoadGUID = "999" }},
		{"received value", func(_ *Policy, o *Observation) { o.Filesystems[0].Properties.Quota.ReceivedValue = "999" }},
		{"version membership", func(_ *Policy, o *Observation) { o.Identities = o.Identities[:2] }},
		{"mount constraint", func(_ *Policy, o *Observation) { o.States[1].Mounted.Effective = "yes" }},
		{"feature state", func(_ *Policy, o *Observation) { o.Features.Values[0].State = "active" }},
		{"vdev identity", func(_ *Policy, o *Observation) { o.Topology.Nodes[1].GUID = "999" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pp, oo := p, o
			oo.Filesystems = append([]Filesystem(nil), o.Filesystems...)
			oo.States = append([]FilesystemState(nil), o.States...)
			oo.Features.Values = append([]PoolFeature(nil), o.Features.Values...)
			oo.Topology.Nodes = append([]Vdev(nil), o.Topology.Nodes...)
			tc.mutate(&pp, &oo)
			if !errors.Is(captured.CompareBaseline(pp, oo), ErrDrift) {
				t.Fatal("changed baseline accepted")
			}
		})
	}
	// A decoded baseline is an observation comparison, never session/expiry admission.
	data, _ := EncodePropertyIntent(captured)
	decoded, err := DecodePropertyIntent(data)
	if err != nil || decoded.CompareBaseline(p, o) != nil {
		t.Fatal("baseline comparison failed", err)
	}
	if (PropertyIntent{}).CompareBaseline(p, o) == nil {
		t.Fatal("zero intent compared")
	}
}

func propertyIntentJSON(data []byte) map[string]any {
	var root map[string]any
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if d.Decode(&root) != nil {
		panic("invalid test JSON")
	}
	return root
}

func propertyIntentObjectPaths(root map[string]any) [][]string {
	var paths [][]string
	var visit func(any, []string)
	visit = func(v any, path []string) {
		switch x := v.(type) {
		case map[string]any:
			paths = append(paths, append([]string(nil), path...))
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				visit(x[k], append(append([]string(nil), path...), k))
			}
		case []any:
			if len(x) > 0 {
				visit(x[0], append(append([]string(nil), path...), "0"))
			}
		}
	}
	visit(root, nil)
	return paths
}

func TestPropertyIntentRequiredFields(t *testing.T) {
	captured, _, _, _ := propertyIntentFixture(t)
	data, _ := EncodePropertyIntent(captured)
	for _, path := range propertyIntentObjectPaths(propertyIntentJSON(data)) {
		obj := jobTestObject(propertyIntentJSON(data), path)
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			for _, mutation := range []string{"missing", "null", "wrong_type", "case", "duplicate", "unknown"} {
				t.Run(strings.Join(path, "/")+"/"+key+"/"+mutation, func(t *testing.T) {
					root := propertyIntentJSON(data)
					object := jobTestObject(root, path)
					value := object[key]
					switch mutation {
					case "missing":
						delete(object, key)
					case "null":
						object[key] = nil
					case "wrong_type":
						if _, ok := value.(bool); ok {
							object[key] = "true"
						} else {
							object[key] = true
						}
					case "case":
						delete(object, key)
						object[strings.ToUpper(key)] = value
					case "unknown":
						object["unexpected"] = value
					case "duplicate":
						object[key] = "horvium-duplicate-placeholder"
					}
					mutated, _ := json.Marshal(root)
					if mutation == "duplicate" {
						val, _ := json.Marshal(value)
						quoted, _ := json.Marshal(key)
						replacement := append(append(append(append([]byte(nil), val...), ','), quoted...), ':')
						replacement = append(replacement, val...)
						mutated = bytes.Replace(mutated, []byte(`"horvium-duplicate-placeholder"`), replacement, 1)
					}
					got, err := DecodePropertyIntent(mutated)
					if err == nil || !reflect.DeepEqual(got, PropertyIntent{}) {
						t.Fatal("invalid shape survived", err)
					}
				})
			}
		}
	}
}

func propertyIntentResign(data []byte) []byte {
	marker := bytes.LastIndex(data, []byte(`,"checksum":`))
	if marker < 0 {
		panic("test record has no checksum")
	}
	content := append(append([]byte(nil), data[:marker]...), '}')
	digest := sha256.Sum256(append([]byte("horvium:property-intent:v1\n"), content...))
	return append(append(append([]byte(nil), data[:marker]...), []byte(`,"checksum":"`+hex.EncodeToString(digest[:])+`"}`)...))
}

func TestPropertyIntentSemanticDenials(t *testing.T) {
	captured, _, _, _ := propertyIntentFixture(t)
	original, _ := EncodePropertyIntent(captured)
	for _, tc := range []struct{ name, old, next string }{
		{"future version", `"version":1`, `"version":2`},
		{"other kind", `"kind":"property-intent"`, `"kind":"job"`},
		{"invalid plan id", strings.Repeat("a", 64), strings.Repeat("A", 64)},
		{"zero policy", `"policy_revision":1`, `"policy_revision":0`},
		{"invalid intent", `"value":"1099511627776"`, `"value":"0"`},
		{"missing affected", `"affected":[{"guid":"101","name":"tank"},{"guid":"102","name":"tank/data"}]`, `"affected":[]`},
		{"wrong inherited source", `"source_guid":"101"`, `"source_guid":"999"`},
		{"volume", `"type":"snapshot"`, `"type":"volume"`},
		{"negative ordinal", `"ordinal":0`, `"ordinal":-1`},
		{"huge ordinal", `"ordinal":0`, `"ordinal":18446744073709551615`},
		{"fractional revision", `"policy_revision":1`, `"policy_revision":1.0`},
		{"exponent revision", `"policy_revision":1`, `"policy_revision":1e0`},
		{"noncanonical UTC", `2026-10-04T23:00:00.123456789Z`, `2026-10-04T23:00:00.123456789+00:00`},
		{"time reversed", `2026-10-04T23:00:01.123456789Z`, `2026-10-04T22:59:59.123456789Z`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutated := bytes.Replace(original, []byte(tc.old), []byte(tc.next), 1)
			if bytes.Equal(mutated, original) {
				t.Fatal("ineffective mutation")
			}
			mutated = propertyIntentResign(mutated)
			if got, err := DecodePropertyIntent(mutated); err == nil || !reflect.DeepEqual(got, PropertyIntent{}) {
				t.Fatal("checksum granted semantics", err)
			}
		})
	}
}

func TestPropertyIntentCanonicalOrderingAndBounds(t *testing.T) {
	captured, _, p, o := propertyIntentFixture(t)
	data, _ := EncodePropertyIntent(captured)
	// Object order/whitespace and row order are not identity; ordinals remain explicit.
	root := propertyIntentJSON(data)
	body := root["intent"].(map[string]any)
	review := body["review"].(map[string]any)
	for _, name := range []string{"context", "states", "affected"} {
		rows := review[name].([]any)
		for a, b := 0, len(rows)-1; a < b; a, b = a+1, b-1 {
			rows[a], rows[b] = rows[b], rows[a]
		}
	}
	reordered, _ := json.MarshalIndent(root, "", "  ")
	got, err := DecodePropertyIntent(reordered)
	if err != nil || got.CompareBaseline(p, o) != nil {
		t.Fatal("ordering changed binding", err)
	}
	canonical, _ := EncodePropertyIntent(got)
	if !bytes.Equal(canonical, data) {
		t.Fatal("canonical ordering changed")
	}
	padded := append(append([]byte(nil), data...), bytes.Repeat([]byte(" "), maxStateRecordBytes-len(data))...)
	if _, err := DecodePropertyIntent(padded); err != nil {
		t.Fatal("exact input ceiling refused", err)
	}
	if got, err := DecodePropertyIntent(append(padded, ' ')); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(got, PropertyIntent{}) {
		t.Fatal("oversized record survived", err)
	}
	for _, bad := range [][]byte{nil, append(append([]byte(nil), data...), []byte("{}")...), []byte("\xff"), []byte(strings.Repeat("[", 10000))} {
		if got, err := DecodePropertyIntent(bad); err == nil || !reflect.DeepEqual(got, PropertyIntent{}) {
			t.Fatal("damaged record survived", err)
		}
	}
	huge := captured
	huge.review.Context = make([]Filesystem, maxRecords+1)
	if b, err := EncodePropertyIntent(huge); !errors.Is(err, ErrLimit) || b != nil {
		t.Fatal("unbounded encoding", err)
	}
	bad := captured
	bad.review.ID = strings.Repeat("x", maxFieldBytes+1)
	if b, err := EncodePropertyIntent(bad); !errors.Is(err, ErrLimit) || b != nil {
		t.Fatal("oversized field survived", err)
	}
	if b, err := EncodePropertyIntent(PropertyIntent{}); err == nil || b != nil {
		t.Fatal("zero metadata encoded")
	}
}

func FuzzPropertyIntent(f *testing.F) {
	captured, _, _, _ := propertyIntentFixture(f)
	seed, _ := EncodePropertyIntent(captured)
	f.Add(seed)
	f.Add([]byte(`{"version":2}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		before := append([]byte(nil), data...)
		got, err := DecodePropertyIntent(data)
		if !bytes.Equal(before, data) {
			t.Fatal("input changed")
		}
		if err != nil {
			if !reflect.DeepEqual(got, PropertyIntent{}) {
				t.Fatal("failure retained metadata")
			}
			return
		}
		encoded, err := EncodePropertyIntent(got)
		if err != nil || len(encoded) > maxStateRecordBytes {
			t.Fatal("accepted metadata not bounded", err)
		}
		again, err := DecodePropertyIntent(encoded)
		if err != nil || !reflect.DeepEqual(again, got) {
			t.Fatal("accepted metadata changed", err)
		}
	})
}

func TestPropertyIntentScopePrecisionAndSessionDigest(t *testing.T) {
	p, o, change := fixture()
	p.Revision = ^uint64(0)
	change.FilesystemGUID = "102"
	change.Value = "18446744073709551615"
	plan, err := Build(instant, "same-secret", p, o, change)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := plan.CapturePropertyIntent(instant, "same-secret", p, o)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := EncodePropertyIntent(snapshot)
	decoded, err := DecodePropertyIntent(data)
	if err != nil || decoded.Review().PolicyRevision != ^uint64(0) || decoded.Review().Intent.Value != change.Value || decoded.CompareBaseline(p, o) != nil {
		t.Fatal("uint64 precision lost", err)
	}
	if len(decoded.Review().Versions) != 0 {
		t.Fatal("empty versions invented")
	}
	oo := o
	other := o.Filesystems[1]
	other.GUID = "103"
	other.Name = "tank/other"
	oo.Filesystems = append(append([]Filesystem(nil), o.Filesystems...), other)
	oo.Identities = append(append([]DatasetIdentity(nil), o.Identities...), DatasetIdentity{GUID: other.GUID, Name: other.Name, Type: FilesystemType})
	oo.States = append(append([]FilesystemState(nil), o.States...), stateFixture(other))
	if decoded.CompareBaseline(p, oo) != nil {
		t.Fatal("unrelated sibling invalidated target context")
	}
	second, err := Build(instant, "same-secret", p, o, change)
	if err != nil {
		t.Fatal(err)
	}
	another, err := second.CapturePropertyIntent(instant, "same-secret", p, o)
	if err != nil || another.sessionDigest == snapshot.sessionDigest {
		t.Fatal("session digest not scoped to plan", err)
	}
	expected := sha256.Sum256([]byte("horvium:property-intent-session:v1\n" + plan.Review().ID + "\nsame-secret"))
	if expected != snapshot.sessionDigest {
		t.Fatal("wrong session digest domain")
	}
	later, err := plan.CapturePropertyIntent(instant.Add(time.Second), "same-secret", p, o)
	if err != nil || later.sessionDigest != snapshot.sessionDigest {
		t.Fatal("capture time altered session binding", err)
	}
	// Protected descendant context remains present despite requiring no write grant.
	p, o, change = fixture()
	change.Property = Compression
	change.Value = "off"
	p.Scopes = p.Scopes[:1]
	o.Filesystems[1].Properties.Compression = Value{"off", Local, "102", ""}
	protected, err := Build(instant, "session", p, o, change)
	if err != nil {
		t.Fatal(err)
	}
	protectedSnapshot, err := protected.CapturePropertyIntent(instant, "session", p, o)
	if err != nil {
		t.Fatal(err)
	}
	data, _ = EncodePropertyIntent(protectedSnapshot)
	decoded, err = DecodePropertyIntent(data)
	if err != nil || len(decoded.Review().Context) != 2 || len(decoded.Review().Affected) != 1 {
		t.Fatal("protected branch context lost", err)
	}
}
