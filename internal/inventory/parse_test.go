// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestZFSInventoryValues(t *testing.T) {
	// M1-PARSE-001: GUIDs remain exact strings beyond JavaScript's integer range.
	pools, err := ParsePools([]byte("tank\t18446744073709551615\t1000\t400\t600\tONLINE\n"), DefaultLimits())
	if err != nil || len(pools) != 1 {
		t.Fatalf("pools: %v", err)
	}
	if p := pools[0]; p.GUID != "18446744073709551615" || p.Size != 1000 || p.Allocated != 400 || p.Free != 600 || p.Health != "ONLINE" {
		t.Fatalf("wrong typed pool: %+v", p)
	}
	datasets, err := ParseDatasets([]byte("tank/data\t9007199254740993\t0\t8192\t4096\n"), DefaultLimits())
	if err != nil || len(datasets) != 1 || datasets[0].GUID != "9007199254740993" || datasets[0].Referenced != 4096 {
		t.Fatalf("datasets: %+v %v", datasets, err)
	}
	if empty, err := ParsePools(nil, DefaultLimits()); err != nil || len(empty) != 0 {
		t.Fatalf("successful empty result: %v", err)
	}
}

func TestZFSRejectsMalformedOutput(t *testing.T) {
	valid := "tank\t9\t1000\t400\t600\tONLINE\n"
	for name, input := range map[string]string{
		"field count":            "tank\t9\t1000\n",
		"human units":            strings.Replace(valid, "1000", "1K", 1),
		"overflow":               strings.Replace(valid, "1000", "18446744073709551616", 1),
		"negative":               strings.Replace(valid, "400", "-1", 1),
		"unknown health":         strings.Replace(valid, "ONLINE", "MYSTERY", 1),
		"unsafe name":            strings.Replace(valid, "tank", "tank\x00", 1),
		"invalid UTF8":           strings.Replace(valid, "tank", "tank\xff", 1),
		"zero GUID":              strings.Replace(valid, "\t9\t", "\t0\t", 1),
		"leading zero GUID":      strings.Replace(valid, "\t9\t", "\t09\t", 1),
		"truncated":              strings.TrimSuffix(valid, "\n"),
		"duplicate identity":     valid + strings.Replace(valid, "tank", "other", 1),
		"duplicate name":         valid + strings.Replace(valid, "\t9\t", "\t10\t", 1),
		"blank row":              valid + "\n",
		"contradictory capacity": strings.Replace(valid, "600", "1001", 1),
	} {
		t.Run(name, func(t *testing.T) {
			items, err := ParsePools([]byte(input), DefaultLimits())
			if !errors.Is(err, ErrMalformed) || len(items) != 0 {
				t.Fatalf("malformed inventory must not publish partial rows: %+v %v", items, err)
			}
		})
	}
	for _, input := range []string{"tank/x\t1\t0\t0\n", "tank/x\t1\tNaN\t0\t0\n", "tank/x\t1\t0\t0\t0\nother\t1\t0\t0\t0\n"} {
		if items, err := ParseDatasets([]byte(input), DefaultLimits()); !errors.Is(err, ErrMalformed) || len(items) != 0 {
			t.Fatalf("dataset malformed result: %+v %v", items, err)
		}
	}
}

func TestParserLimits(t *testing.T) {
	input := []byte("tank\t1\t10\t0\t10\tONLINE\nother\t2\t10\t0\t10\tONLINE\n")
	for name, change := range map[string]func(*Limits){
		"bytes":   func(l *Limits) { l.MaxBytes = 10 },
		"records": func(l *Limits) { l.MaxRecords = 1 },
		"field":   func(l *Limits) { l.MaxFieldBytes = 3 },
	} {
		t.Run(name, func(t *testing.T) {
			limits := DefaultLimits()
			change(&limits)
			if rows, err := ParsePools(input, limits); !errors.Is(err, ErrLimit) || len(rows) != 0 {
				t.Fatalf("limits must fail explicitly: %v", err)
			}
		})
	}
	if _, err := ParsePools(nil, Limits{}); !errors.Is(err, ErrInvalidLimits) {
		t.Fatalf("invalid budget accepted: %v", err)
	}
}

