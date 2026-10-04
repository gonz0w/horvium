// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const digestName = "token.sha256"

// openRootDirectory walks from / using directory descriptors and O_NOFOLLOW.
// The immediate parent must prohibit non-root writes. Root-owned sticky ancestors
// (e.g. /tmp containing the root-owned test anchor) are allowed; other writable
// ancestors and every symlink are denied. No pathname check-then-open fallback.
func openRootDirectory(path string) (int, error) {
	if path != "/" && !cleanPath(path) {
		return -1, ErrState
	}
	fd, e := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if e != nil {
		return -1, ErrState
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if path == "/" {
		parts = nil
	}
	for i := -1; i < len(parts); i++ {
		if i >= 0 {
			next, e := unix.Openat(fd, parts[i], unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
			_ = unix.Close(fd)
			if e != nil {
				return -1, ErrState
			}
			fd = next
		}
		var st unix.Stat_t
		if unix.Fstat(fd, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != 0 || st.Mode&06000 != 0 || (st.Mode&0022 != 0 && (st.Mode&01000 == 0 || i == len(parts)-1)) {
			_ = unix.Close(fd)
			return -1, ErrState
		}
	}
	return fd, nil
}
func openFile(dir int, name string) (*os.File, unix.Stat_t, error) {
	fd, e := unix.Openat(dir, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, unix.Stat_t{}, ErrState
	}
	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 {
		_ = unix.Close(fd)
		return nil, unix.Stat_t{}, ErrState
	}
	return os.NewFile(uintptr(fd), "credential-state"), st, nil
}

// LoadConfig reads only an existing root-owned regular 0640/0644 file. It never
// creates or repairs configuration, follows symlinks or adopts another UID's file.
func LoadConfig(path string) (Config, error) {
	if !cleanPath(path) {
		return Config{}, ErrConfig
	}
	dir, e := openRootDirectory(filepath.Dir(path))
	if e != nil {
		return Config{}, ErrConfig
	}
	defer unix.Close(dir)
	f, st, e := openFile(dir, filepath.Base(path))
	if e != nil {
		return Config{}, ErrConfig
	}
	defer f.Close()
	if st.Uid != 0 || (st.Mode&07777 != 0640 && st.Mode&07777 != 0644) || st.Size < 1 || st.Size > MaxConfigBytes {
		return Config{}, ErrConfig
	}
	return ParseConfig(f)
}
func openState(c Config) (int, error) {
	parent, e := openRootDirectory(filepath.Dir(c.StateDirectory))
	if e != nil {
		return -1, e
	}
	defer unix.Close(parent)
	fd, e := unix.Openat(parent, filepath.Base(c.StateDirectory), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if e != nil {
		return -1, ErrState
	}
	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil || st.Uid != c.ManagementUID || st.Gid != c.SocketGID || st.Mode&07777 != 0700 {
		_ = unix.Close(fd)
		return -1, ErrState
	}
	return fd, nil
}
func readDigest(dir int, c Config) ([32]byte, error) {
	var digest [32]byte
	f, st, e := openFile(dir, digestName)
	if e != nil {
		return digest, ErrState
	}
	defer f.Close()
	if st.Uid != c.ManagementUID || st.Gid != c.SocketGID || st.Mode&07777 != 0600 || st.Size != 32 {
		return digest, ErrState
	}
	if _, e = io.ReadFull(f, digest[:]); e != nil {
		return [32]byte{}, ErrState
	}
	var extra [1]byte
	if n, e := f.Read(extra[:]); n != 0 || e != io.EOF {
		return [32]byte{}, ErrState
	}
	return digest, nil
}

// LoadDigest is a management-process startup operation. Root has no runtime
// authorization exemption; the separate explicit setup action requires root.
func LoadDigest(c Config) ([32]byte, error) {
	if c.Validate() != nil || os.Geteuid() != int(c.ManagementUID) {
		return [32]byte{}, ErrState
	}
	dir, e := openState(c)
	if e != nil {
		return [32]byte{}, ErrState
	}
	defer unix.Close(dir)
	return readDigest(dir, c)
}

// SetupToken is an explicit local OS-administrator action. The parent must
// already exist and be protected. The action creates the service-owned 0700
// leaf if absent and publishes only a digest. Initial setup refuses an existing
// target; rotation requires a safe existing digest. Restart management after
// rotation. The token is returned only after persistence, never logged here.
func SetupToken(c Config, rotate bool) (string, error) {
	if c.Validate() != nil || os.Geteuid() != 0 {
		return "", ErrState
	}
	parent, e := openRootDirectory(filepath.Dir(c.StateDirectory))
	if e != nil {
		return "", ErrState
	}
	defer unix.Close(parent)
	name := filepath.Base(c.StateDirectory)
	if !rotate {
		e = unix.Mkdirat(parent, name, 0700)
		if e == nil {
			// Anchor ownership changes to the opened new directory, not its pathname.
			fd, oe := unix.Openat(parent, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
			if oe != nil {
				return "", ErrState
			}
			var st unix.Stat_t
			safe := unix.Fstat(fd, &st) == nil && st.Uid == 0 && st.Mode&07777 == 0700
			if safe {
				safe = unix.Fchown(fd, int(c.ManagementUID), int(c.SocketGID)) == nil && unix.Fsync(fd) == nil
			}
			_ = unix.Close(fd)
			if !safe || unix.Fsync(parent) != nil {
				return "", ErrState
			}
		} else if e != unix.EEXIST {
			return "", ErrState
		}
	}
	dir, e := openState(c)
	if e != nil {
		return "", ErrState
	}
	defer unix.Close(dir)
	if rotate {
		if _, e = readDigest(dir, c); e != nil {
			return "", ErrState
		}
	}
	token, e := randomSecret(rand.Reader)
	if e != nil {
		return "", ErrState
	}
	digest := sha256.Sum256([]byte(token))
	suffix, e := randomSecret(rand.Reader)
	if e != nil {
		return "", ErrState
	}
	temp := ".token-" + suffix
	fd, e := unix.Openat(dir, temp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if e != nil {
		return "", ErrState
	}
	f := os.NewFile(uintptr(fd), "credential-state")
	defer f.Close()
	defer unix.Unlinkat(dir, temp, 0)
	if n, e := f.Write(digest[:]); e != nil || n != len(digest) {
		return "", ErrState
	}
	if unix.Fchown(fd, int(c.ManagementUID), int(c.SocketGID)) != nil || unix.Fchmod(fd, 0600) != nil || f.Sync() != nil {
		return "", ErrState
	}
	if rotate {
		e = unix.Renameat(dir, temp, dir, digestName)
	} else {
		// linkat publishes atomically without replacing any preexisting entry.
		e = unix.Linkat(dir, temp, dir, digestName, 0)
		if e == nil {
			e = unix.Unlinkat(dir, temp, 0)
		}
	}
	if e != nil || unix.Fsync(dir) != nil {
		return "", ErrState
	}
	// Fail closed on replacement/tampering before returning the one-time token.
	var owned, current unix.Stat_t
	if unix.Fstat(fd, &owned) != nil || unix.Fstatat(dir, digestName, &current, unix.AT_SYMLINK_NOFOLLOW) != nil || owned.Dev != current.Dev || owned.Ino != current.Ino {
		return "", ErrState
	}
	stored, e := readDigest(dir, c)
	if e != nil || stored != digest {
		return "", ErrState
	}
	return token, nil
}
