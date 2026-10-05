// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

const importedList = "tank\t100\t67108864\t4096\t67104768\tONLINE\n"

func TestImportedPoolsMapUseWithoutMounts(t *testing.T) {
	got, err := ParseImportedPoolUse([]byte(systemGEOM), []byte(importedList), []PoolUseCapture{systemCapture()}, inventory.DefaultLimits())
	if err != nil || len(got) != 1 || got[0].Name != "tank" || got[0].GUID != "100" || got[0].Health != "ONLINE" || len(got[0].Disks) != 2 || got[0].Disks[0].Name != "ada0" || got[0].Disks[1].Name != "ada1" || got[0].Disks[0].Identity != inventory.IdentityMissing {
		t.Fatal("imported membership lost without any mount capture", err)
	}
	empty, err := ParseImportedPoolUse([]byte(systemGEOM), nil, nil, inventory.DefaultLimits())
	if err != nil || len(empty) != 0 {
		t.Fatal("successful empty catalogue unavailable", err)
	}
}

func TestImportedPoolsRequireExactCatalogueAndCaptures(t *testing.T) {
	extra := systemCapture()
	extra.Name = "other"
	extra.GUID = []byte("200\n")
	extra.StatusGUIDs = []byte(strings.ReplaceAll(topologyGUIDRows, "tank", "other"))
	extra.StatusPaths = []byte(strings.ReplaceAll(topologyPathRows, "tank", "other"))
	for _, tc := range []struct {
		list     string
		captures []PoolUseCapture
	}{
		{importedList, nil}, {"", []PoolUseCapture{systemCapture()}}, {importedList, []PoolUseCapture{systemCapture(), systemCapture()}},
		{importedList + "other\t200\t67108864\t4096\t67104768\tONLINE\n", []PoolUseCapture{systemCapture()}},
		{importedList, []PoolUseCapture{extra}}, {strings.Replace(importedList, "\t100\t", "\t200\t", 1), []PoolUseCapture{systemCapture()}},
		{importedList + importedList, []PoolUseCapture{systemCapture(), systemCapture()}},
		{importedList + "other\t100\t67108864\t4096\t67104768\tONLINE\n", []PoolUseCapture{systemCapture(), extra}},
		{strings.TrimSuffix(importedList, "\n"), []PoolUseCapture{systemCapture()}},
		{strings.Replace(importedList, "ONLINE", "UNKNOWN", 1), []PoolUseCapture{systemCapture()}},
		{strings.Replace(importedList, "ONLINE", "DEGRADED", 1), []PoolUseCapture{systemCapture()}},
		{strings.Replace(importedList, "tank", "tank/child", 1), []PoolUseCapture{systemCapture()}},
	} {
		if got, err := ParseImportedPoolUse([]byte(systemGEOM), []byte(tc.list), tc.captures, inventory.DefaultLimits()); err == nil || got != nil {
			t.Fatal("missing/extra/duplicate/contradictory catalogue accepted", err)
		}
	}
	for _, change := range []func(*PoolUseCapture){
		func(p *PoolUseCapture) { p.GUID = []byte("0100\n") },
		func(p *PoolUseCapture) {
			p.StatusPaths = []byte(strings.Replace(topologyPathRows, "/dev/ada1", "/dev/missing", 1))
		},
		func(p *PoolUseCapture) {
			p.StatusPaths = []byte(strings.Replace(topologyPathRows, "/dev/ada1", "/tmp/disk", 1))
		},
		func(p *PoolUseCapture) {
			p.StatusGUIDs = []byte(strings.Replace(topologyGUIDRows, "902 ONLINE", "902 DEGRADED", 1))
		},
	} {
		p := systemCapture()
		change(&p)
		if got, err := ParseImportedPoolUse([]byte(systemGEOM), []byte(importedList), []PoolUseCapture{p}, inventory.DefaultLimits()); err == nil || got != nil {
			t.Fatal("unavailable pool backing accepted", err)
		}
	}
}

