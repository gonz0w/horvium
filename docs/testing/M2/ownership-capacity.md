# M2 ownership publication capacity

Date: 2026-10-06. Status: implementing; native qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract and scope

`StateDirectory.PublishOwnership` now requires a complete validated
[ownership catalog](ownership-catalog.md) before creating its temporary record.
An unknown file, unsafe unrelated record, damaged/newer metadata or orphan
cannot be ignored just because the destination itself is safe. Refusal preserves
all existing objects; no eviction, repair, adoption or automatic retry occurs.
A recognized orphan returns `ErrStateUncertain` and blocks subsequent publication
on that lease even if another actor removes the name. Reads remain available;
closing/reopening a lease alone is not qualified reconciliation or recovery.
Unknown/damaged records return `ErrStateRecord`; unsafe objects return
`ErrStateDirectory`, ordinary drift `ErrDrift`, budget exhaustion `ErrLimit`.
Existing non-root, unsupported-platform, ACL and filesystem restrictions remain.

The held writer lease and helper mutex serialize cooperating inspection, budget
reservation, temporary publication and close. Fixed ceilings are 100,000 published
ownership records (the existing model bound), 64 KiB per encoded record and 64 MiB
logical metadata bytes. The byte reservation is **all current published bytes plus
the entire new encoded record**, including replacement: the old record remains
present until rename. Replacement consumes no extra published-record slot. Reject
creation at the count ceiling and reject bytes before creating a temporary.
The empty stable lock contributes zero logical bytes; directory/allocator blocks
are not estimated. Physical ENOSPC remains an independent native failure boundary.
These defaults are independent of CPU, RAM, hardware model or disk count.

After write/file-sync/close, verify the held lease, temporary inode/metadata and
original destination, then inspect the complete namespace again and repeat
revision/capacity checks before rename. Exclude only this publication's exact
internally generated temporary name after matching its written inode; public
inspection has no exclusion. Any other orphan remains uncertain. Recheck original
destination and own temporary again after collection. A new unrelated record that
consumes reserved capacity causes refusal. Pre-rename errors clean up only the
verified owned temporary, retaining their catalog safety category; cleanup failure
or post-rename directory-sync failure remains uncertain. Ordered durability and
stable lock semantics follow the [accepted publisher](ownership-publication.md).

This is ownership-only metadata publication, with no runtime/helper/HTTP/CLI
wiring, trusted enrollment, durable job admission or storage operation authority.
Future job/history/tombstone records must extend the namespace and share the total
metadata budget before they can be admitted. Jobs, restoration, orphan recovery,
abrupt-death/power-loss behavior and the full assembled store remain open gates.
Two bounded full scans are deliberate correctness prerequisites; no commercial
scale, populated workload, latency/resource or malicious-root fence claim follows.

## Behavioral and native gates

Failing-first portable capacity tests prove exact creation/count boundaries,
replacement old-plus-new accounting, revision conflict, invalid ceilings and
subtraction-before-addition overflow refusal. Transaction regressions retain
catalog error categories after cleanup. Evidence regressions refuse old 202-case
logs and every missing, duplicate, failed or skipped capacity assertion.

The state gate requires **234 named assertions**, nineteen parent tests (seven
native and twelve unit) and all 57 codec subtests on both declared FreeBSD targets.
The 32 added native assertions use newly owned private guest directories/files:
exact reduced creation/replacement limits; refusal before temp creation; old bytes/
inode and stable lock preservation; unrelated unknown, damaged, newer, wrong-pool,
mode, symlink, hardlink, FIFO and directory refusal; retained orphan bytes/latch;
real late unknown/orphan/count/byte controls; concurrent last-slot serialization;
readback, foreign preservation, lease close and fixture teardown. Private reduced
limits and phase seams cannot be selected through runtime requests or environment.
They prove the production accounting path, not a populated 64 MiB/100,000-record
scale workload. Existing owned UFS/ZFS ACL, actual ENOSPC and read-only fixture
assertions still run; no operator disks, datasets, credentials or history are used.

```sh
make test-focus PKG=./internal/storage TEST='^TestOwnershipPublication'
PR_TITLE='feat(storage): reserve ownership publication capacity' make review
python3.14 tools/local-qualification.py state 15.1 --run
python3.14 tools/local-qualification.py state 14.5 --run
make lab-status
```

Shared native selectors/evidence validation trigger one complete portable review.
Evidence-only documentation followups may reuse matching executed deep evidence
while rerunning baseline/source/title checks. Native qualification remains pending;
cross-compilation alone demonstrates no native behavior.
