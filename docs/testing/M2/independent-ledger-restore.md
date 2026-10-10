# Independent ledger restore prerequisite

Status: Native fixture prerequisite accepted in PR 118; runtime denied. Date: 2026-10-10.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
[Accepted mechanism](../../milestones/M2/property-recovery-mechanism.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md),
[joint owner](joint-execution-leases.md), [system observer](execution-system-backing.md),
[journal persistence](execution-journal-persistence.md).

PR 117 accepted the publication snapshot correction at merged source
`9b380afcc858796c22d18ae3565c1bef630cdd41`. This change adds native test fixtures
for the independent restoration portion of M2-RECOVERY-001/M2-STATE-001 and
M2-RUNTIME-001. Production Go behavior, original writers and runtime denials remain
unchanged. No installation path, restore command, config key or service is added.

## Fixture procedures and meaningful boundaries

The UFS fixture uses one newly owned no-softupdates memory filesystem, with
separate root-private ledger/journal directories. Its file-level ledger export
contains the older ownership record alone. Restoration stages and syncs that
exact record, renames it and syncs the ledger directory; the original directory
and empty writer lock are retained. This supplies no whole-UFS rollback guarantee.

The ZFS fixture creates a new pool on a private 256 MiB fixture file, with separate
ledger and journal datasets and root-private mountpoints. A snapshot of each
is retained for the distinct tests. The ledger-only procedure rolls back exactly
the ledger snapshot, without a recursive parent rollback. Actual descriptor mount
identities and pool/dataset GUIDs are observed. The guarded persistent identity
producer and full system/pool exclusion remain separately qualified host components;
these fixture callbacks cannot establish commissioning or runtime eligibility.

Both procedures first initialize a private original journal receipt and publish
real ownership and property-job records through the existing native publishers.
Each ledger publication uses a metadata-pending journal acknowledgement, actual
complete after-census and journal confirmation. The native-pending record names
the actual fixture job/submission/filesystem and retains a digest of its reviewed
intent. Helper/boot/supervisor values are synthetic metadata: no executor launches,
and there are no admitted workers to drain in this fixture.

A disarmed successor preserves that native uncertainty. Wrappers call the real
journal file and directory sync operations and check that the newer complete ledger
bytes remain unchanged at both boundaries. Restoration begins only after the
original durable receipt acknowledges the barrier; ordinary rearming denies.
Existing journal persistence/cut gates independently cover lost acknowledgements
and failed sync. This fixture sequencing is not an assembled root restore workflow.

The exact test binary deletes and syncs the newer pending job, reports its boundary
and blocks. The parent sends SIGKILL and actually waits for that child. Fresh joint
observations at this interruption must retain the disarmed journal and native
pending record, expose a third census and remain incapable of publication. After
ledger-only restoration, observations must expose the exact older census/ownership
bytes and omitted job while preserving journal namespace, identities, metadata and
exact bytes. Comparison excludes only access time and mutable physical allocation
accounting under the accepted snapshot contract; all other stat fields remain
compared. Original domain/lock bindings must remain unchanged. No history repair,
job replay, reinitialization, hold release or reconstructed writer is permitted.

A deliberate second procedure restores the older journal too. Its matching old
hashes hide the pending job and restore barrier. The test requires observation-only
reopen and denies initialization/publication through that replacement handle; it
**does not claim to detect joint rollback**. No local checksum can supply that
missing provenance. An assembled startup owner must independently require the
accepted restore declaration/disarming and recovery checks before any new writer.
Missing journal metadata remains observation-only and a missing ledger directory
denies without creating it.

## Qualification and remaining work

Each filesystem requires 47 explicit controls plus a native backing/teardown marker.
The state runner retains all previous parents, native controls and codec children,
and additionally hashes the actual ZFS utilities. The evidence checker rejects
older logs, either missing filesystem, any omitted/duplicate/unknown/failed control,
wrong backing and incomplete teardown. The old-evidence regression failed before
this checker extension, then passed with the new required families.

The state suite ran serially on both pinned disposable FreeBSD/amd64 releases
with local review from clean signed source. Teardown removed the owned pool/datasets,
file backing or memory mount/device, child processes and all host guest artifacts.
Historical manifests retain their original bytes and actual sources. All checks
remain local with GitHub Actions disabled and no persistent services.

The procedures qualify only these newly owned fixture scopes. They do not register
an operator backup policy, establish fresh provisioning, implement root restore
admission/draining/reconciliation, detect root tampering or joint host rollback,
qualify power loss/VM memory resume, grant restart/global launch clearance or close
M2/release acceptance. These remaining gates must compose the original owner and
actual backend/supervisor before property writes can be enabled.


## Executed evidence

Signed source `df1536d8bc5a3250c81c068e37d4d3a3b912a6b0` passed both complete
state corpora on pinned FreeBSD/amd64 15.1-RELEASE-p3 and 14.5-RELEASE. Each
executed 79 selected parents, 1,170 native controls and 1,930 codec children,
including all 77/1,076 retained parents/controls and 47 new restore controls per
filesystem. The same verified executable SHA-256
`f1a465582a9a6018cd84ed19df36309212dc35a797e1b79ba97cba8b1fcb0575`
ran on both guests and was removed. All 22 backing/teardown markers were required.
The actual restore child was killed and reaped on each filesystem on each target.
This is warm process-death and explicit file/dataset restore evidence, not a
hardware power-loss or active-kernel-interruption claim. Tiny fixture timings do
not establish commercial-scale or physical NAS performance.

Local review passed `make check lint test secrets`, 163 tooling tests, eleven race
packages and all 53 bounded fuzz targets. Shared checker/runner maintenance
selected a fresh full portable plan. All 91 historical M2 manifests and all 414
original Go inputs retained their exact bytes; only a new Go test file was added.
GitHub Actions remain disabled, no host service was enabled and every owned
session stopped with verified teardown. The final prose/evidence update reruns
baseline checks and reuses exact passing deep inputs.

The first signed source also passed the complete 15.1 native suite. Its portable
review caught two tooling omissions: the source-build selector and the runner's
synthetic evidence lacked the new restore families. Both were corrected before
the final same-source two-target qualification. An extra unpinned host-Python
diagnostic lacked the pinned PyYAML dependency and failed nine foundation checks;
it was not a clean-source qualification. The pinned local review passed those
checks. Original sources, failed/passing records, log hashes, executable/compiler,
images, native utilities and teardown receipts remain in the
[sanitized manifest](independent-ledger-restore-manifest.json). No earlier attempt
is rewritten or relabeled as the final qualification.
