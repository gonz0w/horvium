# M1 administration security evidence

Status: implementation in review; native execution pending. Approved
[EDD](../../milestones/M1/EDD.md), [Test Design](../../milestones/M1/TDD.md) and
[ADR 0005](../../adr/0005-m1-access-and-state.md).
This slice adds `internal/admin` library contracts only. There is no operator CLI,
web console, installed service, port or production authentication claim.

## Configuration and credential files

Internal configuration version 1 is strict JSON. Every field below is required
exactly once, with its exact decoded spelling. Unknown/case-variant/duplicate keys,
nulls, bad types, trailing documents and more than 16 KiB fail before startup.
There are no implicit parser defaults. The preceding slices' defaults remain the
baseline for constructing a configuration; the full runtime/setup adapter comes
later. This is an internal M1 format, not a published release schema.

| Fields | Contract |
|---|---|
| `version` | Exactly 1 |
| `listen` | Canonical explicit loopback IP and port 1–65535; IPv4/IPv6, no wildcard, DNS bind, zone or mapped IPv6 |
| `allowed_hosts` | 1–16 unique exact canonical host:port values; IP or lowercase DNS name, no wildcard or URL; SSH-forward ports may differ from listener |
| `management_uid`, `socket_gid` | Nonzero numeric IDs; reserved all-ones value denied |
| `helper_socket` | Absolute clean path, at most 103 bytes, no control bytes |
| `state_directory` | Absolute clean path, at most 4,096 bytes, no control bytes; configure on boot/system storage |
| `max_bytes`, `max_records`, `max_field_bytes`, `max_xml_depth`, `max_xml_elements` | Positive trusted inventory budgets, mapped to the existing inventory reader/cache contract |
| `max_frame_bytes` | Exactly 65,536; helper protocol 1 frame size remains fixed |
| `max_json_bytes`, `max_transport_records`, `max_json_depth` | Positive transport budgets; records at least two; fields use `max_field_bytes` |
| `interval_seconds`, `timeout_seconds`, `stale_seconds` | Collection interval/staleness 1–86,400 seconds, helper/cache timeout 1–300 seconds; previous baseline 30/45/90 |
| `max_readers`, `max_helper_clients` | 1–64 each; previous default reader admission 16 |

`LoadConfig` is FreeBSD-only and reads an existing root-owned regular 0640/0644
file with one hard link. Open each ancestor from `/` using directory descriptors
and `O_NOFOLLOW`; verify actual opened ownership/modes. Immediate parent prohibits
non-root writes. Root-owned sticky ancestors such as `/tmp` are permitted only
above a protected root-owned parent. Symlinks, writable/unprivileged ancestors,
unsafe file ownership/modes and nonregular files fail. There is no permission
repair or alternative unprotected read. A 0640 file needs a group readable by the
service identity; the future setup/package owns that provisioning.

`SetupToken` requires local root and validated config, with a preexisting protected
root-owned parent. It creates only the state leaf if absent, owned by the exact
management UID/configured group, mode 0700. It generates 32 random bytes, encodes
them as 64 lowercase hex characters and returns that value after persistence.
The file `token.sha256` contains only the 32-byte SHA-256 digest of that encoded
token, service-owned 0600 with one hard link. High-entropy tokens justify SHA-256;
this is not a human-password hashing scheme. No library action prints a token.
The future CLI must display it once and accept credentials through protected input.

Writes use an exclusive no-follow temporary regular file, descriptor ownership/
mode checks, file sync and directory sync. Initial publication uses `linkat` to
refuse replacement of any existing entry. Explicit rotation first requires a safe
existing digest, then atomically renames a synced replacement. Failed validation
leaves the old digest intact; a failure after publication/sync may already have
changed state, so a caller must not assume rollback or reveal an unpersisted token.
Returned-token verification checks the published inode and digest. OS administrators
remain trusted; concurrent later rotation can replace a prior successfully returned
token. No general privileged path/command API is exposed over HTTP.

`LoadDigest` requires the exact non-root management UID and opens the protected
service-owned directory/file without following links. Root has no runtime exemption.
Non-FreeBSD file operations explicitly return unsupported. Config/digest are kept
separate from volatile inventory. A missing data pool is not consulted by these
libraries; complete boot/startup-with-missing-pool qualification remains pending.

## Volatile authentication

The SHA-256 token digest is compared in constant time. A successful exchange
consumes a random pre-auth challenge and produces distinct random 256-bit session
and CSRF values. Challenges bind to the actual peer IP, expire after two minutes,
are consumed on attempted exchanges, and have a 32-entry cap. Challenge/session
IDs travel in HttpOnly cookies; response payloads contain the corresponding CSRF
value. There is no reusable token endpoint without a challenge.

At most eight sessions exist, with 15-minute idle and eight-hour absolute expiry.
Expired entries are pruned before admission/use. Successful authorized requests
touch idle expiry; bad method/provenance/CSRF does not. Capacity exhaustion denies
new admission without evicting a valid session. Logout removes that session;
Close/restart clears all volatile credentials. Token rotation requires management
restart; an already running instance intentionally holds its startup digest.

Challenge and login operations together count toward a fixed one-minute throttle
window: 20 attempts per actual source IP and 100 globally, including successful
attempts and capacity refusals. Source ports and forwarded headers are not identity.
The source table caps at 128 identities and fails closed when full, rather than
letting attackers evict prior counts. Normal global admission already bounds it to
100 per window. Window rollover resets counts; challenge expiry remains separate.
A flood can temporarily deny legitimate loopback administration; bounded fail-closed
admission is intentional for this M1 lab mode. There is no persistent auth database.

