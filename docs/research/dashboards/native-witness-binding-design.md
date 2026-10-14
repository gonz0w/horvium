# Original native witness design inspection

Status: Source/portable evidence for a proposed mechanism, not implementation.
Date: 2026-10-13. Owner: Cam (`@gonz0w`), [issue 382](https://github.com/gonz0w/horvium/issues/382).
[ADR](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
[manifest](native-witness-binding-design-manifest.json).

The inspected source is maintainer merge `3bdfda1797d9be6b88061005f577fe39b13daa05`
of PR 386. The manager's generated ID is passed unchanged into helper collection
and becomes the published generation only after successful copying/publication.
Every helper response record carries that same request ID, with strict bounded
begin/item/end/count/completeness/EOF checks. This supplies an explicit original
correlation that does not depend on timestamps, equal values or response order.

Existing portable behavioral groups for startup single-flight and ID matching,
failed-source retention, generation paging/expiry/last-complete, publication-lock
cancellation, framing round trip, request ambiguity/authority, incomplete/bounded
framing and malformed response rejection passed through owned focused testing.
They exercise actual core behavior, rather than a newly invented broker model.
Logs/source hashes are bound in the manifest. These tests do not execute a broker,
peer-PID ABI, descriptor handoff, tmpfs or native resource procedure.

Runtime collection logs use the actual completed attempt ID, but the logger is
nonblocking, quota/queue bounded and permitted to drop events. They cannot supply
a mandatory original witness or publication receipt. The proposed mechanism instead
captures original bytes with exact actor/ID provenance, then uses existing explicit
generation metadata only to confirm publication; management values remain outside
the numerical oracle. Unknown/expired-unconfirmed lineage denies qualification.

FreeBSD API references support proposed descriptor/credential primitives, while
the pinned Go credential wrapper's hidden PID member makes an actual native-header
and handoff gate necessary. General documentation/main-branch source is not support
evidence for either pinned release. That implementation must validate both actual
14.5 and 15.1 before activation. No guessed ABI, timestamp inference or UID-only
fallback is accepted by the proposed packet.

The coherent packet defines original custody, unchanged forwarding, independent
parsing, publication/retention, fixed limits, augmented observer/connector budgets,
actual-versus-front socket crash recovery and full teardown. It remains Review;
maintainer adoption precedes implementation, and integrated/native review precedes
activation and complete both-release resource qualification. All 153 historical
manifests and primary operator edits retain their original bytes. No runtime change,
new service/dependency/Actions, native execution or acceptance is claimed here.
