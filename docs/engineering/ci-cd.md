# CI/CD

## Foundation

The repository workflow runs full-tree content/config/link checks, ShellCheck,
actionlint, regression tests for the gates, and Gitleaks. PRs also validate title
format and author DCO trailers. Tools are checksum-pinned; external Actions use
immutable commit SHAs. Default workflow permissions are read-only and checkout
does not retain credentials. Untrusted fork code never runs in a privileged
`pull_request_target` job. Put untrusted metadata in environment variables.

The manually dispatched [BSD workflow](../../.github/workflows/bsd-smoke.yml)
keeps foundation shell/hook smoke separate from the M1 QEMU research jobs. The
research jobs use fresh file-backed storage and explicit disposable-VM opt-in on
FreeBSD 15.1/14.5 amd64. They validate narrow GEOM/ZFS/credential contracts, not
product integration, SMB, performance or production support. See the
[spike evidence](../research/M1/native-spike.md). The probes compile with the
checksum-verified native Go toolchain/base C compiler. Repository checks now
scan Go/C research source as UTF-8 text as well; its regression fails before that
coverage change. M1 pins native versions separately; cross-compilation alone is
not execution evidence.

## Product gates as components arrive

`make tools` now also installs the checksum-pinned Go SDK from
[artifact pins](../../tools/go-pins.json); automatic toolchain download is disabled.
The repository job runs Go formatting/vet, unit/component tests, race detection,
and bounded pool/GEOM/parser and helper framing fuzzing through `make go-check`. Go/C research programs
stay outside the product package build; product code uses the standard library plus the approved focused FreeBSD x/sys
adapter. Module checksums are recorded in go.sum and verified natively.

The BSD workflow has separate actual-product discovery/helper/collector jobs on 15.1/14.5 amd64.
They install Python 3.14 in the disposable guest solely to verify/install the Go
SDK and run [the native script](../../tools/bsd-native-discovery.sh). Python is not
a NAS runtime dependency. Native fixture tests require explicit opt-in/root and
otherwise report skipped; workflow opt-in ensures these cases actually execute.
No installed product service is enabled by these jobs. Helper tests explicitly
create only disposable fixture Unix sockets. The collector integration reexecutes
an unprivileged child against the real helper and verifies cached outage/recovery.
Its synthetic boundary/query tests also run natively. Full process/file-service
restart and performance qualification remain pending.

Continue adding dependency vulnerability
checks; FreeBSD builds for validated architectures; native FreeBSD OS integration;
real protocol clients; restore/failure/upgrade tests; port/package validation;
and reproducible release evidence. Physical controllers/disks/NICs and power-loss
behavior belong in the lab. Keep heavy tests out of routine docs-only checks.

Do not make CI or a release depend on running tests against production storage.
Publish checksums, SBOM/provenance and compatibility evidence for deployable releases.
Enable remote required checks only after verifying their actual names/results.
Feature availability and enforcement are recorded in bootstrap status, not assumed.

## Measured management budgets

The opt-in [performance workflow](../../.github/workflows/bsd-performance.yml)
stages two checksum-pinned builder 2.2.8 images and AnyVM 0.7.2 through
[the fixture tool](../../tools/prepare-performance-vm.py) and
[artifact pins](../../tools/performance-vm-pins.json). It explicitly selects the
QEMU qemu64 CPU model, two cores, 6144 MiB and snapshot mode. After detached VM boot, a separate SSH command executes the guest suite without
copying guest SDK/cache files back to checkout. Base-image and runtime
hashes are rechecked after execution. The Ubuntu 24.04 hosted runner and actual
QEMU version are recorded; hosted scheduling and firmware are not an immutable
physical benchmark. This replaces host-CPU defaults for the measured workload.

The guest first runs existing native fixtures, then the guarded
[performance script](../../tools/bsd-performance.sh). Sixty-second warm-ups,
five-minute small-fixture idle CPU, two 1,000-page ten-reader workloads per size,
generation replacement and separate record/byte exhaustion run only inside these
explicit disposable root fixtures. No VM, Python, QEMU or fake source becomes a
NAS runtime dependency. See [procedure and limits](../testing/M1/performance.md).

Before its own workflow is on main, dispatch the existing BSD workflow with
`performance=true`; it calls the same performance workflow from the selected
branch. Default native runs leave this long measurement disabled.

The manual [independent file-service workflow](../../.github/workflows/bsd-continuity.yml)
provisions fixture-only Samba and base NFS in disposable guests. Its explicit
native/protocol opt-ins and [procedure](../testing/M1/file-continuity.md) are required;
it installs/enables no operator host service and makes no client compatibility claim.

Until its standalone workflow is merged/registered on main, continuity is also
callable with `continuity=true` in the existing BSD native workflow against the
qualification branch. The optional flag provisions Samba only in those guests.


## Browser qualification

The [browser workflow](../../.github/workflows/browser-qualification.yml) runs
checksum-pinned Firefox/geckodriver on Linux against an explicitly synthetic
loopback Go console. Before registration on main, use `browser=true` with the BSD
workflow against the qualification branch. JavaScript enabled/disabled flows,
light/dark and reduced-motion media, keyboard focus, fragment/polling behavior,
escaping, labels, sampled contrast and narrow viewport reflow are asserted by the
stdlib WebDriver runner. The fixture exports only selected synthetic results.
Safari uses the installed macOS driver when remote automation is enabled by the
operator. Manual screen-reader/native-zoom checks remain separate; see the
[checklist](../testing/M1/browser-checklist.md). Browser test tooling is not a NAS
runtime dependency and passing headless checks is not accessibility certification.


## M2 property observation gate

The [property workflow](../../.github/workflows/bsd-properties.yml) uses the
existing checksum-verified VM image/AnyVM tooling with explicit qemu64, two vCPUs,
6,144 MiB and snapshot mode. Both FreeBSD release candidates run the isolated
[storage property script](../../tools/bsd-properties.sh), with both disposable
native/property opt-ins. Its evidence validator refuses skipped, failed,
duplicate or incomplete native assertions, including missing teardown. The
existing BSD workflow calls it from the selected branch until standalone manual
dispatch is registered on main. This qualifies property parsing only, not native
write eligibility, management transport, M2 storage changes or replication.
