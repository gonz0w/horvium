# PRD: read-only package install and lifecycle

Status: Draft for review, not approved or implemented. Owner: Cam (`@gonz0w`).
Date: 2026-10-23. Owning story: [#159](https://github.com/gonz0w/horvium/issues/159).
[EDD](EDD.md) and [Test Design](TDD.md) form this one design packet.

## Operator outcome and scope

An operator installs a private FreeBSD package, explicitly prepares the existing
protected identity/configuration/token state, starts the root inventory helper
and non-root management process, authenticates to HTML and CLI inventory, stops
and restarts management, and retains configuration and access to independent file
services. Installation must not become an implicit storage or security operation.

This is the first read-only **lab package** workflow. The first deployable NAS
release remains M1-M6; M7 still requires separate image qualification. Existing M1
browser gates and M2 writer/recovery gates remain open. Packaging the read-only
runtime enables neither the unregistered storage-control consumer nor a writer.

## Requirements and observable acceptance

| Requirement | Operator/dependent outcome | Test Design |
| --- | --- | --- |
| M5-PKG-001 Provenance | Exact signed source, private distfile, dependency/compiler/Ports-tree inputs and target ABI identify every package; no moving branch or unpublished upstream-port claim | M5-BUILD-001 |
| M5-PKG-002 Inert install | A complete staged payload contains the binary, samples, service integration, licenses and runbook; install starts no process, creates no credential and edits no unrelated configuration | M5-STAGE-001 |
| M5-PKG-003 Explicit setup | Existing non-login management identity, root-owned config/socket parent and token setup satisfy current native validators; unsafe or missing prerequisites fail without permission repair | M5-SETUP-001 |
| M5-PKG-004 Start/use | Helper runs as real/effective root; management runs as its exact configured non-root UID. Authenticated HTML and CLI read the same generation; no public listener or privileged HTTP command appears | M5-LIFE-001 |
| M5-PKG-005 Stop/restart | Explicit orderly stop/restart preserves config/verifier, invalidates old sessions, removes only an originally owned socket and leaves file daemons/data alone; uncertain ownership blocks destructive cleanup/restart | M5-LIFE-002 |
| M5-PKG-006 Operations | Bounded existing operational records reach the selected local log sink without credentials/raw inventory; no durable-audit or automatic crash-restart claim | M5-LOG-001 |
| M5-PKG-007 Upgrade/removal | Quiesced compatible upgrade and supported rollback retain operator state; deinstall removes package-owned payload but preserves active config, verifier, service identity and operator datasets | M5-UPGRADE-001 |
| M5-PKG-008 Actual qualification | Poudriere/staging and installed-lifecycle evidence cover both declared FreeBSD targets, independent original fixture retirement, failures and unchanged acceptance gates | M5-ACCEPT-001 |

## Compatibility and limits

Design targets remain FreeBSD/amd64 15.1 and 14.5; exact installed kernel, running
kernel, userland, package ABI and patch levels are bound in each future receipt.
Other architectures/hardware are unqualified. Preserve existing inventory,
collection, HTTP, session and logging budgets; packaging introduces no higher
resource limit. The bounded native attempt envelope belongs to a separately
reviewed execution request, not this design.

Non-goals: live install/accounts/credentials, remote HTTPS or proxy mode, automatic
boot enablement, automatic crash restart, public source publication, official Ports
submission, appliance images, storage mutation, new identity/password policy or
replacement of stock package/rc infrastructure. No test or operational action is
authorized by the documents.

## Inputs, flow and decisions

Assessment source is `a3a3bb405045e3470d27aad786fb1e3767df1e87`, not a qualified
package candidate. There is no release tag, distfile or package checksum to claim.
The [EDD](EDD.md) records concrete recommendations, source gaps and the finite
bindings required before a build is Ready. [#160](https://github.com/gonz0w/horvium/issues/160)
owns implementation/build/staging; [#161](https://github.com/gonz0w/horvium/issues/161)
owns actual installed lifecycle. Parent [#134](https://github.com/gonz0w/horvium/issues/134)
does not close when a design or component merges.

Delivery follows [Suite-Standards](https://github.com/gonz0w/suite-standards/blob/4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22/docs/backlog-delivery.md).
The dated source-only WIP exception is recorded in #159/#419; all paused batches
remain counted. Existing stronger local checks, review and operational holds apply.
Design acceptance requires independent review, actual required source gates and
maintainer resolution of the EDD decisions. Source preparation alone is not Done.
