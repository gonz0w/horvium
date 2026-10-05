#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_DEVICE_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires disposable device QEMU opt-ins, FreeBSD, and root' >&2
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
/sbin/mount -p | awk '$2 == "/" {printf "DEVICE_ROOT_FS %s\n", $3}'
/sbin/sha256 /sbin/sysctl /sbin/mount /sbin/zpool /usr/sbin/diskinfo /sbin/sha256 /sbin/gpart /sbin/newfs /sbin/swapon /sbin/swapoff /sbin/umount /usr/sbin/swapinfo /usr/sbin/fstyp
.tools/go/bin/go vet ./internal/inventory ./internal/storage
if ! .tools/go/bin/go test -v -count=1 ./internal/inventory -run '^TestNativeDeviceFixtures$' > .tools/device-native.log 2>&1; then
    cat .tools/device-native.log
    exit 1
fi
if ! .tools/go/bin/go test -v -count=1 ./internal/inventory -run '^TestNativeDiskAttachments$' >> .tools/device-native.log 2>&1; then
    cat .tools/device-native.log
    exit 1
fi
if ! .tools/go/bin/go test -v -count=1 ./internal/storage -run '^TestNativeImportedPoolUse$' >> .tools/device-native.log 2>&1; then
    cat .tools/device-native.log
    exit 1
fi
if ! .tools/go/bin/go test -v -count=1 ./internal/storage -run '^TestNativeSystemDeviceUse$' >> .tools/device-native.log 2>&1; then
    cat .tools/device-native.log
    exit 1
fi
cat .tools/device-native.log
python3.14 tools/check-device-evidence.py --guest .tools/device-native.log
