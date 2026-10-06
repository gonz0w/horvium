// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// PublishOwnership publishes bounded metadata only, never enrollment or storage
// authority. expectedRevision zero requires an absent record and initial revision
// one; replacement requires the exact previous revision and its successor. No
// deletion, repair, cross-record transaction or automatic retry is provided.
// Full store budgets/recovery and runtime enrollment remain separate gates.
func (s *StateDirectory) PublishOwnership(p Policy, expectedRevision uint64) error {
	return s.publishOwnership(p, expectedRevision, nil)
}

// wrap is a package-private deterministic test seam, never configurable by the
// operator/request/runtime. Production uses the actual transaction unchanged.
func (s *StateDirectory) publishOwnership(p Policy, expected uint64, wrap func(ownershipPublication) ownershipPublication) error {
	if s == nil || os.Geteuid() != 0 {
		return ErrStateDirectory
	}
	data, err := EncodeOwnershipRecord(p)
	if err != nil {
		return err
	}
	if !validOwnershipPublicationRevision(p.Revision, expected) {
		return ErrDrift
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.publicationBlocked {
		return ErrStateUncertain
	}
	dir, err := s.ownershipDirectoryLocked()
	if err != nil {
		return err
	}
	var fs unix.Statfs_t
	if unix.Fstatfs(dir, &fs) != nil || fs.Flags&unix.MNT_RDONLY != 0 {
		return ErrStateDirectory
	}
	kind := unix.ByteSliceToString(fs.Fstypename[:])
	if kind != "ufs" && kind != "zfs" {
		return ErrStateDirectory
	}
	name := "ownership-" + p.PoolGUID + ".json"
	var prior unix.Stat_t
	exists := true
	if err = unix.Fstatat(dir, name, &prior, unix.AT_SYMLINK_NOFOLLOW); err == unix.ENOENT {
		exists = false
	} else if err != nil {
		return ErrStateDirectory
	}
	if exists {
		current, e := s.readOwnershipLocked(p.PoolGUID)
		if e != nil {
			return e
		}
		var after unix.Stat_t
		if unix.Fstatat(dir, name, &after, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(prior, after) || expected == 0 || current.Revision != expected {
			return ErrDrift
		}
	} else if expected != 0 {
		return ErrDrift
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return ErrStateDirectory
	}
	temporary := ".ownership-" + hex.EncodeToString(nonce[:]) + ".tmp"
	fd, err := unix.Openat(dir, temporary, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if err != nil {
		return ErrStateDirectory
	}
	f := os.NewFile(uintptr(fd), "storage-ownership-temporary")
	closed := false
	closeFile := func() error {
		if closed {
			return nil
		}
		closed = true
		return f.Close()
	}
	var created unix.Stat_t
	// Record the actual exclusively created inode before any write. If even this
	// identity cannot be obtained, retain the unknown name for later reconciliation.
	if unix.Fstat(fd, &created) != nil {
		_ = closeFile()
		s.publicationBlocked = true
		return ErrStateUncertain
	}
	cleanup := func() error {
		e := closeFile()
		var current unix.Stat_t
		if unix.Fstatat(dir, temporary, &current, unix.AT_SYMLINK_NOFOLLOW) != nil || current.Dev != created.Dev || current.Ino != created.Ino || current.Mode&unix.S_IFMT != unix.S_IFREG || current.Nlink != 1 || unix.Unlinkat(dir, temporary, 0) != nil || unix.Fsync(dir) != nil {
			return ErrStateUncertain
		}
		return e
	}
	tx := ownershipPublication{
		write: func() error {
			if !privateOwnershipFile(created) || created.Size != 0 || !safeStateACL(fd, uint32(created.Mode), false) {
				return ErrStateDirectory
			}
			n, e := f.Write(data)
			if e != nil {
				return e
			}
			if n != len(data) {
				return io.ErrShortWrite
			}
			return nil
		},
		syncFile: f.Sync, closeFile: closeFile, cleanup: cleanup,
	}
	// Snapshot the written inode while it is still open; then compare its anchored
	// name after close, alongside the original destination and held lease.
	var written unix.Stat_t
	syncFile := tx.syncFile
	tx.syncFile = func() error {
		if unix.Fstat(fd, &written) != nil || !privateOwnershipFile(written) || written.Size != int64(len(data)) || written.Dev != created.Dev || written.Ino != created.Ino || !safeStateACL(fd, uint32(written.Mode), false) {
			return ErrDrift
		}
		return syncFile()
	}
	tx.verify = func() error {
		if _, e := s.ownershipDirectoryLocked(); e != nil {
			return e
		}
		var temporaryStat, current unix.Stat_t
		if unix.Fstatat(dir, temporary, &temporaryStat, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(written, temporaryStat) {
			return ErrDrift
		}
		e := unix.Fstatat(dir, name, &current, unix.AT_SYMLINK_NOFOLLOW)
		if exists {
			if e != nil || !sameOwnershipMetadata(prior, current) {
				return ErrDrift
			}
		} else if e != unix.ENOENT {
			return ErrDrift
		}
		return nil
	}
	tx.rename = func() error { return unix.Renameat(dir, temporary, dir, name) }
	tx.syncDirectory = func() error { return unix.Fsync(dir) }
	if wrap != nil {
		tx = wrap(tx)
	}
	err = commitOwnershipPublication(tx)
	if errors.Is(err, ErrStateUncertain) {
		s.publicationBlocked = true
	}
	return err
}
