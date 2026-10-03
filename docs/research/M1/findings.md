# Research: M1 native contracts and dependency baseline

Date: 2026-10-03. Status: Evidence only; proposed architecture needs review.
Design: [PRD](../../milestones/M1/PRD.md), [EDD](../../milestones/M1/EDD.md),
[Test Design](../../milestones/M1/TDD.md).

## Sources and versions

The [source manifest](source-manifest.json) records immutable upstream revisions
and SHA-256 values for the inspected files. These are source snapshots, not a
claim that a QEMU image was built from those exact commits. Dynamic support/release
pages were checked on the date above and must be refreshed at implementation.

| Component | Proposed baseline / source observation | Native qualification |
|---|---|---|
| FreeBSD | 15.1 primary; 14.5 compatibility; amd64 first | Disposable QEMU spike; product tests pending |
| Bundled OpenZFS | 15.1 source META: 2.4.2; 14.5 source META: 2.2.9 | Record actual userland/module strings separately |
| Go | 1.27.1; official FreeBSD amd64/arm64 artifact hashes captured | amd64 experiment; arm64 not tested |
| HTMX | 2.0.11, immutable asset source/checksum captured | Not embedded; browser behavior pending |
| x/sys | v0.48.0; FreeBSD credential wrapper, BSD-3-Clause; go.mod minimum 1.26.0 | Proposed dependency; native Go wrapper still pending |
| Ports | Snapshot in manifest; Go framework supports 1.26/1.27 | No poudriere/install/package tests |

[FreeBSD's security table](https://www.freebsd.org/security/) lists 15.1 and 14.5
as supported, with planned release EOL dates 2027-03-31 and 2027-06-30 respectively.
15.0 has passed its listed EOL. These dates are upstream support, not Horvium
support promises. [Go release records](https://go.dev/doc/devel/release) and
[official downloads](https://go.dev/dl/) identify Go 1.27.1; development-host
Go 1.27.0 was observed separately and is not the native spike toolchain.

## Findings

1. FreeBSD's inspected `geom_getxml` implementation reads `kern.geom.confxml` via
   sysctl. The graph has classes, geoms, providers and consumers. Go can decode
   the narrowly tested provider geometry without runtime libgeom/cgo. This does
   not establish stable disk serials across controllers or correct multipath
   collapsing. Pointer-like graph identifiers are not durable hardware identity.
2. Explicit `-H -p` ZFS projections offer tab-separated fields and literal numeric
   values. Restrict the initial projection to resource name/GUID, numeric capacity
   and pool health. Do not parse arbitrary free-text mountpoint/property strings
   this way. Inspected newer manuals have `-j` JSON options; that is insufficient
   to assume JSON on the older branch. Recommend the natively tested common format.
3. The inspected Unix socket manual specifies `LOCAL_PEERCRED` as kernel-provided
   effective credentials. Server sees the client's credentials at connect; client
   sees the server's credentials at listen. Verify the `xucred` layout version.
   A socketpair created before dropping root is not a suitable denial experiment.
   The focused x/sys wrapper exists; product Go transport tests remain necessary.
4. Go's [CrossOriginProtection](https://pkg.go.dev/net/http#CrossOriginProtection)
   permits safe methods and requests without both provenance headers. Pair it
   with authentication, CSRF tokens and explicit Host/Origin policy. Loopback-only
   binding alone is insufficient authorization.
5. [HTMX documentation](https://htmx.org/docs/) supports local assets and security
   configuration. Disable evaluation/script execution/history caching and injected
   styles; use CSP and project CSS. Exact configuration must pass browser tests.
6. The inspected Ports Go framework and [Porter's Handbook](https://docs.freebsd.org/en/books/porters-handbook/uses/)
   favor framework-managed module fetching and compatible Go version ranges.
   Pin the tested toolchain for evidence, while selecting the port's version range
   from its actual module requirements. A private repository is not a publicly
   fetchable upstream distfile; a publishable source artifact and maintainer contact
   must exist before official Ports submission. Port-first distribution remains
   the agreed direction; no fake metadata is introduced during research.

Immutable source links and byte digests for points 1-3/6 are in the manifest.
No upstream source or HTMX asset is copied into the product yet.

## Native experiment and next steps

See [spike record](native-spike.md) for authorization, fixture boundaries,
procedure, source revisions, exact versions, failures and completed assertions.
The first run passed on 14.5. Its 15.1 fixture check failed before discovery due
to the changed verbose mdconfig layout; source inspection showed extra fields.
Use the exact backing-file query rather than assuming the filename is the final
verbose column. The corrected run passed both releases on source
`411ca08d99544173a685422957f7a41e3eeaa4aa`; exact versions and limits are recorded
in the spike. This is native research evidence, not product integration acceptance.

Recommend the two proposed ADRs for review. Research success does not approve
product architecture or satisfy its planned security/resource/UI tests. Remaining
questions are complete physical identity, actual Go peer adapter, bounded large
inventory behavior, browser accessibility and all later file-client/restore gates.
Client priorities are macOS and Linux/BSD; native experiments use QEMU, as requested
by the maintainer. No physical MicroServer or commercial host has been measured.
