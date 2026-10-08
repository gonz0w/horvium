# M2 configured property client

Date: 2026-10-08. Status: test-first; qualification pending.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[original application client](property-client.md), [private listener](property-listener.md).
Partial M2-PRIV-001/M2-PLAN-001/M2-NATIVE-001 evidence only.

## Configured connection ownership

`DialPropertyClient` uses the existing trusted helper `Config`; no new schema,
flag, endpoint or runtime consumer is introduced. Validate configuration, context,
request/correlation syntax, native support and actual real/effective configured
service UID before application checks or filesystem/network I/O. Require the
original live application check before dialing and immediately after dialing.
It supplements rather than replaces actual kernel peer identity.

Apply the unchanged root directory/ancestor and exact root/group/socket policy
before dialing. Retain the original socket file identity; repeat the safe policy
and require the same current socket after dialing, before any protocol bytes.
Do not unlink or adopt replacements. This detects ordinary namespace replacement;
it is not atomic namespace locking or protection against a hostile root process.
The existing concrete peer factory independently checks actual local/service and
remote/root UID before hello and throughout its existing phases.

Create one bounded parent context before the first application check. Dialing,
namespace rechecks, negotiation and the typed operation share its original
at-most-five-minute deadline; a shorter caller deadline wins. Never renew a phase
window. The original returned client owns cancellation of this dial context;
terminal operations, explicit Close and every construction failure retire it and
the socket. Copied clients cannot retire the original. Report genuine close errors;
ordinary repeated descriptor retirement remains harmless.

Private configured admin adapters validate complete typed selections first,
obtain the existing original credential-free session context and install the same
original session checks used by the raw-socket adapters. No credentials travel to
root; context strings remain untrusted correlation there. Both adapter families
remain unregistered. No backend, root application authenticator, policy enrollment,
recovery/global launch clearance, durable admission or native write is enabled.

## Required evidence

Signed denying stubs must fail portable preflight, original lifetime retirement
and configured adapter selection tests. The portable lifetime owner is a private
fixture with no socket or kernel authority. Keep original transport/UID/client/
listener product primitives and the five original case inventories. Optional
client-owned cancellation is added for configured connections. The original
replacement fixture first completes an actual exchange so it cannot mistake
visible startup mode bits for a fully serving listener.

Add twelve actual native cases: denied review/submission through the private root
listener, revocation/cancellation before dial with independently observed empty
accept backlog, wrong real/effective local UID, unsafe directory/socket, replacement
and revocation after actual dial, original shorter deadline through every phase,
cancellation during pending ready negotiation and complete fixture teardown. Children
drop credentials before exec. Callback phase gates model application checks, not
actual human/session authentication or replacement peer UID. Root replacement is a
synchronized safe root/group/socket fixture, isolating file identity rather than
only testing unsafe mode or type. Require no retry on its empty accept backlog; no host/operator storage is involved.

Retain all 92 earlier transport cases. Require all six parents and all 104 exact
cases/result markers on both pinned FreeBSD releases, without expanding the Go
80-second/outer 90-second bounds. Shared checker/selector changes require fresh
full portable review. Preserve historical manifests, actual signed red/green
sources, native executable hashes, base/runtime bytes and verified supervisor/
artifact cleanup. Actions remain disabled; M1 Safari/VoiceOver stays pending.
