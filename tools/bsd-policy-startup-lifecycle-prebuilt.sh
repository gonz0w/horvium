#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_POLICY_ROOT_LIFECYCLE_VM:-}" != 1 ] || [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_POLICY_FACTS_VM:-}" != 1 ] || [ "${HORVIUM_POLICY_ROOT_REVIEW_VM:-}" != 1 ] || [ "${HORVIUM_CLOSED_STARTUP_OWNER_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires disposable QEMU root-review opt-ins, FreeBSD and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
expected=${HORVIUM_TEST_BINARY_SHA256:-}
case "$expected" in ''|*[!a-f0-9]*) echo 'Invalid executable checksum' >&2; exit 1;; esac
[ "${#expected}" = 64 ] || exit 1
binary=.tools/policy-startup-lifecycle-native.test
[ -f "$binary" ] && [ ! -L "$binary" ] && [ -x "$binary" ] || exit 1
[ "$(/sbin/sha256 -q "$binary")" = "$expected" ] || exit 1
printf 'POLICY_ROOT_LIFECYCLE_PREBUILT_SHA256 %s\n' "$expected"
freebsd-version -kru
zfs version
/sbin/sha256 /sbin/mount /sbin/umount /sbin/newfs /sbin/zfs /sbin/zpool /sbin/sysctl /usr/sbin/swapinfo /usr/sbin/diskinfo /bin/freebsd-version
log=.tools/policy-startup-lifecycle-native.log
printf 'POLICY_ROOT_REVIEW_PHASE facts begin timeout=18m\n' > "$log"
if ! "$binary" -test.v -test.count=1 -test.timeout=18m -test.failfast -test.run='^TestNativeMultiplePolicyFacts$' >> "$log" 2>&1; then
    cat "$log"
    exit 1
fi
printf 'POLICY_ROOT_REVIEW_PHASE facts done\nPOLICY_ROOT_REVIEW_PHASE review begin timeout=18m\n' >> "$log"
if ! "$binary" -test.v -test.count=1 -test.timeout=18m -test.failfast -test.run='^TestNativeMultiplePolicyReview(UFS|ZFS)$' >> "$log" 2>&1; then
    cat "$log"
    exit 1
fi
printf 'POLICY_ROOT_REVIEW_PHASE review done\n' >> "$log"
printf 'POLICY_ROOT_LIFECYCLE_PHASE lifecycle begin timeout=18m\n' >> "$log"
if ! "$binary" -test.v -test.count=1 -test.timeout=18m -test.failfast -test.run='^TestNativeMultiplePolicyLifecycle(UFS|ZFS)$' >> "$log" 2>&1; then
    cat "$log"
    exit 1
fi
printf 'POLICY_ROOT_LIFECYCLE_PHASE lifecycle done\n' >> "$log"
cat "$log"
# The pinned host requires all retained/root controls and independent teardown.
