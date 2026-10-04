# M1 foreground development runbook

Status: development only, awaiting slice review. Date: 2026-10-04.
This is the approved loopback lab mode, not a deployment or installation guide.
Use disposable FreeBSD 15.1/14.5 QEMU systems for native development. The commands
below are implemented M1 interfaces; installable Ports/rc.d and remote HTTPS await
later designs. File daemons remain independently administered.

## Build and host preparation

`make build` creates `.tools/bin/horvium` using the checksum-pinned Go SDK,
CGO_ENABLED=0 and embedded console assets. macOS/Linux can build and run synthetic
unit tests and the HTTP client; actual helper/management/token state require
FreeBSD and fail explicitly elsewhere. `make check lint test secrets go-check`
validates repository and Go behavior. Native validation is opt-in through the
[QEMU workflow](../engineering/ci-cd.md).

An OS administrator supplies a dedicated non-login service UID/group, a protected
root-owned config and helper directory, and boot/system storage for credential
state. Horvium does not create users, change directory permissions or enable
services. Config must be a root-owned regular 0640 or 0644 file with protected
ancestors; 0640 requires read access through the service group. The socket's parent
is root-owned 0750 with the configured service group. It must contain no existing
socket at startup. Token setup creates a service-owned 0700 state directory and
0600 digest. Symlinks, unsafe ancestors and unexpected files are rejected.

Example internal config (substitute actual UID/group and protected absolute paths;
521 is an illustrative account value, never a hardware or runtime default):

```json
{
  "version": 1,
  "listen": "127.0.0.1:8088",
  "allowed_hosts": ["127.0.0.1:8088"],
  "management_uid": 521,
  "socket_gid": 521,
  "helper_socket": "/var/run/horvium/inventory.sock",
  "state_directory": "/var/db/horvium",
  "max_bytes": 67108864,
  "max_records": 100000,
  "max_field_bytes": 4096,
  "max_xml_depth": 32,
  "max_xml_elements": 250000,
  "max_frame_bytes": 65536,
  "max_json_bytes": 67108864,
  "max_transport_records": 100000,
  "max_json_depth": 32,
  "interval_seconds": 30,
  "timeout_seconds": 45,
  "stale_seconds": 90,
  "max_readers": 16,
  "max_helper_clients": 8
}
```

All keys are required exactly once. Unknown keys, non-loopback listeners, unsafe
UIDs, paths and limits fail before a listener starts. Exact allowed Hosts apply
even to assets. See [security contracts](../testing/M1/security.md) for complete
ownership/validation semantics. Increasing inventory limits requires measured
budgets; no HTTP request can change them.

## Explicit foreground actions

As root, `horvium token-setup -config /usr/local/etc/horvium.json` creates the
digest and displays the token once. Capture it only in protected operator storage;
no plaintext token is retained in the state directory. Setup refuses replacement.
`horvium token-rotate -config /usr/local/etc/horvium.json` verifies existing state
and replaces the digest atomically; restart management to use it and clear sessions.
There is no HTTP setup/rotation route.

Run `horvium helper -config /usr/local/etc/horvium.json` as root in one foreground
process. Once its restricted socket exists, run
`horvium serve -config /usr/local/etc/horvium.json` under the exact configured
non-root service UID in a separate foreground process. Root management is denied.
A missing/unsafe socket at management startup is an explicit failure; after startup,
a helper outage produces unavailable/stale state and scheduled recovery attempts.
Ctrl-C/SIGTERM stop the foreground processes; they install or enable nothing.

Browse `http://127.0.0.1:8088/login` and submit the local token. Remote lab access
uses an SSH local forward to this loopback listener, with an explicitly allowed
Host. No trusted forwarded headers or reverse-proxy mode exists. HTTP cookies are
HttpOnly/SameSite Strict and volatile; this mode has no Secure-cookie/remote HTTPS
claim. Sign out and restart invalidate sessions.

For a mode-0600 token file owned by the invoking operator:

```sh
horvium inventory -endpoint http://127.0.0.1:8088 -resource overview < /protected/operator/token
horvium inventory -endpoint http://127.0.0.1:8088 -resource datasets -filter photos -limit 50 < /protected/operator/token
```

A pipe is also accepted; terminal echo, token flags and environment credentials
are not. Input accepts exactly 64 lowercase hex characters, optionally followed
by one newline. CLI JSON includes a generation; pass `-generation` to read that
same observation across resources. `-sort`, `-descending`, `-offset`, `-limit`
and `-filter` use the shared resource validation; expired generations require a
new query. There is no storage mutation or helper bypass command.

Refresh requests schedule one deduplicated collection. The current resource
selection stays on its original generation until an explicit reload; status
polling does not replace active details. Failed initial reads, partial sections,
stale data and expired views are distinct. Shares, users, SMART, protection and
jobs remain outside M1. Stopping Horvium does not stop independently running file
services; killed-process continuity and restart acceptance still need evidence.

No raw HTTP request/body/cookie/native output is logged. Fixed errors avoid
argument/credential disclosure. Integrated bounded structured audit logging,
whole-application performance and process lifecycle qualification remain the next
M1 acceptance slice; do not treat foreground development commands as a release.
