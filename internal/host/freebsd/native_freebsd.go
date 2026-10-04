// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "context"

const nativeSupported = true

func readNative(ctx context.Context, p probe, limit int) ([]byte, error) {
	// No program, argument, path or environment comes from an inventory request.
	switch p {
	case versions:
		return execute(ctx, "/bin/freebsd-version", []string{"-kru"}, limit)
	case architecture:
		return execute(ctx, "/usr/bin/uname", []string{"-m"}, limit)
	case memory:
		return execute(ctx, "/sbin/sysctl", []string{"-n", "hw.physmem"}, limit)
	case zfsVersion:
		return execute(ctx, "/sbin/zfs", []string{"version"}, limit)
	case geom:
		return execute(ctx, "/sbin/sysctl", []string{"-b", "kern.geom.confxml"}, limit)
	case pools:
		return execute(ctx, "/sbin/zpool", []string{"list", "-H", "-p", "-o", "name,guid,size,alloc,free,health"}, limit)
	case datasets:
		return execute(ctx, "/sbin/zfs", []string{"list", "-H", "-p", "-t", "filesystem", "-o", "name,guid,used,available,referenced"}, limit)
	default:
		return nil, ErrUnavailable
	}
}
