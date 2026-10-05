# M2 vdev topology observations and review drift

Date: 2026-10-04. Status: behavioral subset passed; pinned native execution pending.
Design: [approved EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[slice sequence](../../milestones/M2/implementation.md). Extends the
[pool](pool-state.md) and [feature](features.md) observations; runtime remains read-only.

## Fixed paired legacy projections

```text
/sbin/zpool status -g -p <pool>
/sbin/zpool status -L -P -p <pool>
```

`ParsePoolTopology` pairs complete successful GUID and resolved full-path views
of the same pool. Require one pool/state/scan/config/header/error record, the
exact five-column NAME/STATE/READ/WRITE/CKSUM header, root name/state agreement,
and matching row counts, indentation, health, counters and annotations. Preserve
non-root vdev GUID, parent GUID, sibling ordinal, current address, observed kind,
health and exact uint64 error counters. GUIDs remain canonical positive decimal
strings. Names and paths are addresses, not enrollment or stable device identity.

The legacy root row prints the pool name even with `-g`, not a pool/root-vdev
GUID. The parser's pool GUID must come from an independent fresh native pool
observation. The future collector must verify name/GUID/import incarnation before
and after capture, retain complete projections and repeat paired reads to detect
observed changes. No separate read or repeated projection is atomic or fences
another root actor. GUID/path rows are paired by matching structural positions;
same-shaped changes between reads still require exclusive-writer and final checks.

Mirror and RAIDZ2 group kinds come from the path view's qualified group names;
absolute leaf paths do not prove disk type, GEOM relationships, local backing,
serial/lunid, sector geometry, alias uniqueness or absence of use. Those independent
provider/system/device-use proofs remain required. No implemented collector can
set the model's complete/eligible flags from status alone.

`VerifyPoolTopology` recognizes ONLINE homogeneous mirror groups (at least two
leaves/group) or homogeneous RAIDZ2 groups (at least four), equal leaf counts in
all groups, and only direct leaf children beneath them. Require zero root/leaf/
intermediate read/write/checksum and pool data errors, no leaf annotation and
scan exactly `none requested`. Missing evidence, stripes, mixed/unequal/nested/
unrecognized layouts, adverse health or errors refuse planning. Known unsupported
bare group names and stripe leaves remain observable; unqualified class sections,
auxiliary annotations and formats return unavailable, never healthy defaults.
M1's broader read-only inventory remains available independently.

Retain qualified status/action/see diagnostics and scan text without interpreting
progress, completion or attribution. All active, paused, completed, cancelled or
unknown scan records block this partial verifier, including healthy pools that
have previously completed a scrub/resilver. Full scan recognition and the newer
JSON format are separate pending gates. The parser has no auto-detection or
fallback to a more permissive format.

Validate unique GUIDs/addresses, existing non-leaf parents, bounded acyclic ancestry
and dense unique sibling ordinals. Normalize input slice order by GUID while
retaining native sibling order. Reviews hold independent copies and require fresh
review on membership, parent/order/address/kind, health/counter/diagnostic/scan drift.
`VerifyTopologyStability` compares two complete normalized observations, including
adverse ones; it is not an eligibility or reservation proof.

## Bounds and failures

Cap each raw projection at 1 MiB and 8,192 newline records, each field/diagnostic/
scan at 4,096 bytes, retained non-root vdevs at 4,096 and nesting at 32 non-root
levels. Require final newlines, valid UTF-8 and no controls except structural
newlines/tabs; reject duplicate/truncated/unknown/contradictory output, unknown
counter units, leading-zero/overflow numbers and unsafe path components.
Only the inspected removal/resilver/repair annotations are retained; they cannot
qualify planning. Failures return a zero topology, never partial membership.

Typed topology participates in the existing 64 MiB aggregate preflight before
maps/sorting/JSON. Full metadata counts against the 64 KiB review cap, without
truncation. These are explicit initial capability limits, not hardware assumptions
or tested throughput/scale claims. The model remains disconnected from helper,
UI/CLI, ownership persistence, jobs and native mutation execution.

## Behavioral and native gates

Behavioral tests first failed against stubs and old review behavior, then passed
for shape/identity/membership/parent/order/bounds, copied reviews, drift, health/
error/scan refusal, homogeneous/mixed/short layouts and unknown formats.
`FuzzPoolTopology` is the twelfth bounded fuzz target. These are partial
M2-PLAN-001, M2-DRIFT-001, M2-HEALTH-001 and M2-SCALE-001; no complete row is satisfied.

Inspected pinned sources define both legacy projections and indentation/counters:
[14.5 manual](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/man/man8/zpool-status.8),
[15.1 manual](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/man/man8/zpool-status.8),
[14.5 formatter](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/cmd/zpool/zpool_main.c),
[15.1 formatter](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/cmd/zpool/zpool_main.c),
[vdev naming](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/lib/libzfs/libzfs_pool.c).
Source facts are not native qualification; no upstream implementation is vendored.

The disposable fixture owns five 512 MiB file-backed md providers. It begins with
two two-leaf mirror groups for the earlier property/identity/state/pool/feature
assertions. It compares GUID/path membership and states independently through
`zpool list -v`, with fresh pool GUID/import/policy observations and repeated
paired status captures. A fixture-only external attach changes one group to three
leaves without changing pool GUID; waiting on actual native resilver completion
allows stable capture, not a product scan attribution or success claim. Fresh
observations reject the old pure-model review. A test-owned offline/online case
retains DEGRADED/OFFLINE state and refuses planning.

Only after those tests destroy their verified owned pool, the same owned providers
exercise RAIDZ2, unsupported stripe refusal and a final two-leaf mirror using the
approved common profile. Recreated vdev GUIDs cannot reuse the original identities.
Every setup/cleanup command is confined to declared fixture resources; no force,
operator inputs or mounted root is used. Imports search only owned providers/GUID;
cleanup verifies pool GUID and each backing/provider correspondence before removal,
then proves absence. Failure invalidates evidence and requires discarding the guest.
Attach/offline/online/destroy remain test setup only, without product counterparts.

Nine new markers join the earlier 42. The checker requires all 51 assertions plus
executed native test and complete teardown. Synthetic trusted policy/eligibility
inputs compare actual native observations; they never qualify md as production
creation devices or establish an assembled eligibility collector.

## Qualification and remaining work

Both pinned guests must execute before native claims; a sanitized source/run/hash
manifest will record actual results. Earlier projection manifests retain their
original source/case sets. JSON/status scan interpretation, provider classes,
stable identity/geometry, system/device-use and complete-impact collection remain
pending. Full M2 operation/state/recovery/client/browser/resource and physical/
commercial gates remain open, including clean cross-release feature-profile import.
The maintainer's deferred manual M1 accessibility gate remains pending.
