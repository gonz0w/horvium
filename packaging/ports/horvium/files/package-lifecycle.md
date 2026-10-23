# Private read-only lab package

Source profile only. Native build/install/lifecycle qualification is still open;
do not treat this as a production installation instruction or execution grant.
The package contains no account provisioning, token, active config, writer
activation, service enablement or automatic crash restart. File daemons remain
independent. Current source still needs the existing #414 correction/qualification.

An authorized OS administrator supplies a dedicated non-login service user/group,
protected config and runtime directory under the existing M1 contract. Copy the
sample to the root-owned active config explicitly and replace its deliberately
invalid zero UID/GID values with that identity. Keep config0640 root:service-group,
runtime directory0750 root:service-group, and protected ancestors. Do not create or
prepopulate the state leaf: existing root-only token setup owns its creation and
digest publication. The state/verifier remains service-owned0700/0600 on boot storage.

Set the same `horvium_user` in trusted rc.conf for the existing identity; no default
account/UID is invented. Both `horvium_enable` and `horvium_helper_enable` default
to NO. After explicit protected setup, the existing token-setup command is
interactive and displays the token once; never run it through daemon/syslog or
put a token in arguments/environment. No HTTP bootstrap is added.

After explicit setup, run `/usr/local/sbin/horvium token-setup -config
/usr/local/etc/horvium.json` interactively as root. The installed front ends use
stock `service horvium_helper onestart`. Confirm the expected protected helper
socket exists before `service horvium onestart`; rcorder orders invocations, not
application readiness. Launch success is not authenticated readiness. Log in at
`http://127.0.0.1:8088/login`, then compare inventory with
`/usr/local/sbin/horvium inventory -endpoint http://127.0.0.1:8088 -resource overview`
using an operator-owned mode0600 token file as redirected stdin. Use the returned
generation for later resource queries. There is no health/readiness bypass route.
Management runs under the selected non-root account; the root helper remains
inventory-only. Shutdown order is management then helper; rcorder reverses the
startup dependency at system shutdown. One service does not kill the other or
independent file daemons. Restart invalidates volatile sessions/cache, preserving
active config/verifier. A helper outage retains existing unavailable/stale states.
Stop with `service horvium onestop` then `service horvium_helper onestop`;
an explicit management-only `onerestart` leaves the helper running. If an ordinary
stop is uncertain, inspect retained state; do not force restart or remove files.
The runtime directory is an explicit prerequisite after a boot that clears it;
this lab profile does not provision boot directories or qualify unattended boot.

This is ordinary trusted OS-administrator PID service control, not an immutable
process capability or root-compromise defense. Separate root-owned0600 supervisor
pidfiles are under the protected directory. Start refuses existing/unknown entries;
stop validates the exact file/expected daemon, sends TERM once, and polls for exit
and pidfile removal. Default `horvium_stop_wait=60` allows60 one-second polls;
1-300 is accepted. This is bounded polling, not a hard latency guarantee against
stalled OS commands. There is no forced KILL, stale-PID adoption, retry loop,
automatic unlink, or restart after an uncertain stop. A PID check/signal race
remains disclosed. Independent original fixture custody is a separate native gate.

Before upgrade/removal, explicitly stop both services and inspect unexpected
pidfile/socket state. Package guards refuse any such entries without signalling
or deleting them. These checks cover this fixed profile, not unknown manually
launched instances; the administrator must establish their quiescence separately.
Use the default pkg setting with automatic rc handling disabled; externally enabled
HANDLE_RC_SCRIPTS is not this profile's promise. No install hook restarts services.
Active config, verifier/state, identities, retained logs and all operator datasets
are absent from the plist and are preserved even if config equals the sample.
Only compatible explicitly reviewed package/config pairs may be rolled back;
never restore an old verifier automatically, because it can restore the old token.
