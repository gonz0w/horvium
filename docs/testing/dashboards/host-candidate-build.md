# Reviewed dashboard host candidate build

Date: 2026-10-12. Status: development component awaiting maintainer review.
Owner: [issue 370](https://github.com/gonz0w/horvium/issues/370), parent
[Capacity outcome 290](https://github.com/gonz0w/horvium/issues/290), coordinated by
[M2 gate 131](https://github.com/gonz0w/horvium/issues/131) and
[Overview outcome 289](https://github.com/gonz0w/horvium/issues/289).
Authority: DASH-REQ-006 / DASH-PERF-001 and the approved
[resource Test Design](../../design/dashboards/resource-qualification.md).
Prerequisites: [strict receipt contract](resource-receipt-validator.md),
[native fixture wiring](native-fixture-bindings.md) and
[per-mode resource observations](resource-measurement-records.md).

## Source, review and compiler boundary

[The inactive host module](../../../tools/dashboard-candidate-build.py) exports
`build_candidate(root, review_directory, directory, source_commit=...,
base_commit=..., pr_title=...)`. It has no CLI, runtime consumer or runner
registration. It accepts no guest transcript and cannot open a performance
execution gate. Its caller supplies the intended full source/base IDs and actual
PR title, rather than deriving expectations from the stdout being qualified.

The existing Git verification requires the exact clean repository HEAD, original
verified signature and author DCO, a nonempty ancestor range, every contributor
signature and parsed author sign-off, and a Conventional Commit PR title. The
module reads the completed schema-2 local review from the owned review directory.
Source/tree/base/title/contributor range, required command, independently selected
plan, non-prose blob inputs and current environment fingerprint must agree.
Required-check and original deep logs are reopened and hashed, and supervisor
records must show successful completion and verified cleanup. Missing, linked,
altered, duplicate-field, unqualified or wrong-candidate records fail closed.

A documentation followup may use only the original executed deep evidence under
the existing exact-input reuse policy. Its safe owned relative path, original
report/log digests, result, base, key and source ancestry must agree. A reused
report cannot substitute for the original execution. This component requires
original deep evidence even for a prose-only range; such a range without it is
not a build candidate for this future native procedure.

The actual opened executable at the existing `.tools/go/bin/go` is checked against
the checksum-pinned bootstrap marker, host archive pin and reported Go version.
Pins, marker and executable are rechecked after compilation. This preserves the
existing bootstrap policy; it does not independently authenticate an entire
installed toolchain tree or turn local developer-controlled records into an
external certification. Maintainers review the actual source, original local
reports and compiler installation provenance.

## Original archive and controlled artifacts

Each invocation requires a fresh owned private directory beneath ignored
`.tools/dashboard-candidates/`. It captures `git archive --format=tar` for the
exact source. A bounded stable opened archive is hashed and extracted from those
same bytes into a temporary private source tree. Only bounded regular files and
directories with unique nonescaping paths are accepted; links and special files
are refused. Builds use that extraction, so untracked files and subsequent
working-tree edits cannot become build inputs. Source changes still invalidate
the overall candidate.

The existing pinned Go environment supplies FreeBSD/amd64/v1, cgo disabled,
local toolchain, GOENV/GOWORK disabled and empty GOFLAGS. Builds use `-trimpath`
and `-buildvcs=false`: source identity is the independently checked archive,
rather than an incidental checkout stamp. GOPROXY is disabled for this operation;
the existing verified dependency cache must already be prepared. No automatic
tool download, new dependency or compiler installation occurs here.

The three existing executable roles are now explicit:

| Contract role | Owned output and build | Use in the existing fixtures |
| --- | --- | --- |
| `management` | `horvium`, `go build ./cmd/horvium` | Actual management and built native helper commands |
| `helper` | `reader-fixture.test`, `go test -c ./internal/host/freebsd` | Existing generated command-boundary projection helper |
| `fixture` | `dashboard-native.test`, `go test -c ./cmd/horvium` | Resource controller and its synthetic re-executed helper |

One target build supplies identical hashes for both `15.1` and `14.5` entries in
the existing strict candidate contract. Those entries identify intended release
consumers, not two executed release builds or compatibility qualification. Each
release still needs its own independently checked artifacts, actual guest
versions, original fixtures, controls, measurements and teardown.

Each archive/build command runs in the existing short-lived owned process
supervisor, registered in the original host process directory even when its
working directory is the extracted source. Private logs, normalized build
arguments, target and artifact/log digests remain in the build record. Native
executables are never run. All source/compiler/review/archive/artifact inputs are
rechecked before the strict candidate identity can be returned. The extraction
is removed on success or failure.

## Failure, cleanup and remaining assembly

Success returns a private build record containing the existing strict candidate
identity; `build_complete` describes preparation and `native_execution` remains
false. Successful outputs stay available for a future reviewed runner to verify
and transfer. That consumer must verify the executed artifacts against this
record and bind actual release provenance and owned command/fixture context.
Simply copying the candidate into guest stdout does not establish execution.

Failure never returns a candidate. It retains the private incomplete record and
logs, records the exception class without publishing raw diagnostics, and removes
only the fixed owned executable/archive outputs. Interruption remains a failed
preparation. The supervised process records retain cleanup uncertainty. Cleanup
errors remain failures and do not fabricate success.

`remove_artifacts(root, directory)` requires an owned candidate directory, removes
those same fixed outputs and records removal while retaining private logs and
metadata. It does not delete unrelated paths or stop services. After work finishes,
use the existing `make lab-stop` / `make lab-status` and require verified teardown.
The current source review exercise removes its unexecuted cross-build outputs.

Portable behavioral tests use explicit synthetic compiler/artifact/archive data
and mocked commands; they do not produce native measurements. They cover source/
DCO and review mismatch, actual log and pin mutation, original deep reuse, archive
path/link/duplicate rejection, controlled builds, shared release identity,
post-build source/compiler/review/artifact mutation and failure/cleanup retention.
Shared-tool changes require the existing conservative full portable fallback.

Runner/guest execution guards, strict and legacy validators, native fixtures,
runtime, dependencies, workflows and historical manifests remain unchanged.
Independent original fixture expectations, actual native control completion,
guest and host supervisor teardown, candidate-to-transferred/executed artifact
binding and fresh sequential both-release execution remain subsequent reviewed
work. Broader #131/#289/#290 and lifecycle #305/#139 stay open; historical startup
uncertainty, browser/manual, continuity, hardware/commercial and release gates
remain. Runtime writes stay disabled. Rollback removes this inactive module,
tests and documentation without altering product behavior or old receipts.
