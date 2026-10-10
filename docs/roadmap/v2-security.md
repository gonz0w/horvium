# Second release security roadmap

Status: Proposed planning; release assignment and implementation designs pending.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).

The second product release should strengthen protection against external attackers,
including attackers using AI to automate discovery, exploitation and data theft.
The scope does not depend on operators running AI agents. The proposed direction
combines software controls, a hardened FreeBSD appliance image and a deployment
security guide. It assigns no SemVer tag, delivery date or approved product contract.

Keep the [first release milestones](milestones.md), [charter](../project/charter.md)
and accepted [package-first decision](../adr/0002-distribution-and-license.md).
M7 already provides the appliance-image milestone. The image consumes the same
tested package; existing FreeBSD installations remain first-class deployments.
An image-only pivot or accelerated image schedule would require an amended ADR
and approved PRD, EDD and Test Design Document.

Live security threat-model pickup, priority and dependencies are in [issue #167](https://github.com/gonz0w/horvium/issues/167)
and the [roadmap index](github-roadmap-index.md). The design obligations below
remain authoritative planning material, not a second progress checklist.

## Security outcomes

Protect confidentiality, recoverable history and availability separately. Define
the attacker authority for each claim: file client, management session, service
process, primary root, replication principal or backup root. Geographic separation
does not establish administrative independence.

The proposed minimum failure boundary is that primary root compromise cannot erase
the backup's protected history or recover its administrator credentials. Backup
root compromise remains capable of destroying its own online storage; an offline
or independently controlled recovery reserve covers that additional failure.
Unlocked plaintext remains exposed to compromised components authorized to read it.

AI increases pressure to reduce exposure and patch promptly. The June 2026
[Five Eyes statement](https://www.ncsc.gov.uk/news/the-ai-shift-in-cyber-risk-why-leaders-must-act-now)
warns that AI accelerates attacks and reduces the discovery-to-exploitation window.
This motivates smaller privileges and independently tested recovery rather than
a claim to identify or block every AI-generated attack.

## First release obligations

Do not defer basic safety to the second release. M3 must qualify identities and
protocol access boundaries; M4 must qualify explicit peer trust, constrained
replication, receiver-controlled retention and restoration; M5/M6 must qualify
installation, updates, credential/configuration recovery and security runbooks.
These existing obligations remain in their existing milestone scope. New controls
need reviewed amendments if first-release acceptance depends on them.

## Proposed software work

These IDs are planning targets. They become implementation requirements only in
a reviewed milestone packet with exact authority, failure and acceptance contracts.

| ID | Operator outcome | Proposed scope | Required evidence |
|---|---|---|---|
| V2-SEC-001 | Stolen daily credentials have limited administrative consequences | Separate routine and destructive authority; evaluate phishing-resistant administration and independent authorization for retention, trust and recovery-key changes; bind approval to the exact action | Stolen session, replay, changed action, factor loss/recovery and revocation; a compromised approving endpoint cannot be assumed trustworthy |
| V2-SEC-002 | Replication cannot erase independently protected history | Strengthen M4 with separately administered protection windows, explicit exceptional deletion and safe capacity-exhaustion behavior; define what a primary or receiver compromise can do | Use actual primary-root and replication authority against disposable receivers; deny history deletion and policy weakening; fail full without silently pruning protected history |
| V2-SEC-003 | Offsite storage can preserve data without plaintext custody | Design encryption, raw encrypted replication, key recovery and rotation/revocation semantics; separate boot, data and recovery-secret policies | Both actual FreeBSD releases, wrong/lost keys, interrupted encrypted replication, replacement-primary restore and residual metadata disclosure; never imply disk encryption protects mounted plaintext |
| V2-SEC-004 | Compromise of a service has bounded reach | Evaluate FreeBSD base confinement, descriptor capabilities, privilege separation and role-specific network access for actual services | Compromised service fixtures cannot access unrelated datasets, secrets or privileged operations; restart, update, protocol and performance compatibility; accepted ADR before confinement adoption |
| V2-SEC-005 | Operators can distinguish configuration, observations and verified protection | Bounded read-only posture checks, configuration drift, update status and independently retained backup/recovery evidence; minimize sensitive logs | Disabled collectors and lost observers show unknown; falsified primary reports cannot qualify backup protection; private paths and credentials stay out of exported evidence |
| V2-SEC-006 | Irreversible changes remain deliberate under stolen authority | Evaluate scoped privileges, independently enforced delays/cancellation and exact-action approval for protected-history deletion and trust/key changes | Alternate UI/CLI/helper paths cannot bypass the qualified boundary; host-root limits explicit; clock/reboot/restore and cancellation races; emergency recovery and capacity denial; delays alone are not independent authorization |
| V2-SEC-007 | Loss or compromise of the primary is recoverable | Extend M4/M6 with clean-host recovery exercises, independently selected historical checkpoints and separately recoverable configuration/keys | Primary and normal admin workstation unavailable; slow corruption and failed newest backup; restore selected files and services without trusting the compromised host |

Protect limits across a complete operation and repeated operations, not just one
request. A series of individually allowed changes can still delete or expose a
large archive. Restrict file clients to the data they need; management confinement
does not contain a stolen SMB/NFS/SFTP credential with broad data access.

Posture checks report observed facts and their scope. They must not produce an
unqualified security score, silently alter host configuration or claim an external
firewall, offline copy or administrator workstation is safe.

## Proposed appliance work

M7 should deliver bootable installation and recovery media for an explicitly
qualified FreeBSD base and hardware matrix. ISO and USB artifact formats, boot
layout and OS update mechanism belong to its reviewed design. Security continues
through updates and recovery rather than ending at the installation image.

- Ship the tested Horvium package and required services only. Keep unrelated
  application hosting and development tooling outside the dedicated NAS image.
- Keep network administration and file services closed until local commissioning
  establishes unique credentials, approved interfaces and explicit service access.
  No shared passwords, baked private keys or reusable setup tokens.
- Provide separately tested primary and backup-receiver profiles. A receiver
  exposes only its selected replication function by default and owns retention
  independently; role selection cannot confer trust in a peer.
- Prefer FreeBSD base facilities for host firewall, logging and isolation.
  Evaluate inbound IPv4/IPv6 defaults and bounded outbound access with required
  update, time, name-resolution, replication and alert paths. Document residual
  data access through authorized protocols; a firewall is not a content policy.
- Verify release identity and authenticated updates, document reproducibility
  limits, retain notices/SBOM/provenance, and define both OS and package support.
  Checksums alone do not authenticate a downloaded image.
- Qualify interrupted updates and clean recovery independently of configuration
  and durable-state migrations. Boot rollback cannot erase storage effects or
  recreate retired execution authority. Never silently upgrade pool features.
- Require explicit installer disk selection, preservation of all unselected
  devices and a local recovery path for failed network policy or credential setup.
  First boot must not adopt data pools or enable shares automatically.

"As much security as possible" becomes a tested security baseline with usable
recovery. Secure levels, immutable flags, read-only system layouts and stronger
confinement are candidates, not switches to enable without proving compatibility
with updates, storage operations, service reloads and emergency recovery.

## Security guide and ownership

The [deployment security guide](../security/deployment-security.md) records the
image/software/operator boundary, residual risks and required evidence. It must
ship with each qualified package/image revision and remain accessible offline.
Package installations need prerequisite checks and explicit remediation guidance;
they must not overwrite an existing host's firewall, SSH or unrelated services.

The image cannot supply an independently secured admin workstation, physical
separation, upstream network policy, offline key custody or a real restore exercise.
Software can guide and record these actions without declaring them complete from
an operator checkbox or the primary's own success report.

## Delivery and research

1. Preserve existing M1-M6 acceptance while drafting a whole-system threat model
   and second-release proposals now. First-release completion precedes delivery
   of the second release; planning need not wait for it.
2. Review the linked security outcomes and split them into PRD/EDD/Test Design
   packets. Decide administrative factors, key custody, retention exceptions,
   confinement and update/recovery ownership through explicit designs and ADRs.
3. Qualify software controls on ordinary supported FreeBSD installations, then
   qualify the image's additional host policy, installer, boot and update behavior.
4. Publish the responsibility guide and compatibility evidence with the release;
   retain hostile-compromise and clean-restore tests as release gates.

Track automatic vulnerability analysis/repair, hardware-enforced compartmentalization
and cryptographic migration as research until adopted. [DARPA AIxCC](https://www.darpa.mil/news/2025/aixcc-results)
demonstrates automated finding/patching; [CheriBSD](https://www.cheribsd.org/) explores
CHERI on specific platforms; [NIST](https://www.nist.gov/cybersecurity-and-privacy/what-post-quantum-cryptography)
describes post-quantum standards and long-lived confidentiality risks. None supplies
Horvium support evidence or authorizes an autonomous defender with unrestricted root.

Documentation checks cannot qualify any control above. Native attacks, updates
and recovery tests use disposable fixtures only, with verified teardown. Carry
source-bound results into the compatibility matrix and leave PRs for review.
