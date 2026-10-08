# M2 typed property request prerequisite

Date: 2026-10-08. Status: implementation in review; partial portable evidence.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[original native peer](action-peer.md), [property coordinator](property-coordinator.md).

## Contract

Two unregistered version-2 request codecs define the approved property review and
submission selections. Each envelope requires exactly `version: 2`, `request_id`,
`operation` and an object `data`. All fields use exact case and occur once; unknown,
duplicate, missing/null fields, wrong types, malformed UTF-8, trailing JSON and
old/unknown versions/operations fail closed. Parsing returns zero output on error.

| Operation | Exact required data fields | Meaning |
|---|---|---|
| `property-review` | `pool_guid`, `filesystem_guid`, `property`, `value` | Request fresh helper-owned review of one literal property intent |
| `property-submit` | `pool_guid`, `plan_id`, `submission_id` | Request admission of an original root-held plan with an idempotency identifier |

GUIDs are positive canonical decimal uint64 strings. Request IDs retain the
existing 1–64 ASCII alphanumeric/hyphen/underscore rule. Plan/submission identifiers
are exactly 64 lowercase hex characters; their syntax does not prove issuance or
ownership. The literal property validation shares the planner's allowlist:
`quota`/`refquota` accept positive canonical uint64 bytes or `none`; `compression`
accepts only `off`/`lz4`. Names, paths, arbitrary property maps, commands, policy,
UID/session assertions, recovery claims, replacement submission intents and
client-chosen job IDs cannot cross these schemas. Pool selection is untrusted:
the owner must independently resolve an enrolled original pool/backend and require
the original plan's exact pool, session, intent and current policy.

The existing four-byte big-endian frame has a fixed 1024-byte JSON payload ceiling,
checked before payload allocation/read, with 64-byte fields and depth at most
three. Writers validate/bound inputs before JSON framing or output; failed/partial
I/O is returned without retry. The reader requires an independently supplied
original negotiated request ID, consumes one frame and leaves later bytes alone.
These are per-frame limits; the future owner still needs whole-exchange budgets,
bounded admission, deadlines and original socket/session checks.

No runtime consumer, response codec, listener, HTTP/CLI route, durable publication
or native operation is enabled. The existing inventory server/client still refuse
these requests. The original peer's quiet check still rejects unexpected later
data: integrating a typed read requires an explicit owned protocol phase rather
than treating this standalone parser as a live peer or launch permit. Durable
admission and subsequent job execution retain the coordinator/backend's independent
recovery/global launch obligations. Logout/HTTP disconnect semantics are unchanged.

## Evidence and limits

Behavior tests cover allowed round trips, canonical maximum values, exact
correlation, operation-specific schema isolation, privilege claims, malformed
fields/envelopes, unsupported literal intents and noncanonical identifiers, every
truncation point, preallocation bounds, preservation of later bytes, nil/short I/O
and inventory isolation. Bounded `FuzzPropertyRequest` requires rejected input to
produce zero output and accepted input to round-trip through the appropriate writer.
Successful cases first failed against denying stubs.

Run the baseline and focused affected-package race/fuzz review on signed source.
Cross-compile helper/storage tests for the FreeBSD target without claiming execution.
This change adds pure unregistered codecs and a syntax validator; existing native
peer, storage effects, fixtures, tooling and all historical evidence remain intact.
No QEMU rerun is required for a new kernel/native behavior claim because none is
made here. Actual typed socket exchange, replies, root recovery/global launch,
operator workflows and assembled M2 acceptance remain pending. M1 Safari/VoiceOver
stays pending; Actions stay disabled. Source-bound evidence is recorded below.

Executed partial evidence: signed source `2c5d89680f6be5be790c92c741f8f965f2f5bf2a` passed
five behavioral parents/61 children and three fuzz seeds, baseline/131 tooling
tests, 9 affected race packages and all 41 bounded fuzz targets in 227.424
seconds. Helper/storage FreeBSD test packages compiled without execution and their
checksummed artifacts were removed. All 623 owned sessions stopped with
verified teardown; Actions remain disabled. All 70 historical M2 manifests and
393 protected existing inputs retain actual bytes. Existing planner validation is
unchanged apart from the new bounded syntax entry point. The earlier lint failure
is explicitly unqualified and corrected before this fresh review. See the
[source-bound manifest](property-requests-manifest.json). No native typed exchange,
root recovery/global launch or assembled acceptance follows.
