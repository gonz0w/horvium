// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const diskInfoFixture = "/dev/ada0\t4096\t3000000000000\t732421875\t0\t0\n"

func TestDiskObservationCorrelatesDirectIdentityAndLiteralGeometry(t *testing.T) {
	for _, data := range []string{diskInfoFixture, strings.TrimSuffix(diskInfoFixture, "\n") + "\t100\t255\t63\n"} {
		o, err := ParseDiskObservation([]byte(geomFixture), []byte(data), []byte("serial-A\n"), DefaultLimits())
		if err != nil || o.Provider.Name != "ada0" || o.Provider.Identity != IdentityUnique || o.SectorCount != 732421875 || o.StripeSize != 0 || o.StripeOffset != 0 {
			t.Fatal("native projections did not correlate", err)
		}
		if len(o.Provider.Identifiers) != 2 || o.Provider.Identifiers[1].Kind != "lunid" {
			t.Fatal("identifier provenance was lost")
		}
	}
	// A different sector and nonzero stripe/offset are observations, not a
	// hardcoded 512/4096 or physical-sector policy.
	g := strings.Replace(geomFixture, "<sectorsize>4096</sectorsize>", "<sectorsize>1024</sectorsize>", 1)
	o, err := ParseDiskObservation([]byte(g), []byte("/dev/ada0\t1024\t3000000000000\t2929687500\t8192\t1024\n"), []byte("serial-A\n"), DefaultLimits())
	if err != nil || o.StripeSize != 8192 || o.StripeOffset != 1024 {
		t.Fatal("literal stripe observations were replaced", err)
	}
}

func TestDiskObservationRefusesUncertainIdentityAndBacking(t *testing.T) {
	for name, graph := range map[string]string{
		"duplicate serial":         strings.Replace(geomFixture, "<geom id=\"g2\"><name>ada1</name>", "<geom id=\"g2\"><name>ada1</name><config><ident>serial-A</ident></config>", 1),
		"missing serial":           strings.Replace(geomFixture, "<ident>serial-A</ident>", "", 1),
		"conflicting provenance":   strings.Replace(geomFixture, "<provider id=\"p1\">", "<provider id=\"p1\"><config><ident>serial-B</ident></config>", 1),
		"logical provider":         strings.Replace(geomFixture, "<name>DISK</name>", "<name>MULTIPATH</name>", 1),
		"backing parents":          strings.Replace(geomFixture, "<geom id=\"g1\">", "<geom id=\"g1\"><consumer id=\"parent\"><provider ref=\"p2\"/></consumer>", 1),
		"conflicting geom address": strings.Replace(geomFixture, "<name>ada0</name><config>", "<name>other</name><config>", 1),
		"duplicate address":        strings.Replace(geomFixture, "<provider id=\"p2\"><name>ada1</name>", "<provider id=\"p2\"><name>ada0</name>", 1),
	} {
		t.Run(name, func(t *testing.T) {
			o, err := ParseDiskObservation([]byte(graph), []byte(diskInfoFixture), []byte("serial-A\n"), DefaultLimits())
			if err == nil || !reflect.DeepEqual(o, DiskObservation{}) {
				t.Fatal("uncertain correlation returned a usable observation")
			}
		})
	}
	for _, serial := range []string{"\n", "serial-B\n", " serial-A\n", "serial-A \n", "serial-A", "serial-A\nextra\n", "serial-A\x00\n"} {
		if _, err := ParseDiskObservation([]byte(geomFixture), []byte(diskInfoFixture), []byte(serial), DefaultLimits()); err == nil {
			t.Fatal("unbound or malformed independent serial accepted")
		}
	}
}

