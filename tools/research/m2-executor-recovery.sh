#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_PROPERTY_VM:-}" != 1 ] || [ "${HORVIUM_RESEARCH_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires disposable recovery research opt-ins, FreeBSD and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"
expected=${HORVIUM_TEST_BINARY_SHA256:-}
case "$expected" in ''|*[!a-f0-9]*) exit 1;; esac
[ "${#expected}" = 64 ] || exit 1
binary=.tools/pool-properties-native.test
[ -f "$binary" ] && [ ! -L "$binary" ] && [ -x "$binary" ] || exit 1
[ "$(/sbin/sha256 -q "$binary")" = "$expected" ] || exit 1
case "${1:-before}" in
    before)
        printf 'EXECUTOR_RECOVERY_PREBUILT_SHA256 %s\n' "$expected"
        freebsd-version -kru
        zfs version
        /usr/bin/cc --version | head -n 1
        /sbin/sha256 /usr/bin/cc /usr/bin/timeout /sbin/zfs /usr/include/sys/user.h /usr/include/sys/procctl.h
        /usr/bin/cc -std=c11 -Wall -Wextra -Werror -O2 tools/research/m2-executor-observer.c -o .tools/executor-observer
        chmod 700 .tools/executor-observer
        parent='TestNative(ExecutorRecoveryBefore|ExecutorSupervision)'
        ;;
    after) parent='TestNative(ExecutorRecoveryAfter|SupervisorBootAfter|SupervisorPropertyControls)' ;;
    *) exit 1;;
esac
"$binary" -test.v -test.count=1 -test.timeout=6m -test.run="^${parent}$"
