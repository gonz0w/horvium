# Dashboard qualification runner integration

Date: 2026-10-13. Status: implementation awaiting maintainer review.
Owner: [issue 380](https://github.com/gonz0w/horvium/issues/380), under
[Capacity 290](https://github.com/gonz0w/horvium/issues/290), with
[Overview 289](https://github.com/gonz0w/horvium/issues/289) and
[M2 131](https://github.com/gonz0w/horvium/issues/131) as broader gates.
Authority: DASH-REQ-006 / DASH-PERF-001 and the adopted
[resource Test Design](../../design/dashboards/resource-qualification.md).

This completes the related test-fixture and host-runner integration in one PR.
Portable fixtures, synthetic supervision and FreeBSD cross-compilation qualify
implementation review only. This candidate has not executed either native
resource suite; browser, resource, recovery, hardware and release acceptance remain
open. Runtime storage writes remain disabled.

## Original candidate, transfer and launch

The existing `local-qualification.py performance` entrypoint delegates to
[the integrated runner](../../../tools/dashboard-qualification.py). Dry runs
remain descriptions. Execution requires the existing explicit `--run`, an exact
clean signed/DCO source, `PR_TITLE`, resolved `BASE_REF` and a completed matching
schema-2 local review. Missing context fails before QEMU prerequisites, image
creation or storage execution. Generic transcript-only validation and the guest
source-build shell still deny; they cannot bypass the complete original host path.

The existing builder exports the signed Git tree, verifies pinned Go, builds the
three cgo-free FreeBSD/amd64/v1 artifacts and retains original supervision/logs.
Preflight rechecks source, archive-to-Git blobs, compiler, review and actual bytes.
Transfers use fixed names and remote commands. An owned worker reads/rechecks
original bytes and feeds its SSH child's stdin; supervisor stdin remains its lease.
It never transfers fixture credentials as source or interpolates operator input.

One snapshot guest retains the existing qemu64/two-CPU/6144-MiB/TCG procedure,
image/key/runtime pins and loopback SSH settings. It installs no guest packages.
Actual artifact/kernel/userland/ZFS/memory preflights bracket the resource command;
original supervisor completion, command digests and times bind their ordering.
Resources use the prepared management, projection helper and command fixture,
original two parent names and 30-minute timeout. The controller inspects its same
original artifact set at each parent's start/end. Every pair launch copy checks
original role bytes, destination and unchanged source before launching. This is
trusted fixture provenance, not attestation against malicious guest root, kernel
or local developers.

## Independent originals and guest cleanup

Each fixture saves an exclusive, bounded, private `dashboard-original-<mode>.json`
before timed requests, containing its original inventory count/digest/origin and
owned count. Original inode, mode and bytes are rechecked at parent completion.
These four files are independent of resource rows; host expectations never come
from the rows being validated.

Each resource parent acquires its own controller's FreeBSD process reaper using
the same amd64 layout as the qualified native supervisor. After subtests and all
their cleanup callbacks, it reaps departed adopted children and requires an
original own-reaper census with zero children and descendants. It does not call
a zombie removed, signal saved PIDs, kill an arbitrary observed process or retry
fixture deletion. Remaining live descendants fail the run; discard the guest.
The parent checks original pair socket and temporary-directory paths are absent:
three synthetic pairs including the raised-cap boundary, and two reader modes
including helper restart. Native-small requires exactly one successful original
storage-cleanup callback.

Only then does the parent save independent original controls, observed artifact
hashes/compiler and cleanup. Final reaper release failure still fails the parent
and whole command. Existing raw measurements, limits, crash controls, storage
ownership checks and legacy markers remain. Added observers run outside the pair.

The host retrieves six fixed original files through owned commands after complete
resource command success. It reopens these files and supervisors. Assembly checks
exact coverage and observed predicate/GUID/replacement/boundary/native/cleanup
counts. It removes only the two known partial parent stdout records, after exact
comparison with independent parent files. Raw stdout remains private and hashed.
Unknown records, failure diagnostics, FAIL/SKIP and missing completion deny.
Host provenance and full controls come from independently bound originals, then
the unchanged strict and legacy validators check the assembled transcript.

## Host ownership and finalization

The worker captures separate original host pins/context before QEMU launch.
After resources it waits original QEMU, observes loopback listener refusal,
rechecks base/runtime and image hashes, and removes key, image, variables and
built source/executables. Independent cleanup attempts prevent one failure from
stranding other resources; failed deletion is not retried or relabeled successful.
Logs and failed records remain private and unqualified.

The outer lease owns the whole guest. Its interrupt fallback also removes the
candidate whose directory name exactly matches that fresh performance lab;
no record-supplied deletion path is followed. Keys/images/variables/groups retain
existing fallback cleanup. After the outer group is verified stopped, the parent
reopens worker/build records, transfer/preflight/resource/readback supervisors,
before/after preflight files, original sidecars and raw/assembled transcripts.
It checks separate original host pins, remaining runtime hashes, removed files/
listener and signed clean source. Only complete strict and legacy validation sets
one release's `qualified` resource receipt. Broader acceptance remains separate.

## Qualification handoff after integration review

Do not run performance from this unreviewed integration branch. After maintainer
review/merge, prepare a clean signed candidate and review with actual source/base/
title. The base must be an ancestor with a nonempty signed contributor range.
`PR_TITLE` / `BASE_REF` are existing review inputs, not runtime configuration.
For that candidate, execute sequentially:

```sh
PR_TITLE='test(dashboards): qualify the assembled resource candidate' \
BASE_REF='<reviewed ancestor commit>' make review
PR_TITLE='test(dashboards): qualify the assembled resource candidate' \
BASE_REF='<same reviewed ancestor commit>' \
  .tools/venv/bin/python3 tools/local-qualification.py performance 15.1 --run
# Wait for verified completion and teardown before starting the other release.
PR_TITLE='test(dashboards): qualify the assembled resource candidate' \
BASE_REF='<same reviewed ancestor commit>' \
  .tools/venv/bin/python3 tools/local-qualification.py performance 14.5 --run
make lab-status
```

Record both actual lab paths. Call the integration module's `verify_pair` with
those two explicit owned directories; it reopens both originals, requires identical
candidates and applies the unchanged two-release contract: eight modes, 16,000
scheduled observations. Never select successful runs to replace a failed candidate.
Retain failures; fixes need a new reviewed source and new relevant execution.
Add actual source-bound receipts in the subsequent qualification PR. Browser/
accessibility, continuity, lifecycle #305/#139 and full NAS acceptance remain open.

Use `make lab-stop` / `make lab-status` to verify development teardown. No service,
Actions workflow, paid automation, external synchronization or new dependency is
introduced. Rollback restores performance main-entry denial and removes this
development integration; no operator data or production migration is involved.

## Portable verification

Behavioral tests cover original-file exclusivity/mutation/link/bounds, launch-copy
byte mismatch and original reaper-census classification. Synthetic host tests cover
complete assembly, missing/changed originals, incomplete controls/cleanup, changed
artifacts, failures/unknown records, fixed transfer, failed construction, final
supervisor/file/runtime mutations and cleanup refusal. Existing denial and real
owned-process regressions remain required. Reaper/syscall, ZFS cleanup and resource
budgets require actual both-release execution after review. Historical manifests
retain their original sources and limitations.
