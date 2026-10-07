# M2 foreground storage job-status diagnostic

Date: 2026-10-07. Status: implementing; native qualification pending.
Authority: [approved M2 design](../milestones/M2/EDD.md),
[qualification contract](../testing/M2/storage-job-status.md).

Use this diagnostic as OS root on FreeBSD to inspect retained storage jobs when
the management service or a data pool is unavailable. It opens only an existing,
root-provisioned storage ledger. It starts no helper, web listener, worker or file
service, and does not require a management token or read standard input.

```sh
horvium storage-jobs -state-dir /root/horvium-lab-ledger
```

The example is an operator-selected disposable lab directory, not a default or
installation instruction. Supply the actual protected storage ledger path: an
absolute clean non-root path, root-owned 0700 directory, protected ancestors and
safe private record/ACL objects. Do not point it at M1's credential state directory,
a data pool, an arbitrary backup copy or an active helper's ledger. A held writer
lock refuses the command; it never evicts a writer or removes the lock file.
FreeBSD base system storage remains the target; macOS/Linux refuse native access.

The existing lease opener may create and sync its empty private `writer.lock` on
first use. It provisions no directory and changes no job/policy record. Successful
inspection checks the complete bounded ownership/job namespace, reads one immutable
view, and closes the lease before writing one JSON document to standard output.
Normal completion, failure and cancellation close owned descriptors. The command
has a 45-second context bound; kernel filesystem calls are not preemptible by that
context, and a blocked output writer has no artificial asynchronous timeout.

## Interpret the result

The version 1 `storage-job-status` diagnostic includes all retained jobs in lexical
ID order, up to the accepted 64 nonterminal plus 1,000 terminal store limits. It
traverses pages of 64 internally; operators do not combine cursors from different
invocations. The complete output is staged with a 1 MiB ceiling. A complete empty
store reports `jobs: []`. Validation failures produce no status document and exit
nonzero; an output failure may leave delivered bytes and also exits nonzero.

`source: retained-metadata`, `recovery_verified: false` and `holds_retained: true`
make the scope explicit. Each row contains job/submission/pool identifiers, the
`last_durable` event and a separate `restart_state`/`restart_reason`:

| Last durable event | Conservative restart presentation |
| --- | --- |
| queued | reconcile-required / review-expired |
| applying or observing | reconcile-required / executor-unverified |
| reconcile-required | Retained unresolved reason |
| succeeded, failed or cancelled | Retained terminal label and reason; holds remain |

A terminal label or stored observation digest is historical metadata. It does not
verify current native effects, executor absence, safe restoration, cancellation
or releasable reservations. Never replay a command because this diagnostic shows
an interrupted job. Full plans, property baselines, policy/session digests and
request credentials are excluded from output. Identifiers remain operator data;
keep real diagnostic output out of the repository and exported test artifacts.

## Failure and recovery limits

Unsafe/corrupt/unknown/newer/full metadata, orphan temporaries or an active writer
refuse inspection without returning partial status. Errors are fixed classes,
with no raw filenames, record bytes or argument echo. Preserve the ledger and its
uncertain objects; this command does not repair, delete, reset or reconcile them.
A missing native pool does not prevent reading its retained job. Ownership bytes,
job history and pool holds remain unchanged.

This is an OS-root diagnostic. Authenticated management UI/CLI job parity, native
reconciliation, policy provisioning, mutation transport, cancellation/release,
metadata restoration and the remaining M2 operations still need implementation
and qualification. The command cannot enable writes or count as M2 acceptance.
