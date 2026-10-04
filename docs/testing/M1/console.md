# M1 console and CLI evidence

Status: implementing slice 5 for maintainer review. M1 remains unaccepted.
Date: 2026-10-04. Authority: approved [EDD](../../milestones/M1/EDD.md) and
[Test Design](../../milestones/M1/TDD.md). Refs [issue 1](https://github.com/gonz0w/horvium/issues/1).

## Implemented contracts

The console and CLI query the same authenticated cached observations. HTML full
pages and HTMX fragments preserve the exact generation, literal filter, numeric
sort, pagination and selected GUID/provider ID. Expired generations return 409
with an explicit reload; failed initial collection returns 503. Failed sections
remain unavailable, rather than reporting zero successful resources. Host facts,
collection status and section freshness are visible; shares, users, SMART and
protection are explicitly not collected. Pool state is not a disk-health guarantee.

The M1 internal JSON routes are GET `/api/overview`, `/api/pools`,
`/api/datasets`, `/api/providers`; POST `/api/refresh` requests one collection.
Queries accept generation, filter, sort, descending, offset and limit; overview
accepts generation only. Unknown, duplicate and malformed fields fail. Full and JSON replies have a 32 MiB encoded output budget; an oversized view
returns 503 before any inventory is written. This logical budget is not an RSS
guarantee; serialization allocation needs the scale gate. HTML adds
selected, which must identify a row in the displayed page. Provider graph IDs
remain collection-local. These interfaces are development contracts, not a public
release API. Capacities decode into Go uint64; GUIDs remain decimal strings.

GET `/login` issues a bounded challenge; POST `/login` exchanges the local token
for a session; POST `/logout` invalidates it. JSON clients use the approved
`/auth/challenge`, `/auth/login`, `/auth/logout` exchange. Every inventory,
fragment, collector-status and refresh handler requires authentication. Refresh
requires CSRF and never collects in a handler. Unknown paths return a gated 404;
they never redirect background asset loads through challenge creation.

The CLI uses a literal loopback HTTP endpoint, no environment proxy, no redirects,
a ten-second deadline and a bounded response. It reads a 64-character lowercase
hex token from an owned mode-0600 regular input, an anonymous pipe or an owned
mode-0600 named FIFO. Public named FIFOs are denied. No credential flag,
URL, environment setting or persistent cookie store exists. A command exchanges
credentials, returns typed JSON, and attempts logout before dropping local cookies.
Malformed/denied responses produce fixed errors without raw bodies or arguments.
Cancellation closes an operator input file/pipe so a missing token cannot leave
the CLI blocked after Ctrl-C.

Foreground `serve`, `helper`, `token-setup` and `token-rotate` commands assemble
existing libraries. Management requires the configured non-root UID, safe digest
and existing safe helper socket before binding validated loopback. Helper requires
root and owns no HTTP listener. Both stop on cancellation. Setup/rotation is a
local root action and prints the token once to its explicit output; no web setup
exists. No rc.d installation or service enablement is performed. See the
[development runbook](../../operations/M1-development.md).

## Browser and assets

The project owns semantic HTML5, CSS, the SVG mark and a small local script.
HTMX 2.0.11 is pinned and embedded; its license/provenance are in
[NOTICE](../../../NOTICE.md). The only change to the upstream minified asset is
one terminal newline. A checksum test prevents unnoticed replacement.

The script disables eval, script insertion, indicator style injection, external
requests and history snapshots. It permits swapping typed 400/409/503 workspace
errors. Links/forms work as full documents without HTMX; background collection
status updates only its polite status element, leaving resource details alone.
CSS provides visible focus, light/dark palettes, a small-screen layout, contained
table scrolling and reduced-motion rules. No SPA, UI kit, CDN or runtime bundler.

Actual browser testing exposed two integration defects, each with regression
coverage. Missing favicon requests previously reached login and replaced the
challenge; an explicit local favicon and unknown-path 404 now prevent that.
`Referrer-Policy: no-referrer` caused ordinary HTML POSTs to carry `Origin: null`.
The [Fetch Origin algorithm](https://fetch.spec.whatwg.org/#append-a-request-origin-header)
explains this interaction. Responses now use same-origin referrer policy, which
suppresses cross-origin referrers while retaining the local form's Origin.
Null/foreign origins remain denied, including when Fetch Metadata says same-origin.

On 2026-10-04 the Codex in-app browser completed real form login/logout,
HTMX filtering, keyboard Enter selection with focus retained on the stable row,
and POST refresh without replacing the selected resource/generation. The generated
host and hostile dataset name rendered as escaped text. No warning/error console
entries were observed during that path. Light appearance was inspected at
1280×720 and 390×844; the narrow page measured client/scroll width 390/390.
[Captured desktop fixture](console-synthetic.jpg) shows a selected dataset and an
explicit synthetic banner. Backend browser/engine version was not captured;
this is limited additional evidence, not the required Safari/Firefox matrix.
Dark appearance and JavaScript-disabled execution were not browser-qualified.

Exact native run/source records will be added after validation. The explicit `TestBrowserPreview` fixture serves generated observations
and marks every page synthetic; it is never a product mode or native evidence.
Safari/macOS, Firefox/Linux, JavaScript-disabled browser execution, assistive
technology and complete WCAG assessment remain separate acceptance gates.

## Tests and remaining gates

Tests were written to fail before implementation for authentication, HTML/fragment/
JSON parity, exact GUIDs, hostile names, initial unavailable state, strict queries,
generation expiry, CSRF, missing assets, referrer policy and CLI input boundaries.
Loopback integration uses the actual console and authenticated HTTP client.
Endpoint rejection, redirects and malformed responses exercise credential handling.
Local gates include `make check lint test secrets build go-check` (including race
and bounded existing fuzz targets).

The new native fixture runs only with root and `HORVIUM_NATIVE_VM=1` in disposable
QEMU. It creates its own directories, credential state and socket, launches the
actual root helper and drops UID before management/client requests. It checks the
foreground management runtime, protected-input CLI, typed host/pool observations,
HTML/fragment generation and GUID parity, anonymous denial and expired cursors.
To avoid a vacuous GUID assertion on a guest with no pools, the fixture also
creates a new sparse file-backed md device, one unique pool and one unmounted
dataset. It independently captures pool GUID/size and dataset GUID, then requires
nonempty CLI/JSON/full-page/fragment/selected-detail agreement. Dataset properties
must remain unchanged and pool/md teardown is verified. No force option or
operator disk/pool argument exists. It never changes existing guest resources,
enables daemons or exports host observations. Existing native source fixtures
also independently check isolated geometry and properties. A skipped test is not evidence.

| Test ID | This slice | Remaining full M1 gate |
|---|---|---|
| M1-HTTP-001 | Shared authenticated full/fragment/JSON routes, local assets, CSRF and strict queries | Required browser matrix and comprehensive abuse review |
| M1-AUTH-001 | Protected operator input and foreground setup/rotation wiring | Process restart and integrated bounded audit logging |
| M1-NATIVE-001 | Native helper/cache/HTTP/CLI/HTML fixture added | Native execution evidence and hardware diversity |
| M1-UI-001 | Semantic views, stable exact-generation selection and placeholders | Assistive technology, full browser/accessibility acceptance |
| M1-BUILD-001 | cgo-free executable with embedded local assets | Release provenance and package lifecycle qualification |
| M1-SCALE-001 | Existing cache limits plus bounded pages/HTTP responses | Measured whole-application RSS, latency and admission budgets |
| M1-NOCHANGE-001 | Query handlers have no native commands or storage mutation path | Independent file-daemon continuity during killed management/helper |

No NAS release, physical hardware performance, installation, file-client access,
backup operation or commercial deployment qualification follows from this slice.
