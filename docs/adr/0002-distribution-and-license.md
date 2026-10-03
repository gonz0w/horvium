# ADR 0002: Port-first distribution and BSD-3-Clause

Status: Accepted. Date: 2026-10-03. Authority: maintainer project discussion.

## Context and decision

The maintainer selected BSD-3-Clause and a FreeBSD port as the initial distribution,
followed by a full appliance image. Project identity is Horvium; slug is `horvium`.

## Consequences

Treat existing FreeBSD installations as first-class deployments. Native package
layout, rc.d, staging, configuration preservation, and upgrade/removal behavior
shape design from M1. The image consumes the same package when introduced.
Official Ports acceptance is an upstream process, not a bootstrap deliverable.
Third-party services/assets retain their own licenses and notices.

Alternatives considered: image-only distribution, or a separate implementation
for the appliance. Revisit distribution if validated operator requirements show
package-first installation cannot provide a dependable experience.
