# Horvium

**Dependable storage on FreeBSD.**

Horvium is a dedicated NAS management system for FreeBSD and ZFS, designed to
remain lightweight on a 16 GB home server and grow to larger commercial storage
systems. The HP MicroServer Gen8 is a reference configuration, not a hardware
requirement or a limit on disks, memory, pools, or datasets.

**Status:** M0 foundation and the main M1 read-only discovery/management slices
and crash containment, native resources, HTTP abuse recovery, independent SMB/NFS
continuity and automated Firefox qualification are accepted. The consolidated
qualification record was accepted in PR 18. M1 is Validating; Safari/manual
accessibility acceptance remains pending. See the
[M1 qualification review](docs/testing/M1/qualification-status.md) for evidence and
remaining gates. There is no installable port, appliance image or production
support claim.

M2 storage lifecycle design is approved in [PR 20](https://github.com/gonz0w/horvium/pull/20): [PRD](docs/milestones/M2/PRD.md),
[EDD](docs/milestones/M2/EDD.md), [Test Design](docs/milestones/M2/TDD.md) and
[implementation sequence](docs/milestones/M2/implementation.md). The first
model subset was accepted in PR 21; bounded property-source parsing passed on
both disposable FreeBSD targets and was accepted in PR 22. Dataset identity/type
binding and version drift were accepted in PR 23. Filesystem mount/encryption
constraints were accepted in PR 24. Pool identity/incarnation and policy
observations were accepted in PR 25. Complete native feature observations and
exact common profile checks were accepted in PR 26. Native vdev membership,
layout refusal and topology drift checks passed on both pinned guests;
[PR 27](https://github.com/gonz0w/horvium/pull/27) is open for review. See the
[topology qualification](docs/testing/M2/topology.md).
Other native gates, durable jobs and storage
writes remain pending; the runtime is still read-only.

## Agreed direction

- FreeBSD and ZFS; BSD-3-Clause license.
- SMB, NFS, and SFTP file serving; local users/groups initially.
- Snapshots, backup-peer registration, incremental replication, retention, and
  tested restoration in the first release.
- Native FreeBSD port/package first; an appliance image later.
- Go, server-rendered HTML5, hand-authored CSS, and HTMX when needed.
- BSD Classic interface, following the BSD Cloud design guidance.
- File serving continues when the management interface is unavailable.
- Fleet management, automatic failover, AD, VMs, and containers are outside the
  first release. Horvium itself does not host application workloads.

## Start here

1. [Project charter](docs/project/charter.md): users, outcomes, and boundaries.
2. [Architecture baseline](docs/architecture/baseline.md): components and state.
3. [Roadmap](docs/roadmap/milestones.md): foundation through the first release.
4. [Bootstrap status](docs/engineering/bootstrap-status.md): evidence and open gates.
5. [Contributing](CONTRIBUTING.md) and [agent instructions](AGENTS.md).
6. [M1 requirements](docs/milestones/M1/PRD.md),
   [engineering design](docs/milestones/M1/EDD.md), and
   [Test Design](docs/milestones/M1/TDD.md): approved implementation contracts.
7. [M1 implementation status](docs/milestones/M1/implementation.md): completed
   portions, evidence and remaining acceptance gates.

## Foundation checks

Development-only tools are checksum-pinned and installed under ignored `.tools/`.
Use Python 3.14.7 on a Linux/macOS development host for this tool installation.
Python is not a planned runtime dependency of the NAS.

```sh
make tools
make hooks
# Commit signed changes first; use the actual proposed PR title.
PR_TITLE='docs: describe the proposed change' make review
make lab-status
```

Native FreeBSD checks and physical hardware tests have separate gates. Passing
these foundation checks does not demonstrate functioning storage or file shares.
Use the [local qualification runner](docs/testing/local-qualification.md) for
supported disposable FreeBSD suites on the development Mac; see
[CI cost controls](docs/engineering/ci-costs.md) for the zero Actions policy. All
checks run locally; no persistent runner is installed. `make lab-stop` stops supervised development sessions and verifies
teardown. Maintainers inspect source-bound evidence before merging.

The Go module provides discovery, cache, administration, console and authenticated
CLI adapters. `make build` creates a cgo-free foreground development executable;
see the [M1 development runbook](docs/operations/M1-development.md). `make tools` also
installs checksum-verified Go 1.27.1 under `.tools/`; it leaves the system toolchain
alone. See [CI/CD](docs/engineering/ci-cd.md) for opt-in disposable native tests.

See [documentation index](docs/README.md), [testing strategy](docs/testing/strategy.md),
and [ports plan](packaging/ports/README.md). The selected remote is the private
`gonz0w/horvium` repository. See bootstrap status for creation and CI evidence.

## License and provenance

[BSD-3-Clause](LICENSE). Adapted BSD Cloud foundation tooling retains upstream
attribution; see [NOTICE](NOTICE.md).
