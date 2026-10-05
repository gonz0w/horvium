// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"bytes"
	"errors"
	"math"
	"strconv"
	"strings"
)

var ErrDiskCorrelation = errors.New("disk identity or geometry does not correlate")

// DiskObservation is a read-only correspondence, never device-use, locality,
// physical-sector, enrollment or mutation authority. Zero stripe fields remain
// native zero observations; diskinfo can also emit zero when an ioctl fails.
type DiskObservation struct {
	Provider     Provider
	SectorCount  uint64
	StripeSize   uint64
	StripeOffset uint64
}

// ParseDiskObservation binds one successful diskinfo/-s capture to the entire
// successful kern.geom.confxml capture. Callers must supply complete, fresh
// native outputs, not a filtered graph or a client assertion of completeness.
// Only direct DISK providers with a unique correlated serial qualify here.
// Lunids remain observations and cannot substitute for independent serial proof.
// This function performs no I/O and is not wired into the runtime collector.
func ParseDiskObservation(geom, info, serial []byte, limits Limits) (DiskObservation, error) {
	if err := limits.Validate(); err != nil {
		return DiskObservation{}, err
	}
	remaining := limits.MaxBytes
	for _, data := range [][]byte{geom, info, serial} {
		if len(data) > remaining {
			return DiskObservation{}, ErrLimit
		}
		remaining -= len(data)
		if err := checkInput(data, limits); err != nil {
			return DiskObservation{}, err
		}
	}
	// Bound delimiters before splitting. Accept exactly the six-column native
	// projection or its three additional firmware-geometry columns, one row.
	if len(info) == 0 || info[len(info)-1] != '\n' || bytes.Count(info, []byte{'\n'}) != 1 {
		return DiskObservation{}, ErrMalformed
	}
	row := info[:len(info)-1]
	tabs := bytes.Count(row, []byte{'\t'})
	if tabs != 5 && tabs != 8 {
		return DiskObservation{}, ErrMalformed
	}
	for rest := row; ; {
		field, next, more := bytes.Cut(rest, []byte{'\t'})
		if len(field) > limits.MaxFieldBytes {
			return DiskObservation{}, ErrLimit
		}
		if !more {
			break
		}
		rest = next
	}
	fields := strings.Split(string(row), "\t")
	for _, field := range fields {
		if err := safeText(field, limits); err != nil {
			return DiskObservation{}, err
		}
	}
	name, ok := strings.CutPrefix(fields[0], "/dev/")
	if !ok || !diskAddress(name) {
		return DiskObservation{}, ErrDiskCorrelation
	}
	nums := make([]uint64, len(fields)-1)
	for i, field := range fields[1:] {
		n, err := number(field)
		if err != nil || strconv.FormatUint(n, 10) != field || n > math.MaxInt64 {
			return DiskObservation{}, ErrMalformed
		}
		nums[i] = n
	}
	if nums[0] == 0 || nums[0] > math.MaxUint32 || nums[1] == 0 || nums[1]%nums[0] != 0 || nums[2] != nums[1]/nums[0] {
		return DiskObservation{}, ErrDiskCorrelation
	}
	if len(nums) == 8 && (nums[6] == 0 || nums[7] == 0 || nums[6] > math.MaxUint32 || nums[7] > math.MaxUint32) {
		return DiskObservation{}, ErrMalformed
	}
	if len(serial) == 0 || serial[len(serial)-1] != '\n' || bytes.Count(serial, []byte{'\n'}) != 1 {
		return DiskObservation{}, ErrMalformed
	}
	if len(serial)-1 > limits.MaxFieldBytes {
		return DiskObservation{}, ErrLimit
	}
	ident := string(serial[:len(serial)-1])
	if ident == "" {
		return DiskObservation{}, ErrDiskCorrelation
	}
	if err := safeText(ident, limits); err != nil {
		return DiskObservation{}, err
	}
	if strings.TrimSpace(ident) != ident {
		return DiskObservation{}, ErrDiskCorrelation
	}
	graph, err := ParseGEOM(geom, limits)
	if err != nil {
		return DiskObservation{}, err
	}
	var selected Provider
	found := false
	for _, p := range graph.Providers {
		if p.Name != name {
			continue
		}
		if found {
			return DiskObservation{}, ErrDiskCorrelation
		}
		selected, found = p, true
	}
	if !found || selected.Kind != DiskCandidate || selected.Class != "DISK" || selected.Identity != IdentityUnique || selected.MediaSize != nums[1] || selected.SectorSize != nums[0] {
		return DiskObservation{}, ErrDiskCorrelation
	}
	serialMatched, direct := false, false
	for _, id := range selected.Identifiers {
		if id.Kind == "serial" {
			if id.Value != ident || (id.Source != "geom.config.ident" && id.Source != "provider.config.ident") {
				return DiskObservation{}, ErrDiskCorrelation
			}
			serialMatched = true
		}
	}
	for _, g := range graph.Geoms {
		if g.ID == selected.GeomID {
			direct = g.Name == selected.Name && len(g.Consumers) == 0
		}
	}
	if !serialMatched || !direct {
		return DiskObservation{}, ErrDiskCorrelation
	}
	// Incoming consumers are deliberately preserved in the source graph and not
	// interpreted as absent use. Partitions/DEV/LABEL/use checks are later gates.
	return DiskObservation{Provider: selected, SectorCount: nums[2], StripeSize: nums[3], StripeOffset: nums[4]}, nil
}

func diskAddress(s string) bool {
	if s == "" || s == "." || s == ".." || s[0] == '-' {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.') {
			return false
		}
	}
	return true
}
