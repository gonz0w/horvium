# Original joint execution-domain leases

Status: Joint observation prerequisite qualified; awaiting maintainer review; runtime denied.
Date: 2026-10-09. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted mechanism](../../milestones/M2/property-recovery-mechanism.md),
[original domain observer](original-backing.md),
[journal persistence](execution-journal-persistence.md).

PR 114 accepted original-lease backing discovery at merged source
`29a9dddc799cb49ee73ee39625bab667e2f6287c`. This owner acquires a pre-existing
journal lease before a pre-existing ledger lease and observes both while holding
their original mutexes. It closes a partially opened pair on any failure and
never creates missing locks, initializes a journal, repairs namespaces or replaces
objects. Candidate paths are trusted internal OS-root inputs; no installation
path, configuration schema, service, HTTP operation or CLI flag is selected.

This is joint observation ownership, not a registered runtime writer. It has no
publisher or launch method, journal durability receipt, startup eligibility,
restore decision or global permit. Complete system/owned-pool exclusion,
independent restore scope and explicit commissioning/startup/global composition
remain required. Different candidate pairs cannot acquire write authority through
this observer. Ordinary advisory locks remain subject to the accepted trusted
OS-root boundary and are not a ZFS fence.

## Original lifetime and complete observations

The owner holds its own mutex, then the journal mutex, then the ledger mutex.
Both protected-object observers surround the entire operation, retaining original
directory/lock metadata, ACLs, no-follow lock names and live mount views. The
individual observer now has a locked implementation; its existing wrapper retains
its 45-second context and safety behavior. The private host bridge chooses only
the previously qualified native producer, with no caller-supplied native facts or
commands. Internal storage callbacks remain trusted metadata transfer contracts.

A complete journal snapshot precedes two journal-then-ledger native identity
passes. Both passes must match, and actual complete ledger censuses before and
after the second pass must have identical exact-byte digests. A final complete
journal snapshot must preserve presence, decoded record, exact bytes and full
actual file metadata. Role-framed physical identities produce the accepted binding
representation; identical ZFS datasets and same-backing object aliases deny.
Distinct UFS directories can be represented without proving restore independence.

The installation field is existing journal metadata, never evidence of fresh
randomness or commissioning. Its format is validated before probes. After a
successful observation, the original owner retains the native object/backing and
installation binding and denies a later change. Observations retain a private
original-owner pointer; copied, inert and closed owners deny. Closing excludes
all observation, releases ledger before journal and preserves persistent names.
A copied Close cannot close the original. Reopening creates a different origin,
never a reconstructed acknowledgement or live execution identity.

A present journal must be valid and match the actual physical binding. Active,
closed, pending, disarmed and mismatched-census history remains observable; none
is interpreted as recovery or startup permission. The returned opaque metadata
contains the actual census and journal independently. No pending state, barrier,
hold or retained record is cleared. The private journal store's initialization
capability is explicitly retired when transferred to this observation owner;
not even empty domains can incidentally initialize it. Existing standalone journal
persistence behavior is unchanged.

Context cancellation before observation starts denies without borrowing. Once
actual inspection begins, a probe, object, namespace, census, identity or context
failure permanently retires that original pair for further observations. Restoring
an unsafe object cannot reset its latch. Close still releases resources. The total
pair context is 90 seconds; every fixed native collection retains its own 45-second,
64 KiB and command deadline limits. Mutex waits and kernel filesystem calls gain
no new preemption guarantee. Root edits, joint rollback, memory resume and async
kernel work remain outside automatic clearance.

## Failing-first tests and required gates

The correspondence stub failed the absent-journal positive assertion before
implementation. Portable cases cover seven valid history observations and nine
invalid/foreign bindings or partial records, plus off-target denial without a
callback. The committed native bridge stub failed both joint pair-open assertions;
retained original-lease assertions still passed. Failed attempts retain their
original sources and teardown evidence.

The focused `joint-leases` suite includes both previous original-backing parents
and all 66 retained assertions. Two new parents require 40 UFS and 41 ZFS joint
assertions: actual guarded identities, journal-first probe/acquisition ordering,
both held locks, partial-open cleanup, no missing-lock creation, copied/inert/closed
owner denial, cross-domain lock drift, exact ledger-byte drift, journal namespace
drift, reader failure/cancellation retirement, real credential drop, Close
exclusion and same-ZFS-dataset denial. All four owned filesystem/device/directory
fixtures must tear down. Run 15.1 and 14.5 serially with the existing QEMU runner.

The affected `state` suite retains all previous cases and adds one owned UFS
journal-pair parent with 21 assertions. Its explicitly supplied backing metadata
exercises only the storage transfer contract; the separate host suite proves
actual guarded native identities. Actual protected objects, ACLs, matching journal
history, full ledger bytes, inode replacement, retirement and absence of any
original publication receipt are checked here. Both targets must pass all 77
parents, 1,075 required native assertions and 1,930 codec children. Historical
manifests and unrelated original sources remain unchanged. Shared runner/checker
maintenance requires one fresh full portable review; later evidence-only changes
reuse its exact deep inputs while rerunning all baseline checks. GitHub Actions
remain disabled and no host service is installed.

## Qualified observations

The [manifest](joint-execution-leases-manifest.json) binds the final implementation
and qualification inputs to signed source
`036847d0ee204eb73af7073559a0a5bf3e9b3a76`. The focused native suite passed all
four parents, 147 identity/ownership assertions and twelve owned fixture teardown
controls on each pinned FreeBSD/amd64 release. The 15.1 joint run executed signed
source `b2d7fc3911f88d61c453b37f1f96e89c3d087a5f`; the subsequent source changed
only the host runner's synthetic unit-test fixture. Native Go sources, tests,
guest scripts and validators stayed identical. The 14.5 run produced the same
native executable SHA-256
`97c9989bbacc149e756011352edec88eeced93741b4d50293dd7350e29a7ba8a`.
The manifest retains each actual source rather than relabelling the earlier run.

Both expanded state suites executed the final implementation source and passed
all 77 parents, 1,075 required native assertions and 1,930 codec children. Their
native executable SHA-256 matched
`e23da35c6f842f91bffdef953a3421c6649f8724a4ee90a3788e5cf691e32cd0`.
The new journal parent's 21 assertions use explicitly supplied backing metadata
for the storage contract. Actual native identity discovery is qualified separately
by the focused host suite; supplied facts cannot grant host clearance.

Full portable review passed check, lint, test and secrets, 158 tooling tests,
eleven race packages and all 53 bounded fuzz targets. The initial native bridge
stub's failed positive assertions and a failed portable attempt remain unqualified
and source-bound. The portable failure exposed missing journal/backing markers in
the runner's synthetic state log; its fixture was corrected before final review.
No failed attempt is counted as a passing gate.

All 88 historical M2 manifests and 347 other original source inputs retained
their exact bytes. The sole intentional original Go source refactor is the locked
single-domain observer, retaining its public wrapper. Every owned session stopped;
runner teardown verified each guest fixture, process group, executable, disposable
image and key. GitHub Actions remained disabled. These small disposable fixtures
provide no production resource or performance qualification.

This completes joint observation ownership. Registered runtime writer ownership,
complete system/owned-pool exclusion, independent restoration, fresh commissioning,
acknowledged journal/ledger transitions and startup/global launch composition
remain required. No runtime consumer, service or storage effect is enabled.
M1 manual Safari/VoiceOver acceptance remains pending separately.
