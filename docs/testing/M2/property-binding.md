# M2 original root property request binding

Date: 2026-10-08. Status: implementation qualification pending.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md) and [Test Design](../../milestones/M2/TDD.md).
Partial M2-PRIV-001/M2-PLAN-001/M2-NATIVE-001 evidence.

## Original request provenance

The private `propertyRequestBinding` is issued only by receiving a typed request
on an original admitted root-side peer. Seal it into that peer's original private
owner. Retain the exact request and kernel/correlation identity, original descriptor,
admission context and deadline. A copied/decoded/reconstructed receipt or foreign
owner cannot create lineage, project metadata or retire the original. Snapshot
values remain copies; changing them cannot change the held request.

Repeated checks require the original live admission, root reply phase, exact typed
request and original actor, actual real/effective root and actual peer UID. Check
quiet socket state to reject disconnects or unexpected bytes before another gate
consumes the receipt. Check the caller context before/after and the unchanged
original admission lifetime. Never renew a phase window. Checks are point-in-time
provenance, not an atomic guarantee of a future operation; future consumers must
repeat them at their own phase boundaries. No credential/UID/close hook is added.

Serialize short checks within the original owner, without consuming protocol bytes
or performing blocking network reads. Refuse a busy exchange rather than wait for
its I/O mutex. A failed valid check retires the binding and both original socket
and admission lease. Close is idempotent and retains its first actual cleanup result;
unissued bindings cannot close another owner. Cancellation/expiry, admission
shutdown and terminal reply also prevent renewed use. Socket closure releases
network admission only; the existing separate worker ledger still waits for the
actual worker exit and says nothing about a native executor or kernel quiescence.

The private default-denial listener consumes this binding and checks it before its
fixed refusal. Both review/submission remain denied. No backend, runtime route,
service consumer, root policy, independent human/session authenticator, durable
job, recovery/global launch clearance or storage effect is enabled. The asserted
application context remains untrusted correlation at root; it does not reconstruct
application SessionAuthority. Future native backend clearance must compose this
live peer prerequisite with its separate policy/plan/recovery/launch gates.

## Required evidence

Signed denying stubs must fail the original-retirement portable test and the new
native binding parent, with every old transport/application case retained. Portable
unissued/copy/decoded/foreign-owner cases preserve the unrelated original lease.
The private portable destructor fixture proves concrete socket/admission retirement
only; its fields manufacture no native/kernel authority.

Add sixteen exact native cases: original review/submission, isolated snapshots,
copied/reconstructed/foreign-owner bindings, synchronized disconnect/extra frame,
original deadline, admission shutdown, explicit close, terminal reply, 32 concurrent
live checks, nil/cancelled caller contexts and full fixture teardown. Child
credentials drop before exec. Use actual admitted root/kernel peers, strict codecs,
real descriptors and a fixed refusal; request context is a synthetic service
assertion, without a human login claim. Earlier real application/client tests
remain required separately. Native close syscall errors cannot safely be forced
without violating concrete descriptor ownership; error retention is inspected
without injecting a fake close or credential implementation.

Require all eight parents/131 exact cases on both pinned disposable FreeBSD
releases, under unchanged Go 80-second/outer 90-second bounds. Shared selector/
checker edits require fresh full portable review. Retain actual signed red/green
sources, executable hashes, historical manifests and complete guest/artifact/
supervisor cleanup with preserved base/runtime bytes. Actions remain disabled.
M1 Safari/VoiceOver remains pending by maintainer choice.
