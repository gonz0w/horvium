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
