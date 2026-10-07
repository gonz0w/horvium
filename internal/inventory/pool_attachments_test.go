// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"errors"
	"strings"
	"testing"
)

func poolAttachmentXML() string {
	return `<mesh><class id="d"><name>DISK</name><geom id="g"><name>ada0</name><config><ident>SERIAL-A</ident></config><provider id="p"><name>ada0</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider></geom></class><class id="dev"><name>DEV</name><geom id="dg"><name>ada0</name><consumer id="dc"><provider ref="p"/></consumer></geom></class><class id="z"><name>ZFS::VDEV</name><geom id="zg"><name>zfs::vdev</name><consumer id="zc"><provider ref="p"/></consumer></geom></class></mesh>`
}
func TestPoolAttachmentsRequireExactNativeConsumerWithoutUnusedAuthority(t *testing.T) {
	x := poolAttachmentXML()
	a, err := ParsePoolDiskAttachments([]byte(x), "ada0", DefaultLimits())
	if err != nil || a.Disk.Name != "ada0" {
		t.Fatal("native pool consumer", err)
	}
	if _, err = ParseDiskAttachments([]byte(x), "ada0", DefaultLimits()); !errors.Is(err, ErrDiskAttachments) {
		t.Fatal("unused classifier widened", err)
	}
	for _, bad := range []string{
		strings.Replace(x, "ZFS::VDEV", "ZFS", 1), strings.Replace(x, "zfs::vdev", "other", 1),
		strings.Replace(x, `<consumer id="zc"><provider ref="p"/></consumer>`, "", 1),
		strings.Replace(x, `<consumer id="zc"><provider ref="p"/></consumer>`, `<consumer id="zc"><provider ref="p"/></consumer><consumer id="zc2"><provider ref="p"/></consumer>`, 1),
		strings.Replace(x, `<geom id="zg">`, `<geom id="zg"><provider id="zp"><name>logical0</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider>`, 1),
		strings.Replace(x, "<name>DEV</name>", "<name>PART</name>", 1),
	} {
		if _, err := ParsePoolDiskAttachments([]byte(bad), "ada0", DefaultLimits()); err == nil {
			t.Fatal("uncertain attachment accepted")
		}
	}
}
func FuzzPoolDiskAttachments(f *testing.F) {
	f.Add([]byte(poolAttachmentXML()))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 32768 {
			t.Skip()
		}
		limits := DefaultLimits()
		limits.MaxBytes = 65536
		limits.MaxRecords = 1024
		limits.MaxXMLElements = 4096
		_, _ = ParsePoolDiskAttachments(data, "ada0", limits)
	})
}
