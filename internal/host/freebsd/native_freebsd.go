// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "context"

const nativeSupported = true

func readNative(ctx context.Context, p probe, limit int) (readOutput, error) {
	// No program, argument, path or environment comes from an inventory request.
	switch p {
	case versions:
		return executeNative(ctx, "/bin/freebsd-version", []string{"-kru"}, limit)
	case architecture:
		return executeNative(ctx, "/usr/bin/uname", []string{"-m"}, limit)
	case memory:
		return executeNative(ctx, "/sbin/sysctl", []string{"-n", "hw.physmem"}, limit)
	case zfsVersion:
		return executeNative(ctx, "/sbin/zfs", []string{"version"}, limit)
	case geom:
		return executeNative(ctx, "/sbin/sysctl", []string{"-b", "kern.geom.confxml"}, limit)
	case pools:
		return executeNative(ctx, "/sbin/zpool", []string{"list", "-H", "-p", "-o", "name,guid,size,alloc,free,health"}, limit)
	case datasets:
		return executeNative(ctx, "/sbin/zfs", []string{"list", "-H", "-p", "-t", "filesystem", "-o", "name,guid,used,available,referenced"}, limit)
	default:
		return readOutput{}, ErrUnavailable
	}
}
