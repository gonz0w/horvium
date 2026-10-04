# Dependency policy

Prefer FreeBSD base facilities, Go standard library, a small focused library,
then a native service with an explicit boundary. Each runtime dependency needs
purpose, version/source pin, license/notice, FreeBSD support, transitive cost,
security maintenance, update plan, and removal path. Large frameworks or changes
to architecture require an ADR. Samba is a separate file-service dependency;
its license and package/distribution obligations must be reviewed before packaging.

No runtime Python, Node, external database, message broker, or CDN is implied by
this foundation. Python and checksum-pinned linters/scanners are development tools.
HTMX is the selected progressive interaction mechanism; its exact version and
asset verification are chosen before embedding it. CSS polish introduces no UI
or animation dependencies.

Tool pins originate from BSD Cloud's reviewed foundation. Bootstrap verifies
artifact hashes. Refresh and review pins at milestone entry; do not describe them
as latest. Dependabot proposes updates for Actions and pip, with Go modules added
when they exist. Automated updates do not bypass normal review. Security features
that require host/subscription support are enabled only after capability verification.

## Current product module

The M1 crash-containment candidate uses FreeBSD base `/usr/bin/timeout` around
fixed native probes. It is supplied/maintained with the selected OS, adds no Go
module or separately packaged service, and uses the standard library's FreeBSD
parent-death signal. Record the actual utility's hash and native behavior per
qualified guest. Its source/procedure are linked in
[crash-containment evidence](../testing/M1/crash-containment.md); native resource
qualification must account for the added transient process. Removal requires
equivalent owner-death/subtree containment, not reverting to an orphaned probe.

The approved M1 helper uses `golang.org/x/sys/unix` v0.48.0 solely for FreeBSD
`GetsockoptXucred`; source `613e2570718ecde85c04e69ebd5585c3881c442c`,
Go minimum 1.26.0, BSD-3-Clause. Go checksum-database verification and go.sum
protect module downloads; native CI also runs `go mod verify`. There are no
transitive modules or cgo requirements. Dependabot proposes gomod updates weekly;
each requires ordinary review and native denial tests. The Go project maintains
the syscall adapter. Remove it if an equivalent base Go wrapper replaces the
FreeBSD credential adapter. See [notice](../../NOTICE.md) and
[helper evidence](../testing/M1/helper.md); no dependency vulnerability scan is
claimed by checksum verification alone.

## Fixture-only native file daemons

M1 continuity tests install FreeBSD net/samba423 only in disposable guests,
recording actual version/origin and daemon/client hashes. It is a separate GPL
file service, not linked or copied into the BSD-licensed Go program. Native base
NFS supplies the companion test. Private fixture configurations are removed and
the guest discarded; no package/production distribution is implied. Review Samba
license/redistribution, transitive packages, security maintenance and selected
version before M3/M5 adoption. See [continuity procedure](../testing/M1/file-continuity.md).
