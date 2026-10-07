// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "context"

func readSystemNative(ctx context.Context, p systemProbe, name string, limit int) (readOutput, error) {
	switch p {
	case systemVersions:
		return executeNative(ctx, "/bin/freebsd-version", []string{"-kru"}, limit)
	case systemGEOM:
		return executeNative(ctx, "/sbin/sysctl", []string{"-b", "kern.geom.confxml"}, limit)
	case systemMounts:
		return executeNative(ctx, "/sbin/mount", []string{"-p"}, limit)
	case systemSwap:
		return executeNative(ctx, "/usr/sbin/swapinfo", []string{"-k"}, limit)
	case systemPools:
		return executeNative(ctx, "/sbin/zpool", []string{"list", "-H", "-p", "-o", "name,guid,size,alloc,free,health"}, limit)
	case systemPoolGUID:
		return executeNative(ctx, "/sbin/zpool", []string{"get", "-H", "-p", "-o", "value", "guid", name}, limit)
	case systemPoolTopologyGUID:
		return executeNative(ctx, "/sbin/zpool", []string{"status", "-g", "-p", name}, limit)
	case systemPoolTopologyPath:
		return executeNative(ctx, "/sbin/zpool", []string{"status", "-L", "-P", "-p", name}, limit)
	default:
		return readOutput{}, ErrUnavailable
	}
}