## HTTP boundary

`Gate` provides challenge/login/logout handlers and protected-handler wrapping;
the future runtime must wire every inventory/full-page/fragment/refresh handler
through it. These library methods do not finalize public route names. Protected
requests authenticate before application lookup and place only the session's CSRF
value in the request context. An HTMX header confers no authority. Protected
adapters must implement only their declared read/refresh contract; registration of
a POST-only refresh route belongs to the console slice. No HTTP handler invokes
native facilities or privileged setup.

All responses, including denials, use `Cache-Control: no-store`, CSP with self-only
assets and no inline/eval exception, `form-action 'self'`, `frame-ancestors 'none'`,
`base-uri 'none'`, nosniff, DENY framing and no-referrer. Exact configured Host values
are checked on safe requests too. Origins, when supplied, must exactly equal
`http://` plus request Host; scheme, port, slash and duplicate headers matter.
Fetch Metadata permits only same-origin/none, never cross-site/same-site. Neither
forwarded Host/IP/protocol nor payload identity is trusted.

Go 1.27.1 `http.CrossOriginProtection.Check` supplements this policy. Its pinned
SDK source `src/net/http/csrf.go` allows safe methods, no provenance headers, and
same-origin Fetch Metadata before checking Origin. The project separately checks
exact Origin even with same-origin metadata, and requires CSRF on every POST.
A CLI without Origin/Fetch Metadata therefore still needs challenge/session cookies
and the corresponding body CSRF token. Cookies use HttpOnly, SameSite=Strict,
Path=/, no Domain; no Secure flag in this explicitly loopback HTTP mode. Network
HTTPS and Secure cookies require the later access design.

Login/logout require POST and no query string. Login accepts exactly one `token`
and one `csrf` form field; logout exactly one `csrf`. Protected POSTs require one
body CSRF field. Form content type is URL-encoded, optionally UTF-8 charset; JSON,
multipart, other parameters and malformed/duplicate credential input fail. No
URL token input is accepted. GET/HEAD bodies are denied. Other methods are denied
before application dispatch; supported read/refresh routing is still the adapter's
responsibility. Duplicate session/challenge cookies fail. Invalid input/error
responses use fixed safe text, never echo tokens, cookies or request bodies.

Bodies cap at 4 KiB including chunked reads. Handler header/target accounting caps
at 16 KiB; `HTTPServer` also sets actual net/http MaxHeaderBytes to 16 KiB with
5-second header, 10-second read/write and 30-second idle timeouts. net/http adds
its own bounded read-ahead allowance; the native fixture exceeds that allowance
and requires a 431 response before application dispatch. The constructor neither
opens a listener nor enables a service, and callers must use gated handlers.

## Tests and traceability

Behavioral tests initially failed against refusal/pass-through stubs. Regression
tests also exposed rejection of ordinary UTF-8 form submissions and acceptance of
GET bodies; both now have failing-before-fix coverage. Tests cover strict config,
loopback/Host limits, token/challenge replay and entropy failure, session idle/
absolute expiry/admission, concurrent eight-session admission, source/global
throttles and rollover, fail-closed source-map saturation, logout/Close/restart,
authentication before reads, fragment parity, login CSRF, conflicting provenance,
missing/bad/duplicate CSRF, spoofed forwarded identities, body/header/content-type
limits and credential-free denial bodies. Bounded configuration fuzzing and race
checks supplement deterministic cases.

Native tests require root and explicit disposable-QEMU opt-in. They generate only
temporary root-owned config anchors and service-owned credential fixtures, plus
an ephemeral HTTP listener; no host user, daemon, pool or unrelated file is changed.
Root copies the test binary; children actually drop UID/GID before execution. The
configured UID reads the digest and exercises login/protected fragments/CSRF/Host/
Origin/transport-header limits over real loopback HTTP. A second numeric UID shares
the service group yet fails both library authorization and the kernel read. Fixture
tokens travel only through a subprocess stdin pipe, never argv/environment/logs.
The direct child invocation skips by design; the parent requires its reexecution.

Filesystem cases check root config load, service-owned 0700/0600 state, digest-only
contents, no-clobber setup, explicit atomic rotation, one winner among eight
concurrent initial setups, no temporary-file leak, complete old/new 32-byte reads
during rotation, unsafe file/ancestor modes, symlinks/hard links, unprivileged
ownership and FIFO rejection. Native tests were written before the FreeBSD file
implementation; macOS cannot execute their kernel ownership cases. Native
execution, rather than cross-compilation, determines their result below.

| Test ID | This slice | Remaining full M1 gate |
|---|---|---|
| M1-AUTH-001 | Token/digest/ownership and volatile bounded auth primitives | Operator prompt/display, integrated logging and installed process restart |
| M1-HTTP-001 | Library auth/CSRF/Host/Origin/cookies/caps/CSP; disposable real HTTP fixture | Console route wiring, HTMX interactions and real browser evidence |
| M1-RESTART-001 | Close/new-auth invalidate sessions; token file remains separate | Kill/restart complete processes, missing data pool and independent file daemons |
| M1-BUILD-001 | Go standard library plus already-reviewed x/sys for FreeBSD descriptor operations | Executables/assets and release build provenance |

## Execution record

Local `make check lint test secrets go-check` and a FreeBSD amd64 test
cross-compilation are the development gates. Native source/run, actual guest
versions and pass/fail results will be recorded after execution. No skipped fixture
is counted as native acceptance. Existing native discovery/helper/cache tests run
alongside this slice. Complete M1, browser, hardware, performance, package and
production security qualification remain pending.
