# M2 native property observation collection

Date: 2026-10-06. Status: both native targets and full portable review passed; maintainer review pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract

`freebsd.PoolPropertySource` joins the accepted pool, identity, property,
filesystem-state, feature and topology parsers into an actual read-only collector.
A trusted helper-side caller supplies pool GUID and its enrolled current name;
the shared storage name/GUID validator rejects unsafe selections before launch.
It does not accept executables, flags, property names, environment or device paths.
There is no new HTTP/helper operation, CLI flag or connection to M1 inventory.

Each of two full independent passes reads these fixed projections through the
existing bounded FreeBSD timeout/reaper, with its sanitized environment:

| Projection | Fixed native operation |
|---|---|
| Format | `/bin/freebsd-version -kru` |
| Pool | `/sbin/zpool get -H -p -o name,property,value,source guid,load_guid,health,readonly,bootfs,altroot,cachefile,compatibility,multihost <pool>` |
| Full identities | `/sbin/zfs list -H -p -r -t filesystem,volume,snapshot,bookmark -o name,guid,type <pool>` |
| Properties | `/sbin/zfs get -H -p -r -t filesystem -o name,property,value,received,source guid,quota,refquota,compression <pool>` |
| Filesystem state | `/sbin/zfs get -H -p -r -t filesystem -o name,property,value,received,source guid,mountpoint,canmount,readonly,mounted,encryption,encryptionroot,keystatus <pool>` |
| Features | `/sbin/zpool get -H -p -o name,property,value,source all <pool>` |
| GUID topology | `/sbin/zpool status -g -p <pool>` |
| Path topology | `/sbin/zpool status -L -P -p <pool>` |

Installed/running/userland release lines must all select the qualified 14.5 or
15.1 RELEASE format; patch suffixes are retained. Unknown/mixed formats refuse
collection. Pool GUID/name/import policy, complete identity/property/state
agreement, supported features/layout and filesystem readonly/encryption checks
use the accepted validators. Volumes, unsupported/adverse resources and failed
or malformed captures return no partial observation. Comparing the two complete
normalized passes and exact release lines rejects ordinary drift, including
renames, import changes, descendant/version changes and native property/source
changes. Equal repeated observations are not an atomic snapshot, a root fence
or proof that no change occurred between/after reads.

Both passes share 64 MiB of combined stdout/stderr capture and 100,000 parsed
model records, with the existing per-parser limits. Each read has a ten-second
deadline and the entire collection has 45 seconds. Cancellation or invalid reader
consumption accounting returns zero. There is no hardware-specific bound, cache,
long-lived process or public knob to raise these limits. These ceilings are
capability limits, not populated commercial-scale performance evidence.

On success `Observation.Complete` describes only these fully collected projections.
`Observation.Eligible` always stays false. Root/system/provider/device-use gates,
live root-owned policy/ownership binding and store/recovery eligibility remain
unconnected. The collector therefore cannot create an executable property review
or implement the full [coordinator backend](property-coordinator.md). Native
metadata publication, storage execution and runtime writes remain disabled.

## Qualification

Failing-first component tests cover successful dual collection, no eligibility,
independent results, failed/malformed captures, drift in every projection, unsafe
selection, missing pool, byte/record limits, invalid accounting, unsupported format
and cancellation. A bounded fuzz target exercises hostile output and selection.

`TestNativePoolPropertyCollection` requires both disposable guest opt-ins and root.
It creates two declared 256 MiB file-backed md providers, a random mirror pool,
filesystem and snapshot. The actual product collector is checked against separate
native GUID/property observations. It preserves properties and an independent
sentinel, observes a GUID-preserving external rename, denies wrong pool GUID,
readonly filesystem and foreign volume, and still fails the eligibility gate.
All 16 fixed assertions require verified pool/provider/backing teardown before the
result marker. Test setup mutations are fixture preparation, not product writes.

The focused `pool-properties` lab suite cross-builds the exact committed package
on the host and verifies its SHA-256 inside each guest. It selects only this native
test and requires the full result plus actual test pass. Missing/duplicate/skipped/
failed assertions and checksum mismatch fail qualification. Ordinary completion,
failure and supervisor interruption remove the new owned binary along with the
owned guest/image/key/variables. The previous state suite retains its target and
artifact selection. Shared runner/checker/cleanup changes require one complete
portable review; future collector-only work can use affected-package qualification.

```sh
make test-focus PKG=./internal/host/freebsd TEST='^TestPoolProperty'
PR_TITLE='feat(storage): collect native property review observations' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

No physical hardware, native policy enrollment, mutation transport, executor,
recovery, independent file-client or complete M2 workflow qualification follows.
Prior accepted manifests retain their actual tested sources.

## Executed qualification

Signed source `57ac9dde8c87db90d6cda26c28e0f3ee55ea408e`, based on PR 60 merge
`1ae83a781ebaad932d4d7126cbebaf01e34b3cdf`, passed all 16 native assertions on
FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE using the same checksum-bound executable.
The actual collector fixtures took 7.92/8.15 seconds; complete guest workloads
took 43.432/43.554 seconds including setup/verification/teardown. These are lab
durations, not NAS performance results. See the
[sanitized execution manifest](native-property-collection-manifest.json).

The required complete portable review passed the baseline (109 tooling tests),
race checks for all 11 portable packages and all 30 compiler-listed fuzz targets
in 170.446 seconds. The new binary's normal and interrupted cleanup is covered.
Both native supervisors and portable workers completed with verified teardown;
all guest/image/key/variable/executable artifacts were removed and base/runtime
hashes preserved. `make lab-status` verified all 257 owned sessions stopped.
No host services were enabled. All 36 prior M2 manifests and native storage
metadata sources remain byte-identical to the base.

The first 15.1 attempt at signed source `7de7e43ee52669f87a03de182f4ce31c794a0d8c`
failed a fixture assertion that assumed traversal order for GUID-sorted topology.
It remained unqualified with complete host/guest cleanup. A failing-first
regression plus identity/parent matching repaired only tests; product reader
code did not change. Its superseded portable review was stopped through the owned
supervisor before the correction and remains unqualified. Original failed logs,
hashes and source IDs are retained separately, not relabeled as successful runs.
Documentation-only followups rerun the baseline/current-source checks and may
reuse the exact successful deep evidence; they add no native authority.
