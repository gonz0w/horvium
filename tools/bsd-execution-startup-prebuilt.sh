#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_EXECUTION_STARTUP_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires disposable backing QEMU opt-ins, FreeBSD, and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
expected=${HORVIUM_TEST_BINARY_SHA256:-}
case "$expected" in ''|*[!a-f0-9]*) echo 'Invalid executable checksum' >&2; exit 1;; esac
[ "${#expected}" = 64 ] || exit 1
binary=.tools/execution-startup-native.test
[ -f "$binary" ] && [ ! -L "$binary" ] && [ -x "$binary" ] || exit 1
[ "$(/sbin/sha256 -q "$binary")" = "$expected" ] || exit 1
printf 'EXECUTION_STARTUP_PREBUILT_SHA256 %s\n' "$expected"
freebsd-version -kru
zfs version
/sbin/sha256 /sbin/mount /sbin/umount /sbin/mdconfig /sbin/newfs /sbin/dumpfs /sbin/zfs /sbin/zpool /usr/bin/timeout /sbin/sysctl /usr/sbin/swapinfo /usr/sbin/diskinfo /bin/freebsd-version
if ! "$binary" -test.v -test.count=1 -test.timeout=18m -test.run='^TestNativeExecutionStartup(UFS|ZFS)$' > .tools/execution-startup-native.log 2>&1; then
    cat .tools/execution-startup-native.log
    exit 1
fi
cat .tools/execution-startup-native.log
# Complete assertions are validated by the pinned host-side evidence checker.
