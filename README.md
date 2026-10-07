# Horvium

**Dependable storage on FreeBSD.**

Horvium is a dedicated NAS management system for FreeBSD and ZFS, designed to
remain lightweight on a 16 GB home server and grow to larger commercial storage
systems. The HP MicroServer Gen8 is a reference configuration, not a hardware
requirement or a limit on disks, memory, pools, or datasets.

**Status:** M0 foundation and the main M1 read-only discovery/management slices
and crash containment, native resources, HTTP abuse recovery, independent SMB/NFS
continuity and automated Firefox qualification are accepted. The consolidated
qualification record was accepted in PR 18. M1 is Validating; Safari/manual
accessibility acceptance remains pending. See the
[M1 qualification review](docs/testing/M1/qualification-status.md) for evidence and
remaining gates. There is no installable port, appliance image or production
support claim.

M2 storage lifecycle design is approved in [PR 20](https://github.com/gonz0w/horvium/pull/20): [PRD](docs/milestones/M2/PRD.md),
[EDD](docs/milestones/M2/EDD.md), [Test Design](docs/milestones/M2/TDD.md) and
[implementation sequence](docs/milestones/M2/implementation.md). The first
model subset was accepted in PR 21; bounded property-source parsing passed on
both disposable FreeBSD targets and was accepted in PR 22. Dataset identity/type
binding and version drift were accepted in PR 23. Filesystem mount/encryption
constraints were accepted in PR 24. Pool identity/incarnation and policy
observations were accepted in PR 25. Complete native feature observations and
exact common profile checks were accepted in PR 26. Native vdev membership,
layout refusal and topology drift checks were accepted in
[PR 27](https://github.com/gonz0w/horvium/pull/27). See the
[topology qualification](docs/testing/M2/topology.md).
The [virtual device fixture](docs/testing/M2/devices.md) passed its eight native
identity/geometry assertions and complete teardown on both targets;
[PR 28](https://github.com/gonz0w/horvium/pull/28) accepted the fixture.
The [native disk correlator](docs/testing/M2/disk-correlation.md) passed all twelve
assertions and complete teardown on both targets and was accepted in
[PR 31](https://github.com/gonz0w/horvium/pull/31). The
[attachment classifier](docs/testing/M2/disk-attachments.md) distinguishes exact
DEV/disk-ident correspondence and refuses PART/unsupported attachments; native
partition-denial qualification passed all seventeen assertions and full teardown
on both targets and was accepted in [PR 32](https://github.com/gonz0w/horvium/pull/32).
The [system device-use subset](docs/testing/M2/system-device-use.md) maps
root, mount and swap observations to their direct GEOM disks; native UFS/swap
qualification passed all twenty-three assertions and complete teardown on both
targets and was accepted in [PR 33](https://github.com/gonz0w/horvium/pull/33).
The [imported-pool use subset](docs/testing/M2/imported-pool-use.md) maps
all imported pools to their direct GEOM disks, including pools with no mounted
filesystems; native qualification passed all twenty-nine assertions and complete
teardown on both targets and was accepted in [PR 34](https://github.com/gonz0w/horvium/pull/34).
The [filesystem signature subset](docs/testing/M2/filesystem-signatures.md)
binds positive base-probe UFS/ZFS observations to freshly correlated disks;
native qualification passed all thirty-six assertions and complete teardown on
both targets and was accepted in [PR 35](https://github.com/gonz0w/horvium/pull/35).
The [combined device-use subset](docs/testing/M2/device-use-review.md) retains
selected positive use facts in an immutable baseline and requires new review on
identity/geometry/mount/swap/pool/signature drift. All fifty-one assertions and full
teardown passed on both targets; PR 36 accepted this subset.
The accepted [pool-label subset](docs/testing/M2/pool-labels.md) binds positive exported
pool identities and recorded state to the same disk; all fifty-nine assertions
and full teardown passed on both targets; accepted in PR 37.
The accepted [disk-access subset](docs/testing/M2/disk-access.md) binds aggregate
GEOM access counters to disk identity and invalidates review on access drift;
all sixty-eight native assertions and full preservation/teardown passed on both
targets; accepted in PR 38.
The accepted [state-directory subset](docs/testing/M2/state-directory.md) establishes
the root-owned single-writer boundary for future durable metadata. All thirty-five
native assertions and complete preservation/teardown passed on both targets;
accepted in PR 39.
The accepted [ownership record codec](docs/testing/M2/ownership-records.md) validates
bounded versioned metadata and checksums without granting enrollment or writing
files. Five behavioral tests, fifty-seven subtests and all twenty-one bounded fuzz
targets passed; accepted in PR 40.
The [native ownership reader](docs/testing/M2/ownership-read.md) binds those
records to safe opened file objects and the held state directory. All sixty-three
lease/read assertions and complete teardown passed on both targets; accepted in
PR 41. Default-ACL disposable fixtures define the qualified scope.
The accepted [state ACL validation](docs/testing/M2/state-acl.md) checks actual opened
objects before trusting mode bits. All 108 expanded assertions and complete
teardown passed on both targets; accepted in PR 42.
The [ownership publication prerequisite](docs/testing/M2/ownership-publication.md)
adds private metadata publication and explicit uncertainty after failed durability.
All 161 assertions and complete teardown passed on both targets; accepted in PR 43.
The [bounded ownership catalog](docs/testing/M2/ownership-catalog.md) passed all 202 assertions
and complete teardown on both targets; accepted in PR 46. The
[ownership capacity guard](docs/testing/M2/ownership-capacity.md) passed all 234 assertions
and complete teardown on both targets; accepted in PR 47. The
[job lifecycle codec](docs/testing/M2/job-records.md) passed portable qualification;
accepted in PR 48. The [native job reader](docs/testing/M2/job-read.md) and
immutable successor guard passed all 275 assertions and full teardown on both
targets; accepted in PR 49. The [mixed state catalog](docs/testing/M2/state-catalog.md)
passed all 336 state assertions and complete teardown on both targets;
accepted in PR 50. The [durable job publisher](docs/testing/M2/job-publication.md)
passed all 432 state assertions and full teardown on both targets; accepted in
PR 51. The [reviewed property intent](docs/testing/M2/property-intent.md) capture/codec
passed portable qualification; accepted in PR 52. The
[property job binding](docs/testing/M2/property-jobs.md) combines that baseline
with immutable lifecycle metadata and passed portable qualification; accepted in
PR 53. The [native property-job store](docs/testing/M2/property-job-storage.md)
passed all 555 state assertions and complete teardown on both targets; accepted
in PR 54. The [job reservation checks](docs/testing/M2/job-reservations.md) add
bounded duplicate/plan/pool metadata checks; accepted in PR 55. The
[native reservation inspector](docs/testing/M2/native-job-reservations.md) connects
that view to the held root state lease; both targets passed all 622 assertions
and full teardown; accepted in PR 56. Runtime admission
remains disabled.
Runtime enrollment and full store/recovery gates remain open.
Other native gates, runtime job admission and storage writes remain pending; the runtime is
still read-only.

The [reserved submission publisher](docs/testing/M2/property-submission.md)
combines complete retry/pool checks and initial record publication under one
writer lock. Both targets passed all 710 assertions and teardown; accepted in
[PR 57](https://github.com/gonz0w/horvium/pull/57). Executable admission remains disabled.

The [bounded restart projection](docs/testing/M2/job-restart.md) passed portable qualification and was accepted in
[PR 58](https://github.com/gonz0w/horvium/pull/58); it presents interrupted work as requiring reconciliation while retaining durable
history and pool holds. Actual native recovery remains pending.

The [metadata process-death qualification](docs/testing/M2/property-publication-cuts.md)
passed all 777 state/backing assertions on both native targets. It tests interrupted
publication and conservative restart metadata on disposable fixtures; runtime writes remain disabled.

The [property review/submission coordinator](docs/testing/M2/property-coordinator.md)
connects volatile reviews, fresh baseline checks, single claims and immutable
metadata acknowledgements through a trusted backend contract. Portable baseline,
storage race/all 21 fuzz checks and a FreeBSD cross-build passed; no native backend
or runtime storage writer is enabled.

The [native property observer](docs/testing/M2/native-property-collection.md)
assembles fixed FreeBSD projections in two bounded passes and refuses ordinary
drift. All 16 collector assertions passed on both native targets, with complete
portable review and teardown; successful collection still grants no write
eligibility or runtime submission authority.

The [root policy/native binding](docs/testing/M2/property-policy-binding.md)
derives pool addresses from the held root metadata reader, refuses policy drift
and checks every explicit scope against fresh native observations. Both native
targets passed all 22 assertions, complete portable review and teardown;
successful binding still grants no write eligibility.

## Agreed direction

- FreeBSD and ZFS; BSD-3-Clause license.
- SMB, NFS, and SFTP file serving; local users/groups initially.
- Snapshots, backup-peer registration, incremental replication, retention, and
  tested restoration in the first release.
- Native FreeBSD port/package first; an appliance image later.
- Go, server-rendered HTML5, hand-authored CSS, and HTMX when needed.
- BSD Classic interface, following the BSD Cloud design guidance.
- File serving continues when the management interface is unavailable.
- Fleet management, automatic failover, AD, VMs, and containers are outside the
  first release. Horvium itself does not host application workloads.

## Start here

1. [Project charter](docs/project/charter.md): users, outcomes, and boundaries.
2. [Architecture baseline](docs/architecture/baseline.md): components and state.
3. [Roadmap](docs/roadmap/milestones.md): foundation through the first release.
4. [Bootstrap status](docs/engineering/bootstrap-status.md): evidence and open gates.
5. [Contributing](CONTRIBUTING.md) and [agent instructions](AGENTS.md).
6. [M1 requirements](docs/milestones/M1/PRD.md),
   [engineering design](docs/milestones/M1/EDD.md), and
   [Test Design](docs/milestones/M1/TDD.md): approved implementation contracts.
7. [M1 implementation status](docs/milestones/M1/implementation.md): completed
   portions, evidence and remaining acceptance gates.

## Foundation checks

Development-only tools are checksum-pinned and installed under ignored `.tools/`.
Use Python 3.14.7 on a Linux/macOS development host for this tool installation.
Python is not a planned runtime dependency of the NAS.

```sh
make tools
make hooks
# Commit signed changes first; use the actual proposed PR title.
PR_TITLE='docs: describe the proposed change' make review
# Fresh complete portable qualification at milestones/releases:
# PR_TITLE='type: actual proposed PR title' make review-full
make lab-status
```

Native FreeBSD checks and physical hardware tests have separate gates. Passing
these foundation checks does not demonstrate functioning storage or file shares.
Use the [local qualification runner](docs/testing/local-qualification.md) for
supported disposable FreeBSD suites on the development Mac; see
[CI cost controls](docs/engineering/ci-costs.md) for the zero Actions policy. The [focused testing workflow](docs/engineering/local-review.md) keeps everyday
review proportional and reserves fresh complete portable runs for milestones,
releases and broad changes. All checks run locally; no persistent runner is installed. `make lab-stop` stops supervised development sessions and verifies
teardown. Maintainers inspect source-bound evidence before merging.

The Go module provides discovery, cache, administration, console and authenticated
CLI adapters. `make build` creates a cgo-free foreground development executable;
see the [M1 development runbook](docs/operations/M1-development.md). `make tools` also
installs checksum-verified Go 1.27.1 under `.tools/`; it leaves the system toolchain
alone. See [CI/CD](docs/engineering/ci-cd.md) for opt-in disposable native tests.

See [documentation index](docs/README.md), [testing strategy](docs/testing/strategy.md),
and [ports plan](packaging/ports/README.md). The selected remote is the private
`gonz0w/horvium` repository. See bootstrap status for creation and CI evidence.

## License and provenance

[BSD-3-Clause](LICENSE). Adapted BSD Cloud foundation tooling retains upstream
attribution; see [NOTICE](NOTICE.md).
