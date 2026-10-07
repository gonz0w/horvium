# M2 job reservation metadata checks

Date: 2026-10-06. Status: portable qualification passed; maintainer review pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract

`BuildJobReservations(records)` creates an immutable, bounded view of a supplied
complete job-record set. It strictly decodes both accepted formats, retaining the
entire property review when present. Reject duplicate job, submission or plan IDs
across formats, invalid/unknown/damaged records and budget exhaustion. Return zero
state on any error; never retain a partial census. Bound count and actual input
bytes before decoding/copying: 64 nonterminal jobs, 1,000 terminal jobs, 64 KiB per
record and 64 MiB job bytes. Ownership/transient bytes still need the independently
qualified mixed-store capacity check. Reduced test bounds are internal fixtures,
not a runtime tuning interface or commercial scale qualification.

`Pools()` returns independent pool/holder summaries in lexical pool-GUID/job-ID
order. Every retained job holds its entire pool, including legacy jobs and queued,
applying, observing, reconcile-required and terminal metadata. Multiple existing
holders remain visible rather than being arbitrarily selected. This conservative
prerequisite has no release method: success/failure/cancellation labels and digests
alone cannot prove native completion, durable acknowledgement or old-executor
quiescence. A later qualified recovery/release boundary must resolve those holds;
this is not the complete scheduler's terminal-retention policy.

`CheckPropertySubmission(candidate)` evaluates an initial queued, consistency-
checked property job against the immutable view. A matching submission returns
the existing complete property job/current history and `duplicate=true`; new
metadata returns zero/false/nil. Every denial returns zero/false/error. Retry
meaning requires exact job selection, entire immutable review and policy/session
digests. A newly generated job ID and capture-clock difference can vary; capture
still must bind before the candidate's initial event and within review lifetime.
An ID belonging to another submission, different meaning under the same submission
or reuse of a plan for a new submission is denied. A legacy record cannot supply
the missing review/session/policy binding. Matching retries work at capacity and
return current applying/cancelled/reconciliation history without resetting it.

New candidates are denied if any retained job holds their pool, nonterminal slots
are full or job bytes would exceed the bound. Other pools can pass metadata checks.
Property jobs change existing filesystems and need no new-device/new-namespace
claim; creation, replacement, maintenance scheduling and their resource sets remain
later operation contracts.

These are pure checks: concurrent callers can both pass against an empty view.
They acquire no reservation, claim no plan, acknowledge no admission and launch
nothing. The zero view is unavailable rather than an empty trusted census. No file
format, native reader/publisher, runtime/helper/HTTP/CLI caller or config changes.

## Native integration still required

Byte validity cannot prove the caller supplied every durable record. A future
adapter must collect and bind the complete namespace to opened private objects
under the root writer lease, retain publication uncertainty, authenticate the
caller, validate enrolled root policy, live sessions/clock and full fresh native
eligibility, claim/reserve atomically, recheck drift and mixed/transient budgets,
and persist admission before acknowledging or launching. Matching retries do not
restore an expired session or authorize access to an existing job. Missing native
executor/boot attribution, quiescence, recovery/release and acknowledgement-cut
tests continue to keep runtime writes disabled.

## Portable procedure

Failing behavioral tests precede implementation. Cover whole-set refusal,
cross-format identifier collisions, complete duplicate meaning/current history,
pool holds in every lifecycle state, unrelated pools, actual class ceilings,
reduced byte limits, zero returns, copy isolation and concurrent read-only checks.
Hostile single-record fuzzing must preserve bounded zero-on-error behavior and
never produce a false duplicate. Run:

```sh
make test-focus PKG=./internal/storage TEST='^TestReservation'
# Signed committed source; actual proposed PR title:
PR_TITLE='feat(storage): check duplicate submissions and pool reservations' make review
make lab-status
```

The review runs the full cheap baseline plus storage-only race/all compiler-listed
storage fuzz checks. FreeBSD cross-compilation demonstrates portability only.
Native facilities/fixtures are unchanged, so no new QEMU run is needed for this
pure model. Prior native evidence stays bound to its original signed sources.
Complete M2 admission/recovery, resource/performance, hardware/client and manual
browser acceptance remain separate gates.


## Executed qualification

Signed source `eec1ad7c34485bc2b548f8a1c88f68cdaaaae647` passed all six parent tests and
thirty subtests without skips/failures. The [sanitized manifest](job-reservations-manifest.json)
records exact source/base/input/toolchain/log hashes, case inventory, prior evidence
preservation and owned teardown. All baseline checks (95
tooling regressions), storage race and all nineteen compiler-listed storage fuzz
targets passed in 99.829 seconds. The review selected only
`./internal/storage`; no fresh full milestone run or native QEMU suite ran.

The same source cross-compiled its test binary for `freebsd/amd64` with cgo
disabled; it was not executed and the owned artifact was removed. All thirty
prior M2 manifests remain byte-identical to the base. `make lab-status` verified
all 196 owned sessions stopped. A final documentation/evidence-only
head must rerun baseline/title/signature/DCO/source checks and explicitly cite this
original source if deep checks are reused. Runtime admission/release and complete
M2 acceptance remain unqualified.
