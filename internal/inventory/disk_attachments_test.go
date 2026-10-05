// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const attachmentFixture = `<mesh>
<class id="disk"><name>DISK</name><geom id="d"><name>ada0</name><config><ident>serial-A</ident></config><provider id="p"><name>ada0</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider></geom></class>
<class id="dev"><name>DEV</name><geom id="v"><name>ada0</name><consumer id="c"><provider ref="p"/></consumer></geom><geom id="av"><name>diskid/DISK-serial-A</name><consumer id="ac"><provider ref="ap"/></consumer></geom></class>
<class id="label"><name>LABEL</name><geom id="a"><name>ada0</name><consumer id="lc"><provider ref="p"/></consumer><provider id="ap"><name>diskid/DISK-serial-A</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider></geom></class>
</mesh>`

func TestDiskAttachmentsRecognizesDevAndExactWholeDiskAlias(t *testing.T) {
	a, err := ParseDiskAttachments([]byte(attachmentFixture), "ada0", DefaultLimits())
	if err != nil || a.Disk.Name != "ada0" || len(a.Aliases) != 1 || a.Aliases[0].Name != "diskid/DISK-serial-A" || a.Aliases[0].Kind != Logical {
		t.Fatal("native device-node/alias correspondence unavailable", err)
	}
	// Device aliases never become independent direct-disk selections.
	if _, err := ParseDiskAttachments([]byte(attachmentFixture), "diskid/DISK-serial-A", DefaultLimits()); err == nil {
		t.Fatal("alias became a separate disk")
	}
}

func TestDiskAttachmentsPermitsDevWithoutInventingAnAlias(t *testing.T) {
	g := strings.Split(attachmentFixture, "<class id=\"label\">")[0] + "</mesh>"
	g = strings.Replace(g, "<geom id=\"av\"><name>diskid/DISK-serial-A</name><consumer id=\"ac\"><provider ref=\"ap\"/></consumer></geom>", "", 1)
	a, err := ParseDiskAttachments([]byte(g), "ada0", DefaultLimits())
	if err != nil || a.Disk.Name != "ada0" || len(a.Aliases) != 0 {
		t.Fatal("DEV-only correspondence invented/refused an absent alias", err)
	}
}

func TestDiskAttachmentsRefusesPartitionAndUnknownConsumers(t *testing.T) {
	for _, class := range []string{"PART", "MULTIPATH", "ELI", "MIRROR", "UNKNOWN"} {
		g := strings.Replace(attachmentFixture, "<name>LABEL</name>", "<name>"+class+"</name>", 1)
		a, err := ParseDiskAttachments([]byte(g), "ada0", DefaultLimits())
		if !errors.Is(err, ErrDiskAttachments) || !reflect.DeepEqual(a, DiskAttachments{}) {
			t.Fatal("partition/unsupported use produced an attachment proof", err)
		}
	}
	g := strings.Replace(attachmentFixture, "</mesh>", "<class id=\"part\"><name>PART</name><geom id=\"table\"><name>ada0</name><consumer id=\"pc\"><provider ref=\"p\"/></consumer></geom></class></mesh>", 1)
	if a, err := ParseDiskAttachments([]byte(g), "ada0", DefaultLimits()); !errors.Is(err, ErrDiskAttachments) || !reflect.DeepEqual(a, DiskAttachments{}) {
		t.Fatal("empty PART table accepted", err)
	}
}

func TestDiskAttachmentsRefusesBrokenAliasAndDeviceNodeChains(t *testing.T) {
	for _, g := range []string{
		strings.Replace(attachmentFixture, "<class id=\"label\">", "<class id=\"dev2\"><name>DEV</name><geom id=\"v2\"><name>ada0</name><consumer id=\"c2\"><provider ref=\"p\"/></consumer></geom></class><class id=\"label\">", 1),
		strings.Replace(attachmentFixture, "</mesh>", "<class id=\"label2\"><name>LABEL</name><geom id=\"chained\"><name>nested</name><consumer id=\"nested\"><provider ref=\"ap\"/></consumer></geom></class></mesh>", 1),
		strings.Replace(attachmentFixture, "<geom id=\"v\"><name>ada0</name>", "<geom id=\"v\"><name>other</name>", 1),
		strings.Replace(attachmentFixture, "<geom id=\"av\"><name>diskid/DISK-serial-A</name>", "<geom id=\"av\"><name>other</name>", 1),
		strings.ReplaceAll(attachmentFixture, "diskid/DISK-serial-A", "diskid/DISK-serial-B"),
		strings.Replace(attachmentFixture, "<provider id=\"ap\"><name>diskid/DISK-serial-A</name><mediasize>67108864</mediasize>", "<provider id=\"ap\"><name>diskid/DISK-serial-A</name><mediasize>67108352</mediasize>", 1),
		strings.Replace(attachmentFixture, "<consumer id=\"lc\"><provider ref=\"p\"/></consumer>", "<consumer id=\"lc\"><provider ref=\"p\"/></consumer><consumer id=\"extra\"/>", 1),
		strings.Replace(attachmentFixture, "<consumer id=\"lc\"><provider ref=\"p\"/></consumer>", "<consumer id=\"lc\"><provider ref=\"ap\"/></consumer>", 1),
		strings.Replace(attachmentFixture, "<geom id=\"d\"><name>ada0</name>", "<geom id=\"d\"><name>ada0</name><consumer id=\"parent\"><provider ref=\"ap\"/></consumer>", 1),
		strings.Replace(attachmentFixture, "<consumer id=\"ac\"><provider ref=\"ap\"/></consumer>", "<consumer id=\"ac\"><provider ref=\"missing\"/></consumer>", 1),
		strings.Replace(attachmentFixture, "<geom id=\"v\"><name>ada0</name><consumer id=\"c\"><provider ref=\"p\"/></consumer></geom>", "", 1),
		strings.Replace(attachmentFixture, "<geom id=\"av\"><name>diskid/DISK-serial-A</name><consumer id=\"ac\"><provider ref=\"ap\"/></consumer></geom>", "", 1),
	} {
		if a, err := ParseDiskAttachments([]byte(g), "ada0", DefaultLimits()); err == nil || !reflect.DeepEqual(a, DiskAttachments{}) {
			t.Fatal("broken/ambiguous/cyclic attachment accepted")
		}
	}
}

func TestDiskAttachmentsRequiresUniqueSerialAndBoundedFullGraph(t *testing.T) {
	for _, g := range []string{strings.Replace(attachmentFixture, "<ident>serial-A</ident>", "", 1), strings.Replace(attachmentFixture, "<class id=\"dev\">", "<class id=\"second\"><name>DISK</name><geom id=\"d2\"><name>ada1</name><config><ident>serial-A</ident></config><provider id=\"p2\"><name>ada1</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider></geom></class><class id=\"dev\">", 1)} {
		if _, err := ParseDiskAttachments([]byte(g), "ada0", DefaultLimits()); err == nil {
			t.Fatal("missing/duplicate serial accepted")
		}
	}
	l := DefaultLimits()
	l.MaxBytes = len(attachmentFixture)
	if _, err := ParseDiskAttachments([]byte(attachmentFixture), "ada0", l); !errors.Is(err, ErrLimit) {
		t.Fatal("combined graph/address budget ignored", err)
	}
	l = DefaultLimits()
	l.MaxRecords = 1
	if _, err := ParseDiskAttachments([]byte(attachmentFixture), "ada0", l); !errors.Is(err, ErrLimit) {
		t.Fatal("graph bound ignored", err)
	}
	l = DefaultLimits()
	l.MaxFieldBytes = 3
	if a, err := ParseDiskAttachments([]byte(attachmentFixture), "ada0", l); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(a, DiskAttachments{}) {
		t.Fatal("address field bound ignored", err)
	}
}

func FuzzDiskAttachments(f *testing.F) {
	f.Add(attachmentFixture, "ada0")
	f.Add("<mesh/>", "ada0")
	f.Fuzz(func(t *testing.T, data, name string) {
		l := DefaultLimits()
		l.MaxBytes, l.MaxRecords, l.MaxFieldBytes = 64<<10, 1024, 1024
		a, err := ParseDiskAttachments([]byte(data), name, l)
		if err != nil {
			if !reflect.DeepEqual(a, DiskAttachments{}) {
				t.Fatal("refusal exposed a partial proof")
			}
			return
		}
		if a.Disk.Identity != IdentityUnique || a.Disk.Kind != DiskCandidate || a.Disk.Name != name {
			t.Fatal("non-disk identity was promoted")
		}
		seen := map[string]bool{}
		for _, p := range a.Aliases {
			if p.Kind != Logical || seen[p.Name] || p.MediaSize != a.Disk.MediaSize || p.SectorSize != a.Disk.SectorSize {
				t.Fatal("alias duplicated a device or changed geometry")
			}
			seen[p.Name] = true
		}
	})
}
