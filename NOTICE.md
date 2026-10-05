# Foundation provenance

Horvium's repository tooling, hooks, tool pins, workflow baseline, issue forms,
and ruleset definitions are adapted from BSD Cloud Console, BSD-3-Clause,
local source revision `bfbc526e136c2b8d9ea6088df40b3b8d357b8c62`.
The adapted executable Actions workflows have since been removed under the local
review policy; this historical attribution remains.
Engineering and document-lifecycle conventions also follow BSD Cloud Core,
local source revision `4c427f282fb2447b020ac7da177558c0017dda02`.

Copyright (c) 2026, BSD Cloud contributors. The upstream copyright and license
conditions are retained in [LICENSE](LICENSE). External tools and actions retain
their own licenses; their reviewed versions and artifact checksums are recorded
in [tool pins](tools/pins.json). They are development dependencies, not bundled
Horvium runtime components.

## Disposable VM assets

The local development runner fetches existing anyvm-org/freebsd-builder 2.2.8
images and public builder fixture SSH keys. Source URLs and checksums are recorded
in [image pins](tools/performance-vm-pins.json) and
[fixture-key pins](tools/local-vm-pins.json); key digests were checked against
[publisher release metadata](https://api.github.com/repos/anyvm-org/freebsd-builder/releases/tags/v2.2.8)
on 2026-10-05. Assets remain in ignored development fixtures, with keys removed
after use; no image, private-key content or upstream launcher code is vendored.
The operator supplies QEMU/UEFI; actual runtime hashes are captured per run.
This introduces no NAS runtime dependency or redistribution claim.

## Go syscall adapter

`golang.org/x/sys` v0.48.0, source revision
`613e2570718ecde85c04e69ebd5585c3881c442c`, supplies the FreeBSD
`unix.GetsockoptXucred` wrapper. Copyright 2009 The Go Authors, BSD-3-Clause;
the full upstream license is retained in [third_party/xsys-LICENSE.txt](third_party/xsys-LICENSE.txt).
Module/archive hashes are verified through Go's checksum database and recorded
in [go.sum](go.sum). It introduces no transitive modules or product cgo requirement.

## Local HTMX asset

HTMX 2.0.11, upstream `bigskysoftware/htmx` revision
`fa978b24e75fb03c137bf2cdae4fef0e711cf8a1`, is embedded in the console.
License: Zero-Clause BSD, retained in
[third_party/htmx-LICENSE.txt](third_party/htmx-LICENSE.txt).
The upstream `dist/htmx.min.js` SHA-256 is
`d6fdc75f204e6bdefa99b69bf1e6d4ac69b8a364f77929f45c13476b4000f717`.
The vendored file adds one terminal newline; its SHA-256 is
`7f14135fe8ee3ed5f1a85fea21ba4dfba8236d697c4f95ab575d1c9091bb06ca`.
License SHA-256: `d3d2456f76414f2456104660ebd65aff1c04cd7966b942bdabd63f3cdb316a38`.
See the [source manifest](docs/research/M1/source-manifest.json). No runtime CDN,
Node dependency, UI framework or transitive JavaScript modules are introduced.
