// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"math"
	"strconv"
	"strings"
)

// AccessMode is the kernel's aggregate GEOM access count, not a process/handle
// count, raw-open attribution, reservation or external-host ownership proof.
type AccessMode struct{ Read, Write, Exclusive uint32 }

// DiskAccess retains a selected direct disk's access snapshot. Zero counters do
// not establish absent signatures/partitions, enrollment, locality or clearance.
type DiskAccess struct {
	Disk DiskObservation
	Mode AccessMode
}

// ParseDiskAccess requires complete successful GEOM and independent diskinfo/-s
// captures. It checks the selected provider's canonical r#w#e# counts against
// all its immediate attached consumers. Aliases and other layers contribute via
// their consumer; this does not classify their entire topology or attribute PIDs.
// The caller establishes execution provenance/freshness; there is no I/O here.
func ParseDiskAccess(geom, info, serial []byte, limits Limits) (DiskAccess, error) {
	if err := limits.Validate(); err != nil {
		return DiskAccess{}, err
	}
	// Two bounded XML parses and separate mode records must share the budget.
	per := limits
	per.MaxXMLElements /= 2
	per.MaxRecords /= 2
	if per.MaxRecords <= 2 || per.MaxXMLElements == 0 {
		return DiskAccess{}, ErrLimit
	}
	diskLimits := per
	diskLimits.MaxRecords -= 2 // independent geometry and serial scalar records
	disk, err := ParseDiskObservation(geom, info, serial, diskLimits)
	if err != nil {
		return DiskAccess{}, err
	}
	root, err := xmlTree(geom, per)
	if err != nil {
		return DiskAccess{}, err
	}
	remaining := per.MaxRecords
	readMode := func(n *node) (AccessMode, error) {
		if remaining == 0 {
			return AccessMode{}, ErrLimit
		}
		remaining--
		child, err := n.one("mode", true)
		if err != nil || len(child.children) != 0 || child.id != "" || child.ref != "" {
			return AccessMode{}, ErrMalformed
		}
		return parseAccessMode(child.text)
	}
	var provider *node
	var consumers []*node
	for _, class := range root.childrenNamed("class") {
		for _, g := range class.childrenNamed("geom") {
			for _, p := range g.childrenNamed("provider") {
				if p.id == disk.Provider.ID {
					provider = p
				}
			}
			for _, c := range g.childrenNamed("consumer") {
				p, err := c.one("provider", false)
				if err != nil {
					return DiskAccess{}, err
				}
				if p != nil && p.ref == disk.Provider.ID {
					consumers = append(consumers, c)
				}
			}
		}
	}
	if provider == nil {
		return DiskAccess{}, ErrMalformed
	}
	mode, err := readMode(provider)
	if err != nil {
		return DiskAccess{}, err
	}
	var sum AccessMode
	for _, c := range consumers {
		m, err := readMode(c)
		if err != nil {
			return DiskAccess{}, err
		}
		// Counts are signed native int values. Reject contradictory/overflowing sums.
		if uint64(sum.Read)+uint64(m.Read) > math.MaxInt32 || uint64(sum.Write)+uint64(m.Write) > math.MaxInt32 || uint64(sum.Exclusive)+uint64(m.Exclusive) > math.MaxInt32 {
			return DiskAccess{}, ErrMalformed
		}
		sum.Read += m.Read
		sum.Write += m.Write
		sum.Exclusive += m.Exclusive
	}
	if sum != mode {
		return DiskAccess{}, ErrMalformed
	}
	return DiskAccess{Disk: disk, Mode: mode}, nil
}

func parseAccessMode(text string) (AccessMode, error) {
	rest, ok := strings.CutPrefix(text, "r")
	if !ok {
		return AccessMode{}, ErrMalformed
	}
	r, rest, ok := strings.Cut(rest, "w")
	if !ok {
		return AccessMode{}, ErrMalformed
	}
	w, e, ok := strings.Cut(rest, "e")
	if !ok {
		return AccessMode{}, ErrMalformed
	}
	fields := []string{r, w, e}
	var nums [3]uint32
	for i, s := range fields {
		n, err := strconv.ParseUint(s, 10, 31)
		if err != nil || strconv.FormatUint(n, 10) != s {
			return AccessMode{}, ErrMalformed
		}
		nums[i] = uint32(n)
	}
	return AccessMode{Read: nums[0], Write: nums[1], Exclusive: nums[2]}, nil
}
