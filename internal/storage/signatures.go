// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"

	"github.com/gonz0w/horvium/internal/inventory"
)

// FilesystemProbeCapture records the exact validated native argument and complete
// successful stdout from fstyp -u, without -l or -s. Stdout contains no address;
// the caller must establish execution success, freshness and capture provenance.
type FilesystemProbeCapture struct {
	Device string
	Output []byte
}

// FilesystemSignature is positive probe recognition on a correlated direct disk.
// It is not filesystem/label integrity, pool identity, absence or write clearance.
type FilesystemSignature struct {
	Disk inventory.DiskObservation
	Type string
}

// ParseFilesystemSignature accepts only the qualified scalar UFS/ZFS formats.
// Unknown, empty or unsupported output never produces a blank/eligible disk.
// It performs no I/O and introduces no collector or mutation authority.
func ParseFilesystemSignature(geom, info, serial []byte, probe FilesystemProbeCapture, limits inventory.Limits) (FilesystemSignature, error) {
	zero := FilesystemSignature{}
	if limits.Validate() != nil {
		return zero, ErrInvalid
	}
	remaining := limits.MaxBytes
	for _, size := range []int{len(geom), len(info), len(serial), len(probe.Device), len(probe.Output)} {
		if size > remaining {
			return zero, ErrLimit
		}
		remaining -= size
	}
	if err := systemField(probe.Device, limits); err != nil {
		return zero, err
	}
	if !devicePath(probe.Device) {
		return zero, ErrInvalid
	}
	if len(probe.Output) > 0 && len(probe.Output)-1 > limits.MaxFieldBytes {
		return zero, ErrLimit
	}
	kind := ""
	switch string(probe.Output) {
	case "ufs\n":
		kind = "ufs"
	case "zfs\n":
		kind = "zfs"
	default:
		return zero, ErrUnavailable
	}
	// Three fixed row/scalar captures share the graph ID-entry budget. The probe
	// address is a field of its one capture, not an independently admitted record.
	if limits.MaxRecords <= 3 {
		return zero, ErrLimit
	}
	observationLimits := limits
	observationLimits.MaxRecords -= 3
	disk, err := inventory.ParseDiskObservation(geom, info, serial, observationLimits)
	if err != nil {
		switch {
		case errors.Is(err, inventory.ErrLimit):
			return zero, ErrLimit
		case errors.Is(err, inventory.ErrDiskCorrelation):
			return zero, ErrDrift
		default:
			return zero, ErrInvalid
		}
	}
	if probe.Device != "/dev/"+disk.Provider.Name {
		return zero, ErrDrift
	}
	return FilesystemSignature{Disk: disk, Type: kind}, nil
}
