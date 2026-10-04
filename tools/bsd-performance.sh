#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_PERF_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires explicit disposable FreeBSD performance VM opt-in and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
export GOTOOLCHAIN=local GOENV=off CGO_ENABLED=0
export GOCACHE="$root/.tools/go-cache" GOMODCACHE="$root/.tools/go-modules"
.tools/go/bin/go version
freebsd-version -kru
zfs version
sysctl hw.ncpu hw.model hw.physmem
.tools/go/bin/go test -v -count=1 -timeout=20m ./cmd/horvium -run '^TestNativePerformanceBudgets$'