func TestImportedPoolsRetainOverlapAndSortByGUID(t *testing.T) {
	other := systemCapture()
	other.Name = "aaa"
	other.GUID = []byte("200\n")
	other.StatusGUIDs = []byte(strings.ReplaceAll(topologyGUIDRows, "tank", "aaa"))
	other.StatusPaths = []byte(strings.ReplaceAll(topologyPathRows, "tank", "aaa"))
	list := importedList + "aaa\t200\t67108864\t4096\t67104768\tONLINE\n"
	got, err := ParseImportedPoolUse([]byte(systemGEOM), []byte(list), []PoolUseCapture{other, systemCapture()}, inventory.DefaultLimits())
	if err != nil || len(got) != 2 || got[0].GUID != "100" || got[1].GUID != "200" || !reflect.DeepEqual(got[0].Disks, got[1].Disks) {
		t.Fatal("positive overlapping pool uses lost or order unstable", err)
	}
}

func TestImportedPoolsBoundWholeProjection(t *testing.T) {
	for _, change := range []func(*inventory.Limits){
		func(l *inventory.Limits) { l.MaxBytes = len(systemGEOM) + len(importedList) },
		func(l *inventory.Limits) { l.MaxRecords = 23 },
		func(l *inventory.Limits) { l.MaxFieldBytes = 8 },
	} {
		l := inventory.DefaultLimits()
		change(&l)
		if got, err := ParseImportedPoolUse([]byte(systemGEOM), []byte(importedList), []PoolUseCapture{systemCapture()}, l); !errors.Is(err, ErrLimit) || got != nil {
			t.Fatal("aggregate/field bound ignored", err)
		}
	}
	if got, err := ParseImportedPoolUse([]byte(systemGEOM), []byte(importedList), []PoolUseCapture{systemCapture()}, inventory.Limits{}); !errors.Is(err, ErrInvalid) || got != nil {
		t.Fatal("invalid limits accepted", err)
	}
	l := inventory.DefaultLimits()
	l.MaxRecords = 100
	if got, err := ParseImportedPoolUse([]byte(systemGEOM), []byte(strings.Repeat(importedList, 101)), nil, l); !errors.Is(err, ErrLimit) || got != nil {
		t.Fatal("catalogue row flood not bounded before parsing", err)
	}
}

func FuzzImportedPoolUse(f *testing.F) {
	f.Add(systemGEOM, importedList, "tank", "100\n", topologyGUIDRows, topologyPathRows)
	f.Add(systemGEOM, "", "", "", "", "")
	f.Fuzz(func(t *testing.T, geom, list, name, guid, statusGUIDs, statusPaths string) {
		l := inventory.DefaultLimits()
		l.MaxBytes = 64 << 10
		l.MaxRecords = 1024
		l.MaxFieldBytes = 1024
		var captures []PoolUseCapture
		if name != "" || guid != "" || statusGUIDs != "" || statusPaths != "" {
			captures = []PoolUseCapture{{Name: name, GUID: []byte(guid), StatusGUIDs: []byte(statusGUIDs), StatusPaths: []byte(statusPaths)}}
		}
		got, err := ParseImportedPoolUse([]byte(geom), []byte(list), captures, l)
		if err != nil {
			if got != nil {
				t.Fatal("partial imported use")
			}
			return
		}
		for _, pool := range got {
			if !validGUID(pool.GUID) || len(pool.Disks) == 0 {
				t.Fatal("pool identity/backing absent")
			}
			for _, disk := range pool.Disks {
				if disk.Kind != inventory.DiskCandidate {
					t.Fatal("logical disk promoted")
				}
			}
		}
	})
}

func TestImportedPoolsPreserveAdverseHealthAndAmbiguousDiskUse(t *testing.T) {
	p := systemCapture()
	p.StatusGUIDs = []byte(strings.ReplaceAll(topologyGUIDRows, "ONLINE", "DEGRADED"))
	p.StatusPaths = []byte(strings.ReplaceAll(topologyPathRows, "ONLINE", "DEGRADED"))
	geom := strings.ReplaceAll(systemGEOM, "<mediasize>67108864</mediasize>", "<config><ident>duplicate</ident></config><mediasize>67108864</mediasize>")
	got, err := ParseImportedPoolUse([]byte(geom), []byte(strings.Replace(importedList, "ONLINE", "DEGRADED", 1)), []PoolUseCapture{p}, inventory.DefaultLimits())
	if err != nil || len(got) != 1 || got[0].Health != "DEGRADED" || len(got[0].Disks) != 2 || got[0].Disks[0].Identity != inventory.IdentityAmbiguous || got[0].Disks[1].Identity != inventory.IdentityAmbiguous {
		t.Fatal("adverse health/duplicate identity hid positive pool use", err)
	}
}
