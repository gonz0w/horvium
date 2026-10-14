# Native dashboard value invariance

Status: Bounded research, with no changed resource acceptance contract.
Date: 2026-10-13. Owner: Cam (`@gonz0w`).
Refs [382](https://github.com/gonz0w/horvium/issues/382),
[289](https://github.com/gonz0w/horvium/issues/289),
[290](https://github.com/gonz0w/horvium/issues/290).
Authority: adopted [PRD](../../design/dashboards/PRD.md),
[EDD](../../design/dashboards/EDD.md), [Test Design](../../design/dashboards/TDD.md)
and [resource supplement](../../design/dashboards/resource-qualification.md),
DASH-REQ-006 / DASH-PERF-001. Preserve the
[previous exact failures](table-location-diagnosis.md) and historical manifests.

## Question and time box

Do the native fixture's own resources or other disposable-guest storage change
across replacement, and can a known bounded fixture write invalidate the frozen
original values while the cache continues to display current native observations?
One clean signed/DCO reviewed candidate runs one disposable FreeBSD 15.1 probe,
with an eight-minute Go deadline and ten-minute owned command deadline. Existing
preparation/boot, byte-verified artifact transfer, before/after preflight and host
teardown remain bound to the actual source. The private scoped controller retains
the documented 20-second research transfer connection window. Preparation and
execution failures remain retained; never retry a failed candidate for a pass.
The canonical performance runner, original oracle and budgets remain unchanged.
No FreeBSD 14.5 or full resource acceptance follows from this research.

## Independent ownership and controlled observations

Create the existing disposable file-backed md/ZFS fixture with 100 independently
observed filesystems. Its exact original pool GUID and separate zfs TSV witness
identify owned resources; names/prefixes and cached values do not establish
ownership. A private controller comparison reports only fixed field-change counts
for owned pools/datasets and other guest pools/datasets, rejecting incomplete,
duplicate, missing or changed identities and uncertain original membership.
It does not authorize replacing the original expected numeric values.

Observe four stages: baseline, original receipt written, scratch written and
scratch removed. Each uses an independent native collection, complete bounded
same-generation pool/dataset API pages and 50 sequential HTML requests through
the frozen original predicate (ten per existing dashboard case). Original-to-native,
original-to-cache and native-to-cache differences are diagnostics, not new passing
predicates. Collection/query sequencing can expose timing differences; a nonzero
native-to-cache difference is not itself a product defect. This probe has no
original warm-up/idle, ten-reader resource or crash/recovery coverage.

The scratch directory is uniquely created inside the disposable guest's verified
work tree, outside executable artifacts. Kernel statfs must identify ZFS backing;
its exact mounted dataset and pool must occur in the independent original and be
outside the owned test pool. The fixture writes one 4 MiB deterministic payload
with exclusive creation, complete write/fsync/close and SHA-256 readback, then
syncs only that identified guest pool and explicitly replaces the cache generation.
Removal also syncs that identified pool and replaces the generation. No operator
path, disk or backup is used; no host service or product writer is enabled.
The mounted scratch dataset's independent Used/Referenced must increase at the
write stage. Whether removal returns every counter to baseline is an observation,
not an asserted ZFS accounting guarantee. Public output contains fixed counts,
booleans, original digest and bounded sanitized response samples, with no names,
GUIDs, response bodies, paths, credentials or actual private byte values.

## Teardown and correction boundary

Stop only original paired processes. Verify owned md/pool removal, directory/socket
and scratch absence, empty original descendant reaper, unchanged original receipt
and executable artifacts. The custom cleanup is a bounded diagnostic observation;
it does not claim the complete resource parent receipt. Independently reopen host
QEMU/listener/key/image/variables cleanup and original base/QEMU/firmware bytes.
Keep qualified/native_acceptance false, even when the diagnostic Go test passes.
Run required checks, signed clean review and lab-stop/status; preserve all 152
historical manifests and primary operator edits. A failed setup or uncertain cleanup
remains unqualified and is retained with its original reason.

The current Test Design requires constant native values through replacement.
If actual fixture activity changes those values, a reviewed contract correction
must retain independent original provenance for every expected observation; it
cannot silently refresh expectations from management responses, exclude mismatches,
relax exact tables/comparisons, reduce request counts or raise resource budgets.
Publish actual findings before proposing that correction. Runtime writes, full
both-release dashboard acceptance, browser/manual accessibility, startup/recovery,
hardware and release gates remain open.


## Actual result and scope of the inference

Signed source `d975f7d5f16f5a61d57fd183e8748da80e2c1944`, base
`97ebb60836329730b3976629d255b584ddc443ec`, passed required checks and clean
signed/DCO affected-package review before execution. The diagnostic passed in
38.83 seconds on actual FreeBSD kernel/userland 15.1-RELEASE-p3, Go 1.27.1 and
ZFS 2.4.2, with the original qemu64/two-CPU/6144 MiB/TCG pins. Two pools and
115 datasets included 100 independently owned datasets. The
[source-bound manifest](native-value-invariance-manifest.json) retains original
candidate/compiler/artifact, preflight, transcript, review and supervisor hashes.

| Stage | Original HTML successes | Original HTML failures | Independent accounting changes from original |
| --- | ---: | ---: | --- |
| Baseline | 50 | 0 | None |
| Receipt written, identified-pool sync and refresh | 1 | 49 | Other guest pool Allocated/Free; other datasets Used/Available/Referenced |
| Scratch written, sync and refresh | 1 | 49 | Same changed-field counts; no claimed incremental magnitude |
| Scratch removed, sync and refresh | 1 | 49 | Original differences retained; no return-to-baseline claim |

At every stage the 100 owned datasets and their pool had **zero field changes**.
Other guest storage had one changed Allocated/Free pool, three changed Used
datasets, all fifteen other datasets with changed Available, and one changed
Referenced dataset. Every generation-bound cached pool/dataset value matched
the separately collected current native value: zero native-to-cache field
differences. Identity, name, size and health remained unchanged. All 200 responses
were complete HTTP 200; the original oracle still rejected 147 of them. Each
post-baseline stage had 30 Allocated first failures, 18 Used first failures and
one Available first failure, with one successful dataset case observation.
There is no successful resource receipt for that workload.

The receipt stage combines original receipt creation, pool sync and refresh,
so it does not isolate every natural writer or assign all changes to one syscall.
It does demonstrate that this fixture sequence does not retain constant values
for other guest storage. Current cached numbers agreeing with fresh independent
native numbers is bounded evidence against interpreting this controlled sequence
as stale or incorrect accounting. It does not prove the cause of each historical
candidate failure or establish general runtime correctness. Historical manifests
and unclassified failures stay unchanged.

The 4 MiB scratch payload had exclusive creation, complete fsync/close and exact
SHA-256 readback inside the independently identified other guest ZFS dataset.
Its Used/Referenced booleans were already above the original baseline at the
receipt stage. They compare with that original, not with the immediately preceding
stage; this receipt makes no claim about the scratch write's incremental accounting
amount or removal restoring old values. The fixed field counts alone cannot show
that magnitude. No private observed byte values or resource identities are published.

Original bounded guest pool/md, process descendants, temporary directory/socket,
scratch absence, preserved receipt and executable checks passed. Independently
reopened before/after preflight artifacts/environment matched, host QEMU/listener/
key/image/variables cleanup passed and original base/QEMU/firmware bytes remained
unchanged. This is diagnostic cleanup, not the complete four-mode resource parent
receipt. Command and outer worker groups completed cleanly without escalation.
All 152 historical manifests and primary operator edits retain their original bytes.
Required checks, 306 tooling tests and fresh cmd/horvium race review passed, with
no eligible fuzz targets. The later documentation head reruns required baseline
checks and may reuse only verified exact-input original deep evidence.

## Reviewable correction direction

The [proposed Test Design requirement refinement](../../design/dashboards/native-invariance-correction.md)
keeps synthetic expectations frozen, but requires independently bound original
native producer values for each displayed generation if native numbers change.
The packet is **Draft**, not an adopted witness mechanism or permission to run
qualification. It names the unresolved producer/generation provenance, actual
helper artifact, resource overhead/retention, independent parsing and cleanup
requirements that must be concretely designed and reviewed. Comparing tables to
another management response, silently refreshing old expectations, dropping other
guest rows or raising budgets would not satisfy it.

This PR combines ownership diagnostics, actual native evidence and the proposed
requirement correction. It ends at that concrete review boundary because the
producer witness mechanism is unresolved. #382, broader dashboards, native
startup/recovery, manual accessibility, hardware and release acceptance stay open.
Next work is the reviewed generation-binding mechanism, its implementation and
fresh complete execution on both releases. The earlier large-mode latency failure
also remains unresolved; native invariance diagnosis does not claim to fix it.
Runtime writes remain disabled.
