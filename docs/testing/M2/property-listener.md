# M2 private root property listener

Date: 2026-10-08. Status: test-first; qualification pending.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved [EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[admitted exchange](property-admitted.md), [application client](property-client.md).

## Ownership and default denial

The private listener joins the original root-owned socket policy, bounded admission
leases and actual FreeBSD kernel-authenticated typed exchange. It is unregistered:
no CLI/HTTP route, runtime consumer, configuration flag, service or deployment is
added. Reuse existing trusted helper configuration and its 1–64 connection bound
and positive at-most-five-minute timeout; no hardware profile controls behavior.
Require actual real/effective root UID, native support, canonical socket selection
and the unchanged safe root directory/ancestor policy before listening.

One original server gets one serve attempt. Nil, decoded, reconstructed or copied
servers cannot start the listener or change an unrelated original worker ledger.
Refuse existing regular files, symlinks and sockets without changing them. Own only
the newly created socket, set root/configured group and exact 0660 permissions,
validate the actual socket, and remove only that same current socket on shutdown.
Preserve replacements. Report real cleanup errors instead of declaring teardown
success. This does not establish a storage singleton lock or native launch fence.

Admit each accepted socket before starting a goroutine or consuming protocol bytes.
Overflow retires it immediately; there is no pending queue. Keep a separate bounded
worker ledger keyed by the original lease. Socket expiry/retirement may release
transport capacity, but never releases worker capacity. Release the worker entry
only when that goroutine exits. Duplicate/copy/unissued leases cannot reserve or
finish an unrelated original entry. This ledger counts transport workers; it proves
no storage executor, kernel maintenance or durable job quiescence.

Each worker negotiates and parses one typed request through the unchanged actual
kernel UID/correlation/deadline owner. Both review and submission receive the fixed
`denied` refusal. Malformed/legacy/unauthorized streams close without a successful
reply. No callback can supply replacement UID, observations, policy or successful
storage admission. No collector, coordinator, job publisher or native control is
called. Root policy, original plans, authenticated application context, recovery,
executor absence/global launch and durable admission must be implemented and
qualified independently before replacing this default denial.

Cancellation closes the original listener/admission owner, interrupts negotiation
and request I/O, waits for every owned worker and joins descriptor/socket cleanup
errors. No new phase deadline is minted; the original admission context remains
bounded by the server parent. The current workers do only bounded socket I/O.
Future backend assembly must separately define durable job lifetime and execution
shutdown rather than infer either from a management disconnect.

## Verification

Start with signed denying stubs. Portable actual-socket tests require independent
worker/socket lifetime, original lease/owner identity, duplicate denial, concurrent
last-slot exclusivity, factory configuration bounds and unsupported-host refusal.
The private portable worker-accounting fixture supplies no root or kernel authority.

Add a separate opted-in native parent with eighteen cases: complete denied typed
review/submission, actual configured/wrong/root peers, malformed and legacy input,
idle expiry, immediate capacity recovery, shutdown during negotiation and request,
existing file/symlink/socket preservation, unsafe directory/ownership/alias refusal,
nonroot construction denial, replacement preservation, single original serve and
fixture removal. Children drop credentials before exec/connect. UID and overflow
denials must precede their original idle timeout; canonical wrong-UID hellos cannot
qualify by timing out. Assert drained workers and removed owned sockets. All data
is disposable; no operator storage or backend state is used.

Retain all 74 original transport cases and require all five parents and exact
once-only case/result inventories. Preserve Go 80-second/outer 90-second deadlines,
artifact checksums, base/runtime preservation and verified supervisor cleanup.
Shared checker/selector changes require fresh full portable review and serial
native runs on both pinned FreeBSD releases. Historical manifests keep their actual
sources. M1 Safari/VoiceOver remains pending; GitHub Actions remain disabled.
