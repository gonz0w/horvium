# M2 original native action-peer prerequisite

Date: 2026-10-08. Status: test-first implementation in review; not yet qualified.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[negotiation](action-negotiation.md), [local runner](../local-qualification.md).

## Contract

Two private factories take ownership of an already accepted/connected original
Unix socket. The helper requires actual FreeBSD/root and the exact configured
nonroot management peer UID before reading; the application requires its actual
configured nonroot UID and kernel root peer before writing. Neither user fields,
root clients, shared groups nor repaired socket file ownership can replace kernel
identity. Constructors, decoded structs and strings cannot mint a live peer.

Use the existing strict version-2 hello/ready with the original request/context
correlation. A fixed trusted timeout (positive, at most five minutes) bounds the
connection context and both read/write deadlines. Cancellation closes the socket;
denial or partial exchange closes it without retry. Success retains only the
original connection, context, kernel UID and correlation tuple. Projection repeats
live/UID checks. Idempotent close retires the binding; timeout, cancellation,
local identity change, disconnected peer or unexpected later data deny further
use. A nonblocking peek detects observed EOF/data without consuming an operation.
This is local connection liveness, not peer process containment or recovery proof.

These factories remain unregistered: they create no listener, install no service,
change no socket policy or runtime route, and grant no storage operation. The
future transport owner must retain existing root socket/directory safeguards,
bounded admission, authenticated original application session and independently
validated policy/recovery/global launch clearance. The parsed canonical context
is an assertion by the authorized service, not a bearer secret or independently
verified human/session identity. No general command/path/property map exists.

## Test and qualification boundaries

The native parent `TestNativeActionPeer` runs eighteen named children with an
actual root listener and separately reexecuted configured/wrong/root UIDs. It
checks exact UID/request/context projection, service-asserted context semantics,
pre-read/pre-write denials, forged UID/version/operation rejection, truncated and
idle streams, local UID mismatch, cancellation/expiry after negotiation, peer
disconnect, extra data, repeated close and owned socket/binary fixture teardown.
For the nonroot-helper attack root relocates and repairs an unprivileged listener
socket's file ownership; kernel credentials must still reject it before sending.
Portable tests reject nil/zero/decoded peers and missing connections.

The focused `action-peer` runner cross-builds one checksummed helper executable,
uses the existing snapshot/loopback guest supervisor and installs no guest
packages or host services. A strict checker requires every actual PASS child and
one complete result; skipped/failed/missing/duplicate assertions never qualify.
New executable cleanup is covered in both normal and interruption paths.
Shared tooling changes require fresh full portable review; both FreeBSD releases
are tested serially. Native results retain actual source/artifact/guest/runtime
hashes. Earlier suite evidence retains its original sources and deadlines.

Actual red/green results and teardown are recorded after execution. Root policy,
executor-death/reboot recovery/global launch, typed operations, operator workflows,
assembled native safety/continuity/resources and M2 acceptance remain open.
M1 Safari/VoiceOver stays pending and GitHub Actions stay disabled.
