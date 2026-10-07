# M2 retained property-job observation assessment

Date: 2026-10-07. Status: qualified subset; maintainer review pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

`PropertyJob.AssessObservation(policy, observation)` joins an immutable retained
property job to a complete independently fresh graph and trusted policy. It returns
two independent facts, `BaselineMatches` and `PostconditionMatches`, alongside
`Restart`, the unchanged last durable event and accepted conservative restart
projection. This is evidence for a future recovery coordinator, with no executor,
startup consumer, native backend, transport or operator workflow.

## Interpretation and refusals

| Baseline | Postcondition | Meaning of the observed graph |
|---|---|---|
| true | false | Matches the retained original relevant review |
| false | true | Matches the reviewed local property effect and preserved context |
| true | true | Already-matching local value; cannot distinguish execution |
| false | false | Matches neither projection or the full policy digest changed |

No row proves causation, absence of earlier effects, executor quiescence, restored
metadata validity or durable job success. Another root actor can produce either
matching graph, including by changing then restoring it. Do not replay, resolve,
release a reservation or overwrite history using these facts. A stored succeeded
or failed label is retained even when today's graph differs; a queued or applying
label never becomes successful merely because the expected effect is present.

Normalize and validate the whole retained binding, preflight bounded inputs and
validate the complete collected graph and explicit policy before exposing a result.
Malformed, incomplete, oversized, inconsistent or unprojectable inputs return an
error and a zero assessment, including no partial history. A well-formed changed
policy digest or differing relevant review returns neither match. Reuse the exact
relevant projection and expected-effect construction from the
[accepted postcondition comparison](property-postconditions.md). Full policy scope/
member binding and relevant pool/incarnation/features/topology/filesystem/state/
version/received/inheritance context remain required. Unrelated sibling rows remain
outside the accepted captured review; no whole-pool preservation claim follows.

Collected false eligibility is accepted for fact comparison and never changes.
`Build`, plan validation, capture and `CompareBaseline` retain their existing
eligibility gates. Assessment restores no clock, session or expired review, writes
no file, appends no event and changes no input or encoded record. Caller-owned
bytes/model labels do not establish root/native provenance; the caller must obtain
fresh evidence independently. No schema, dependency or command is added.

## Portable and native gates

Failing-first behavioral tests cover baseline/expected/neither for nine retained
histories, both matches for an already-local value, zero-result errors, decoded
binding, policy/member and relevant-context drift, input ownership, expired review
history and preserved precondition refusal. A bounded fuzz target checks arbitrary
quota values, exact comparison results, zero-on-error and unchanged inputs/history.

Run the existing disposable `pool-properties` fixture on both declared releases.
Retain its 88 original assertions and eleven postcondition subtests, plus these
eight mandatory assessments beneath
`TestNativePoolPropertyCollection/property-postconditions`:
`job-baseline`, `job-already-matching`, `quota/job-postcondition`,
`refquota/job-postcondition`, `no-limit/job-postcondition`, `job-inherited`,
`wrong-source/job-neither`, `received/job-received`.
Each joins actual false-eligibility native graphs to nine encoded/decoded fixture
histories and verifies restart-page equivalence, unchanged record bytes and retained
pool holders. Each also denies a deliberately incomplete copy with zero result.
The histories are fixture metadata, not acknowledged native execution or a native
persistence/restart qualification. No extra native mutation is required: reuse the
owned fixture-only changes, received stream, independent source checks and original
projection restoration from the accepted postcondition procedure.

```sh
make test-focus PKG=./internal/storage TEST='^Test(RetainedPropertyAssessment|PropertyPostcondition)'
PR_TITLE='feat(storage): assess retained property jobs against observations' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Native writer/eligibility/admission, executor attribution/containment, actual
reconciliation and restored-state clearance, durable outcomes, management parity
and complete M2 acceptance remain open. Existing shared runners/selectors/checkers,
metadata implementations/codecs and historical manifests retain their actual scope.

## Portable selector correction

The first local review refused the portable test names because they also matched
the existing native metadata-suite selector. Rename these tests to preserve that
suite's exact inventory; do not widen the selector or checker or relabel prior
metadata evidence. The corrected missing-target case also uses an actually
complete surviving graph rather than a corrupted captured binding. These changes
affect only the separately compiled storage tests and documentation. The product
library and native fixture remain byte-identical; require exact independently
rebuilt native binary equality before reusing the original 15.1 result.

## Source-bound results

Corrected signed source `2ddc0bb971af47ac6d21f0898deb374cf94ef0d5` passed focused
portable review: `make check lint test secrets`, 122 tooling tests, nine affected
race packages and all 31 selected bounded fuzz targets. FreeBSD 15.1 passed the
88 existing assertions, eleven postcondition subtests and eight mandatory
assessments at original source `81517c35a7ad68125ebd2241c6ec89e932d6e7e8`;
14.5 passed the same inventory at the corrected source. Both executed the exact
same cross-built binary, independently rebuilt at the corrected source; product
library and native fixture bytes are unchanged after the portable-only correction.
Keep each run's actual source: no native rerun or source relabel is claimed.

Native parent/group times were 159.99/54.04 seconds on 15.1 and 228.55/87.30 seconds
on 14.5. These are fixture timings under QEMU TCG, not NAS performance evidence.
Every mandatory assessment checked nine fixture histories and incomplete-copy
refusal, with unchanged encoded records, restart pages and pool holders. Actual
root/default-ACL fixture, received/inherited properties, original projection,
provider/backing/device, guest and process preservation/teardown passed. The earlier
portable selector failure remains unqualified with completed group cleanup.
See the [manifest](property-assessment-manifest.json) for actual source, compiler,
binary, image, log and report hashes. All 49 historical M2 manifests, native
metadata implementations/tests/codecs, shared runners/checkers/selectors, CLI,
modules and Makefile are byte-identical; no 819-case metadata rerun or relabel.
No active or unverified owned sessions remained after qualification.
