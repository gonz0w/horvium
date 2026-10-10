# Exclusive fresh execution domains

Status: private prerequisite qualified; awaiting maintainer review; runtime denied. Date: 2026-10-10.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
[Accepted recovery mechanism](../../milestones/M2/property-recovery-mechanism.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md),
[joint leases](joint-execution-leases.md),
[original journaled publication](journaled-publication.md).

## Contract

Fresh commissioning cannot infer provenance from empty existing directories.
This private FreeBSD component exclusively creates the journal and ledger leaf
directories below two already protected existing local UFS/ZFS parents. It opens
both parents and proves both names absent before its first effect. Absolute bounded
paths, protected root-owned ancestors, actual ACLs, writable local backing and
softupdates refusal are required. Equal or nested domains deny. It creates journal
before ledger; each new 0700 directory receives a new exclusive empty 0600 lock,
an original nonblocking lifetime lease, file sync, directory sync and parent sync.
There is no existing-lock fallback, chmod repair, truncation, unlink or replacement.

The private original receipt retains both parent descriptors and exact original
directory/lock metadata. Repeated inspection validates the original leases,
parent-path correspondence, named directory entries, actual modes/ACLs/mounts and
both complete untouched namespaces. Copied receipts cannot inspect or close the
original; any drift/uncertainty retires it. Closure releases ledger then journal
and the original parents; it preserves every file and is idempotent. Any creation
failure releases all acquired descriptors, preserves partial objects and forbids
resuming that partial installation. Root recovery must preserve and review it;
this component offers no cleanup/reset command.

This is namespace provenance within the trusted OS-root boundary, not an active
installation. No journal record is initialized and initialization eligibility stays
disabled. There is no runtime consumer, public route, installation path/configuration,
new dependency or enabled host service. The future qualified root commissioning
consumer still needs fresh random installation/helper identities, repeated actual
boot/native/system/owned-pool checks, independent restore scope, durable active
journal acknowledgement and one registered startup/global owner. Existing empty
or reopened handles, decoded bytes and this receipt alone cannot launch work or
use the private metadata writer. Root tampering and joint rollback remain outside
local freshness proof. No physical hardware, power-loss or scale claim follows.

## Behavioral and native gates

Two parents require 91 controls each on actual disposable UFS without softupdates
and ZFS with separate owned datasets. Cases require exclusive creation, exact
journal-first order, original descriptors/private objects/locks, repeated inspection,
actual persistent-binding observation, denied writer/initializer, copied/closed
receipts, invalid/context/symlink paths, existing empty domains and missing halves,
all sixteen creation-stage failures and fifteen preserved retries, in-creation
directory replacement and lock collision, same-inode rewrite, lock/directory
replacement, late entries, unsafe parents and observation-only public reopen.
The fixture backing callback supplies native identity observations only; these
cases do not qualify complete system/restore/boot commissioning.

A third UFS parent requires eleven controls. The exact binary pauses after each
real domain parent sync. The parent sends SIGKILL and actually waits; a new
provisioning attempt must deny and preserve partial bytes/objects. A separate
child drops its actual kernel UID and must create neither domain. These are warm
process-death boundaries, not interrupted kernel effects or hardware power loss.
All existing state safety cases remain required: 87 parents, 1,597 native controls,
1,945 codec children and 28 actual backing/teardown markers. The checker rejects
previous evidence and missing/duplicate/unknown/failed cases and wrong backing.

Run only clean signed source, guests sequentially:

```sh
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
PR_TITLE='feat: add exclusive fresh execution domains' make review
make lab-status
```

Existing state openers, codecs, journal publishers, historical manifests and
runtime consumers retain their accepted inputs/behavior. The native evidence is
bound to the current accepted x/sys 0.49.0 dependency. Shared verification tooling
requires a fresh complete portable race/fuzz plan. All guests, fixture children,
pools/datasets, memory backing, mounts and compiled artifacts must be removed.
No operator data or persistent development service is used. Executed results are recorded in the
[source-bound manifest](fresh-execution-domains-manifest.json).

## Executed evidence

Clean signed source `42dad713aae291503c7cc9b2ba1951ad3a6ebe65`, based on
merged main `5527bb596b1c6d345a8adb3af7f4b073072445cc`, passed both complete
state corpora on FreeBSD/amd64 15.1-RELEASE-p3 and 14.5-RELEASE. Each ran all
87 parents, 1,597 native controls, 1,945 codec children and 28 required backing
markers. The same verified executable SHA-256
`330a64c663ace04f69367b771ccf3390cef9d4a8fc6fdb7860e4c0de876cf9d4`
ran on both guests and was removed. Each killed and waited for two paused
original children and waited for the actual UID-drop child. Every previous
state safety case remained required and passed.

Local review passed `make check lint test secrets`, 168 tooling tests, eleven
race packages and all 54 bounded fuzz targets. Shared native evidence tooling
selected a fresh full portable plan. The 60 focused verifier/runner consumer
tests passed. All 419 original Go inputs and 98 historical manifests (93 in M2)
retained their exact bytes. All owned sessions stopped with verified teardown;
GitHub Actions remain disabled and the concurrent dashboard planning remains
untouched in the shared checkout.

The signed failing-first source `56af5f2aac5ae780bdc90e66ef082b1e2aab78ea`
used a denying provisioning stub. Exactly the three new native parents failed;
all 84 previous parents passed. Its guest and fixtures were removed. The
previous-evidence verifier regression also failed before the checker was
extended. Those failures retain distinct source/log hashes in the manifest.
The final prose commit reruns clean baseline checks; unchanged deep inputs may
reuse the executed source-bound evidence only in the exact same environment.

This qualifies exclusive namespace provenance. Full commissioning, active
journal publication, registered startup/global ownership, native backend and
supervisor composition, recovery/restore workflow and runtime writes remain
required; the property workflow and NAS release are not complete.
