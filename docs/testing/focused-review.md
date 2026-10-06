# Focused review qualification

Status: implementation under validation; not yet qualified. This tooling change
implements the maintainer-approved focused testing policy. Product behavior and
accepted M2 manifests remain unchanged.

Behavioral regressions cover host/FreeBSD/test consumer selection, broad/unknown
fallback, eligible prose, exact-input/environment/base evidence matching, failed
or tampered evidence, compiler-listed fuzz discovery, bounded typed commands,
prebuilt artifact verification and interrupted fixture cleanup. Initial planner,
prebuilt selection and executable-cleanup regressions failed before implementation.

Qualification requires fresh full portable review of the workflow, actual prebuilt
state execution on both declared FreeBSD guests with existing complete assertions,
exact-input reuse on a documentation followup and verified teardown. Results and
sanitized source/artifact provenance will be appended after execution. This does
not qualify other prebuilt suites, Linux-host execution or new product behavior.

See [policy](../engineering/local-review.md) and [native procedure](local-qualification.md).
