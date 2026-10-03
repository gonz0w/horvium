#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
# Research only: run solely inside a disposable FreeBSD QEMU guest.
set -eu
if [ "${HORVIUM_RESEARCH_VM:-}" != 1 ] || [ "$(uname -s)" != FreeBSD ] || [ "$(id -u)" != 0 ]; then
    echo 'Requires explicit disposable-VM opt-in, FreeBSD, and root' >&2
    exit 1
fi
if [ "$(uname -m)" != amd64 ]; then
    echo 'This pinned experiment is amd64 only' >&2
    exit 1
fi
root=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
work=$(mktemp -d /tmp/horvium-m1.XXXXXXXX)
pool=horvium_m1_$$
device=''
created=0
cleanup() {
    result=$?
    trap - EXIT HUP INT TERM
    if [ "$created" = 1 ]; then
        if ! zpool destroy "$pool"; then
            echo 'Fixture pool cleanup failed; discard the guest' >&2
            exit 1
        fi
    fi
    if [ -n "$device" ]; then
        if ! mdconfig -d -u "${device#md}"; then
            echo 'Fixture device cleanup failed; discard the guest' >&2
            exit 1
        fi
    fi
    rm -rf -- "$work"
    exit "$result"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
echo 'Native research environment (fixture-only evidence)'
freebsd-version -kru
uname -m
zfs version
fetch -q -o "$work/go.tgz" https://go.dev/dl/go1.27.1.freebsd-amd64.tar.gz
expected=9761194512938a640948ae5ba273df73a441f4229ea21a1503937e55c69bf107
[ "$(sha256 -q "$work/go.tgz")" = "$expected" ]
tar -xzf "$work/go.tgz" -C "$work"
export GOTOOLCHAIN=local CGO_ENABLED=0
"$work/go/bin/go" version
"$work/go/bin/go" build -o "$work/geom-probe" "$root/tools/research/m1-geom.go"
cc -Wall -Wextra -Werror -o "$work/peer-probe" "$root/tools/research/m1-peer.c"
# No operator-supplied pool/device arguments; never force pool creation.
truncate -s 512M "$work/disk.img"
device=$(mdconfig -a -t vnode -f "$work/disk.img")
case "$device" in
    md[0-9]*) ;;
    *) echo 'Unexpected fixture device' >&2; exit 1 ;;
esac
case "${device#md}" in
    ''|*[!0-9]*) echo 'Invalid fixture unit' >&2; exit 1 ;;
esac
mdconfig -l -n -f "$work/disk.img" > "$work/backing-unit"
awk -v unit="${device#md}" '
    NF != 1 || $1 != unit { exit 1 }
    END { if (NR != 1) exit 1 }
' "$work/backing-unit"
echo 'PASS: generated md unit verified by exact backing-file query'
zpool create -m none "$pool" "/dev/$device"
created=1
zfs create -o mountpoint=none "$pool/data"
zfs list -H -p -t filesystem -o name,guid,used,available,referenced -r "$pool" > "$work/before"
zpool list -H -p -o name,guid,size,alloc,free,health "$pool" > "$work/pool"
awk -F '\t' -v pool="$pool" '
    NF != 6 || $1 != pool || $2 !~ /^[0-9]+$/ || $3 !~ /^[0-9]+$/ ||
    $4 !~ /^[0-9]+$/ || $5 !~ /^[0-9]+$/ || $6 != "ONLINE" { exit 1 }
    END { if (NR != 1) exit 1 }
' "$work/pool"
awk -F '\t' '
    NF != 5 || $2 !~ /^[0-9]+$/ || $3 !~ /^[0-9]+$/ ||
    $4 !~ /^[0-9]+$/ || $5 !~ /^[0-9]+$/ { exit 1 }
    END { if (NR != 2) exit 1 }
' "$work/before"
"$work/geom-probe" "$device"
# Verify discovery did not alter fixture dataset identities/properties.
zfs list -H -p -t filesystem -o name,guid,used,available,referenced -r "$pool" > "$work/after"
# Space accounting can change asynchronously; only identity is invariant here.
cut -f 1,2 "$work/before" > "$work/identity-before"
cut -f 1,2 "$work/after" > "$work/identity-after"
cmp "$work/identity-before" "$work/identity-after"
guid=$(zfs get -H -p -o value guid "$pool/data")
zfs rename "$pool/data" "$pool/renamed"
[ "$(zfs get -H -p -o value guid "$pool/renamed")" = "$guid" ]
echo 'PASS: TSV fields, numeric units, and GUID preserved through fixture rename'
# Permit nobody to traverse this disposable fixture directory for the socket test.
chmod 755 "$work"
"$work/peer-probe" "$work/socket"
echo 'PASS: native research contracts; production adapter remains unimplemented'
