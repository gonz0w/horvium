# Original idle activation retirement

Status: test-first private prerequisite; qualification pending; runtime denied.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted recovery mechanism](../../milestones/M2/property-recovery-mechanism.md),
[fixed native activation](native-journal-activation.md).

## Contract and test design

M2-STATE-001/M2-RECOVERY-001 and the accepted bootstrap/restart gate require an
original durable close transition. This component closes only the initial unused
activation, with its exact generation-one active receipt and independently
confirmed empty census. It has never admitted or launched work. It cannot close
an advanced, pending, disarmed, reconstructed or externally changed incarnation;
worker draining and general orderly runtime shutdown still need their own owner.

The opaque internal FinishIdle operation excludes observation and release. Nil
or cancelled contexts before entry have no effect. Once entered, it permanently
retires observation/finish authority regardless of its result. Both original
leases remain held until explicit idempotent release. Copies and decoded values
cannot finish the original. Unsupported hosts still cannot obtain an activation.

Under the original fresh/pair/journal/ledger mutexes, repeat exact receipt,
namespace, native backing, complete system and raw boot checks. Publish only the
single generation-two closed successor through the accepted durable pipeline;
confirm its exact receipt, complete census and unchanged native/system/boot facts.
Capture its own post-publication directory before subsequent producers, so later
touches cannot be accepted as part of that publication. Every entered outcome
retires both stores; uncertainty preserves visible active/staged/closed history.
Release alone retains its previous behavior and does not publish closed state.

T2/T3 disposable UFS/ZFS tests require context/copy/serialization rejection,
one-time exact closed history, both held locks, metadata-only reopen, no fresh
recommissioning, pending metadata/native and restore-barrier refusal, collector
failures, same-object late touches, cancellation and every publication failure
boundary. Exact-binary children stop at actual file sync, directory sync and
returned acknowledgement; another drops credentials while retaining original
root descriptors. All are killed/waited; every visible history and lease release
must be preserved. Fixed native host fixtures independently exercise FinishIdle
with actual boot/backing/system producers on both pinned releases. Previous
activation-only evidence must be rejected by both consumers. The complete
affected state corpus and local review remain required, with verified teardown.

No startup/global registration, worker drain, native launch permission, recovery
workflow, operator API/configuration or service is introduced. Closed bytes alone
do not grant restart eligibility. Warm fixture observations cannot qualify
hardware power loss, active kernel work, throughput or commercial scale.
