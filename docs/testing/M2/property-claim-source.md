# M2 native property observation through an original root claim

Date: 2026-10-08. Status: implementation merged in PR 84; native observer qualification pending.
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
Collection uses the existing 135-second context and discards every projection on
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
callback type correction is not a product or native failure. The first native
15.1 attempt correctly returned a canonically ordered policy; the new test
incorrectly compared it to an unsorted input fixture. Compare the complete
canonical encoding instead, retain the failed attempt under its actual source,
and rerun both releases without weakening any product gate.

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

Native runs are serial and source/artifact bound. Avoid overlapping deadline-sensitive
pool qualification with portable deep review or unrelated VM qualification. Shared runners, selectors,
checkers, supervisors, historical manifests and dependencies remain unchanged.
Mandatory baseline and affected race/fuzz checks remain required. All owned
fixtures, guest images/keys/firmware, executable artifacts and processes must be
removed or verified stopped. Full backend/eligibility, interruption/reboot/recovery,
terminal resolution and authenticated operator workflows remain separate gates.

## Source-bound results and pending gates

Signed source `aa0794996733706c47cce45ad6e0aad3f4155d28`, based on accepted PR 83
merge `36a919d78c03a5572087a7da8f6e6c9467189ef2`, passed all four new portable parents,
mandatory `make check lint test secrets` with 122 tooling tests, nine affected
race packages and all 38 selected fuzz targets. Focused review completed in
191.718 seconds with verified worker teardown. See the
[sanitized manifest](property-claim-source-manifest.json), explicitly marked
`qualification_complete=false`.

Both disposable state runs passed every original 819 control, 67-parent and
1,876-codec gate plus all 19 live-state children, including the new claim-bound
readers, actual root/foreign/closed/unsafe/orphan checks. Identical checksum-bound
state artifacts were executed on both releases. The live-state groups completed
in 4.90/5.60 seconds on 15.1/14.5. Their policy and physical inputs remain synthetic;
this result qualifies the metadata readers, not the combined native observer.

Two actual 15.1 pool attempts passed all eight new native claim children,
including independent GEOM/system/kernel backing agreement, queued/applying
collection with false eligibility, foreign/closed/reopened denials, exact data/
metadata preservation and complete new-pool/mount restoration. Both enclosing
parents nevertheless hit the unchanged five-minute deadline during later old
postcondition/control tests. Neither qualifies the native pool gate. The first
also overlapped portable review; the identical-source rerun without that review
still timed out. Several unrelated active QEMU workloads were observed on the
Mac; contention is a possible contributor, not a proven sole cause. No unrelated
process was stopped. The 14.5 pool gate has not been attempted under that load.

Both full current-source pool gates remain mandatory, with all 88 old assertions,
old subgroups and original deadlines retained. Resume them when capacity is
available; portable deep evidence may be reused only under the existing exact-input
rules. Keep the PR draft until these native gates pass. No native observer or full
backend acceptance, runtime eligibility or operator writer follows from partial
results. The original failed policy-order fixture attempt and the original
successful portable review retain their actual sources and are not relabeled.

All 59 prior M2 manifests and 412 retained implementation/tool/fixture paths were
verified byte-identical. Shared runners, selectors, checkers, supervisors,
dependencies, policies and the old pool assertions/deadline remain unchanged.
Every attempted guest and owned executable/image/key/firmware/device backing was
removed or verified stopped; host base/runtime hashes were preserved. The two
unexecuted development cross-builds were removed and supply no qualification.
`make lab-status` verified no active or unverified Horvium-owned session. Repository
Actions remain disabled; no persistent service was enabled. Unrelated workloads
are outside Horvium's teardown claim.

## Qualification fixture maintenance

[PR 84](https://github.com/gonz0w/horvium/pull/84) merged at
`41140a8061267a000bd321b5c5f4f536954cfb25` with the pool gates still outstanding.
A retry without other QEMU guests observed at start also hit the unchanged parent
deadline. The [separate fixture contract](property-claim-fixtures.md) preserves the
old five-minute collector and all historical assertions, gives the new claim
scenario its own 135-second context, and makes both actual parents/every claim
child mandatory in the shared checker. This supersedes the earlier fixture layout
and unchanged-selector assumption; product behavior and old manifests are unchanged.
Current full native pool gates remain pending until both releases qualify.
