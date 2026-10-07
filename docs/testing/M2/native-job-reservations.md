# M2 native job reservation inspection

Date: 2026-10-06. Status: implementing; native qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Native contract

`StateDirectory.InspectJobReservations()` returns an immutable complete job view
from the held root-owned state lease. It uses the existing mixed census, including
all ownership records, and never takes a request path or caller-supplied record set.
Require effective kernel UID 0, held directory/lock identity, independently opened
enumeration, canonical names, private safe opened files/ACLs, strict complete
records and filename identity. Validate every record's metadata again and then
the directory/lock/namespace before any result escapes. Return zero on every error.
An independent descriptor keeps repeated scans independent of prior enumeration
offsets. Advisory locking and ordinary metadata checks do not fence OS root or
turn this into a cross-file atomic snapshot.

Both accepted job formats retain their existing strict codecs; property jobs keep
the entire review/policy/session binding. A sealed streaming builder retains
validated decoded metadata instead of a second entire encoded record set. Failed
additions leave its prior accumulation unchanged, and finishing transfers ownership
before sealing the builder. The inspector exposes no partial builder when a late
ownership, record or namespace check fails. Cross-format job/submission/plan
collisions are denied. Legacy jobs retain pool holds but cannot prove a complete
property retry binding.

The native view includes all ownership and job presentation bytes against the
existing 64 MiB mixed-store bound. `CheckPropertySubmission` therefore counts
ownership bytes before considering space for a new candidate. Shared publication
still independently reserves old-plus-new transient bytes and rechecks the complete
namespace; successful inspection/checking is neither that transaction nor durable
admission. Retain existing 64 nonterminal/1,000 terminal jobs, 64 KiB records and
100,000 ownership-record ceilings. Reduced fixture bounds do not expose a runtime
tuning API or qualify populated production workloads.

Whole-pool holds and retry meaning follow the accepted [pure model](job-reservations.md).
Terminal labels retain holds; matched retries return existing complete/current
metadata. Inspection and checks acknowledge no admission, claim no plan, acquire
or release no reservation, launch nothing and restore no session. Reads remain
available after publication uncertainty and never clear the write latch. Unknown/
orphan/unsafe state is preserved rather than repaired or deleted. Platforms other
than FreeBSD return `ErrUnavailable` without filesystem work.

## Qualification procedure

Use the existing disposable root UFS/ZFS and owned POSIX/NFSv4 UFS fixtures on
FreeBSD 15.1/14.5 amd64. Failing-first portable API/builder and evidence-checker
tests precede implementation. Require all 622 state assertions, 58 parent tests
and 1,837 distinct codec/model subtests without skips/failures on both releases.
New coverage includes complete mixed holders/baselines, exact duplicates/current
history, unrelated pool checks without admission, ownership/presentation bytes,
actual default class ceilings/exhaustion, reduced-byte bounds, cross-format
collisions, malformed/unsafe objects, unknown/orphan preservation, final native
record/namespace drift, renamed anchoring, lock/directory/closed/inert denial,
publication uncertainty preservation, full teardown and copy isolation.

ACL controls must deny actual named permissions/default inheritance on the held
directory, writer lock, ownership, legacy and property records, preserve ACLs and
restore valid inspection. An actual UID-65534 child with inherited descriptors and
effective kernel NFSv4 access must still be denied by the new root API. Keep the
existing publication/ENOSPC/read-only gates. Exact shell selectors and evidence
validation require every new family/assertion/codec child; old 555-case logs cannot
qualify this extension.

```sh
make test-focus PKG=./internal/storage TEST='^TestReservation'
# Signed committed source; actual proposed PR title:
PR_TITLE='feat(storage): inspect trusted FreeBSD job reservations' make review
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
make lab-status
```

Shared evidence/selector changes conservatively select one complete portable
review. Both native runs use the exact same clean signed source; do not edit that
source during qualification. Verify guest fixtures, overlays, keys, test artifacts
and process groups are removed; base images/runtime caches remain inert and
unchanged. Every prior manifest retains its actual historical source/bytes.

Native qualification is pending. Complete fresh storage eligibility/root-policy/
caller authorization, live-session/clock checks, atomic durable admission, old
executor/boot attribution, recovery/release, acknowledgement cuts and M2 acceptance
remain unqualified. No runtime/helper/HTTP/CLI/startup consumer or storage mutation,
metadata format migration, service installation or deployment is added.
