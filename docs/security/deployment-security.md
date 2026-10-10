# Deployment security and recovery responsibilities

Status: Proposed security guide; deployment controls and executable procedures
require reviewed designs and qualification. Date: 2026-10-10.
Owner: Cam (`@gonz0w`).

Horvium should protect a private archive across a primary, an independently
administered backup and a recovery reserve. External attackers may use AI to
accelerate reconnaissance, exploitation and theft. This guide applies whether
or not the operator uses AI. It defines proposed responsibilities and residual
risks; no hardened image or production security baseline is currently qualified.

Read the [second release proposal](../roadmap/v2-security.md),
[architecture baseline](../architecture/baseline.md),
[security and supply chain policy](supply-chain.md) and
[compatibility matrix](../compatibility/matrix.md). Current M1 loopback/SSH access
and M2 qualification fixtures are development interfaces, not a deployment recipe.

## Assets and attacker authority

Protect files, historical versions, filenames/metadata, identity and ACL mappings,
service/configuration state, encryption and recovery keys, peer identity, update
trust and evidence needed to restore. Availability and confidentiality require
different controls: a backup can retain ciphertext while lacking the keys to read it.

Model stolen file credentials, a compromised client, stolen management sessions,
service exploitation, primary root, replication-credential theft, backup root,
physical theft and loss of the normal administrator workstation. Include router,
VPN, remote-management controller, firmware and supply-chain compromise. Do not
infer trust from a private address, a network segment or a successful login.

| Attacker position | Proposed protection boundary | Residual risk |
|---|---|---|
| File client or stolen file credential | Resource-specific access and independent retained history | All authorized readable data can be copied; authorized writes can poison new backups |
| Management session | Scoped authority and separately enforced protection of exceptional destructive changes | Valid allowed operations can still be malicious; session authentication does not establish operator intent |
| Compromised service | Qualified OS/service isolation and narrow privileged contracts | Kernel or stronger privilege compromise may escape the boundary; confinement needs actual integration tests |
| Primary root or replication principal | No backup administration secrets or authority to erase protected receiver history | Primary plaintext and current data may be lost; malicious content can still arrive through replication |
| Backup root | Independently controlled reserve and separate plaintext-key custody where designed | Receiver root can destroy its online copy; encryption does not prevent deletion |
| Physical theft or simultaneous host loss | Appropriate encryption, separately held recovery material and geographic separation | Loaded keys and unlocked machines differ from stolen powered-off media; recoverability depends on intact keys and copies |

These are design targets. Until tested, show the relevant protection as planned
or unknown. No control establishes universal resistance to an AI-enabled attacker.

## Responsibility boundaries

| Area | Software or image responsibility | Operator or external responsibility | Evidence needed |
|---|---|---|---|
| Network exposure | Closed commissioning state, explicit interfaces/services, reviewed host firewall including IPv6 | Router/VPN rules, independent admin segment where available, no unintended public forwarding; inspect BMC/IPMI access | Actual reachability from allowed and forbidden networks, including after reboot/upgrade |
| Administration | Unique identity/credentials, scoped privileges, session revocation and reviewed destructive-action authority | Protect administrator devices and factors; keep backup administration credentials unavailable to primary and daily file clients | Wrong/stolen session and exact-action/replay denials; factor-loss recovery |
| File access | Qualified SMB/NFS/SFTP identity and ACL behavior; no automatic broad shares | Assign access by need, secure clients, review departed/lost devices and accounts | Real allowed/denied clients and multi-protocol permission tests |
| Backup independence | Constrained peer trust/replication; receiver-owned protected retention and explicit full-store behavior | Separate backup administration, location and reserve custody; choose recovery objectives and history duration | Primary-root/replication abuse and offline/full receiver tests; no silent protected-history deletion |
| Encryption | Reviewed storage/transport/key lifecycle; encrypted replication only when qualified | Decide plaintext/key custody, unlocking policy and independently protected recovery copies | Actual encrypted send/receive and clean replacement-host restore; document metadata exposure |
| Updates | Authenticated releases, exact versions/provenance, separate OS/package lifecycle and tested recovery | Verify media against a separately trusted release identity, maintain supported versions, schedule required updates | Wrong signer/altered artifact and interrupted update tests; recover without storage rollback assumptions |
| Logging and alerts | Bounded redacted records, clear freshness/errors and qualified external evidence transfer | Select an independently administered evidence/alert destination and check delivery failures | Lost collector/receiver/logging and falsified-primary cases; secrets absent from exports |
| Physical environment | Hardware-specific capabilities only as qualified | Locked access, offsite custody, power/fire/water planning, firmware/BMC maintenance | Declared hardware/boot evidence and physical recovery exercise |
| Recovery | Offline-readable runbooks, separate configuration/key requirements and clean-host workflow | Maintain reserve/media/keys; rehearse recovery and arrange trusted emergency access | Restore without primary or usual admin workstation; open representative files and verify identities/permissions |

