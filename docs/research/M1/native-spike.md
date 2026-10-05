# Spike: portable FreeBSD discovery and Unix peer identity

Status: Completed; evidence only. Owner: Cam (`@gonz0w`). Date: 2026-10-03.
Time box: one research session; stop after three repair attempts per failing probe.
Authorization: maintainer requested QEMU for native experiments on 2026-10-03.
Related work: [M1 issue](https://github.com/gonz0w/horvium/issues/1).

## Questions and procedure

Can both candidate releases expose numeric ZFS inventory, GEOM geometry, and
kernel-authenticated Unix socket identity without production implementation?

The original manual [historical workflow](https://github.com/gonz0w/horvium/blob/4eb60c868aa90ea72c11884c8c606d734ea2fe2f/.github/workflows/bsd-smoke.yml) runs isolated
QEMU amd64 guests for FreeBSD 15.1 and 14.5. The pinned VM Action provisions the
guest; it is test infrastructure, not Horvium runtime infrastructure. Capture
actual kernel/userland and ZFS versions from logs, separately from source snapshots.
Go 1.27.1 is downloaded from its official distribution and checksum-verified.

The [script](../../../tools/research/m1-native.sh) requires explicit disposable-VM
opt-in, FreeBSD, amd64, and root. It creates a fresh 512 MiB file-backed md device,
checks its backing filename, and creates a uniquely named pool without force.
No operator-supplied disk, pool, or dataset arguments are accepted. The only
mutation fixtures are that pool and a dataset with mountpoint `none`. Teardown
destroys the owned pool, detaches the owned md unit, and removes temporary files;
cleanup failure fails the experiment and requires discarding the guest.

Assertions:

- Fixed `zpool list`/`zfs list` TSV projections have exact field counts, literal
  numeric values, and the expected fixture rows/health.
- Go's XML decoder locates the generated GEOM provider and validates its geometry.
- Discovery preserves dataset names/GUIDs; GUID survives a fixture dataset rename.
  Asynchronous space accounting is not incorrectly treated as immutable.
- Base C `LOCAL_PEERCRED` observes root and an unprivileged client after identity
  drop and a real connect. A forged root UID in the payload cannot override kernel
  identity. The toy root-only policy allows root and denies nobody.

## Evidence and limits

The corrected [native run](https://github.com/gonz0w/horvium/actions/runs/37162318433)
passed both research jobs and foundation smoke on
`411ca08d99544173a685422957f7a41e3eeaa4aa`, on 2026-10-03. All assertions listed
above passed, including the generated backing-file check. Successful job exit
includes successful pool destruction and md detachment in the cleanup trap.

| Guest | Installed/running kernel and userland | ZFS userland / module | Toolchain |
|---|---|---|---|
| amd64 15.1 | 15.1-RELEASE-p3 for all three probes | 2.4.2-1-FreeBSD_g6330a45b0 for both | go1.27.1 freebsd/amd64 |
| amd64 14.5 | 14.5-RELEASE for all three probes | 2.2.9-FreeBSD_g079ba86d7 for both | go1.27.1 freebsd/amd64 |

The [first run](https://github.com/gonz0w/horvium/actions/runs/37161902196) on
`4eb60c868aa90ea72c11884c8c606d734ea2fe2f` passed 14.5 but failed 15.1's fixture
backing-file assertion before discovery. Source inspection showed verbose
mdconfig has additional trailing fields in 15.1. The repair uses the exact
backing-file query (`mdconfig -l -n -f <generated-file>`) and validates its sole
numeric unit against the unit returned
at creation. One repair attempt was needed. A redundant dispatch of the original
revision (run 37162246407) was cancelled; it is not acceptance evidence.

Source/version pins are in [the manifest](source-manifest.json).

Environment: Ubuntu 24.04 GitHub-hosted runner; FreeBSD amd64 guests from the pinned
VM Action's image selection/cache, fresh 512 MiB file-backed md fixture; default
Action CPU/memory configuration. Image digests and host hardware were not captured
in this contract experiment. It does not establish an immutable release-image or
performance baseline. Product qualification must capture image checksums and
explicit CPU/memory alongside the actual updated OS patch level.

The C program is an ABI/policy experiment, not a production helper. The Go program
is a narrow XML experiment, not complete hardware discovery. This does not test
physical disk identities, SMART, controller failure, client protocols, production
authorization, socket deployment, memory budgets, or NAS recovery. No product
service is installed or enabled. Spike code requires ordinary design/tests/review
before any production reuse.

Current execution uses the [local qualification runner](../../testing/local-qualification.md);
GitHub Actions are disabled. Historical source/run links above retain provenance.
