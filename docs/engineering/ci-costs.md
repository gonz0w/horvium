# CI cost controls

Date: 2026-10-05. Goal: reduce hosted compute while retaining independent required
checks and complete local/native acceptance evidence.

## Audit

187 Horvium runs were inspected from 2026-10-03 through 2026-10-05. Summing each
executed job's start/completion duration, rounded up to a minute, gives this
estimate (unstarted/skipped jobs and free Dependabot jobs excluded):

| Work | Rounded minutes |
|---|---:|
| Native FreeBSD qualification, including performance | 842 |
| Repository hygiene and Go checks | 338 |
| PR title and DCO | 176 |
| Total | 1,356 |

At the published Linux rate of $0.006/minute, this is approximately $8.14 before
the account's included allowance; October 4 Denver time accounts for roughly
$7.10. Job timestamps are an estimate, not an invoice. The billing API was
unavailable to the current credential scope, so the reported $10/day remains
unreconciled and may include other account usage. Do not enlarge credential
scope or change billing settings merely to run repository checks.

No uploaded artifacts were present. Current caches held approximately 9.46 GiB,
mostly upstream VM-action image copies across branches. Avoid adding hosted
image caches near the existing 10 GiB allowance. Logs/job summaries are not
artifact storage. Dependabot service jobs are free; ordinary CI triggered by
a dependency PR still follows repository workflow policy.

Sources checked on this date: [runner pricing and per-job rounding](https://docs.github.com/en/billing/reference/actions-runner-pricing),
[allowances, storage and self-hosted billing](https://docs.github.com/en/billing/concepts/product-billing/github-actions).

## Automatic review gate

Keep the existing `repo-hygiene`, `pr-title`, and `dco` job names and protections.
Never path-skip the entire required workflow. Repository content/link/config
checks, tool lint/tests and secrets checks run on every PR update and main push.
Only an allowlisted change consisting entirely of root policy/prose files or
`docs/*.md` recursively omits Go installation, formatting, vet and unit tests.
Unknown paths, tooling/workflow/build changes, mixed changes, code deletion or
rename, and missing/empty comparisons run Go checks. Diff failures fail the job.

Routine hosted CI does not repeat race/fuzz checks. Run `make go-check` locally
before review and name the tested source in the PR. A manually dispatched
repository workflow with `thorough=true` is the paid fallback; it runs the deep
gate without repeating vet/unit tests within that job. Main retains the cheap
integration check after a merge. No required check is replaced by an unverified
success status or removed from the ruleset.

## Native suite selection

The BSD dispatcher now accepts **one `suite` choice**. Default `none` starts no
VM. `properties`, `discovery`, `research` and `continuity` each select two release
guests; `foundation` selects one guest; `performance` selects two long-running
guests; `browser` selects only the existing Linux Firefox job. `full` explicitly
selects all suites, with continuity included in discovery guests. It is reserved
for assembled milestone/release review, not every narrow iteration.

The previous boolean `browser`/`continuity`/`performance` dispatcher inputs are
replaced by this choice. Historical execution records keep their original
inputs; new requests use, for example:

```sh
gh workflow run bsd-smoke.yml --ref YOUR_REVIEW_BRANCH -f suite=properties
```

Prefer [local qualification](../testing/local-qualification.md) on the Mac for
supported suites. Existing standalone manual workflows remain available as paid
fallbacks. Do not dispatch broad qualification while iterating on logs/docs.
Historical research should run only when its contracts/toolchain change or when
explicitly reviewing the complete matrix.

The planning target is 70–80% fewer hosted minutes at a similar workload, driven
by local native/deep checks and narrow selection. It is not a measured saving or
a promise about the account's bill. Review the next billing period after the
change is merged; no paid qualification is dispatched solely to measure savings.
