// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// Caller holds mu and supplies a typed canonical destination, encoded record,
// fresh-census capacity guard and current-predecessor validator. Both publishers
// use this actual descriptor-relative sync/rename pipeline without shell work.
func (s *StateDirectory) publishStateLocked(name, prefix string, data []byte, expected uint64, limits stateCatalogLimits, validate func(StateCatalog) error, predecessor func() error, wrap func(ownershipPublication) ownershipPublication) error {
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
	catalog, err := s.inspectStateLocked(limits, nil, "")
	if errors.Is(err, ErrStateUncertain) {
		s.publicationBlocked = true
	}
	if err != nil {
		return err
	}
	if err = validate(catalog); err != nil {
		return err
	}
	var prior unix.Stat_t
	exists := true
	if err = unix.Fstatat(dir, name, &prior, unix.AT_SYMLINK_NOFOLLOW); err == unix.ENOENT {
		exists = false
	} else if err != nil {
		return ErrStateDirectory
	}
	if exists {
		if e := predecessor(); e != nil {
			return e
		}
		var after unix.Stat_t
		if unix.Fstatat(dir, name, &after, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(prior, after) || expected == 0 {
			return ErrDrift
		}
	} else if expected != 0 {
		return ErrDrift
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return ErrStateDirectory
	}
	temporary := prefix + hex.EncodeToString(nonce[:]) + ".tmp"
	fd, err := unix.Openat(dir, temporary, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if err != nil {
		return ErrStateDirectory
	}
	f := os.NewFile(uintptr(fd), "storage-state-temporary")
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
	verifyObjects := func() error {
		if _, e := s.ownershipDirectoryLocked(); e != nil {
			return e
		}
		var temporaryStat, current unix.Stat_t
		if unix.Fstatat(dir, temporary, &temporaryStat, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(written, temporaryStat) {
			return fmt.Errorf("temporary metadata: %w", ErrDrift)
		}
		e := unix.Fstatat(dir, name, &current, unix.AT_SYMLINK_NOFOLLOW)
		if exists {
			if e != nil || !sameOwnershipMetadata(prior, current) {
				return fmt.Errorf("destination metadata equality dev=%t inode=%t mode=%t uid=%t gid=%t links=%t size=%t mtime=%t ctime=%t birth=%t flags=%t generation=%t: %w", prior.Dev == current.Dev, prior.Ino == current.Ino, prior.Mode == current.Mode, prior.Uid == current.Uid, prior.Gid == current.Gid, prior.Nlink == current.Nlink, prior.Size == current.Size, prior.Mtim == current.Mtim, prior.Ctim == current.Ctim, prior.Btim == current.Btim, prior.Flags == current.Flags, prior.Gen == current.Gen, ErrDrift)
			}
		} else if e != unix.ENOENT {
			return ErrDrift
		}
		return nil
	}
	tx.verify = func() error {
		if e := verifyObjects(); e != nil {
			return e
		}
		catalog, e := s.inspectStateLocked(limits, nil, temporary)
		if e != nil {
			return fmt.Errorf("catalog verification: %w", e)
		}
		if e = validate(catalog); e != nil {
			return fmt.Errorf("capacity verification: %w", e)
		}
		return verifyObjects()
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
