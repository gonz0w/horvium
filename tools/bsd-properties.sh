#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_PROPERTY_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires explicit disposable property QEMU opt-ins, FreeBSD, and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
export GOTOOLCHAIN=local CGO_ENABLED=0 GOENV=off
export GOCACHE="$root/.tools/go-cache"
export GOMODCACHE="$root/.tools/go-modules"
python3.14 --version
python3.14 tools/bootstrap-go.py
.tools/go/bin/go version
.tools/go/bin/go mod download
.tools/go/bin/go mod verify
freebsd-version -kru
zfs version
/sbin/sha256 /sbin/zfs /sbin/zpool /sbin/mdconfig
test -z "$(.tools/go/bin/gofmt -l internal/storage)"
.tools/go/bin/go vet ./internal/storage
if ! .tools/go/bin/go test -v -count=1 ./internal/storage > .tools/property-native.log 2>&1; then
    cat .tools/property-native.log
    exit 1
fi
cat .tools/property-native.log
python3.14 tools/check-property-evidence.py .tools/property-native.log
echo 'PASS: executed native property projection, identity/source assertions and owned-fixture teardown'
