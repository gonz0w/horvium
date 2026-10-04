# M1 independent SMB/NFS continuity qualification

Status: native qualification passed; under maintainer review. Owner: Cam (`@gonz0w`).
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
It runs foreground base rpcbind (`-d`), debug/foreground mountd (`-d`) with a
private exports file, and debug/foreground nfsd (`--debug`) with
two kernel threads. Daemon PID/export kernel state is guest-global. mountd automatically appends loopback addresses to explicit bind hosts. The
fixture creates/removes an owned 127.0.0.2 loopback alias to avoid binding the same
fixed port twice; preexisting alias state is refused. Bindings/exports admit loopback only;
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
only the owned pool/md. Base rpcbind deliberately returns exit status 2 after handling SIGTERM; only
that exact daemon/signal/status is accepted in addition to success or termination
by the requested signal. Other nonzero exits fail. A leak or failed assertion fails the test; skipped opt-in
runs do not qualify continuity. No operator's disks, backups, exports, users or
credentials are fixture input.

## Execution record

Source `95d94754f634bd8c1eee840d5dc8662472790493` passed all five executed jobs in
[run 37233530125](https://github.com/gonz0w/horvium/actions/runs/37233530125).
Both protocol subtests and their parent passed, including daemon/client/export,
owned loopback alias and pool/md teardown. Seven fresh exact-byte reads per
protocol survived the helper/management outages, stale recovery and authenticated
management restart. Old sessions were invalidated; configs/digest and original
owned dataset properties remained unchanged.

| Actual guest / Go | Samba package / daemon | Suite duration | Result |
|---|---|---|---|
| FreeBSD 15.1-RELEASE-p3 amd64 / 1.27.1 | net/samba423 4.23.11_3 / 4.23.11 | 3.50s | SMB + NFSv3 TCP and teardown pass |
| FreeBSD 14.5-RELEASE amd64 / 1.27.1 | net/samba423 4.23.11_2 / 4.23.11 | 6.34s | SMB + NFSv3 TCP and teardown pass |

ZFS is 2.4.2-1-FreeBSD_g6330a45b0 / 2.2.9-FreeBSD_g079ba86d7 with matching
kernel strings. VM Action a2f9a41fa97f6848b8c3b791087dfcdaa5b473ff, builder 2.2.8,
6144 MiB and default CPU; no immutable image/performance claim. Exact six
server/client/base hashes, source file hashes and versions are retained in the
[source manifest](../../research/M1/source-manifest.json).

Early runs failed because Samba shutdown signaled the runner's shared process
group (37230301602), fixture exit handling rejected a requested signal
(37230695969), incorrect foreground flags/duplicate mountd loopback fixed-port
binding blocked startup (37231466130,37231992233,37232452266,37232922830), and
rpcbind's intentional termination status was rejected after successful reads
(37233214442). Each was repaired in the fixture; those failed runs do not qualify
the suite. Daemons now use verified separate owned groups; FreeBSD foreground
flags, an owned loopback alias and exact rpcbind shutdown status follow base
source behavior. No arbitrary nonzero exit is accepted.

Local `make check lint test secrets build go-check` passed, including race and
bounded fuzz checks; subsequent native fixture repairs passed repository checks
and actual native execution. Product protocol management, external clients,
ACL/locking/restore, sustained throughput, hardware and full M1 acceptance remain
separate gates. These results establish management/file-daemon separation only.
