# M2 foreground property-job assessment qualification

Date: 2026-10-07. Status: accepted subset in [PR 76](https://github.com/gonz0w/horvium/pull/76).
Authority: [approved PRD](../../milestones/M2/PRD.md),
[EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

Expose the accepted root-leased native assessment through the OS-root foreground
`storage-assess` command. The [runbook](../../operations/M2-property-assessment.md)
defines implemented flags, output and refusal/lifetime semantics. No observation,
policy, native address or executable intent can be supplied by a caller. Keep the
same native source and private ledger safeguards; close the lease before output.
Use the existing status row with explicit fresh baseline/expected-effect flags,
false recovery/executor verification and retained holds. The projection is not a
new durable schema, management transport or recovery capability.

## Behavioral and native gates

Failing-first portable tests require all four comparisons with unchanged history,
private baseline exclusion, zero/wrong/invalid selections, cancellation, unavailable
output, complete 4 KiB staging, short/raw-error output refusal and cancelled
delivery. Arguments fail before native access or reading authentication input.
The existing lifetime watchdog must retain the old 45-second status behavior,
add the 145-second assessment bound and disarm on completion. Unsupported hosts
remain unavailable; they cannot qualify native access.

Reuse the existing checksum-bound `job-status` suite and actual compiled public
main/flag/signal re-exec helper. Its original 21 retained-status assertions and
backing/teardown record must pass unchanged. A new isolated subgroup under
`TestNativeStorageJobStatus/native-property-assessment` must pass every child:
`baseline`, `expected`, `both-terminal`, `neither`, `unknown-job`, `legacy-job`,
`unsafe-record`, `orphan`, `active-writer`, `wrong-uid`, `native-missing`,
`private-baseline-hidden`, `signal-cancelled`, `preserved-and-released`,
`fixture-teardown`. Require the group itself and every child once, with no failed,
skipped or missing cases. The shared checker retains its 21-case scope; the new
source-bound manifest additionally checks all fifteen actual PASS results.

The new subgroup creates two exclusive sparse backing files under its private
root temporary directory, attaches only those vnode md providers, and creates an
unmounted uniquely named mirror with `cachefile=none`. It captures actual fresh
native observations; a synthetic eligibility label is used only to construct test
intents, never product admission. Directly provision applying and cancelled test
job records plus explicit policy. These bytes are not native execution or durable
publication acknowledgements. Quota/refquota changes affect only the owned fixture;
restore them and independently compare the complete raw native projection.

The command processes independently open and release the real private lease.
Actual kernel UID 65534, an active writer, missing native pool address, legacy job,
unsafe mode, orphan and missing selection must return no document and only the
fixed failure class. Preserve bytes/lock inode and both nonterminal/terminal holds.
SIGTERM with a full owned output pipe must exit 1 within three seconds; it may
interrupt collection or delivery, and is not proof it reached the delivery phase.
Wait/reap every child. Verify selected pool GUID before destruction and each md
provider's exact backing before detach, then verify absence and remove all files.
The old parent owns the copied public executable and retained-status fixtures.
Every guest/base/runtime/process/host preservation and teardown gate remains
required. The parent's existing two-minute context remains unchanged.

```sh
make test-focus PKG=./cmd/horvium TEST='^TestStorage(PropertyAssessment|JobStatus)'
PR_TITLE='feat(cli): assess retained property jobs against native state' make review
.tools/venv/bin/python3 tools/local-qualification.py job-status 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py job-status 14.5 --run
make lab-status
```

No shared runner, selector, checker, supervisor, Makefile or dependency changes
are needed. All existing storage/native source implementation and 51 prior M2
manifests remain byte-identical. Their tested revisions/scopes are preserved;
the old 819-state and 88-pool suites are not rerun or relabeled. Qualification of
this command adds no write eligibility, executor attribution, recovery resolution,
reservation release, durable outcome or hardware/client support.

## Fixture timestamp correction

The first 15.1 run failed before command assessment cases: a directly provisioned
queued event rounded the nanosecond capture time down to milliseconds, violating
the existing rule that admission cannot precede capture. The pool/providers and
all host artifacts were cleaned; this failed run is not qualification. Move the
test-only constructor into portable tests and reproduce the failure with a fixed
fractional capture timestamp. Queue at the following millisecond and append the
applying/cancelled event one millisecond later. Product/source validation, native
assertions and every time limit remain unchanged. Preserve the failed record.

## Source-bound results

Signed source `b8d078d0097ae47f2e9d5f9b035fccbd2f144e9c` passed focused portable
review: `make check lint test secrets`, 122 tooling tests and race checks for
the changed command package. That package has no compiler-listed portable fuzz
target; unrelated prior fuzz evidence was not rerun or relabeled. Both
disposable releases independently built the exact same test binary
`ed27a9ed1434149b1e35bea0330526010cdd9e555b38790a1d435bea30cb428d` and passed
all 21 original status assertions plus fifteen mandatory native assessment
command cases.

- FreeBSD 15.1: native parent 14.24 seconds; assessment subgroup 11.20 seconds.
- FreeBSD 14.5: native parent 12.80 seconds; assessment subgroup 10.15 seconds.

All owned mirror/provider/ledger/executable fixtures were removed, and base/runtime
and process/host teardown were verified. No active or unverified owned session
remained. The [manifest](property-assessment-diagnostic-manifest.json) retains actual
source/binary/tool/log/report hashes and the earlier unqualified timestamp fixture
with complete cleanup. Every time limit remains unchanged. All 51 prior M2
manifests, existing storage/native source code/tests/codecs and shared tools remain
byte-identical; the historical 819-state and 88-pool evidence keeps its old scope.
No service, runner or GitHub Actions was enabled. Runtime writes, resolution and
complete recovery/UI/CLI/M2/hardware/client acceptance remain open.
