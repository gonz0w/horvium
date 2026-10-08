# M2 original property admission execution handoff

Date: 2026-10-07. Status: portable subset accepted in PR 82.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0006](../../adr/0006-storage-ownership-and-plans.md),
[ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-PLAN-001, M2-JOB-001, M2-DRIFT-001 and M2-RECOVERY-001 evidence.

## Original live handoff

`PropertyCoordinator.ClaimPropertyExecution` accepts an authenticated session
context and job identifier only. It finds the job in the original bounded set of
volatile reviews and freshly acknowledged submissions. A successful new admission
can transfer its live claim to one `PropertyExecution` object under the original
coordinator mutex. Matching confirmation replies remain copies of the original
admission acknowledgement, not current status or new launch authority.

Reject wrong sessions, unknown/unsubmitted jobs, expired plans, cancelled contexts,
backwards civil clocks, publication-blocked coordinators, backend-returned existing
admissions and retained nonqueued acknowledgements. A reconstructed coordinator
has no original reviews and cannot hand off a stored job. There is no record,
policy, observation, eligibility flag, clock, executable or replacement-backend
input. Repeat handoffs are denied even when the first object never runs. Copies
of the issued pointer still share the execution sequence's one-attempt mutex.

The exact same original backend must implement both admission and execution
contracts. An envelope retains that backend and the owned canonical queued
acknowledgement. Before any applying publication, its independently prepared
current durable job must equal the entire acknowledged record, including intent,
capture time, selection, history and identifier. Selection or job-ID equality
alone is insufficient. The accepted sequence independently validates fresh policy
and the complete eligible baseline; the envelope supplies no eligibility.

The worker uses the original coordinator's civil-clock lineage. Check clock and
context again after bounded handoff validation and before returning the object;
worker rollback also blocks the original coordinator. Check original publication
uncertainty, expiry, clock and context before and after preparation and refresh,
and immediately before delegating the control. Locks are released before backend
I/O. These sequential checks are not native fencing or global launch scheduling.

No new attempt is authorized after expiry. Once a control has actually returned a
verified acknowledgement, expiry/reclamation of its review entry cannot erase the
effect or prohibit the existing sequence's independent observation and durable
outcome obligations. Original publication, quiescence, lost-acknowledgement,
reconciliation and hold-retention rules remain unchanged. Cancellation of the
handoff request does not become the issued worker's lifetime context; the trusted
helper must supply its independent worker context to `Run`.

## Native and runtime obligations

This portable handoff creates no process, goroutine, file, service or lease. The
combined original backend must still independently authenticate the session,
hold the same original root lease, validate complete native eligibility, recovered
state and the actual current applying predecessor, and serialize global launches
and pool holds. No product backend implements those combined obligations yet.
Synthetic test adapters do not qualify root, FreeBSD or durable execution.

The [private native controls](property-control.md) remain disconnected. This
handoff does not bind their private authorization callback or promote false
eligibility observations. The generic [execution sequence](property-execution.md)
constructor remains a trusted orchestration contract; reconstructing one cannot
restore native authority. No operator writer, helper/HTTP/CLI route, release of a
reservation, replay, rollback, recovery resolution or configuration format is added.

## Test-first qualification

Eight behavioral parents require original review/admission-to-outcome ordering,
owned acknowledgements, 32 concurrent handoffs and 32 calls through the issued
object, duplicate/restored/missing-backend/input denials, exact fresh-record binding,
final clock/context refusal, actual coordinator publication-uncertainty propagation,
live prelaunch gates, late durable outcomes and lost acknowledgement cuts. Bounded
action-sequence fuzzing refuses second admission, handoff or control attempts.

The initial absent-contract compile failure and the separately failing final-clock
regression are retained. The fixture's initial ambiguous embedded policy selector
was corrected before behavioral execution; it is not a product failure or native
finding. Historical native evidence stays bound to its actual executed sources.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyExecutionHandoff'
PR_TITLE='feat(storage): hand off original admitted property execution' make review
make lab-status
```

Require the mandatory baseline, affected-package race/every selected portable fuzz
target and a FreeBSD test cross-build. Native implementations, fixtures, shared
selectors/supervisors and prior manifests are unchanged; no native rerun or support
claim follows from an unexecuted cross-build. Complete root backend, native
eligibility/scheduling, real interruption/reboot/recovery, transport and assembled
M2 acceptance remain separate gates.

## Source-bound results

Signed source `01af1c83f2deb4e1c492aaee1098344b3def8dcc`, based on PR 81 merge
`e16e130aa502f66202386e3578095d607947b0ba`, passed all eight new behavioral
parents, the final-clock regression and the mandatory `make check lint test secrets`
baseline with 122 tooling tests. Focused review ran 9 affected race packages and
all 36 compiler-listed selected fuzz targets, including `FuzzPropertyExecutionHandoff`,
in 206.880 seconds. Both review workers completed with verified group teardown.
See the [sanitized manifest](property-handoff-manifest.json) for actual source/input/
report/log hashes, failing-first records, compiler/binary hashes and inventory.

The same source cross-compiled the storage tests for `freebsd/amd64`, cgo disabled;
the binary was not executed and was removed. All 57 historical M2 manifests and
418 existing implementation/fixture/tool/policy paths were byte-identical to the
base. Every native implementation/fixture, shared selector/supervisor, CLI,
module and packaging source is unchanged. No native suite was rerun or prior
source relabeled. `make lab-status` found no active or unverified owned sessions;
no guest, daemon, service or GitHub Action was started. This is portable live
handoff evidence only; root/native backend, eligibility, global launch/pool
scheduling, actual interruption/recovery and assembled operator gates remain open.
