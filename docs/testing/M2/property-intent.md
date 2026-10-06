# M2 reviewed property intent capture and codec

Date: 2026-10-06. Status: accepted in PR 52; portable qualification passed.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Capture and comparison

`Plan.CapturePropertyIntent(at, session, policy, observation)` first performs the
existing complete plan revalidation: session, five-minute expiry, policy digest
and fresh review context must match. It returns an immutable `PropertyIntent`
with independent copies, or zero metadata on refusal. It claims no plan, reserves
no resource and performs no file/native operation. Inputs remain caller-owned
throughout the call. `Review()` returns independent copies of every retained row.

Capture retains the entire **current property review model**: desired property
step, exact target/ancestor/subtree context, original effective/source/received
values, affected identities, locally owned snapshot/bookmark identities,
mount/readonly/encryption constraints, pool GUID/import incarnation/policy values,
complete feature projection and vdev GUID/parent/order/address/health/error/scan
projection. Protected descendant branches remain in context. This excludes
unrelated sibling property/version rows, as the existing planner does.

The full policy digest is the existing planner's normalized policy binding,
including all scopes, not just affected scopes. A plan-scoped session digest uses
SHA-256 over `horvium:property-intent-session:v1`, newline, plan ID, newline and the
exact session string. No raw session credential, human name or authentication
secret is serialized. A digest cannot authenticate or restore a session, prove a
human actor, provide kernel peer identity or grant root policy authority.

`CompareBaseline(policy, observation)` recomputes the complete current review
and policy digest from independently fresh trusted inputs. It refuses drift; it
is an observation comparison without session/expiry admission, reservation,
quiescence, native postcondition, authorization or execution. Decoder labels and
checksums cannot provide freshness/completeness/native eligibility. Shape/semantic
validation constructs only a temporary consistency-check observation and derived
affected scopes; those are never enrolled policy or native adapter evidence.

Unsubmitted plans remain volatile and are never restored. The codec has no
`Plan` restore constructor, job embedding, `StateDirectory` reader/publisher,
helper/startup/HTTP/CLI caller or mutation route. Existing job v1 and ownership
formats, publication and native namespace rules remain unchanged. A standalone
property-intent file in that namespace remains unknown and blocks writes.

## Version one format

Every field is required, nonnullable and case-sensitive. Explicit private wire
structs isolate this format from planning Go field names. Unknown/missing/duplicate
keys, wrong types, incompatible kind/version, trailing records and UTF-8 damage
fail with zero metadata/nil bytes. Fixed-shape traversal bounds nesting; its added
boolean branch applies only where the new topology shape declares a boolean.
Existing ownership/job shapes still reject booleans in their string/integer fields.

| Object | Required fields in canonical order |
|---|---|
| Envelope | `version`: 1, `kind`: `property-intent`, `intent`, `checksum` |
| Intent | `captured_at`, `policy_digest`, `session_digest`, `review` |
| Review | `id`, `created_at`, `expires_at`, `policy_revision`, `pool_guid`, `pool_name`, `change`, `context`, `affected`, `versions`, `states`, `pool`, `features`, `topology` |
| Change | `filesystem_guid`, `property`, `value` |
| Context filesystem | `guid`, `name`, `parent_guid`, `properties` |
| Properties | `quota`, `refquota`, `compression` |
| Property value | `effective`, `source`, `source_guid`, `received_value` |
| Affected identity | `guid`, `name` |
| Version identity | `guid`, `name`, `type`, `owner_guid` |
| Filesystem state | `guid`, `name`, `encryption_root_guid`, `mountpoint`, `canmount`, `readonly`, `mounted`, `encryption`, `encryption_root`, `keystatus` |
| Pool | `guid`, `load_guid`, `name`, `guid_source`, `load_guid_source`, `health`, `readonly`, `bootfs`, `altroot`, `cachefile`, `compatibility`, `multihost` |
| Pool value | `effective`, `source` |
| Feature projection | `pool_guid`, `pool_name`, `format`, `values` |
| Feature | `name`, `state`, `source` |
| Topology | `scan_reported`, `pool_guid`, `pool_name`, `health`, `scan`, `diagnostics`, `read_errors`, `write_errors`, `checksum_errors`, `data_errors`, `nodes` |
| Vdev | `guid`, `parent_guid`, `address`, `kind`, `health`, `annotation`, `ordinal`, `read_errors`, `write_errors`, `checksum_errors` |

Timestamps are canonical UTC RFC3339Nano strings, preserving nanoseconds and
removing process-local monotonic/location state. Require instants after the Unix
epoch through year 9999, exact five-minute review lifetime, and capture within
`[created_at, expires_at)`. Offsets, redundant fractional zeros, reversed/expired
capture and guessed timestamps are refused. These recorded times are metadata,
not proof of a trusted current clock or permission to resume work.

