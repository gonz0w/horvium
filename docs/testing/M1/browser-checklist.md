# M1 browser and accessibility qualification

Status: procedure implemented; execution pending. Owner: Cam (`@gonz0w`).
Date: 2026-10-04. Requirements M1-REQ-004,005,006; tests M1-UI-001,
M1-HTTP-001 and M1-AUTH-001. The approved TDD requires actual selected browsers
and assistive-technology evidence before UI acceptance.

## Automated procedure

`tools/browser-qualification.py` controls an actual browser using stdlib W3C
WebDriver requests. Firefox/geckodriver downloads have pinned official hashes.
The Linux workflow exercises JavaScript enabled/disabled in light/dark appearances
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
has no usable production credential. Stop with Control-C. Use the same functional
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
That run does not qualify 320/390-pixel reflow. The runner now asserts the actual
CSS width and uses native keyboard 200% zoom in a 640/780-pixel window for the two
narrow Firefox cases. Product CSS and browser chrome are not overridden. A complete
rerun must reach the requested viewport; native 400% and human usability remain
manual gates.
