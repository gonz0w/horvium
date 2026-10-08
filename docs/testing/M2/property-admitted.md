# M2 admitted property exchange

Date: 2026-10-08. Status: partial native transport qualification; maintainer review pending.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[connection admission](property-connections.md), [typed exchange](property-exchange.md).

## Contract

Compose one original private admission lease with one original root-side typed
exchange. Claim negotiation once under its original admission owner's mutex;
concurrent or repeated negotiation retires that lease and interrupts I/O. Zero,
decoded, copied or reconstructed leases/owners cannot manufacture the composed
binding or retire an unrelated original. Failed valid attempts close/release the
original lease; no factory fallback, duplicate handshake or publication occurs.

Call the existing actual FreeBSD kernel peer factory with both the original
connection and admission context. Its timeout can never extend that context's
original deadline, including earlier parent deadlines and time spent before
negotiation. Retain original descriptor/context/deadline equality and ownership
before exposing parsed request assertions. Typed methods still revalidate actual
kernel UID, original correlation, phase and bounded frames. No identity hook,
caller-claimed UID, new framing or native command is introduced.

Successful terminal review/admission/refusal replies, invalid requests, failed
writes/negotiation and explicit close retire both the typed exchange and admission
lease. Close the descriptor before returning its slot. Each composed terminal path
releases capacity without relying on its caller to notice typed-peer closure. Parent
shutdown, expiry and overlapping negotiation interrupt both negotiation and request
reads; this remains socket I/O containment, never proof that a worker or kernel
mutation stopped. Explicit close reports both ownership errors, and factory failure
joins final cleanup errors rather than discarding them.

No listener, worker, HTTP/CLI route, application-session adapter, collector, root
policy/store dispatcher or storage write is registered. Kernel/correlation snapshots
and synthetic display replies do not grant authenticated human/session authority,
durable admission, recovery/executor absence/global launch clearance or safe retry.
Future assembly must independently supply those gates and safe root-owned listener/
directory ownership, then qualify complete native safety/recovery/continuity/resources.

## Tests and limits

Portable cases deny unissued/copy/retired owners, preserve unrelated original
leases and release slots on invalid UID/unsupported-host factory failure. A separate
opted-in FreeBSD parent checks real root/configured/wrong UID peers, review/submission/
refusal metadata, terminal/malformed slot reuse, overflow before bytes, negotiation/
phase/earlier-parent deadlines, shutdown in both I/O phases, overlapping negotiation,
duplicate lease retirement and fixture teardown. Child credentials apply before
exec/connect. Replies use the existing synthetic views; no operator storage is used.

The focused peer suite selects all three separate native parents. Retain the original
18 peer and 26 typed exchange cases, existing 80-second Go/90-second outer deadlines,
artifact hashes and full fixture/base/runtime/supervisor cleanup. The strict checker
requires every parent/case/result once; historical uncomposed proofs retain their
actual sources and cannot qualify this extension. Shared selector/checker edits
require fresh full portable review and serial runs on both pinned FreeBSD releases.
Signed source `24fd2777481139386f8e824579f22e9579755a8e` passed all 60 cases
(18 original peer, 26 original typed exchange and 16 admitted exchange) on both
disposable FreeBSD 15.1 and 14.5. Four portable behavioral parents passed, as did
fresh baseline checks/133 tooling tests, all 11 race packages and 49 bounded fuzz
targets. Deep portable review took 214.903 seconds. The native denying-stub run
remains unqualified; earlier successful results retain their actual earlier source.
See the [source-bound manifest](property-admitted-manifest.json).

All 647 owned sessions stopped with verified teardown; native executable/key
artifacts were removed and both guests' base/runtime inputs preserved. All 74
historical M2 manifests and 378 protected existing internal/tooling inputs retain
their bytes. M1 Safari/VoiceOver remains pending and Actions stay disabled;
full M2/native write acceptance remains open.

The interruption fixture must fail if negotiation returns any owner or request
metadata, rather than converting unexpected success into a synthetic error. Normal
close/retirement paths are exercised on real sockets. A socket-close syscall error
cannot safely be forced through this concrete owned connection without bypassing
its descriptor ownership; factory error joining is reviewed directly without
introducing an injected close/credential hook.
