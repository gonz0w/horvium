# Compatibility matrix

**No production support is claimed by M0.** Architecture targets and observed
validation are separate. Refresh exact versions in each active research manifest.

| Area | Target | Evidence/status |
|---|---|---|
| Runtime | Supported FreeBSD releases; exact set selected in M1 | Planned; no native NAS runtime tested |
| Architecture | amd64 first candidate; other FreeBSD architectures as validated | Hardware-neutral design, no universal support claim |
| Storage | ZFS pools/datasets on supported direct-access devices | Planned; no pool modified by M0 |
| File clients | SMB, NFS, SFTP on selected desktop/Unix clients | Client/version/permission matrix pending |
| Backup peer | Compatible ZFS receiver over authenticated SSH | Planned; feature negotiation and restoration pending |
| Management | Go, HTML5/CSS, HTMX; modern browsers | No product UI implemented |
| Distribution | FreeBSD port/package, then image | Packaging plan only; not in official Ports |
| Small hardware | 16 GB MicroServer Gen8 / four 3 TB SATA disks | Reference profile supplied by maintainer, unmeasured |
| Larger hardware | Representative commercial FreeBSD deployments | Scale budgets and lab evidence pending |
| Tooling | Python 3.14.7 + checksum-pinned tools on Linux/macOS | See bootstrap status for checks actually run |

Evidence levels: D0 designed; D1 unit/component verified; D2 native/client verified;
D3 workflow/restore verified; D4 release-qualified with failure, upgrade, security,
and performance gates. Mark unavailable/skipped/failed separately from passed.
Commercial deployment size alone is not a D4 claim.
