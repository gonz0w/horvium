# CI/CD

## Foundation

The repository workflow runs full-tree content/config/link checks, ShellCheck,
actionlint, regression tests for the gates, and Gitleaks. PRs also validate title
format and author DCO trailers. Tools are checksum-pinned; external Actions use
immutable commit SHAs. Default workflow permissions are read-only and checkout
does not retain credentials. Untrusted fork code never runs in a privileged
`pull_request_target` job. Put untrusted metadata in environment variables.

A manually dispatched FreeBSD smoke workflow validates native shell and hooks
only. It is not ZFS, SMB, performance, or production qualification. M1 pins native
OS/toolchain/packages separately; do not assume cross-compilation proves execution.

## Product gates as components arrive

Add Go formatting/vet, unit/component tests, targeted fuzzing and vulnerability
checks; FreeBSD builds for validated architectures; native FreeBSD OS integration;
real protocol clients; restore/failure/upgrade tests; port/package validation;
and reproducible release evidence. Physical controllers/disks/NICs and power-loss
behavior belong in the lab. Keep heavy tests out of routine docs-only checks.

Do not make CI or a release depend on running tests against production storage.
Publish checksums, SBOM/provenance and compatibility evidence for deployable releases.
Enable remote required checks only after verifying their actual names/results.
Feature availability and enforcement are recorded in bootstrap status, not assumed.
