// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"testing"
	"unsafe"
)

func testModeACL(kind int, mode uint32) stateACL {
	a := stateACL{Max: 254, Count: 3}
	if kind == 2 {
		for i, tag := range []uint32{1, 4, 32} {
			a.Entries[i] = stateACLEntry{Tag: tag, ID: ^uint32(0), Perm: (mode >> uint(6-i*3)) & 7}
		}
	} else {
		for i, tag := range []uint32{1, 4, 64} {
			p := uint32(0x40 | 0x200 | 0x1000 | 0x8000)
			if i == 0 {
				p |= 0x80 | 0x100 | 0x400 | 0x800 | 0x2000 | 0x4000
			}
			bits := (mode >> uint(6-i*3)) & 7
			if bits&4 != 0 {
				p |= 8
			}
			if bits&2 != 0 {
				p |= 0x10 | 0x20
			}
			if bits&1 != 0 {
				p |= 1
			}
			a.Entries[i] = stateACLEntry{Tag: tag, ID: ^uint32(0), Perm: p, Type: 0x100}
		}
	}
	return a
}

func TestStateACLPolicy(t *testing.T) {
	// Independent literal ABI cases distinguish read-named (0x40) from
	// write-named (0x80); a mode-generated fixture must not hide a shared typo.
	attribute := stateACL{Max: 254, Count: 3}
	for i, tag := range []uint32{1, 4, 64} {
		attribute.Entries[i] = stateACLEntry{Tag: tag, ID: ^uint32(0), Type: 0x100, Perm: 0x200 | 0x1000 | 0x8000}
	}
	readAttribute := attribute
	readAttribute.Entries[1].Perm |= 0x40
	if !safeStateACLValue(readAttribute, 4, 0700, false) {
		t.Error("ordinary nonowner read-named attribute right refused")
	}
	writeAttribute := attribute
	writeAttribute.Entries[1].Perm |= 0x80
	if safeStateACLValue(writeAttribute, 4, 0700, false) {
		t.Error("nonowner write-named attribute right accepted")
	}
	if unsafe.Sizeof(stateACLEntry{}) != 16 || unsafe.Sizeof(stateACL{}) != 4088 || unsafe.Offsetof(stateACL{}.Entries) != 24 {
		t.Fatal("FreeBSD ACL ABI layout mismatch")
	}
	for _, mode := range []uint32{0600, 0700, 0755, 0711, 0740} {
		for _, kind := range []int{2, 4} {
			a := testModeACL(kind, mode)
			if !safeStateACLValue(a, kind, mode, false) {
				t.Fatalf("mode-equivalent ACL refused: %o/%d", mode, kind)
			}
		}
	}
	for _, kind := range []int{2, 4} {
		good := testModeACL(kind, 0700)
		changes := []func(*stateACL){
			func(a *stateACL) { a.Max = 255 }, func(a *stateACL) { a.Count = 255 }, func(a *stateACL) { a.Count = 0 },
			func(a *stateACL) { a.Spare[0] = 1 }, func(a *stateACL) { a.Entries[1].Tag = 2; a.Entries[1].ID = 65534 },
			func(a *stateACL) { a.Entries[2].Tag = 8; a.Entries[2].ID = 65534 },
			func(a *stateACL) { a.Entries[0].ID = 0 }, func(a *stateACL) { a.Entries[0].Flags = 1 },
			func(a *stateACL) { a.Entries[1].Perm |= 0x400 }, func(a *stateACL) { a.Entries[2].Perm |= 0x10 },
			func(a *stateACL) { a.Entries[2].Type = 0x400 }, func(a *stateACL) { a.Entries[1] = a.Entries[0] },
		}
		for i, change := range changes {
			a := good
			change(&a)
			if safeStateACLValue(a, kind, 0700, false) {
				t.Fatalf("unsafe ACL accepted: %d/%d", kind, i)
			}
		}
	}
	a := testModeACL(2, 0700)
	a.Count = 4
	a.Entries[3] = stateACLEntry{Tag: 16, ID: ^uint32(0)}
	if safeStateACLValue(a, 2, 0700, false) {
		t.Fatal("POSIX mask accepted")
	}
	a = stateACL{Max: 254}
	if !safeStateACLValue(a, 3, 0700, true) || safeStateACLValue(a, 3, 0700, false) {
		t.Fatal("default ACL context confused")
	}
	a = testModeACL(2, 0700)
	if safeStateACLValue(a, 3, 0700, true) {
		t.Fatal("inheritable default accepted")
	}
	a = testModeACL(4, 0700)
	a.Count = 4
	a.Entries[3] = stateACLEntry{Tag: 64, ID: ^uint32(0), Type: 0x200, Perm: 0x10}
	if !safeStateACLValue(a, 4, 0700, false) {
		t.Fatal("additional restriction refused")
	}
	a.Entries[3].Flags = 0x80
	if safeStateACLValue(a, 4, 0700, false) {
		t.Fatal("inherited ACL accepted")
	}
	if safeStateACLValue(testModeACL(4, 0700), 99, 0700, false) {
		t.Fatal("unknown ACL type accepted")
	}
	for _, unknown := range []uint32{2, 4, 1 << 16} {
		a = testModeACL(4, 0700)
		a.Entries[0].Perm |= unknown
		if safeStateACLValue(a, 4, 0700, false) {
			t.Fatal("unknown NFSv4 permission accepted")
		}
	}
}
