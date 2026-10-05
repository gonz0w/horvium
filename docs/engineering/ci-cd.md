# CI/CD

## Local review

All repository checks run locally. GitHub Actions are disabled and runnable
workflow files are removed. Use the [local review gate](local-review.md) for
signed-commit/DCO/title checks and complete portable validation with source-bound
reports and verified process teardown. [Cost controls](ci-costs.md) explain the
change and the live protection record is in [bootstrap status](bootstrap-status.md).
The local content checker rejects reintroduced Actions workflows, required hosted
status checks and loss of signed-history/PR protections.

`make tools` installs checksum-pinned ShellCheck, Gitleaks, Python development
requirements and Go from [artifact pins](../../tools/go-pins.json). Automatic Go
toolchain download is disabled. Product code uses the standard library and the
approved focused FreeBSD x/sys adapter. Go/C research remains outside the product
package build. Python, QEMU and browser tools are development dependencies only.

## Native qualification

Use [the local runner](../testing/local-qualification.md) to explicitly select one
suite and one release in a fresh snapshot guest. Run 15.1 and 14.5 sequentially.
Only documented executed combinations count as evidence. Source is transferred
with `git archive HEAD`; no ignored host files or credentials enter the guest.
There is no automatic guest startup, production storage fixture or installed
operator host service.

| Suite | Guest procedure | Scope |
|---|---|---|
| foundation | [shell/hooks](../../tools/bsd-foundation-smoke.sh) | Repository shell and DCO smoke |
| research | [native spike](../../tools/research/m1-native.sh) | Narrow GEOM/ZFS/credential contracts |
| discovery | [native product tests](../../tools/bsd-native-discovery.sh) | Reader/helper/cache/admin/foreground runtime fixtures |
| devices | [owned virtual disks](../../tools/bsd-devices.sh) | Eight native identity/geometry observations plus host preservation/teardown |
| properties | [property tests](../../tools/bsd-properties.sh) | Complete 51-assertion observation suite and teardown |
| continuity | [independent services](../../tools/bsd-continuity.sh) | Fixture-only Samba/NFS through management outages |
| performance | [measurement](../../tools/bsd-performance.sh), after discovery | Separate combined-process/resource budgets |

The runner authenticates image/runtime artifacts before and after execution and
validates actual suite assertions; boot, skipped tests and missing teardown never
qualify. Properties require identity/type/GUID binding, mount/readonly/encryption
constraints, pool import/policy observations, installed feature profile and paired
vdev topology/rename/reuse checks. See the [M2 property evidence](../testing/M2/properties.md)
and linked slice documents. These observations do not authorize storage writes,
qualify provider safety, or establish full M2 completion.

Performance keeps explicit qemu64, two CPUs and 6144 MiB, warm-ups, idle sampling,
ten-reader page workloads, generation replacement and cap/recovery assertions.
Mac amd64 TCG emulation is not equivalent to hosted/KVM or physical measurement.
Existing accepted hosted manifests retain their source/run/version provenance;
new local runs have separate evidence and unchanged budget limits. See
[measurement procedure](../testing/M1/performance.md) and [continuity procedure](../testing/M1/file-continuity.md).

## Browser qualification

The [stdlib WebDriver runner](../../tools/browser-qualification.py) tests a
synthetic loopback console, with separately supervised fixture and driver groups.
Firefox/geckodriver artifact pins currently support Linux only. Safari uses the
installed macOS driver when the operator enables remote automation. Use the
[browser checklist](../testing/M1/browser-checklist.md) for exact commands and
manual screen-reader/native-zoom checks; manual accessibility remains pending.
Automated flows do not certify accessibility or actual NAS client compatibility.
External browser applications launched outside the owned group require their
normal browser/session cleanup and cannot be claimed stopped by group supervision.

## Release gates

Add dependency vulnerability checks, native OS/architecture integration, actual
protocol clients, restore/failure/upgrade tests, port validation and reproducible
release evidence as components arrive. Physical controller/disk/NIC and power-loss
tests belong in a disposable lab. Publish checksums, SBOM/provenance and supported
versions for deployable releases. Local success and cross-compilation do not
establish these results. No deployment or production service is enabled by checks.
