// SPDX-License-Identifier: BSD-3-Clause
package storage

// Native names from the pinned FreeBSD feature registration tables. See
// docs/testing/M2/features.md for source revisions and qualification limits.
const commonFeatureNames = "allocation_classes async_destroy blake3 block_cloning bookmark_v2 bookmark_written bookmarks device_rebuild device_removal draid edonr embedded_data empty_bpobj enabled_txg encryption extensible_dataset filesystem_limits head_errlog hole_birth large_blocks large_dnode livelist log_spacemap lz4_compress multi_vdev_crash_dump obsolete_counts project_quota redacted_datasets redaction_bookmarks resilver_defer sha512 skein spacemap_histogram spacemap_v2 userobj_accounting vdev_zaps_v2 zilsaxattr zpool_checkpoint zstd_compress"
const newerFeatureNames = "block_cloning_endian dynamic_gang_header fast_dedup large_microzap longname physical_rewrite raidz_expansion redaction_list_spill"
