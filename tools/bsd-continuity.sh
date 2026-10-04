#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
if [ "${HORVIUM_NATIVE_VM:-}" != 1 ] || [ "${HORVIUM_CONTINUITY_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires explicit disposable FreeBSD protocol VM opt-in and root' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
export GOTOOLCHAIN=local GOENV=off CGO_ENABLED=0
export GOCACHE="$root/.tools/go-cache" GOMODCACHE="$root/.tools/go-modules"
python3.14 tools/bootstrap-go.py
.tools/go/bin/go mod download
.tools/go/bin/go mod verify
.tools/go/bin/go version
freebsd-version -kru
zfs version
pkg query '%n %v %o' samba423
/usr/local/sbin/smbd --version
/sbin/sha256 /usr/local/sbin/smbd /usr/local/bin/smbclient /usr/sbin/nfsd /usr/sbin/mountd /usr/sbin/rpcbind /sbin/mount_nfs
.tools/go/bin/go build -trimpath -o .tools/horvium ./cmd/horvium
.tools/go/bin/go test -v -count=1 -timeout=10m ./cmd/horvium -run '^TestNativeIndependentSMBNFSContinuity$'
echo 'PASS: actual independent SMB/NFS fixture reads through helper and management outages'
