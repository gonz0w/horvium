# M2 original-claim applying acknowledgement

Date: 2026-10-08. Status: metadata subset qualified; maintainer review pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-JOB-001, M2-STATE-001 and M2-RECOVERY-001 evidence.

## Exact metadata boundary

The portable execution sequence requires an exact durable applying acknowledgement
before independently refreshed launch checks. This subset supplies that metadata
prerequisite through the original opaque `PropertyStateClaim`; it implements no
combined execution backend and promotes no observation to eligibility.

`PublishApplyingPropertyJob` accepts one typed whole successor. Only the original
open root lease, active claim and new post-sync admission receipt can publish it.
Require actual root EUID, a complete strict census, the independently read exact
queued predecessor, selected ownership presence, unchanged whole intent/history,
and exactly one applying append. Other pending jobs and selected-pool retained
holds still deny. Decoded metadata, reopened leases and copied state handles
cannot replace this provenance. No job identifier or expected revision can be
substituted by the caller; the destination derives from the original claim and
the only expected revision is one.

Under the original mutex, retain one publication attempt across copies of the
claim. Consume it after candidate/census validation and before staging; a failed
valid attempt cannot automatically retry even after successful cleanup. Invalid
candidates and failed initial metadata inspection create no publication attempt.
Existing read-only claim operations keep their semantics; an applying label or
attempt flag never proves launch, quiescence, attribution or recovery clearance.

Use the unchanged descriptor-relative publisher and supported metadata backing:
ZFS or UFS without soft updates/read-only flags, actual private objects/ACLs,
whole-temporary capacity, exact predecessor/inode checks, file sync, rename and
parent-directory sync. Repeated strict censuses bind all ordered names and actual
presentation bytes, excluding only the transaction's verified temporary. Any
policy, job or presentation drift before rename denies. Advisory locking and
sequential checks cannot fence a competing OS-root actor.

After successful sync, independently reread through the original claim. Require
exact canonical successor bytes and the complete expected census, where only the
selected job presentation changed to its canonical applying encoding. Return an
owned acknowledgement only after this check. A failed sync/cleanup or lost/changed
readback returns zero and preserves publication uncertainty; a visible destination
is never rolled back or acknowledged as durable. The old queued receipt remains
immutable, and subsequent calls cannot acknowledge/replay the applying record.
No terminal outcome, cancellation, history repair, reservation release or eviction
is added; existing raw metadata publication retains its separate contract.

This primitive does not validate actual current session/clock/expiry, fresh native
eligibility, original coordinator handoff, executor/recovery clearance or global
launch serialization. Timestamps are validated metadata, not proof of a live clock.
The complete original backend must establish these independently around admission,
applying publication and refresh before any fixed native control. This method
starts no worker, listener, helper request or native storage command. Runtime
writes remain disabled, and no public transport/schema/dependency is introduced.

## Test-first and actual native gates

Portable tests initially failed for the absent publisher/validation contract.
They cover exact immutable successors, foreign/nil claims and current records,
missing/rewritten/late stages, complete holds and retained unrelated terminal jobs.
Non-FreeBSD calls return zero/unavailable. Later test-fixture identifiers were
corrected to distinct canonical values; those fixture mistakes are not product
or native failures.

Extend the existing opted-in submission parent with 23 `claim-applying` cases:
exact acknowledgement, shared-copy/16-call serialization, malformed/foreign/changed
intent and cancelled candidates, closed/copied/foreign leases, missing ownership,
other pending/selected terminal holds, orphan uncertainty, late policy/presentation/
pending drift, whole-temporary capacity, file-sync/rename/cleanup/directory-sync
cuts and changed post-sync readback. Actual credential-child denial exercises the
new method. Verify exact surviving bytes, no unexpected temporaries, unchanged
writer-lock inode, owned outputs and complete lease/directory cleanup. These are
real kernel metadata operations on the existing supported disposable anchors;
policy/intent selection is synthetic and establishes no native eligibility.

Retain every historical state/backing/codec assertion, all 19 live-state cases,
all original fixture deadlines, selectors, shared checkers and supervisors.
Both pinned FreeBSD releases must run serially using identical source-bound
artifacts. Audit all 23 children and the actual UID child in addition to the
unchanged shared checker. Previously qualified pool observer/control fixtures are
unchanged and are not rerun or relabeled. Portable affected-package race/fuzz and
mandatory baseline checks remain required; use exact-input reuse for subsequent
eligible prose/evidence updates rather than another deep suite.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyClaimApplying'
PR_TITLE='feat(storage): acknowledge applying metadata through original claims' make review
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
make lab-status
```

Source-bound qualification results and teardown are recorded in the
[manifest](property-claim-applying-manifest.json).
Full root backend, native effects/death/reboot/recovery, terminal outcomes/hold
release, authenticated operator workflows and assembled M2 acceptance remain open.

## Source-bound results

Signed source `80f3588a89abcf8484f45da06f37eca18bcfa6dd`, based on merged PR 85
source `1c8afa7ff49052a6db1ef997204eedd55246becd`, passed both new portable
parents/all 15 exact-successor cases and mandatory `make check lint test secrets`
with 124 tooling tests. Focused review passed 9 affected race packages and all
38 selected bounded fuzz targets in 208.828 seconds.

Both serial pinned FreeBSD releases passed all 23 actual applying cases, the actual
non-root credential child, all 19 live-state cases and every original 819
state/backing assertion, 67 parent and 1,876 codec child. The applying groups took
3.92/3.83 seconds on 15.1/14.5; the enclosing submission parents took
37.75/33.75 seconds. Identical checksum-bound artifacts ran on both guests.
Metadata operations/lease/UID/backing facts are actual; intent/policy selections
remain synthetic and qualify neither native eligibility nor launch authority.

All shared publisher/readers, selectors/checkers, supervisors, native pool/observer/
control sources/fixtures, schemas, dependencies, old limits and 61 prior M2
manifests remain byte-identical. The original admission logic changes only by
adding a private attempt field consumed solely by this new method. Previously
qualified pool evidence is retained under its actual source rather than rerun or
relabeled. No terminal-resolution or recovered-state claim follows.

Both guests, owned metadata mounts/backings, lease directories, images/keys/firmware,
executables and process groups were removed or verified stopped. Host base/runtime
hashes were preserved; `make lab-status` verified zero active/unverified owned
sessions. Actions remain disabled and no persistent service was enabled. The
unexecuted development cross-build was removed; it is not native qualification.
The [manifest](property-claim-applying-manifest.json) records actual source,
artifact/report/log hashes, scope and remaining gates.

## Subsequent metadata acknowledgement

The [original-claim successor contract](property-claim-successor.md) extends the
metadata chain after an exact applying acknowledgement. The shared transaction
now retains a private canonical-record hash/revision only after sync and exact
record/census readback. The original queued receipt, one applying attempt and
queued/applying readiness semantics remain unchanged. This extension permits no
native effect, terminal resolution or hold release; its own qualification is
separate from the historical applying evidence above.
