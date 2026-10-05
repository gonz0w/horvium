// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

var signatureGEOM = strings.Replace(systemGEOM, `<geom id="d1"><name>ada1</name>`, `<geom id="d1"><name>ada1</name><config><ident>sig-unique</ident></config>`, 1)

const signatureInfo = "/dev/ada1\t512\t67108864\t131072\t0\t0\n"
const signatureSerial = "sig-unique\n"

func TestFilesystemSignatureBindsPositiveTypeToDisk(t *testing.T) {
	for _, kind := range []string{"ufs", "zfs"} {
		got, err := ParseFilesystemSignature([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), FilesystemProbeCapture{Device: "/dev/ada1", Output: []byte(kind + "\n")}, inventory.DefaultLimits())
		if err != nil || got.Type != kind || got.Disk.Provider.Name != "ada1" || got.Disk.Provider.Identity != inventory.IdentityUnique || got.Disk.SectorCount != 131072 {
			t.Fatal("positive signature lost its native disk correspondence", err)
		}
	}
}

func TestFilesystemSignatureNeverTurnsUnknownOrFailureIntoBlank(t *testing.T) {
	for _, output := range []string{"", "\n", "none\n", "unknown\n", "ufs", "UFS\n", " ufs\n", "ufs \n", "ufs label\n", "zfs pool-name\n", "ufs\nzfs\n", "ufs\n\n", "ufs\x00\n", "ufs\r\n", "ext2fs\n", string([]byte{255, '\n'})} {
		got, err := ParseFilesystemSignature([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), FilesystemProbeCapture{Device: "/dev/ada1", Output: []byte(output)}, inventory.DefaultLimits())
		if err == nil || !reflect.DeepEqual(got, FilesystemSignature{}) {
			t.Fatal("unsupported/truncated probe exposed a blank or partial signature", err)
		}
	}
}

func TestFilesystemSignatureRefusesAddressIdentityAndGeometryDrift(t *testing.T) {
	for _, address := range []string{"", "/dev/ada2", "/dev/ada1p1", "/dev/gpt/rootfs", "/dev/../ada1", "/tmp/disk", "ada1"} {
		got, err := ParseFilesystemSignature([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), FilesystemProbeCapture{Device: address, Output: []byte("ufs\n")}, inventory.DefaultLimits())
		if err == nil || !reflect.DeepEqual(got, FilesystemSignature{}) {
			t.Fatal("wrong probe address bound a different disk", err)
		}
	}
	for _, tc := range []struct{ geom, info, serial string }{
		{signatureGEOM, signatureInfo, "other\n"},
		{signatureGEOM, strings.Replace(signatureInfo, "512", "4096", 1), signatureSerial},
		{strings.Replace(signatureGEOM, "<ident>sig-unique</ident>", "", 1), signatureInfo, signatureSerial},
		{strings.Replace(signatureGEOM, `<geom id="d2"><name>ada2</name>`, `<geom id="d2"><name>ada2</name><config><ident>sig-unique</ident></config>`, 1), signatureInfo, signatureSerial},
		{strings.Replace(signatureGEOM, "<name>DISK</name>", "<name>MD</name>", 1), signatureInfo, signatureSerial},
	} {
		got, err := ParseFilesystemSignature([]byte(tc.geom), []byte(tc.info), []byte(tc.serial), FilesystemProbeCapture{Device: "/dev/ada1", Output: []byte("zfs\n")}, inventory.DefaultLimits())
		if err == nil || !reflect.DeepEqual(got, FilesystemSignature{}) {
			t.Fatal("uncertain disk accepted a signature", err)
		}
	}
}

func TestFilesystemSignatureBoundsWholeCapture(t *testing.T) {
	probe := FilesystemProbeCapture{Device: "/dev/ada1", Output: []byte("ufs\n")}
	l := inventory.DefaultLimits()
	l.MaxBytes = len(signatureGEOM) + len(signatureInfo) + len(signatureSerial) + len(probe.Device) + len(probe.Output) - 1
	if got, err := ParseFilesystemSignature([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), probe, l); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(got, FilesystemSignature{}) {
		t.Fatal("aggregate byte bound ignored", err)
	}
	l = inventory.DefaultLimits()
	l.MaxRecords = 18
	if _, err := ParseFilesystemSignature([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), probe, l); err != nil {
		t.Fatal("exact aggregate record budget refused", err)
	}
	l.MaxRecords = 17
	if got, err := ParseFilesystemSignature([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), probe, l); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(got, FilesystemSignature{}) {
		t.Fatal("probe/serial/diskinfo records escaped graph budget", err)
	}
	for _, records := range []int{1, 2, 3} {
		l = inventory.DefaultLimits()
		l.MaxRecords = records
		if _, err := ParseFilesystemSignature([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), probe, l); !errors.Is(err, ErrLimit) {
			t.Fatal("insufficient scalar record budget accepted", err)
		}
	}
	l = inventory.DefaultLimits()
	l.MaxFieldBytes = 3
	probe.Device = strings.Repeat("x", 4)
	if _, err := ParseFilesystemSignature([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), probe, l); !errors.Is(err, ErrLimit) {
		t.Fatal("probe address field bound ignored", err)
	}
	if _, err := ParseFilesystemSignature(nil, nil, nil, FilesystemProbeCapture{}, inventory.Limits{}); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid limits accepted", err)
	}
}

func FuzzFilesystemSignature(f *testing.F) {
	f.Add(signatureGEOM, signatureInfo, signatureSerial, "/dev/ada1", "ufs\n")
	f.Add(signatureGEOM, signatureInfo, signatureSerial, "/dev/ada1", "zfs\n")
	f.Fuzz(func(t *testing.T, geom, info, serial, address, output string) {
		l := inventory.DefaultLimits()
		l.MaxBytes = 64 << 10
		l.MaxRecords = 1024
		l.MaxFieldBytes = 1024
		got, err := ParseFilesystemSignature([]byte(geom), []byte(info), []byte(serial), FilesystemProbeCapture{Device: address, Output: []byte(output)}, l)
		if err != nil {
			if !reflect.DeepEqual(got, FilesystemSignature{}) {
				t.Fatal("partial signature observation")
			}
			return
		}
		if (got.Type != "ufs" && got.Type != "zfs") || got.Disk.Provider.Kind != inventory.DiskCandidate || got.Disk.Provider.Identity != inventory.IdentityUnique || address != "/dev/"+got.Disk.Provider.Name {
			t.Fatal("unbound or unknown successful signature")
		}
	})
}
