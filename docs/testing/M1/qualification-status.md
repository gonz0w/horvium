# M1 qualification review

Status: under review; M1 remains Implementing. Owner: Cam (`@gonz0w`).
Date: 2026-10-04. Approved [PRD](../../milestones/M1/PRD.md),
[EDD](../../milestones/M1/EDD.md) and [Test Design](../../milestones/M1/TDD.md).
This consolidates evidence across the accepted slices and remaining review PRs.
It does not approve M1, waive a test, or authorize merging any PR.

## Evidence and current limits

| Test ID | Evidence | Current result / boundary |
|---|---|---|
| M1-NATIVE-001 | [Discovery](discovery.md), [helper](helper.md), [console/CLI](console.md) | Actual adapters, independent owned GUID/size, unprivileged transport and CLI/HTML passed on both selected FreeBSD guests; hardware/controller support remains unqualified |
| M1-PARSE-001 | Discovery unit/native cases and pinned parser fuzz checks | Typed empty/hostile/malformed/truncated/overflow/control-byte/enum/XML and allocation bounds passed; no customer inventory fixtures |
| M1-FAIL-001 | [Collector](collector.md), [cleanup](probe-cleanup.md), [crash containment](crash-containment.md), [native resources](native-resources.md) | Native/process/transport failure classes and stale/recovery pass; integrated native resource/crash workflow passed on both pinned guests (37233391796); combined assembled-source confirmation running |
| M1-IDENTITY-001 | Discovery serial/lunid conflicts, provider classification and native rename assertions | Passed fixture/model identity contracts; controller persistence and later storage adoption safety require new evidence |
| M1-PRIV-001 | Helper kernel credentials and [denial events](helper-events.md) | Actual configured UID allowed; wrong/root/forged caller, unsafe socket/client peer cases denied; no HTTP shell/privilege path |
| M1-NOCHANGE-001 | Owned properties, immutable config/digest, mutation denials and [file continuity](file-continuity.md) | Passed native reader/helper/runtime and independent SFTP/SMB/NFS fixtures; only disposable storage used |
| M1-AUTH-001 | [Security](security.md), runtime restart and [HTTP abuse](http-abuse.md) | Token/digest permissions, challenge/session expiry/capacity/throttle/rotation/logout/restart and log redaction passed |
| M1-HTTP-001 | Security/router tests, actual TCP abuse and [browser flows](browser-checklist.md) | Native denials, bounded headers/body/accepted connections/deadlines/recovery pass; Firefox synthetic flows pass; Safari acceptance pending |
| M1-CONTRACT-001 | Console full/fragment/JSON/CLI parity and cursor expiry | Exact-generation pagination/selection and honest unavailable/partial/stale states passed; no mixed-generation rows |
| M1-UI-001 | Actual Firefox and prepared manual browser checklist | Four JS on/off, light/dark, keyboard/focus, polling and verified narrow flows pass; Safari, screen readers, human zoom/usability pending by maintainer choice |
| M1-SCALE-001 | One-worker/deduplicated refresh, bounded transport/cache/response tests and HTTP connection cap | Portable/race/native admission/cancellation/bounds/recovery pass; finite pre-handler admission supplements response admission |
| M1-PERF-001 | [Synthetic budgets](performance.md), native resource workflow | Accepted synthetic measurements; native-small/10,000-row projection budgets passed on both pinned guests (37233391796); final combined confirmation running; not whole-NAS ARC/disk throughput |
| M1-RESTART-001 | [Runtime](runtime.md), crash containment and file continuity | Actual process restart/session invalidation/config preservation, built-helper owner death and seven independent SMB/NFS reads with teardown pass; integrated measured crash/stale/recovery/boot-state assertions passed (37233391796) |
| M1-BUILD-001 | SDK/module/local asset pins, native cgo-free builds, provenance manifest | Both guest builds and Linux/macOS development checks pass; no runtime Python/Node/browser/cgo/framework/container dependency; release/package lifecycle remains M5 |

Evidence includes exact functional revision/file hashes and actual versions in
[the source manifest](../../research/M1/source-manifest.json). Reports name the
fixture limits; skipped opt-in tests never qualify a capability. Go performance
budgets exclude native child processes, OS/ZFS ARC and fixture/file-daemon processes.
Ten thousand projected rows are not ten thousand real ZFS datasets or commercial
qualification. Loopback file-daemon continuity is separate from macOS/Linux/BSD
protocol interoperability, ACLs, locking, restoration and release configuration.

## Open review and acceptance actions

The remaining focused PRs are stacked in this order; each stays open for the
maintainer and requires its own explicit merge authorization:

1. [PR 13](https://github.com/gonz0w/horvium/pull/13): abrupt-owner-death native probe containment.
2. [PR 14](https://github.com/gonz0w/horvium/pull/14): native resources and integrated built-helper crash/recovery.
3. [PR 15](https://github.com/gonz0w/horvium/pull/15): pre-handler HTTP bound and assembled denial/recovery.
4. [PR 16](https://github.com/gonz0w/horvium/pull/16): independent SMB/NFS continuity and teardown.
5. [PR 17](https://github.com/gonz0w/horvium/pull/17): Firefox qualification, contrast repair and manual UI checklist.
6. [PR 18](https://github.com/gonz0w/horvium/pull/18): this consolidated qualification and acceptance record.

The combined assembled-source workflow must finish and its results/hashes must be
recorded before the remaining automated qualification is complete. Required
repository, DCO and title checks must pass for the reviewed heads. Rebasing a
review branch preserves signed commits and DCO; recorded historical test revisions
remain named explicitly rather than substituted by a documentation commit.

The maintainer explicitly chose to keep manual accessibility pending on 2026-10-04.
Safari automation was unavailable before session creation because remote automation
was disabled; no operator browser setting was changed. The approved Test Design
states “All listed product tests are M1 acceptance gates” and requires actual
selected browsers and manual screen-reader/keyboard findings. Therefore M1 cannot
be marked accepted until those observations, remaining review and explicit
maintainer acceptance are recorded. See the [prepared checklist](browser-checklist.md).

M2 storage operations require their approved milestone PRD/EDD/Test Design before
implementation. SMB/NFS service management, users/ACLs, snapshots/replication,
retention/restoration, ports packaging and the image remain the following milestone
work in [the roadmap](../../roadmap/milestones.md). No production NAS or commercial
support claim follows from this M1 read-only qualification.
