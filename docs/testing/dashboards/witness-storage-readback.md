# Reproducible offline storage-custody readback

Status: implementation for review; retained-archive replay results below.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[accepted EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-009.

The [partial-storage controls](witness-partial-storage.md) already retain native
originals and immutable candidate receipts. Their original independent readback
used a private development script. The repository now owns a tested, read-only
validator so another reviewer can repeat the archive consistency checks without
that script, a running guest, root access or extracting archive contents.

## Independent expectations and reproduction

Use [the validator](../../../tools/witness-storage-evidence.py) with a retained
uncompressed original tar and a separate, privately saved expectation JSON file:

```sh
.tools/venv/bin/python3 tools/witness-storage-evidence.py /path/to/witness-originals.tar /path/to/expectation.json
```

The expectation schema has exactly these fields:

| Field | Independent authority |
| --- | --- |
| `schema` | Integer `1` |
| `source_commit`, `base_commit`, `pr_title` | Original clean signed/DCO review and host build |
| `release` | Original pinned host/guest preflight: `15.1` or `14.5` |
| `source_archive_sha256`, `compiler_sha256` | Original verified candidate build inputs |
| `fixture_sha256` | Original release's compiled fixture artifact |
| `probe_sha256` | Independently retained framed ABI-probe receipt |
| `archive_sha256` | Independently retained original archive digest |

Do not obtain these expectations from the archive being validated. For a replay,
the immutable, reviewed native manifest and separately preserved host build/run
records supply them. Source/base are full 40-character lowercase commit IDs;
hashes are full lowercase SHA-256. Do not commit private paths or raw archives.
Supply a new expectation file deliberately for each actual release/candidate.

The archive does not contain an independent source or release attestation. The
externally verified digest-to-candidate/release mapping is a trust input, not
something the tool can rediscover. Replacing both the archive and its trusted
expectations breaks that trust. The tool checks identity syntax and binds its
report to the supplied context; it does not independently verify Git signatures,
build/compiler execution, guest ABI support, supervision, preflight or host cleanup.
Those original gates remain separate and required.

## Checked scope and rejection

Version 1 supports exactly the three successful synthetic custody cases: normal,
zero-byte tmpfs ENOSPC and child-only file-limit EFBIG. It reopens and hashes all
five original files, checks root/object/mode/size references, exact counts and caps,
framing/request ID/end count, delivery versus incomplete capture, original errno,
resource restoration, continued denial, child PID/wait/limits and original mount
cleanup receipts. Probe source/binary bytes must match their pinned receipt.
The filler itself was not exported: only its original metadata/hash consistency
is checked offline; actual filler readback remains the native observation.

The complete independent Go wire decoder remains required. This scoped validator's
framing/correlation checks are not a second full protocol/numeric oracle, and
synthetic originals establish no native producer accounting or publication.

Reads use pinned regular descriptors with final symlinks/FIFOs refused, a 272 MiB
archive ceiling, at most 128 entries and a 1 MiB entry ceiling. JSON receipt/frame
limits stay 64 KiB with depth/type/duplicate-field checks; raw synthetic originals
are bounded at 64 KiB. Tar paths, namespaces, root ownership, modes, duplicate and
missing files, links, sparse/extension headers, compression and nonzero trailing
data fail closed. Extension/member bounds are checked before tar payload reads.
Original input identity/size/timestamps are rechecked after validation. There is
an original-size read budget even if a regular file keeps growing during readback.
There is
no extraction, network, process launch or output file write in the validator.

On success, stdout contains one JSON consistency report; qualification, native
acceptance, resource acceptance and activation are always false. On rejection,
exit is nonzero, stdout is empty, and stderr carries a generic refusal without
private paths or receipt contents. Failed, incomplete and older schema archives
remain rejected by this successful-matrix validator. Preserve them and their
original failure scope; do not rewrite them to fit this schema.

## Validation and remaining work

Synthetic behavioral tests exercise the complete report and adversarial input,
including fully rebound hashes with wrong original framing, replaced input
objects, oversized headers, archive namespace attacks, false restoration/wait
claims and separate expectation pins. These tests are not native evidence.

Actual retained-archive replay is reported with its validator source separately
from the original native source. It does not rerun a workload, upgrade a failed
candidate, or establish native behavior at the validator's newer commit. No
fixture, runtime, API, configuration, dependency, service or Action is added.
Broker raw/archive limits, integrated admission, remaining cancellation/recovery,
cadence/calibration/overhead, full resource/activation, browser/manual accessibility,
hardware and release acceptance remain open. Runtime storage writes stay denied.
