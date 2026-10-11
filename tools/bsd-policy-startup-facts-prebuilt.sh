#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_POLICY_FACTS_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires disposable QEMU opt-ins, FreeBSD, and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
expected=${HORVIUM_TEST_BINARY_SHA256:-}
case "$expected" in ''|*[!a-f0-9]*) echo 'Invalid executable checksum' >&2; exit 1;; esac
[ "${#expected}" = 64 ] || exit 1
binary=.tools/policy-startup-facts-native.test
[ -f "$binary" ] && [ ! -L "$binary" ] && [ -x "$binary" ] || exit 1
[ "$(/sbin/sha256 -q "$binary")" = "$expected" ] || exit 1
printf 'POLICY_FACTS_PREBUILT_SHA256 %s\n' "$expected"
freebsd-version -kru
zfs version
/sbin/sha256 /sbin/mount /sbin/zfs /sbin/zpool /sbin/sysctl /usr/sbin/swapinfo /usr/sbin/diskinfo /bin/freebsd-version
log=.tools/policy-startup-facts-native.log
if ! "$binary" -test.v -test.count=1 -test.timeout=18m -test.failfast -test.run='^TestNativeMultiplePolicyFacts$' > "$log" 2>&1; then
    cat "$log"
    exit 1
fi
cat "$log"
# The pinned host checker requires all child/parent controls and independent teardown.
