#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_STATE_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires disposable state QEMU opt-ins, FreeBSD, and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
expected=${HORVIUM_TEST_BINARY_SHA256:-}
case "$expected" in ''|*[!a-f0-9]*) echo 'Invalid executable checksum' >&2; exit 1;; esac
[ "${#expected}" = 64 ] || exit 1
binary=.tools/state-native.test
[ -f "$binary" ] && [ ! -L "$binary" ] && [ -x "$binary" ] || exit 1
[ "$(/sbin/sha256 -q "$binary")" = "$expected" ] || exit 1
printf 'STATE_PREBUILT_SHA256 %s\n' "$expected"
freebsd-version -kru
zfs version
/sbin/mount -p | awk '$2 == "/" {printf "STATE_ROOT_FS %s\n", $3}'
/sbin/sha256 /sbin/mount /sbin/umount /sbin/mdconfig /sbin/newfs
if ! "$binary" -test.v -test.count=1 -test.run='^Test(Native(StateDirectory|OwnershipRead|OwnershipCatalog|OwnershipCapacity|JobRead|JobPublication|StateCatalog|StateACL|OwnershipPublish|OwnershipPublishFilesystem)|StateACLPolicy|MixedOwnershipPublicationCapacity|StateCatalog(Names|Records|Limits)|Ownership.*|Job.*)$' > .tools/state-native.log 2>&1; then
    cat .tools/state-native.log
    exit 1
fi
cat .tools/state-native.log
# Complete assertions are validated by the pinned host-side evidence checker.
