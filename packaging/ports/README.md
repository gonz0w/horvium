# Private FreeBSD Ports source overlay

The concrete overlay is [horvium](horvium/Makefile), following the reviewed
[#159](https://github.com/gonz0w/horvium/issues/159) stock Ports/pkg/rc.d design.
[Source status and input bindings](../../docs/milestones/M5/source-profile.md)
distinguish implementation from native package/lifecycle acceptance. This is a
private experimental lab profile, not an official port or production release.
No package build/install, account, credential or service change is authorized by
these instructions. #160/#161 retain their real native acceptance requirements.

## Offline source and layout

Use the signed application source and explicit lab version in `horvium/source.mk`.
The source bundle is private and operator supplied; no public GitHub/proxy fetch
or upstream Ports submission is implied. The actual current archive is checked
by `horvium/distinfo`. Keep it outside Git, under operator custody.

`tools/prepare-port-source.py` prepares a fresh private output directory from a
full signed Git commit and an existing checksum-pinned Go SDK/module cache. Its
required flags are `--ref`, `--version`, `--go`, `--go-sha256`, `--module-cache`,
`--allowed-signers` and `--out`; run `--help` for the argument descriptions.
It verifies the source signature, runs offline `go mod verify`/`go mod vendor`,
preserves module inputs, normalizes the archive and writes `bundle.json`,
`source.mk` and `distinfo`. Failures retain their private preparation directory.
It builds or executes no Horvium binary, and installs no package or SDK.

For a future explicitly approved native build, place that archive at
`DISTDIR/horvium/<DISTNAME>.tar.gz` and copy this overlay into the bound private
Ports tree. Fetching is disabled: a missing archive fails. `GO_MODULE` is unset
deliberately so the framework consumes the bundle's vendor tree without its
private module-proxy/tidy path. `USES=go:1.27,modules` selects the compatible
framework family; acceptance additionally checks the exact Go1.27.1 compiler,
Ports/jail/OS identities and resulting package/staged manifest on both targets.
This profile fixes `PREFIX=/usr/local` and FreeBSD/amd6415.1/14.5; actual native
compatibility is unqualified until those checks run.

## Explicit operator lifecycle

Payload includes the binary, both disabled rc.d front ends, shared service logic,
an intentionally invalid-until-configured sample, licenses and an operator guide.
The active config is deliberately not an `@sample` plist entry: removal must
preserve it even if its bytes equal the sample. Package hooks refuse existing
profile pidfile/socket entries; they never signal, unlink, provision or restart.

The [installed operator guide](horvium/files/package-lifecycle.md) describes the
existing explicit identity/config/token setup and ordered helper/management
launch, authenticated use, stop/restart and preservation. A launch acknowledgement
is not readiness. Ordinary trusted-administrator PID control has a disclosed
check/signal race; qualification separately requires original fixture waits and
retirement. The known #414 cleanup correction/qualification remains required.

Appliance images later consume the tested package, with separate boot/install,
hardening, monitoring and OS-update evidence. The read-only package does not
activate storage commands or replace the M1-M6 NAS release gates.
