// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// ReadOwnership reads only a GUID-selected record through the held directory.
// It produces metadata, not enrollment/admission authority. Descriptor ACL checks
// refuse unsafe effective grants; provisioning and runtime store
// wiring remain open. Missing/unsafe/damaged files never become empty enrollment.
func (s *StateDirectory) ReadOwnership(poolGUID string) (Policy, error) {
	if s == nil || os.Geteuid() != 0 {
		return Policy{}, ErrStateDirectory
	}
	if len(poolGUID) > 20 || !validGUID(poolGUID) {
		return Policy{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readOwnershipLocked(poolGUID)
}

// Caller holds s.mu and has checked the kernel UID and selector.
func (s *StateDirectory) readOwnershipLocked(poolGUID string) (Policy, error) {
	data, err := s.readStateBytesLocked("ownership-" + poolGUID + ".json")
	if err != nil {
		return Policy{}, err
	}
	p, err := DecodeOwnershipRecord(data)
	if err != nil {
		return Policy{}, err
	}
	if p.PoolGUID != poolGUID {
		return Policy{}, ErrDrift
	}
	return p, nil
}

// Caller holds mu and has validated the kernel UID and typed selector. Only
// ownership/job wrappers and the private fixed journal construct names; requests cannot supply a file path.
// Shared descriptor validation retains the accepted reader's native semantics.
func (s *StateDirectory) readStateBytesLocked(name string) ([]byte, error) {
	dir, err := s.ownershipDirectoryLocked()
	if err != nil {
		return nil, err
	}
	var prior unix.Stat_t
	if unix.Fstatat(dir, name, &prior, unix.AT_SYMLINK_NOFOLLOW) != nil || !privateOwnershipFile(prior) {
		return nil, ErrStateDirectory
	}
	if prior.Size > maxStateRecordBytes {
		return nil, ErrLimit
	}
	fd, err := unix.Openat(dir, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, ErrStateDirectory
	}
	f := os.NewFile(uintptr(fd), "storage-state-record")
	defer f.Close()
	var before, after, current unix.Stat_t
	if unix.Fstat(fd, &before) != nil || !privateOwnershipFile(before) || !safeStateACL(fd, uint32(before.Mode), false) {
		return nil, ErrStateDirectory
	}
	if before.Size > maxStateRecordBytes {
		return nil, ErrLimit
	}
	if !sameOwnershipMetadata(prior, before) {
		return nil, ErrDrift
	}
	data, err := io.ReadAll(io.LimitReader(f, maxStateRecordBytes+1))
	if err != nil {
		return nil, ErrStateDirectory
	}
	if len(data) > maxStateRecordBytes {
		return nil, ErrLimit
	}
	if int64(len(data)) != before.Size || unix.Fstat(fd, &after) != nil || !sameOwnershipMetadata(before, after) || unix.Fstatat(dir, name, &current, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(after, current) {
		return nil, ErrDrift
	}
	if f.Close() != nil {
		return nil, ErrStateDirectory
	}
	return data, nil
}

func privateOwnershipFile(st unix.Stat_t) bool {
	return st.Mode&unix.S_IFMT == unix.S_IFREG && st.Uid == 0 && st.Mode&07777 == 0600 && st.Nlink == 1 && st.Size >= 0
}

// Reading may advance atime; identity, ownership, content/change timestamps,
// link count, flags and generation must remain unchanged. These observations
// detect ordinary drift, not malicious root modification or a filesystem fence.
func sameOwnershipMetadata(a, b unix.Stat_t) bool {
	return a.Dev == b.Dev && a.Ino == b.Ino && a.Mode == b.Mode && a.Uid == b.Uid && a.Gid == b.Gid && a.Nlink == b.Nlink && a.Size == b.Size && a.Mtim == b.Mtim && a.Ctim == b.Ctim && a.Btim == b.Btim && a.Flags == b.Flags && a.Gen == b.Gen
}

// Caller holds s.mu. A held descriptor is revalidated before every operation.
func (s *StateDirectory) ownershipDirectoryLocked() (int, error) {
	if s.directory == nil || s.writer == nil {
		return -1, ErrStateDirectory
	}
	dir, writer := int(s.directory.Fd()), int(s.writer.Fd())
	var dst, wst, named unix.Stat_t
	if unix.Fstat(dir, &dst) != nil || dst.Mode&unix.S_IFMT != unix.S_IFDIR || dst.Uid != 0 || dst.Mode&07777 != 0700 || unix.Fstat(writer, &wst) != nil || !validStateLock(wst) || unix.Fstatat(dir, stateWriterName, &named, unix.AT_SYMLINK_NOFOLLOW) != nil || !validStateLock(named) || named.Dev != wst.Dev || named.Ino != wst.Ino {
		return -1, ErrStateDirectory
	}
	if !safeStateACL(dir, uint32(dst.Mode), true) || !safeStateACL(writer, uint32(wst.Mode), false) {
		return -1, ErrStateDirectory
	}
	return dir, nil
}
