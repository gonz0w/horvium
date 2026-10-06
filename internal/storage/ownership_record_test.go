// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

const ownershipGoldenContent = `{"version":1,"kind":"ownership","policy":{"revision":1,"pool_guid":"100","pool_name":"tank","scopes":[{"guid":"101","name":"tank","parent_guid":"","properties":["compression","quota","refquota"]},{"guid":"102","name":"tank/data","parent_guid":"101","properties":["compression","quota"]}]}}`

func ownershipGolden() []byte {
	return uncheckedOwnership(ownershipGoldenContent)
}

func uncheckedOwnership(content string) []byte {
	h := sha256.Sum256([]byte("horvium:ownership:v1\n" + content))
	return []byte(strings.TrimSuffix(content, "}") + `,"checksum":"` + hex.EncodeToString(h[:]) + `"}`)
}

func TestOwnershipRecordRoundTripAndCanonicalBytes(t *testing.T) {
	p, _, _ := fixture()
	before, _ := json.Marshal(p)
	encoded, err := EncodeOwnershipRecord(p)
	if err != nil || !bytes.Equal(encoded, ownershipGolden()) {
		t.Fatal("canonical ownership bytes changed", err)
	}
	after, _ := json.Marshal(p)
	if !bytes.Equal(before, after) {
		t.Fatal("encoding mutated caller policy")
	}
	got, err := DecodeOwnershipRecord(encoded)
	if err != nil || got.Revision != p.Revision || got.PoolGUID != p.PoolGUID || got.Scopes[1].ParentGUID != "101" {
		t.Fatal("identity or scope lost", err)
	}
	reencoded, err := EncodeOwnershipRecord(got)
	if err != nil || !bytes.Equal(reencoded, encoded) {
		t.Fatal("record did not round trip", err)
	}
	// Decoded policy and later calls must not retain or alias input/caller state.
	got.Scopes[0].Properties[0] = Quota
	encoded[0] = 'x'
	again, err := DecodeOwnershipRecord(ownershipGolden())
	if err != nil || again.Scopes[0].Properties[0] != Compression || p.Scopes[0].Properties[0] != Quota {
		t.Fatal("record aliasing", err)
	}
	// Property/scope order and object-field/whitespace presentation are not grants.
	p.Scopes[0], p.Scopes[1] = p.Scopes[1], p.Scopes[0]
	reordered, err := EncodeOwnershipRecord(p)
	if err != nil || !bytes.Equal(reordered, ownershipGolden()) {
		t.Fatal("scope order changed checksum", err)
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, ownershipGolden(), "", "  ") != nil {
		t.Fatal("fixture indent failed")
	}
	if _, err := DecodeOwnershipRecord(pretty.Bytes()); err != nil {
		t.Fatal("valid presentation rejected", err)
	}
	escaped := strings.Replace(string(ownershipGolden()), `"version"`, `"versi\u006fn"`, 1)
	if _, err := DecodeOwnershipRecord([]byte(escaped)); err != nil {
		t.Fatal("valid escaped field rejected", err)
	}
	max := append(ownershipGolden(), bytes.Repeat([]byte(" "), (64<<10)-len(ownershipGolden()))...)
	if _, err := DecodeOwnershipRecord(max); err != nil {
		t.Fatal("exact byte ceiling refused", err)
	}
}

func TestOwnershipChecksumDoesNotGrantInvalidScope(t *testing.T) {
	// Recompute the independent damage checksum on deliberately invalid content.
	// Denial must come from schema/semantic checks, even with a matching checksum.
	for name, content := range map[string]string{
		"zero revision":         strings.Replace(ownershipGoldenContent, `"revision":1`, `"revision":0`, 1),
		"bad pool identity":     strings.Replace(ownershipGoldenContent, `"pool_guid":"100"`, `"pool_guid":"0100"`, 1),
		"foreign scope":         strings.Replace(ownershipGoldenContent, `"tank/data"`, `"other/data"`, 1),
		"self parent":           strings.Replace(ownershipGoldenContent, `"parent_guid":"101"`, `"parent_guid":"102"`, 1),
		"wrong selected parent": strings.Replace(ownershipGoldenContent, `"tank/data"`, `"tank/foreign/data"`, 1),
		"duplicate identity":    strings.Replace(ownershipGoldenContent, `"guid":"102"`, `"guid":"101"`, 1),
		"unknown property":      strings.Replace(ownershipGoldenContent, `"compression"`, `"mountpoint"`, 1),
		"duplicate property":    strings.Replace(ownershipGoldenContent, `["compression","quota","refquota"]`, `["compression","quota","quota"]`, 1),
		"null properties":       strings.Replace(ownershipGoldenContent, `["compression","quota","refquota"]`, `null`, 1),
		"newer version":         strings.Replace(ownershipGoldenContent, `"version":1`, `"version":2`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if p, err := DecodeOwnershipRecord(uncheckedOwnership(content)); err == nil || !reflect.DeepEqual(p, Policy{}) {
				t.Fatal("damage checksum granted invalid policy", err)
			}
		})
	}
	deep := []byte(`{"version":` + strings.Repeat("[", 10000) + strings.Repeat("]", 10000) + `}`)
	if p, err := DecodeOwnershipRecord(deep); err == nil || !reflect.DeepEqual(p, Policy{}) {
		t.Fatal("wrong shape recursed or granted policy", err)
	}
}

