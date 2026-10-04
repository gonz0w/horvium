# M1 research manifest

Status: initial research complete; M1 design approved and implementation active.
Updated 2026-10-03. Product progress: [implementation status](../../milestones/M1/implementation.md).
Evidence: [findings](findings.md), [source/version pins](source-manifest.json),
[QEMU spike](native-spike.md). Design: [PRD](../../milestones/M1/PRD.md),
[EDD](../../milestones/M1/EDD.md), [Test Design](../../milestones/M1/TDD.md).

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
and fixture/hardware details. FreeBSD/Go/HTMX/x/sys/Ports source candidates are now
recorded in the linked source manifest. No Samba, browser or file-client package
baseline is selected yet; select those at the applicable native/client milestone.

## Scope of this pass

Questions 1-3/8 have source inspection and a narrowly scoped QEMU experiment.
Question 4 has a proposed M1-only access/cache design; durable jobs remain open.
Questions 5-6 need later active sharing/protection designs before implementation.
Question 7 has proposed application budgets and test fixtures, not measured
hardware/throughput claims. Client priorities are macOS and Linux/BSD. The
maintainer authorized QEMU for native tests; no physical test host is assumed.

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
pass. The linked findings/spike now supply dated evidence and limitations for the
current pass. Neither research nor an initial VM pass closes M1 acceptance.
