# M2 original-claim property successor acknowledgement

Date: 2026-10-08. Status: metadata subset qualified; maintainer review pending; runtime writes disabled.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-JOB-001, M2-STATE-001 and M2-RECOVERY-001 evidence.

## Exact acknowledged metadata chain

`PropertyStateClaim.PublishPropertyJobSuccessor` supplies the execution sequence's
observing/outcome metadata prerequisite through the same original opaque root
claim. It invokes no native command and implements no complete execution backend.
Only the original applying publisher's successful sync and exact independent
whole-census readback populate a volatile canonical-record hash/history revision.
Raw publication, decoded records, visible but unacknowledged applying metadata,
copied state handles and reopened leases cannot supply that lineage. The immutable
original queued receipt remains unchanged; opaque claim copies share one identity.

Require actual root EUID, the original open lease/active claim, a complete strict
mixed census, selected ownership presence and exact independently read current
record matching the private acknowledged hash/revision. Preserve the whole intent
and prior history. Only these execution-sequence edges are permitted:

| Acknowledged predecessor | Exact appended event |
|---|---|
| Applying (revision 2) | Observing or reconcile-required |
| Observing (revision 3) | Succeeded or reconcile-required |

No caller supplies a destination or expected revision. Deny skipped observing,
queued/cancelled/failed candidates, rewritten history/intent, raw advancement,
resolution from reconciliation and terminal replay. Other nonterminal jobs on any
pool and retained jobs on the selected pool still block; unrelated terminal records
remain inspectable. Success releases no active claim or selected-pool hold.

## Durability and uncertainty

Both claim publishers share one descriptor-relative transaction. Generic publishers,
readers and backing guards remain unchanged. Bind every ordered name and actual
presentation byte before staging and during final checks, excluding only the
independently verified owned temporary. Require exact predecessor, whole-temporary
capacity, ACL/inode checks, file sync, rename and directory sync. Supported backing
remains ZFS or UFS without softupdates/read-only flags. Sequential checks/advisory
locking cannot fence a competing OS-root actor.

Consume a valid successor attempt under the original mutex before staging. Initial
candidate/census denials perform no transaction. Failed valid attempts cannot retry,
even after successful cleanup. Independently reread the exact successor and
projected whole census after sync before returning an owned acknowledgement or
advancing the receipt. Uncertain cleanup/sync/readback blocks publication; visible
metadata is never rolled back or reconstructed as a successful acknowledgement.
The caller retains its last verified record. Exact acknowledgement permits only
the next allowed edge.

The complete original backend must independently authorize actual control/
quiescence, fresh native observation and terminal outcome. Caller-supplied digests
are validated metadata, not native evidence. This adds no eligibility, clock/
session authority, global serialization, recovered-state clearance, worker/service,
transport/schema, terminal resolution or reservation release. Existing claim
readiness/readers remain limited to queued/applying and acquire no later launch
or observation exception.

## Test-first and native gates

Portable tests cover the four allowed edges, missing/foreign/changed acknowledged
hashes/revisions, queued/changed current records, intent/history drift, missing
records, pending/selected terminal holds, unrelated terminal records and outcome
replay. Non-FreeBSD calls return zero/unavailable. The initial absent-contract
compile failure also caught a fixture using a FreeBSD-only helper; that fixture
was corrected before behavioral execution.

The disposable submission parent adds 38 `claim-successor` children: full success
and both reconciliation edges; copy/16-call serialization; no-ack/lost-applying/
raw-advance/foreign/copied/closed/reopened denials; candidate/current/ownership/hold/
orphan drift; late policy/presentation/pending changes; capacity and publication
phase cuts; changed record/census readback; separate succeeded/reconciliation cuts.
Require exact surviving bytes, unchanged private acknowledgements on failure,
owned replies, stable writer-lock inode, no unexpected temporaries, retained active
claims and complete lease/directory teardown. The actual non-root credential child
must also exercise this method. These are actual metadata operations with synthetic
intents/policies; they establish no native eligibility or effect attribution.

Retain every historical state/backing/codec assertion, 19 live-state children,
23 applying children, original deadlines and all shared selectors/checkers/
supervisors. Run both pinned releases serially with identical artifacts and audit
all 38 new children once in addition to the unchanged shared checker. Pool observer/
control sources and readiness/reader logic remain unchanged, so their historical
proofs retain their actual sources without rerun. Focused portable review and
mandatory baseline checks are required; eligible documentation followups may reuse
exact-input deep evidence.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyClaim(Successor|Applying)'
PR_TITLE='feat(storage): acknowledge property successors through original claims' make review
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
make lab-status
```

See the [manifest](property-claim-successor-manifest.json) for source-bound results,
artifacts and teardown. Full original backend, native effects/death/reboot/recovery,
authenticated operator workflow, terminal resolution/hold release and assembled M2
acceptance remain separate gates.

## Source-bound results

Signed source `7af2f1cda24179f4e9f8ceebc4a20503dc6c9c25`, based on PR 87 merge
`412cf3a9bd49c435eb77dbd7664ed0ed8981a378`, passed both new portable parents,
all 18 lineage cases, prior applying tests and mandatory `make check lint test secrets`
with 125 tooling tests. Focused review passed 9 affected race packages and all
38 selected bounded fuzz targets in 195.271 seconds, with both supervisors cleaned.

Both pinned FreeBSD releases passed all 38 new children, actual UID denial,
23 applying/19 live-state children and every old 819 state/backing assertion,
67 parent and 1,876 codec child. Identical artifact
`19638da705a51331ef8fccf811b26baa7b52d85bfeb67143a89292595f41536b`
ran on both guests. Successor groups took 11.03/10.19 seconds on 15.1/14.5;
full qualification took 83.637/75.337 seconds. These are lab timings and real
metadata operations, with synthetic intent/policy/digest values and no native
outcome or hardware performance claim.

All 63 prior manifests and 120 protected shared/native observer/reader/publisher
inputs remain unchanged. Original ledger/readiness logic, claim readers, generic
publisher/backing checks, selectors/checkers/supervisors, old deadlines and native
pool/control sources are byte-identical; pool evidence retains its original source.
Both guests, owned metadata fixtures/backings, images/keys/variables/binaries and
process groups were removed or verified stopped. Base/runtime hashes were preserved;
lab status found no active/unverified owned sessions. The unexecuted development
cross-build was removed and is not native evidence. Actions remain disabled.
See the [manifest](property-claim-successor-manifest.json) for actual artifact/report/
log hashes, scope and remaining gates.
