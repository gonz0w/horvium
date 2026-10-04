//go:build freebsd || darwin || linux

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"os"
	"syscall"
)

func ownedInput(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(os.Getuid())
}

// Anonymous pipes have no filesystem link. Linked FIFOs must be operator-owned
// and inaccessible to group/other, like regular credential input files.
func privatePipe(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && (st.Nlink == 0 || (ownedInput(info) && info.Mode().Perm()&0077 == 0))
}