func TestOwnershipRecordStrictDenials(t *testing.T) {
	good := string(ownershipGolden())
	for name, data := range map[string]string{
		"empty": "", "truncated": good[:len(good)-1], "trailing value": good + "{}",
		"top array": "[" + good + "]", "top null": "null",
		"duplicate version":   strings.Replace(good, `"version":1`, `"version":1,"version":1`, 1),
		"escaped duplicate":   strings.Replace(good, `"version":1`, `"version":1,"versi\u006fn":1`, 1),
		"case version":        strings.Replace(good, `"version"`, `"Version"`, 1),
		"unknown envelope":    strings.Replace(good, `"version":1`, `"extra":false,"version":1`, 1),
		"unknown policy":      strings.Replace(good, `"revision":1`, `"extra":{},"revision":1`, 1),
		"unknown scope":       strings.Replace(good, `"guid":"101"`, `"extra":[],"guid":"101"`, 1),
		"duplicate revision":  strings.Replace(good, `"revision":1`, `"revision":1,"revision":1`, 1),
		"duplicate scope key": strings.Replace(good, `"guid":"101"`, `"guid":"101","guid":"101"`, 1),
		"missing version":     strings.Replace(good, `"version":1,`, "", 1),
		"missing parent":      strings.Replace(good, `"parent_guid":"",`, "", 1),
		"missing properties":  strings.Replace(good, `,"properties":["compression","quota","refquota"]`, "", 1),
		"newer version":       strings.Replace(good, `"version":1`, `"version":2`, 1),
		"zero version":        strings.Replace(good, `"version":1`, `"version":0`, 1),
		"wrong kind":          strings.Replace(good, `"kind":"ownership"`, `"kind":"job"`, 1),
		"version string":      strings.Replace(good, `"version":1`, `"version":"1"`, 1),
		"version exponent":    strings.Replace(good, `"version":1`, `"version":1e0`, 1),
		"revision fraction":   strings.Replace(good, `"revision":1`, `"revision":1.0`, 1),
		"revision overflow":   strings.Replace(good, `"revision":1`, `"revision":18446744073709551616`, 1),
		"revision negative":   strings.Replace(good, `"revision":1`, `"revision":-1`, 1),
		"null policy":         strings.Replace(good, `"policy":{`, `"policy":null,"other":{`, 1),
		"null scopes":         strings.Replace(good, `"scopes":[`, `"scopes":null,"other":[`, 1),
		"null property":       strings.Replace(good, `"compression"`, `null`, 1),
		"invalid utf8":        strings.Replace(good, `"tank"`, "\"ta\xffnk\"", 1),
		"surrogate":           strings.Replace(good, `"tank"`, `"ta\ud800nk"`, 1),
		"control name":        strings.Replace(good, `"tank"`, `"ta\u0000nk"`, 1),
		"checksum damage":     strings.Replace(good, `"revision":1`, `"revision":2`, 1),
		"empty checksum":      good[:strings.LastIndex(good, `"checksum"`)] + `"checksum":""}`,
	} {
		t.Run(name, func(t *testing.T) {
			before := []byte(data)
			got, err := DecodeOwnershipRecord(before)
			if err == nil || !reflect.DeepEqual(got, Policy{}) || string(before) != data {
				t.Fatal("invalid record returned authority or modified input", err)
			}
		})
	}
	if got, err := DecodeOwnershipRecord(bytes.Repeat([]byte(" "), (64<<10)+1)); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(got, Policy{}) {
		t.Fatal("oversized record accepted", err)
	}
}

