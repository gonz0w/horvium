# FreeBSD port/package plan

Status: planned distribution contract. No installable port exists at M0.

The maintainer selected a FreeBSD port first, then a complete appliance image.
Develop a local port against real tagged source before requesting upstream Ports
inclusion. Category, maintainer address, source URL, distinfo, Go version, exact
package dependencies, and installed filenames are finalized with release design.
Do not publish speculative Makefiles or claim upstream acceptance.

## Required behavior

- Native FreeBSD Go build using the appropriate ports framework facilities.
- Correct staged paths, configuration samples, ownership, restricted credentials,
  rc.d startup/stop/restart, logging, and upgrade behavior.
- Preserve operator datasets/config/secrets on upgrade and removal.
- Installation never formats disks, creates pools, enables shares, upgrades pool
  features, or silently changes unrelated system configuration.
- Optional native services use explicit dependency/license decisions.
- Test clean builds with poudriere, staging/plist correctness, permissions,
  install/upgrade/deinstall, and service recovery on supported releases.
- Appliance images later consume the tested package, with distinct boot/install
  and OS-update evidence.

Primary references: [Porter's Handbook](https://docs.freebsd.org/en/books/porters-handbook/),
[USES macros](https://docs.freebsd.org/en/books/porters-handbook/uses/),
and [rc.d scripting](https://docs.freebsd.org/en/articles/rc-scripting/).
