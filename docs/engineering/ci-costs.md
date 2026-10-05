# CI cost controls

Date: 2026-10-05. Goal: eliminate repository GitHub Actions execution and keep
complete local/native review evidence.

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

## Zero Actions policy

The maintainer approved moving all checks to local execution. Delete runnable
workflows, disable Actions at repository level, remove only Actions-required
status checks from the live main ruleset, and retain signed commits, PRs, linear
history, resolved conversations, deletion/force-push protection and no bypasses.
Release-tag protection is unchanged. Dependabot Go/pip update proposals remain;
the obsolete Actions dependency ecosystem is removed. No self-hosted runner is
registered and no success statuses are posted to simulate independent CI.

Use [the complete local review gate](local-review.md) and
[explicit native qualification](../testing/local-qualification.md). GitHub no
longer independently verifies check results: maintainers inspect source-bound
portable/native evidence before merging. Historical hosted run links remain
provenance, not instructions to start more paid jobs.

Repository Actions disabling stops new workflow execution, including workflows
still present on main/older branches while this PR awaits review. Removing files
and the local policy regression prevents accidental reintroduction. Existing
charges and other repositories' usage are unaffected; account billing totals are
not inferred from the repository setting. No paid run is dispatched to measure
savings. The prior 70–80% planning estimate is superseded by zero repository
workflow execution; no account-wide dollar saving is claimed.

## Infrastructure still needed

The Mac handles portable checks and the already executed foundation/property
FreeBSD guests. Nothing persistent needs to be installed. The PR 28 device
fixture uses Linux process identity/device attachment and still needs a Linux
host or a reviewed Mac adaptation before native evidence can pass. Pinned
Firefox also needs Linux, and emulated Mac performance is not a replacement for
KVM/physical qualification.

A later disposable x86-64 Linux VM with four vCPUs, 16 GB RAM, approximately 40 GB
scratch and virtualization/KVM access is a useful starting lab profile, not a
measured hardware requirement. Run local scripts there over operator-controlled
access; an Actions runner is unnecessary. Replication/restore needs a second
isolated FreeBSD peer when implemented; physical power-loss/controller evidence
needs hardware. Those gates remain open, rather than moved into misleading
portable checks.

Every foreground lab/review/preview process is supervised. Use `make lab-status`
and `make lab-stop`; read [teardown and its limits](../testing/local-qualification.md#teardown).
Teardown removes owned live guest images, variable stores and fixture keys.
Checksum caches and private evidence remain inert files, with no running service.
