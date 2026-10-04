# Compatibility matrix

**No production support is claimed.** Architecture targets and observed
validation are separate. Refresh exact versions in each active research manifest.

| Area | Target | Evidence/status |
|---|---|---|
| Runtime | Approved M1 FreeBSD 15.1 primary / 14.5 compatibility | Native reader/helper fixture tests passed; full NAS runtime pending |
| Architecture | amd64 first candidate; other FreeBSD architectures as validated | Hardware-neutral design, no universal support claim |
| Storage | ZFS pools/datasets on supported direct-access devices | Planned; no pool modified by M0 |
| File clients | macOS and Linux/BSD first; SMB/NFS/SFTP as designed in M3 | Priorities confirmed; version/permission matrix and actual tests pending |
| Backup peer | Compatible ZFS receiver over authenticated SSH | Planned; feature negotiation and restoration pending |
| Management | Go 1.27.1 / HTMX 2.0.11 proposed; Safari/macOS and Firefox/Linux first | No product UI implemented; browser versions/tests pending |
| Distribution | FreeBSD port/package, then image | Packaging plan only; not in official Ports |
| Small hardware | 16 GB MicroServer Gen8 / four 3 TB SATA disks | Reference profile supplied by maintainer, unmeasured |
| Larger hardware | Representative commercial FreeBSD deployments | Scale budgets and lab evidence pending |
| Foundation smoke | FreeBSD 15.1 amd64 | Shell/hooks passed; not native NAS evidence |
| Native research | FreeBSD 15.1-RELEASE-p3 / 14.5-RELEASE amd64; Go 1.27.1 | Narrow TSV/GEOM/credential fixture assertions passed; see spike |
| Product discovery reader | Same two observed release/patch levels, Go 1.27.1 amd64 | Actual Go reader/parser and isolated pool fixture passed; see product evidence |
| Product helper | Same two observed release/patch levels, Go 1.27.1 + x/sys v0.48.0 amd64 | Actual kernel UID, socket ownership/lifecycle, admission/timeout and reader-over-transport tests passed |
| Collector/cache | Hardware-neutral Go library; FreeBSD helper integration | Local generation/ownership/concurrency/boundary tests and actual unprivileged native helper outage/recovery passed |
| Tooling | Python 3.14.7 + checksum-pinned tools on Linux/macOS | See bootstrap status for checks actually run |

Evidence levels: D0 designed; D1 unit/component verified; D2 native/client verified;
D3 workflow/restore verified; D4 release-qualified with failure, upgrade, security,
and performance gates. Mark unavailable/skipped/failed separately from passed.
Commercial deployment size alone is not a D4 claim.

See [M1 native spike](../research/M1/native-spike.md) for source/run IDs, bundled
ZFS version strings and fixture limits, and [M1 Test Design](../milestones/M1/TDD.md)
for product tests still required and [product reader evidence](../testing/M1/discovery.md)
for the completed reader slice and [helper evidence](../testing/M1/helper.md)
for actual Go transport tests; [collector evidence](../testing/M1/collector.md)
records the unprivileged cache/outage/recovery path. The earlier C spike is ABI
evidence only. No
physical identity, file-client, scale or throughput claim follows.
