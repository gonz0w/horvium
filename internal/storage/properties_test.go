// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const propertyRows = "tank\tguid\t101\t-\t-\n" +
	"tank\tquota\t0\t-\tdefault\n" +
	"tank\trefquota\t0\t-\tdefault\n" +
	"tank\tcompression\tlz4\t-\tlocal\n" +
	"tank/data\tguid\t102\t-\t-\n" +
	"tank/data\tquota\t4096\t8192\tlocal\n" +
	"tank/data\trefquota\t0\t-\tdefault\n" +
	"tank/data\tcompression\tlz4\t-\tinherited from tank\n"

func TestParsePropertySourcesAndLiteralLimits(t *testing.T) {
	files, err := ParseProperties([]byte(propertyRows), "tank")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].GUID != "101" || files[1].ParentGUID != "101" {
		t.Fatal("identity mapping incorrect")
	}
	if files[0].Properties.Quota.Effective != "none" || files[1].Properties.Quota.Effective != "4096" || files[1].Properties.Quota.ReceivedValue != "8192" || files[1].Properties.Compression.SourceGUID != "101" {
		t.Fatalf("incorrect projection: %+v", files)
	}
	shuffled := strings.Split(strings.TrimSuffix(propertyRows, "\n"), "\n")
	for left, right := 0, len(shuffled)-1; left < right; left, right = left+1, right-1 {
		shuffled[left], shuffled[right] = shuffled[right], shuffled[left]
	}
	reordered, err := ParseProperties([]byte(strings.Join(shuffled, "\n")+"\n"), "tank")
	if err != nil || !reflect.DeepEqual(files, reordered) {
		t.Fatal("order affected projection", err)
	}
}

func TestPropertyParserFailsWithoutPartialRows(t *testing.T) {
	for _, tc := range []struct{ name, data string }{
		{"empty", ""},
		{"truncated", strings.TrimSuffix(propertyRows, "\n")},
		{"missing property", strings.Replace(propertyRows, "tank\tquota\t0\t-\tdefault\n", "", 1)},
		{"duplicate property", propertyRows + "tank\tquota\t0\t-\tdefault\n"},
		{"duplicate guid", strings.Replace(propertyRows, "guid\t102", "guid\t101", 1)},
		{"unknown property", strings.Replace(propertyRows, "\tquota\t", "\tmountpoint\t", 1)},
		{"unknown source", strings.Replace(propertyRows, "\tlocal\n", "\ttemporary\n", 1)},
		{"false inherited source", strings.Replace(propertyRows, "inherited from tank", "inherited from tank/data", 1)},
		{"missing parent", strings.ReplaceAll(propertyRows, "tank/data", "tank/missing/data")},
		{"other pool", strings.ReplaceAll(propertyRows, "tank/data", "foreign/data")},
		{"guid source", strings.Replace(propertyRows, "guid\t101\t-\t-", "guid\t101\t-\tlocal", 1)},
		{"guid received", strings.Replace(propertyRows, "guid\t101\t-\t-", "guid\t101\t102\t-", 1)},
		{"quota suffix", strings.Replace(propertyRows, "quota\t4096", "quota\t4K", 1)},
		{"quota noncanonical", strings.Replace(propertyRows, "quota\t4096", "quota\t04096", 1)},
		{"overflow", strings.Replace(propertyRows, "quota\t4096", "quota\t18446744073709551616", 1)},
		{"invalid received", strings.Replace(propertyRows, "\t8192\t", "\tunknown\t", 1)},
		{"control", strings.Replace(propertyRows, "\tlz4\t", "\tlz4\r\t", 1)},
		{"delimiter flood", strings.Replace(propertyRows, "\tlz4\t", "\tlz4\t\t", 1)},
		{"invalid utf8", propertyRows + string([]byte{255})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files, err := ParseProperties([]byte(tc.data), "tank")
			if err == nil || files != nil {
				t.Fatal("malformed property data retained", err)
			}
		})
	}
}

func TestReceivedValuesAndDrift(t *testing.T) {
	rows := strings.Replace(propertyRows, "quota\t4096\t8192\tlocal", "quota\t8192\t8192\treceived", 1)
	files, err := ParseProperties([]byte(rows), "tank")
	if err != nil {
		t.Fatal(err)
	}
	if files[1].Properties.Quota.Source != Received || files[1].Properties.Quota.SourceGUID != "102" {
		t.Fatal("received source lost")
	}
	rows = strings.Replace(rows, "quota\t8192\t8192\treceived", "quota\t8192\t4096\treceived", 1)
	if _, err := ParseProperties([]byte(rows), "tank"); err == nil {
		t.Fatal("contradictory received value accepted")
	}
	p, o, i := fixture()
	o.Filesystems[0].Properties.Compression.ReceivedValue = "off"
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	o.Filesystems[0].Properties.Compression.ReceivedValue = "lz4"
	if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
		t.Fatal("shadowed received drift ignored", err)
	}
}

func TestPropertyParserBounds(t *testing.T) {
	for _, data := range [][]byte{make([]byte, maxObservationBytes+1), []byte(strings.Repeat("x", maxFieldBytes+1) + "\tguid\t101\t-\t-\n"), []byte(strings.Repeat("a\tb\tc\td\te\n", 4*maxRecords+1))} {
		files, err := ParseProperties(data, "tank")
		if !errors.Is(err, ErrLimit) || files != nil {
			t.Fatal(err)
		}
	}
}

func FuzzProperties(f *testing.F) {
	f.Add([]byte(propertyRows))
	f.Add([]byte("tank\tguid\t0\t-\t-\n"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		files, err := ParseProperties(data, "tank")
		if err != nil {
			if files != nil {
				t.Fatal("partial rows on failure")
			}
			return
		}
		if len(files) == 0 {
			t.Fatal("empty successful projection")
		}
		// Parsing alone never proves native write eligibility.
		if _, err := indexObservation(Observation{Complete: true, PoolGUID: "100", PoolName: "tank", Filesystems: files}); !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
	})
}
