# Explicit storage commissioning and clean-startup contract

Status: design admission proposal, 2026-10-20. Owner: Cam (`@gonz0w`).
Refs [runtime gate #139](https://github.com/gonz0w/horvium/issues/139) and
[policy lifecycle #305](https://github.com/gonz0w/horvium/issues/305).
Authority: [PRD](PRD.md), [EDD](EDD.md), [Test Design](TDD.md),
[runtime delivery](property-runtime.md), [recovery mechanism](property-recovery-mechanism.md)
and [ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
The [named acceptance plan](storage-control-test-design.md) is part of this proposal.

This completes the operator selection, ownership and failure boundary for the
existing private commissioning and policy-only startup mechanisms. Merge admits
this design, not runtime registration or native qualification. The [complete source consumer](../../testing/M2/storage-control.md) is now
implemented privately; no command below is registered. The exact-key M1 configuration, `serve` and `helper`
remain unchanged and read-only. No service installation or automatic startup is
introduced. Existing failed, skipped and exhausted attempts remain unchanged;
in particular this proposal does not authorize another #305 experiment or amend
the observer/containment work in draft #415.

## Explicit foreground selection

Propose one new OS-root-only foreground interface:

```text
horvium storage-control <commission-fresh|start-closed|inspect> --config <absolute-file>
```

Exactly one action and one explicit config are required. Reject duplicate flags,
positional additions, abbreviated/unknown actions, relative or unclean paths,
environment/home discovery, endpoint/identity overrides and any force, retry,
repair, reset, import or enable-writes option before opening a domain. Application
authentication, membership in the management socket group and inherited root
descriptors do not substitute for actual OS-root admission. Require real and
effective UID 0, checked again at original admission and retained transitions.

The separate root selection file is UTF-8 JSON, 1..16,384 bytes, with exactly
four case-sensitive decoded keys, each present once:

| Key | Exact value |
|---|---|
| `format` | `horvium-storage-control` |
| `version` | Unsigned decimal integer token `1`; no quoted, signed, exponent or fraction form |
| `journal_directory` | Explicit canonical absolute journal leaf path |
| `ledger_directory` | Explicit canonical absolute storage-ledger leaf path |

Reject malformed UTF-8/BOM, duplicate decoded aliases, null, wrong types,
unknown fields, nesting, arrays, trailing documents and control bytes. Paths use
the existing state-directory grammar: valid UTF-8, at most 4,096 bytes, absolute,
non-root, already clean, with no control/DEL characters. No expansion or
normalization grants admission. Equal or component-nested domains reject.
Lexical distinction does not prove native object or restore independence.

This is a proposed separate format, not another interpretation of M1 v1. It
contains no credential, policy, pool selection, boot ID, observed inventory,
installation identity, timeout override, copied receipt or readiness flag.
The action is selected at invocation, never inferred from file existence.

Read the selection from a retained no-follow descriptor: regular, root-owned,
single-link, mode 0600, below protected root-owned ancestors. Validate the native
ACL/filesystem profile and exact object/path correspondence before and after
reading; unsafe or unsupported custody rejects. Keep the original descriptor
and binding through the selected finite operation or foreground lifetime.
Replacement, metadata/content drift or root loss retires the owner; do not
reload configuration inside that lifetime. The selection must be outside both
domains and their ledger-only restore scope. The M1 credential directory is not
a storage ledger or execution journal and cannot supply either target.

## State and action matrix

| Selected action / observed state | Required result |
|---|---|
| `commission-fresh`; both leaf names absent under protected existing parents | Eligible only for exclusive fresh creation after full native parent/backing/system checks. Neither an empty existing directory nor a precreated lock is fresh. |
| `commission-fresh`; either leaf already exists, including an empty directory | Refuse before creation. Never fall through to startup, adopt existing files, repair permissions or replace a lock. |
| `start-closed`; both complete original domains and locks exist | Review the exact closed journal, complete census, bindings and current native facts. No creation or initialization fallback. |
| `start-closed`; either half missing, partial, newer, corrupt, full, replaced or inconsistent | Refuse, preserve all visible objects and report unavailable or recovery-required as below. |
| Clean empty ledger | May use the existing empty-history startup mechanism only after the assembled consumer's applicable qualification. |
| Member-bearing policies and zero jobs | Remains blocked until the complete same-candidate #305 both-release lifecycle/fault gate passes. No copied policy list, subset, diagnostic success or portable test waives that gate. |
| Any job in either format/state, including terminal or another-pool records | Refuse this startup/unused-close path. Preserve every job, policy and pool hold. |
| Active prior journal, pending metadata/native transition, disarmed restore barrier or unknown provenance | Recovery-required. No same-boot adoption, automatic changed-boot clearance, replay, reset or inferred clean close. |
| `inspect` | Requires both existing protected namespaces/locks; read-only complete bounded inspection through original journal-first leases. Missing halves are unavailable. Never commission, activate, finish, enroll or mint recovery/launch authority. Busy is not healthy or proof of a live writer. |

Paths select objects; they are never identity. Retain actual protected parents,
directories, mounts, lock objects and persistent backing bindings. Require the
accepted writable local UFS/ZFS/ACL profile and complete system/owned-pool
exclusions. Journal and ledger may share physical system storage, but must have
independent qualified restore scope. A distinct ZFS dataset may supply that
boundary; UFS needs the accepted ledger-file restoration procedure. A whole
filesystem rollback replacing both halves is not a ledger-only restore.

Fresh creation follows the accepted producer: prove both names absent before
the first effect; create journal then ledger with exclusive 0700 directories,
0600 locks, original leases and all required syncs. Any failure preserves every
visible partial object and consumes the entered attempt. No deletion, re-seeding,
partial-install resume or another path can turn it into a fresh attempt.

## Original lifetime and observable completion

Reserve the existing single-use process-wide root registration before entering
commissioning or registered startup. Construction failures after entry, successful
unused close and release do not reset it. Preserve the original capability chain,
journal-before-ledger lease order and existing owner/pair/journal/ledger mutex
order. Copies, decoded records, reopened descriptors, new contexts and success
booleans cannot reconstruct authority. Unsupported platforms return unavailable.

The registration is process-wide, not a cross-host or trusted-root fence. These
objects cannot detect a different root process deliberately selecting another
configuration. Claims about a shared registered writer/global launch owner still
require #139's later complete assembly and qualification.

`commission-fresh` is finite: compose the accepted fixed native original activation,
confirm its unused observation, publish its exact durable unused closed successor
and release originals. Report completion only after acknowledged finish and
successful release. The resulting protected domains contain no ownership
enrollment or admitted job; their existence alone does not authorize later writes.

`start-closed` is foreground: use the fixed policy-aware root consumer, transfer
the original reviewed startup once into its acknowledged active lifetime, and
retain both leases until shutdown. It emits no output while holding those leases
and provides no live readiness report or native launch permit. No worker, backend
or write dispatcher is attached by this contract. The first SIGINT/SIGTERM or
foreground cancellation initiates shutdown once. Stop admission, latch that
original attempt and give `FinishIdle` one separate cleanup context capped at the
existing 90-second operation limit; do not pass the already-cancelled foreground
context to its pre-entry guard. Later signals/cancellation cannot reset the latch,
retry finish or extend that same deadline. `FinishIdle` is eligible only for this
original never-used lifetime; it is not worker draining or recovery.
Every entered finish outcome retires observe/finish authority. `Close` attempts
original release once and preserves the result; release alone publishes no
closed marker. Always attempt original release after finish, including errors.
A failed finish, cleanup deadline or release suppresses clean completion and
preserves visible active/uncertain history. Abrupt process death supplies no
closed acknowledgement. No emergency clean marker or retry path is introduced.

Retain existing 90-second per-operation contexts across the complete bounded
policy/census/native operation, including waits. Do not renew time per policy,
add retries, widen existing internal deadlines or claim supported capacity from
a small fixture. Foreground lifetime is distinct from each bounded operation.
Preserve generation headroom for close and all existing record/count/64-MiB
ledger and 64-KiB journal bounds. Context cancellation before entry has no effect;
after entry, visible state and original retirement rules govern the result.

## Output and failure contract

Bound each structured result to 4,096 UTF-8 bytes and one LF. Exact JSON keys
are `format` (`horvium-storage-control-result`), integer `version` (1), `action`
(the selected action), `state` and `writes_enabled` (always false in this slice).
Allowed states are `closed` and `recovery-required`.
They describe this observation, not a serialized capability. `inspect` does not
observe absent namespaces or create a freshness receipt. Active/pending/unknown
history is recovery-required, never an assertion that a process is alive.

Stage the complete result privately, finish any owned lifecycle and release both
original leases before writing stdout or stderr. Inspection likewise releases
its complete original read-only observation before delivery. This avoids holding
storage ownership behind a blocked output sink. The 4-KiB bound limits output
bytes, not an arbitrary destination's syscall duration; no wall-time delivery
guarantee or unowned asynchronous writer is implied. Cancellation or output
failure cannot delay or repeat original finish/release or turn an undelivered
result into permission to repeat the action. No success is claimed before output
delivery itself finishes. Future implementation tests must include a blocked
writer and prove original cleanup has already completed before it is called.

Finite success emits one result. Foreground startup emits only one final closed
result after confirmed orderly shutdown and release. It emits no startup/polling
output. No process inventory, raw journal/policy, pool list or unbounded log stream
is introduced. The existing inventory and independent file daemons remain
independent; this interface grants no management-session authority.

Successful fresh commissioning and orderly startup shutdown report only `closed`.
Complete inspection
reports `closed` only for that exact definite state; a complete
inspection of active, pending or unsafe history emits `recovery-required` and exits
3. Missing or uninspectable namespaces emit no structured result. A foreground
failure emits no fabricated closed result.

Exit 0 requires a complete requested observation or confirmed finite completion.
Exit 2 uses one fixed stderr class (`invalid-input`, `not-root`, `busy` or
`unavailable`) for a definite refusal before any state-changing entry. A cancelled
pre-entry attempt is `unavailable`. Exit 3 is `recovery-required` for an entered
attempt with incomplete acknowledgement/cleanup, or unsafe retained provenance.
Never print raw errors, arguments/config paths, SQL, credentials, native command
output or host inventory. Output failure after an effect is an undelivered result,
not permission to replay; preserve originals and report recovery-required.

No automatic retry follows timeout, lock loss, changed boot, lost acknowledgement
or process restart. Do not signal a recorded PID. Operator recovery preserves
objects and evidence and uses a separately reviewed procedure; this contract
supplies no cleanup/reset command. Rolling back the software preserves both
domains; an older reader must reject unsupported state rather than rewrite it.

## Admission and delivery boundary

This newly reviewed contract permits later implementation of its byte parser and
real original consumer in one coherent source outcome. It is not a reason to add
an unused helper-only PR. Native UID/ACL/descriptor/lease/backing/publication and
both-release lifecycle evidence is required before registering these commands.
The named test plan also requires unchanged M1 configuration and read-only views.

#139 remains open for complete startup/global-permit/backend composition. #305
remains open for its exact native lifecycle/fault vectors. Root worker admission,
global native launch, retained-job recovery/hold release, restoration/rebinding,
unclean or changed-boot resolution, pool/filesystem adoption/creation, service
installation and shared mutation UI/CLI remain outside this contract. No current
runtime write denial, native profile, guard, cache or historical receipt changes.
