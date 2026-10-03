# M1 research manifest

Status: prepared, not executed. Prepared 2026-10-03. Pin versions at M1 entry.

## Questions

1. Which supported FreeBSD releases, Go toolchain, architectures, and packages
   define the initial runtime and port matrix?
2. Which native inventory outputs/contracts remain stable, bounded, and useful
   for device identity, ZFS health, service state, and large inventories?
3. Which privilege separation mechanism supports typed allowlisted operations
   without granting a root shell to the management service?
4. Which authentication, durable state, config ownership/drift, and job-recovery
   designs meet small-system budgets and commercial operation needs?
5. How should local identities/ACLs behave across SMB/NFS/SFTP and existing pools?
6. How should a backup peer pair securely, retain independent history, resume
   transfers, and restore data/configuration with explicit key ownership?
7. Which measurable memory/CPU/latency/inventory budgets and client workloads
   qualify the small machine and a representative larger deployment?
8. What port category/maintainer/dependency and staging choices fit the product?

## Versions to record

FreeBSD release/patch/architecture, bundled OpenZFS/features, Go, Samba, SSH,
HTMX asset/source checksum, clients/browsers, ports-tree commit, package repository,
and fixture/hardware details. None is pinned by this prepared manifest.

## Primary starting references

- [FreeBSD security and support](https://www.freebsd.org/security/)
- [FreeBSD ZFS handbook](https://docs.freebsd.org/en/books/handbook/zfs/)
- [FreeBSD network servers](https://docs.freebsd.org/en/books/handbook/network-servers/)
- [OpenZFS documentation](https://openzfs.github.io/openzfs-docs/)
- [Samba documentation](https://www.samba.org/samba/docs/)
- [Ports framework](https://docs.freebsd.org/en/books/porters-handbook/uses/)
- [rc.d scripting](https://docs.freebsd.org/en/articles/rc-scripting/)

Foundation reference inspection on 2026-10-03 checked the FreeBSD Ports, rc.d,
and security pages for research entry points. This is not a completed M1 research
pass. Store dated observations, reproducible fixtures, limitations, and resulting
ADR recommendations beside this manifest as the work proceeds.
