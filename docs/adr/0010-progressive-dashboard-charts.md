# ADR 0010: progressive dashboard charts

Status: Proposed for review; not adopted. Date: 2026-10-12. Owner: Cam (`@gonz0w`).
Refs [issue 166](https://github.com/gonz0w/horvium/issues/166).
[PRD](../design/dashboards/PRD.md), [EDD](../design/dashboards/EDD.md),
[Test Design](../design/dashboards/TDD.md), [source inventory](../design/dashboards/inventory.md).
The [maintainer's uPlot preference](../roadmap/dashboards.md) is settled; this
record proposes a concrete dependency/integration boundary for review.

## Context and alternatives

Current pool/dataset observations support comparisons, not time-series telemetry.
The product uses Go, semantic HTML, owned CSS and local HTMX. Exact tables and
keyboard/no-JavaScript access must survive an unavailable chart enhancement.

| Option | Assessment |
| --- | --- |
| Owned HTML/CSS and Go-rendered SVG only | Preferred for stage A current comparisons; adds no chart dependency, but does not implement future numeric-series interaction |
| uPlot plus a bounded owned adapter | Matches maintainer direction for later numeric plots; requires asset provenance, actual CSP/browser and lifecycle qualification |
| SPA/UI kit, second chart engine or telemetry service | No requirement justifies additional architecture/runtime cost |
| Render operator trend charts from two cached generations | No qualified history/identity/time contract; misleading persistence and coverage |

## Proposed decision and pin

Deliver stage A with owned presentation and exact tables. Investigate uPlot only
in a separately scoped, labelled stage B synthetic fixture. Do not load uPlot into
operator views before applicable qualification. Stage C needs a separate history
contract. Select **uPlot 1.6.32**, upstream tag commit
`e995b061e9fc5476a6d862cd2fb2ebc7452ca012`; this is the inspected candidate, not an
ongoing “latest” promise or security audit. [Tag](https://github.com/leeoniya/uPlot/tree/e995b061e9fc5476a6d862cd2fb2ebc7452ca012),
[package metadata](https://github.com/leeoniya/uPlot/blob/e995b061e9fc5476a6d862cd2fb2ebc7452ca012/package.json),
[license](https://github.com/leeoniya/uPlot/blob/e995b061e9fc5476a6d862cd2fb2ebc7452ca012/LICENSE).

Inspected original upstream files on 2026-10-12 through GitHub's contents API;
base64-decoded original bytes were retained privately, not vendored into product
assets. SHA-256 below binds those exact bytes without terminal-newline edits.
Gzip sizes use Python gzip, level 9, mtime 0; they are reproducible compressed
input sizes, not measured HTTP transfer, browser memory or application performance.

| Proposed runtime input | Original bytes | Gzip bytes | SHA-256 |
| --- | ---: | ---: | --- |
| [dist/uPlot.iife.min.js](https://github.com/leeoniya/uPlot/blob/e995b061e9fc5476a6d862cd2fb2ebc7452ca012/dist/uPlot.iife.min.js) | 51,081 | 21,991 | `19c8d4c6ad88929a79f4ae49d6f7161566dfd0ba3d15cc495e974f787eb78f1f` |
| [dist/uPlot.min.css](https://github.com/leeoniya/uPlot/blob/e995b061e9fc5476a6d862cd2fb2ebc7452ca012/dist/uPlot.min.css) | 1,857 | 758 | `df630c6a8d6f8eeaff264b50f73ce5b114f646ffd9a0bb74f049b0a00135fa04` |
| LICENSE (required accompanying notice) | 1,078 | Not a page asset | `8f989229699b4fe2f1a0432d0e9edc338a8a911e250e2d1b01ecd770a5f5b1bd` |

The JS/CSS pair consumes 22,749 of the proposed 32,768 gzip-byte budget, leaving
10,019 bytes for the owned adapter. Measure that adapter when it exists; no
unused headroom is a passing result. Pin original content and notices before
embedding, add checksum regression checks and record any deliberate transformation
with both original and served hashes. No asset or notice file is introduced here.

Upstream MIT notice names Leon Sorokin, copyright 2022. Preserve its full license
with distributed assets and in Horvium's third-party notices when actually added;
Horvium's own code remains BSD-3-Clause. Selected prebuilt IIFE has no runtime
package imports; inspected package metadata lists build-time Rollup/Terser tools,
not production dependencies. Horvium does not adopt those tools, npm or Node.
This browser dependency does not run in FreeBSD privilege/collection processes;
actual Safari/Firefox and local embedding still need qualification.

## Integration and security qualification

[Inspected readable distribution](https://github.com/leeoniya/uPlot/blob/e995b061e9fc5476a6d862cd2fb2ebc7452ca012/dist/uPlot.iife.js)
SHA-256 `1b71fc5e6b5b572922ed9941ed21d067207c8e5ecac0d35de66fd65d9686e791`
and [type contract](https://github.com/leeoniya/uPlot/blob/e995b061e9fc5476a6d862cd2fb2ebc7452ca012/dist/uPlot.d.ts)
SHA-256 `af44374f6e808be4f7f3fa5888b9c7574dca9f8a15a66d54b6763535802c8080`
were source-inspected. This is provenance/source review, not a vulnerability scan.

Source destroy unsubscribes its synchronization member, clears cursor/listener
tracking and removes its roots. Labels use textContent. Dynamic dimensions/cursor/
legend styling uses element style properties. The owned adapter must also free
its own observers/listeners/frames/data. Source review does not prove strict-CSP
layout, leak freedom, accessible Canvas or browser performance.

Follow [HTMX lifecycle events](https://htmx.org/events/): initial/added-subtree load
and before-cleanup disposal, with separate navigation handling. Preserve existing
local assets and CSP, eval/script-insertion/history restrictions. No unsafe-inline,
unsafe-eval, CDN, arbitrary plugin or script supplied by inventory. Real strict-CSP
browser failures block adoption and need an explicit reviewed compatible adaptation.
Exact tables and keyboard controls remain available regardless of enhancement.
Detailed bounds, lifecycle failure policy and observable tests are in the packet.

## Maintenance, removal and adoption

Upstream leeoniya/uPlot maintains the selected dependency; Horvium's maintainer
owns pin review, notices, adapter behavior and qualification. Inspect upstream
security/change reports manually at milestone/release dependency review and when
an advisory or compatibility problem is reported. Propose updates through signed
DCO PRs with new hashes and affected browser/CSP/lifecycle/resource tests. No
new paid automation, Actions workflow, external synchronization or background
update service is authorized. Checksum verification is not security qualification.

Removal deletes local assets, notices specific to removed copies and adapter
registration while retaining useful semantic tables/forms. No data migration,
collector or file-daemon change is required. Revisit if selected browsers cannot
meet CSP/accessibility/resource targets, a security issue cannot be addressed,
new interaction requires additional plugins or a real history contract changes
scope. No pan gesture, telemetry storage, public schema or shared library is chosen.

Maintainer review/merge must record adoption before bounded implementation.
An accepted ADR would authorize its stated design/slices only; actual browser,
resource/native, manual accessibility and broad dashboard/release gates stay open.
