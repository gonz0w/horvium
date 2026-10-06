// SPDX-License-Identifier: BSD-3-Clause
package storage

// Fixed-width FreeBSD struct acl/acl_entry layout from sys/sys/acl.h and
// sys/sys/_types.h. This bounded kernel ABI is not an operator record schema.
type stateACLEntry struct {
	Tag, ID, Perm uint32
	Type, Flags   uint16
}

type stateACL struct {
	Max, Count uint32
	Spare      [4]int32
	Entries    [254]stateACLEntry
}

// Require no named principals or inheritance. POSIX access ACLs must be exactly
// the three mode entries; default ACLs must be empty. NFSv4 allows only owner,
// group and everyone entries. Nonowner grants are bounded by mode and FreeBSD's
// standard ancillary read permissions; root-owner entries may use known rights.
// Additional deny entries can only restrict access.
// This is a conservative allowlist, not general ACL equivalence/normalization.
func safeStateACLValue(a stateACL, kind int, mode uint32, directory bool) bool {
	if a.Max != 254 || a.Count > 254 || a.Spare != [4]int32{} || mode&0022 != 0 {
		return false
	}
	if kind == 3 {
		return directory && a.Count == 0
	}
	if kind == 2 {
		if a.Count != 3 {
			return false
		}
		for i, tag := range []uint32{1, 4, 32} {
			e := a.Entries[i]
			if e.Tag != tag || e.ID != ^uint32(0) || e.Type != 0 || e.Flags != 0 || e.Perm != (mode>>uint(6-i*3))&7 {
				return false
			}
		}
		return true
	}
	if kind != 4 || a.Count < 3 {
		return false
	}
	var seen uint32
	for _, e := range a.Entries[:a.Count] {
		if e.ID != ^uint32(0) || e.Flags != 0 || e.Perm&^uint32(0xfff9) != 0 || e.Type != 0x100 && e.Type != 0x200 {
			return false
		}
		var bits uint32
		switch e.Tag {
		case 1:
			bits = (mode >> 6) & 7
		case 4:
			bits = (mode >> 3) & 7
		case 64:
			bits = mode & 7
		default:
			return false
		}
		if e.Type == 0x200 {
			continue
		}
		seen |= e.Tag
		allowed := uint32(0x80 | 0x200 | 0x1000 | 0x8000) // read named attrs, status, ACL, synchronize
		if e.Tag == 1 {
			allowed = 0xfff9
		} else {
			if bits&4 != 0 {
				allowed |= 8
			}
			if bits&1 != 0 {
				allowed |= 1
			}
		}
		if e.Perm&^allowed != 0 {
			return false
		}
	}
	return seen == 1|4|64
}
