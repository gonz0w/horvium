# Prepared dashboard workload and phase-receipt adapter

Date: 2026-10-12. Status: portable implementation awaiting maintainer review.
Owner: [issue 364](https://github.com/gonz0w/horvium/issues/364), under
[M2 gate 131](https://github.com/gonz0w/horvium/issues/131).
Authority: DASH-REQ-006 / DASH-PERF-001 and the adopted
[resource Test Design](../../design/dashboards/resource-qualification.md).
Foundation: reviewed [request driver](resource-request-driver.md),
[original-inventory HTML predicates](resource-html-predicates.md) and
[offline resource validator](resource-receipt-validator.md).

## Prepared original observations

[Controller helpers](../../../cmd/horvium/dashboard_adapter_helpers_test.go) are
compiled only into tests. `newDashboardPreparedFixture` accepts a declared mode,
a bounded original typed snapshot, and independent owned-dataset witnesses for
native small. It performs no HTTP/API request, native collection or storage work.
The future fixture must obtain these inputs independently outside measured requests;
passing copied response HTML or rebaselined observations is not qualification.

The factory owns a validated clone. Complete sections and nonempty datasets are
required for these four measurement modes; other portable empty/unavailable UI
fixtures retain their separate coverage. Synthetic 100 and 10,000 require the
exact existing deterministic host/GEOM/pool/dataset generator, one pool/four
providers and zero owned native datasets. Projection 10,000 checks every declared
emitter name/GUID/Used/Available/Referenced row, with original host/GEOM/pool facts
supplied independently; it has zero owned datasets. Native small requires at least
100 total datasets and exactly 100 distinct original name/GUID witnesses that
match the supplied snapshot. A typed witness alone does not attest to a real
native command; future wiring must independently count and observe those GUIDs.

Original values are canonicalized in the owned copy, then hashed as SHA-256 of
Go JSON containing typed host facts, GEOM, pools and datasets. Names/GUIDs and
opaque local GEOM identities have deterministic ordering; consumer/identifier
ordering is canonical too. Observation clocks are excluded, since they change
per generation; complete-status requirements remain enforced. Counts, origin and
digest are the only fixture metadata supplied to later receipts. No inventory
names, GUID lists, bodies, cookies or credentials are emitted by the adapter.
This digest format is a development evidence contract requiring review if changed.

The prepared plan and immutable HTML predicate are constructed once. Later caller
mutation cannot change their expectations. Real native accounting can change:
the caller must establish the original observation after the declared warm-up/
idle, preserve it throughout both phases, and fail if values change. This component
never refreshes expectations from a measured response, freezes native collectors,
hides boot pools, adds budgets or retries. The approved constant-inventory fixture
assumption must be checked in actual execution, not inferred from this preparation.

## Actual workload observations and phase JSON

The reviewed driver now retains complete `BodyBytes` and the successful predicate
outcome with each raw sample, after bounded read/close timing has ended. A 200
response receives `original-inventory` only after the supplied predicate passes;
a permitted replacement Overview 409 receives `safe-expiry` only after validation.
Failed bodies/predicates never receive a successful predicate marker. The general
low-level driver still accepts a test callback; the prepared adapter installs the
reviewed original-inventory HTML predicate, with no permissive caller override.

`load` runs the canonical five-case workload through that prepared predicate and
returns a phase receipt plus raw observations, or an error and all available raw
observations. It never emits a passing receipt for an invalid phase, failed
predicate/body, incomplete schedule, all-expiry case or failed initial latency
budget. All scheduled requests still run; failures remain observable rather than
being retried, dropped or converted to a pass.

A phase receipt matches the reviewed Python contract exactly: `samples` contain
`index`, `case`, `outcome`, `status`, integer `duration_ns`, `body_bytes`, `predicate`.
`summary` has `all` and the five case names with scheduled/success/expiry totals,
separate nearest-rank `p50_ns`/`p95_ns`/`p99_ns` objects for success and expiry,
and null expiry latency when no expiry occurs. Raw sample order/values are
preserved; returned receipts own their arrays/maps and do not alias caller samples.
Large initial p95 must remain below 250 ms for every case and aggregate. Small
fixtures acquire no new latency budget; replacement denial timings remain separate.

This is only the phase portion of the full resource contract. The adapter does
not measure warm-up, idle CPU, sampled/high-water RSS, real replacements, native
crash/cap recovery, artifacts/compiler, release provenance or teardown. It does
not emit a whole `DASHBOARD_RESOURCE` record or candidate acceptance. Later reviewed
native/runner wiring must supply all actual observations, preserved controls and
independent host/supervisor bindings required by the strict validator.

## Behavioral validation

[Behavioral tests](../../../cmd/horvium/dashboard_adapter_test.go) cover:

- All four mode preparations, exact synthetic/projection scopes, independent
  owned GUID membership, invalid/missing/duplicate witnesses and original ownership.
- Deterministic digests under reordered rows, valid changed observation clocks
  and opaque GEOM identifiers; original input remains unmodified.
- Exact raw byte/nanosecond preservation, detached serialized arrays, separate
  expiry summaries, malformed body/predicate/sample rejection and hidden per-case
  budget denial.
- Both complete phases through actual authenticated console/observation-manager
  handlers with an incoming in-memory HTTP transport: 2,000 requests, 200 successes
  per case per phase, and zero hidden API reads. These are not network/RSS/CPU results.
- A changed pool-accounting fixture retains all 1,000 raw outcomes and 600 failed
  predicates without emitting a passing receipt. Original values never rebaseline.
- A real manager eviction produces the safe expiry response; its body is injected
  into alternating Overview responses to deterministically exercise 100 expiries
  and 100 successes. This is not a native timing-race or availability measurement.
- Go-generated initial/replacement JSON passed directly to the reviewed independent
  Python `validate_phase` implementation. Pinned Python is required on the portable
  development host; its absence skips that explicit check without a native claim.

Development uses affected command-package tests. Required signed clean review runs
`make check lint test secrets` and affected race checks; a FreeBSD/amd64 test
cross-build checks compilation only. Owned-process status/stop verification remains
required. Invalid timestamp test-fixture attempts remain in private development
logs; their corrected inputs retain valid containing observation intervals.

## Remaining scope and rollback

The runtime, native performance/reader fixtures, runner and historical validator
remain unchanged. No native VM, service, product flag/route/schema, dependency,
Actions workflow or operator storage is introduced. Historical source-bound receipts
remain unchanged. Removing these test-only helpers and sample metadata reverses
the adapter without changing product behavior.

Actual native fixture bindings and complete independent host-runner receipts must
be reviewed before both-release resource execution. Native observations may fail
if the constant-inventory assumption does not hold; retain that failure rather
than disabling a check. #131/#289/#290, lifecycle #305/#139 and historical startup
uncertainty, browser/manual, continuity, hardware/commercial and release gates
remain open. Runtime writes remain disabled.
