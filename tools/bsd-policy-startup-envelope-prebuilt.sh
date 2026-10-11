#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_POLICY_STARTUP_ENVELOPE_VM:-}" != 1 ] || [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_POLICY_FACTS_VM:-}" != 1 ] || [ "${HORVIUM_CLOSED_STARTUP_OWNER_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires owned diagnostic opt-ins, FreeBSD and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
expected=${HORVIUM_TEST_BINARY_SHA256:-}
case "$expected" in ''|*[!a-f0-9]*) echo 'Invalid executable checksum' >&2; exit 1;; esac
[ "${#expected}" = 64 ] || exit 1
binary=.tools/policy-startup-envelope-native.test
[ -f "$binary" ] && [ ! -L "$binary" ] && [ -x "$binary" ] || exit 1
[ "$(/sbin/sha256 -q "$binary")" = "$expected" ] || exit 1
printf 'POLICY_STARTUP_ENVELOPE_PREBUILT_SHA256 %s\n' "$expected"
freebsd-version -kru
zfs version
/sbin/sha256 /usr/bin/timeout /usr/bin/true /bin/freebsd-version /sbin/zpool /sbin/zfs /sbin/sysctl /sbin/mount /sbin/umount /sbin/newfs /sbin/dumpfs /usr/sbin/diskinfo /usr/sbin/swapinfo
[ "$(/sbin/sysctl -n hw.ncpu)" = 1 ] || exit 1
printf 'POLICY_STARTUP_ENVELOPE_PROFILE cpu_count=1 child_seconds=210 parent_seconds=480 phase_minutes=18\n'
log=.tools/policy-startup-envelope-native.log
if ! "$binary" -test.v -test.count=1 -test.timeout=18m -test.failfast -test.run='^TestNativeMultiplePolicyStartupEnvelopeUFS$' > "$log" 2>&1; then
    cat "$log"
    exit 1
fi
cat "$log"
