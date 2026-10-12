# GitHub roadmap index

Live status, priority and dependencies are in the [shared private Project](https://github.com/users/gonz0w/projects/2).
This index supplies stable links, not a second progress checklist. The [capability roadmap](milestones.md)
retains sequencing; the [GitHub workflow](../engineering/github-workflow.md) defines pickup, triage and acceptance.
Specifications, contracts, ADRs, Test Designs and exact candidate receipts remain authoritative in Git.

## Ownership and phase gates

`gonz0w/horvium` is currently the only Horvium repository. It owns management, native storage,
console/CLI, packaging and acceptance. Future related repositories join this same Project with matching
phase milestones and cross-repository sub-issues. No repository split or universal hardware support is implied.

Existing M0/M1 milestone identities are retained. M2–M7 use Horvium’s original phases; no due dates.
First deployable release spans M1–M6; M7 consumes the tested package for a later image.
Close milestones only after explicit acceptance of the entire scope, independently of component merges.

| Phase | Coordinating acceptance gate | Repository milestone |
|---|---|---|
| M0 Repository foundation | [Gate #129](https://github.com/gonz0w/horvium/issues/129) | [Milestone](https://github.com/gonz0w/horvium/milestone/1) |
| M1 Discovery and design | [Gate #130](https://github.com/gonz0w/horvium/issues/130) | [Milestone](https://github.com/gonz0w/horvium/milestone/2) |
| M2 Storage lifecycle | [Gate #131](https://github.com/gonz0w/horvium/issues/131) | [Milestone](https://github.com/gonz0w/horvium/milestone/3) |
| M3 Sharing and identities | [Gate #132](https://github.com/gonz0w/horvium/issues/132) | [Milestone](https://github.com/gonz0w/horvium/milestone/4) |
| M4 Protection | [Gate #133](https://github.com/gonz0w/horvium/issues/133) | [Milestone](https://github.com/gonz0w/horvium/milestone/5) |
| M5 Port/package | [Gate #134](https://github.com/gonz0w/horvium/issues/134) | [Milestone](https://github.com/gonz0w/horvium/milestone/6) |
| M6 Release qualification | [Gate #135](https://github.com/gonz0w/horvium/issues/135) | [Milestone](https://github.com/gonz0w/horvium/milestone/7) |
| M7 Appliance image | [Gate #136](https://github.com/gonz0w/horvium/issues/136) | [Milestone](https://github.com/gonz0w/horvium/milestone/8) |

## Saved views

| View | Saved filter | Grouping |
|---|---|---|
| [All work](https://github.com/users/gonz0w/projects/2/views/2) | `is:issue` | Phase |
| [Current phase](https://github.com/users/gonz0w/projects/2/views/3) | `phase:M2 is:open` | Repository groups; Status columns |
| [Ready next](https://github.com/users/gonz0w/projects/2/views/4) | `status:Ready is:open` | Repository |
| [Acceptance gates](https://github.com/users/gonz0w/projects/2/views/5) | `is:issue label:"type:tracking","type:acceptance"` | Phase |
| [Completed evidence](https://github.com/users/gonz0w/projects/2/views/6) | `is:closed` | Phase |

The current-phase filter is M2. Triage updates that saved filter deliberately when the active phase changes.
Ready research can proceed independently of blocked delivery gates. Deferred dashboard/security
issues retain proposal status in their Git documents; their phase grouping does not approve implementation.

## Scoped work

Existing trackers #1, #19, #29 and #44 are retained. Completed backfill is scoped to merged
outcomes and their original evidence. Historical failed, skipped, unstarted and unsupported cases
remain in linked PRs and Git receipts. PR 15/17 were integrated through the PR 14/16 branches;
separate trackers distinguish HTTP admission, native-resource recovery, Firefox and file-continuity outcomes.

### M0 Repository foundation

- [M0: eliminate Actions with supervised local qualification (#29)](https://github.com/gonz0w/horvium/issues/29)
- [M0: focus local testing on affected changes (#44)](https://github.com/gonz0w/horvium/issues/44)
- [M0: Bump golang.org/x/sys from 0.48.0 to 0.49.0 (#278)](https://github.com/gonz0w/horvium/issues/278)

### M1 Discovery and design

- [M1: record Safari and manual accessibility acceptance (#137)](https://github.com/gonz0w/horvium/issues/137)
- [M1: design native discovery and the read-only console (#1)](https://github.com/gonz0w/horvium/issues/1)
- [M1: add bounded native FreeBSD inventory (#169)](https://github.com/gonz0w/horvium/issues/169)
- [M1: authenticate bounded FreeBSD inventory transport (#170)](https://github.com/gonz0w/horvium/issues/170)
- [M1: collect and page bounded inventory generations (#171)](https://github.com/gonz0w/horvium/issues/171)
- [M1: add bounded M1 administration security (#172)](https://github.com/gonz0w/horvium/issues/172)
- [M1: add authenticated read-only console and CLI (#173)](https://github.com/gonz0w/horvium/issues/173)
- [M1: bound responses and verify foreground lifecycle (#174)](https://github.com/gonz0w/horvium/issues/174)
- [M1: qualify combined management and helper budgets (#175)](https://github.com/gonz0w/horvium/issues/175)
- [M1: log bounded denial and transport failure events (#176)](https://github.com/gonz0w/horvium/issues/176)
- [M1: terminate descendants after native probe completion (#177)](https://github.com/gonz0w/horvium/issues/177)
- [M1: contain native probes after abrupt helper death (#178)](https://github.com/gonz0w/horvium/issues/178)
- [M1: qualify native resources and integrated helper crash recovery (#179)](https://github.com/gonz0w/horvium/issues/179)
- [M1: bound slow HTTP connections and qualify abuse recovery (#180)](https://github.com/gonz0w/horvium/issues/180)
- [M1: qualify independent SMB and NFS file continuity (#181)](https://github.com/gonz0w/horvium/issues/181)
- [M1: qualify Firefox flows and document manual UI gate (#182)](https://github.com/gonz0w/horvium/issues/182)
- [M1: consolidate qualification and remaining acceptance gates (#183)](https://github.com/gonz0w/horvium/issues/183)

### M2 Storage lifecycle

- [M2: Native executor/recovery mechanism acceptance (#138)](https://github.com/gonz0w/horvium/issues/138)
- [M2: Original startup recovery and shared global launch ownership (#139)](https://github.com/gonz0w/horvium/issues/139)
- [M2: Root property admission and independent workers (#140)](https://github.com/gonz0w/horvium/issues/140)
- [M2: Explicit operator recovery decisions (#141)](https://github.com/gonz0w/horvium/issues/141)
- [M2: Authenticated public property workflows (#142)](https://github.com/gonz0w/horvium/issues/142)
- [M2: Assembled property runtime qualification (#143)](https://github.com/gonz0w/horvium/issues/143)
- [M2: Supported pool/filesystem creation and adoption (#144)](https://github.com/gonz0w/horvium/issues/144)
- [M2: Attributed health and scrub lifecycle (#145)](https://github.com/gonz0w/horvium/issues/145)
- [M2: Redundant replacement and created-pool reimport (#146)](https://github.com/gonz0w/horvium/issues/146)
- [M2: Public storage review/job/recovery workflows (#147)](https://github.com/gonz0w/horvium/issues/147)
- [M2: Complete storage lifecycle acceptance evidence (#148)](https://github.com/gonz0w/horvium/issues/148)
- [M2: review Overview and Storage capacity dashboard packet (#166)](https://github.com/gonz0w/horvium/issues/166) — [authoritative adopted packet](../design/dashboards/PRD.md)
- [Adopt issue-driven Horvium roadmap and contributor workflow (#168)](https://github.com/gonz0w/horvium/issues/168)
- [M2: design safe storage ownership and durable jobs (#19)](https://github.com/gonz0w/horvium/issues/19)
- [M2: add scoped immutable property reviews (#184)](https://github.com/gonz0w/horvium/issues/184)
- [M2: parse bounded native property sources (#185)](https://github.com/gonz0w/horvium/issues/185)
- [M2: bind reviews to dataset identities and version membership (#186)](https://github.com/gonz0w/horvium/issues/186)
- [M2: bind reviews to filesystem mount and encryption state (#187)](https://github.com/gonz0w/horvium/issues/187)
- [M2: bind reviews to pool import identity and policy (#188)](https://github.com/gonz0w/horvium/issues/188)
- [M2: bind reviews to complete pool feature observations (#189)](https://github.com/gonz0w/horvium/issues/189)
- [M2: bind reviews to native vdev topology and health (#190)](https://github.com/gonz0w/horvium/issues/190)
- [M2: qualify owned virtual disk identities and geometry (#191)](https://github.com/gonz0w/horvium/issues/191)
- [M2: correlate native disk identity and geometry (#192)](https://github.com/gonz0w/horvium/issues/192)
- [M2: classify disk attachments and refuse partitions (#193)](https://github.com/gonz0w/horvium/issues/193)
- [M2: correlate root mount and swap disk backing (#194)](https://github.com/gonz0w/horvium/issues/194)
- [M2: correlate imported pools with disk backing (#195)](https://github.com/gonz0w/horvium/issues/195)
- [M2: bind filesystem signatures to disk observations (#196)](https://github.com/gonz0w/horvium/issues/196)
- [M2: retain immutable device-use observations (#197)](https://github.com/gonz0w/horvium/issues/197)
- [M2: bind exported pool labels to disk review (#198)](https://github.com/gonz0w/horvium/issues/198)
- [M2: bind GEOM access counts to disk review (#199)](https://github.com/gonz0w/horvium/issues/199)
- [M2: establish the private state writer boundary (#200)](https://github.com/gonz0w/horvium/issues/200)
- [M2: validate bounded ownership state records (#201)](https://github.com/gonz0w/horvium/issues/201)
- [M2: read bounded ownership records through the state lease (#202)](https://github.com/gonz0w/horvium/issues/202)
- [M2: validate state ACLs through held descriptors (#203)](https://github.com/gonz0w/horvium/issues/203)
- [M2: publish private ownership records atomically (#204)](https://github.com/gonz0w/horvium/issues/204)
- [M2: inspect bounded ownership metadata catalogs (#205)](https://github.com/gonz0w/horvium/issues/205)
- [M2: reserve ownership publication capacity (#206)](https://github.com/gonz0w/horvium/issues/206)
- [M2: validate bounded job lifecycle records (#207)](https://github.com/gonz0w/horvium/issues/207)
- [M2: read private job records and guard successors (#208)](https://github.com/gonz0w/horvium/issues/208)
- [M2: inspect bounded mixed state records (#209)](https://github.com/gonz0w/horvium/issues/209)
- [M2: publish durable jobs with shared capacity guards (#210)](https://github.com/gonz0w/horvium/issues/210)
- [M2: capture bounded reviewed property intent (#211)](https://github.com/gonz0w/horvium/issues/211)
- [M2: bind reviewed property baselines to jobs (#212)](https://github.com/gonz0w/horvium/issues/212)
- [M2: persist reviewed property jobs on FreeBSD (#213)](https://github.com/gonz0w/horvium/issues/213)
- [M2: check duplicate submissions and pool reservations (#214)](https://github.com/gonz0w/horvium/issues/214)
- [M2: inspect trusted FreeBSD job reservations (#215)](https://github.com/gonz0w/horvium/issues/215)
- [M2: publish reserved property submissions (#216)](https://github.com/gonz0w/horvium/issues/216)
- [M2: expose bounded job restart projections (#217)](https://github.com/gonz0w/horvium/issues/217)
- [M2: qualify property publication process-death cuts (#218)](https://github.com/gonz0w/horvium/issues/218)
- [M2: coordinate property review and submission (#219)](https://github.com/gonz0w/horvium/issues/219)
- [M2: collect native property review observations (#220)](https://github.com/gonz0w/horvium/issues/220)
- [M2: bind native observations to root policy (#221)](https://github.com/gonz0w/horvium/issues/221)
- [M2: exclude native system backing from property review (#222)](https://github.com/gonz0w/horvium/issues/222)
- [M2: bind native pool providers for property review (#223)](https://github.com/gonz0w/horvium/issues/223)
- [M2: require explicit property member enrollment (#224)](https://github.com/gonz0w/horvium/issues/224)
- [M2: check native property store readiness (#225)](https://github.com/gonz0w/horvium/issues/225)
- [M2: review explicit property enrollment (#226)](https://github.com/gonz0w/horvium/issues/226)
- [M2: bind enrollment reviews to root state (#227)](https://github.com/gonz0w/horvium/issues/227)
- [M2: collect native enrollment reviews (#228)](https://github.com/gonz0w/horvium/issues/228)
- [M2: publish reviewed enrollment metadata (#229)](https://github.com/gonz0w/horvium/issues/229)
- [M2: publish enrollment from fresh native collection (#230)](https://github.com/gonz0w/horvium/issues/230)
- [M2: inspect retained storage job status (#231)](https://github.com/gonz0w/horvium/issues/231)
- [M2: verify reviewed property postconditions (#232)](https://github.com/gonz0w/horvium/issues/232)
- [M2: assess retained property jobs against observations (#233)](https://github.com/gonz0w/horvium/issues/233)
- [M2: bind native assessment to the root ledger (#234)](https://github.com/gonz0w/horvium/issues/234)
- [M2: assess retained property jobs against native state (#235)](https://github.com/gonz0w/horvium/issues/235)
- [M2: observe native pool health and scan state (#236)](https://github.com/gonz0w/horvium/issues/236)
- [M2: qualify native scrub state observations (#237)](https://github.com/gonz0w/horvium/issues/237)
- [M2: qualify native checksum findings (#238)](https://github.com/gonz0w/horvium/issues/238)
- [M2: sequence durable property execution (#239)](https://github.com/gonz0w/horvium/issues/239)
- [M2: bound native property controls (#240)](https://github.com/gonz0w/horvium/issues/240)
- [M2: hand off original admitted property execution (#241)](https://github.com/gonz0w/horvium/issues/241)
- [M2: bind property readiness to fresh root admissions (#242)](https://github.com/gonz0w/horvium/issues/242)
- [M2: collect native properties through original root claims (#243)](https://github.com/gonz0w/horvium/issues/243)
- [M2: isolate native claim qualification fixtures (#244)](https://github.com/gonz0w/horvium/issues/244)
- [M2: acknowledge applying metadata through original claims (#245)](https://github.com/gonz0w/horvium/issues/245)
- [M2: propagate worker publication uncertainty (#246)](https://github.com/gonz0w/horvium/issues/246)
- [M2: acknowledge property successors through original claims (#247)](https://github.com/gonz0w/horvium/issues/247)
- [M2: bind property controls to original applying claims (#248)](https://github.com/gonz0w/horvium/issues/248)
- [M2: bind execution to original coordinator authority (#249)](https://github.com/gonz0w/horvium/issues/249)
- [M2: compose the private property execution backend (#250)](https://github.com/gonz0w/horvium/issues/250)
- [M2: bind action context to live sessions (#251)](https://github.com/gonz0w/horvium/issues/251)
- [M2: bound application context negotiation (#252)](https://github.com/gonz0w/horvium/issues/252)
- [M2: bind negotiation to original kernel peers (#253)](https://github.com/gonz0w/horvium/issues/253)
- [M2: bound typed property requests (#254)](https://github.com/gonz0w/horvium/issues/254)
- [M2: bound property review and admission replies (#255)](https://github.com/gonz0w/horvium/issues/255)
- [M2: own single property socket exchanges (#256)](https://github.com/gonz0w/horvium/issues/256)
- [M2: bound property connection admission (#257)](https://github.com/gonz0w/horvium/issues/257)
- [M2: bind admitted property exchanges (#258)](https://github.com/gonz0w/horvium/issues/258)
- [M2: bind property clients to live sessions (#259)](https://github.com/gonz0w/horvium/issues/259)
- [M2: own private root property listeners (#260)](https://github.com/gonz0w/horvium/issues/260)
- [M2: dial configured property peers safely (#261)](https://github.com/gonz0w/horvium/issues/261)
- [M2: qualify native original-session property integration (#262)](https://github.com/gonz0w/horvium/issues/262)
- [M2: qualify original root request bindings (#263)](https://github.com/gonz0w/horvium/issues/263)
- [M2: define property runtime recovery and launch gates (#264)](https://github.com/gonz0w/horvium/issues/264)
- [M2: qualify property executor recovery observations (#265)](https://github.com/gonz0w/horvium/issues/265)
- [M2: specify property executor recovery mechanism (#266)](https://github.com/gonz0w/horvium/issues/266)
- [M2: supervise private FreeBSD property executors (#267)](https://github.com/gonz0w/horvium/issues/267)
- [M2: validate execution journal records and census (#268)](https://github.com/gonz0w/horvium/issues/268)
- [M2: observe native execution census under original lease (#269)](https://github.com/gonz0w/horvium/issues/269)
- [M2: persist original execution journal acknowledgements (#270)](https://github.com/gonz0w/horvium/issues/270)
- [M2: bind execution identity metadata and native objects (#271)](https://github.com/gonz0w/horvium/issues/271)
- [M2: discover persistent backing through guarded probes (#272)](https://github.com/gonz0w/horvium/issues/272)
- [M2: observe persistent backing through original leases (#273)](https://github.com/gonz0w/horvium/issues/273)
- [M2: hold original journal and ledger leases together (#274)](https://github.com/gonz0w/horvium/issues/274)
- [M2: validate original execution-domain system backing (#275)](https://github.com/gonz0w/horvium/issues/275)
- [M2: distinguish publication allocation accounting (#276)](https://github.com/gonz0w/horvium/issues/276)
- [M2: qualify independent ledger restoration (#277)](https://github.com/gonz0w/horvium/issues/277)
- [M2: add original journaled metadata publication (#279)](https://github.com/gonz0w/horvium/issues/279)
- [M2: add exclusive fresh execution domains (#280)](https://github.com/gonz0w/horvium/issues/280)
- [M2: initialize the journal from original fresh domains (#281)](https://github.com/gonz0w/horvium/issues/281)
- [M2: connect journal activation to fixed FreeBSD collectors (#282)](https://github.com/gonz0w/horvium/issues/282)
- [M2: durably close original idle journal activations (#283)](https://github.com/gonz0w/horvium/issues/283)
- [M2: register one original commissioning owner per helper (#284)](https://github.com/gonz0w/horvium/issues/284)
- [M2: retain original closed-incarnation startup review (#285)](https://github.com/gonz0w/horvium/issues/285)
- [M2: publish an original closed-startup active incarnation (#286)](https://github.com/gonz0w/horvium/issues/286)
- [M2: close an original unused restarted lifetime (#287)](https://github.com/gonz0w/horvium/issues/287)
- [M2: define the next retained-history startup recovery slice (#288)](https://github.com/gonz0w/horvium/issues/288)
- [M2: deliver the qualified Overview dashboard (#289)](https://github.com/gonz0w/horvium/issues/289)
- [M2: add generation-bound Overview pool comparisons (#354)](https://github.com/gonz0w/horvium/issues/354) — [slice contract/evidence](../testing/dashboards/overview-capacity.md)
- [M2: review current-dashboard resource qualification Test Design (#356)](https://github.com/gonz0w/horvium/issues/356) — [proposed supplement](../design/dashboards/resource-qualification.md)
- [M2: deliver the qualified Storage capacity dashboard (#290)](https://github.com/gonz0w/horvium/issues/290)
- [M2: add bounded pool-page capacity comparisons (#350)](https://github.com/gonz0w/horvium/issues/350) — [slice contract/evidence](../testing/dashboards/pool-capacity.md)
- [M2: add bounded dataset-page Used comparisons (#352)](https://github.com/gonz0w/horvium/issues/352) — [slice contract/evidence](../testing/dashboards/dataset-capacity.md)
- [M2: deliver the qualified Storage health dashboard (#292)](https://github.com/gonz0w/horvium/issues/292)
- [M2: deliver the qualified Jobs dashboard (#295)](https://github.com/gonz0w/horvium/issues/295)

### M3 Sharing and identities

- [M3: Review sharing and local identity contracts (#149)](https://github.com/gonz0w/horvium/issues/149)
- [M3: Local identity and permission lifecycle (#150)](https://github.com/gonz0w/horvium/issues/150)
- [M3: Validated SMB configuration and client access (#151)](https://github.com/gonz0w/horvium/issues/151)
- [M3: Validated native NFS configuration and client access (#152)](https://github.com/gonz0w/horvium/issues/152)
- [M3: Scoped SSH/SFTP file access (#153)](https://github.com/gonz0w/horvium/issues/153)
- [M3: deliver the qualified Shares dashboard (#293)](https://github.com/gonz0w/horvium/issues/293)

### M4 Protection

- [M4: Review protection and backup-peer contracts (#154)](https://github.com/gonz0w/horvium/issues/154)
- [M4: Bounded snapshot scheduling and retention (#155)](https://github.com/gonz0w/horvium/issues/155)
- [M4: Explicit backup-peer trust and constrained receive (#156)](https://github.com/gonz0w/horvium/issues/156)
- [M4: Incremental replication and interruption recovery (#157)](https://github.com/gonz0w/horvium/issues/157)
- [M4: Independent backup history and replacement-primary restore (#158)](https://github.com/gonz0w/horvium/issues/158)
- [M4: deliver the qualified Protection dashboard (#294)](https://github.com/gonz0w/horvium/issues/294)

### M5 Port/package

- [M5: Review native package and lifecycle contracts (#159)](https://github.com/gonz0w/horvium/issues/159)
- [M5: Reproducible FreeBSD port and package (#160)](https://github.com/gonz0w/horvium/issues/160)
- [M5: Install upgrade and removal preserve operator state (#161)](https://github.com/gonz0w/horvium/issues/161)

### M6 Release qualification

- [M6: Qualify small and representative larger hardware (#162)](https://github.com/gonz0w/horvium/issues/162)
- [M6: Qualify end-to-end recovery upgrade and security runbooks (#163)](https://github.com/gonz0w/horvium/issues/163)
- [M6: deliver the qualified Storage performance dashboard (#291)](https://github.com/gonz0w/horvium/issues/291)
- [M6: deliver the qualified System dashboard (#296)](https://github.com/gonz0w/horvium/issues/296)

### M7 Appliance image

- [M7: Review package-based appliance image and host policy (#164)](https://github.com/gonz0w/horvium/issues/164)
- [M7: Qualify appliance boot install update and recovery (#165)](https://github.com/gonz0w/horvium/issues/165)
- [M7: review the second-release security threat model (#167)](https://github.com/gonz0w/horvium/issues/167)
- [Proposal: Scoped administration after credential theft (#297)](https://github.com/gonz0w/horvium/issues/297)
- [Proposal: Independent protected backup history (#298)](https://github.com/gonz0w/horvium/issues/298)
- [Proposal: Encryption replication and recovery-key custody (#299)](https://github.com/gonz0w/horvium/issues/299)
- [Proposal: Bounded FreeBSD service confinement (#300)](https://github.com/gonz0w/horvium/issues/300)
- [Proposal: Honest protection posture and recovery observations (#301)](https://github.com/gonz0w/horvium/issues/301)
- [Proposal: Deliberate irreversible trust key and retention changes (#302)](https://github.com/gonz0w/horvium/issues/302)
- [Proposal: Clean-host recovery after primary compromise (#303)](https://github.com/gonz0w/horvium/issues/303)

- [M2: review cumulative native lifecycle fixture budgets (#318)](https://github.com/gonz0w/horvium/issues/318): separate design prerequisite for budget-changing work in issue 305; authoritative [proposal](../milestones/M2/retained-policy-fixture-budgets.md).

[M2: review original startup deadline evidence and stack diagnostic contract (#321)](https://github.com/gonz0w/horvium/issues/321)
is a separately reviewed research prerequisite for unfinished issue 305, under
parent 139. The [analysis and proposed method](../research/M2/policy-startup-deadline.md)
remain distinct from implementation and native acceptance.

[M2: capture bounded original startup stack diagnostics (#323)](https://github.com/gonz0w/horvium/issues/323)
owns the [adopted sampler implementation/evidence](../research/M2/policy-startup-stack.md),
under parent 139 with satisfied prerequisite 321. Full lifecycle acceptance is
separate under issue 305; the [corrected source-bound diagnostic receipt](../research/M2/policy-startup-stack-qualification-manifest.json)
records point observations and publication uncertainty, not lifecycle acceptance.
GitHub owns live status.

[M2: review original startup inspections and caller envelope (#326)](https://github.com/gonz0w/horvium/issues/326)
owns the [offline source review and proposed separate experiment](../research/M2/policy-startup-inspection.md),
under parent 139 with completed diagnostic 323 as prerequisite. Proposal adoption,
fixture implementation and full lifecycle acceptance under 305 remain separate.
GitHub owns the live decision and status.

[M2: qualify separate startup caller-envelope diagnostic (#328)](https://github.com/gonz0w/horvium/issues/328)
owns the [implementation/candidate evidence](../research/M2/policy-startup-envelope.md),
under parent 139 after adopted review 326. Completeness is distinct from original
startup outcome and full lifecycle acceptance under 305. GitHub owns live status.

## Migration receipt

[M2: review original lifecycle startup caller composition (#331)](https://github.com/gonz0w/horvium/issues/331)
owns the [offline review and fixture-only proposal](../research/M2/policy-lifecycle-caller.md),
under parent 139 after completed prerequisite 328. PR 332 adopted its fixture-only amendment; the unchanged-profile second-cycle
failure and [implementation/qualification under 305](../testing/M2/retained-policy-lifecycle-caller.md)
remain distinct. GitHub owns live status and the review decision.

See the [dated verification record](github-migration.md) for counts, preserved identities,
evidence and recoverability. Consult GitHub for live state rather than treating that receipt as a board.

The [focused original startup operation timeout investigation](../research/M2/policy-startup-operation-timeout.md)
reuses the adopted separate diagnostic under [owner #305](https://github.com/gonz0w/horvium/issues/305).
Its source review and candidate evidence remain separate from full lifecycle
acceptance; GitHub owns live status and priority.

[Startup deadline provenance design (#335)](https://github.com/gonz0w/horvium/issues/335)
owns the [fixture-only method/Test Design](../research/M2/policy-startup-timeout-provenance.md),
adopted by merged PR 336 for [owner #305](https://github.com/gonz0w/horvium/issues/305)
under [parent #139](https://github.com/gonz0w/horvium/issues/139). The
[offline replay](../research/M2/policy-startup-timeout-provenance-manifest.json)
preserves prior candidate limits. Implementation, native diagnostic completeness
and both full lifecycle release gates remain separate; GitHub owns live status.

[Portable deadline adapter (#337)](https://github.com/gonz0w/horvium/issues/337)
owns the [bounded annotation/extraction prerequisite](../testing/M2/deadline-provenance-core.md).
It is a native sub-issue of #305 with adopted decision #335 as its prerequisite.
Original-return instrumentation, the separate strict 50-control fixture/consumer
and one fresh diagnostic remain distinct work; portable passes do not satisfy them.

[Original deadline return sites (#339)](https://github.com/gonz0w/horvium/issues/339)
owns the [bounded integration](../testing/M2/deadline-return-sites.md), under #305
after merged adapter #337. Portable placement/source review and cross-builds do
not accept the separate strict fixture, native diagnostic or lifecycle gates.

[Strict deadline consumer (#341)](https://github.com/gonz0w/horvium/issues/341)
owns the [offline transcript contract and parser](../testing/M2/deadline-provenance-consumer.md),
under #305 after merged return-site #339. Native fixture/runner and the one fresh
diagnostic remain separate; GitHub owns live status.

[Deadline runner input binding (#343)](https://github.com/gonz0w/horvium/issues/343)
owns the [offline candidate verification boundary](../testing/M2/deadline-provenance-candidate.md),
under #305 after merged strict consumer #341. File/source identity does not
complete build attestation, tagged fixture, runner integration or native
execution; GitHub owns live status and dependencies.

[Tagged deadline fixture/runner (#345)](https://github.com/gonz0w/horvium/issues/345)
owns the [50-control implementation and isolated build prerequisite](../testing/M2/deadline-provenance-fixture.md),
under #305 after merged input binding #343. Offline implementation and cross-build
receipts do not complete the one adopted native diagnostic or lifecycle gates.
GitHub owns live status, priority and native dependencies.

[One return-path diagnostic (#347)](https://github.com/gonz0w/horvium/issues/347)
owns the [single-candidate evidence record](../research/M2/policy-startup-return-path.md),
under #305 after merged fixture #345. Recording a complete or unsuccessful
diagnostic does not accept either full lifecycle release gate. GitHub owns live
status, priority and native dependencies.
