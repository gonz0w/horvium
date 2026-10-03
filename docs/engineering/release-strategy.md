# Release strategy

Use SemVer-style `vMAJOR.MINOR.PATCH` annotated signed tags. Pre-1.0 changes still
identify breaking behavior and migration limits. No release number is assigned
to this repository foundation.

Release artifacts start with source and FreeBSD port/package inputs. Builds must
be reproducible or their remaining variance documented; pin toolchains, dependencies,
assets, and test environments. Include checksums, SBOM/provenance, compatibility
matrix, known limits, restore evidence, and upgrade/rollback instructions.

Port upgrade/removal preserves datasets, credentials, and operator configuration.
State migrations and package rollback are tested independently of OS upgrades.
Never automatically upgrade ZFS pool features; irreversibility and replication
compatibility require explicit operator choice. An appliance image later consumes
the same package rather than maintaining a second NAS implementation.

Upstream Ports inclusion depends on FreeBSD review. A local port or CI build is
not a claim that Horvium is already in the official Ports Collection.
