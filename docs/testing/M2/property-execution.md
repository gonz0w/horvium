# M2 one-attempt property execution sequence

Date: 2026-10-07. Status: portable subset qualified; accepted in [PR 80](https://github.com/gonz0w/horvium/pull/80).
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-JOB-001, M2-PROPERTY-001, M2-DRIFT-001 and M2-RECOVERY-001 work.

## Contract and authority boundary

`PropertyExecution` connects the accepted immutable property intent, job successor,
postcondition and durable publisher contracts through a trusted root-worker
`PropertyExecutionBackend`. It represents exactly one attempt, owns no goroutine,
process, timer, file or lease and is consumed even by preflight denial. Concurrent
calls on that object cannot repeat the attempt. It is not the global scheduler,
cross-process lock, pool reservation or ZFS fence. No native backend, runtime,
HTTP handler, helper transport or operator CLI invokes it in this slice.

`Run` accepts only a validated job identifier and a worker lifetime context. The
backend must independently read the current durably queued job and claim its
original live plan/session, establish root/store/recovery/executor safety and
pool/global launch reservations, then collect fresh eligible native facts. The
existing native sources deliberately return false eligibility and cannot implement
this contract by changing a flag. A retained/decoded job, public input boolean,
digest or reconstructed executor supplies no authority. Restart cannot restore
the live claim. The backend must remain bounded and hold the original root lease.

## Durable ordering and ambiguity

1. Validate the complete returned queued job/identifier and fresh eligible baseline.
   Check the trusted civil clock, retained timestamp, five-minute review expiry
   and worker context before and after bounded comparison.
2. Publish one exact `applying / launch-intent` successor. Require an exact
   canonical acknowledgement of the whole immutable record before proceeding.
3. Through the original live claim/lease, refresh current policy/native baseline
   and the exact durable applying predecessor. Repeat baseline/clock/context
   checks. The native backend must resolve/recheck the actual GUID/address at its
   sole allowlisted invocation; this sequential observation is not atomic fencing.
4. Invoke one typed immutable property intent. The backend must use fixed
   allowlisted arguments and report quiescence only after its native child/guard
   is verified gone. Neither timeout nor command exit alone establishes safety.
5. After a successful quiescent control, independently collect complete native
   facts. Hash the bounded sorted relevant review projection and full policy
   digest in a dedicated observation domain. Publish exact `observing` metadata
   before an exact postcondition match can lead to durable `succeeded` metadata.

Any failed or foreign durability acknowledgement stops subsequent publication
and launch, even when the record may already exist. Before applying, ordinary
denial returns zero. Publication uncertainty returns the verified queued
predecessor. After applying, an error returns only the last exact acknowledged
record; it never invents a durable result. Successful outcome acknowledgement
after launch is retained even after review expiry or worker cancellation during
publication. No additional operation is authorized by that acknowledgement.

An unverified native child yields `executor-unverified`; a quiescent control error
yields `ambiguous-effect`; absent/malformed post-effect observations yield
`storage-unavailable`. Do not observe an unverified child as completed. Valid
differing observations are retained in `observing`, then `ambiguous-effect`
reconciliation with the same digest. An external identical change or an
already-matching value cannot itself establish executor attribution; the backend
must qualify the actual invocation and exclusive-writer assumptions independently.

Reconciliation uses existing codec edges, remains nonterminal and retains holds.
It can continue bounded metadata publication after worker cancellation, but a
backwards civil clock cannot fabricate a new timestamp. Failed reconciliation
publication leaves the last acknowledged applying/observing record. No retry,
rollback, force, cancellation command, terminal failure inference, reservation
release, eviction or recovery resolution is added. Repair requires separate
native attribution, restored-state and quiescence gates.

## Test-first portable qualification

New behavioral tests first require the absent execution contract. They cover:
durable ordering/owned records, publication failure at each boundary, effects
whose acknowledgements were lost, unavailable/nonquiescent controls, malformed
and differing observations, fresh launch denials at both observation windows,
expiry/cancellation/civil rollback/future history, foreign/rewritten/earlier
acknowledgements, 32 concurrent calls, retained applying/observing/terminal/
reconciliation refusal, late completion and invalid inputs. Bounded cut-point
fuzzing requires no second attempt and no success without the complete sequence.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyExecution'
PR_TITLE='feat(storage): sequence durable property execution' make review
make lab-status
```

Require baseline/122 tooling tests, affected-package race and every selected
compiler-listed fuzz target, plus a FreeBSD test cross-build. No guest storage
write or native-source/store/fixture/selector change occurs here. Historical
evidence retains actual tested revisions; an unexecuted cross-build and synthetic
backend do not qualify a native executor. Root live-claim/eligibility/transport,
one global launch/pool serialization, native argument/process containment,
real durability/effect/death/reboot cuts, recovery and complete M2 remain open.

## Source-bound results

Signed source `cceb615b82aedc98d4e93fb1cdb90545b05916ea` passed all ten new behavioral
groups and the mandatory baseline (`make check lint test secrets`, 122 tooling
tests). Focused review ran nine affected race packages and all 34 compiler-listed
fuzz targets, including `FuzzPropertyExecutionCut`, in 192.379 seconds. Both supervised
review workers completed with verified process-group teardown. See the
[manifest](property-execution-manifest.json) for actual source/input/report/log
hashes, selected inventory and the initial absent-contract compile failure.

The same source compiled the storage tests for `freebsd/amd64`, cgo disabled.
The binary was not executed and was removed. All 55 historical M2 manifests and
397 existing code/tool/manifest paths remain byte-identical to the base, including
every native source/store/fixture and shared selector/supervisor. Prior native
evidence was not rerun or relabeled; no native guest or service was started.
`make lab-status` reports no active or unverified owned sessions. This is portable
sequence qualification; synthetic backend authority and effects establish no
real native execution, root claim, restored-state safety, release or M2 acceptance.


The later [original admission handoff](property-handoff.md) binds one sequence
object to the original coordinator/backend, the exact queued acknowledgement and
original civil-clock lineage. The generic constructor and this sequence's durable
ordering remain unchanged. Complete native/root eligibility, global scheduling,
actual interruption/recovery and runtime workflows still require separate gates.
