// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// InspectOwnership validates the complete bounded ownership-only namespace
// through the held lease. It never deletes, repairs or publishes files, resets
// uncertainty, enrolls a resource or authorizes storage work. Unknown/damaged
// entries and orphan temporaries return no usable or partial catalog. Future
// store recovery and runtime admission remain separate gates.
func (s *StateDirectory) InspectOwnership() (OwnershipCatalog, error) {
	return s.inspectOwnership(maxRecords, maxOwnershipCatalogBytes, nil)
}

// Lower fixture bounds and afterRead permit deterministic native budget/drift
// tests only. There is no runtime/request/environment failpoint or tuning API.
func (s *StateDirectory) inspectOwnership(limit int, byteLimit uint64, afterRead func()) (OwnershipCatalog, error) {
	if s == nil || os.Geteuid() != 0 {
		return OwnershipCatalog{}, ErrStateDirectory
	}
	if limit < 1 || limit > maxRecords || byteLimit == 0 || byteLimit > maxOwnershipCatalogBytes {
		return OwnershipCatalog{}, ErrLimit
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inspectOwnershipLocked(limit, byteLimit, afterRead, "")
}

// Caller holds mu. Publication alone may exclude its exact inode-validated
// temporary name; all other temporaries remain uncertain. Public inspection
// never supplies an exclusion.
func (s *StateDirectory) inspectOwnershipLocked(limit int, byteLimit uint64, afterRead func(), ownTemporary string) (OwnershipCatalog, error) {
	dir, err := s.ownershipDirectoryLocked()
	if err != nil {
		return OwnershipCatalog{}, err
	}
	var before unix.Stat_t
	if unix.Fstat(dir, &before) != nil {
		return OwnershipCatalog{}, ErrStateDirectory
	}
	// A fresh descriptor has its own enumeration offset. Dup would share the
	// held descriptor's offset and could make later inspection omit entries.
	fd, err := unix.Openat(dir, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return OwnershipCatalog{}, ErrStateDirectory
	}
	scan := os.NewFile(uintptr(fd), "storage-ownership-catalog")
	defer scan.Close()
	var opened unix.Stat_t
	if unix.Fstat(fd, &opened) != nil || !sameOwnershipMetadata(before, opened) {
		return OwnershipCatalog{}, ErrDrift
	}
	var names []string
	nameLimit := limit + 1
	if ownTemporary != "" {
		nameLimit++
	}
	for {
		batch, e := scan.Readdirnames(64)
		if len(names)+len(batch) > nameLimit {
			return OwnershipCatalog{}, ErrLimit
		}
		names = append(names, batch...)
		if e == io.EOF {
			break
		}
		if e != nil {
			return OwnershipCatalog{}, ErrStateDirectory
		}
	}
	if scan.Close() != nil {
		return OwnershipCatalog{}, ErrStateDirectory
	}
	if ownTemporary != "" {
		found := false
		filtered := make([]string, 0, len(names))
		for _, name := range names {
			if name == ownTemporary {
				if found {
					return OwnershipCatalog{}, ErrDrift
				}
				found = true
			} else {
				filtered = append(filtered, name)
			}
		}
		if !found {
			return OwnershipCatalog{}, ErrDrift
		}
		names = filtered
	}
	guids, err := ownershipCatalogNames(names, limit)
	if err != nil {
		return OwnershipCatalog{}, err
	}
	result := OwnershipCatalog{Records: make([]OwnershipSummary, 0, len(guids))}
	metadata := make([]unix.Stat_t, 0, len(guids))
	for _, guid := range guids {
		name := "ownership-" + guid + ".json"
		var prior, after unix.Stat_t
		if unix.Fstatat(dir, name, &prior, unix.AT_SYMLINK_NOFOLLOW) != nil || !privateOwnershipFile(prior) {
			return OwnershipCatalog{}, ErrStateDirectory
		}
		size := uint64(prior.Size)
		if size > maxStateRecordBytes || size > byteLimit-result.Bytes {
			return OwnershipCatalog{}, ErrLimit
		}
		policy, e := s.readOwnershipLocked(guid)
		if e != nil {
			return OwnershipCatalog{}, e
		}
		if unix.Fstatat(dir, name, &after, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(prior, after) {
			return OwnershipCatalog{}, ErrDrift
		}
		result.Records = append(result.Records, OwnershipSummary{PoolGUID: guid, Revision: policy.Revision, Bytes: size})
		result.Bytes += size
		metadata = append(metadata, after)
	}
	if afterRead != nil {
		afterRead()
	}
	// Recheck every named record after collection, as well as the anchored
	// directory/lock and namespace metadata. No external-root filesystem fence
	// or cross-record atomic transaction is claimed by these drift observations.
	for i, guid := range guids {
		var current unix.Stat_t
		if unix.Fstatat(dir, "ownership-"+guid+".json", &current, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(metadata[i], current) {
			return OwnershipCatalog{}, ErrDrift
		}
	}
	if _, err = s.ownershipDirectoryLocked(); err != nil {
		return OwnershipCatalog{}, err
	}
	var after unix.Stat_t
	if unix.Fstat(dir, &after) != nil || !sameOwnershipMetadata(before, after) {
		return OwnershipCatalog{}, ErrDrift
	}
	return result, nil
}
