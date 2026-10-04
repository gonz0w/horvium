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
