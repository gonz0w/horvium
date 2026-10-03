# Operational runbook plan

Runbooks become executable guidance as the relevant feature is implemented and
tested. M0 supplies topics only; there are no product commands to run yet.

Required before first release: install/first login; storage discovery/adoption;
share/identity setup; disk replacement and scrub failures; capacity exhaustion;
backup-peer trust/pairing; replication interruption/common-snapshot loss;
file/dataset/full-host restore; config/credential recovery; management unavailable
while shares continue; package upgrade/rollback/removal; and secure update handling.

Each runbook includes prerequisites, affected resources, validation, recovery,
operator decision points, and links to tests. Restore procedures identify which
state is replicated and which configuration/keys must be backed up separately.
Do not make recovery depend solely on an operational GUI or an available data pool.
