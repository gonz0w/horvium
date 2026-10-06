// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestOwnershipCatalogNames(t *testing.T) {
	names := []string{"ownership-200.json", "writer.lock", "ownership-100.json"}
	before := append([]string(nil), names...)
	guids, err := ownershipCatalogNames(names, 2)
	if err != nil || !reflect.DeepEqual(guids, []string{"100", "200"}) || !reflect.DeepEqual(names, before) {
		t.Fatal("catalog did not retain exact sorted selectors and input", guids, err)
	}
	guids[0] = "changed"
	again, err := ownershipCatalogNames(names, 2)
	if err != nil || again[0] != "100" {
		t.Fatal("catalog aliased caller memory")
	}
	for _, name := range []string{"ownership-0.json", "ownership-0100.json", "ownership-18446744073709551616.json", "ownership-100.JSON", "ownership-100.json/child", "../writer.lock", "job-1.json", "unknown", ".ownership-" + strings.Repeat("a", 31) + ".tmp", ".ownership-" + strings.Repeat("A", 32) + ".tmp"} {
		got, err := ownershipCatalogNames([]string{"writer.lock", name}, 2)
		if !errors.Is(err, ErrStateRecord) || got != nil {
			t.Fatal("unknown name accepted", name, err)
		}
	}
	got, err := ownershipCatalogNames([]string{"writer.lock", ".ownership-" + strings.Repeat("a", 32) + ".tmp"}, 2)
	if !errors.Is(err, ErrStateUncertain) || got != nil {
		t.Fatal("orphan became a usable catalog", err)
	}
	for _, names := range [][]string{{}, {"ownership-100.json"}, {"writer.lock", "writer.lock"}, {"writer.lock", "ownership-100.json", "ownership-100.json"}} {
		if got, err := ownershipCatalogNames(names, 2); err == nil || got != nil {
			t.Fatal("incomplete/duplicate names accepted", names)
		}
	}
}

func TestOwnershipCatalogLimits(t *testing.T) {
	for _, limit := range []int{-1, 0, maxRecords + 1} {
		if got, err := ownershipCatalogNames([]string{"writer.lock"}, limit); !errors.Is(err, ErrLimit) || got != nil {
			t.Fatal("invalid bound accepted", limit, err)
		}
	}
	if got, err := ownershipCatalogNames([]string{"writer.lock"}, 1); err != nil || len(got) != 0 {
		t.Fatal("validated empty catalog denied", err)
	}
	if got, err := ownershipCatalogNames([]string{"writer.lock", "ownership-100.json", "ownership-200.json"}, 1); !errors.Is(err, ErrLimit) || got != nil {
		t.Fatal("record limit truncated or accepted", err)
	}
	if maxOwnershipCatalogBytes != 64<<20 {
		t.Fatal("approved inspection byte ceiling changed")
	}
}

func FuzzOwnershipCatalogNames(f *testing.F) {
	for _, seed := range []string{"writer.lock", "writer.lock\x00ownership-100.json", "writer.lock\x00.ownership-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.tmp", "writer.lock\x00../bad"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data string) {
		if len(data) > 4096 {
			return
		}
		names := strings.Split(data, "\x00")
		got, err := ownershipCatalogNames(names, 64)
		if strings.Join(names, "\x00") != data {
			t.Fatal("input changed")
		}
		if err != nil {
			if got != nil {
				t.Fatal("partial catalog on denial")
			}
			return
		}
		if len(got) != len(names)-1 || len(got) > 64 {
			t.Fatal("missing selectors or unbounded result")
		}
		for i, guid := range got {
			if !validGUID(guid) || i > 0 && got[i-1] >= guid {
				t.Fatal("invalid/duplicate/unordered selector")
			}
		}
	})
}
