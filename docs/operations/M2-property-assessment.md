# M2 foreground property-job assessment

Date: 2026-10-07. Status: implemented; qualification pending.
Authority: [approved M2 design](../milestones/M2/EDD.md),
[qualification contract](../testing/M2/property-assessment-diagnostic.md).

As OS root on FreeBSD, inspect one retained property job against current native
policy/property observations without starting a service or changing storage:

```sh
horvium storage-assess -state-dir /root/horvium-lab-ledger -job-id <canonical-job-id>
```

Choose the existing protected storage ledger and a 64-character lowercase hex
job ID from `storage-jobs`. The example is an explicit disposable lab path, not a
provisioning default. The ledger requires the accepted root/ancestor/ACL/object
checks; a held writer refuses access. Do not select credential state, an arbitrary
backup copy, a data pool or an active helper's ledger. macOS/Linux refuse native
access. The command reads no stdin/token and opens no listener/helper/file service.
Initial creation/sync of the existing opener's empty `writer.lock` remains possible;
no policy/job record is written and the stable lock is never removed.

## Read the result

One bounded version 1 JSON document has kind `storage-property-assessment` and
source `retained-metadata-and-native-properties`. The `job` row retains the same
identifiers, `last_durable` event and separate `restart_state`/`restart_reason`
used by `storage-jobs`. The two fresh observations mean:

| baseline_matches | expected_effect_matches | Observed comparison |
| --- | --- | --- |
| true | false | The reviewed baseline matches |
| false | true | The reviewed expected effect matches |
| true | true | Both match, for example an already-matching selected value |
| false | false | Neither matches; valid context/policy differences can cause this |

Every successful result says `recovery_verified: false`, `executor_verified: false`
and `holds_retained: true`. A match cannot attribute a native executor, show it is
absent, validate restored metadata, acknowledge durable success or release a hold.
Terminal job labels remain historical even if fresh comparisons differ. Never
replay or repair a job because of these booleans. Full plans, intent/baselines,
policy/session/ledger fingerprints, native addresses/rows and credentials are
excluded. Output identifiers are still operator data; keep real results private.

The actual root-leased source validates the entire ledger twice around independent
native collection. Ordinary drift/unsafe/missing/unavailable/cancelled input returns
no result. A missing pool differs from `storage-jobs`: retained status can still
be shown there, but assessment needs fresh native observations. Equal bytes and
repeat projections are not atomic fencing or restored-state provenance.

## Lifetime and errors

Validate flags, path and job ID before opening the ledger. Release the original
lease before stdout delivery. Stage the complete document under 4 KiB. Invalid
arguments return a fixed input error; acquisition/assessment failure returns
`storage assessment unavailable`, without paths, raw native errors or arguments.
The command exits nonzero on validation, collection, cancellation or output error;
output failure cannot retract already delivered bytes.

The process lifetime is 145 seconds, including the source's unchanged 135-second
bound and output. SIGINT/SIGTERM/deadline cancellation exits nonzero even when
stdout blocks; normal completion disarms the cancellation exit. Kernel filesystem
calls need not be preemptible, and this is not a rollback promise. Owned native
commands use their existing bounded cancellation/reaping behavior. No persistent
runner or service is installed.

This diagnostic is a partial operator workflow. Authenticated UI/CLI parity,
native eligibility/admission/execution, reconciliation resolution, restored-state
validation, durable outcomes and full M2 qualification remain open.
