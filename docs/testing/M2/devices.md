# M2 disposable virtual device identity fixture

Date: 2026-10-05. Status: local fixture checks passed; repaired native qualification and PR 28 CI blocked by GitHub account limits.
[Approved EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[implementation sequence](../../milestones/M2/implementation.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Purpose and scope

M2-DEVICE-001 needs real guest-attached block devices with explicit identifiers,
not a bypass that treats file-backed md as a creation candidate. This fixture
establishes that foundation under the approved Test Design. It exercises existing
M1 observation models against declared QEMU disks; it adds no product provider
selector, policy enrollment, native write adapter or mutation authority.
M1 `unique_in_collection` means only observed identifier uniqueness. It proves
neither durable identity across controllers/restarts nor direct local physical
provenance, absent use or safety to create/replace storage.

Five newly owned 512 MiB sparse raw files back five virtio block devices:

| Fixture serial | Count | Configured logical/physical block size | Required native observation |
|---|---|---|---|
| `HVM-UNIQUE-512` | 1 | 512 / 4096 | Unique serial, 512 logical sector, 4096 stripe size and zero offset |
| `HVM-UNIQUE-4K` | 1 | 4096 / 4096 | Unique serial, 4096 logical sector; retain native zero stripe size |
| `HVM-DUPLICATE` | 2 | 512 / 512 | Both ambiguous; never choose one by path |
| No configured serial | 1 | 512 / 512 | Missing identity; never invent one from provider path |

The pinned virtio driver reports a stripe size for a positive physical-block
exponent; equal configured physical/logical sizes report zero here. Configured
QEMU geometry is not physical disk/controller qualification, and a zero native
stripe value is not fabricated into a physical-sector measurement.

Inspected primary sources explain these projections; hashes will be retained with
the native record. They do not prove which sources built the guest binaries:
[14.5 diskinfo](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/usr.sbin/diskinfo/diskinfo.c),
[15.1 diskinfo](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/usr.sbin/diskinfo/diskinfo.c),
[14.5 virtio geometry/identifier driver](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/dev/virtio/block/virtio_blk.c),
[15.1 virtio driver](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/dev/virtio/block/virtio_blk.c),
[14.5 GEOM serial producer](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/geom/geom_disk.c),
[15.1 GEOM serial producer](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/geom/geom_disk.c).

[Pinned DEV producer](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/geom/geom_dev.c),
[LABEL producer](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/geom/label/g_label.c)
and [disk-ident alias](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/geom/label/g_label_disk_ident.c)
explain why a blank disk can have incoming consumers without partition/data use.
Both release revisions are retained in the source hash record.

## Ownership, observations and teardown

`tools/device-fixture.py` exclusively creates the fixed backing set and a wrapper
in a new private fixture directory. It has no arbitrary host disk argument.
Version/accelerator probes pass through; launch requires a snapshot boot drive,
with bounded optional firmware drives. It appends only the five declared devices.
The wrapper calls `/usr/bin/qemu-system-x86_64` directly; the actual QEMU and pinned
AnyVM remain unmodified. Before launch, verify backing file shape, size and SHA-256.
Reject option-delimiter paths and symlink paths/files.

The guest runs only read operations:

```text
/sbin/sysctl -b kern.geom.confxml
/usr/sbin/diskinfo -s /dev/<declared-provider>
/usr/sbin/diskinfo /dev/<declared-provider>
/sbin/sha256 -q /dev/<declared-provider>
/sbin/mount -p
/sbin/zpool status -L -P <guest-root-pool>
/sbin/zpool get -H -p -o value guid <guest-root-pool>
```

Fixture providers are resolved from the complete native graph, then checked against
exact declared serial/count/size/class expectations. This test's fixed size and
`vtbd` address checks describe its virtual fixture, not a product selection rule.
Independent diskinfo ioctls confirm serial, logical sector, byte/sector capacity,
stripe size and offset. Require direct DISK provenance without backing parents. Native DEV attachments
and optional disk-ident LABEL aliases are checked by class/parent/address/geometry;
other incoming consumers refuse this fixture recognizer. These attachments do
not establish absent labels/use. Consumer absence alone is never a safety proof.
Trace the mounted ZFS guest root's actual leaf paths through GEOM backing edges
and prove the root disks are outside the declared set. Preserve that root pool GUID.
No boot disk content or arbitrary operator resource is opened for hashing or writes.

Repeat graph observations and whole-device hashes after the reads. This fixture
adds no partitioning, labels, pool creation, mounts, swap or guest disk writes.
The whole-device hashes verify the read-only test's content preservation; they
are not a production signature probe or destructive-use allowlist.

The host records the launched QEMU PID, kernel process start ticks and exact argument
hash. Cleanup checks all three and the real executable before sending SIGTERM;
PID/command mismatch refuses the signal. Verify termination (or a zombie with no
remaining executor/file descriptors), then rehash the complete raw backings,
remove only the declared files/wrapper/records and require directory absence.
Global snapshot mode keeps the original backings and verified base image unchanged.
Any startup, assertion, cleanup or pin failure invalidates evidence; discard the
runner/guest. Process identity checks here are Linux QEMU test infrastructure,
not a FreeBSD product dependency or M2 executor-containment qualification.

## Execution and evidence

`bsd-devices.yml` is a checksum-pinned opt-in two-release workflow. Before that
new workflow exists on main, use the existing registered dispatch entry:

```sh
gh workflow run bsd-properties.yml --ref <review-branch> -f device_fixture=true
```

The property workflow's default behavior stays unchanged; the opt-in fixture selects
the new sibling workflow. It uses the same verified disposable base images,
AnyVM, Go toolchain and fresh two-CPU/6,144 MiB snapshot guests as prior evidence.
No NAS service is installed or enabled.

Failing-first fixture/evidence tests cover declared-only launch, non-snapshot or
extra-boot-drive refusal, backing size/content/symlink changes, process identity
mismatch and skipped/incomplete/duplicate/failed assertion rejection. Local unit,
race, parser fuzz and FreeBSD cross-build results are separate from native evidence.
The checker requires one executed native test and eight exact guest assertions,
plus one successful host preservation/termination/teardown result; booting a VM,
a guest-only result or a skipped test cannot qualify this fixture.

The initial run [37327282454](https://github.com/gonz0w/horvium/actions/runs/37327282454)
failed on both guests at source `b35c7f7f68e98a3dbe561f3d6d1d8215382cf9c5`: the fixture
incorrectly required no incoming GEOM consumers for blank disks. Owned host cleanup
and base-image/AnyVM post-checks passed, but neither guest supplied acceptance
evidence. Inspected DEV/LABEL producers explain normal device-node and disk-ident
attachments. A failing regression now checks their exact correspondence and rejects
partition/unknown/contradictory attachment shapes; no product use check was relaxed.
At repaired source `3ded164fe4cab801e15fe6cd95cbc0e4766c3f23`,
[run 37328280188](https://github.com/gonz0w/horvium/actions/runs/37328280188)
did not start either guest: GitHub reported failed account payments or a spending
limit requiring adjustment. Zero steps executed. The same scheduling annotation
blocked all checks in [PR CI run 37328289533](https://github.com/gonz0w/horvium/actions/runs/37328289533).
These are unstarted gates, not failing product assertions or native passes.

[The sanitized execution record](devices-manifest.json) preserves the initial
failures, successful initial host cleanup/pin checks, repaired local checked
source/input hashes, inspected primary sources and the unstarted native/CI jobs.
Local `make check lint test secrets`, `make go-check` (vet/unit/race/twelve bounded
fuzz targets) and the FreeBSD amd64 inventory test-binary cross-build passed.
The repaired suite still needs actual execution on both guests, complete teardown
and successful current-head CI. PR 28 remains a draft pending those gates.

Once GitHub Actions account billing/limits are restored, dispatch the device fixture
from the latest PR branch with the command above and rerun the current-head
repository checks. Do not accept this fixture or enable any device operation from
the locally passing regression alone.

Native qualification must record both run/job IDs, exact tested inputs and tool/
source/image hashes, actual guest versions and observed geometry. Do not export raw
GEOM/boot inventory, credentials or private keys. Earlier projection manifests
retain their original sources/case sets. This is partial M2-DEVICE-001/M2-NATIVE-001
infrastructure evidence, not complete device safety, slice 1 or M2 acceptance.

Pending product gates include strict provider/alias/backing correspondence,
serial/lunid enrollment and fresh drift resolution, unsupported/multipath/locality
refusal, partitions/signatures, imported/exported pool membership, boot/swap/mount
use, geometry policy and reservations at review/launch. Stable identity across
virtual reattachment, physical controllers and sector variants outside this fixture
still require evidence. Runtime stays read-only. M1's deferred manual accessibility
gate remains pending.
