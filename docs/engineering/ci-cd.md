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
and bounded pool/GEOM parser fuzzing through `make go-check`. Go/C research programs
stay outside the product package build; product code uses the standard library
only. Module checksums appear when an external module is actually introduced.

The BSD workflow has separate actual-product discovery jobs on 15.1/14.5 amd64.
They install Python 3.14 in the disposable guest solely to verify/install the Go
SDK and run [the native script](../../tools/bsd-native-discovery.sh). Python is not
a NAS runtime dependency. Native fixture tests require explicit opt-in/root and
otherwise report skipped; workflow opt-in ensures these cases actually execute.
No product service or network listener is enabled by these jobs.

Continue adding dependency vulnerability
checks; FreeBSD builds for validated architectures; native FreeBSD OS integration;
real protocol clients; restore/failure/upgrade tests; port/package validation;
and reproducible release evidence. Physical controllers/disks/NICs and power-loss
behavior belong in the lab. Keep heavy tests out of routine docs-only checks.

Do not make CI or a release depend on running tests against production storage.
Publish checksums, SBOM/provenance and compatibility evidence for deployable releases.
Enable remote required checks only after verifying their actual names/results.
Feature availability and enforcement are recorded in bootstrap status, not assumed.
