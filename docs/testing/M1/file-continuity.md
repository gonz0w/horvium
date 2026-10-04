# M1 independent SMB/NFS continuity qualification

Status: implemented; native execution pending. Owner: Cam (`@gonz0w`).
Date: 2026-10-04. Requirements M1-REQ-007,003; tests M1-RESTART-001 and
M1-NOCHANGE-001. This supplements [SFTP continuity](runtime.md). File-service
configuration, client interoperability, ACLs, locking and restoration remain
M3/M4/M6 designs and gates.

## Exclusive disposable fixture

The manual `bsd-continuity.yml` workflow installs the official FreeBSD
`net/samba423` package in fresh QEMU guests and invokes the guarded
`tools/bsd-continuity.sh`. Exact installed package version/origin and server/client
binary hashes are captured. Repository packages can change; this is functional
native evidence, not a reproducible package or immutable performance baseline.
The pinned VM Action, selected release and source revision accompany results.
No Samba source/assets are copied into Horvium, and no product Go dependency is
added. Samba remains a separate GPL service; redistribution/license/transitive
cost and release configuration must be reviewed before packaging.

Each protocol uses a fresh file-backed md pool and new mounted fixture dataset.
The existing unmounted `data` fixture and its property checks are retained. Exports
are on that owned filesystem; the guest boot filesystem is never exported. Only
generated fixture content is shared. Daemons run manually, without rc.d enablement
or changing rc.conf, /etc/exports or smb4.conf. Before/after assertions preserve
those operator-facing files, including their absence. Helpers and management never
configure these daemons.

SMB uses standalone smbd, loopback-only binding, an ephemeral TCP port, SMB2 or
newer, a read-only guest fixture, and private config/state/cache/lock/PID/log
locations. Each checkpoint creates a new smbclient session and retrieves the
fixture to a local generated path, comparing exact bytes and unchanged daemon
config. No reusable credentials or public guest service is provisioned.
See the official [smbd](https://www.samba.org/samba/docs/current/man-html/smbd.8.html),
[smbclient](https://www.samba.org/samba/docs/current/man-html/smbclient.1.html) and
[smb.conf](https://www.samba.org/samba/docs/current/man-html/smb.conf.5.html) contracts.
This fixture does not adopt guest access as Horvium's future authentication design.

NFS refuses a guest with existing rpcbind/mountd/nfsd or occupied loopback 2049.
It runs base rpcbind, mountd with a private exports file, and foreground nfsd with
a private PID file and two kernel threads. Bindings/exports admit loopback only;
NFSv3 TCP mounts are read-only, with bounded soft retries. Each checkpoint creates
a fresh fixture filename and a fresh client mount, compares transferred bytes,
requires local mount writes to fail, then unmounts. Fresh names prevent a client
cache from substituting for a protocol read. Server exports/config remain unchanged.
NFS kernel exports and mountd's standard PID metadata are guest-global; exclusively
disposable guests are required, and are discarded after cleanup. The fixture may
load base NFS modules but never edits persistent enablement. See official
[nfsd](https://man.freebsd.org/cgi/man.cgi?query=nfsd&sektion=8&manpath=FreeBSD+15.1-RELEASE),
[mountd](https://man.freebsd.org/mountd%288%29) and
[mount_nfs](https://man.freebsd.org/mount_nfs%288%29) contracts.

## Checkpoints and teardown

Seven exact-byte protocol reads run with management healthy, immediately after
helper SIGKILL, after stale retention, after fresh helper recovery, while management
is killed, after management restart/new login and after both Horvium processes
stop. Daemon PIDs remain the originally started processes. An old management
session must be unauthorized after restart; config/digest remain unchanged. Stale
socket removal is restricted to its captured owned inode. No product repair or
restart service is added.

Cleanup stops only fixture daemons (nfsd uses SIGUSR1), unmounts the owned client
and dataset, verifies the unchanged original data properties and destroys/detaches
only the owned pool/md. A leak or failed assertion fails the test; skipped opt-in
runs do not qualify continuity. No operator's disks, backups, exports, users or
credentials are fixture input.

## Execution record

FreeBSD test cross-compilation passes; this is build evidence only. Actual native
execution, source hashes, package/base versions, linked run IDs and results will
be recorded before qualification. Nothing here claims macOS/Linux file-client
compatibility, sustained throughput, production shares or full M1 acceptance.