func TestDiskObservationRejectsMalformedOrContradictoryGeometry(t *testing.T) {
	for _, data := range []string{
		"", strings.TrimSuffix(diskInfoFixture, "\n"), diskInfoFixture + diskInfoFixture,
		strings.ReplaceAll(diskInfoFixture, "\t", " "),
		strings.Replace(diskInfoFixture, "/dev/ada0", "ada0", 1),
		strings.Replace(diskInfoFixture, "/dev/ada0", "/dev/../ada0", 1),
		strings.Replace(diskInfoFixture, "/dev/ada0", "/dev/ada0p1", 1),
		strings.Replace(diskInfoFixture, "4096", "04096", 1),
		strings.Replace(diskInfoFixture, "4096", "512", 1),
		// Internally consistent rows must still bind to GEOM, rather than
		// succeeding merely because their own capacity arithmetic agrees.
		"/dev/ada0\t512\t3000000000000\t5859375000\t0\t0\n",
		"/dev/ada0\t4096\t3000000004096\t732421876\t0\t0\n",
		strings.Replace(diskInfoFixture, "4096", "0", 1),
		strings.Replace(diskInfoFixture, "3000000000000", "3000000000001", 1),
		strings.Replace(diskInfoFixture, "3000000000000", "9223372036854775808", 1),
		strings.Replace(diskInfoFixture, "732421875", "732421876", 1),
		strings.Replace(diskInfoFixture, "\t0\t0", "\t-1\t0", 1),
		strings.Replace(diskInfoFixture, "\t0\t0", "\t18446744073709551616\t0", 1),
		strings.TrimSuffix(diskInfoFixture, "\n") + "\t0\t0\tunknown\n",
		diskInfoFixture + "\n",
	} {
		if _, err := ParseDiskObservation([]byte(geomFixture), []byte(data), []byte("serial-A\n"), DefaultLimits()); err == nil {
			t.Fatal("malformed or contradictory geometry accepted")
		}
	}
}

func TestDiskObservationBoundsAllInputsBeforeProjection(t *testing.T) {
	l := DefaultLimits()
	l.MaxBytes = len(geomFixture) + len(diskInfoFixture) + len("serial-A\n") - 1
	if _, err := ParseDiskObservation([]byte(geomFixture), []byte(diskInfoFixture), []byte("serial-A\n"), l); !errors.Is(err, ErrLimit) {
		t.Fatal("aggregate byte bound ignored", err)
	}
	l = DefaultLimits()
	l.MaxFieldBytes = 4
	if _, err := ParseDiskObservation([]byte(geomFixture), []byte(diskInfoFixture), []byte("serial-A\n"), l); !errors.Is(err, ErrLimit) {
		t.Fatal("field bound ignored", err)
	}
	l = DefaultLimits()
	l.MaxRecords = 1
	if _, err := ParseDiskObservation([]byte(geomFixture), []byte(diskInfoFixture), []byte("serial-A\n"), l); !errors.Is(err, ErrLimit) {
		t.Fatal("graph record bound ignored", err)
	}
	l.MaxBytes = 0
	if _, err := ParseDiskObservation(nil, nil, nil, l); !errors.Is(err, ErrInvalidLimits) {
		t.Fatal("invalid budget accepted", err)
	}
}

func TestDiskObservationRejectsOversizedFieldsAndInvalidEncoding(t *testing.T) {
	for _, input := range []struct{ info, serial []byte }{
		{[]byte(strings.Repeat("a", 4097) + "\t4096\t3000000000000\t732421875\t0\t0\n"), []byte("serial-A\n")},
		{[]byte(diskInfoFixture), []byte(strings.Repeat("a", 4097) + "\n")},
	} {
		if _, err := ParseDiskObservation([]byte(geomFixture), input.info, input.serial, DefaultLimits()); !errors.Is(err, ErrLimit) {
			t.Fatal("oversized field accepted", err)
		}
	}
	if _, err := ParseDiskObservation([]byte(geomFixture), []byte(diskInfoFixture), []byte{0xff, '\n'}, DefaultLimits()); !errors.Is(err, ErrMalformed) {
		t.Fatal("invalid UTF-8 accepted", err)
	}
}

func FuzzDiskObservation(f *testing.F) {
	f.Add(geomFixture, diskInfoFixture, "serial-A\n")
	f.Add(geomFixture, diskInfoFixture, "\n")
	f.Add("<mesh/>", "/dev/ada0\t0\t0\t0\t0\t0\n", "serial-A\n")
	f.Fuzz(func(t *testing.T, geom, info, serial string) {
		l := DefaultLimits()
		l.MaxBytes, l.MaxRecords, l.MaxFieldBytes = 64<<10, 1024, 1024
		o, err := ParseDiskObservation([]byte(geom), []byte(info), []byte(serial), l)
		if err != nil {
			if !reflect.DeepEqual(o, DiskObservation{}) {
				t.Fatal("failure exposed a partial projection")
			}
			return
		}
		p := o.Provider
		if p.Kind != DiskCandidate || p.Identity != IdentityUnique || p.SectorSize == 0 || p.MediaSize%p.SectorSize != 0 || o.SectorCount != p.MediaSize/p.SectorSize || strings.ContainsAny(p.Name, "/\x00\n") {
			t.Fatal("invalid successful correspondence")
		}
	})
}
