# Original journaled metadata publication

Status: private prerequisite under qualification; runtime denied. Date: 2026-10-10.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
[Accepted M2 design](../../milestones/M2/EDD.md),
[recovery mechanism](../../milestones/M2/property-recovery-mechanism.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md),
[journal persistence](execution-journal-persistence.md),
[joint leases](joint-execution-leases.md),
[independent restore fixtures](independent-ledger-restore.md).

## Contract and scope

This private FreeBSD component connects an original acknowledged journal store to
its original ledger lease. Typed ownership/property-job successors validate the
current complete ledger, exact predecessor bytes, revision/history and catalog
capacity. A streaming projection computes the exact before/after census with one
bounded successor body; it does not retain the complete ledger's record bodies.
Format and generation capacity for both journal transitions is checked before
publication. Physical filesystem space remains uncertain and failures retire the
owner conservatively.

The owner retains both original mutexes, journal first, through three publications:
metadata-pending intent, ledger successor, and journal confirmation of the complete
actual after-census. Each existing publisher syncs its file, validates the opened
staged object, renames and syncs the directory. Repeated original native identity,
whole namespace and exact original journal receipt checks surround those phases.
A successful return follows the confirmed durable journal acknowledgement and a
final complete observation. A pending native launch record is preserved byte for
byte as metadata; terminal job histories retain their existing recovery holds.

An error after the original receipt is inspected retires the joint owner and both
stores. Actual visible old/new files and uncertain pending history remain intact;
there is no rollback, retry, repair, receipt reconstruction or hold release. Nil,
foreign, copied, closed and unprivileged handles deny. Reopened public handles
remain observation-only, even when their journal and ledger hashes agree.

The fixture alone initializes a new original journal and transfers its volatile
receipt into a private original pair. Production has no commissioning path or
receipt transfer. This component does not open an HTTP/CLI route, register a root
writer, launch native storage effects, create an installation, or establish startup,
restore or global admission. Those future consumers must compose complete native
system/owned-pool exclusion, accepted restore provenance, boot/owner recovery and
supervisor/backend clearance under the registered original owner. Trusted fixture
backing callbacks cannot stand in for those system checks.

## Behavioral and native gates

Four successor-census scenarios cover empty creation, sorted insertion, replacement
and exact whitespace bytes. Eleven denial scenarios cover identity, partial/wrong
order input, invalid namespace/binding/body/limits and independent input ownership.
The bounded fuzz target compares accepted results against independent full framing.

Each real UFS (softupdates disabled) and ZFS fixture requires 101 controls: typed
create/replace, actual sync order, lifetime locks, exact acknowledged census,
native pending/terminal hold preservation, reopened denial, 21 failures at the
three stages, retired retries preserving all visible bytes, rollback, same-byte
journal inode replacement, cancellation after pending acknowledgement, disarming,
generation exhaustion and an orphan namespace entry. Each failure point replaces
only the declared stage operation; the after-sync drift cases call the real sync
before changing owned fixture bytes. ZFS uses a newly owned 256 MiB file-backed
pool and separate journal/ledger datasets; UFS uses disposable memory backing.

A separate UFS parent requires 32 controls. The exact binary pauses after each
real directory sync; the parent sends SIGKILL, actually waits, then observes the
pending/ledger/confirmation histories through newly opened handles. The fourth
child actually drops its kernel UID and must deny without a ledger effect. These
are warm process-death boundaries, not hardware power-loss, memory-resume or
active-kernel-effect qualification. Both complete existing state corpora remain
required, with 84 parents, 1,404 native controls, 1,945 codec children and 25 actual
backing/teardown markers per target. The checker rejects previous evidence and
any missing, duplicated, unknown or failed control, parent, child or backing.

Run from clean signed source using the local qualification runner, sequentially:

```sh
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
PR_TITLE='feat: add original journaled metadata publication' make review
make lab-status
```

All fixtures are new owned disposable data. Cleanup reaps children, destroys the
owned pool/datasets, unmounts/removes memory backing and discards every guest and
host artifact. No operator disks, persistent services or GitHub Actions are used.
Shared evidence tooling requires a fresh complete portable race/fuzz plan. Native
and portable source-bound results will be recorded before review; historical
manifests retain their original bytes and scopes. Hardware/scale, commissioning,
startup/global/restore workflow and full M2/release acceptance remain pending.
