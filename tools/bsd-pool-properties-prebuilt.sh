#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_PROPERTY_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires disposable property QEMU opt-ins, FreeBSD, and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
expected=${HORVIUM_TEST_BINARY_SHA256:-}
case "$expected" in ''|*[!a-f0-9]*) echo 'Invalid executable checksum' >&2; exit 1;; esac
[ "${#expected}" = 64 ] || exit 1
binary=.tools/pool-properties-native.test
[ -f "$binary" ] && [ ! -L "$binary" ] && [ -x "$binary" ] || exit 1
[ "$(/sbin/sha256 -q "$binary")" = "$expected" ] || exit 1
printf 'POOL_PROPERTIES_PREBUILT_SHA256 %s\n' "$expected"
freebsd-version -kru
zfs version
/sbin/sha256 /sbin/zfs /sbin/zpool /sbin/mdconfig /usr/bin/timeout /usr/sbin/diskinfo /sbin/sysctl /sbin/mount /usr/sbin/swapinfo /sbin/swapon /sbin/swapoff /bin/freebsd-version
if ! "$binary" -test.v -test.count=1 -test.run='^TestNative(PoolPropertyCollection|PropertyClaimCollection)$' > .tools/pool-properties-native.log 2>&1; then
    cat .tools/pool-properties-native.log
    exit 1
fi
cat .tools/pool-properties-native.log
# Complete native assertions are validated by the pinned host-side checker.
