# Local qualification

Status: development tooling; actual executions and limitations must be recorded
separately. Local results do not replace previously accepted manifests or qualify
new native behavior merely because a VM boots.

## Mac first

The maintainer selected the current Mac as the first lab host. Run repository
checks and deep Go checks there before requesting review:

```sh
make tools
make check lint test secrets
make go-check
```

Attach the checked commit and results to the PR. Race detection and bounded fuzzing
remain review expectations; automatic hosted runs now perform formatting, vet
and unit/component tests for changes beyond allowlisted prose. `make go-check`
still runs the complete portable Go gate.

Automated Firefox currently has Linux-only artifact pins; its existing Linux
workflow remains a manually selected fallback. Safari automation can use the
existing installed-driver procedure; manual accessibility acceptance remains
pending. This change does not invent macOS Firefox qualification.

## Disposable FreeBSD guests

Prerequisites: Python 3.14, Git, OpenSSH, QEMU with x86 UEFI firmware and `unzstd`.
The current Mac already has QEMU and zstd; this runner installs no host packages,
services, GitHub runner or SSH configuration. On a future Linux lab, provision
QEMU/OVMF/zstd first. No nested virtualization is required by the current TCG
runner.

Use a clean committed checkout. The runner transfers `git archive HEAD`, never
the host's ignored files, `.tools`, Git credentials or private configuration.

```sh
# Describe the selected guest without downloading or starting it.
python3.14 tools/local-qualification.py properties 15.1

# Explicitly start one disposable guest and run only the selected suite.
python3.14 tools/local-qualification.py properties 15.1 --run
python3.14 tools/local-qualification.py properties 14.5 --run
```

Supported suites: `foundation`, `research`, `discovery`, `properties`,
`continuity`, `performance`. Run the two release candidates **sequentially** on
the Mac. Each guest has two CPUs, 6,144 MiB RAM, qemu64 and snapshot storage.
Reserve additional RAM for macOS; stop other large workloads while testing.
These are lab allocations, not Horvium runtime hardware requirements.

The runner uses TCG on both hosts. On an ARM Mac, amd64 executes through software
emulation and can be substantially slower. Performance procedures retain their
original assertions: slow results are failures, never adjusted to pass. Record
TCG/host/QEMU/firmware independently; results cannot be compared directly with
hosted/KVM measurements or physical performance. KVM acceleration and macOS ARM
FreeBSD qualification require a separately reviewed extension.

The virtual-disk fixture in PR 28 is not supported by this runner yet. Its Linux
process-identity and device attachment wrapper needs a separate Mac adaptation
and native execution before that gate can pass. Do not count property-suite
success as device/provider safety proof.

## Isolation, cache and evidence

Only a randomly allocated **127.0.0.1 SSH port** is forwarded; there is no VNC,
public listener, host NFS export or host disk attachment. OpenSSH uses `-F
/dev/null`, a checksum-verified public builder fixture key, disabled SSH agent,
and a private per-run known-hosts file. Fixture credentials never authorize the
guest to log into the host. The key is deleted after execution and is never
cached or committed. Guest package/SDK downloads still require internet access;
local execution means no GitHub-hosted runner minutes, not an air-gapped test.

Compressed images and the pinned upstream AnyVM artifact are cached by SHA-256
under ignored `.tools/lab-cache`. Every cache hit is authenticated; corrupt
entries fail closed. Fresh image copies are decompressed per run, with snapshot
mode and before/after image verification. Cached images are never writable
guest disks. The local runner drives QEMU directly; it does not execute AnyVM or
allow it to change the host's SSH configuration.

Private logs and `result.json` are retained under ignored `.tools/lab-runs`.
The record names the exact source commit, suite/release, image/runtime hashes,
QEMU/firmware, host architecture, emulation mode, assertions and cleanup result.
Qualification requires a successful guest command, suite-specific assertion
validation, unchanged base/runtime and verified owned-process termination.
Missing/skipped/failed assertions and cleanup/preservation failures remain
unqualified. Normal teardown removes the fresh image, private key and variable
store; logs and the known-hosts file remain for inspection. Failed preservation
may leave a quarantined image for diagnosis.

Use Ctrl-C to interrupt: the owning process terminates/waits for its child and
records failure. SIGKILL or host power loss prevents this cleanup; discard that
run and terminate its specific QEMU child before starting another. No broad
process-name kill or production storage cleanup is part of the procedure.

Review/sanitize selected evidence before putting it in a PR; never commit raw
guest/host inventories, key material or credentials. A local record is input to
the milestone evidence review, not automatic milestone acceptance. Preserve
both release candidates and requirement/test traceability.

## Optional later home lab

A dedicated x86-64 Linux host/VM can run the same sequential fixtures. Suggested
starting allocation: four vCPUs, 16 GB RAM, 40 GB scratch space and internet
access. These are unmeasured provisioning suggestions. A later KVM runner needs
virtualization or nested virtualization, controlled `/dev/kvm` access and its own
recorded execution evidence. This host stays separate from live NAS storage.

An isolated self-hosted GitHub runner is optional later work. This PR does not
register one, introduce persistent execution of PR code on the Mac, or change
branch protection. Local suite selection provides savings without that service.
