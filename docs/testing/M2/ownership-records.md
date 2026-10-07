# M2 strict ownership record codec

Date: 2026-10-05. Status: partial codec accepted in
[PR 40](https://github.com/gonz0w/horvium/pull/40).
Authority: [approved M2 EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Current boundary

`storage.EncodeOwnershipRecord` and `storage.DecodeOwnershipRecord` implement
the current version 1 ownership metadata format using Go's standard library.
They operate on caller-owned bytes and the existing `Policy` model. They do not
open a directory/file, enroll resources, acknowledge a job, admit work or invoke
ZFS. The runtime has no record-store wiring. A decoded policy still requires
trusted root-controlled file provenance, actual enrollment, fresh eligibility and
native identity/recovery reconciliation before any write can be authorized.

This is an independent partial slice 2 prerequisite, following the
[state-directory lease](state-directory.md) accepted in
[PR 39](https://github.com/gonz0w/horvium/pull/39). Missing slice 1 observations do
not become write authority through serialization. File provenance, ACL-qualified
provisioning, atomic publication and durable jobs remain separate open gates.

## Version 1 ownership format

Every field below is required and nonnullable. An empty `scopes` array is valid
and confers no filesystem scope. Unknown, missing, duplicate and case-variant
keys are refused at every object level. JSON escapes resolve before matching
keys, so an escaped spelling cannot hide a duplicate. Object-field order and
whitespace may vary; multiple top-level values and malformed UTF-8 are refused.

| Object | Fields |
| --- | --- |
| Envelope | `version`: integer 1; `kind`: string `ownership`; `policy`: policy object; `checksum`: lowercase 64-character SHA-256 hex |
| Policy | `revision`: nonzero uint64; `pool_guid`: canonical nonzero decimal GUID string; `pool_name`: supported pool name; `scopes`: array of scope objects |
| Scope | `guid`: canonical nonzero decimal filesystem GUID string; `name`: supported dataset name in this pool; `parent_guid`: empty for the pool root filesystem, otherwise canonical nonzero decimal GUID; `properties`: nonempty array of distinct `quota`, `refquota`, `compression` strings |

Integer fields reject strings, fractions, exponent notation, negative values and
overflow. Version 0, older/newer unknown versions, other record kinds and null
objects/arrays/elements fail. No migration, repair or guessed-default fallback
exists. Errors return no policy/encoded bytes and leave inputs unchanged.

Pool and dataset names use the already accepted planning subset; native names
outside it remain observable elsewhere but cannot gain ownership here. Duplicate
dataset names/GUIDs, foreign names, self/missing parents and invalid property
scopes fail. If a parent is also selected, its name must match the direct parent
address. A selected child need not enroll its parent; an unselected parent GUID
is retained for later independent native binding. The codec neither expands
parent/descendant scope nor claims that these internally consistent identities
exist on a host.

Canonical encoding orders scopes lexicographically by GUID string and properties
lexicographically by name, preserving all identities and explicit scope. It copies
caller slices and uses explicit private wire types, so changing the planning
struct's Go field names does not implicitly change the persisted format.
The checksum covers `horvium:ownership:v1` followed by a newline and compact JSON
with fixed field order: `version`, `kind`, `policy`; the checksum field itself is
excluded. The canonical policy has the field order listed above. Presentation
changes or equivalent scope ordering do not alter that digest. The digest detects
damage; a root actor can recompute it, so it provides no authentication or enrollment.

Input and encoded output may not exceed 64 KiB. Before copying/sorting an input
policy, the encoder checks count/field ceilings and a conservative 64 KiB budget
including structural overhead. The decoder applies the same semantic budget.
Some policies whose JSON alone fits the ceiling can therefore be refused; none
is truncated. Revision values retain full uint64 precision. JSON token traversal
follows this fixed schema and refuses unknown/wrong structures before recursion;
arbitrary nested input cannot increase the schema depth.

## Tests and acceptance limits

Tests first failed because both codec entry points were absent. Behavioral tests
cover independently computed canonical bytes, round trips, input immutability,
caller/decoder aliasing, scope order, escaped keys, whitespace at the exact byte
ceiling, malformed/truncated/trailing data, unknown/duplicate/case/missing fields,
nulls, unsupported version/kind, numeric type/precision/overflow, malformed UTF-8,
invalid names/identities/parent/property scopes, checksum damage, capacity refusal
without truncation and no implicit enrollment. Independently recomputed checksums
on invalid records still fail semantic/schema validation. A deeply nested wrong
shape is refused; `FuzzOwnershipRecord` requires every accepted input to round-trip
within bounds and every denial to return no policy. It is part of the full local
review gate, which now runs twenty-one bounded fuzz targets.

Traceability: partial M2-OWN-001 and M2-STATE-001 at T1. These tests have no native
filesystem effect. FreeBSD/amd64 compilation is a portability check, not an actual
FreeBSD file/store qualification. PR 39's thirty-five native directory assertions
retain their original tested source and immutable manifest; they are not rerun or
relabeled as record-store evidence here.

## Executed evidence

Signed/DCO functional source `e9f685c2f4460b7b4545bfa761c9089a2743439a`, based on
merged PR 39 source `b84e0572d389078ec0b8514ff4d7919b56b96b7f`, passed all five
behavioral tests and fifty-seven subtests with no failures/skips. Complete local
`make check lint test secrets go-deep-check` passed seventy tooling tests, Go
unit/race checks and all twenty-one bounded fuzz targets, including the new codec.
The [manifest](ownership-records-manifest.json) records those results and hashes,
eighty-four unchanged code/test/tool inputs and seventeen unchanged prior M2
manifests. Final documentation-head review is separate; these results retain the
actual functional source.

The same source compiled the storage test binary for FreeBSD/amd64 using pinned
Go 1.27.1 with cgo disabled. The binary was hashed and removed; it was not executed.
Behavioral execution and full review used the development Mac (Darwin/arm64).
This slice starts no VM or host service. The supervised review stopped its owned
process group; Actions remain disabled. Native file/store qualification still
requires disposable FreeBSD execution when those operations are implemented.

Strict job/outcome/history schemas, record enumeration/file ownership/ACL checks,
same-directory sync/rename publication, full-store/read-only/I/O failure, duplicate
admission/tombstones, reservations, acknowledgement cuts, reboot/power-loss recovery,
metadata export/restore and helper/runtime wiring remain ahead. M2 slice 2 and
assembled acceptance are incomplete. The management runtime remains read-only;
M1 Safari/manual accessibility stays maintainer-deferred. No installation/service,
dependency, physical-controller or commercial support claim follows.

## Version 2 extension

The [explicit member contract](property-member-enrollment.md) adds a second strict
schema with required nonempty member baselines. This document and its manifest
retain the original version 1 qualification. Version 1 canonical bytes, checksum
domain and readable policy semantics remain unchanged; no member identity is
invented. A legacy policy cannot pass the complete property safety enrollment
gate. Version 2 uses its own checksum domain and preserves existing scope checks.
