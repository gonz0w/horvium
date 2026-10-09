# Native execution journal file persistence prerequisite

Status: Implemented prerequisite; qualification and maintainer review pending; runtime denied.
Date: 2026-10-09. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted mechanism](../../milestones/M2/property-recovery-mechanism.md),
[portable core](execution-journal.md), [native ledger census](native-execution-census.md).

PR 110 accepted the native ledger census at merged source
`3d9c5c49d8717ff4f8fd19c1a68a048d5244f415`. This component adds private FreeBSD
journal file persistence and an original in-memory publication receipt. It is
one prerequisite of M2-STATE-001/M2-RUNTIME-001/M2-RECOVERY-001. The original
registered dual-lease owner, persistent binding production, startup/global gates,
reconciliation and actual backend/supervisor composition remain required.

## Private domain and observation

The reviewed relative filename is `execution.json`; exclusive staging names are
`.execution-<32 lowercase hex digits>.tmp`. No installation path, exported API,
config schema, service or HTTP/CLI operation is introduced. The journal domain is
separate from the ownership/job ledger namespace and its exports. Its file holds
only the existing bounded version-1 codec, capped at 64 KiB. At most three names
are collected: the original empty `writer.lock`, optional journal and one staging
entry. Retained staging, unknown entries and excess names deny use. Only the
common publisher can exclude its independently inode-verified temporary.

Opening requires an already provisioned root-private directory and **pre-existing**
lock. It reuses the qualified ancestor, descriptor, kernel effective UID, mode,
ACL, no-follow, single-link, local writable UFS/ZFS and lifetime nonblocking lock
checks. The existing public ledger opener retains its original first-lock creation
behavior; the private journal opener never creates or repairs a lock. There is
no production commissioning workflow in this prerequisite. Disposable tests create
fixture locks explicitly and use synthetic installation/object/backing bindings.
Those values are metadata, not provenance or proof of independent restore scope.

Each read independently enumerates the entire bounded domain, reads through the
accepted actual-object reader/decoder, and verifies record/directory/lock metadata
again before returning. Errors return zero metadata. Nil, inert, copied, closed,
unsafe and non-root handles deny; a copied close cannot close the original.
Restoring an ACL permits observation, while previously detected uncertainty keeps
writes retired. Closing the handle releases descriptors; it does not publish a
closed lifecycle or establish executor quiescence.

## Original durable acknowledgements

Explicit private initialization requires an empty valid domain and generation 1,
active owner, idle metadata/native domains and no restore barrier. Opening does
not initialize a journal. Existing readable journals, including active records,
remain observation-only after reopening: serialized bytes cannot reconstruct the
original in-memory receipt or authorize overwriting initial state.

Only the original successful initializer can propose a strict single generation
successor. Before publication it must observe the exact previously acknowledged
bytes and full actual file metadata, including inode identity. Canonical typed
successor validation preserves bindings and independently pending domains; gaps,
replays, closed resurrection and ordinary restore rearming deny before effects.
This storage primitive validates metadata transitions; it does not prove census
interpretation, native completion, startup eligibility or permission to clear holds.

The component reuses the existing exclusive temporary/write/file-sync/close/
predecessor/whole-namespace/rename/directory-sync pipeline. Existing ledger
publishers use the unchanged wrapper. An additional private acknowledgement
callback verifies the actual published inode, exact canonical bytes and complete
journal domain **after** directory sync. A final independent observation after the
callback must still match before the volatile receipt advances. Every publication
failure retires that store for writes. Lost acknowledgement or drift after rename
is uncertain; visible readable metadata never substitutes for durability. There
is no retry, destination rollback, orphan deletion, restore, rearm or hold release.

## Tests and required qualification

Failing-first portable tests cover namespace/initial boundaries and bounded fuzzing.
Native tests require 65 store assertions, 74 actual process-death assertions,
eighteen additional POSIX/NFSv4 ACL and credential-dropped read-grant assertions,
six actual UFS full/read-only controls and one actual softupdates refusal. The
fixture sequence publishes journal metadata-pending, a real ownership record,
observes the actual complete after-census and publishes confirmation. It also
preserves native-pending while confirming a subsequent ownership update. This is
an explicit fixture sequence; production publishers are not integrated here.

Failure controls cover write, file sync, close, verify, rename, directory sync,
cleanup, lost acknowledgement, post-acknowledgement drift, late predecessor and
lock replacement. Eight exact-binary child barriers kill and reap the actual
writer before staging, after staging/write/file sync, before/after rename, after
directory sync and after return. Reacquisition preserves the lock and observed
old/new record, preserves retained orphan files and never reconstructs a writer.
These are warm process-death observations, not power-loss, reboot, rollback,
active ZFS control/kernel interruption or integrated recovery qualification.

Run both pinned disposable FreeBSD/amd64 guests serially, requiring all 72 selected
parents, 1,022 native markers and 1,897 codec children in the affected state corpus.
The shared checker maintenance selects one fresh full portable review; later
prose/evidence can reuse its exact deep inputs while rerunning all baseline checks.
Historical manifests retain their original bytes and source. Verify owned guest,
QEMU and development process teardown before reporting completion. Production
support, persistent bindings, provisioning, restore/startup/global owner, integrated
native effects and M2/release acceptance remain pending.
