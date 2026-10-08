# M2 owned property exchange prerequisite

Date: 2026-10-08. Status: partial native transport qualification; maintainer review pending.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[native peer](action-peer.md), [requests](property-requests.md), [replies](property-replies.md).

## Contract

Private factories own one original Unix connection, including failure. Retain
actual FreeBSD, real/effective local UID, exact configured nonroot helper peer and
kernel root application peer checks before reading/sending negotiation. A dedicated
property phase permits the request to arrive after ready; the existing negotiation-
only factories and their quiet behavior stay unchanged. All subsequent identity
checks use the original descriptor/context/correlation, with no injected UID hook.

One connection carries exactly one hello, one ready, one typed review/submission
request and one terminal review/admission/refusal reply. Root receives one strict
request, rechecks live kernel identity and rejects queued extra bytes before
returning parsed assertions and an identity snapshot. It can reply only to the
stored original request: exact pool/intent for review and pool/plan/submission for
admission. These remain metadata assertions, never original plan, enrollment,
durability, native eligibility, recovery or launch proof. No backend dispatch is
registered and native fixture replies use synthetic portable display metadata.

The trusted positive timeout, at most five minutes, bounds the original context
and socket deadlines for all phases; no phase renews it. Cancellation/explicit close
interrupt blocked I/O. Phase ownership uses nonblocking exclusivity: overlapping,
wrong-role or second operations deny and retire the socket, rather than queueing a
second request. Errors consume the attempt; there is no fallback or retry.

Existing fixed payload caps apply: hello/ready at most 512 bytes each, request at
most 1024 and reply at most 65536. Four frames total bound aggregate JSON to 67584
bytes, plus sixteen framing bytes. The root closes after its terminal reply. A
client requires actual EOF after the complete correlated reply under the original
deadline; extra bytes or a peer withholding EOF produce zero metadata and no valid
refusal result. EOF is expected terminal framing, not proof of a live root process.
The client repeats original descriptor UID/context checks without asserting socket
liveness after terminal EOF, then retires its binding. A valid refusal remains a
bounded peer assertion, without safe-retry/absence-of-admission meaning.

No listener, admission manager, HTTP/CLI route, application-session adapter,
collector, root policy/store operation or storage command is added. Future assembly
must retain safe root-owned socket/directory policy, bounded concurrent admission,
original authenticated application session checks, enrolled backend selection,
root-held plans and independent recovery/executor-absence/global launch clearance.
The negotiated application context is an authorized service assertion; no human
identity or session credential is independently attested at this boundary.

## Qualification plan

Portable tests deny nil/zero/decoded owners and close owned sockets on unsupported
hosts. A new native parent runs separately from all eighteen unchanged original
peer cases. It checks real configured/root/wrong UIDs, repaired file ownership with
an unprivileged helper, local UID mismatch, complete synthetic review/admission/
duplicate/refusal exchanges, queued post-ready input, single-operation ownership,
malformed/miscorrelated/extra/truncated/oversize requests, idle deadline, cancellation,
explicit close, overlap, wrong reply identity/selection, extra terminal bytes,
withheld EOF and complete fixture teardown. Child credentials are applied before
reexec/connect; no live disk or metadata history is used.

The existing focused peer runner selects both actual native parents with its
unchanged 80-second Go / 90-second outer deadlines. The checker requires each exact
parent/child/result once; old peer-only, skipped/failed/missing/duplicate/unknown
proofs cannot qualify the new exchange. Shared checker/selector edits require fresh
full portable review and serial qualification on both pinned FreeBSD releases.
All previous native fixture code and historical evidence retain their sources.
Source-bound red/green results and teardown are recorded below. M1 Safari/VoiceOver and
assembled M2/native safety/recovery/continuity/resources remain open; Actions stay
disabled and storage writes remain unregistered.

## Executed evidence

Actual FreeBSD 15.1 execution against signed denying-stub source
`21b5bcfa982310c3b9e4ab9689f7e2b41a952495` passed all eighteen original peer cases
and failed the new exchange parent. This test-first run remains unqualified;
fixture/base/runtime preservation and supervised teardown were verified.

Signed implementation `f26e51413b8325e2ffa3d7681f0213564f108c4f` passed all eighteen
original peer cases and all twenty-six exchange cases on both pinned releases.
Each host-cross-built FreeBSD/amd64 executable was hash checked before actual
guest execution. Both complete case inventories, preserved base/runtime, fixture
cleanup, removed executables/fixture keys and stopped process groups were verified.
Replies were synthetic display views; no storage backend or command was exercised.

Fresh full local review passed `make check lint test secrets`, all 132 tooling
tests, all eleven portable race packages and all 49 bounded fuzz targets. The
earlier restricted-sandbox tooling attempt could not bind supervisor sockets and
is not qualification evidence; the fresh supervised baseline used the required
socket permissions. All 72 historical M2 manifests and 374 protected existing
internal/tooling inputs retain actual bytes. All 634 owned sessions were stopped
with verified teardown at this evidence snapshot. Actions remain disabled.
See the [manifest](property-exchange-manifest.json). The documentation followup
requires a fresh baseline and may reuse only this successful deep evidence when
its exact behavioral inputs/base/toolchain remain unchanged.
