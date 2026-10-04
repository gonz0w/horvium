// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

func frame(s string) []byte {
	b := make([]byte, 4+len(s))
	binary.BigEndian.PutUint32(b, uint32(len(s)))
	copy(b[4:], s)
	return b
}

func observation() inventory.Snapshot {
	t := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	s := inventory.Snapshot{StartedAt: t, FinishedAt: t.Add(time.Second)}
	s.Host = inventory.Section[inventory.Host]{Status: inventory.Partial, StartedAt: t, FinishedAt: t, Data: inventory.Host{Architecture: "amd64"}, Problems: []inventory.Problem{{Source: "physical-memory", Code: "permission"}}}
	s.GEOM = inventory.Section[inventory.GEOM]{Status: inventory.OK, StartedAt: t, FinishedAt: t, Data: inventory.GEOM{
		Classes:   []inventory.Class{{ID: "c", Name: "MD"}},
		Geoms:     []inventory.Geom{{ID: "g", ClassID: "c", Name: "md0", Consumers: []inventory.Consumer{{ID: "consumer", ProviderID: "p"}}}},
		Providers: []inventory.Provider{{ID: "p", GeomID: "g", Name: "md0", Class: "MD", Kind: inventory.Memory, MediaSize: 512, SectorSize: 512, Identity: inventory.IdentityNotApplicable}},
	}}
	s.Pools = inventory.Section[[]inventory.Pool]{Status: inventory.OK, StartedAt: t, FinishedAt: t, Data: []inventory.Pool{{Name: "fixture", GUID: "18446744073709551615", Size: 1024, Allocated: 128, Free: 896, Health: "ONLINE"}}}
	s.Datasets = inventory.Section[[]inventory.Dataset]{Status: inventory.Unavailable, StartedAt: t, FinishedAt: t, Problems: []inventory.Problem{{Source: "datasets", Code: "timeout"}}}
	return s
}

func TestWireRoundTrip(t *testing.T) {
	limits := DefaultLimits()
	var b bytes.Buffer
	if err := writeRequest(&b, "request-1", limits); err != nil {
		t.Fatal(err)
	}
	if id, err := readRequest(&b, limits); err != nil || id != "request-1" {
		t.Fatalf("id=%q error=%v", id, err)
	}
	want := observation()
	if err := writeSnapshot(&b, "request-1", want, limits); err != nil {
		t.Fatal(err)
	}
	got, err := readSnapshot(&b, "request-1", limits)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("typed values, relationships, section errors or timestamps changed")
	}
	if got.Complete() {
		t.Fatal("partial observation became complete")
	}
}

