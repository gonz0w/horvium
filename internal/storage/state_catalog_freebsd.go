// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// InspectState validates the complete bounded mixed ownership/job namespace
// through the held lease. Every error returns a zero catalog. It never repairs,
// evicts, publishes, resets uncertainty, admits jobs or authorizes native work.
func (s *StateDirectory) InspectState() (StateCatalog, error) {
	return s.inspectState(defaultStateCatalogLimits, nil)
}

// Reduced bounds and the afterRead seam exist only for deterministic owned
// fixtures. There is no runtime/request/environment limit or failpoint API.
func (s *StateDirectory) inspectState(limits stateCatalogLimits, afterRead func()) (StateCatalog, error) {
	if s == nil || os.Geteuid() != 0 {
		return StateCatalog{}, ErrStateDirectory
	}
	if !limits.valid() {
		return StateCatalog{}, ErrLimit
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inspectStateLocked(limits, afterRead, "")
}

// Caller holds mu; only publication may exclude its exact inode-validated temp.
func (s *StateDirectory) inspectStateLocked(limits stateCatalogLimits, afterRead func(), ownTemporary string) (StateCatalog, error) {
	return s.inspectStateRecordsLocked(limits, afterRead, ownTemporary, nil)
}

// Consume only independently validated complete bytes, while retaining all
// existing final object/namespace checks. A consumer error discards the census.
// The reservation inspector alone uses consume; publication keeps it nil.
func (s *StateDirectory) inspectStateRecordsLocked(limits stateCatalogLimits, afterRead func(), ownTemporary string, consume func(stateRecordName, []byte) error) (StateCatalog, error) {
	dir, err := s.ownershipDirectoryLocked()
	if err != nil {
		return StateCatalog{}, err
	}
	var before unix.Stat_t
	if unix.Fstat(dir, &before) != nil {
		return StateCatalog{}, ErrStateDirectory
	}
	// An independent descriptor avoids sharing directory enumeration offsets.
	fd, err := unix.Openat(dir, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return StateCatalog{}, ErrStateDirectory
	}
	scan := os.NewFile(uintptr(fd), "storage-state-catalog")
	defer scan.Close()
	var opened unix.Stat_t
	if unix.Fstat(fd, &opened) != nil || !sameOwnershipMetadata(before, opened) {
		return StateCatalog{}, fmt.Errorf("state census directory open: %w", ErrDrift)
	}
	var names []string
	nameLimit := limits.ownership + limits.nonterminal + limits.terminal + 1
	if ownTemporary != "" {
		nameLimit++
	}
	for {
		batch, e := scan.Readdirnames(64)
		if len(names)+len(batch) > nameLimit {
			return StateCatalog{}, ErrLimit
		}
		names = append(names, batch...)
		if e == io.EOF {
			break
		}
		if e != nil {
			return StateCatalog{}, ErrStateDirectory
		}
	}
	if scan.Close() != nil {
		return StateCatalog{}, ErrStateDirectory
	}
	if ownTemporary != "" {
		found := false
		filtered := make([]string, 0, len(names))
		for _, name := range names {
			if name == ownTemporary {
				if found {
					return StateCatalog{}, fmt.Errorf("state census metadata: %w", ErrDrift)
				}
				found = true
			} else {
				filtered = append(filtered, name)
			}
		}
		if !found {
			return StateCatalog{}, fmt.Errorf("state census metadata: %w", ErrDrift)
		}
		names = filtered
	}
	entries, err := stateCatalogNames(names, limits)
	if err != nil {
		return StateCatalog{}, err
	}
	b := newStateCatalogBuilder(limits)
	metadata := make([]unix.Stat_t, 0, len(entries))
	for _, entry := range entries {
		var prior, after unix.Stat_t
		if unix.Fstatat(dir, entry.name, &prior, unix.AT_SYMLINK_NOFOLLOW) != nil || !privateOwnershipFile(prior) {
			return StateCatalog{}, ErrStateDirectory
		}
		size := uint64(prior.Size)
		if size > maxStateRecordBytes || size > limits.bytes-b.catalog.Bytes {
			return StateCatalog{}, ErrLimit
		}
		data, e := s.readStateBytesLocked(entry.name)
		if e != nil {
			return StateCatalog{}, e
		}
		if e = b.add(entry, data); e != nil {
			return StateCatalog{}, e
		}
		if unix.Fstatat(dir, entry.name, &after, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(prior, after) {
			return StateCatalog{}, fmt.Errorf("state census record read: %w", ErrDrift)
		}
		metadata = append(metadata, after)
		if consume != nil {
			if e = consume(entry, data); e != nil {
				return StateCatalog{}, e
			}
		}
	}
	if afterRead != nil {
		afterRead()
	}
	for i, entry := range entries {
		var current unix.Stat_t
		if unix.Fstatat(dir, entry.name, &current, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(metadata[i], current) {
			return StateCatalog{}, fmt.Errorf("state census record final: %w", ErrDrift)
		}
	}
	if _, err = s.ownershipDirectoryLocked(); err != nil {
		return StateCatalog{}, err
	}
	var after unix.Stat_t
	if unix.Fstat(dir, &after) != nil || !sameOwnershipMetadata(before, after) {
		return StateCatalog{}, fmt.Errorf("state census namespace: %w", ErrDrift)
	}
	return b.catalog, nil
}
