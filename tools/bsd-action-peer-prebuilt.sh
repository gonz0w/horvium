#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_ACTION_PEER_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires disposable peer QEMU opt-ins, FreeBSD, and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
expected=${HORVIUM_TEST_BINARY_SHA256:-}
case "$expected" in ''|*[!a-f0-9]*) echo 'Invalid executable checksum' >&2; exit 1;; esac
[ "${#expected}" = 64 ] || exit 1
binary=.tools/action-peer-native.test
[ -f "$binary" ] && [ ! -L "$binary" ] && [ -x "$binary" ] || exit 1
[ "$(/sbin/sha256 -q "$binary")" = "$expected" ] || exit 1
printf 'ACTION_PEER_PREBUILT_SHA256 %s\n' "$expected"
freebsd-version -kru
/sbin/sha256 /bin/freebsd-version /usr/bin/timeout
if ! /usr/bin/timeout 90 "$binary" -test.v -test.count=1 -test.timeout=80s -test.run='^TestNative(ActionPeer|PropertyExchange|AdmittedPropertyExchange|SessionPropertyClient)$' > .tools/action-peer-native.log 2>&1; then
    cat .tools/action-peer-native.log
    exit 1
fi
cat .tools/action-peer-native.log
# The host checker requires every kernel/lifecycle case and fixture teardown.
