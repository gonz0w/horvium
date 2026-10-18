# Security threat-model review record

Date: 2026-10-18. Scope: documentation/source review for
[#167](https://github.com/gonz0w/horvium/issues/167).
Candidate: [whole-system threat model](threat-model.md).
Author/source reviewer: Codex, for maintainer review by Cam (`@gonz0w`).
Source baseline: `546487832a7f76f679334dee75b29eccdd93a171`.
This is not independent third-party security approval, design adoption or runtime
qualification. GitHub owns live status. Scope ends at this review packet and the
existing seven future trackers; no attack experiment or native rerun is included.

## Questions and findings

| Review question | Recorded finding |
|---|---|
| Is the attacker authority explicit? | Nine positions distinguish unauthenticated/file/session/service, primary root, replication, backup root, compromised admin/recovery endpoint and physical/supply-chain authority; combinations are modeled. |
| Are confidentiality and recovery conflated? | Seven asset groups separate live plaintext, retained history, identity/ACLs, secrets, execution state, evidence and software trust. Authorized plaintext access and poisoned new backups remain explicit losses. |
| Does a second NAS establish independence? | Eight boundaries and the target-flow diagram expose shared administration, credentials, recovery, update and BMC paths; physical distance alone supplies no authority boundary. |
| Are proposed controls mistaken for current support? | Current loopback/session/inventory assembly is source-linked; future sharing/replication/keys/confinement/image outcomes remain proposed. No production authentication or integrated writer claim is added. |
| Is every proposed outcome reviewable? | All seven IDs have a named existing owner, attacker/boundary coverage, abuse/recovery scenarios and one bounded first design/research packet. |
| Are denied effects independently checked? | The scenario catalog requires original principals, positive controls, protected-state observations and declared sentinels; a denied request or a primary success report alone is insufficient. |
| Are recovery and availability first-class? | Capacity/offline, delayed corruption, failed newest copy, key/factor loss, interrupted updates and clean-host/workstation loss have explicit future evidence. |
| Does this bypass first-release work? | The review maps M3/M4/M5/M6/M7 responsibilities and preserves the package-first path, accepted local-identity scope, M2 runtime denial and all native/manual/resource gates. |

## Source inspection

These observations describe the baseline implementation/design, not new tests.
Links remain authoritative in Git; future implementation must inspect its own
actual candidate rather than inherit this source review as support evidence.

| Source | Inspected boundary and limit |
|---|---|
| [Charter](../project/charter.md), [architecture](../architecture/baseline.md), [ADR 0001](../adr/0001-freebsd-and-hardware-scope.md) | Dedicated generic FreeBSD NAS; reference hardware is not the platform contract; file traffic remains independent of management. |
| [ADR 0002](../adr/0002-distribution-and-license.md), [ADR 0003](../adr/0003-peer-and-identity-scope.md) | BSD-3-Clause, package-first, backup-peer/local-identity first release; no fleet/failover/AD requirement. |
| [ADR 0005](../adr/0005-m1-access-and-state.md), [M1 EDD](../milestones/M1/EDD.md) | Lab loopback/SSH token access and bounded sessions; production remote-admin, factor and recovery choices remain separate. |
| [Config.Validate](../../internal/admin/config.go) | Requires validated loopback listener and non-root management identity; this is not a remote HTTPS commissioning design. |
| [Auth](../../internal/admin/auth.go), [SessionAuthority](../../internal/admin/session_authority.go) | Bounded volatile sessions and live-session checks; the type explicitly does not prove root policy, recovery clearance or mutation permission. |
| [service.Helper/Serve](../../internal/service/service.go), [inventory helper](../../internal/helper/server.go) | Root inventory server and unprivileged foreground manager; session checks and Unix peer checks are separate boundaries. Private primitives do not prove integrated write enablement. |
| [M2 runtime contract](../milestones/M2/property-runtime.md), [recovery mechanism](../milestones/M2/property-recovery-mechanism.md) | Existing storage ownership/recovery/launch obligations cannot be replaced by a future security factor or restored configuration. |
| [Supply-chain policy](supply-chain.md), [deployment guide](deployment-security.md), [roadmap](../roadmap/v2-security.md) | No paid tracking/Actions dependency, no qualified hardened image or universal compromise claim; software/image/operator responsibilities remain distinct. |
| [Latest publication/idle record](../testing/dashboards/witness-metadata-lifecycle.md) | 15.1 functional evidence, failed 14.5 preparation and sampled CPU uncertainty retain their original scope; this review supplies no new native execution. |

## Requirement traceability

Outcome labels are the existing **proposed** V2-SEC IDs. TM scenario IDs name
analysis cases only; they are not accepted implementation tests.

| Proposed ID / existing owner | Authority/boundary focus | Scenario coverage | Review result |
|---|---|---|---|
| V2-SEC-001 / [#297](https://github.com/gonz0w/horvium/issues/297) | Session/approving endpoint; B2/B3/B8 | TM-03/04/12 | Exact-action and recovery authority packet identified; factor mechanism undecided. |
| V2-SEC-002 / [#298](https://github.com/gonz0w/horvium/issues/298) | File/primary-root/replication/receiver-root; B1/B4/B5/B6 | TM-02/05/06/07/08/12/14 | Receiver history, capacity and reserve packet identified; no protection window or privilege set selected. |
| V2-SEC-003 / [#299](https://github.com/gonz0w/horvium/issues/299) | Primary/receiver/key custodian; B4/B6/B7 | TM-07/08/09/14 | Stream/key/recovery packet identified; version/key-policy adoption pending. |
| V2-SEC-004 / [#300](https://github.com/gonz0w/horvium/issues/300) | Network/service/stronger host privilege; B1/B2/B3/B7 | TM-01/02/10/13 | One-service authority inventory and alternative spike identified; no confinement chosen. |
| V2-SEC-005 / [#301](https://github.com/gonz0w/horvium/issues/301) | Falsified primary/lost observer; B7/B8 | TM-01/06/11/13/14 | One-claim evidence/freshness/redaction packet identified; no security score or new telemetry contract. |
| V2-SEC-006 / [#302](https://github.com/gonz0w/horvium/issues/302) | Stolen/exceptional authority and root bypass; B2/B3/B4/B5/B8 | TM-03/04/05/07/12 | Independently enforced exceptional-action packet identified; delays are not independent authorization. |
| V2-SEC-007 / [#303](https://github.com/gonz0w/horvium/issues/303) | Lost/compromised primary/workstation/receiver; B1/B6/B7/B8 | TM-02/08/09/13/14 | Clean-host checkpoint selection and restored file/identity/service exercise specified for future design. |

Review disposition: no missing proposed outcome or owner in this packet. Major
future decisions remain open rather than defaulting to an unreviewed mechanism.
The proposed target boundaries are not accepted risk treatment, implementation
permission, or evidence that an attack would currently be contained.

## Primary references and evidence limits

Reviewed on 2026-10-18. Upstream mechanisms are references, not claims about the
versions bundled by a future FreeBSD package/image. Future native packets must
pin actual OS/ZFS/service versions and recheck each selected mechanism.

- [FreeBSD security handbook](https://docs.freebsd.org/en/books/handbook/security/): capability facilities are available for evaluation; integration and service authority need their own evidence.
- [OpenZFS 2.3 delegation manual](https://openzfs.github.io/openzfs-docs/man/v2.3/8/zfs-allow.8.html): permissions and descendant scope motivate an explicit receive/delegation contract.
- [OpenZFS 2.3 receive manual](https://openzfs.github.io/openzfs-docs/man/v2.3/8/zfs-receive.8.html): destructive rollback/receive behavior motivates hostile receive-option and protected-history scenarios.
- [OpenZFS 2.3 send manual](https://openzfs.github.io/openzfs-docs/man/v2.3/8/zfs-send.8.html): raw/non-raw encryption distinctions motivate key/stream and metadata-disclosure tests.
- [CISA StopRansomware guide](https://www.cisa.gov/stopransomware/ransomware-guide): offline backup and tested recovery support the reserve/restoration direction. Direct page and [official guide PDF](https://www.cisa.gov/sites/default/files/2023-10/StopRansomware-Guide-508C-v3_1.pdf) fetches were unavailable/forbidden; only official search-index excerpts supplied this limited reference. No full-guide review or latest-version assertion is made. No incident-response procedure is adopted here.

No security test or native experiment was run for this review. Repository checks
validate documentation integrity and normal source health, not hostile attack
resistance. No existing native result, failed preparation, skipped case, support
matrix or manual gate is upgraded by document coverage. Source-bound local review
and preservation verification accompany the PR; maintainers decide research
acceptance through review/merge. First-release delivery and the seven feature
packets still require their own approved designs and actual acceptance evidence.