Policy revision/error counters/ordinals use exact canonical unsigned decimal
integers; never floating-point values, exponents, signs, strings or fractions.
Existing literal quota strings preserve the entire uint64 range. GUIDs and
property/source/state/layout semantics use the existing model validators.
Ordinals must be below the existing vdev ceiling before conversion to native Go
`int`; explicit parent/sibling order remains identity. Row ordering normalizes to
the existing review order, including feature names/topology; no row is truncated.
Empty version arrays are `[]`. Context/affected/state completeness and source/
identity relationships must be internally consistent, even with a matching checksum.

The checksum is lowercase SHA-256 over `horvium:property-intent:v1`, newline and
compact canonical JSON containing `version`, `kind`, `intent`, in that order.
The checksum field is excluded. Whitespace/object/row order are not identity.
This detects damage/domain confusion; forged but self-consistent metadata remains
possible without trusted root provenance. There is no automatic upgrade or migration.

Input/output are each at most 64 KiB. A conservative 64 KiB row/string budget
runs before copying/indexing/marshalling in-memory values; it may refuse an input
whose JSON alone fits. Retain existing 4 KiB field, 100,000 observation-row,
128-feature and 4,096-vdev ceilings. These are safety bounds, not scale evidence.

## Qualification and remaining gates

Tests first fail for absent capture/codec APIs, then cover exact binding/precision,
independent copies, fresh baseline drift, scoped session digests without raw
credentials, protected branches/unrelated siblings, full uint64 precision, every
required object field/type, valid-checksum semantic refusals, cross-kind denial,
canonical ordering, exact/oversized input, field/count bounds and hostile nesting.
`FuzzPropertyIntent` requires bounded immutable round trips or zero denied output.
Existing ownership/job codec tests remain regression coverage for the parser.

```sh
make test-focus PKG=./internal/storage TEST='^Test(PropertyIntent|Job|OwnershipRecord|OwnershipChecksum)'
PR_TITLE='feat(storage): capture bounded reviewed property intent' make review
make lab-status
```

Use package-scoped race checks and every compiler-listed storage fuzz target,
plus required `make check lint test secrets`. No shared verification tooling or
native filesystem behavior changes: no full milestone/QEMU suite is warranted.
Cross-compile FreeBSD/amd64 as a portability check only; never label it native
execution. Preserve previous native manifests and their actual tested sources.

Traceability: partial M2-PLAN-001/M2-DRIFT-001/M2-STATE-001 at T1. This closes the
current property-review capture gap, not complete executable intent. Qualified
native eligibility assembly, reviewed job embedding/version handling, durable
admission/deduplication, resource reservations, applying executor/boot attribution,
recovery postconditions/quiescence, retention/tombstones/export/restore and actual
acknowledgement/reboot/power-cut gates remain open. Runtime stays read-only;
M2 acceptance and deferred M1 manual accessibility remain pending.


## Executed qualification

Signed functional/test source `e66d59eec2e05137a11d8533b327206c3682e423` passed seven parent tests
and 769 distinct subtests, including 741 required-shape checks, without skips or
failures. See the [sanitized manifest](property-intent-manifest.json) for exact
source/input/toolchain/case and log provenance. The expected missing-API compile
failure and superseded test-helper Go-vet refusal retain their original evidence.

Local `make review` passed all baseline checks (92 tooling regressions),
storage-only race checks and all 17 compiler-listed storage fuzz targets in
87.101 seconds. No unrelated package deep run or complete milestone suite ran.
The new record and existing ownership/job shape regressions passed together.

A supervised cgo-free FreeBSD/amd64 test binary compiled at the same source and
was removed without execution; this proves portability only. No VM, mount,
filesystem fixture or host service was started. All owned supervisor groups stopped
and teardown was verified. All 27 prior M2 manifests remain byte-identical and
retain their actual sources. Final documentation-only review reruns baseline/
signature/DCO/title/source checks and may reuse this exact-input deep execution.

This qualifies bounded property-review capture and internal codec consistency.
Complete executable intent/job embedding, native eligibility, admission/reservations,
executor attribution/postconditions, recovery/cuts, M2 acceptance and deferred M1
manual accessibility remain open.

The capture prerequisite was accepted in [PR 52](https://github.com/gonz0w/horvium/pull/52), merged source `77d48453cfedd4ec4dfaa1da62dbcfa75e0674e9`. The [property job binding](property-jobs.md) is the separate metadata embedding prerequisite; native persistence and admission still need their own gates.
