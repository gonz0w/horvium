# Focused review qualification

Status: portable workflow and prebuilt state execution qualified; maintainer
review pending. Product behavior and all 21 accepted M2 manifests remain unchanged.
See the [sanitized source-bound manifest](focused-review-manifest.json).

Behavioral regressions cover host/FreeBSD/test consumer selection, broad/unknown
fallback, eligible prose, exact-input/environment/base evidence matching, failed
or tampered evidence, compiler-listed fuzz discovery, bounded typed commands,
empty selected tests, prebuilt artifact verification and interrupted cleanup.
Planner, prebuilt selection and executable-cleanup tests failed first. The first
full run remained unqualified because Go's generated compiler scratch path made
its environment digest unstable; a failing regression drove the fix. The second
fresh full run passed at signed source `3e026d5cc881dd73de6d72a49171a9fa1864421d`:
85 tooling tests, all ordinary Go tests, race checks across 11 packages and all
21 compiler-listed bounded fuzz targets, with completed clean supervisors.

The actual focused-run diagnostic used the real host/FreeBSD/test graph for a
representative `internal/admin/config.go` change. It selected eight packages and
three fuzz targets, passed in about 16 seconds and completed clean teardown.
This is a runner demonstration, not fabricated evidence of an actual changed
product PR. A named storage codec regression also passed in 0.093 seconds.
The tooling PR itself conservatively selects full mode because shared tools and
review policy changed. Its final documentation followup must attach the current
baseline/identity report and original tested source/hash if deep evidence is reused.

| Native run | Executed source | Result | Observed elapsed |
|---|---|---|---|
| FreeBSD 15.1-RELEASE-p3, ZFS root | `e87e1041d36ed3ca56bf087633da64bf021e1f98` | All 161 assertions, 14 parent tests and 57 subtests | 34.47 seconds |
| FreeBSD 14.5-RELEASE, UFS root | `3e026d5cc881dd73de6d72a49171a9fa1864421d` | Same complete cases | 26.95 seconds |

Both fresh guests verified the same executable SHA-256 before execution. The
review-fingerprint-only fix between those runs did not change executable bytes;
original source commits remain explicit. No guest package/SDK was installed.
Actual ENOSPC/read-only/ACL publication controls, descriptor-bound reads, lock/process
controls and all fixture teardown passed. Base/runtime hashes remained unchanged;
keys, images, variables and host executables were removed and process groups stopped.

The preceding guest-source-build runs of the same 161-case suite took about
292/260 seconds; these prebuilt observations were about 34/27 seconds. These are
local warm-cache observations, not a controlled benchmark, physical NAS performance
or revised budget. The final lab status reported every owned session stopped with
verified teardown. Other prebuilt suites, Linux-host execution, source-build SDK
qualification, new runtime behavior and commercial/hardware support remain outside
this evidence. Manual M1 accessibility acceptance remains pending.

See [policy](../engineering/local-review.md) and [native procedure](local-qualification.md).
