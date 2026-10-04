#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires explicit disposable QEMU opt-in, FreeBSD, and root' >&2
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
test -z "$(.tools/go/bin/gofmt -l cmd internal tools/research/m1-geom.go)"
.tools/go/bin/go vet ./...
.tools/go/bin/go build -trimpath -o .tools/horvium ./cmd/horvium
.tools/go/bin/go test -v -count=1 ./...
.tools/go/bin/go test ./internal/console -run '^$' -bench '^BenchmarkAuthenticatedDatasetPages$' -benchtime=1x
echo 'PASS: product discovery/helper/cache/admin/console/runtime unit/component and native fixture tests'
