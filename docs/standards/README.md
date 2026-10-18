# Advisory suite standards reference

The owner authorized shared standards adoption on 2026-10-18. The
[adoption manifest](adoption-manifest.json) pins Suite-Standards `7a99fb2e6855d9f73752b182118637d9e79dc03c` and
the exact SHA-256 of [shared standards v0.1.0](shared-v0.1.0.md), copied unchanged
from `docs/standards.md` at that revision. The source is the private
[Suite-Standards repository](https://github.com/gonz0w/suite-standards/tree/7a99fb2e6855d9f73752b182118637d9e79dc03c);
its [BSD-3-Clause license](https://github.com/gonz0w/suite-standards/blob/7a99fb2e6855d9f73752b182118637d9e79dc03c/LICENSE)
and Git history retain provenance. This is development documentation, with no
runtime dependency.

## Scope and precedence

The `go-service` profile is an advisory classification, not an assertion that
every suite recommendation is adopted or implemented. Existing AGENTS,
CONTRIBUTING, security, review and milestone rules remain authoritative.
Horvium's Actions remain disabled. Local evidence still needs manual review;
this reference grants no runner, remote success publisher, unattended merge,
credential, service, deployment or live-system authority. Conflicts require an
explicit reviewed policy decision before enforcement.

Preserve FreeBSD storage, disposable-fixture safety, both-release qualification, manual Safari/VoiceOver observations and runtime-write denial. A component merge or advisory drift result supplies no
native, browser, security, performance or beta acceptance. The immutable
[BSD Classic token reference](https://github.com/gonz0w/suite-standards/blob/7a99fb2e6855d9f73752b182118637d9e79dc03c/design/bsd-classic-v1.json)
is available for future reviewed UI work; this change copies no token file and
changes no runtime stylesheet, interaction or UI acceptance claim.

## Foreground drift inspection

Obtain the pinned standards revision through the reviewed source-acquisition
process into a separate trusted checkout. Verify its HEAD equals the full
manifest revision and inspect its source before invoking the inspector. From
the Horvium checkout, with `standards_checkout` set to that verified checkout:

```sh
python3 "$standards_checkout/tools/suite.py" adoption \
  docs/standards/adoption-manifest.json --root .
```

Do not fetch a moving branch during checks. The inspector only compares adopted
bytes with declared hashes and reports advisory drift; it does not authenticate
the source acquisition, run candidate code, attest tests or authorize a merge.
Use the project's existing clean-source review and native gates separately.

Updates require a normal signed, DCO-compliant reviewed PR with an explicit new
pin and adopted-file hashes. Rollback removes this reference directory and its
index/compatibility references; no product data or runtime migration is involved.
