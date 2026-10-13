# Disposable native storage cleanup observations

Date: 2026-10-12. Status: test-only implementation awaiting maintainer review.
Owner: [issue 376](https://github.com/gonz0w/horvium/issues/376), under
[Capacity gate 290](https://github.com/gonz0w/horvium/issues/290).
Authority: DASH-REQ-006 / DASH-PERF-001 and the approved
[resource Test Design](../../design/dashboards/resource-qualification.md).

## Ownership and actual absence

The existing [native storage fixture](../../../cmd/horvium/native_freebsd_test.go)
creates one fresh nonce-named pool on its newly allocated 512 MiB file-backed md
inside an explicitly disposable FreeBSD guest. Its cleanup previously interpreted
failed `zpool list` and `mdconfig` commands as absence, including permission,
transport or inspection failures. It also used the recorded pool/device names
without rechecking their identity immediately before destructive commands.

The [test-only coordinator](../../../cmd/horvium/dashboard_storage_cleanup_helpers_test.go)
requires the generated pool name, canonical md unit, absolute normalized backing
image path and original positive canonical uint64 pool GUID. Setup captures that
GUID immediately after pool creation and requires the later property query to
agree. Failed GUID capture leaves cleanup uncertain; it does not authorize pool
removal by name. An aborted creation with no GUID may clean its verified md only.

Cleanup re-reads the backing-file-to-unit association, compares the pool GUID,
destroys only that pool without force, then requires a successful complete
`zpool list -H -o name` which excludes its exact name. It rechecks the backing-file
association immediately before detaching the original unit without force, then
requires a successful complete `mdconfig -l` which excludes that unit. Unrelated
pool/device names are observed transiently and never printed, retained in a
receipt, destroyed or detached. Duplicate/malformed names, oversized outputs,
remaining owned storage or command failures return uncertainty. The coordinator
stops at the first uncertainty, returns no successful receipt, and requires the
disposable guest to be discarded; it does not try an alternate destructive command.

Command arguments use direct fixed-path argv with generated fixture identities.
The original cleanup callback has a 30-second total context, one-second pipe wait
delay, fixed locale/PATH, discarded stderr and a 64 KiB stdout writer limit. The
same limit and a 4,096-name bound apply to parsed inventories. This is disposable
fixture policy, not an operator storage API or authorization boundary. Concurrent
external root mutation and identity reuse between observation and operation are
outside this controlled guest procedure; the observations are not attestation.

The command shapes follow the [FreeBSD mdconfig manual](https://man.freebsd.org/cgi/man.cgi?query=mdconfig&sektion=8&format=html)
and [OpenZFS zpool-list manual](https://openzfs.github.io/openzfs-docs/man/master/8/zpool-list.8.html).
FreeBSD's [mdconfig implementation](https://github.com/freebsd/freebsd-src/blob/main/sbin/mdconfig/mdconfig.c)
shows why filtered `-f` listing can fail when no match exists, while an unfiltered
successful listing can prove unit absence. Its normal trailing-space list format
is accepted. These sources inform argv/parsing; actual native release behavior
still requires candidate-specific execution.

## Callback completion and observation boundaries

The fixture keeps the original dataset-property preservation callback. Go's LIFO
cleanup ordering runs that callback and later-registered process cleanup before
the fixture's storage removal callback; temporary directory removal follows it.
Only after successful verified pool/device removal, and when the test has not
failed, does the final storage callback emit `NATIVE_STORAGE_CLEANUP` with exactly
`pool_created`, `pool_removed` and `md_removed`, all true for a completed fixture.
No names, GUIDs, paths or inventory dump appear in that observation. A failed test
may still clean storage but emits no success observation. Aborted pool creation
never emits a full storage success record.

This record is deliberately separate from `DASHBOARD_CONTROLS`: it does not prove
that process descendants or sockets were removed, that temporary directory cleanup
succeeded, or that host QEMU/listener/key/image/variables teardown completed. A
future reviewed assembly must independently bind those facts and complete-suite
pass markers. The strict resource validator and performance execution guards stay
unchanged. Whole-guest cleanup must never be inferred from this storage record.

## Review and remaining acceptance

Five portable behavioral parents cover successful created/aborted sequences,
each failed command, changed ownership, remaining resources, duplicate/malformed/
truncated/oversized output, invalid generated identities and actual bounded writer
overflow. Command outputs are synthetic; the tests execute no storage commands.
The original missing-coordinator red log is retained privately. An initial test
fixture mistakenly supplied a GUID for an aborted creation and was corrected;
that failed portable run is not a native observation.

Required signed clean review runs `make check lint test secrets` and affected
command-package race/fuzz checks. FreeBSD/amd64 cross-compilation checks fixture
wiring only; native executables are not run and are removed afterward. Historical
receipts remain byte-identical. Native behavior on both pinned releases, original
fixture/control binding, complete process/socket/host teardown, transfer/launch
and resource execution remain open. Overview/Capacity, M2 lifecycle #305/#139,
manual accessibility and release gates remain open; runtime writes stay disabled.
Rollback restores this fixture and removes the two test-only coordinator/test
files, without changing product behavior or historical evidence.

## Recorded source candidate

[Receipt](native-storage-cleanup-manifest.json): signed/DCO source
`a8fc1058d502d41689c2992e1a2a007921f965f8`, base `358dd76b9621bb242527ede296f51f3ec3708dd5`, title
`test(dashboards): verify disposable storage cleanup`. Required checks passed with 294 tooling
tests, ordinary Go tests and five cleanup behavioral parents. Fresh proportional
deep review ran the command-package race check; that package has no listed fuzz
targets. The FreeBSD/amd64/v1, cgo-free cross-build passed, was not executed and
was removed. All 146 historical manifests remain byte-identical; owned process
teardown was verified. No actual native cleanup or broader acceptance is claimed.

A signed documentation-only followup runs the current-head baseline review and
may reuse only this completed exact-input deep evidence.
