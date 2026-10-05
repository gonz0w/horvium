# M2 disposable virtual device identity fixture

Date: 2026-10-05. Status: all eight repaired assertions and complete preservation/teardown passed
on both FreeBSD targets through the supervised Mac runner;
accepted in PR 28 at merged source `27dafaba7a5d04a4896f27776ef79f4b1f4c2bbc`.
GitHub Actions remain disabled.
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

Five newly owned 64 MiB sparse raw files back five virtio block devices:

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

`tools/device-fixture.py` exclusively creates the fixed backing set in a new
private directory beneath the disposable guest fixture. It accepts no host disk
input. The local runner launches the actual installed QEMU directly, using its
snapshot boot drive and bounded optional firmware drives; the appender attaches
only the five declared test devices after verifying size, mode and SHA-256.
Reject option-delimiter paths and symlink paths/files. No PATH wrapper, hardcoded
Linux QEMU path or host `/proc` identity machinery is required. QEMU and the
pinned upstream runtime/cache artifacts remain unchanged.

The original `TestNativeDeviceFixtures` runs only read operations:

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
Trace the actual mounted UFS/ZFS guest root's leaf/alias paths through GEOM backing edges
and prove the root disks are outside the declared set. Preserve the root mount and backing correspondence; preserve its pool GUID when
the root is ZFS. UFS image roots require no invented pool identity.
No boot disk content or arbitrary operator resource is opened for hashing or writes.

Repeat graph observations and whole-device hashes after the reads. This original read-only test
adds no partitioning, labels, pool creation, mounts, swap or guest disk writes.
The subsequent [attachment denial test](disk-attachments.md) creates an owned
GPT table/partition on the uniquely identified 512-byte snapshot fixture,
verifies empty/populated PART refusal and deletes the owned partition/table.
It preserves the other four disks and root correspondence; it grants no product
partition/wipe authority. The modified guest backing is discarded.
The whole-device hashes verify the read-only test's content preservation; they
are not a production signature probe or destructive-use allowlist.

The [lease supervisor](../local-qualification.md#teardown) owns the foreground
guest worker and all its child processes. It reserves the worker leader PID until
group cleanup completes, stops/reaps the owned group and never signals stored
PIDs from an earlier run. The worker stops/reaps its direct QEMU child before
rehashing backings and removing the five declared files/manifest. The supervisor
provides fixed-path fallback removal after abrupt controller death; interrupted
runs remain unqualified. Unknown entries are preserved and make cleanup fail.
Backing tampering fails preservation even when disposable files are removed.
Global snapshot mode preserves the raw originals and verified base image.

## Execution and evidence

Use a clean signed checkout and select each disposable guest explicitly:

```sh
python3.14 tools/local-qualification.py devices 15.1 --run
python3.14 tools/local-qualification.py devices 14.5 --run
make lab-status
```

Run releases sequentially on the Mac. The two-CPU, 6144 MiB, amd64 TCG guests
inherit the authenticated base-image/cache, firmware, source archive and isolated
SSH procedure. No host SSH configuration, service or production disk is used.
GitHub Actions remain disabled; no hosted dispatch or billing adjustment is needed.

Failing-first fixture/evidence tests cover declared-only launch, non-snapshot or
extra-boot-drive refusal, backing size/content/symlink changes, private-directory refusal, cleanup after changed/partially prepared backings,
foreign-entry preservation and skipped/incomplete/duplicate/failed assertions. Local unit,
race, parser fuzz and FreeBSD cross-build results are separate from native evidence.
For the [disk correlation extension](disk-correlation.md), the checker requires
both executed native tests, eight fixture assertions, four correlation assertions
and five attachment assertions, plus one successful host
preservation/termination/teardown result; booting a VM,
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
The first supervised Mac run on `ecc6c1daa7eca4112f9b0ebbbfa917b9eb27c448`
failed at the three-minute native fixture deadline (180.02 seconds). Base/runtime
and all five 512 MiB backings were preserved; process/key/image/backing teardown
completed. This is failed evidence. The fixture now uses five 64 MiB disks,
reducing two complete hash passes from 5 GiB to 640 MiB while retaining all eight
identity/geometry/boot/preservation assertions and the three-minute bound. This
is a test-volume change, not a relaxed product or performance budget.

The first repaired 14.5 run on `4f669f21aead2094cfa4942adac534fd4bde8073`
failed at the fixture's ZFS-only root assumption (12.25 seconds); preservation
and teardown completed. Root control now
recognizes UFS and ZFS mounts, follows exact leaf/alias GEOM backing on both,
rejects ambiguous/missing/cyclic controls, and verifies mount/backing stability
after the reads. ZFS roots additionally retain their pool GUID check.

The repaired 64 MiB fixture on source `069d38f98bcc39e94861b22bf7db40a4fac70969` passed all eight
assertions on FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE using Mac arm64 / QEMU 11.1.1
TCG. The actual mounted roots were ZFS on 15.1 and UFS on 14.5. Both runs preserved
the complete raw backings, base image and QEMU/firmware
inputs, removed disks/image/key/variables and verified the owned group stopped.
See the [sanitized local execution record](devices-local-manifest.json), which
also retains both the first Mac deadline failure and the 14.5 root-control failure
separately from the original hosted record.

A further live device-fixture controller was killed with SIGKILL. Its QEMU/SSH
listener stopped, all five backings/image/key/variables were removed, and the
interrupted run remained unqualified. The complete local review gate passed on
the tested functional source; final PR-head portable evidence is reported in the
PR. Source/input hashes bind these native results; fixture input changes require fresh
qualification. PR 28 was accepted by maintainer merge; this is not milestone acceptance.

Native qualification must record local source/suite records, exact tested inputs and tool/
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
