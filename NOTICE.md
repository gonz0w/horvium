# Foundation provenance

Horvium's repository tooling, hooks, tool pins, workflow baseline, issue forms,
and ruleset definitions are adapted from BSD Cloud Console, BSD-3-Clause,
local source revision `bfbc526e136c2b8d9ea6088df40b3b8d357b8c62`.
Engineering and document-lifecycle conventions also follow BSD Cloud Core,
local source revision `4c427f282fb2447b020ac7da177558c0017dda02`.

Copyright (c) 2026, BSD Cloud contributors. The upstream copyright and license
conditions are retained in [LICENSE](LICENSE). External tools and actions retain
their own licenses; their reviewed versions and artifact checksums are recorded
in [tool pins](tools/pins.json). They are development dependencies, not bundled
Horvium runtime components.

## Go syscall adapter

`golang.org/x/sys` v0.48.0, source revision
`613e2570718ecde85c04e69ebd5585c3881c442c`, supplies the FreeBSD
`unix.GetsockoptXucred` wrapper. Copyright 2009 The Go Authors, BSD-3-Clause;
the full upstream license is retained in [third_party/xsys-LICENSE.txt](third_party/xsys-LICENSE.txt).
Module/archive hashes are verified through Go's checksum database and recorded
in [go.sum](go.sum). It introduces no transitive modules or product cgo requirement.
