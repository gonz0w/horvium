# Independent ledger restore prerequisite

Status: Native qualification pending; runtime denied. Date: 2026-10-10.
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

Run the existing state suite serially on both pinned disposable FreeBSD/amd64
releases and local review from clean signed source. Remove the owned pool/datasets,
file backing or memory mount/device, child processes and all host guest artifacts.
Historical manifests retain their original bytes and actual sources. All checks
remain local with GitHub Actions disabled and no persistent services.

The procedures qualify only these newly owned fixture scopes. They do not register
an operator backup policy, establish fresh provisioning, implement root restore
admission/draining/reconciliation, detect root tampering or joint host rollback,
qualify power loss/VM memory resume, grant restart/global launch clearance or close
M2/release acceptance. These remaining gates must compose the original owner and
actual backend/supervisor before property writes can be enabled.
