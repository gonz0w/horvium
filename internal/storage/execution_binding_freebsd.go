// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"os"

	"golang.org/x/sys/unix"
)

// Actual protected object observation only. The live mount view is retained for
// in-call drift checks, never encoded as a persistent filesystem identity.
type executionBindingObjects struct {
	directory, lock executionObjectIdentity
	mount           propertyStateBacking
}

func (s *StateDirectory) inspectExecutionBindingObjects(afterRead func()) (executionBindingObjects, error) {
	if s == nil || os.Geteuid() != 0 {
		return executionBindingObjects{}, ErrStateDirectory
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.propertyOrigin != s {
		return executionBindingObjects{}, ErrDenied
	}
	dir, err := s.ownershipDirectoryLocked()
	if err != nil {
		return executionBindingObjects{}, err
	}
	writer := int(s.writer.Fd())
	var beforeDir, beforeLock, named unix.Stat_t
	if unix.Fstat(dir, &beforeDir) != nil || unix.Fstat(writer, &beforeLock) != nil || unix.Fstatat(dir, stateWriterName, &named, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(beforeLock, named) || beforeDir.Dev != beforeLock.Dev {
		return executionBindingObjects{}, ErrDrift
	}
	mount, err := propertyStateFilesystem(dir)
	if err != nil {
		return executionBindingObjects{}, err
	}
	if mount.kind != "ufs" && mount.kind != "zfs" || mount.readonly || mount.softupdates {
		return executionBindingObjects{}, ErrStateDirectory
	}
	objects := executionBindingObjects{executionStatIdentity(beforeDir), executionStatIdentity(beforeLock), mount}
	if !objects.directory.valid() || !objects.lock.valid() || beforeDir.Btim.Nsec < 0 || beforeDir.Btim.Nsec >= 1000000000 || beforeLock.Btim.Nsec < 0 || beforeLock.Btim.Nsec >= 1000000000 {
		return executionBindingObjects{}, ErrUnavailable
	}
	if afterRead != nil {
		afterRead()
	}
	// Full current metadata/ACL/mount checks surround observation. Mutable
	// timestamps detect in-call drift but never enter persistent object hashes.
	if _, err := s.ownershipDirectoryLocked(); err != nil {
		return executionBindingObjects{}, err
	}
	var afterDir, afterLock unix.Stat_t
	if unix.Fstat(dir, &afterDir) != nil || unix.Fstat(writer, &afterLock) != nil || unix.Fstatat(dir, stateWriterName, &named, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(beforeDir, afterDir) || !sameOwnershipMetadata(beforeLock, afterLock) || !sameOwnershipMetadata(afterLock, named) {
		return executionBindingObjects{}, ErrDrift
	}
	afterMount, err := propertyStateFilesystem(dir)
	if err != nil || afterMount != mount {
		return executionBindingObjects{}, ErrDrift
	}
	return objects, nil
}

func executionStatIdentity(st unix.Stat_t) executionObjectIdentity {
	return executionObjectIdentity{st.Ino, st.Gen, st.Btim.Sec, uint32(st.Btim.Nsec)}
}