const geomFixture = `<mesh>
 <class id="c1"><name>DISK</name>
  <geom id="g1"><class ref="c1"/><name>ada0</name><config><ident>serial-A</ident><lunid>lun-A</lunid></config>
   <provider id="p1"><geom ref="g1"/><name>ada0</name><mediasize>3000000000000</mediasize><sectorsize>4096</sectorsize></provider>
  </geom>
  <geom id="g2"><name>ada1</name><provider id="p2"><name>ada1</name><mediasize>3000000000000</mediasize><sectorsize>512</sectorsize></provider></geom>
 </class>
 <class id="c2"><name>PART</name>
  <geom id="g3"><name>ada0</name><consumer id="x1"><geom ref="g3"/><provider ref="p1"/></consumer>
   <provider id="p3"><name>ada0p1</name><mediasize>1000000</mediasize><sectorsize>4096</sectorsize></provider>
  </geom>
 </class>
</mesh>`

func TestGEOMRelationshipsAndIdentity(t *testing.T) {
	// M1-IDENTITY-001: graph IDs are scoped references; serial/lunid are observations.
	graph, err := ParseGEOM([]byte(geomFixture), DefaultLimits())
	if err != nil || len(graph.Providers) != 3 || len(graph.Classes) != 2 || len(graph.Geoms) != 3 {
		t.Fatalf("graph: %+v %v", graph, err)
	}
	if p := graph.Providers[0]; p.Class != "DISK" || p.Kind != DiskCandidate || p.MediaSize != 3000000000000 || p.SectorSize != 4096 || p.Identity != IdentityUnique || len(p.Identifiers) != 2 {
		t.Fatalf("disk identity/geometry: %+v", p)
	}
	if graph.Providers[1].Identity != IdentityMissing || graph.Providers[2].Kind != Partition || graph.Providers[2].Identity != IdentityNotApplicable || graph.Geoms[2].Consumers[0].ProviderID != "p1" {
		t.Fatalf("missing/logical identity or parent relation: %+v", graph)
	}
	duplicate := strings.Replace(geomFixture, "<geom id=\"g2\"><name>ada1</name>", "<geom id=\"g2\"><name>ada1</name><config><ident>serial-A</ident></config>", 1)
	graph, err = ParseGEOM([]byte(duplicate), DefaultLimits())
	if err != nil || graph.Providers[0].Identity != IdentityAmbiguous || graph.Providers[1].Identity != IdentityAmbiguous {
		t.Fatalf("duplicate serial must remain ambiguous: %+v %v", graph, err)
	}
	conflict := strings.Replace(geomFixture, "<provider id=\"p1\">", "<provider id=\"p1\"><config><ident>other-serial</ident></config>", 1)
	graph, err = ParseGEOM([]byte(conflict), DefaultLimits())
	if err != nil || graph.Providers[0].Identity != IdentityAmbiguous {
		t.Fatalf("conflicting provenance: %+v %v", graph, err)
	}
}

func TestGEOMSysctlTerminator(t *testing.T) {
	// kern.geom.confxml's SYSCTL_OUT copies sbuf_len + 1, including its NUL.
	if graph, err := ParseGEOM([]byte(geomFixture+"\x00"), DefaultLimits()); err != nil || len(graph.Providers) != 3 {
		t.Fatalf("native sysctl terminator rejected: %v", err)
	}
	for _, input := range []string{geomFixture + "\x00\x00", strings.Replace(geomFixture, "ada0", "ada\x000", 1), geomFixture + "junk\x00"} {
		if _, err := ParseGEOM([]byte(input), DefaultLimits()); !errors.Is(err, ErrMalformed) {
			t.Fatal("embedded/repeated NUL or trailing junk accepted")
		}
	}
}

