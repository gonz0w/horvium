#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_CLOSED_STARTUP_OWNER_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires disposable backing QEMU opt-ins, FreeBSD, and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
expected=${HORVIUM_TEST_BINARY_SHA256:-}
case "$expected" in ''|*[!a-f0-9]*) echo 'Invalid executable checksum' >&2; exit 1;; esac
[ "${#expected}" = 64 ] || exit 1
binary=.tools/policy-startup-owner-native.test
[ -f "$binary" ] && [ ! -L "$binary" ] && [ -x "$binary" ] || exit 1
[ "$(/sbin/sha256 -q "$binary")" = "$expected" ] || exit 1
printf 'POLICY_STARTUP_OWNER_PREBUILT_SHA256 %s\n' "$expected"
freebsd-version -kru
zfs version
/sbin/sha256 /sbin/mount /sbin/umount /sbin/mdconfig /sbin/newfs /sbin/dumpfs /sbin/zfs /sbin/zpool /usr/bin/timeout /sbin/sysctl /usr/sbin/swapinfo /usr/sbin/diskinfo /bin/freebsd-version
log=.tools/policy-startup-owner-native.log
printf 'POLICY_STARTUP_OWNER_PHASE policy begin timeout=34m\n' > "$log"
if ! "$binary" -test.v -test.count=1 -test.timeout=34m -test.failfast -test.run='^TestNativePolicyStartup(Lifecycle|Revocation|Denial)Owner(UFS|ZFS)$' >> "$log" 2>&1; then
    cat "$log"
    exit 1
fi
printf 'POLICY_STARTUP_OWNER_PHASE policy done\nPOLICY_STARTUP_OWNER_PHASE retained begin timeout=34m\n' >> "$log"
if ! "$binary" -test.v -test.count=1 -test.timeout=34m -test.failfast -test.run='^TestNative(ExecutionStartup|ClosedStartupOwner)(UFS|ZFS)$' >> "$log" 2>&1; then
    cat "$log"
    exit 1
fi
printf 'POLICY_STARTUP_OWNER_PHASE retained done\n' >> "$log"
cat "$log"
# Complete assertions are validated by the pinned host-side evidence checker.
