# M2 native property observation through an original root claim

Date: 2026-10-08. Status: implementation; qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md)
and [issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-STATE-001, M2-DRIFT-001, M2-DEVICE-001 and M2-RECOVERY-001 evidence.

## Original lease and fresh native inputs

The accepted [state claim](property-live-state.md) proves volatile root submission
provenance, but its standalone positive readiness fixture supplies synthetic policy
and physical system facts. This subset joins that same claim to guarded actual
property, provider, mount/swap/imported-pool and backing observations.

`PropertyStateClaim.InspectPropertyJob` performs the existing strict complete
mixed census under the original root mutex, then verifies the exact queued or
single applying job against the original post-sync receipt and independent holds.
The digest includes every ordered record name and presentation byte.
`ReadOwnership` selects only that job's pool and brackets the descriptor-anchored
ownership reader with complete claim/census reads. Actual root EUID, original
handle, active claim, uncertainty and closed-lease checks precede output. Failure
returns zero projections; recognized publication orphans retain the existing latch.
The existing unclaimed diagnostic reader keeps its read-only semantics.

`NewPropertyClaimSource` accepts only the opaque original claim. Metadata readers
and the final state check come from that claim; there is no separately supplied
state handle, caller policy/system facts or command path. Identifier-only collection
brackets the existing guarded native safety source with whole job/census reads.
The claim's actual backing check runs inside the final policy window; an independent
final ownership read, exact whole job and full census digest must remain stable.
The operation has the existing 135-second bound and discards every projection on
failure, including cancellation. Success always returns `Eligible=false`.

Facts may differ from the captured baseline or describe a later effect. Read-only
assessment establishes neither attribution nor execution authority. This subset
has no runtime consumer, launch, native control invocation, outcome publication,
reservation release, restored-state clearance or empty-store recovery inference.
Original coordinator/session/clock lineage and executor/global launch proof still
need the complete root backend before eligibility can be promoted.

## Test-first and native gates

Four portable behavioral parents cover original whole-record/ledger binding,
required state checks, selectors before I/O, cancellation, final policy drift,
native provider/system failures and overlapping root use. Bounded selector fuzzing
must not allow invalid input to reach I/O or accept missing trusted facts. The
initial absent-contract compile failure precedes implementation; a later fixture
callback type correction is not a product or native failure.

Extend the existing 19 native live-state cases with actual claim-bound readers,
foreign selection, root EUID, drift/orphan/unsafe/closed denials and zero outputs.
Retain all 819 historical state/backing controls, 67 parents, 1,876 codec children
and original deadlines on both pinned disposable releases.

The existing five-disk pool fixture supplies its uniquely identified enrolled
mirror and one declared nonboot 64 MiB disk for a separate temporary ZFS metadata
pool. That disk's duplicate serial is positive system-backing evidence only; it
never becomes an eligible data member. Actual GEOM/system facts and kernel statfs
must agree on the independent metadata backing. The eight new children are
`independent-backing`, `queued`, `applying`, `foreign-selection`, `closed`,
`reopened`, `preserved` and `teardown`. Test capture explicitly sets eligibility
only to construct immutable fixture intent; both actual source results and plan
building from them must retain/refuse false eligibility.

The helper publishes metadata only, verifies actual new submission/claim,
refuses restored authority after reopening/retry, and preserves exact policy/job
bytes, the data sentinel and the complete data-property projection. Destroy the
GUID-checked owned metadata pool and remove its private directory before the
existing fixture continues. The guest root filesystem and its backing settings
remain unchanged. Retain all 88 old pool assertions, all old assessment/control/
postcondition children and the original five-minute parent deadline.

```sh
make test-focus PKG=./internal/host/freebsd TEST='^TestPropertyClaimSource'
PR_TITLE='feat(storage): collect native properties through original root claims' make review
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Native runs are serial and source/artifact bound. Shared runners, selectors,
checkers, supervisors, historical manifests and dependencies remain unchanged.
Mandatory baseline and affected race/fuzz checks remain required. All owned
fixtures, guest images/keys/firmware, executable artifacts and processes must be
removed or verified stopped. Full backend/eligibility, interruption/reboot/recovery,
terminal resolution and authenticated operator workflows remain separate gates.
