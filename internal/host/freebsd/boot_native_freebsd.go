// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"golang.org/x/sys/unix"
	"runtime"
	"unsafe"
)

func readNativeBoot() (bootIdentity, error) {
	// __sysctlbyname's fixed native ABI is qualified initially on amd64 only.
	if runtime.GOARCH != "amd64" {
		return bootIdentity{}, ErrUnsupported
	}
	name := []byte("kern.boot_id")
	var id bootIdentity
	size := uintptr(len(id))
	_, _, errno := unix.Syscall6(unix.SYS___SYSCTLBYNAME, uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)), uintptr(unsafe.Pointer(&id[0])), uintptr(unsafe.Pointer(&size)), 0, 0)
	runtime.KeepAlive(name)
	runtime.KeepAlive(&id)
	if errno != 0 || size != uintptr(len(id)) {
		return bootIdentity{}, ErrUnavailable
	}
	return id, nil
}
