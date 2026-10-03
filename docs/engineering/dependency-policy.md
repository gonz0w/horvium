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
