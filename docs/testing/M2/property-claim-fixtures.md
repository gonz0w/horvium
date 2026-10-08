# M2 separate native claim qualification fixture

Date: 2026-10-08. Status: implementation; native qualification pending.
Authority: [approved Test Design](../../milestones/M2/TDD.md),
[EDD](../../milestones/M2/EDD.md), [issue 19](https://github.com/gonz0w/horvium/issues/19).
This is fixture/checker maintenance for the existing read-only observer contract;
no product behavior or execution authority changes.

## Problem and bounded fixture layout

PR 84 merged the original claim observer with full pool gates outstanding. Its
new eight native children ran inside the historical pool collector's five-minute
context. Two original attempts timed out during old tests. A retry at merged
source `41140a8061267a000bd321b5c5f4f536954cfb25` with no unrelated QEMU guests observed
at start also exceeded that unchanged deadline, during the old control group.
The new claim group passed in 20.84 seconds, and all postcondition cases passed.
That attempt remains unqualified. Host contention cannot be assumed as the sole
cause; fixture accumulation consumes the existing deadline budget.

Keep `TestNativePoolPropertyCollection`, its five-minute context, all 88 named
assertions and every old subgroup. Remove only the new claim call from its
system fixture. `TestNativePropertyClaimCollection` has its own 135-second
context and a fresh root-excluded enrolled mirror supplied by the same declared
five-disk setup. It returns from setup before the old enrollment/system scenarios.
The historical setup path retains its original operations and assertions.
Both actual native parents are selected by the existing checksum-bound prebuilt
suite and run serially, without parallel fixture use or new opt-in flags.

The claim fixture retains all eight actual backing/queued/applying/foreign/closed/
reopened/preservation/teardown children unchanged. After it destroys the independent
metadata pool, the outer GUID-bound data mirror and private lease/directory must
also be removed. Actual mount and swap presentations must match their original
values. The new parent emits exactly one `CLAIM_PROPERTY_COLLECTION_RESULT
fixture_teardown=pass` only after this check. Guest root backing settings remain
unchanged. Every storage device is one of the existing disposable snapshots.

## Complete qualification and test-first checks

The host checker still requires the original once-only collector parent and all
88 exact assertions. It now also requires the new once-only claim parent, group,
all eight exact child names, and outer teardown result. Missing, unknown, repeated,
failed or skipped children and missing/failed/repeated teardown reject. Nested
failure/skip lines are denied. Host backing/base/runtime/group cleanup and
checksum-bound artifacts remain separate mandatory checks.

Two failing behavioral tooling tests showed that the old checker accepted legacy
output without claim execution and ignored absent/repeated/failed/skipped new
children. After implementation, the checker and local runner regressions pass.
These synthetic logs test the validator; they are not native evidence. The runner
tests use a complete synthetic fixture record and still require host device
teardown. The selector test binds both actual native parents and the retained
five-minute/new 135-second contexts.

Require both pinned disposable FreeBSD releases, run serially after portable
review completes. Because shared qualification tools change, the existing review
planner requires its full portable mode; this is a verification-input change,
not an expansion of product scope. Preserve all old manifests and product bytes.
No historical evidence is relabeled, and this change does not accept M2 or a
runtime backend. Earlier state-reader/race/fuzz results retain their own sources.

```sh
PR_TITLE='test(storage): isolate native claim qualification fixtures' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Actual native results, artifact/report hashes, the unqualified merged-source retry
and complete owned teardown will be recorded separately before review. Root
backend/eligibility, executor/recovery clearance, interruptions, reservation release
and authenticated operator workflows remain open.