func TestOwnershipRecordRejectsInvalidPolicies(t *testing.T) {
	for name, change := range map[string]func(*Policy){
		"zero revision":         func(p *Policy) { p.Revision = 0 },
		"pool guid":             func(p *Policy) { p.PoolGUID = "0100" },
		"pool path":             func(p *Policy) { p.PoolName = "tank/child" },
		"duplicate guid":        func(p *Policy) { p.Scopes[1].GUID = "101" },
		"duplicate name":        func(p *Policy) { p.Scopes[1].Name = "tank" },
		"foreign name":          func(p *Policy) { p.Scopes[1].Name = "other/data" },
		"root parent":           func(p *Policy) { p.Scopes[0].ParentGUID = "999" },
		"missing parent":        func(p *Policy) { p.Scopes[1].ParentGUID = "" },
		"self parent":           func(p *Policy) { p.Scopes[1].ParentGUID = "102" },
		"wrong selected parent": func(p *Policy) { p.Scopes[1].Name = "tank/foreign/data" },
		"no properties":         func(p *Policy) { p.Scopes[0].Properties = nil },
		"unknown property":      func(p *Policy) { p.Scopes[0].Properties[0] = "mountpoint" },
		"duplicate property":    func(p *Policy) { p.Scopes[0].Properties[0] = RefQuota },
		"huge field":            func(p *Policy) { p.Scopes[0].Name = strings.Repeat("x", 65537) },
		"huge count":            func(p *Policy) { p.Scopes = make([]Scope, 100001) },
	} {
		t.Run(name, func(t *testing.T) {
			p, _, _ := fixture()
			change(&p)
			if got, err := EncodeOwnershipRecord(p); err == nil || got != nil {
				t.Fatal("invalid policy encoded", err)
			}
		})
	}
	// Empty enrollment and a selected child without selecting its parent grant
	// no implicit parent/descendant scope. Native validation remains necessary.
	for _, scopes := range [][]Scope{{}, {{GUID: "102", Name: "tank/data", ParentGUID: "101", Properties: []Property{Quota}}}} {
		p := Policy{Revision: 1, PoolGUID: "100", PoolName: "tank", Scopes: scopes}
		b, err := EncodeOwnershipRecord(p)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeOwnershipRecord(b)
		if err != nil || len(got.Scopes) != len(scopes) {
			t.Fatal("implicit scope created", err)
		}
	}
}

func TestOwnershipRecordCapacityAndRevision(t *testing.T) {
	p, _, _ := fixture()
	p.Revision = math.MaxUint64
	b, err := EncodeOwnershipRecord(p)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeOwnershipRecord(b)
	if err != nil || decoded.Revision != math.MaxUint64 {
		t.Fatal("revision precision lost", err)
	}
	for n := 0; n < 500; n++ {
		p.Scopes = append(p.Scopes, Scope{GUID: fmt.Sprintf("%d", 1000+n), Name: fmt.Sprintf("tank/child%d", n), ParentGUID: "101", Properties: []Property{Quota}})
	}
	before, _ := json.Marshal(p)
	if b, err := EncodeOwnershipRecord(p); b != nil || !errors.Is(err, ErrLimit) {
		t.Fatal("metadata budget did not refuse whole policy", err)
	}
	after, _ := json.Marshal(p)
	if !bytes.Equal(before, after) {
		t.Fatal("capacity refusal truncated or changed enrollment")
	}
}

func FuzzOwnershipRecord(f *testing.F) {
	f.Add(ownershipGolden())
	f.Add([]byte(`{"version":2}`))
	f.Add([]byte("null"))
	f.Fuzz(func(t *testing.T, data []byte) {
		p, err := DecodeOwnershipRecord(data)
		if err != nil {
			if !reflect.DeepEqual(p, Policy{}) {
				t.Fatal("denial returned policy")
			}
			return
		}
		b, err := EncodeOwnershipRecord(p)
		if err != nil || len(b) > 64<<10 {
			t.Fatal("accepted record cannot be bounded/encoded", err)
		}
		again, err := DecodeOwnershipRecord(b)
		if err != nil || !reflect.DeepEqual(again, p) {
			t.Fatal("accepted record changed on round trip", err)
		}
	})
}
