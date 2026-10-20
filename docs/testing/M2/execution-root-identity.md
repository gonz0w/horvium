# Original execution-owner root identity

The [admitted storage-control contract](../../milestones/M2/storage-control-contract.md)
requires real and effective UID 0 at original admission and retained transitions.
The existing fixed fresh commissioning, closed-startup review, closed-startup
activation and fresh-journal activation paths checked only effective UID. A
setuid-root identity with a non-root real UID could therefore pass that check.

All four existing paths now use one private actual-process check that reads both
UIDs on every invocation. The three commissioned-owner consumers retain that
check as their original root closure, so construction, Observe, FinishIdle and
Close recheck the pair at their existing boundaries. No new public factory,
capability, identity override, command, configuration or privilege is introduced.

The single-use process registry, original pointer lineage and mutex order are
unchanged. Root loss retires observation/finish; later restoration cannot revive
them or reset registration. Construction failure still attempts original release.
Close always attempts release once, including after root loss, and preserves its
own original close error. A caller must also retain any earlier Observe/Finish
failure; a successful release alone is no durable closed acknowledgement.

## Portable evidence and limits

`TestExecutionRootIdentityAdmission` exercises real/effective pairs 0/0,
65534/0, 0/65534, 65534/65534 and invalid negative values through the existing
original-owner constructor. Wrong pairs must call no factory, consume no registry
and produce no owner. The valid pair preserves the existing lifetime.

`TestExecutionRootIdentityLossRetiresOriginal` changes only the supplied real UID
while effective UID remains 0: during construction, before/during observation,
before/during finish, and before/during release. It checks refusal/retirement,
no revived callbacks after identity restoration, one release and retained
registration. The existing commissioning lifecycle/exclusion/copy tests remain.
These synthetic identities and callbacks prove source behavior only; they do not
produce native storage authority or alter the developer host's identities.

Failing-first execution retained the former effective-only predicate behind the
new common check. It failed the non-root-real/root-effective construction and all
seven real-UID-loss cases. The corrected predicate passes the focused race suite.
This is a reproduced legacy semantic control, not an executed unmodified native
baseline. Exact commands, hashes and full required portable review belong in the
candidate receipt/PR. FreeBSD compilation, when recorded, is not native execution.

Refs [#139](https://github.com/gonz0w/horvium/issues/139) and
[#305](https://github.com/gonz0w/horvium/issues/305); closes neither. Historical
native receipts retain their original source. Actual UID-drop/inherited-descriptor
and both-release lifecycle/fault proof remain required. The subsequent
[storage-control source consumer](storage-control.md) assembles the retained 0600
selection, original observation-only inspector and lifecycle/output. Its native
qualification remains open. No storage-control command is registered, runtime write denial
changes, native campaign, service installation or deployment follows from this
source correction. Draft #415 and retained failed/unknown artifacts are unchanged.
