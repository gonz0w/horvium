# ADR 0007: root-owned bounded storage job records

Status: Accepted. Date: 2026-10-04. Owner: Cam (`@gonz0w`).
Authority: maintainer merge of [PR 20](https://github.com/gonz0w/horvium/pull/20),
source `753f601960529cd17596d0ffc628afe38d041c82`, on 2026-10-04.
See the approved [M2 design](../milestones/M2/EDD.md); native gates remain required.
[Test Design](../milestones/M2/TDD.md), [primary evidence](../research/M2/findings.md).

## Context and decision

A native change can complete after the client disconnects or the helper dies.
M1's volatile cache and sessions cannot represent that outcome. Use strict
versioned JSON ownership/job records on system storage, owned/written only by
the root helper. The unprivileged service reads bounded typed projections. Use
Go's standard library and the existing approved FreeBSD syscall adapter; introduce
no embedded database, external worker or distributed infrastructure now.

One helper holds a stable lifetime advisory lock. Sync same-directory temporary
records, rename them, and sync the directory; qualify each failure and native
filesystem support. Durable intent precedes launch, and independently verified
postconditions plus durable outcome precede success. A job's durable intent is
the recovery anchor for non-atomic registry/job updates. Unknown/corrupt/newer
records disable writes instead of being reset. Checksums detect damage, not root
tampering or external ZFS transactions.

Bound pending work, active launches, per-pool reservations, record/history bytes
and terminal retention. Never evict uncertain jobs or ownership to reclaim space.
Submission identifiers/tombstones prevent retries after a lost acknowledgement;
after the replay window, an expired plan cannot authorize a fresh mutation.
Kernel scans are observed after launch; restart does not replay them or promise
to cancel a resilver. Ambiguity remains reconcile-required until resolved against
actual GUIDs and attributable state. Backed-up metadata is reconciled before use.

## Alternatives and consequences

Memory-only jobs lose destructive intent on restart. A plain unsynced log or
rename alone does not establish the durability contract. SQLite could
simplify indexing/cross-record transactions, but introduces a new dependency and
does not make native ZFS changes atomic with database writes. Adopt it only if
measured scale or recovery complexity justifies an amended design/ADR. A network
job queue is outside this single-host requirement.

The file-store design requires tested multi-record recovery, bounded enumeration,
compaction/tombstones and export/restore barriers. Pause metadata transitions,
sync/capture a sequence-consistent set with file hashes, then resume; kernel work
can continue while its last durable intent is exported. Restoration still requires
fresh policy/identity reconciliation. It is not a file-data backup.

Actual directory-sync, state-filesystem failure and abrupt restart tests are
acceptance gates. QEMU power-cut tests do not qualify physical cache/controller
durability. Revisit for proven ledger/index performance problems, package-state
migrations, backup/config restoration or multi-administrator audit requirements.
