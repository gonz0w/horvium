# M2 native repaired-checksum finding qualification

Date: 2026-10-07. Status: qualified subset; maintainer review pending.
Authority: [approved PRD](../../milestones/M2/PRD.md),
[EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-HEALTH-001, M2-SCRUB-001, M2-DRIFT-001 and M2-NATIVE-001 evidence.

## Scope

Exercise the accepted [health/scan observer](pool-health.md) against actual
kernel checksum findings on one leaf of the already-owned disposable mirror.
The other replica must preserve independently generated file content. Completed
scan state/error count and retained member checksum counters are distinct facts;
a repaired scrub can complete with zero final scan errors while a member reports
nonzero checksum findings. Never infer error-free history from completed/ONLINE.

Product code, eligibility gates, job history and operator write workflows remain
unchanged. These new assertions exercise existing code; there is no new product
implementation requiring a failing portable test. Retain every actual native
attempt at its source. This is read-buffer fault injection, not manually corrupted
backing bytes, physical media/controller resilience, unrecoverable data errors,
attributed scrub execution or full maintenance acceptance.

## Owned native fixture and mandatory cases

Reuse the existing checksum-bound `job-status` suite with no shared tool changes.
Preserve all 21 original status assertions, fifteen assessment children, six
health children and eight scan-state children. After those checks and before
GUID/backing-verified pool teardown, add `native-checksum`. Require all eight
children and the group once, with no FAIL/SKIP:

- `preflight`: kernel root and both disposable opt-ins; stable owned permanent/
  import identity and selected leaf GUID/address; healthy baseline/raw properties;
  no registered injectors and restored zero progress control. Record actual base
  injector binary SHA-256.
- `content-fixture`: create one exclusive child filesystem with a private fixture
  mount and 1 MiB quota; record its native GUID. Exclusively write/sync/close
  32 KiB random sentinel content, then sync the owned pool.
- `fault-registered`: recheck pool/leaf identity and empty injector baseline.
  Register a read-buffer corruption handler using the native decimal leaf GUID,
  fixed `corrupt`, `read` and 100-percent frequency arguments. Require canonical
  acknowledged handler ID and an actual native list entry matching ID/pool/leaf.
- `scrub-findings`: start only the GUID-verified owned pool and independently
  collect actual completed scrub facts. Require nonzero selected-leaf checksum
  counters, explicit zero final scan errors, zero unrepairable data errors and
  unchanged import identity. Keep counters separate from completion/progress.
- `fault-cleared`: clear only the acknowledged handler ID and independently
  verify the exact original empty handler list.
- `content-preserved`: independently read and compare the sentinel with generated
  expected bytes after removing the fault.
- `clean-scrub`: clear counters only on the GUID-verified fixture pool; run and
  independently observe a fresh clean scrub with healthy zero-error members and
  intact sentinel content.
- `fixture-restored`: remove only the GUID-verified child filesystem/mount,
  verify absence, original complete raw properties and unchanged pool/import
  state; require the original empty injector list.

Abort dependent cases after any failed child. Each scan convergence wait uses a
20-second child context inside the existing two-minute parent. No assertion,
product limit or original deadline is weakened.

## Primary-source controls and cleanup

FreeBSD base installs `zinject`; no extra package or runtime dependency is added.
Pinned source selects one vdev GUID with `-d`; `-e corrupt -T read -f 100`
flips a bit in a read buffer. It leaves backing files untouched by the test and
lets real ZFS checksum detection/repair/status execute. The kernel holds the pool
while a handler exists, so cleanup must remove the handler before storage cleanup.

- [FreeBSD base build](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/cddl/usr.bin/zinject/Makefile).
- [15.1 injector command](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/cmd/zinject/zinject.c)
  and [GUID translation](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/cmd/zinject/translate.c).
- [14.5 injector command](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/cmd/zinject/zinject.c).
- [15.1 kernel injector](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/module/zfs/zio_inject.c)
  and [14.5 kernel injector](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/module/zfs/zio_inject.c).

Arm cleanup before registration, including uncertain command failure. Cleanup
uses an independent fifteen-second context, clears only this fixture's known ID,
requires the empty injector list, then verifies child GUID before destroying it.
A lost/unknown handler acknowledgement with remaining handlers makes the run
unqualified and requires discarding the guest; never issue clear-all. The parent
verifies selected pool GUID and exact vnode backing before destruction/detach,
verifies absence and removes all fixtures. Guest/base/runtime/host/process cleanup
remains mandatory. No fault/control applies to the developer Mac or operator NAS.

```sh
PR_TITLE='test(storage): qualify native checksum findings' make review
.tools/venv/bin/python3 tools/local-qualification.py job-status 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py job-status 14.5 --run
make lab-status
```

Keep all existing storage/native-source product code/tests, prior health/scan
fixtures, shared tools/checkers/selectors/supervisor/Makefile and 54 historical
M2 manifests unchanged. Focused review selects the command package, which has no
portable compiler-listed fuzz target. Advancing progress, unrecoverable errors,
full scan formats, attribution/executor/recovery, protocol continuity and complete
M2/hardware/client acceptance remain separate. Actions/services stay disabled.

## Source-bound results

Signed source `8078e1d0c694a0384b5183ae33ea95fddc6a45fb` passed focused review: `make check lint test secrets`,
122 tooling tests and command-package race checks. The selected package has no
compiler-listed portable fuzz targets; unrelated prior evidence was not rerun or
relabeled. Both releases independently rebuilt the identical binary
`27cd25247124cceefae3d2c483835a7e384d39432b01580875655c45a4c5144b`
and passed all eight new children/group, eight scan children, six health children,
fifteen assessment children and 21 original status assertions.

- FreeBSD 15.1: native parent 33.54 seconds; new checksum group 5.99 seconds.
- FreeBSD 14.5: native parent 31.03 seconds; new checksum group 5.44 seconds.

The [manifest](checksum-findings-manifest.json) records pinned injector provenance,
actual tool/source/binary/image/report/log hashes and preservation/teardown. Actual
member checksum findings coexisted with completed scrub and zero final scan/data
errors; sentinel content remained intact. The acknowledged handler was removed,
the native list returned to empty and a fresh clean scrub restored healthy zero
member counters. Child GUID/absence and original raw properties were verified.
All owned storage/metadata/executable/VM files and process groups were removed;
base/runtime preservation and host teardown passed. No active or unverified owned
session remains. Product code, existing health/scan fixtures, shared tools and all
54 prior M2 manifests remain byte-identical. Every original assertion/deadline is
preserved. No Actions or persistent services were enabled. Advancing progress,
unrecoverable errors, full formats, attribution/executor/recovery and complete
M2/hardware/client acceptance remain open.
