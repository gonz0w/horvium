# Spike: portable FreeBSD discovery and Unix peer identity

Status: Running. Owner: Cam (`@gonz0w`). Date: 2026-10-03.
Time box: one research session; stop after three repair attempts per failing probe.
Authorization: maintainer requested QEMU for native experiments on 2026-10-03.
Related work: [M1 issue](https://github.com/gonz0w/horvium/issues/1).

## Questions and procedure

Can both candidate releases expose numeric ZFS inventory, GEOM geometry, and
kernel-authenticated Unix socket identity without production implementation?

The manual [workflow](../../../.github/workflows/bsd-smoke.yml) runs isolated
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

Native result and source commit are pending. Record them here after the run.
Source/version pins are in [the manifest](source-manifest.json).

The C program is an ABI/policy experiment, not a production helper. The Go program
is a narrow XML experiment, not complete hardware discovery. This does not test
physical disk identities, SMART, controller failure, client protocols, production
authorization, socket deployment, memory budgets, or NAS recovery. No product
service is installed or enabled. Spike code requires ordinary design/tests/review
before any production reuse.
