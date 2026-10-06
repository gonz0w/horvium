// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

var accessFixture = strings.Replace(attachmentFixture, `<provider id="p">`, `<provider id="p"><mode>r2w1e1</mode>`, 1)

func accessGraph() string {
	g := strings.Replace(accessFixture, `<consumer id="c">`, `<consumer id="c"><mode>r1w1e0</mode>`, 1)
	return strings.Replace(g, `<consumer id="lc">`, `<consumer id="lc"><mode>r1w0e1</mode>`, 1)
}

const accessInfo = "/dev/ada0\t512\t67108864\t131072\t0\t0\n"

func TestDiskAccessBindsAggregateToIndependentDiskIdentity(t *testing.T) {
	got, err := ParseDiskAccess([]byte(accessGraph()), []byte(accessInfo), []byte("serial-A\n"), DefaultLimits())
	if err != nil || got.Disk.Provider.Name != "ada0" || got.Mode != (AccessMode{Read: 2, Write: 1, Exclusive: 1}) {
		t.Fatal("aggregate access was not bound to selected disk", err)
	}
	// No consumer-class interpretation, PID attribution or eligibility is claimed.
	for _, mode := range []string{"r0w0e0", "r2147483647w0e0"} {
		g := strings.Replace(accessGraph(), "r2w1e1", mode, 1)
		g = strings.Replace(g, "r1w1e0", mode, 1)
		g = strings.Replace(g, "r1w0e1", "r0w0e0", 1)
		if _, err := ParseDiskAccess([]byte(g), []byte(accessInfo), []byte("serial-A\n"), DefaultLimits()); err != nil {
			t.Fatal("valid native counter refused", err)
		}
	}
}

func TestDiskAccessRefusesMissingMalformedOrContradictoryModes(t *testing.T) {
	for _, mode := range []string{"", "r-1w0e0", "r+1w0e0", "r01w0e0", "r2147483648w0e0", "r1w0e0x", "r1w0", "r1 w0e0", " r2w1e1", "r2w1e1\n", "r2w1e1\x00", "<nested/>", "r1w1e1"} {
		g := strings.Replace(accessGraph(), "r2w1e1", mode, 1)
		if got, err := ParseDiskAccess([]byte(g), []byte(accessInfo), []byte("serial-A\n"), DefaultLimits()); err == nil || !reflect.DeepEqual(got, DiskAccess{}) {
			t.Fatalf("invalid aggregate escaped: %q", mode)
		}
	}
	for _, g := range []string{
		strings.Replace(accessGraph(), "<mode>r2w1e1</mode>", "", 1),
		strings.Replace(accessGraph(), "<mode>r1w0e1</mode>", "", 1),
		strings.Replace(accessGraph(), "<mode>r2w1e1</mode>", "<mode>r2w1e1</mode><mode>r2w1e1</mode>", 1),
		strings.Replace(accessGraph(), "r1w0e1", "r2147483647w0e1", 1),
		strings.Replace(accessGraph(), "serial-A", "", 1),
		strings.Replace(accessGraph(), `<provider ref="p"/>`, `<provider ref="missing"/>`, 1),
	} {
		if got, err := ParseDiskAccess([]byte(g), []byte(accessInfo), []byte("serial-A\n"), DefaultLimits()); err == nil || !reflect.DeepEqual(got, DiskAccess{}) {
			t.Fatal("partial/contradictory access escaped", err)
		}
	}
	if _, err := ParseDiskAccess([]byte(accessGraph()), []byte(accessInfo), []byte("other\n"), DefaultLimits()); !errors.Is(err, ErrDiskCorrelation) {
		t.Fatal("independent serial disagreement ignored", err)
	}
}

func TestDiskAccessBudgetsAndNoPartialResult(t *testing.T) {
	for _, edit := range []func(*Limits){
		func(l *Limits) { l.MaxBytes = len(accessGraph()) + len(accessInfo) + len("serial-A\n") - 1 },
		func(l *Limits) { l.MaxRecords = 10 }, func(l *Limits) { l.MaxXMLElements = 20 }, func(l *Limits) { l.MaxFieldBytes = 3 },
	} {
		l := DefaultLimits()
		edit(&l)
		if got, err := ParseDiskAccess([]byte(accessGraph()), []byte(accessInfo), []byte("serial-A\n"), l); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(got, DiskAccess{}) {
			t.Fatal("limit or zero-result contract failed", err)
		}
	}
}

func FuzzDiskAccess(f *testing.F) {
	f.Add(accessGraph(), accessInfo, "serial-A\n")
	f.Fuzz(func(t *testing.T, geom, info, serial string) {
		l := DefaultLimits()
		l.MaxBytes, l.MaxRecords, l.MaxFieldBytes = 64<<10, 1024, 1024
		got, err := ParseDiskAccess([]byte(geom), []byte(info), []byte(serial), l)
		if err != nil {
			if !reflect.DeepEqual(got, DiskAccess{}) {
				t.Fatal("partial access")
			}
			return
		}
		if got.Disk.Provider.Identity != IdentityUnique || got.Disk.Provider.Kind != DiskCandidate {
			t.Fatal("unbound access")
		}
	})
}
