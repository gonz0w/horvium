# M2 bounded property connection admission

Date: 2026-10-08. Status: test-first contract; portable qualification pending.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[owned exchange](property-exchange.md).

## Contract

A private, unregistered transport owner admits original accepted Unix connections
before an eventual listener starts a worker or reads negotiation. The trusted
connection limit is 1–64, matching the existing helper configuration bound; the
positive timeout is at most five minutes. These are internal inputs, not a new
configuration schema or client controls. Exhaustion denies immediately, closes the
incoming connection without bytes and creates no waiting worker or pending queue.

One original connection occupies one slot. An opaque private lease retains its
original connection and admission context. Zero/decoded/reconstructed/copied
leases cannot expose or retire the original. Duplicate admission of the same
original connection retires its earlier lease and denies the second attempt.
A closed or nil descriptor cannot consume capacity. Admission reads/writes no
protocol data and proves no UID, session, enrollment, durable job or launch authority.

The original deadline begins at admission and is bounded by an earlier parent
deadline. Set it on the owned socket and retain the original child context for
later kernel-authenticated negotiation and typed exchange. Traffic, accessor calls
and slot reuse cannot extend a lease. Expiry, parent cancellation, explicit lease
close or owner shutdown interrupt socket I/O and retire the lease. Release occurs
after closing the owned descriptor, before reporting lease retirement. Explicit
shutdown synchronously retires all original owned descriptors; repeated close is
idempotent, and a closed owner cannot restart. Close errors are reported, not
silently converted into teardown success.

Transport capacity is distinct from worker/native-command quiescence and durable
storage admission. Closing a socket cannot prove that a caller's goroutine or
kernel mutation stopped. Future assembly must retain safe root-owned listener/
directory policy, actual peer UID checks, original authenticated session checks,
bounded worker ownership, enrolled policy and independent recovery/global launch
clearance. The owner starts no listener, worker, service, storage command or daemon;
it is deliberately disconnected from the runtime.

## Tests and limits

Failing-first portable behavioral tests use actual disposable Unix sockets:
configuration bounds, original descriptor/context and untouched input, no authority
reconstruction, immediate overflow/EOF, unrelated socket preservation, slot reuse,
closed/nil/duplicate descriptor denial, concurrent last-slot admission and concurrent
shutdown, parent cancellation, blocked reads, fixed/earlier-parent deadlines and
complete retirement. No operator disks, host inventory or backup history are used.

Run the baseline and affected-package race/fuzz gate, then compile the helper tests
for FreeBSD without execution. This ordinary Go transport/accounting slice changes
no kernel peer primitive, typed codec/exchange or native fixture/runner/checker.
Actual FreeBSD admission/assembly behavior remains unqualified; the existing native
peer/exchange evidence retains its original source and is not relabeled. Future
assembled native safety/recovery/continuity/resources and M2 acceptance remain open.
M1 Safari/VoiceOver stays pending; GitHub Actions remain disabled.
