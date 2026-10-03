# ADR 0001: FreeBSD and hardware-neutral NAS scope

Status: Accepted. Date: 2026-10-03. Authority: maintainer project discussion.

## Context

A dedicated NAS needs file services, storage integrity, and efficient backup.
The maintainer considered OpenBSD/FreeBSD and selected FreeBSD with ZFS. The
MicroServer defines their environment, while larger commercial deployments are
an explicit target.

## Decision

Target FreeBSD with ZFS. Do not implement an OpenBSD storage backend in release
one. Avoid device/controller/disk-count/memory assumptions tied to the reference
machine. Use lightweight Go and server-rendered HTML5/CSS/HTMX management with
BSD Classic UI. File services run independently of management availability.

## Consequences and alternatives

One native storage design keeps the initial product tractable. Larger systems
need bounded discovery/jobs and measured qualification. Hardware neutrality does
not promise every FreeBSD driver/architecture is already tested. OpenBSD support
would require a separate storage/recovery contract and is deferred.

Revisit if a concrete deployment requirement cannot be met on supported FreeBSD,
or verified scale evidence requires changing management architecture.
