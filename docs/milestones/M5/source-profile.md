# Private read-only package source profile

Status: source implementation for review, native qualification open.
Date: 2026-10-23. Refs [#159](https://github.com/gonz0w/horvium/issues/159),
[#160](https://github.com/gonz0w/horvium/issues/160),
[#161](https://github.com/gonz0w/horvium/issues/161).

## What is implemented

The [overlay](../../../packaging/ports/horvium/Makefile) composes the existing
read-only helper and management commands through stock rc.subr and base daemon.
It contains fixed command/identity selection, separate protected supervisor
pidfiles, disabled service defaults, bounded TERM observation and refusal to
restart after uncertain stop. Package guards refuse active/unknown profile state
without signals or cleanup. Active config, credential state, identities and data
are not package-owned files. Setup/credential actions remain explicit existing
OS-administrator commands, never install hooks or logged service actions.

No new privileged controller, HTTP bootstrap, storage writer or restart loop is
introduced. The stock baseline follows the maintainer's continuation direction;
it needs no further architecture decision with these unchanged guarantees.
The [operator guide](../../../packaging/ports/horvium/files/package-lifecycle.md)
discloses the administrator/PID trust limit and readiness/cleanup distinctions.
Its protected `/var/run/horvium` directory is an explicit setup prerequisite;
after a boot that clears it, an administrator must prepare it again. Automatic
boot provisioning/enablement is outside this first lab profile.

## Application and overlay are two bound inputs

The signed application source is
`d392a97a79da9c5374a1a7f1c5e8cee2f479f380`, tree
`6952bc5cd1fa79cdb7a04e4a41398566e9fbb1ab`. Its product Go code is unchanged
from assessment baseline `a3a3bb405045e3470d27aad786fb1e3767df1e87`.
The overlay is the separately reviewed source candidate containing this document;
future package receipts must bind both. No release tag is invented.

The private lab version is `0.0.0.20261023`. Prepared distfile:
`horvium-0.0.0.20261023-d392a97a79da9c5374a1a7f1c5e8cee2f479f380.tar.gz`,
4,284,588 bytes, SHA256
`da5e116776f5b96939ff68a8131f356a944b7ed18f2e460c184a0fa6d6c1a040`.
`source.mk` and `distinfo` contain these actual bindings. Preparation verified the
source SSH signature, unchanged go.mod/go.sum, the cached module contents and
325 vendored files. `PORT-SOURCE.json` in the bundle records the signed tree,
original Git-archive hash, SDK binary/version and vendor manifest hash. The Go
SDK was the already cached Go1.27.1; no new SDK/module was downloaded. Module
downloads were disabled with GOPROXY=off, not an OS-level network isolation claim.
No native package or installed runtime was produced.

The overlay uses the pinned M1 Ports framework commit
`2e67f384802ff04abd34e706fb098e4cd999e7e6`, including its
[Go USES source](https://github.com/freebsd/freebsd-ports/blob/2e67f384802ff04abd34e706fb098e4cd999e7e6/Mk/Uses/go.mk).
Its module mode supports a pre-vendored source tree and fixed staged Go targets.
The native build still needs the full actual Ports/jail/ABI/compiler binding;
source inspection cannot qualify framework execution or staging/plist correctness.

Service review inspected both existing pinned FreeBSD source revisions:
15.1 `e38a7085f3a8ecd317c947591fe42b5ca9ab317b`
([daemon](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/usr.sbin/daemon/daemon.c),
[rc.subr](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/libexec/rc/rc.subr))
and 14.5 `231eee334c098a8398bc959202415c48058f67f0`
([daemon](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/usr.sbin/daemon/daemon.c),
[rc.subr](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/libexec/rc/rc.subr)).
The selected supervisor flags create mode0600 parent pidfiles, select syslog and
drop identity in the management child; TERM is forwarded to that child. The
custom stop/restart methods retain uncertainty instead of using force/default
restart behavior. Actual child identity, signal/wait semantics and exit ordering
must still be qualified on both installed targets.

## Story-to-acceptance mapping and remaining work

| Owning story / criterion | Source result | Acceptance still required |
| --- | --- | --- |
| #159 operator design / Ready-DoD | Independently reviewed stock design, concrete setup/use/restart/preservation contracts and bounded source batch | Final source gate/review; native acceptance is not design closure evidence |
| #160 M5-BUILD-001 | Signed immutable private source and complete offline vendor closure; concrete port and license payload | Both-release clean poudriere build, exact native compiler/Ports/jail/ABI, package digest |
| #160 M5-STAGE-001 | Explicit payload/plist/sample, inert hooks, disabled rc defaults | Real staged manifest/modes, missing/extra payload and package script behavior |
| #161 M5-SETUP/LIFE/LOG/UPGRADE/ACCEPT | Implemented service composition and operator guide, portable control-flow cases | All named native install/start/authenticated-use/restart/upgrade/removal, interruption, resource and original-fixture cleanup cases |

Portable tests replace id/stat/daemon/signal commands with fixture stubs and
exercise actual shell branches. They establish fixed argv, root-helper/non-root
selection, refused unsafe ancestor/pidfile/command state, guarded package hooks,
TERM-only stop and no restart after uncertainty. Archive tests reject traversal,
links and special objects. They start no product daemon and signal no real PID.
ShellCheck is included in the normal lint target. Passing these checks does not
establish FreeBSD permissions, rc.subr/daemon compatibility or native containment.

Native #414 correction/qualification, package build and #161 installed acceptance
remain indispensable before usable-package claims. No image/VM, native install,
credential, service enablement, live host/security setting or deployment was run.
Historical failures, unknown cleanup evidence and consumed attempts are preserved.
This is the existing coherent #159/#160 batch under the dated WIP exception;
blocked GitHub updates are not reported as posted or as changed issue status.
