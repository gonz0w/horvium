// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"net"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

const nativeSupported = true

func peerUID(c *net.UnixConn) (uint32, error) {
	raw, err := c.SyscallConn()
	if err != nil {
		return 0, ErrPermission
	}
	var uid uint32
	var lookupErr error
	err = raw.Control(func(fd uintptr) {
		cred, e := unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if e != nil || cred == nil {
			lookupErr = ErrPermission
			return
		}
		uid, lookupErr = credentialUID(cred.Version, cred.Uid)
	})
	if err != nil || lookupErr != nil {
		return 0, ErrPermission
	}
	return uid, nil
}

func safeDirectory(c Config) error {
	dir := filepath.Dir(c.Socket)
	first := true
	for {
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return ErrConfig
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 {
			return ErrConfig
		}
		if first {
			if stat.Gid != c.SocketGID || info.Mode().Perm() != 0750 || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
				return ErrConfig
			}
			first = false
		} else if info.Mode().Perm()&0022 != 0 && info.Mode()&os.ModeSticky == 0 {
			return ErrConfig
		}
		// A root-owned sticky ancestor (e.g. /tmp) protects the root-owned
		// child entry. All other ancestors must prohibit non-root writes.
		if dir == "/" {
			break
		}
		dir = filepath.Dir(dir)
	}
	return nil
}
func safeSocket(c Config) error {
	if err := safeDirectory(c); err != nil {
		return err
	}
	info, err := os.Lstat(c.Socket)
	if err != nil {
		return ErrConfig
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 || stat.Gid != c.SocketGID || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm() != 0660 || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return ErrConfig
	}
	return nil
}
