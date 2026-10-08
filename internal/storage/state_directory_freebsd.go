// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

const stateWriterName = "writer.lock"

// OpenStateDirectory opens an existing root-provisioned private directory. It
// never creates/repairs a directory or lock metadata. On first use only, it may
// create an empty exclusive regular 0600 lock file. Every success holds a lifetime
// nonblocking advisory lock and has synchronized that file and directory.
// No typed records or mutation authority are produced. Other root actors and
// remote hosts are outside this local advisory lock's protection.
func OpenStateDirectory(path string) (*StateDirectory, error) {
	if os.Geteuid() != 0 || len(path) > 4096 || !filepath.IsAbs(path) || path == "/" || filepath.Clean(path) != path || !utf8.ValidString(path) || strings.IndexFunc(path, func(r rune) bool { return r < 32 || r == 127 }) >= 0 {
		return nil, ErrStateDirectory
	}
	flags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
	dir, err := unix.Open("/", flags, 0)
	if err != nil {
		return nil, ErrStateDirectory
	}
	fail := func() (*StateDirectory, error) { _ = unix.Close(dir); return nil, ErrStateDirectory }
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	// Inspect every opened ancestor descriptor; no sticky/writable exceptions.
	for i := -1; i < len(parts); i++ {
		if i >= 0 {
			next, err := unix.Openat(dir, parts[i], flags, 0)
			_ = unix.Close(dir)
			dir = next
			if err != nil {
				return nil, ErrStateDirectory
			}
		}
		var st unix.Stat_t
		if unix.Fstat(dir, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != 0 || st.Mode&07022 != 0 || i == len(parts)-1 && st.Mode&07777 != 0700 || !safeStateACL(dir, uint32(st.Mode), true) {
			return fail()
		}
	}
	var filesystem unix.Statfs_t
	if unix.Fstatfs(dir, &filesystem) != nil || filesystem.Flags&unix.MNT_RDONLY != 0 {
		return fail()
	}
	kind := unix.ByteSliceToString(filesystem.Fstypename[:])
	if kind != "ufs" && kind != "zfs" {
		return fail()
	}
	// Reject known special/unsafe objects before opening them; descriptor
	// validation after anchored open still establishes the actual object.
	var prior unix.Stat_t
	if err := unix.Fstatat(dir, stateWriterName, &prior, unix.AT_SYMLINK_NOFOLLOW); err == nil {
		if !validStateLock(prior) {
			return fail()
		}
	} else if err != unix.ENOENT {
		return fail()
	}
	lockFlags := unix.O_RDWR | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_CLOEXEC
	// O_EXCL applies only to first creation. Existing files are opened without
	// truncation and checked independently; they are never replaced after denial.
	fd, err := unix.Openat(dir, stateWriterName, lockFlags|unix.O_CREAT|unix.O_EXCL, 0600)
	if err == unix.EEXIST {
		fd, err = unix.Openat(dir, stateWriterName, lockFlags, 0)
	}
	if err != nil {
		return fail()
	}
	badLock := func() (*StateDirectory, error) { _ = unix.Close(fd); return fail() }
	var st, current unix.Stat_t
	if unix.Fstat(fd, &st) != nil || !validStateLock(st) || !safeStateACL(fd, uint32(st.Mode), false) {
		return badLock()
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = unix.Close(fd)
		_ = unix.Close(dir)
		// Any failed lock attempt denies; only actual contention gets the busy result.
		if err == unix.EWOULDBLOCK {
			return nil, ErrStateLocked
		}
		return nil, ErrStateDirectory
	}
	if unix.Fstatat(dir, stateWriterName, &current, unix.AT_SYMLINK_NOFOLLOW) != nil || current.Dev != st.Dev || current.Ino != st.Ino || current.Mode != st.Mode || current.Uid != st.Uid || current.Nlink != 1 || current.Size != 0 {
		return badLock()
	}
	if unix.Fsync(fd) != nil || unix.Fsync(dir) != nil {
		return badLock()
	}
	s := &StateDirectory{directory: os.NewFile(uintptr(dir), "storage-state-directory"), writer: os.NewFile(uintptr(fd), "storage-state-writer")}
	s.propertyOrigin = s
	return s, nil
}

func validStateLock(st unix.Stat_t) bool {
	return st.Mode&unix.S_IFMT == unix.S_IFREG && st.Mode&07777 == 0600 && st.Uid == 0 && st.Nlink == 1 && st.Size == 0
}
