#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_STATE_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires disposable state QEMU opt-ins, FreeBSD, and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
export GOTOOLCHAIN=local CGO_ENABLED=0 GOENV=off
export GOCACHE="$root/.tools/go-cache" GOMODCACHE="$root/.tools/go-modules"
python3.14 --version
python3.14 tools/bootstrap-go.py
.tools/go/bin/go version
.tools/go/bin/go mod download
.tools/go/bin/go mod verify
freebsd-version -kru
zfs version
/sbin/mount -p | awk '$2 == "/" {printf "STATE_ROOT_FS %s\n", $3}'
/sbin/sha256 /sbin/mount /sbin/umount /sbin/mdconfig /sbin/newfs
.tools/go/bin/go vet ./internal/storage
if ! .tools/go/bin/go test -v -count=1 ./internal/storage -run '^Test(Native(StateDirectory|OwnershipRead|OwnershipCatalog|OwnershipCapacity|JobRead|JobPublication|StateCatalog|StateACL|OwnershipPublish|OwnershipPublishFilesystem)|StateACLPolicy|MixedOwnershipPublicationCapacity|StateCatalog(Names|Records|Limits)|Ownership.*|Job.*)$' > .tools/state-native.log 2>&1; then
    cat .tools/state-native.log
    exit 1
fi
cat .tools/state-native.log
python3.14 tools/check-state-evidence.py .tools/state-native.log
