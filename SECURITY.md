# Security policy

## Reporting

Do not place exploitable details, credentials, or customer information in public
issues. A confidential security-reporting channel must be selected and verified
before the repository becomes public or a deployable release is offered. The
BSD Cloud email address is not automatically adopted for Horvium.

Current status: local foundation only; no supported production release and no
verified confidential reporting channel. Report foundation concerns directly to
the maintainer through the existing private project conversation for now.

## Sensitive boundaries

Authentication, sessions, CSRF, authorization, privileged helper requests, share
permissions, configuration changes, SSH credentials, replication retention,
restore operations, audit records, and release integrity need explicit abuse
cases and regression tests.

The management service is unprivileged. Privileged operations use a narrow
boundary with validation and authorization at the point of action. Restrict
management exposure independently from file-share exposure. Define supported
versions, disclosure handling, update guidance, and response expectations before
production release; do not promise service levels without a support policy.
