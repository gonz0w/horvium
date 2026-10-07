// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "context"

// Only fixed read-only base projections reach the accepted native reaper.
// Pool names were validated by Collect; neither paths nor properties are inputs.
func readPoolNative(ctx context.Context, p poolProbe, name string, limit int) (readOutput, error) {
	var path string
	var args []string
	switch p {
	case poolVersions:
		return executeNative(ctx, "/bin/freebsd-version", []string{"-kru"}, limit)
	case poolState:
		path = "/sbin/zpool"
		args = []string{"get", "-H", "-p", "-o", "name,property,value,source", "guid,load_guid,health,readonly,bootfs,altroot,cachefile,compatibility,multihost", name}
	case poolIdentities:
		path = "/sbin/zfs"
		args = []string{"list", "-H", "-p", "-r", "-t", "filesystem,volume,snapshot,bookmark", "-o", "name,guid,type", name}
	case poolProperties:
		path = "/sbin/zfs"
		args = []string{"get", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,property,value,received,source", "guid,quota,refquota,compression", name}
	case poolFilesystemState:
		path = "/sbin/zfs"
		args = []string{"get", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,property,value,received,source", "guid,mountpoint,canmount,readonly,mounted,encryption,encryptionroot,keystatus", name}
	case poolFeatures:
		path = "/sbin/zpool"
		args = []string{"get", "-H", "-p", "-o", "name,property,value,source", "all", name}
	case poolTopologyGUID:
		path = "/sbin/zpool"
		args = []string{"status", "-g", "-p", name}
	case poolTopologyPath:
		path = "/sbin/zpool"
		args = []string{"status", "-L", "-P", "-p", name}
	default:
		return readOutput{}, ErrUnavailable
	}
	return executeNative(ctx, path, args, limit)
}
