//go:build !freebsd && !darwin && !linux

// SPDX-License-Identifier: BSD-3-Clause
package console

func peakRSS() uint64 { return 0 }
