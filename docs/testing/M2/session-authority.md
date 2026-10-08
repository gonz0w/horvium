# M2 live application-session authority

Date: 2026-10-08. Status: implementing; portable qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-PLAN-001, M2-PRIV-001 and M2-HTTP-001 evidence at T1/T2.
Runtime storage writes remain disabled.

## Application boundary

The coordinator's context string does not authenticate an application session.
`admin.SessionAuthority` supplies an opaque volatile prerequisite from the original
M1 administrator authentication store. It proves neither a human identity nor
kernel peer, root policy, enrollment, executor quiescence, recovery or global
launch clearance. The private native backend still has fixture-only external
clearance; it consumes none of this new authority today.

`Gate.ProtectAction` is an explicit POST-only wrapper for future typed adapters.
It registers no runtime route and executes no storage, helper or native operation.
Require the existing Host/Origin/fetch-site/header policy, an unambiguous canonical
session cookie, bounded form media/body parsing and exact CSRF before issuing a
capability. Query parameters and cancelled requests cannot mint authority. Only
this wrapper places it in a private request-context key, with the existing CSRF
contract. Ordinary `Protect`, `ProtectPage` and public handlers mint no authority;
an unprotected request, credential-looking field or decoded object cannot supply
it. Operation-specific adapters must still validate their complete typed inputs.

Each successful login allocates a private nonzero-size session identity, bound to
its original `Auth`, credential fingerprint and original creation time. Mint one
random 256-bit canonical hexadecimal context for that same original entry, distinct
from the setup token, every live session/CSRF/context and retained challenge.
Entropy failure/collision, changed membership, expiry or cancellation installs
nothing. Repeat original membership, civil clock, expiry and context checks after
entropy acquisition; it may take time. Repeated authorized requests share the same
binding/context rather than minting replacement launch authority.

`RequestSessionAuthority` reads only that private context value. Copies of the
opaque object share its original binding. `CheckLive` and `Context` independently
lock/revalidate the original authentication store, exact session identity,
credential fingerprint, creation time, binding and current idle/absolute bounds.
They never touch idle expiry or retain the HTTP request context. A caller supplies
its own context for each check; request cancellation denies that check and does
not revoke otherwise valid credentials. Accepted work's eventual lifetime/outcome
policy remains a separate executor contract.

The returned context is correlation data, not a bearer cookie or proof that the
helper can trust. Credential strings stay private and never enter this API's
projection. JSON produces an empty object; decoding it yields no capability.
Default formatting exposes no context or credentials. A string/copy of public
session fields cannot reconstruct the private binding. No transport serialization,
helper session registry, audit schema or persistent authentication is introduced.

Logout, expiry pruning and `Close` permanently retire original action identity
before deleting credentials. Replacing/copying/restoring an entry cannot transfer
its binding to another authentication store or revive an old handle. A civil
clock before original creation/touch/last observation, zero/unavailable clock or
unsupported year blocks that identity permanently. Normal M1 session checks also
record a private clock witness; intervening read touches cannot hide rollback.
Read/login return behavior remains unchanged, including reads without authority
entropy. All state remains bounded by eight active sessions and existing M1 caps.

## Required verification

Start with an authenticated action failing against a denying contract stub.
Additional regressions must first expose changed CSRF/creation-time acceptance
and revival after logout/expiry/unavailable clock, then pass after exact original
binding and sticky retirement are implemented. Preserve all M1 authentication,
HTTP provenance/CSRF/body-cap and consumer behavior tests.

Nine new parents and 51 children require stable credential-free context, normal
copy/live checks, no idle extension, serialization/formatting denial, context-as-
cookie refusal, seventeen request denials, eleven revocation/invalid-context cases,
ten original-entry/copy/reconstruction denials, ten entropy/final-check cuts,
32 concurrent requests, read-gate isolation and sticky read-clock/retirement checks.
`FuzzSessionAuthorityActions` bounds sequences to 32 actions and exercises live
checks, copies, logout/close, expiry/rollback, cancelled contexts, JSON decoding,
ordinary touches, ambiguous requests and reconstructed records.

Run the required baseline and affected-package race/all-selected-fuzz review at a
clean signed source. Compile the FreeBSD target without claiming native execution,
remove its artifact and verify owned review/lab teardown. Native metadata, peers,
collectors, controls, fixtures and shared tools remain unchanged; do not rerun
unaffected QEMU storage suites or relabel their historical evidence. Qualification
of new root/kernel or exposed runtime behavior still requires its actual native
and browser/continuity/recovery gates.

## Remaining gates

No route, CLI operation, runtime backend consumer, negotiated mutation protocol,
actual kernel-peer application-context binding, production identity/RBAC/HTTPS,
root recovery/global launch fence, enrollment provisioning, reservation release
or operator storage write follows. M1 Safari/VoiceOver remains pending. The next
root transport slice must bind actual kernel-authorized service UID to explicit
application context; this local object cannot be serialized as root authority.
