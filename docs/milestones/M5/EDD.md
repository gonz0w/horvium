# EDD: read-only package install and lifecycle

Status: Draft recommendation, not adopted service behavior. Owner: Cam (`@gonz0w`).
Date: 2026-10-23. [PRD](PRD.md), [Test Design](TDD.md),
[story #159](https://github.com/gonz0w/horvium/issues/159).

## Existing assembly and exact gaps

Read-only assessment at `a3a3bb405045e3470d27aad786fb1e3767df1e87`:

| Existing source | Implemented contract | Missing package integration |
| --- | --- | --- |
| [CLI](../../../cmd/horvium/main.go) | `helper`, `serve`, `token-setup`, `token-rotate`, authenticated `inventory`; SIGINT/SIGTERM context | No installer, rc.d script, readiness command or package-version command |
| [Service](../../../internal/service/service.go) | Separate root helper/non-root management, bounded logs, orderly HTTP shutdown | No daemonization, service startup/stop ownership or rc readiness bridge |
| [Config](../../../internal/admin/config.go) and [native files](../../../internal/admin/files_freebsd.go) | Strict v1 config; root-owned 0640/0644 config; service-owned 0700 state/0600 digest | No account/config provisioning or automatic permission repair |
| [Helper](../../../internal/helper/native_freebsd.go) | Protected socket directory, exact peer UID, root socket ownership | Existing/unknown socket is a refusal, not an entry to unlink on startup |
| [Port plan](../../../packaging/ports/README.md) | Package-first direction and required poudriere/lifecycle acceptance | No port Makefile, plist, distinfo or installable package |

Go inputs are `go 1.27.0`, toolchain `go1.27.1`, and `golang.org/x/sys v0.49.0`
in the actual go.mod/go.sum. Do not substitute the older proposed syscall version
from M1 research prose. The historical Ports-framework research pin is
`2e67f384802ff04abd34e706fb098e4cd999e7e6`; it is evidence to refresh/verify for the
selected build, not a claim that a current native package was built.

The native probe cleanup finding in [#414](https://github.com/gonz0w/horvium/issues/414)
also affects the assessed source. Its reviewed successor PR415 is not merged or
natively qualified here. A package may not promote that baseline into a safe
installed-runtime claim. Reuse the existing correction/qualification lane; do not
reproduce its vulnerable fixture or build another supervisor project in this story.

## Recommended distribution and staging

Use a private local Ports overlay and stock poudriere/pkg. Keep source private:
an authorized operator supplies a checksum-bound source distfile and module inputs
to the build's private distfile cache. No GitHub token enters a Makefile, package,
install script or build log. Public source hosting/upstream submission remain
separate decisions. Do not invent a public MASTER_SITES URL or reserve a UID.

Use the standard Go modules build facilities and native tested compiler, with
CGO disabled and embedded console assets. No Go/Python/Node toolchain, source tree
or module cache is installed as a runtime dependency. Build/staging is separate
from host installation. These are proposed payload paths under the existing
FreeBSD default prefix `/usr/local`, not files already delivered:

| Class | Proposed path/ownership | Package/removal behavior |
| --- | --- | --- |
| Executable | `sbin/horvium`, root:wheel 0555 | Contains CLI/helper/management and embedded assets; package-owned |
| Config sample | `etc/horvium.json.sample`, root:wheel 0644 | Only a sample; never silently becomes a usable active config |
| Service integration | `etc/rc.d/horvium_helper`, `etc/rc.d/horvium`, root:wheel 0555 | Proposed rc.d front ends; disabled by default; ownership gate below applies |
| Notices/runbook | `share/doc/horvium/`, root:wheel, directories 0755/files 0644 | Payload manifest lists every installed file/license |
| Active config | `/usr/local/etc/horvium.json`, root:service-group 0640 | Explicit administrator-owned state, not an overwrite/remove target |
| Runtime directory | `/var/run/horvium`, root:service-group 0750 | Explicit protected setup/start prerequisite; never recursively repair an existing entry |
| Credential state | `/var/db/horvium`, service-UID:service-group 0700; `token.sha256` 0600 | Created by existing explicit setup; never package payload or automatic removal |

Reject unsupported PREFIX/path substitutions rather than silently ignoring them;
the first concrete package documents and tests its chosen prefix. Samples must
require the actual selected nonzero UID/GID, never ship illustrative UID521 as a
working default. Every v1 key remains explicit and validated. Installation may
add package-owned inert files only; it changes no account, rc.conf, sysctl, pool,
mount, firewall, SSH, sudo or independently managed service configuration.

Use normal plist/staging accounting. Since an unedited generated active config
must also survive removal, keep active config outside package ownership rather
than relying on `@sample` alone for that stronger preservation promise. A future
package message explains prerequisites and exact accepted commands without
including generated secrets or silently running setup.

## Identity and explicit operator workflow

Preserve the current M1 identity model: an OS administrator explicitly supplies a
dedicated non-login service account/group and protected ancestors. The package
does not auto-create/delete an account. Static registered USERS/GROUPS allocation
is a possible later public-port decision, not a private UID default or a runtime
credential helper. No account provisioning command is added by this design.

The operator sequence is specified below; steps marked integration are not
implemented commands. It is a future disposable-lab procedure, not a run request.

| Step | Existing operation or proposed integration | Required observation |
| --- | --- | --- |
| Install | Stock `pkg` installation of the exact reviewed artifact; #160 supplies the actual artifact identity | Only declared inert payload changes; no listening process or credential |
| Prepare | Explicit OS-admin identity, protected directories and complete config from the reviewed sample | Numeric UID/GID match config; no unsafe ancestors/symlinks/unknown socket |
| Bootstrap | Root: `horvium token-setup -config /usr/local/etc/horvium.json` | Existing action persists only digest and displays token once to the operator; no package logging/capture of the token |
| Start helper | Root: `horvium helper -config /usr/local/etc/horvium.json` through the proposed rc bridge | Exact safe socket exists and original helper is owned; a PID alone is not readiness |
| Start management | Exact configured non-root UID: `horvium serve -config /usr/local/etc/horvium.json` through the proposed rc bridge | Root management refused; successful bind/authenticated inventory establishes usable readiness |
| Use HTML | Existing loopback `http://127.0.0.1:8088/login` | Token/CSRF/session/Host controls unchanged; remote access only the existing explicitly configured SSH forward |
| Use CLI | Existing `horvium inventory -endpoint http://127.0.0.1:8088 -resource overview` with protected stdin | Same authenticated generation/resource contract; operator-owned 0600 token file/pipe, no argv/env token |
| Stop/restart | Proposed service bridge stops management before helper, then starts helper before management | Original waits/ownership, expected socket retirement, config/digest preservation, session invalidation, no file-daemon disruption |

The existing [foreground runbook](../../operations/M1-development.md) remains the
only implemented command guide. No `/health`, check-config, setup wizard, storage
command activation, reload mode or HTTP credential route is invented. Readiness
is not declared from a daemon fork or a log line. An initial collection failure
must remain distinguishable from successful startup with unavailable inventory.

## Recommended stock service control and trust limits

Use rc.subr/rcorder for service conventions and ordering, with base `daemon` as
parent of each existing foreground command. Proposed rc front ends are
`horvium_helper` and `horvium`; both remain disabled on installation. Each uses a
separate root-owned supervisor pidfile under the protected runtime directory:
`helper.pid` and `management.pid`. Normal stop targets that supervisor, which
forwards SIGTERM to its child and waits. The management child's selected service
account is passed using daemon's existing `-u` facility; native evidence must
confirm both actual UID values match the existing application contract.

The proposed fixed launch composition uses `daemon -P` plus `-S` and separate
local log tags around the fixed absolute `horvium helper -config ...` or
`horvium serve -config ...` argv; management additionally selects the service user.
No `-r`/`-R`, shell-evaluated request arguments, HTTP control route or new runtime
controller is introduced. Exact rc script/daemon source and flags on both selected
releases are reviewed in #160 before the profile is called implemented.

Trust boundary: ordinary service control is an explicit OS-administrator action.
The root-owned rc configuration, installed executable, protected pidfile directory
and pidfiles are trusted administrative state. Use the exact pidfile plus expected
daemon command for rc status/stop, never broad process-name or process-group
search. Validate expected root ownership/type/mode and reject symlinks or foreign
entries. A duplicate start fails. An unexpected/unlocked stale pidfile or existing
unknown helper socket is reported for explicit administrator recovery; startup
never adopts it, auto-unlinks it or signals an inferred old PID.

This stock control convention is not an immutable original-process capability:
PID/name checks have a check-to-signal race if the original supervisor disappears.
We do not add that stronger guarantee to M1 or claim a root-compromise defense.
If an immutable control handoff is wanted later, it is a separate proposed design
choice with a demonstrated need, not a prerequisite invented for this lab package.
The known #414 probe cleanup defect remains independently required to fix/qualify;
it does not by itself justify a new service-control protocol.

Normal stop is management first, helper second; start is helper then management.
Use a finite reviewed stop-wait budget in the concrete rc integration/qualification
profile; expiration returns nonzero with uncertain state, without a name-based
kill, automatic restart, upgrade or deinstall. The existing five-second HTTP drain
is not a whole-process shutdown bound. Do not use force/fast methods that bypass
prerequisite checks. A helper-only outage retains existing unavailable/stale
behavior, with no package crash-restart loop. Readiness still requires actual
validated endpoints/authenticated observation rather than a daemon fork.

Qualification has a separate stronger obligation: its supervising fixture retains
actual original child/wait ownership, independently checks unrelated sentinels and
retirement, and never reconstructs custody from a later pidfile or ps snapshot.
A successful rc status/stop is not that proof. This distinction preserves the
existing native/test ownership gates without expanding the product architecture.

## Logging, update and recovery

Route existing bounded operational stderr records to a reviewed local system-log
sink using base facilities. Keep helper/management tags separate; credential
setup/rotation is interactive and never runs under that sink. Do not promise
lossless/durable audit delivery, change host logging settings on install, or add
raw inventory/body/cookie output. A blocked sink retains existing nonblocking
application semantics and its documented possible shutdown record loss.

Upgrade requires explicit operator quiescence, known original process retirement
and an artifact/config/ABI compatibility check before replacement. No post-install
restart and no migration is performed silently. Keep the previous signed package,
config and state metadata under operator custody. Config v1/token digest format
stay unchanged in this slice; unknown format refuses startup rather than rewriting.
Rollback is allowed only across the declared compatible pair, with services
quiesced and explicit restart. There is no imaginary older package to test today.

Do not restore an older verifier as an automatic credential-recovery technique:
it can restore an old token. Existing root-only token rotation plus management
restart remains the recovery path. If digest state is lost/unsafe, report it and
require an explicit separately reviewed replacement action; installation does not
re-bootstrap credentials or adopt an unexpected state directory.

Removal requires explicit service stop first. Unknown ownership/stop uncertainty
blocks package removal/upgrade rather than deleting a live executable and calling
it clean. Remove only plist-owned payload. Preserve active config, state/verifier,
service identity, retained logs and all datasets/file services; never use recursive
state deletion, account deletion or ZFS commands as a deinstall hook.

## Decisions before implementation is Ready

| Decision/binding | Concrete recommendation | Acceptance owner / remaining evidence |
| --- | --- | --- |
| D1 Source distribution | Private operator-supplied immutable distfile/local Ports overlay; no public source/privacy change | @gonz0w accepts distribution/version naming and actual maintainer identity; #160 binds signed source/tag or explicit lab version, archive SHA256 and custody |
| D2 Identity/setup | Existing explicit OS-admin non-login identity/config, existing root-only token setup; no automatic user creation | @gonz0w accepts package setup policy; native fixtures prove selected UID/GID and preservation |
| D3 Service control | Stock rc.subr plus daemon supervisor pidfiles, explicit administrator trust, no crash retry; retain the disclosed PID-race limitation | @gonz0w reviews this ordinary service profile; #160 reviews exact release source/rc scripts, #161 qualifies failure behavior and original fixture retirement; #414 remains independently required |
| D4 Exact build closure | Actual go.mod/go.sum, tested Go1.27.1, pinned compatible Ports tree and both FreeBSD targets | #160 records source/archive/module/compiler/Ports-tree/jail/ABI identities and full staged manifest before installation is Ready |

No version/tag, distfile checksum, upstream category, registered account number,
public URL or successful package/native run is fabricated to fill these fields.
Resolving routine implementation details follows the adopted contract; a change
to privilege, privacy or promised ownership needs its explicit reviewed decision.

## Primary basis and limits

Inspected 2026-10-23, as design references rather than frozen build inputs:
[Go USES](https://docs.freebsd.org/en/books/porters-handbook/uses/#uses-go),
[staging/rc/accounts](https://docs.freebsd.org/en/books/porters-handbook/special/),
[plist/config](https://docs.freebsd.org/en/books/porters-handbook/plist/),
[rc conventions](https://docs.freebsd.org/en/articles/rc-scripting/),
[daemon](https://man.freebsd.org/cgi/man.cgi?query=daemon&sektion=8&format=html),
[rc.subr](https://man.freebsd.org/cgi/man.cgi?query=rc.subr&sektion=8&format=html).
The selected release's exact source must be pinned/reviewed before implementation
claims. This source-only design ran no service, account/credential change, package
build/install, VM or native experiment. Existing holds/consumed attempts remain.
