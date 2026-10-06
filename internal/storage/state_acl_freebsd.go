// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

func readStateACL(fd, kind int) (stateACL, error) {
	a := stateACL{Max: 254}
	_, _, errno := unix.Syscall(unix.SYS___ACL_GET_FD, uintptr(fd), uintptr(kind), uintptr(unsafe.Pointer(&a)))
	runtime.KeepAlive(&a)
	if errno != 0 {
		return stateACL{}, errno
	}
	return a, nil
}

// Fpathconf queries the actual opened object. Errors/unknown answers deny; only
// two confirmed zeros establish mode-only access. No ENOTSUP fallback is used.
func safeStateACL(fd int, mode uint32, directory bool) bool {
	posix, err := unix.Fpathconf(fd, 59) // _PC_ACL_EXTENDED
	if err != nil || posix < 0 || posix > 1 {
		return false
	}
	nfs4, err := unix.Fpathconf(fd, 64) // _PC_ACL_NFS4
	if err != nil || nfs4 < 0 || nfs4 > 1 || posix == 1 && nfs4 == 1 {
		return false
	}
	if posix == 0 && nfs4 == 0 {
		return true
	}
	kind := 2
	if nfs4 == 1 {
		kind = 4
	}
	a, err := readStateACL(fd, kind)
	if err != nil || !safeStateACLValue(a, kind, mode, directory) {
		return false
	}
	if posix == 1 && directory {
		a, err = readStateACL(fd, 3)
		if err != nil || !safeStateACLValue(a, 3, mode, true) {
			return false
		}
	}
	return true
}
