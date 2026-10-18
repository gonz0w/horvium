# Offline broker byte-budget evidence

Status: implementation for review; source-bound replay results pending.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[accepted EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-009.

The [aggregate archive](witness-archive-budget.md) and
[single-original raw](witness-raw-budget.md) controls retain original native files
and source-bound receipts. Their original readback used private development
scripts. The repository-owned [budget validator](../../../tools/witness-budget-evidence.py)
makes both consistency checks repeatable without a guest, root access, archive
extraction or the original private scripts. The existing
[partial-storage validator](witness-storage-readback.md) and its smaller bounds
remain unchanged. Neither tool runs in Horvium's FreeBSD runtime.

## Independent inputs

```sh
.tools/venv/bin/python3 tools/witness-budget-evidence.py /path/to/witness-originals.tar /path/to/expectation.json
```

The expectation JSON has exactly these fields:

| Field | Independent source |
| --- | --- |
| `schema` | Integer `1` for this tool |
| `profile` | `raw-limit` or `archive-limit`, deliberately selected |
| `source_commit`, `base_commit`, `pr_title` | Original signed clean review and host build |
| `release` | Original pinned preflight, `15.1` or `14.5` |
| `source_archive_sha256`, `compiler_sha256` | Verified original candidate build |
| `fixture_sha256`, `management_sha256` | Original release's test and product executables |
| `probe_sha256` | Independently retained framed native ABI/provenance receipt |
| `archive_sha256` | Independently retained full original tar digest |

Use the reviewed immutable native manifest and separately preserved host records;
do not derive expectations from the archive being tested. Every hash is full
lowercase SHA-256; source/base are distinct full lowercase 40-character commit IDs.
Supply a separate expectation file per profile, release and actual candidate.
The tool checks syntax and receipt consistency, not the authority of those inputs.
Changing both an archive and its trusted expectations breaks that external trust.

The archive contains no independent source/release attestation. Git signatures,
compiler/build execution, kernel ABI support, supervision, preflight, live socket/
process identity and original host teardown remain separate native/host gates.
Opaque ABI/header provenance is preserved and hashed, not reinterpreted as proof
of kernel support. Offline cleanup and publication checks verify recorded receipts;
they are not new observations of a running guest or its management API.

## Checked scope

Both profiles require normal product-helper originals, actual captured bytes,
sealed object/size/hash/mode references, matching delivered indexes and a recorded
publication ID. Framing, correlation, record kinds, end count and deadline/forward
receipts are checked. The full independent Go protocol/numerical decoder remains
required; this tool is not a replacement for it. Normal and negative pairs must
retain distinct original management/helper/observer lifetimes and request IDs.

For **raw-limit**, verify exactly 67,508,864 sealed malformed zero bytes, a separate
source receipt acknowledging 67,508,865 zero bytes, original request/ID, source
artifact, peer UID/PID matched to the actual connector identity/wait receipt, and
the dedicated ABI bootstrap matched to the complete original probe. The source's
recorded write and half-close succeed on these supported successful profiles.
These bytes remain synthetic fault injection, not normal helper traffic, native
accounting, a valid reply, publication or a resource sample. Archive space must
remain available; it cannot supply the raw-limit failure.

For **archive-limit**, reopen every actual preceding normal original and four
separately classified sealed zero-stream pressure captures, sum their recorded
bytes and require exactly 80 remaining bytes. Reopen the final 80-byte producer
prefix: its header must describe an incomplete bounded frame, but no complete
frame or numerical oracle is reconstructed. Actual originals sum to precisely
268,435,456 bytes. Pressure remains synthetic; no ordinary producer-scale claim.

Both negatives require the original failed boundary and byte count, no capture
syscall error, completion, forwarding, index or publication. Boundary/followup/
final receipts must retain the same original and failure with one reaped connector;
a real recorded refresh/busy join must stay denied. Actual unavailable/failure
response bytes are checked, together with recorded helper/configuration/token and
cleanup observations. Missing, failed, incomplete, mixed-profile and historical
predecessor archives cannot become successful evidence.

## Input bounds and refusal

Inputs use the existing pinned regular-descriptor and original-size read budget;
final symlinks/FIFOs and input replacement/growth fail. Only uncompressed plain
tar is accepted: 272 MiB input/aggregate bound, 128 entries, canonical safe names,
root ownership and exact supported modes. Header bounds precede payload reads:
only declared original wire slots may reach 67,508,864 bytes; framed receipts stay
at 64 KiB plus framing, and probe/log files at 1 MiB. Originals are read one at a
time; the archive itself is not loaded wholesale. There is no extraction, network,
process launch, output-file write, cap override or hidden acceptance mode.

Strict JSON rejects duplicate fields, floats/nonfinite values, wrong types, unknown
receipt fields and excess nesting. Unexpected files, duplicate entries, mixed
namespaces, links, sparse/extension headers, compression, incomplete end markers
and nonzero trailing bytes fail. Success prints a scoped JSON consistency report
with qualification/native/resource acceptance and activation always false.
Rejection returns nonzero with empty stdout and a generic stderr message, without
printing input paths or receipt contents.

## Validation and boundaries

Thirteen portable behavioral methods exercise both actual byte bounds and
adversarial rebindings. Additional failing regressions exposed acceptance of a
jointly rebound manager lifetime and invalid incomplete-frame headers; those
checks were added before signed-source qualification. These are development test
failures, not native workload results. Synthetic fixtures create real full-size
zero-stream tar payloads without lowering production limits or committing blobs.

Source-bound retained-archive replay records its validator source separately from
the original native sources. No VM/workload runs at the validator commit, and no
failed/interrupted predecessor is upgraded. Full integrated admission ordering,
cancellation/recovery, cadence/calibration/overhead, resource/activation, browser/
manual accessibility, hardware and release gates remain open. Canonical resource
instrumentation stays inactive and runtime writes stay denied. No runtime
interface, configuration, dependency, service or Action changes.
