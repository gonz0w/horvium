# Security and supply chain

Every product EDD includes trust boundaries, assets, threats, authorization,
credential lifecycle, denial behavior, and failure recovery. Test abuse cases
for path/device selection, helper requests, remote targets, config generation,
sessions/CSRF, and retention/destructive operations. Keep privileges at the point
of action; use argument vectors and bounded output rather than shell interpolation.

Store private keys and credentials outside Git; redact logs and test evidence.
Use explicit peer identity verification and narrowly scoped receive privileges.
Authentication, service accounts, audit retention, encrypted storage, and key
recovery require designs before being advertised as supported features.

GitHub Actions are disabled. Tools are checksum-pinned and the foundation
scanner runs in the complete local review gate. Source-bound evidence requires
maintainer review; GitHub does not independently attest local check results. Dependency
updates require review of source/license/security impact. Preserve upstream
notices. Confirm GitHub security feature availability instead of recording disabled
or unavailable controls as passed. No paid service is adopted automatically.

Before a deployable release: verify confidential reporting, release-signing
procedure, SBOM/provenance, restore evidence, update/rollback instructions, and
supported-version policy. Image security inherits the package contract and needs
its own installer/update validation.
