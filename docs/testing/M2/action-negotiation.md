# M2 application-context negotiation prerequisite

Date: 2026-10-08. Status: implementation in review; partial portable evidence.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[application authority](session-authority.md).

## Contract

The approved explicitly negotiated transport begins with a separate version-2
hello and correlated ready frame. These codecs and the private application adapter
are unregistered: the existing helper server/client still accept version-1
inventory only. Neither negotiation nor parsing grants permission, calls a
collector, submits a job, accesses metadata or changes storage.

| Direction | Exact required fields | Meaning |
|---|---|---|
| Application hello | `version: 2`, `request_id`, `operation: "negotiate"`, `session_context` | Request the new wire format with correlation data |
| Helper ready | `version: 2`, same `request_id`, `type: "negotiated"`, same `session_context` | Acknowledge wire/context agreement only |

Each JSON object uses the existing four-byte big-endian length framing. The fixed
512-byte payload ceiling is checked before allocation/read; fields are at most
64 bytes and typed depth at most two. Request IDs retain the existing 1–64 byte
ASCII alphanumeric/hyphen/underscore rule. Context is exactly 64 lowercase hex
characters from the original live session. All fields occur once with exact case;
missing/null/unknown/duplicate fields, bad types/UTF-8, trailing JSON, old/future
versions, unsupported operations and mismatched replies fail closed. No fallback,
retry or successful operation result is encoded.

Parsing a canonical context supplies an **untrusted service assertion**, not
session proof, kernel UID, human identity, enrolled scope, recovery clearance or
a launch capability. The low-level writer accepts correlation strings, without
attesting them. Only the private application adapter obtains its context through
the original opaque `SessionAuthority`; no setup token, cookie or CSRF secret
crosses this codec. It rechecks that original session after writing and after the
reply, never extending idle expiry. Revocation during writes prevents awaiting
a reply. A partial/failed exchange cannot be retried by this adapter.

One frame is consumed per direction. Later bytes remain for a future typed
exchange and are not dispatched here. The eventual socket owner must authenticate
the root peer before sending, authenticate the exact nonroot management peer
before helper parsing, enforce deadlines and whole-connection budgets, close
failed connections and independently apply policy/recovery/global launch gates.
Generic `io.Reader`/`io.Writer` codecs themselves cannot interrupt blocked I/O or
attest kernel peers. No runtime caller or operation schema is introduced.

## Evidence and limits

Twelve parents/57 children cover exact bytes and correlation, malformed/forged
claims and downgrade denials, every truncation point, preallocation bounds,
writer failures, version-1 isolation and an owned temporary Unix-socket rejection
by the existing server. The server test injects its UID: it establishes portable
dispatch behavior, not actual FreeBSD kernel identity. Application tests use real
login and authenticated POST issuance, nil/zero/decoded/revoked authorities,
mid-exchange logout/expiry/cancellation, no credential transmission or idle
renewal, and no read/retry after failed writes. `FuzzActionNegotiation` exercises
both frame readers with bounded 516-byte inputs and accepted-request round trips.
The successful exchange tests first failed against denying contract stubs.

Focused review and FreeBSD compilation retain their actual source in the
[qualification manifest](action-negotiation-manifest.json). Existing native peer, metadata,
collector/control, fixtures and shared tools remain unchanged; unrelated QEMU
suites are not rerun for these unregistered codecs. Compilation is not execution.
Actual negotiated kernel peer binding, root recovery/global launch, typed job
operations, operator workflows and assembled M2 qualification remain open.
M1 Safari/VoiceOver stays pending; Actions stay disabled.

Executed partial evidence: signed source `e449b8c005f2f4635f471305a6e2cf8ab11a1100` passed
all twelve parents/57 children and three fuzz seeds, baseline/125 tooling tests,
eight affected race packages and all twelve bounded fuzz targets in 67.911
seconds. Both admin/helper FreeBSD test packages compiled without execution;
checksummed artifacts were removed. Review supervisors and all 612 owned
sessions stopped with verified teardown; Actions remain disabled. All 68
historical M2 manifests and 386 existing Go/tooling inputs are byte-identical
to the merged base. See the [manifest](action-negotiation-manifest.json).
