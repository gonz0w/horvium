# M2 native application property client

Date: 2026-10-08. Status: implementation qualification pending.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md) and [Test Design](../../milestones/M2/TDD.md).
Partial M2-PRIV-001/M2-PLAN-001/M2-NATIVE-001 evidence.

## Application contract

Expose the existing configured adapters as internal Go APIs
`ReviewConfiguredProperty` and `SubmitConfiguredProperty`. Keep their typed
selection, original opaque SessionAuthority, trusted helper Config, session-phase
checks, metadata withholding and single original deadline/connection contracts.
No HTTP route, CLI command or service consumer is registered. The root listener
continues to deny every typed operation; root correlation is an untrusted service
assertion, not independent authentication of a human or session.

## Native qualification design

Add an external test-package child to the existing helper test executable. Drop
actual real/effective UID before exec. Use public Auth challenge/login and
ProtectAction, with random in-memory credentials, real clock and entropy, to
obtain the original authority. The HTTP request is an in-memory fixture, not a
running server or registered route. Never emit tokens, cookies, CSRF or contexts.
No private Auth fields, forged binding, callback phase counter or UID hook is used.

Require eleven new cases: denied review and submission through the actual private
root listener; logout, Auth.Close, decoded authority, invalid selection and
cancelled context before dial, with independently observed empty accept backlog;
logout after actual hello but before ready; logout and Auth.Close after an actual
typed request but before the root refusal; complete fixture teardown. Synchronize
revocation through fixed pipe messages, acknowledge original-store revocation
before root continues, and require the actual revocation error rather than a root
refusal or generic transport error. Revoked exchanges return no metadata and the
original authority stays revoked. Late-phase root fixtures use the original
kernel peer, bounded admission and codecs and grant no backend authority.

Retain all 104 earlier cases. Require seven parents and all 115 exact cases/result
markers on both pinned disposable releases under the unchanged Go 80-second and
outer 90-second bounds. Shared checker changes require fresh full portable review.
Signed denying stubs must fail the configured portable contract and native
assembly cases. Preserve actual source-bound failed evidence, historical manifests,
base/runtime bytes and verified guest/supervisor/artifact teardown. GitHub Actions
remain disabled. M1 Safari/VoiceOver remains pending by maintainer choice.

## Remaining gates

This slice covers actual native application authority and client integration.
Independent root authentication/backend/policy/plan, recovery/executor absence,
global launch, durable admission, runtime workflows and full M2 acceptance remain
open. It adds no root writer, storage effects, installable service or hardware claim.
