# Native retained-job history refusal

Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[M2-RUNTIME-002](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted retained-policy Test Design](../../milestones/M2/retained-policy-startup.md),
[M2 PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md) and [ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirement IDs: M2-RUNTIME-002, M2-STATE-001, M2-OWN-001 and M2-RECOVERY-001.

## Bounded outcome and seed construction

Extend the existing denial-only native storage fixture to complete mixed
ownership/job histories. Use its original pre-existing journal/ledger leases,
strict ownership publishers and both accepted job publishers, publishing every
history prefix. Observe the complete native census before initializing the
original journal and publishing its strict controlled closed successor. No
record is written as raw JSON and no acknowledgement is reconstructed.

This is controlled metadata for a refusal test, not commissioned runtime
ownership, an admitted worker, real execution provenance or clean shutdown after
work. It extends the accepted denial-only job seed, not the positive commissioned
policy-only seed. UFS without softupdates and ZFS files, leases, census and sync
are actual native facilities. Policy/physical-use/raw-boot transfers remain
synthetic, as in the existing storage gate; fixed-root composition is separate.

Each of three cases contains all seven valid job states in both accepted formats,
fourteen unique jobs/review/submission identities and eight nonterminal/six
terminal records. Same-pool uses ownership/job pool 100. Other-pool owns 200 but
retains jobs on 100. Mixed owns 100 and 200, with seven jobs on each pool. Strict
codec metadata and complete native reservation projections must agree.

## Inventory declared before execution

Keep the existing selector
`^TestNative(ClosedStartup(Publication|Retirement)(UFS|ZFS|Cuts)|PolicyStartup(UFS|ZFS))$`,
eight parents, eighteen-minute process bound and all 544 retained controls.
Each existing policy parent retains all 139 controls and adds 24:
`history-{same-pool,other-pool,mixed}_{catalog,leases-held,denied,preserved,holds,released,reopened,retry}`.
The complete gate requires 870 controls, including 326 policy controls.

| Control suffix | Observable assertion |
|---|---|
| `catalog` | The original ledger's native inspection matches every encoded ownership/job summary, total bytes and eight nonterminal/six terminal records. |
| `leases-held` | Competing actual openers fail for both domains during the startup boot callback; that callback is reached. |
| `denied` | Original policy review returns `ErrDenied`, no review/native handle, and never invokes a policy producer. |
| `preserved` | Independent journal/ledger namespace snapshots retain exact objects and bytes after refusal. |
| `holds` | Actual native reservation inspection retains every original pool/job holder. |
| `released` | Both original leases are independently reacquirable after denial. |
| `reopened` | A new diagnostic pair reads the same complete catalog and holder projection, then closes both leases. |
| `retry` | Repaired context/new review still refuses without policy collection, mutation or hold release. |

Pin the expanded inventory in a failing-first evidence-consumer regression;
old-only, missing, failed or duplicated history controls must not qualify.
That parser red is not a reproduced product defect. Product implementation is
unchanged; native cases extend evidence for already implemented all-job refusal.
Qualification remains pending until the entire checksum-bound gate completes on
both pinned FreeBSD/amd64 15.1 and 14.5 targets with independent owned teardown.

## Acceptance limits

Combined histories prove refusal of these complete retained namespaces, not each
possible sparse history, valid multi-policy startup/last-policy failure,
policy-source freshness, configured startup, worker draining or native launch.
Remaining adopted concurrency, census/object and context/capacity families and
fixed-root multiple-policy/mixed-history acceptance remain open under issue 305.
Existing manifests retain their original candidate sources and limitations.
