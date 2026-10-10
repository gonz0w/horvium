# UI baseline: BSD Classic

Follow the BSD Cloud Console design direction reviewed on 2026-10-03: readable
modern sans-serif typography, monospace technical values, restrained warm red
branding, neutral surfaces, fine rules, precise tables, and subtle control corners.
Use coherent light/dark appearances and responsive layouts. Labels and icons
reinforce status; color alone never communicates success, warning, or danger.

## Information architecture

Overview, Storage, Shares, Users, Protection, Jobs, and System. Keep host identity
and health visible. Lists support filtering, pagination, selection, and a resource
preview; full details expose identity, state, configuration, relationships,
protection, and relevant events. Avoid cloud regions/tenants or other context
that a single NAS does not have. Show the backup peer without implying failover.

## Interaction contract

HTML5 forms and links are primary. Go renders full pages and validates every
action. HTMX progressively updates fragments, dependent fields, and job status.
Do not make authorization depend on UI visibility or JavaScript. Pin and serve
critical assets locally; no runtime CDN requirement.

Use project-owned CSS for hover, focus, selection, loading, error, empty, and
success states. Brief fades/slides may clarify changes. Honor reduced motion;
preserve focus and stable rows during refresh. No animation engine, UI kit, SPA,
or client-state framework. Necessary custom JavaScript needs a documented reason.

The planned [uPlot dashboard story](../roadmap/dashboards.md) extends this baseline
with progressive charts, exact-value tables, coordinated drill-down and semantic
color roles. It records deferred work and data prerequisites, not implemented UI.

Destructive workflows identify affected resources, consequences, and preconditions
in a review step. Do not present irreversible storage operations as ordinary
settings toggles. Long operations show durable job status and honest cancellation
limits. First-use flows lead through discovery, storage adoption/creation, identities,
shares, and protection, with advanced options available where justified.
