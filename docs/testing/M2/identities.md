# M2 dataset identity binding

Date: 2026-10-04. Status: behavioral implementation; pinned native execution pending.
Design: [approved M2 Test Design](../../milestones/M2/TDD.md),
[slice sequence](../../milestones/M2/implementation.md).

## Boundary and fixed projection

The property parser cannot detect dropping every row of a filesystem. This slice
adds an independent complete dataset identity/type table, required by the internal
property-review model. The runtime remains read-only; no helper collector,
eligibility proof, enrollment, submission or native executor is connected.

```text
/sbin/zfs list -H -p -r -t filesystem,volume,snapshot,bookmark -o name,guid,type <pool>
```

`ParseIdentities` requires exactly three nonempty tab-separated fields per
newline-terminated record, canonical positive uint64 GUIDs, known types, unique
names, a filesystem pool root and complete native owner/parent relationships.
Filesystem and volume GUIDs must be unique. Snapshot and bookmark GUIDs can
repeat: a received snapshot preserves the source GUID, and a bookmark can refer
to that snapshot. Bind each version to its local owning dataset GUID as well as
its type/name/GUID. Observation never grants snapshot or bookmark mutation scope.

Names use the existing conservative management subset. Snapshot/bookmark names
use exactly `@`/`#` and a 1–64 byte ASCII suffix starting with a letter or digit,
then letters/digits/underscore/hyphen. Unsupported native names deny this planner;
M1 observation remains separate. Input bounds are 64 MiB, 100,000 identity records
and 4,096 bytes per field. Combined policy/property/identity preflight accounting
and the existing 64 KiB review limit apply, without truncation or paging impact.

`VerifyFilesystemIdentities` validates exact filesystem name/GUID agreement with
the independently parsed property table. Missing whole filesystems and reused
names fail. Volumes are classified but conservatively refuse pool qualification:
volume impacts are not represented in this filesystem-only slice. This does not
qualify mixed pools or claim volume management support.

A future trusted collector must check bounded capture, successful command exit,
full type coverage and independent identity enumerations before and after
property reads. `VerifyIdentityStability` compares normalized complete tables,
including version membership and ownership. It detects observed drift; it cannot
provide an atomic fence or detect a change that appears and disappears between
reads. Final native checks, exclusive writer rules, eligibility and external-root
race handling remain unimplemented obligations.

## Review drift and tests

Reviews retain an owned, sorted version slice for target, ancestors and the
entire target subtree. Adding/removing/renaming/reusing a version or changing its
owner invalidates revalidation. Unrelated sibling versions remain outside that
review. The selected filesystem scopes and property permissions remain explicit.

Tests were observed failing against deny-only stubs before implementation.
`TestPropertyIdentityCoverageCannotOmitWholeFilesystem`, version ownership/drift,
volume refusal, ordering and budget tests cover subsets of M2-OWN-001,
M2-DRIFT-001, M2-PLAN-001, M2-PROPERTY-001 and M2-SCALE-001. `FuzzIdentities`
joins the bounded fuzz gate. The evidence checker requires all original property
assertions plus nine new native identity assertions and executed teardown; unit
pass, boot, skip or incomplete markers cannot qualify.

The opt-in owned-fixture test extends the previous property procedure with
snapshot/bookmark correlation, replicated GUIDs, snapshot rename/name reuse,
complete filesystem coverage, volume classification/refusal and stable reads.
It creates a sparse volume with `volmode=none` only inside its declared pool,
then removes that owned fixture. No operator disks, credentials or inventory are
inputs. These setup commands are test code, not product storage operations.

## Qualification

Native execution and its sanitized manifest will be recorded here when both
pinned FreeBSD 15.1/14.5 guests pass. Prior property results remain separately
preserved in [the property record](properties.md). Full slice 1, every complete
M2 Test Design row, state/recovery/privilege/mutation/resource/browser gates and
hardware/commercial support remain pending. Manual M1 accessibility remains
pending by maintainer choice.
