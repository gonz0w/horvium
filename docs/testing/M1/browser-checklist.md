# M1 browser and accessibility qualification

Status: Firefox automated qualification passed; under review. Safari/manual
accessibility acceptance pending by maintainer choice. Owner: Cam (`@gonz0w`).
Date: 2026-10-04. Requirements M1-REQ-004,005,006; tests M1-UI-001,
M1-HTTP-001 and M1-AUTH-001. The approved TDD requires actual selected browsers
and assistive-technology evidence before UI acceptance.

## Automated procedure

GitHub Actions are disabled. On a prepared Linux lab, run
`python3.14 tools/bootstrap-browser.py` then
`python3.14 tools/browser-qualification.py --browser firefox --output .tools/browser-result.json`.
On the Mac use `--browser safari` with remote automation already enabled by the
operator. The runner supervises fixture/driver groups; external browser app
lifecycle and manual accessibility require separate observations. No persistent
runner is installed. Historical hosted results below retain their provenance.

`tools/browser-qualification.py` controls an actual browser using stdlib W3C
WebDriver requests. Firefox/geckodriver downloads have pinned official hashes.
The local Linux runner exercises JavaScript enabled/disabled in light/dark appearances
at desktop and 390/320 CSS-pixel widths. It verifies observed dark/reduced-motion
media, keyboard order and visible focus, literal hostile resource names, labels,
polite status, full-page fallback, HTMX filter/detail focus, actual status polling
without workspace replacement, sampled text/control contrast, collection feedback,
logout and protected navigation. Output whitelists versions and synthetic results.
A failed/skipped case cannot produce a complete passing report.

Safari uses installed `/usr/bin/safaridriver` and records the actual browser version.
The runner does not enable macOS automation settings. The first local attempt on
Safari 27.0 (22625.1.29.11.25), macOS 27.0 arm64, failed before session creation:
remote automation is disabled. No Safari flow passed and no settings were changed.
JavaScript-disabled Safari and assistive technology require manual observations.
Headless computed checks do not establish visual quality, screen-reader usability,
all element contrast, native zoom or WCAG conformance.

The first Firefox run (37232695284), source 6f1f229, failed the control-boundary
contrast assertion after keyboard/fragment/status-polling checks. Controls now
use distinct stronger light/dark borders; fine decorative rules retain their
existing colors. A complete rerun is required before recording browser pass.

The maintainer chose to keep manual accessibility acceptance pending on
2026-10-04. Automated results do not waive the cases below.

## Manual fixture and observations

Run `make browser-preview` in the checkout. It prints a loopback URL, displays a
“Synthetic browser fixture” banner, and expires after twenty minutes. The deliberately
public login value is 64 lowercase `a` characters. It observes no host/storage and
has no usable production credential. Stop with Control-C or `make lab-stop`; verify `make lab-status` afterward. Use the same functional
revision as the recorded automated run; record OS/browser versions and date.

For Safari on macOS (VoiceOver) and Firefox on Linux (available screen reader),
record pass/fail and a concrete observation for each item below. If no screen reader
is available, record that case pending rather than inferring it from markup.

| Case | Required observation |
|---|---|
| JavaScript enabled/disabled | Login, navigation, filtering, pagination, selection, collection feedback and logout work; disabled mode uses complete pages |
| Keyboard only | Skip link, navigation, filter/sort/apply, selected row, collection and logout are reachable in useful order with visible focus and no trap |
| Progressive refresh | Filter/apply and detail selection retain focus; status polling preserves details and current typing/focus |
| VoiceOver/reader | Landmarks, heading hierarchy, table headers, filter labels, selected-resource identity, help/error text and collection/status feedback are understandable; no excessive repeat announcements |
| Native zoom | At 200% and 400% browser zoom, text/controls remain usable; no overlap or lost actions; wide tables scroll inside their region |
| Narrow viewport | At 320 and 390 CSS pixels, navigation/forms/details remain reachable without page-level horizontal scrolling |
| Appearance | Light/dark show readable status and disabled/selected/focus states; meaning does not depend only on color |
| Reduced motion | With OS preference enabled, interactions remain useful without required animation |
| Session | Logout and subsequent protected navigation require login; no stale authenticated content is exposed by back navigation |

Report the functional revision, browser/OS/reader version, JavaScript mode,
appearance, zoom/viewport and observations. Keep screenshots/reports synthetic;
do not include real tokens, cookies, host inventory or customer data. Failures
need regression fixes and reruns. Manual acceptance remains explicitly pending
until observations and maintainer review are recorded.


The first contrast-fix run (37233083244) passed four flows, but its recorded narrow
viewports were 500 CSS pixels: Firefox's desktop window minimum clamped the request.
That run does not qualify 320/390-pixel reflow. A native-keyboard zoom attempt also
failed the asserted target width (37233378014), so it supplies no zoom/reflow pass.
The runner now uses Mozilla's explicit `--allow-system-access` and `/moz/context`
endpoint solely to remove the owned disposable Firefox chrome's minimum width,
then returns to content context. This changes no product CSS/CSP, OS setting or
operator browser profile. Actual `innerWidth` must match the requested CSS width.
See [Mozilla driver flags](https://firefox-source-docs.mozilla.org/testing/geckodriver/Flags.html).
Human zoom/usability and Safari remain pending by maintainer choice.


## Actual Firefox execution record

[Run 37233700130](https://github.com/gonz0w/horvium/actions/runs/37233700130), source
`062233c0158e7e6472be5d72f891cae1f4a3acf0`, passed all six executed jobs, including
actual Firefox and both FreeBSD native suites. The browser used Firefox 157.0,
geckodriver 0.37.1 (300705c65d1b), Python 3.14.7 and Go 1.27.1 on Ubuntu 24.04.5
LTS, kernel 6.17.0-1022-azure amd64/glibc 2.39. Runner image ubuntu24/20260927.320.1.

| JavaScript / appearance | Requested and actual CSS width | Text / control minimum contrast | Result |
|---|---|---|---|
| Enabled / light | 1280 | 5.046 / 3.141 | Pass, including real status polling |
| Enabled / dark | 390 | 5.529 / 3.999 | Pass |
| Disabled / light | 1280 | 5.046 / 3.141 | Pass |
| Disabled / dark | 320 | 5.529 / 3.999 | Pass |

All four flows verified keyboard order/visible focus, filter/detail behavior,
escaping, labels, reflow, collection feedback and logout. HTMX-enabled cases
retained apply/selection focus; actual status polling retained workspace/details
and typing focus. Disabled cases had no page HTMX execution and used full-page
forms. Observed theme and reduced-motion media matched their requested values.
Only narrow owned Firefox browser chrome minima were adjusted, using Mozilla's
explicit driver capability; the product document and CSS were unchanged by the
harness. This does not emulate a phone, certify all visual contrast, or replace
manual screen-reader/native-zoom usability evidence.

The [source manifest](../../research/M1/source-manifest.json) retains exact pins,
source file hashes and whitelisted per-scenario results. Local `make check lint
test secrets build go-check` passed; tests include checksum-tampering protection
and known contrast/alpha-rejection assertions. Full M1 acceptance remains open.

A final harness cleanup regression reproduces skipped owned-driver shutdown when
WebDriver session deletion fails. A finally block now always stops the owned
driver before propagating the error; failed cleanup cannot emit a complete passing
report. This changes the development harness only, with the passing UI flows
unchanged.
