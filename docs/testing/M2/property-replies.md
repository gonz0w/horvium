# M2 bounded property reply prerequisite

Date: 2026-10-08. Status: implementation in review; partial portable evidence.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[typed requests](property-requests.md), [native peer](action-peer.md).

## Contract

Unregistered version-2 replies use the existing four-byte big-endian framing.
Each envelope requires exactly `version: 2`, `request_id`, `type` and object `data`.
Unknown/duplicate/missing/null fields, wrong case/types, invalid UTF-8, trailing JSON,
unknown versions/types and mismatched original request correlation fail closed.

| Reply type | Required data | Meaning |
|---|---|---|
| `property-reviewed` | Complete explicit review view | Display the exact planned intent and observed baseline |
| `property-admitted` | Complete explicit `job` view and boolean `duplicate` | Report lifecycle metadata and the helper's admission/idempotency assertion |
| `refused` | Only `code` | Report one bounded refusal assertion |

The review view preserves plan ID, creation/expiry, policy revision, exact pool and
literal intent, filesystem context/property sources, affected resources, versions,
mount/encryption states, pool incarnation/properties, full feature observations and
vdev topology/errors. It reuses the existing explicit review wire fields and bounded
consistency validator. This is a display projection, not a captured property intent:
no session/policy digest, capture timestamp, private proof or checksum is included.
A temporary private carrier uses creation time solely for that consistency check;
it is never exposed or used for authority. Owned returned views cannot alias original
plans. Expiry/session/policy/native eligibility still need independent live checks.

The job view preserves the ID, original plan/submission/pool/policy/literal intent and
complete lifecycle history under the existing job rules. It omits the durable record
version/kind/checksum envelope and private captured baseline. A later duplicate reply
may show valid later history; state labels and `duplicate` do not prove durable
publication, authorize native retry or release a reservation. Review readers require
exact expected pool and literal intent; admission readers require exact expected
pool/plan/submission identifiers. Expected selections come from the original request.

Refusal codes are exactly `denied`, `unavailable`, `review-required`, `limit-exceeded`
or `uncertain`. Readers return zero metadata and a typed `PropertyRefusalError` for
a valid correlated refusal; unknown/malformed refusals remain protocol errors.
There is no free-form message, native error/path, setup credential or retry directive.
Refusal alone cannot prove that a submission was never admitted. The future owner
must determine actual publication uncertainty and durable matching-job lookup.

The fixed reply payload ceiling is 64 KiB including its envelope, with 4096-byte
fields and depth at most sixteen. Length bounds precede payload allocation/read;
writers validate/bound views before JSON and reject envelope overflow before output.
Complete views/history are never truncated to fit. A model view that fits its own
64 KiB cap may still be refused if the enclosing reply exceeds the frame cap.
One frame is consumed; later bytes remain for the owner. Partial/failed writes return
without retry. Standalone codecs do not impose deadlines or whole-exchange budgets.

No socket phase, runtime dispatcher, HTTP/CLI route, admission or native write is
connected. Existing inventory transport and native quiet checks are unchanged.
Before integration, independently authenticate the original kernel peer and
application context, own the typed read/write phases with bounded admission/deadlines,
and retain enrolled root policy, original coordinator, recovery/global launch gates.

## Evidence and limits

Test-first cases cover complete review/lifecycle round trips, owned-copy isolation,
no private metadata transmission or durable-record reinterpretation, existing
baseline consistency, exact selection/correlation, malformed/unknown schemas,
refusals, allocation/output limits, truncation, later-frame preservation, nil/short
I/O and no retry. Two bounded fuzz targets exercise view and framed reply decoding.
The synthetic helper fixture was generated from the pre-extraction storage review
fixture, never from an operator's inventory. Golden regression hashes independently
compare existing intent/property-job records against merged PR 95's synthetic
fixtures; the shared converters retain the existing durable schemas and checksums.

Run baseline/focused affected-package race/fuzz checks and unexecuted helper/storage
FreeBSD compilation. Existing native controls, peer checks, fixtures and shared
tooling remain unchanged; this portable codec/converter slice makes no new native
execution claim and does not rerun QEMU. Source-bound evidence is recorded below.
Owned exchange, root recovery/global launch, operator workflows, native assembled
safety/continuity/resources and full M2 acceptance remain open. M1 Safari/VoiceOver
stays pending; Actions stay disabled.

Executed partial evidence: signed source `728387ba21d322f5ee42426bdf617d527e2afdda` passed
twelve behavioral parents/36 children and five fuzz seeds, baseline/131 tooling
tests, 9 affected race packages and all 43 bounded fuzz targets in 225.190
seconds. Helper/storage FreeBSD test packages compiled without execution; exact
checksummed artifacts were removed. Independent old/new synthetic intent/property-job
record hashes match merged PR 95 and are guarded by golden regression tests.
All 71 historical M2 manifests and 395 protected existing inputs retain actual
bytes. All 628 owned sessions stopped with verified teardown; Actions
remain disabled. The earlier native-selector naming collision is explicitly
unqualified and corrected before this fresh review. See the
[source-bound manifest](property-replies-manifest.json). Owned exchange, root
recovery/global launch, actual admission and full M2 acceptance remain pending.
