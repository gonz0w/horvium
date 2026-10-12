# Strict deadline provenance consumer

Status: offline consumer component; tagged fixture, runner and native evidence pending.
Date: 2026-10-12. Owner: Cam (`@gonz0w`).
Refs [issue 341](https://github.com/gonz0w/horvium/issues/341),
[parent 305](https://github.com/gonz0w/horvium/issues/305) and
[broader gate 139](https://github.com/gonz0w/horvium/issues/139).
Authority: PR 336 adopted the [method and Test Design](../../research/M2/policy-startup-timeout-provenance.md),
refining the approved [M2 Test Design](../../milestones/M2/TDD.md),
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[retained-policy contract](../../milestones/M2/retained-policy-startup.md) and
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirements: consumer subsets of M2-POLICY-DEADLINE-PROVENANCE-003/004/006,
M2-RUNTIME-002, M2-STATE-001, M2-RECOVERY-001, M2-OWN-001, M2-DRIFT-001,
M2-PRIV-001. Synthetic transcripts do not satisfy complete native test IDs.

## Transcript contract

[The development-only consumer](../../../tools/check-policy-startup-provenance.py)
accepts one sanitized transcript, capped at 128 KiB. The future fixture's child
output remains capped at 16 KiB; these are separate boundaries. No raw errors,
version banners, arguments, inventories, stacks or undeclared lines are accepted.
A Go t.Log filename/line prefix is permitted only on fixed control-result rows.
The future runner must retain its private full log separately and extract this
canonical transcript without silently dropping unexpected lines inside the parent.

The prefix is `POLICY_STARTUP_PROVENANCE_`. Before the sole parent header,
`PREBUILT_SHA256` must match the independently supplied artifact checksum;
`PROFILE` is exactly `cpu_count=1 child_seconds=210 parent_seconds=480 phase_minutes=18`.
The `SOURCE` row is at most 1024 bytes, with exactly:

| Field | Required value |
| --- | --- |
| schema | integer 1 |
| profile | deadline-provenance-v1 |
| source_commit | externally supplied lowercase 40-digit Git commit ID |
| compiler_sha256 | externally supplied lowercase 64-digit compiler checksum |
| artifact_sha256 | externally supplied lowercase 64-digit artifact checksum |

Expected identities are required arguments, never inferred from the transcript.
Matching strings bind a record to the caller's candidate; they are not independent
attestation of source cleanliness, signatures, compilation or execution. Those
remain requirements of the reviewed runner and qualification evidence.

The parent is `TestNativeMultiplePolicyStartupProvenanceUFS`, with one RUN,
normal PASS ending and final PASS. Its elapsed duration must contain the original
operation and fit the unchanged 18-minute process phase. Inside it, require one
`TRACE`, one `SUMMARY`, three `STACK` rows, one `JOURNAL`, one `DOMAIN_TEARDOWN`,
one `UFS_RESULT`, and the original `POLICY_FACTS_CHILD_RESULT` and
`POLICY_FACTS_PARENT_RESULT`. Duplicate, missing or unknown fields/records,
nonfinite scalars, excessive nesting and invalid types are rejected. The summary,
stacks, journal and teardown retain the original envelope schemas and three
30/75/110-second samples. Journal generation booleans are rejected as nonintegers.

## Exact inventory

The original eight child controls remain. Five new child controls are
`trace_enabled`, `error_preserved`, `snapshots_before_cancel`, `trace_inventory`
and `original_contexts`: **13 child controls**. The parent carries all thirteen as
`child_` controls and retains its thirteen original nonchild controls. It adds
`trace_extracted`, `trace_sanitized` and `source_isolated`: **29 diagnostic parent
controls**. The unchanged facts child/parent supply **21 controls**, making **50
unique parent-visible controls**. This preserves all original 42 controls.

The parent must verify the actual child inventory and normally waited children;
control strings alone cannot prove those observations. The future native fixture
must implement each check, including normal/tagged source isolation, and retain
one parent/two children. The parser does not execute that fixture. Existing
42-control envelope and 293-control lifecycle sources cannot satisfy this contract.

## Error trace semantics

`TRACE` is the existing tagged adapter's exact JSON `{status, frames}`, at most
8192 encoded bytes and twelve frames. Every frame has precisely the fields of
[deadlineFrame](../../../internal/storage/deadline_return_fixture.go), with fixed
site/phase/class identifiers, original 90/45/10-second allowances, explicit child
presence, bounded branch identity and relative monotonic values. Parent-only
pre-child guards have zero allowance and absent child observations. Only original
review/activation and command sites have such guards; their actual parent error
must remain deadline-expired. Collector guard frames and live-parent guard
claims are rejected.

The consumer independently checks deadlines, capture bounds, monotonic context
error classes and constraint classification. Complete traces have one unambiguous
branch `0`, no overlap or unknown context error, and an original review/activation
outer frame whose parent deadline is the unchanged 180-second caller. Inner
command/collector/outer ordering and shared deadlines must match the actual
source path; provider collection reuses system-command capture. These checks do
not identify earliest timer delivery or precise command duration.

Success requires an empty success trace; a nondeadline operation requires the
appropriate empty not-deadline trace. Unattributed, ambiguous, invalid and overflow
observations remain distinct and return `diagnostic_complete: false`. Valid
competing branch records are retained. Malformed or falsely complete records are
rejected. Native acceptance is always false, including successful startup and
complete deadline traces. The CLI prints valid scalar results and exits 0 for
complete diagnostic evidence, 1 for well-formed incomplete evidence, and 2 for
invalid input. A future runner must inspect this outcome rather than treating any
parsed result as qualified.

Only after validating the new candidate/control inventory does the consumer
project the original scalar records and 42 controls into the unchanged envelope
parser. Projection cannot turn an earlier envelope receipt into new evidence.

## Verification and limits

Signed failing-first tests cover absent consumer behavior. Focused portable tests
exercise all 50 controls, external identity mismatches, profiles, raw/duplicate/
unknown/unbounded records, original paths, guards, inherited/local/unresolved/
canceled/overlapping observations, branch retention, samples, journal, teardown
and CLI exit status. The focused suite uses synthetic records and disposable
text files only. All 24 preceding receipts, product sources, native fixtures,
existing consumers, runners and deadlines remain byte-identical.

The clean signed/DCO actual-title `make review` requires check/lint/test/secrets;
shared tooling selects the existing conservative full portable race/fuzz fallback.
Final receipt-only changes require fresh baseline and verified exact-input reuse.
Owned-process teardown is required. No new dependency, VM, native execution,
service, runtime interface or tracking automation is introduced.

The separately reviewed tagged fixture/runner and one adopted fresh 15.1 diagnostic
remain pending. Both complete lifecycle and all broader native safety/recovery,
worker/backend/global launch, manual browser/client, hardware/performance and
release gates remain open. Runtime writes remain disabled. Failed, skipped and
unverified prior limits remain bound to their original candidates.
