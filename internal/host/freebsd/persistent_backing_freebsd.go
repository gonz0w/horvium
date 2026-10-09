// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func collectNativePersistentBacking(ctx context.Context, fd int) (persistentBackingIdentity, error) {
	if ctx == nil || fd < 0 {
		return persistentBackingIdentity{}, ErrUnavailable
	}
	if os.Geteuid() != 0 {
		return persistentBackingIdentity{}, ErrPermission
	}
	var original unix.Stat_t
	if unix.Fstat(fd, &original) != nil || original.Mode&unix.S_IFMT != unix.S_IFDIR || original.Uid != 0 || original.Mode&07777 != 0700 {
		return persistentBackingIdentity{}, ErrUnavailable
	}
	var device unix.Stat_t
	deviceObserved := false
	s := persistentBackingCollector{read: readPersistentBackingNative, bytes: 64 << 10, commandTimeout: 10 * time.Second, totalTimeout: 45 * time.Second}
	s.observe = func() (persistentBackingMount, error) {
		var st unix.Stat_t
		var fs unix.Statfs_t
		if os.Geteuid() != 0 || unix.Fstat(fd, &st) != nil || !samePersistentStat(original, st) || unix.Fstatfs(fd, &fs) != nil || fs.Flags&unix.MNT_RDONLY != 0 || fs.Flags&unix.MNT_SOFTDEP != 0 {
			return persistentBackingMount{}, ErrUnavailable
		}
		m := persistentMount(fs)
		if !validPersistentMount(m) || m.fsid == ([2]int32{}) {
			return persistentBackingMount{}, ErrUnavailable
		}
		// Match the current mount root to the held directory. Mount IDs support
		// this live correspondence only; they never become persistent identities.
		root, e := unix.Open(m.target, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if e != nil {
			return persistentBackingMount{}, ErrUnavailable
		}
		var rootFS unix.Statfs_t
		var rootStat unix.Stat_t
		bad := unix.Fstatfs(root, &rootFS) != nil || unix.Fstat(root, &rootStat) != nil || rootStat.Dev != st.Dev || rootStat.Uid != 0 || persistentMount(rootFS) != m
		if unix.Close(root) != nil || bad {
			return persistentBackingMount{}, ErrUnavailable
		}
		if m.kind == "ufs" {
			var current unix.Stat_t
			if unix.Stat(m.source, &current) != nil || current.Mode&unix.S_IFMT != unix.S_IFCHR || current.Uid != 0 || current.Rdev != st.Dev {
				return persistentBackingMount{}, ErrUnavailable
			}
			if deviceObserved && !samePersistentStat(device, current) {
				return persistentBackingMount{}, ErrUnavailable
			}
			device = current
			deviceObserved = true
		}
		return m, nil
	}
	return s.collect(ctx)
}

func persistentMount(fs unix.Statfs_t) persistentBackingMount {
	return persistentBackingMount{unix.ByteSliceToString(fs.Fstypename[:]), unix.ByteSliceToString(fs.Mntfromname[:]), unix.ByteSliceToString(fs.Mntonname[:]), fs.Fsid.Val, uint64(fs.Flags)}
}
func samePersistentStat(a, b unix.Stat_t) bool {
	return a.Dev == b.Dev && a.Ino == b.Ino && a.Mode == b.Mode && a.Nlink == b.Nlink && a.Uid == b.Uid && a.Gid == b.Gid && a.Rdev == b.Rdev && a.Size == b.Size && a.Mtim == b.Mtim && a.Ctim == b.Ctim && a.Btim == b.Btim && a.Flags == b.Flags && a.Gen == b.Gen
}
func readPersistentBackingNative(ctx context.Context, probe, name string, limit int) (readOutput, error) {
	switch probe {
	case "versions":
		return executeNative(ctx, "/bin/freebsd-version", []string{"-kru"}, limit)
	case "ufs-id":
		if !validPersistentProvider(name) {
			return readOutput{}, ErrUnavailable
		}
		return executeNative(ctx, "/sbin/dumpfs", []string{"-l", name}, limit)
	case "zfs-id":
		if !validPersistentDataset(name) {
			return readOutput{}, ErrUnavailable
		}
		return executeNative(ctx, "/sbin/zfs", []string{"get", "-H", "-p", "-o", "name,property,value", "guid,type,mounted,mountpoint", name}, limit)
	case "pool-id":
		if !validPersistentDataset(name) || strings.Contains(name, "/") {
			return readOutput{}, ErrUnavailable
		}
		return executeNative(ctx, "/sbin/zpool", []string{"get", "-H", "-p", "-o", "value", "guid", name}, limit)
	}
	return readOutput{}, ErrUnavailable
}
