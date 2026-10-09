# Native execution census prerequisite

Status: Native prerequisite accepted in PR 110; runtime denied.
Date: 2026-10-08. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted mechanism](../../milestones/M2/property-recovery-mechanism.md),
[record/census core](execution-journal.md).

PR 110 accepted this prerequisite at merged source
`3d9c5c49d8717ff4f8fd19c1a68a048d5244f415`.
PR 109 accepted the portable journal core at merged source
`2fc51de735e2df4657b8a61df8ebb171c9857311`. This private FreeBSD consumer connects
that framing to complete descriptor-anchored ledger observations. It performs
no publication and grants no commissioning, restart, launch or recovery authority.
The independent journal owner and real publisher/backend ordering remain required
before runtime use; this prerequisite does not satisfy their assembled gates.

## Native observation

Only the original open root lease can inspect. Nil, inert, copied and closed
handles return zero catalog and no digest; copied-handle refusal does not retire
the original. Kernel effective UID is checked even for inherited descriptors with
an actual ACL read grant. No new exported method, command, path or configuration
is introduced. The caller holds the lease mutex for the entire observation.

The existing mixed inspector supplies the complete validated name set before
streaming sorted record bytes. The consumer uses the already accepted bounded
record decoders and framing, including the original validated empty writer lock.
Records are hashed as exact stored bytes; harmless whitespace changes the digest.
Only bounded summaries, hashes and object metadata remain, rather than complete
histories or a retained copy of every record. Existing 64 KiB record, 64 MiB total,
ownership and terminal/nonterminal limits remain. Existing consumers retain the
same read/namespace checks through a wrapper with no initialization callback.

Return the digest only after the reader verifies every record's final metadata,
the held directory/lock objects and the final namespace metadata. Unknown or
orphan names, unsafe ACLs/objects, invalid records, duplicate submissions, capacity
exhaustion and ordinary drift yield neither a partial catalog nor a digest.
Successful observation never resets publication uncertainty or releases holds.
It can describe readable uncertain metadata, which remains blocked for publication.

The binding parameter is **supplied metadata**, not a native identity producer.
These tests deliberately use synthetic installation/object/backing digests. They
prove framing and native file validation, not independent journal provenance,
system-backing identity, restore scope, anti-rollback attestation or executor
quiescence. A consistent omission between observations changes the digest; only
comparison with an independently durable journal can interpret that difference.

## Tests and qualification

Failing-first tests require exact framing for empty and published mixed ledgers,
actual ownership replacement, a journal record consuming the native digest,
exact presentation, omission, repeated observation, unchanged inodes and copied
handle isolation. Refusal controls cover bounds, bindings, duplicate submissions,
unsafe objects, corrupt/oversized records, orphan preservation, late ownership/job/
namespace drift and late writer replacement. The existing ACL parent adds complete
census refusal/restoration on both POSIX and NFSv4 models, including the actual
credential-dropped inherited-descriptor process.

The native selector/checker requires all 38 new census assertions, all fifteen
new ACL assertions, the executed parent and verified backing teardown alongside
the entire affected state publication/capacity/ACL/crash corpus. Checker regressions
refuse missing, duplicate, failed or old evidence. No historical manifest is
rewritten. Run the existing state suite serially on pinned disposable 15.1 and
14.5 guests, then verify all owned sessions stopped. Shared checker maintenance
requires one fresh full portable review; subsequent prose evidence can reuse its
exact deep inputs while rerunning baseline checks.

Native journal file/directory durability, dual-lease provisioning, persistent
binding production, startup/global ownership, restore/reconciliation and active
kernel-cut qualification remain open. Runtime writes remain disabled.

## Executed evidence

Signed native source `261fdb533723055d9d1a7142c272f46c9fb4ef19` passed all 68
selected native/codec parents, 858 required native marker assertions and 1,876
codec children on each pinned disposable FreeBSD/amd64 target. The same prebuilt
executable SHA-256
`d0cad5af498d2e89b2adf05f510aa13802c99d4d4448a2e78b984f8271099c64`
ran on both. Native census parent times were 0.19 s on the 15.1 ZFS fixture and
0.29 s on the 14.5 UFS fixture with softupdates disabled. These are small fixture
execution times, not commercial-scale resource or throughput qualification.

The signed denial stub at `cf0db4b4026bbd6cd568e5342583a33d19b14cc9` failed the new
census and ACL controls while 66 other parents passed. It remains unqualified,
with verified teardown. An initial portable review then found an older runner
unittest fixture missing the new mandatory result marker. Its baseline failure
remains unqualified; only that test fixture was corrected. Qualified native code,
checker, scripts and runtime inputs remain unchanged and retain their actual source.

Signed portable source `6662618c9206d8c9850e974af7789d9130374092` passed all baseline
checks, 146 tooling tests and one fresh full review of eleven race packages/all
51 bounded fuzz targets. All 83 historical M2 manifests and 141 protected existing
native host/helper/command inputs retain their actual bytes. All 721 owned sessions
were stopped with verified teardown at evidence capture; Actions remain disabled.
The final prose/evidence review reruns baseline checks and reuses these exact deep
inputs. See the [source-bound manifest](native-execution-census-manifest.json).
No persistent binding, journal durability, operator provisioning, startup/global
launch, recovery or M2/release acceptance follows from this qualification.
