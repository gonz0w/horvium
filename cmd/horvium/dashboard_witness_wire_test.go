// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"encoding/binary"
	"github.com/gonz0w/horvium/internal/inventory"
	"io"
	"strings"
	"testing"
)

const witnessTestID = "0123456789abcdef0123456789abcdef-1"

// Literal producer grammar: neither the helper writer nor the witness decoder
// constructs the expected byte values. Include integers outside float64's range.
func witnessGolden() []string {
	return []string{
		`{"version":1,"request_id":"` + witnessTestID + `","type":"begin","data":{"started_at":"2026-10-13T00:00:00Z","finished_at":"2026-10-13T00:00:01Z","host":{"status":"ok","started_at":"2026-10-13T00:00:00Z","finished_at":"2026-10-13T00:00:01Z","data":{"installed_kernel":"fixture","running_kernel":"fixture","userland":"fixture","architecture":"amd64","physical_memory_bytes":1,"zfs_userland":"fixture","zfs_kernel":"fixture"}},"geom":{"status":"ok","started_at":"2026-10-13T00:00:00Z","finished_at":"2026-10-13T00:00:01Z"},"pools":{"status":"ok","started_at":"2026-10-13T00:00:00Z","finished_at":"2026-10-13T00:00:01Z"},"datasets":{"status":"ok","started_at":"2026-10-13T00:00:00Z","finished_at":"2026-10-13T00:00:01Z"}}}`,
		`{"version":1,"request_id":"` + witnessTestID + `","type":"pool","data":{"name":"fixture","guid":"18446744073709551615","size_bytes":18446744073709551615,"allocated_bytes":9007199254740993,"free_bytes":18437736874454810622,"health":"ONLINE"}}`,
		`{"version":1,"request_id":"` + witnessTestID + `","type":"dataset","data":{"name":"fixture/data","guid":"42","used_bytes":9007199254740993,"available_bytes":18446744073709551615,"referenced_bytes":1}}`,
		`{"version":1,"request_id":"` + witnessTestID + `","type":"end","data":{"records":4,"complete":true}}`,
	}
}

func witnessFrames(records []string) []byte {
	var b bytes.Buffer
	for _, record := range records {
		_ = binary.Write(&b, binary.BigEndian, uint32(len(record)))
		b.WriteString(record)
	}
	return b.Bytes()
}

func TestDashboardWitnessOriginalGrammarAndExactIntegers(t *testing.T) {
	raw := witnessFrames(witnessGolden())
	s, err := dashboardDecodeWitness(bytes.NewReader(raw), witnessTestID)
	if err != nil || len(s.Pools.Data) != 1 || len(s.Datasets.Data) != 1 || !s.Complete() {
		t.Fatalf("original complete grammar: %v", err)
	}
	if s.Pools.Data[0].Size != ^uint64(0) || s.Datasets.Data[0].Used != 9007199254740993 || s.Datasets.Data[0].Available != ^uint64(0) {
		t.Fatal("original integer precision lost")
	}
	if _, err := dashboardDecodeWitness(bytes.NewReader(raw), strings.Replace(witnessTestID, "-1", "-2", 1)); err == nil {
		t.Fatal("equal values with wrong request ID accepted")
	}
}

