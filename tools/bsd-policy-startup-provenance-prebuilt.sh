#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_POLICY_STARTUP_PROVENANCE_VM:-}" != 1 ] || [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_POLICY_FACTS_VM:-}" != 1 ] || [ "${HORVIUM_CLOSED_STARTUP_OWNER_VM:-}" != 1 ] || [ "${HORVIUM_POLICY_PROVENANCE_SOURCE_ISOLATED:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires owned diagnostic opt-ins, verified build isolation, FreeBSD and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
expected=${HORVIUM_TEST_BINARY_SHA256:-}
source=${HORVIUM_SOURCE_COMMIT:-}
compiler=${HORVIUM_COMPILER_SHA256:-}
for value in "$expected" "$source" "$compiler"; do
    case "$value" in ''|*[!a-f0-9]*) echo 'Invalid candidate identity' >&2; exit 1;; esac
done
[ "${#expected}" = 64 ] && [ "${#source}" = 40 ] && [ "${#compiler}" = 64 ] || exit 1
binary=.tools/policy-startup-provenance-native.test
[ -f "$binary" ] && [ ! -L "$binary" ] && [ -x "$binary" ] || exit 1
[ "$(/sbin/sha256 -q "$binary")" = "$expected" ] || exit 1
freebsd-version -kru
zfs version
/sbin/sha256 /usr/bin/timeout /usr/bin/true /bin/freebsd-version /sbin/zpool /sbin/zfs /sbin/sysctl /sbin/mount /sbin/umount /sbin/newfs /sbin/dumpfs /usr/sbin/diskinfo /usr/sbin/swapinfo
[ "$(/sbin/sysctl -n hw.ncpu)" = 1 ] || exit 1
printf 'POLICY_STARTUP_PROVENANCE_TRANSCRIPT_BEGIN\n'
printf 'POLICY_STARTUP_PROVENANCE_PREBUILT_SHA256 %s\n' "$expected"
printf 'POLICY_STARTUP_PROVENANCE_PROFILE cpu_count=1 child_seconds=210 parent_seconds=480 phase_minutes=18\n'
printf 'POLICY_STARTUP_PROVENANCE_SOURCE {"schema":1,"profile":"deadline-provenance-v1","source_commit":"%s","compiler_sha256":"%s","artifact_sha256":"%s"}\n' "$source" "$compiler" "$expected"
log=.tools/policy-startup-provenance-native.log
status=0
"$binary" -test.v -test.count=1 -test.timeout=18m -test.failfast -test.run='^TestNativeMultiplePolicyStartupProvenanceUFS$' > "$log" 2>&1 || status=$?
cat "$log"
printf 'POLICY_STARTUP_PROVENANCE_TRANSCRIPT_END\n'
exit "$status"
