// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
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
	return s.publishStateInspectedLocked(name, prefix, data, expected, validate, predecessor,
		func(temporary string) (StateCatalog, error) { return s.inspectStateLocked(limits, nil, temporary) }, wrap)
}

// The inspector must validate the complete descriptor-bound namespace and may
// exclude only this transaction's independently verified temporary. Submission
// publication adds full baseline/reservation checks to each actual census.
func (s *StateDirectory) publishStateInspectedLocked(name, prefix string, data []byte, expected uint64, validate func(StateCatalog) error, predecessor func() error, inspect func(string) (StateCatalog, error), wrap func(ownershipPublication) ownershipPublication) error {
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
	catalog, err := inspect("")
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
	var original []byte
	exists := true
	if err = unix.Fstatat(dir, name, &prior, unix.AT_SYMLINK_NOFOLLOW); err == unix.ENOENT {
		exists = false
	} else if err != nil {
		return ErrStateDirectory
	}
	if exists {
		original, err = s.readStateBytesLocked(name)
		if err != nil {
			return err
		}
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
	// Keep the original inode and exact bytes across staging. On UFS, deferred
	// rename bookkeeping can change ctime when later I/O flushes dependencies.
	// Bind the authoritative full metadata snapshot after staging sync, then
	// retain strict before/read/after and pre-rename comparisons. No retries or
	// timestamp exceptions are used in that final verification window.
	initial := prior
	tx.closeFile = func() error {
		if e := closeFile(); e != nil {
			return e
		}
		if !exists {
			return nil
		}
		if unix.Fstatat(dir, name, &prior, unix.AT_SYMLINK_NOFOLLOW) != nil || prior.Dev != initial.Dev || prior.Ino != initial.Ino {
			return ErrDrift
		}
		current, e := s.readStateBytesLocked(name)
		if e != nil {
			return e
		}
		if !bytes.Equal(original, current) {
			return ErrDrift
		}
		if e = predecessor(); e != nil {
			return e
		}
		var after unix.Stat_t
		if unix.Fstatat(dir, name, &after, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(prior, after) {
			return ErrDrift
		}
		return nil
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
				return fmt.Errorf("destination metadata: %w", ErrDrift)
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
		catalog, e := inspect(temporary)
		if e != nil {
			return fmt.Errorf("catalog verification: %w", e)
		}
		if e = validate(catalog); e != nil {
			return fmt.Errorf("capacity verification: %w", e)
		}
		return verifyObjects()
	}
	tx.rename = func() error { return unix.Renameat(dir, temporary, dir, name) }
	tx.syncDirectory = func() error {
		// UFS can defer the renamed inode's ctime bookkeeping until later I/O.
		// Flush our exact published inode before the final directory barrier so
		// the next strict census need not ignore timestamps or retry drift.
		if err := syncPublishedState(dir, name, data, written); err != nil {
			return err
		}
		return unix.Fsync(dir)
	}
	if wrap != nil {
		tx = wrap(tx)
	}
	err = commitOwnershipPublication(tx)
	if errors.Is(err, ErrStateUncertain) {
		s.publicationBlocked = true
	}
	return err
}

// Rename already occurred. Any failure is uncertain, never rollback/cleanup of
// the destination. Only the exact owned inode may be synchronized; verify its
// canonical bytes and strict metadata again after the expected rename flush.
func syncPublishedState(dir int, name string, data []byte, written unix.Stat_t) error {
	fd, err := unix.Openat(dir, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return ErrStateDirectory
	}
	f := os.NewFile(uintptr(fd), "storage-state-published")
	defer f.Close()
	var before, after, named unix.Stat_t
	if unix.Fstat(fd, &before) != nil || !privateOwnershipFile(before) || !safeStateACL(fd, uint32(before.Mode), false) || before.Dev != written.Dev || before.Ino != written.Ino || before.Size != int64(len(data)) || before.Mtim != written.Mtim || before.Gid != written.Gid || before.Flags != written.Flags || before.Gen != written.Gen || before.Btim != written.Btim {
		return ErrDrift
	}
	if err := f.Sync(); err != nil {
		return ErrStateDirectory
	}
	// This is the authoritative post-flush window; no timestamp exception or
	// retry is made for later reads or the complete reservation/state census.
	if unix.Fstat(fd, &before) != nil || !privateOwnershipFile(before) || before.Size != int64(len(data)) || !safeStateACL(fd, uint32(before.Mode), false) {
		return ErrDrift
	}
	current, err := io.ReadAll(io.LimitReader(f, int64(len(data))+1))
	if err != nil || !bytes.Equal(current, data) || unix.Fstat(fd, &after) != nil || !sameOwnershipMetadata(before, after) || unix.Fstatat(dir, name, &named, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(after, named) {
		return ErrDrift
	}
	if f.Close() != nil {
		return ErrStateDirectory
	}
	return nil
}