func TestRequestsRejectAuthorityAndAmbiguity(t *testing.T) {
	for name, payload := range map[string]string{
		"operation":    `{"version":1,"request_id":"r","operation":"destroy"}`,
		"version":      `{"version":2,"request_id":"r","operation":"inventory"}`,
		"forged uid":   `{"version":1,"request_id":"r","operation":"inventory","uid":0}`,
		"command":      `{"version":1,"request_id":"r","operation":"inventory","command":"zpool"}`,
		"duplicate":    `{"version":1,"request_id":"r","operation":"destroy","operation":"inventory"}`,
		"case":         `{"Version":1,"request_id":"r","operation":"inventory"}`,
		"missing":      `{"version":1,"operation":"inventory"}`,
		"null":         `{"version":1,"request_id":null,"operation":"inventory"}`,
		"invalid utf8": "{\"version\":1,\"request_id\":\"\xff\",\"operation\":\"inventory\"}",
		"trailing":     `{"version":1,"request_id":"r","operation":"inventory"}{}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := readRequest(bytes.NewReader(frame(payload)), DefaultLimits()); err == nil {
				t.Fatal("accepted invalid request")
			}
		})
	}
}

func TestFramingBoundsAndInterruptedCollections(t *testing.T) {
	var b bytes.Buffer
	if err := writeSnapshot(&b, "r", observation(), DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	data := b.Bytes()
	for _, cut := range []int{0, 1, 3, 4, len(data) - 1, len(data) - 4} {
		if s, err := readSnapshot(bytes.NewReader(data[:cut]), "r", DefaultLimits()); err == nil || !reflect.DeepEqual(s, inventory.Snapshot{}) {
			t.Fatalf("cut %d published interrupted state", cut)
		}
	}
	if _, err := readSnapshot(bytes.NewReader(data), "other", DefaultLimits()); err == nil {
		t.Fatal("wrong request accepted")
	}
	if _, err := readSnapshot(bytes.NewReader(append(append([]byte{}, data...), frame(`{}`)...)), "r", DefaultLimits()); err == nil {
		t.Fatal("trailing record accepted")
	}
	for _, limits := range []Limits{{MaxBytes: 8, MaxRecords: 100, MaxFieldBytes: 4096, MaxDepth: 32}, {MaxBytes: 1 << 20, MaxRecords: 2, MaxFieldBytes: 4096, MaxDepth: 32}} {
		if _, err := readSnapshot(bytes.NewReader(data), "r", limits); !errors.Is(err, ErrLimit) {
			t.Fatalf("limit: %v", err)
		}
	}
	oversize := make([]byte, 4)
	binary.BigEndian.PutUint32(oversize, maxFrame+1)
	if _, err := readRequest(bytes.NewReader(oversize), DefaultLimits()); !errors.Is(err, ErrLimit) {
		t.Fatalf("oversize preallocation: %v", err)
	}
}

func TestWriterDoesNotTruncateOversizedRecords(t *testing.T) {
	s := observation()
	s.Pools.Data[0].Name = strings.Repeat("x", maxFrame)
	if err := writeSnapshot(&bytes.Buffer{}, "r", s, DefaultLimits()); !errors.Is(err, ErrLimit) {
		t.Fatalf("oversized record: %v", err)
	}
}

func FuzzRequest(f *testing.F) {
	f.Add(frame(`{"version":1,"request_id":"r","operation":"inventory"}`))
	f.Add([]byte{255, 255, 255, 255})
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > maxFrame+4 {
			return
		}
		_, _ = readRequest(bytes.NewReader(b), DefaultLimits())
	})
}

func TestMalformedResponseCannotPublish(t *testing.T) {
	var b bytes.Buffer
	if err := writeSnapshot(&b, "r", observation(), DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	for name, replace := range map[string]func([]byte) []byte{
		"unknown type": func(b []byte) []byte { return bytes.Replace(b, []byte(`"type":"pool"`), []byte(`"type":"evil"`), 1) },
		"unknown field": func(b []byte) []byte {
			return rewriteRecord(t, b, "pool", func(s string) string { return strings.Replace(s, `"guid":`, `"unknown":`, 1) })
		},
		"false completeness": func(b []byte) []byte {
			return rewriteRecord(t, b, "end", func(s string) string { return strings.Replace(s, `"complete":false`, `"complete":true`, 1) })
		},
		"wrong count": func(b []byte) []byte {
			return rewriteRecord(t, b, "end", func(s string) string { return strings.Replace(s, `"records":7`, `"records":1`, 1) })
		},
		"dangling graph": func(b []byte) []byte {
			return rewriteRecord(t, b, "provider", func(s string) string { return strings.Replace(s, `"geom_local_id":"g"`, `"geom_local_id":"?"`, 1) })
		},
		"failed section data": func(b []byte) []byte {
			return rewriteRecord(t, b, "begin", func(s string) string { return strings.Replace(s, `"status":"ok"`, `"status":"unavailable"`, 1) })
		},
		"missing end": func(b []byte) []byte { return b[:lastRecordOffset(b)] },
	} {
		t.Run(name, func(t *testing.T) {
			data := replace(append([]byte{}, b.Bytes()...))
			got, err := readSnapshot(bytes.NewReader(data), "r", DefaultLimits())
			if err == nil || !reflect.DeepEqual(got, inventory.Snapshot{}) {
				t.Fatal("malformed collection published")
			}
		})
	}
}
func lastRecordOffset(b []byte) int {
	offset, last := 0, 0
	for offset < len(b) {
		last = offset
		offset += 4 + int(binary.BigEndian.Uint32(b[offset:]))
	}
	return last
}
func rewriteRecord(t *testing.T, b []byte, kind string, change func(string) string) []byte {
	t.Helper()
	var result []byte
	for offset := 0; offset < len(b); {
		n := int(binary.BigEndian.Uint32(b[offset:]))
		data := string(b[offset+4 : offset+4+n])
		if strings.Contains(data, `"type":"`+kind+`"`) {
			changed := change(data)
			if changed == data {
				t.Fatal("test mutation did not change record")
			}
			data = changed
		}
		result = append(result, frame(data)...)
		offset += 4 + n
	}
	return result
}
func TestNestedJSONFieldAndDepthLimits(t *testing.T) {
	for _, payload := range []string{`{"a":` + strings.Repeat(`[`, 33) + `0` + strings.Repeat(`]`, 33) + `}`, `{"a":"` + strings.Repeat("x", 4097) + `"}`} {
		if err := checkJSON([]byte(payload), nil, DefaultLimits()); !errors.Is(err, ErrLimit) {
			t.Fatalf("nested bounds: %v", err)
		}
	}
}
func FuzzSnapshot(f *testing.F) {
	var b bytes.Buffer
	_ = writeSnapshot(&b, "r", observation(), DefaultLimits())
	f.Add(b.Bytes())
	f.Add([]byte{0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 1<<20 {
			return
		}
		s, err := readSnapshot(bytes.NewReader(b), "r", Limits{1 << 20, 1000, 4096, 32})
		if err != nil && !reflect.DeepEqual(s, inventory.Snapshot{}) {
			t.Fatal("failed response published state")
		}
	})
}