func TestDashboardWitnessMalformedOriginalsFail(t *testing.T) {
	cases := map[string]func([]string) []string{
		"duplicate": func(r []string) []string {
			r[1] = strings.Replace(r[1], `"version":1`, `"version":1,"version":1`, 1)
			return r
		},
		"nested duplicate": func(r []string) []string {
			r[1] = strings.Replace(r[1], `"name":"fixture"`, `"name":"fixture","name":"fixture"`, 1)
			return r
		},
		"unknown": func(r []string) []string {
			r[1] = strings.Replace(r[1], `"health":"ONLINE"`, `"health":"ONLINE","secret":1`, 1)
			return r
		},
		"combined health": func(r []string) []string {
			r[1] = strings.Replace(r[1], `"health":"ONLINE"`, `"health":"ONLINE|DEGRADED"`, 1)
			return r
		},
		"missing": func(r []string) []string { r[1] = strings.Replace(r[1], `,"health":"ONLINE"`, "", 1); return r },
		"null required": func(r []string) []string {
			r[1] = strings.Replace(r[1], `"health":"ONLINE"`, `"health":null`, 1)
			return r
		},
		"fraction": func(r []string) []string {
			r[2] = strings.Replace(r[2], `"referenced_bytes":1`, `"referenced_bytes":1.0`, 1)
			return r
		},
		"exponent": func(r []string) []string {
			r[2] = strings.Replace(r[2], `"referenced_bytes":1`, `"referenced_bytes":1e0`, 1)
			return r
		},
		"negative": func(r []string) []string {
			r[2] = strings.Replace(r[2], `"referenced_bytes":1`, `"referenced_bytes":-1`, 1)
			return r
		},
		"zero guid":         func(r []string) []string { r[2] = strings.Replace(r[2], `"guid":"42"`, `"guid":"0"`, 1); return r },
		"noncanonical guid": func(r []string) []string { r[2] = strings.Replace(r[2], `"guid":"42"`, `"guid":"042"`, 1); return r },
		"wrong status": func(r []string) []string {
			r[0] = strings.Replace(r[0], `"status":"ok"`, `"status":"done"`, 1)
			return r
		},
		"clock reversal": func(r []string) []string {
			r[0] = strings.ReplaceAll(r[0], `2026-10-13T00:00:01Z`, `2026-10-12T00:00:00Z`)
			return r
		},
		"end count": func(r []string) []string { r[3] = strings.Replace(r[3], `"records":4`, `"records":3`, 1); return r },
		"end completeness": func(r []string) []string {
			r[3] = strings.Replace(r[3], `"complete":true`, `"complete":false`, 1)
			return r
		},
		"begin order":   func(r []string) []string { r[0], r[1] = r[1], r[0]; return r },
		"stage order":   func(r []string) []string { r[1], r[2] = r[2], r[1]; return r },
		"record replay": func(r []string) []string { r = append(r[:2], append([]string{r[1]}, r[2:]...)...); return r },
		"trailing":      func(r []string) []string { return append(r, r[3]) },
		"truncated end": func(r []string) []string { return r[:3] },
		"control byte": func(r []string) []string {
			r[1] = strings.Replace(r[1], `"name":"fixture"`, `"name":"fixture\u0001"`, 1)
			return r
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := dashboardDecodeWitness(bytes.NewReader(witnessFrames(mutate(witnessGolden()))), witnessTestID); err == nil {
				t.Fatal("malformed producer accepted")
			}
		})
	}
	for _, raw := range [][]byte{nil, {0, 0, 0, 0}, {0, 1, 0, 1}, append(witnessFrames(witnessGolden()), 0), witnessFrames(witnessGolden())[:50]} {
		if _, err := dashboardDecodeWitness(bytes.NewReader(raw), witnessTestID); err == nil {
			t.Fatal("invalid frame/EOF accepted")
		}
	}
}

func FuzzDashboardWitnessOriginalGrammar(f *testing.F) {
	f.Add(witnessFrames(witnessGolden()))
	f.Add([]byte{0, 1, 0, 1})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 2<<20 {
			t.Skip()
		}
		s, e := dashboardDecodeWitness(bytes.NewReader(raw), witnessTestID)
		if e == nil && inventory.ValidateSnapshot(s) != nil {
			t.Fatal("independent grammar accepted a contract-invalid original")
		}
	})
}

type witnessAllocationGuard struct{ reads int }

func (g *witnessAllocationGuard) Read(b []byte) (int, error) {
	g.reads++
	if g.reads == 1 {
		copy(b, []byte{0, 1, 0, 1})
		return 4, nil
	}
	panic("oversized frame must be rejected before body read/allocation")
}
func TestDashboardWitnessBoundsBeforeBody(t *testing.T) {
	g := &witnessAllocationGuard{}
	if _, err := dashboardDecodeWitness(g, witnessTestID); err == nil || g.reads != 1 {
		t.Fatal("frame ceiling")
	}
	if _, err := dashboardDecodeWitness(io.LimitReader(bytes.NewReader(witnessFrames(witnessGolden())), 2), witnessTestID); err == nil {
		t.Fatal("short header accepted")
	}
}