func TestGEOMRejectsUnsafeOrBrokenGraph(t *testing.T) {
	for name, input := range map[string]string{
		"truncated":            strings.TrimSuffix(geomFixture, "</mesh>"),
		"DTD":                  `<!DOCTYPE mesh [<!ENTITY x SYSTEM "file:///etc/passwd">]>` + geomFixture,
		"wrong root":           "<other/>",
		"multiple roots":       geomFixture + "<mesh/>",
		"duplicate ID":         strings.Replace(geomFixture, `id="p2"`, `id="p1"`, 1),
		"dangling reference":   strings.Replace(geomFixture, `provider ref="p1"`, `provider ref="absent"`, 1),
		"contradictory parent": strings.Replace(geomFixture, `geom ref="g1"`, `geom ref="g2"`, 1),
		"overflow":             strings.Replace(geomFixture, "3000000000000", "18446744073709551616", 1),
		"duplicate field":      strings.Replace(geomFixture, "<name>ada0</name>", "<name>ada0</name><name>other</name>", 1),
		"missing geometry":     strings.Replace(geomFixture, "<sectorsize>512</sectorsize>", "", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if graph, err := ParseGEOM([]byte(input), DefaultLimits()); !errors.Is(err, ErrMalformed) || len(graph.Providers) != 0 {
				t.Fatalf("broken graph must fail: %v", err)
			}
		})
	}
	for _, change := range []func(*Limits){
		func(l *Limits) { l.MaxXMLDepth = 2 },
		func(l *Limits) { l.MaxXMLElements = 5 },
		func(l *Limits) { l.MaxRecords = 2 },
	} {
		limits := DefaultLimits()
		change(&limits)
		if _, err := ParseGEOM([]byte(geomFixture), limits); !errors.Is(err, ErrLimit) {
			t.Fatalf("XML limit not enforced: %v", err)
		}
	}
}

func FuzzPools(f *testing.F) {
	f.Add("tank\t1\t10\t0\t10\tONLINE\n")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		limits := DefaultLimits()
		limits.MaxBytes = 4096
		rows, err := ParsePools([]byte(input), limits)
		if err != nil && len(rows) != 0 {
			t.Fatal("partial output on failure")
		}
	})
}

func FuzzGEOM(f *testing.F) {
	f.Add(geomFixture)
	f.Add("<mesh/>")
	f.Fuzz(func(t *testing.T, input string) {
		limits := DefaultLimits()
		limits.MaxBytes = 4096
		graph, err := ParseGEOM([]byte(input), limits)
		if err != nil && len(graph.Providers) != 0 {
			t.Fatal("partial graph on failure")
		}
	})
}

func datasetFixture(count int) []byte {
	var data strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&data, "tank/data%d\t%d\t1\t10240\t512\n", i, uint64(9007199254740993)+uint64(i))
	}
	return []byte(data.String())
}

func TestLargeDatasetProjection(t *testing.T) {
	// M1-SCALE-001 parser portion: no small-host dataset-count assumption.
	input := datasetFixture(10000)
	items, err := ParseDatasets(input, DefaultLimits())
	if err != nil || len(items) != 10000 || items[9999].GUID != strconv.FormatUint(9007199254740993+9999, 10) {
		t.Fatalf("large inventory lost rows/precision: %v", err)
	}
	limits := DefaultLimits()
	limits.MaxRecords = 9999
	if rows, err := ParseDatasets(input, limits); !errors.Is(err, ErrLimit) || len(rows) != 0 {
		t.Fatal("large inventory silently truncated at record budget")
	}
}

func BenchmarkDatasetProjection(b *testing.B) {
	for _, count := range []int{100, 10000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			input := datasetFixture(count)
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			b.ResetTimer()
			for b.Loop() {
				if _, err := ParseDatasets(input, DefaultLimits()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
