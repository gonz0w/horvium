# Fixed native journal activation qualification

Status: private fixed native prerequisite qualified; awaiting maintainer review; runtime denied.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted recovery mechanism](../../milestones/M2/property-recovery-mechanism.md),
[original activation](fresh-journal-activation.md),
[execution system backing](execution-system-backing.md).

## Contract

The unregistered root host boundary fixes the exact bounded kernel boot identity,
original descriptor backing producer and complete guarded system-use collector.
It consumes the existing exclusive fresh-domain receipt once. The internal
storage bridge returns an opaque original lifetime offering observation and
release only. Copies, nil/decoded values, closed handles and uncertain observations
cannot reconstruct it. Observation excludes release; failed observation retires
the lifetime and releases no partial metadata. Release remains possible after
retirement and is idempotent, including its original error.

Already cancelled or nil contexts and nonroot host callers cannot create domains.
The original component keeps its independent namespace, dual locks, durable
acknowledgement, native identity/census and repeated producer checks. Successful
release leaves an active unclean journal, without clean-shutdown, startup/global
registration or native launch authority. No operator route, schema, installation
path or service is added.

## Test design

Traceability: M2-STATE-001, M2-RECOVERY-001 and the accepted journal/bootstrap
gates under the M2 Test Design (T2/T3 on disposable E1/E2 fixtures).

Portable behavioral tests cover lifetime origin, serialization, repeated
observation, invalid contexts, retirement, exact release and concurrent exclusion.
Unsupported hosts expose no activation and call no producers. Disposable UFS and
ZFS host fixtures add 37 required controls per filesystem: exact actual kernel boot
bytes, original repeated native/system observations, fresh active record, both
held locks, copied/serialized refusal, context refusal without creation and actual
credential-drop denial despite inherited root descriptors. Actual producer faults
before publication and after acknowledgement preserve the permitted visible
history, release both locks and allow metadata-only observation. Changed ledger
metadata retires the original lifetime; releasing it cannot recommission the
existing namespaces or publish clean shutdown. Every child is waited and all
owned metadata filesystems/devices and data fixtures must be restored/removed.
The evidence checker rejects earlier system evidence without these controls.

The complete affected state corpus remains required on both pinned targets.
These are warm disposable VM observations, not power-loss, appliance, throughput
or commercial-scale support claims. Fixed native composition is not registered
startup/global ownership; runtime writes remain disabled.

## Executed evidence

The [source-bound manifest](native-journal-activation-manifest.json) records the
signed implementation, merged PR 122 base, compiler/image/utility pins, exact
binary and log hashes, required control inventories and verified teardown.
Both pinned FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE/amd64 targets passed the
six-parent `execution-system` suite with 274 controls each: 147 retained original/
joint controls, 53 retained system controls and 74 actual activation controls.
The same checksummed host executable ran on both releases. Both targets also
passed the complete affected state corpus: 90 selected parents, 1,806 native
controls, 1,945 codec children and 31 actual backing/teardown markers each.
The corresponding state executable was identical across both releases.

The actual fixtures read `kern.boot_id` through the already qualified fixed
sixteen-byte kernel boundary, retain descriptor-correlated UFS/ZFS backing and
collect the complete guarded native mount/swap/GEOM/imported-pool view. The active
journal boot bytes match that actual read. Fault transfers wrap real producers;
they are fixture-only errors and never alternate production input selectors.
Three failure modes cover boot error before publication, boot error after actual
acknowledgement and system error after acknowledgement. Post-ack active bytes
survive release, metadata-only reopen and rejected retry. The pre-publication
case leaves no active record and cannot adopt the existing fresh names on retry.
A ledger namespace change retires the original lifetime without journal repair;
removing the changed entry cannot reset it. Actual nonroot children inherited
opened root directories, were denied before creation, and were waited.

The signed denial source failed precisely both new activation positives; all
four retained original/joint native parents passed. Its guest deferred cleanup
reported no failure, and host fixtures/processes were removed. Complete positive
parent teardown markers are claimed only for qualifying runs. Three portable
receipt parents failed against their stubs; all four receipt/unsupported-host
parents passed after implementation. The prior-evidence regression failed before
the checker required the new controls; its 28 focused consumer tests passed after
that change. Each missing, failed or duplicate control remains a refusal.

Local review passed `make check lint test secrets`, all 172 tooling tests,
eleven race packages and 54 bounded fuzz targets. The shared evidence-checker
change selects one fresh complete portable review. Final prose/evidence reruns
baseline checks and reuses deep results only for the exact unchanged inputs,
base and environment. All 422 unaffected original Go inputs and all 100
historical manifests retain their bytes. Concurrent dashboard planning is
excluded from this branch. All owned children, guests, fixture pools/datasets,
mounts and device backings were removed or restored; local compiled artifacts
were removed. GitHub Actions and host services remain disabled.

## Remaining acceptance

This qualifies fixed native initial activation as an unregistered prerequisite.
Registered startup/global ownership, orderly shutdown, actual supervisor/backend
assembly, explicit root recovery/restore workflow, independent workers and the
shared HTML/CLI property workflow remain required. Active kernel interruption,
assembled property acceptance, other storage operations and NAS release gates
remain open. Equal native views are not atomic snapshots or a kernel fence;
90-second activation/observation deadlines may conservatively refuse large or
busy systems. These small QEMU TCG fixtures do not qualify throughput, hardware
power loss, full-filesystem UFS restoration or commercial scale. Runtime writes
remain denied and the NAS release is not complete.
