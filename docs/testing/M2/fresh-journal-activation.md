# Fresh journal activation qualification

Status: private prerequisite qualified; awaiting maintainer review; runtime denied.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted recovery mechanism](../../milestones/M2/property-recovery-mechanism.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md),
[exclusive fresh domains](fresh-execution-domains.md),
[original journal persistence](execution-journal-persistence.md).

## Contract

The private FreeBSD component consumes the original exclusive fresh-domain
receipt exactly once. Both protected namespaces must still be empty, unbound and
unchanged under the original journal-before-ledger leases. Preexisting names,
partial creation, a previously bound pair, copied receipts and reopened journal
bytes cannot establish this receipt. Invalid interfaces or an already cancelled
context have no effect; uncertainty after inspection retires both stores and
preserves visible history.

The private root composition fixes `crypto/rand.Reader`. Two distinct sixteen-byte
identities become the installation binding and original helper incarnation. The
exact sixteen-byte boot transfer is compared before publication and after its
acknowledgement; it is not parsed as a UUID. Complete native identity, namespace,
census and system-use transfers are repeated under both original leases. The
confirmed census must equal an independently constructed empty namespace census.
ZFS domains need distinct native dataset identities and cannot be ancestors of
each other. UFS still requires the qualified file-level ledger restore procedure;
this component does not accept full-filesystem restoration or prove an operator's
future restore workflow.

Only this original one-shot receipt enables the already-locked journal initializer.
The accepted version-one active record has generation one and no pending state
or restore barrier. Actual staged-file sync, rename, directory sync and exact
post-sync object/content confirmation establish the original acknowledgement.
Further native/system/boot observations and receipt checks precede return. Only
the original post-publication directory snapshot can replace the earlier empty
snapshot; later metadata touches cannot be absorbed as an expected publication.

The private activation retains its original producers and permits repeated
observation while preserving both leases. Copies and serialized bytes cannot
observe, close or reconstruct the original owner. Drift or collector failure
retires it without repairing files. Close releases lifetimes and leaves an active,
unclean journal; it does not implement orderly shutdown or publish a closed marker.
No operator API, configuration or installation path is introduced.

## Executed evidence

Signed implementation source `5d375aae4b85e65fcf42fc2ed7fdf47352f9f493`, based on
PR 121's merged source `e97479338572b8fd3f4348f8721ac1e84be4050a`, passed the
complete affected storage corpus on pinned FreeBSD 15.1-RELEASE-p3 and
14.5-RELEASE/amd64. Each ran 90 selected parents, 1,806 native controls, 1,945
codec children and 31 actual backing/teardown markers. The new families require
93 UFS controls, 97 ZFS controls and 19 interruption controls. The same
checksummed executable ran on both targets; see the
[source-bound manifest](fresh-journal-activation-manifest.json).

Controls include fresh distinct identities, repeated facts, exact initial bytes
and original acknowledgement, empty census, copies/reopen, invalid interfaces,
entropy/boot/collector failure, prebinding and late ledger entries; every file
publication failure boundary, post-ack cancellation and same-object metadata drift;
and actual ZFS ancestor relationships. Three exact-binary children per release
were SIGKILLed and waited after actual file sync, directory sync and returned
acknowledgement. Partial/staged/active history remained intact, without another
fresh receipt or reconstructed initializer. These are warm process cuts, not
hardware power-loss or active storage-command/kernel interruption qualification.

The fixtures use actual UFS/ZFS objects, locks, native identities, reads, syscalls
and durability. Their physical-use and boot transfers are synthetic. The fixed
kernel/native backing/system producers have separate accepted host gates; their
composition with this receipt remains unqualified. QEMU TCG on Darwin/arm64 uses
two guest CPUs and 6,144 MiB, with UFS softupdates disabled and a disposable
256 MiB file-backed ZFS fixture. This is not a hardware, throughput or commercial
scale qualification.

The denying activation stub at signed source
`e9849045508b7fc617dcc95456e2321797aa616c` failed exactly the three new native
parents; all 87 previous parents passed. Its fixtures and guest were removed.
The previous-evidence regression failed before the checker required activation
controls. Both red results retain distinct source/log hashes. The updated evidence
consumer tests passed all 62 cases and reject missing, failed, duplicate or
invented controls, incomplete parents and incorrect backing/teardown.

A fresh local review passed `make check lint test secrets`, all 170 tooling tests,
eleven race packages and 54 bounded fuzz targets. The final prose commit reruns
baseline checks and may reuse deep results only for the exact unchanged inputs,
base and environment. All 419 unaffected original Go inputs and 99 historical
manifests retained their bytes. All owned sessions and native fixtures stopped;
compiled test artifacts were removed. GitHub Actions and host services remain
disabled.

## Remaining acceptance

This component is unregistered and grants no launch authority. A fixed OS-root
kernel/native backing/system composition, registered startup/global owner,
orderly shutdown, actual backend/supervisor assembly, explicit recovery/restore
workflow and assembled property acceptance remain required. Runtime writes remain
denied; the property workflow and NAS release are not complete.
