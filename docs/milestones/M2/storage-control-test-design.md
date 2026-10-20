# Explicit storage-control acceptance plan

Status: design proposal, 2026-10-20; no new tests executed by this document.
Authority: [operator contract](storage-control-contract.md), [M2 Test Design](TDD.md),
M2-RUNTIME-002, M2-OWN-001, M2-STATE-001, M2-RECOVERY-001 and M2-PRIV-001.
Refs [#139](https://github.com/gonz0w/horvium/issues/139) and
[#305](https://github.com/gonz0w/horvium/issues/305); closes neither.

Later implementation must exercise the real selected operator consumer together
with its owned dependencies, rather than test a supplied clearance boolean. Add
failing-first independent literal input/output and barrier tests. Synthetic source
transfers prove only those transfers; native authority cannot be mocked into a pass.

| Portable family | Independently observable requirement |
|---|---|
| SC-P01 exact selection | Each of three actions accepts exactly one explicit config argument; duplicate/unknown/abbreviated flags, secret/force/retry overrides and positional extras fail before any domain callback. |
| SC-P02 exact bytes | Literal valid four-field v1 input; escaped duplicate keys, invalid UTF-8/BOM, null/types/nesting, trailing document, 16-KiB+1 and path boundaries reject; no environment/default discovery. |
| SC-P03 branch isolation | Fresh with either existing leaf refuses without create; startup or inspect with either missing half refuses without create; inspect never invokes create/activate/finish or supplies a freshness receipt. |
| SC-P04 complete history | Empty and complete policies-only candidates remain distinct; every job format/state, other-pool/terminal job, missing member, partial census and final-policy fault deny; no policy/hold is omitted or removed. |
| SC-P05 original ownership | Copied/decoded/reopened owners and second or alternate-path attempts deny; entered failure and successful release preserve consumed registration; cancellation before entry consumes nothing. |
| SC-P06 retirement and status | Hold original observe/finish/release across concurrent calls; first signal/cancellation initiates exactly one bounded cleanup context, separate from the cancelled foreground context. Repeated signals cannot retry or extend it. Failed/expired finish and release suppress clean completion. Busy never reports healthy; active metadata never proves a running process. |
| SC-P07 output and failure | Exact literal result keys/states/false writes flag, 4-KiB bound and fixed failure classes; raw config/secret/command sentinels absent. A blocked/failing writer is called only after original finish/release has completed, receives no live owner and cannot trigger another effect. No startup output can block foreground shutdown. |
| SC-P08 unchanged consumers | Exact M1 v1 rejects new storage fields; serve/helper/inventory/credential paths remain unchanged. No new native launch, pool enrollment, config reload, service installation or automatic restart. |

| Native family, later on both pinned disposable FreeBSD/amd64 targets | Required actual evidence |
|---|---|
| SC-N01 config custody | Real/effective root; UID drop with inherited descriptors; no-follow/type/link/mode/ACL/ancestor/object drift and retained selection replacement refusal. |
| SC-N02 fresh creation | Both original names absent before first effect, journal-first exclusive creation and all file/directory/parent syncs; every cut preserves partial objects and original release. |
| SC-N03 configured startup | Both original namespaces/locks, exact binding/current raw boot/full census and system/backing/provider observations; missing/foreign/aliased/unsupported inputs refuse without repair. |
| SC-N04 restore separation | Qualified UFS file-only and ZFS dataset scope; ledger rollback preserves independent disarmed journal. Joint rollback never becomes automatic clearance. |
| SC-N05 owner contention | Competing fresh/start/inspect processes, alternative paths, same-process copies and root loss; original leases retained to final release, no saved-PID authority or second writer. |
| SC-N06 exact lifecycle | Original active/closed successors, generation headroom, finish/release failure, cancelled or lost acknowledgement and post-publication drift; no clean-marker fabrication. |
| SC-N07 policies-only gate | Complete same-candidate #305 both-release vectors, including retained full controls/children/scalars and fault families, before nonempty configured activation is registered. Preserve exhausted/failed diagnostics. |
| SC-N08 assembled preservation | Original configured command and exact executable; preserve independent file-daemon/data sentinels, every job/policy/hold and all unrelated host/device/base inputs; own bounded fixtures and independently verify teardown. |

Keep original operation/count/size/deadline limits. Pin the implementation,
compiled artifacts, supported release/kernel/userland and every fixture before
any separately authorized execution. Full #139/global/backend, real storage
effects and recovery/client/browser/resource/release acceptance remain separate.
This design-only PR runs existing proportional repository review; it does not
create an execution packet, new harness, native attempt or fabricated result.
