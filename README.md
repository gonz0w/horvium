# Horvium

**Dependable storage on FreeBSD.**

Horvium is a dedicated NAS management system for FreeBSD and ZFS, designed to
remain lightweight on a 16 GB home server and grow to larger commercial storage
systems. The HP MicroServer Gen8 is a reference configuration, not a hardware
requirement or a limit on disks, memory, pools, or datasets.

**Status:** M0 foundation and M1 design accepted; native discovery implementation
is under review. No NAS daemon, GUI, installable port,
appliance image, or production support claim exists yet.

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
make check lint test secrets
make go-check
```

Native FreeBSD checks and physical hardware tests have separate gates. Passing
these foundation checks does not demonstrate functioning storage or file shares.

The Go module currently provides internal read-only discovery libraries and tests;
there is no operator command or management listener to launch. `make tools` also
installs checksum-verified Go 1.27.1 under `.tools/`; it leaves the system toolchain
alone. See [CI/CD](docs/engineering/ci-cd.md) for opt-in disposable native tests.

See [documentation index](docs/README.md), [testing strategy](docs/testing/strategy.md),
and [ports plan](packaging/ports/README.md). The selected remote is the private
`gonz0w/horvium` repository. See bootstrap status for creation and CI evidence.

## License and provenance

[BSD-3-Clause](LICENSE). Adapted BSD Cloud foundation tooling retains upstream
attribution; see [NOTICE](NOTICE.md).
