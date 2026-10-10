# Fresh journal activation qualification

Status: qualification in progress under the accepted M2 recovery mechanism.

This slice consumes the original exclusive fresh-domain receipt exactly once,
creates fresh installation/helper identities and initializes the active journal
under both original leases. Repeated trusted native/system/boot transfers must
agree before and after its actual file/directory durability acknowledgement.
Existing directories, reopened journals, copied receipts and partial history
cannot initialize another incarnation. Releasing the receipt preserves the active
journal; it does not publish a clean shutdown.

The private storage component remains unregistered. Its native fixtures exercise
real UFS/ZFS objects, leases, namespace reads and durability. Physical-use and
boot transfers in these fixtures are synthetic; the fixed native producers have
separate accepted host gates. Their assembled commissioning path, registered
startup/global owner, native execution and restore workflow remain required.
Runtime writes stay denied.

The new behavioral tests must first fail against a denying activation stub, then
pass on both pinned disposable FreeBSD targets. The strict state checker must
reject previous evidence and incomplete activation/interrupt/teardown controls.
All guest, process, backing and compiled-artifact cleanup is required. Qualification
records will bind the executed source, toolchain and logs without host inventory.
