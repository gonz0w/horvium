# Project charter

Status: agreed product direction, 2026-10-03. Detailed implementation designs remain
milestone-specific. Project identity: Horvium; repository/package slug: `horvium`.

## Purpose

Provide dependable, understandable NAS administration on FreeBSD. Serve files
through established native services and make storage health, permissions,
protection, recovery, and configuration easy to understand. Home and commercial
operators use the same core product; deployment size is not fixed by the reference
machine.

## First-release outcomes

- Discover disks, pools, datasets, shares, and service health without modifying them.
- Manage supported ZFS operations with clear preconditions, review, and recovery.
- Configure SMB, NFS, and SFTP with local users/groups and tested permission behavior.
- Register a backup peer, validate its identity/capabilities, schedule snapshot
  replication, see backup age/failures, and prove restoration.
- Monitor capacity, scrubs, disk health, and jobs with bounded resource overhead.
- Install through a tested FreeBSD port/package with native service integration.
- Keep file serving available independently of the administration service.

## Constraints

Go standard-library-first management; semantic HTML5 and hand-authored CSS;
HTMX only where it improves server-driven interactions. BSD Classic UI. ZFS is the
storage foundation. BSD-3-Clause is the project license. Package installed services
and state without overwriting unrelated host configuration.

## Deployment range

The 16 GB HP MicroServer Gen8 with four 3 TB SATA disks defines an initial small
reference profile. Larger commercial hosts are an explicit design target, but
supported hardware/architectures and scale limits depend on native evidence.
No controller, bay count, memory size, pool count, or dataset count is hardcoded
from the reference machine. Bound caches, job concurrency, subprocess output,
request sizes, and polling; paginate inventories. OS/hardware support is limited
by FreeBSD, dependencies, and Horvium's tested compatibility matrix.

## Deferred scope

Fleet management, automatic failover/HA, AD integration, and an appliance image
follow later milestones. Backup-peer awareness in release one is not clustering.
VM/container hosting, application stores, and generic workload orchestration are
outside the dedicated-NAS mission. Optional Time Machine and additional file
protocols require their own client compatibility evidence.

## Unresolved before product design

Supported FreeBSD/Go/dependency versions; representative clients/workloads;
identity and ACL rules; imported-system adoption; config/state format; encryption
and key lifecycle; restore objectives; resource budgets; backup-peer trust model;
port maintainer/category; confidential reporting; additional lab availability.
These do not require guessing final schemas during M0.
