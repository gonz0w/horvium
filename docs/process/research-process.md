# Research process

At milestone entry, refresh primary FreeBSD/OpenZFS/Samba/Go/HTMX documentation,
release/support status, candidate packages, clients, and relevant prior evidence.
Record the date, exact versions/commits, sources, observations, and unresolved
questions in a research manifest. Cite upstream docs for claims and distinguish
observed behavior, source statements, inference, and proposed decisions.

Use disposable fixtures for spikes. Record bounded question/time box, commands
or reproducible procedure, environment, results, and limits. Do not promote spike
code to production without design and review. Never capture live credentials or
customer files as evidence. Pool-feature compatibility needs observations on both
send and receive endpoints, not only a version string.

Adopt decisions through ADRs and EDDs. Research does not automatically authorize
production deployment, irreversible disk changes, or new scope.
