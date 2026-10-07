// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func enrolledPolicy() Policy {
	p, _, _ := fixture()
	p.Members = []PoolMember{{VdevGUID: "901", Serial: "SERIAL-A", SerialSource: "geom.config.ident", MediaSize: 67108864, SectorSize: 512}, {VdevGUID: "902", Serial: "SERIAL-B", SerialSource: "provider.config.ident", MediaSize: 67108864, SectorSize: 4096, StripeSize: 4096}}
	return p
}
func TestOwnershipMembersVersionedCanonicalRoundTrip(t *testing.T) {
	p := enrolledPolicy()
	before, _ := json.Marshal(p)
	b, err := EncodeOwnershipRecord(p)
	if err != nil || !bytes.Contains(b, []byte(`"version":2`)) {
		t.Fatal("enrolled record version", err)
	}
	var wire map[string]json.RawMessage
	if json.Unmarshal(b, &wire) != nil {
		t.Fatal("invalid wire")
	}
	// Independently compute the damage checksum over the exact content/domain.
	content := []byte(`{"version":2,"kind":"ownership","policy":` + string(wire["policy"]) + `}`)
	h := sha256.Sum256(append([]byte("horvium:ownership:v2\n"), content...))
	var checksum string
	_ = json.Unmarshal(wire["checksum"], &checksum)
	if checksum != hex.EncodeToString(h[:]) {
		t.Fatal("checksum domain")
	}
	got, err := DecodeOwnershipRecord(b)
	if err != nil || !reflect.DeepEqual(got.Members, p.Members) {
		t.Fatal("member facts lost", err)
	}
	got.Members[0].Serial = "changed"
	after, _ := json.Marshal(p)
	if !bytes.Equal(before, after) {
		t.Fatal("input aliased")
	}
	p.Members[0], p.Members[1] = p.Members[1], p.Members[0]
	again, err := EncodeOwnershipRecord(p)
	if err != nil || !bytes.Equal(again, b) {
		t.Fatal("member order changes canonical bytes", err)
	}
	p.Members = nil
	legacy, err := EncodeOwnershipRecord(p)
	if err != nil || !bytes.Equal(legacy, ownershipGolden()) {
		t.Fatal("v1 bytes changed", err)
	}
	decoded, err := DecodeOwnershipRecord(legacy)
	if err != nil || len(decoded.Members) != 0 {
		t.Fatal("invented legacy enrollment", err)
	}
}
func TestOwnershipMembersRefuseInvalidSchemaAndFacts(t *testing.T) {
	for _, mode := range []string{"duplicate-guid", "duplicate-serial", "empty-serial", "unknown-source", "control", "spaces", "geometry", "sector", "overflow", "count"} {
		t.Run(mode, func(t *testing.T) {
			p := enrolledPolicy()
			switch mode {
			case "duplicate-guid":
				p.Members[1].VdevGUID = p.Members[0].VdevGUID
			case "duplicate-serial":
				p.Members[1].Serial = p.Members[0].Serial
			case "empty-serial":
				p.Members[0].Serial = ""
			case "unknown-source":
				p.Members[0].SerialSource = "client"
			case "control":
				p.Members[0].Serial = "serial\x00value"
			case "spaces":
				p.Members[0].Serial = " SERIAL-A"
			case "geometry":
				p.Members[0].MediaSize++
			case "sector":
				p.Members[0].SectorSize = 0
			case "overflow":
				p.Members[0].MediaSize = ^uint64(0)
			case "count":
				p.Members = make([]PoolMember, maxVdevs+1)
			}
			if b, err := EncodeOwnershipRecord(p); err == nil || b != nil {
				t.Fatal("invalid enrollment encoded")
			}
		})
	}
	b, err := EncodeOwnershipRecord(enrolledPolicy())
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		strings.Replace(string(b), `"version":2`, `"version":1`, 1), strings.Replace(string(b), `"version":2`, `"version":3`, 1),
		strings.Replace(string(b), `"members":[`, `"members":null,"ignored":[`, 1),
		strings.Replace(string(b), `"serial":"SERIAL-A"`, `"serial":"SERIAL-A","serial":"SERIAL-A"`, 1),
		strings.Replace(string(b), `"serial":"SERIAL-A"`, `"Serial":"SERIAL-A"`, 1),
		strings.Replace(string(b), `,"stripe_offset":0`, "", 1),
	} {
		if p, err := DecodeOwnershipRecord([]byte(bad)); err == nil || !reflect.DeepEqual(p, Policy{}) {
			t.Fatal("invalid v2 shape exposed policy")
		}
	}
}
func TestOwnershipMemberChangeInvalidatesExistingPlan(t *testing.T) {
	p := enrolledPolicy()
	_, o, i := fixture()
	at := time.Now()
	plan, err := Build(at, "session", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	p.Members[0].Serial = "CHANGED"
	if err = plan.Validate(at, "session", p, o); !errors.Is(err, ErrDrift) {
		t.Fatal("same-revision member drift ignored", err)
	}
}

func TestOwnershipMembersChecksummedSemanticRefusal(t *testing.T) {
	for _, mode := range []string{"empty", "serial", "source", "guid", "geometry", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			wire := ownershipWireV2(enrolledPolicy())
			switch mode {
			case "empty":
				wire.Members = []ownershipMember{}
			case "serial":
				wire.Members[0].Serial = ""
			case "source":
				wire.Members[0].SerialSource = "client"
			case "guid":
				wire.Members[0].VdevGUID = "0901"
			case "geometry":
				wire.Members[0].MediaSize++
			case "duplicate":
				wire.Members[1].Serial = wire.Members[0].Serial
			}
			data, err := json.Marshal(ownershipRecordV2{2, "ownership", wire, ownershipChecksumV2(wire)})
			if err != nil {
				t.Fatal(err)
			}
			if got, err := DecodeOwnershipRecord(data); err == nil || !reflect.DeepEqual(got, Policy{}) {
				t.Fatal("checksummed invalid facts accepted", err)
			}
		})
	}
}
func TestOwnershipMembersPersistedIntentDrift(t *testing.T) {
	p := enrolledPolicy()
	_, o, change := fixture()
	plan, err := Build(instant, "session", p, o, change)
	if err != nil {
		t.Fatal(err)
	}
	intent, err := plan.CapturePropertyIntent(instant.Add(time.Second), "session", p, o)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodePropertyIntent(intent)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := DecodePropertyIntent(data)
	if err != nil || restored.CompareBaseline(p, o) != nil {
		t.Fatal("enrolled intent roundtrip", err)
	}
	p.Members[0].Serial = "CHANGED"
	if err := restored.CompareBaseline(p, o); !errors.Is(err, ErrDrift) {
		t.Fatal("durable intent ignored member drift", err)
	}
}
