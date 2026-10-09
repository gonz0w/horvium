# Execution journal record and census core

Status: Implemented portable prerequisite; review evidence pending; runtime denied.
Date: 2026-10-08. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19).
[Accepted mechanism](../../milestones/M2/property-recovery-mechanism.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).

PR 107 accepted the journal obligations at merged source
`02a995e5681f38b4a40141402dd2d0a9f0244933`; PR 108 accepted the original native
boot/reaper consumer at `a8a96403b40922b4751493d837f5a6d37b1f5e0e`.
This is the internal portable record, successor validator and streaming census
consumer for the next integrated journal owner. It creates no files, selects no
installation path, reads no native identity and issues no write/restart authority.
Native journal persistence, provisioning, startup/global ownership and the real
publisher/backend consumers remain required together before runtime use.

## Version 1 record

The private fixed-depth JSON shape is independent of ownership/job formats.
Every top-level/nested field is required and nonnullable; absent pending state
uses `pending:false` and empty strings, never a missing object or null. Encoding
produces canonical JSON. Decoding accepts harmless JSON presentation differences
but refuses duplicate, unknown, case-variant or missing fields, invalid UTF-8,
trailing objects, noncanonical unsigned numbers, unsupported versions and damage.
All errors return zero metadata. One record is at most 64 KiB; caller-owned fields
are bounded before marshalling. There are no inventories, arrays or copied histories.

| Top-level field | Required representation and meaning |
|---|---|
| `version`, `kind` | Exactly `1`, `execution-journal`; separate from all existing record domains |
| `generation` | Nonzero uint64; an ordinary successor advances exactly one; overflow denies |
| `binding` | `installation`: 32 lowercase hex characters; `directory`, `lock`, `ledger_backing`, `journal_backing`: 64-character digests of the future independently verified bindings |
| `owner` | `id`, `boot`: 32 lowercase hex characters; `state`: `active` or `closed` |
| `census` | 64-character complete exact-byte ledger census digest |
| `metadata` | `pending`, `before`, `after`, `name`, `predecessor`, `successor`; pending binds the confirmed census to a different expected census and one exact ownership/job record successor |
| `native` | `pending`, `job`, `submission`, `pool_guid`, `filesystem_guid`, `intent`, `owner`, `boot`, `supervisor`; pending keeps the original execution identity distinct from a metadata transition |
| `disarmed` | Boolean restore barrier; ordinary successors cannot clear it |
| `checksum` | SHA-256 of `horvium:execution-journal:v1` followed by newline and canonical content without the checksum field |

Identity hex encodes sixteen raw bytes, preserving leading/trailing zeros. The
codec does not generate fresh random identities or turn caller-supplied bytes
into observed `kern.boot_id`. Directory/lock/backing digest producers and original
native object validation remain unimplemented; digest syntax is not provenance.
No persisted identifier, lifecycle label or checksum is a live capability.

Pending metadata names use only the already accepted `ownership-<GUID>.json` and
`job-<digest>.json` namespace. Empty predecessor represents first publication;
otherwise it is a content digest distinct from the required successor digest.
A pending transition's before census must equal the confirmed census. Pending
native identifiers/GUIDs/digests are bounded and canonical. Historical native
pending state may retain an older helper/boot after recovery inspection; decoding
it never changes the current owner or permits ordinary native clearance. Closed
records cannot contain either pending domain or a restore barrier.

## Successor checks

The pure validator accepts exactly one logical transition candidate: set metadata
pending; confirm its exact expected census and clear only metadata pending; set
native pending for the same helper/boot; clear native pending for that same
incarnation when no metadata transition is pending; close an otherwise idle owner;
or disarm while retaining all existing facts. Bindings and original owner/boot
remain immutable. Metadata confirmation preserves unresolved native execution.
Simultaneous confirmation/native clearance, pending replacement, no-op generation
bumps, generation gaps, ordinary rearming/reopening and historical native clearance
are denied. A failed check changes neither input.

Successful validation is **not** an acknowledged transition. In particular, the
native-clear candidate cannot prove applying/outcome publication, command
completion or a postcondition. The eventual original dual-lease owner must prove
those facts and the accepted journal/publication ordering before publishing any
candidate. These checks perform no automatic recovery, replay or hold release.

## Complete census consumer

The private single-use consumer classifies the complete supplied name set with
the existing mixed catalog rules, including `writer.lock`. It rejects unknown,
duplicate, missing and temporary entries, and retains existing ownership/job/
terminal/nonterminal/64 MiB byte limits. Each record must arrive in sorted order
and pass the existing complete bounded decoder, GUID/job selector and submission
uniqueness checks before hashing. It retains bounded summaries and hashes,
rather than record bodies or job histories.

Framing is domain-separated SHA-256: `horvium:execution-census:v1` plus newline,
uint64 big-endian length and canonical binding JSON, uint64 entry count including
the lock, then sorted entries. Each entry contributes uint64 name length, exact
name bytes, uint64 content length and the raw 32-byte SHA-256 of exact content.
The accepted ownership/job names sort before `writer.lock`; its empty content
and the independently bound lock digest are included even for an empty ledger.
Record whitespace or other checksum-valid byte changes alter the census; the
consumer does not normalize stored bytes away.

A partial/failed/out-of-order/over-limit consumer returns no digest and cannot
resume. Copies cannot mutate or retire the original. Successful finish is one-time.
The caller owns inputs throughout calls. This component consumes supplied bytes;
it does not enumerate a real filesystem or attest completeness. Its future native
reader must stream through the original held writer, validate every opened object,
recheck namespace/objects and derive the actual binding before using its result.

## Tests and remaining gates

Working-tree behavioral red results preceded both implementations. Ten new
portable parents/235 child results cover canonical golden bytes, every required
field, strict shape/type/damage, separate pending domains, close/disarm, exact
64 KiB/overflow limits, immutable single-step transitions, historical owner denial,
exact census framing/content/binding, partial/error/copy retirement and a real
in-memory census-to-journal consumer. Two bounded fuzz targets exercise hostile
record bytes and name/content inputs. FreeBSD compilation is explicitly unexecuted;
no new native or production claim follows from it.

Affected-package review, signed source and teardown evidence are recorded after
execution. The native independent journal domain, root commissioning, durable
sync/rename/census ordering, helper/process-death and ledger-only restore cuts,
original startup leases, global launch owner and backend/publisher integration
still need implementation and qualification on both disposable releases.
Active kernel property interruption and assembled runtime/resource/client/UI
acceptance remain open. Runtime root operations remain denied; M1 manual
accessibility stays pending. No host service or background process is installed.
