# M2 session-bound property client

Date: 2026-10-08. Status: test-first; qualification pending.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[application authority](session-authority.md), [admitted exchange](property-admitted.md).

## Contract

Connect an original application `SessionAuthority` to the actual kernel-authenticated
property client. The private admin adapters validate the exact typed selection
before negotiation, obtain the original credential-free context and install a
callback that compares the same original authority/context on every phase check.
Cookies, setup tokens and CSRF stay inside the authentication store. Checks never
touch idle time or reconstruct authority from a string, JSON or public session fields.

`helper.PropertyClient` owns the supplied Unix socket on entry, including every
failure. It uses the actual FreeBSD root peer/local configured service UID checks,
original bounded context/deadline and unchanged framing/typed codecs. Its trusted
application callback supplements those checks; it cannot replace kernel identity
or authenticate a service assertion at root. Nil callbacks fail closed. Only its
original opaque owner can operate/retire the exchange; copies/decoded/zero objects
cannot use or close an unrelated original. One request and one terminal reply are
allowed, with no automatic retry or protocol fallback.

The original public client closes guarded exchanges after the typed call returns,
joins its first close error and withholds metadata on failure. Unguarded private
clients retain their existing terminal closure. Native checks distinguish actual
local descriptor closure from remote EOF before any deferred fixture cleanup or
second operation. A genuine close syscall error cannot safely be forced through
this concrete owned Unix socket without bypassing descriptor ownership; error
joining is reviewed directly, without introducing a close or kernel UID hook.

Check the original session before hello, after hello before waiting for ready,
after ready, before the typed request, after writing before waiting for a reply,
and after the terminal EOF before returning any metadata (including a refusal).
Changed context, logout, expiry, rollback and cancellation deny acceptance.
Active I/O remains bounded by the original socket deadline/cancellation; session
revocation is checked at phase boundaries, without a new polling worker or push
revocation channel. Bytes already sent cannot be undone; a closed connection or
late application refusal is no proof of no durable admission or safe native retry.
Accepted jobs still need their independent durable lifetime/execution policy.

No route, CLI operation, dialer, listener, runtime service, backend dispatcher or
native write is registered. Root must independently validate enrolled policy,
original plans/coordinator, recovery/executor absence/global launch and durable
admission. Session and transport success are prerequisites, not full M2 acceptance.

## Verification

Failing-first portable tests use the original login/authenticated action fixture
for stable authority/context, no idle extension and nil/zero/decoded/mismatched/
revoked/cancelled/expired/clock-rollback denial. Original factories retire invalid
or unsupported sockets without bytes. Native fixture callbacks model phase cuts;
they are explicitly not fabricated kernel identity or real human authentication.
Actual configured nonroot reexec clients talk to root over Unix sockets and check
review/submission/duplicate/refusal metadata, every phase cut, copied owner denial,
invalid selection, bounded missing reply and full fixture teardown. Synthetic
metadata never reaches an actual storage backend.

Keep the original 60 peer/exchange/admission cases and add fourteen client cases
as a separate parent. Strict evidence requires all four parents and exact once-only
case/result inventories within unchanged Go 80-second/outer 90-second bounds.
Shared selector/checker maintenance requires fresh full portable review and serial
qualification on both pinned FreeBSD releases. Preserve all historical manifests
at their actual sources. M1 Safari/VoiceOver remains pending; Actions stay disabled.
