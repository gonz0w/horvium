# Compatibility matrix

**No production support is claimed.** Architecture targets and observed
validation are separate. Refresh exact versions in each active research manifest.

| Area | Target | Evidence/status |
|---|---|---|
| Runtime | Approved M1 FreeBSD 15.1 primary / 14.5 compatibility | Native reader/helper/cache/admin fixture tests passed; foreground runtime/console/CLI owned-fixture tests passed on both releases; full NAS runtime pending |
| Architecture | amd64 first candidate; other FreeBSD architectures as validated | Hardware-neutral design, no universal support claim |
| Storage | ZFS pools/datasets on supported direct-access devices | Planned; no pool modified by M0 |
| File clients | macOS and Linux/BSD first; SMB/NFS/SFTP as designed in M3 | Priorities confirmed; version/permission matrix and actual tests pending |
| Backup peer | Compatible ZFS receiver over authenticated SSH | Planned; feature negotiation and restoration pending |
| Management | Go 1.27.1 / HTMX 2.0.11 proposed; Safari/macOS and Firefox/Linux first | Read-only semantic console and authenticated CLI implemented in accepted slice 5; required Safari/Firefox versions/tests pending |
| Distribution | FreeBSD port/package, then image | Packaging plan only; not in official Ports |
| Small hardware | 16 GB MicroServer Gen8 / four 3 TB SATA disks | Reference profile supplied by maintainer, unmeasured |
| Larger hardware | Representative commercial FreeBSD deployments | Synthetic management budgets passed; physical/commercial lab evidence pending |
| Foundation smoke | FreeBSD 15.1 amd64 | Shell/hooks passed; not native NAS evidence |
| Native research | FreeBSD 15.1-RELEASE-p3 / 14.5-RELEASE amd64; Go 1.27.1 | Narrow TSV/GEOM/credential fixture assertions passed; see spike |
| Product discovery reader | Same two observed release/patch levels, Go 1.27.1 amd64 | Actual Go reader/parser and isolated pool fixture passed; see product evidence |
| Product helper | Same two observed release/patch levels, Go 1.27.1 + x/sys v0.48.0 amd64 | Actual kernel UID, socket ownership/lifecycle, admission/timeout and reader-over-transport tests passed |
| Collector/cache | Hardware-neutral Go library; FreeBSD helper integration | Local generation/ownership/concurrency/boundary tests and actual unprivileged native helper outage/recovery passed |
| Administration security | Hardware-neutral auth/config parser; FreeBSD descriptor ownership and loopback HTTP | Local denial/expiry/capacity tests and native credential ownership/atomicity/dropped-UID real HTTP fixtures passed on both releases; UI/CLI and deployable auth pending |
| Console/CLI | Go 1.27.1 with locally embedded HTMX 2.0.11 | Local auth/parity/escaping/strict-query/expiry tests pass; limited in-app browser login/filter/keyboard/refresh/logout and 390-pixel layout verified; actual root-helper/unprivileged foreground runtime/CLI/HTML with owned GUID/property/teardown assertions passed on both releases |
| Runtime lifecycle / budgets | FreeBSD foreground processes; hardware-neutral response admission/logging | Local admission/logging/lifecycle tests pass; actual built-process kill/restart/outage/recovery and independent fixture-only SFTP reads passed on both FreeBSD targets; 100/10,000-row handler diagnostics recorded; synthetic combined RSS/CPU/latency thresholds passed on both targets; native-reader resource and SMB/NFS continuity acceptance pending |
| Performance qualification | Checksum-pinned FreeBSD 15.1/14.5 QEMU images; qemu64, two vCPUs, 6144 MiB | Actual suite and evidence validation passed on both targets (run 37218753397); small idle CPU 0.020–0.023%, large p95 66–90 ms and combined RSS 43–44 MiB; native-reader resource/hardware/commercial support remains unmeasured |
| Helper denial events | FreeBSD kernel authorization and fixed-schema bounded logging | Local denial/redaction/blocked-sink/cancellation tests and actual dropped-UID, admission recovery and built-helper stderr assertions passed on both targets (run 37221158595); sampled logs and full abuse review remain open |
| Native probe cleanup | FreeBSD 15.1-RELEASE-p3 / 14.5-RELEASE amd64, Go 1.27.1 | Local synchronized regressions and all eight actual executor/collection cases passed on both native targets (run 37225094799); unrelated fixture group survived; abrupt helper SIGKILL and escaped groups remain outside this result |
| Helper crash containment | FreeBSD parent-death and base process-control facilities | Synchronized owner-death regression under development; no claim of descendant cleanup after helper SIGKILL yet |
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
records the unprivileged cache/outage/recovery path;
[security evidence](../testing/M1/security.md) tracks configuration, credentials
and the administration boundary; [console evidence](../testing/M1/console.md)
records native foreground/CLI/HTML and limited synthetic browser execution;
[runtime evidence](../testing/M1/runtime.md) records built-process restart,
independent SFTP continuity and bounded response/logging behavior; [performance qualification](../testing/M1/performance.md)
tracks the combined-process synthetic workload separately. The earlier C spike is ABI
evidence only. No
physical identity, file-client, scale or throughput claim follows.