The image should make its own safe defaults automatic and its unresolved external
dependencies visible. It cannot validate an offline reserve by inspecting the
primary or prove administrator-device security through a checkbox. Posture
inspection must be bounded and read-only unless a separately authorized change
workflow exists.

## Image commissioning and service lifetime

The proposed appliance is install/recovery media consuming the tested package,
not a separate NAS implementation. Bootable media alone does not establish a
trusted installed system. M7 must define verified installation, per-host secrets,
disk selection, first login, selected role/interfaces and authenticated updates.

Keep network administration and file services closed until locally commissioned.
No factory password, shared SSH host key or embedded recovery secret. Installing
must preserve unselected devices; selecting a backup role must not silently pair
with a discovered primary or share that primary's administrator authority.

Test firewall changes with retained local-console recovery. Account for IPv4,
IPv6 and service/update dependencies; no universal operator subnet or static
hardware layout is assumed. Choose base facilities in reviewed designs rather
than bundling every hardening toggle. FreeBSD documents [security mechanisms](https://docs.freebsd.org/en/books/handbook/security/),
[firewalls](https://docs.freebsd.org/en/books/handbook/firewalls/) and
[OS update options](https://docs.freebsd.org/en/books/handbook/cutting-edge/);
their existence is not Horvium integration evidence.

Package installations must document host prerequisites and inspect their scope
without taking ownership of unrelated firewall, SSH, boot or service configuration.
Declare which security guarantees depend on the image's host policy.

## Backup retention and confidentiality

Receiver-controlled retention must survive primary root and replication-credential
compromise within the qualified contract. Neither normal replication nor storage
pressure should remove protected history. Alert and refuse incoming work safely
when capacity is exhausted; resolve exceptional pruning through independent authority.
Choose duration against the time a compromise may remain unnoticed, not merely
the normal replication interval. Keep independently selected historical checkpoints
for irreplaceable material so slow corruption does not age out every good version.

ZFS checksums detect storage corruption, not malicious but valid file edits.
Snapshots and holds are not absolute immutability against receiver root or physical
destruction. State precisely which principal cannot delete them and preserve a
reserve outside that authority. [OpenZFS snapshot semantics](https://openzfs.github.io/openzfs-docs/Basic%20Concepts/Datasets/Snapshots%20and%20Clones.html)
describe destruction/hold behavior; [CISA guidance](https://www.cisa.gov/stopransomware/ransomware-guide)
recommends offline encrypted backups and tested recovery.

An encrypted offsite receiver without plaintext keys is a candidate design.
[OpenZFS raw sends](https://openzfs.github.io/openzfs-docs/man/v2.3/8/zfs-send.8.html)
can preserve encrypted blocks without loading source keys; ordinary non-raw sends
of encrypted datasets expose plaintext within the send pipeline, even when SSH
protects network transport. Qualify exact bundled versions, stream/property
handling and restoration before advertising support. Encryption can leave metadata
exposed and cannot prevent root on an unlocked source from reading its plaintext.

Define key replacement, recovery and compromise consequences explicitly. Changing
a wrapping key must not be advertised as making every previously stolen key or
historical copy harmless. Keep recovery material accessible when data pools, normal
devices, credentials or network services are unavailable; do not put the only key
inside the encrypted backup it unlocks. Design any emergency/estate access to match
the operator's confidentiality needs without granting routine backup access.

## Compromise and recovery exercises

Approved designs should turn these scenarios into source-bound tests on disposable
fixtures. This list is not an executable incident-response procedure.

1. Compromise a normal file client and try broad reads, writes and deletion.
   Demonstrate the exact accessible scope and restoration from independent history.
2. Give the fixture attacker primary root and separately the replication principal.
   Try receiver-history deletion, retention weakening, namespace escape and access
   to backup administrator secrets. Preserve sentinel history and record denials.
3. Compromise the backup. Demonstrate remaining plaintext confidentiality where keys
   are absent, and recovery from a reserve outside the receiver's control.
4. Replicate gradual valid corruption, then fail the newest backup and fill the
   receiver. Preserve protected history, expose failed freshness and restore an
   independently selected earlier checkpoint.
5. Lose the primary and normal administrator workstation. Restore onto a clean
   supported host using separately held configuration, identities, keys and media;
   do not restore executable trust or active management sessions from suspect state.
6. Interrupt commissioning/updates and lose an administrative factor. Demonstrate
   local recovery, unchanged unselected disks and no weakening of peer/retention
   policy. Test firmware/boot features only on their declared hardware.

During a real incident, independently isolate affected access and preserve usable
history before changing the archive. The qualified runbook must specify who may
pause replication or exceptional deletion, how alerts/evidence remain available,
and how recovery avoids the compromised administration path. An autonomous response
must not erase evidence, prune backups or declare restored bytes trustworthy.

Carry results and limits into the compatibility matrix. Documentation, a successful
replication job or a booted ISO cannot substitute for hostile-compromise and actual
restore evidence. Retest the affected boundary when credentials, policy, software,
OS, hardware or recovery custody changes.
