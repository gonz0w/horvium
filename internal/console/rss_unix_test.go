//go:build freebsd || darwin || linux

// SPDX-License-Identifier: BSD-3-Clause
package console

import (
	"runtime"
	"syscall"
)

func peakRSS() uint64 {
	var r syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &r) != nil || r.Maxrss < 0 {
		return 0
	}
	rss := uint64(r.Maxrss)
	if runtime.GOOS != "darwin" {
		rss *= 1024
	}
	return rss
}
