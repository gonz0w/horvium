# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest

ROOT=Path(__file__).resolve().parents[2]
SPEC=importlib.util.spec_from_file_location('state_evidence',ROOT/'tools/check-state-evidence.py')
CASES=['root_directory', 'private_lock', 'cloexec', 'second_writer_denial', 'close_release', 'reopen', 'stable_inode', 'first_writer_race', 'directory_sync', 'lease_close', 'path_denial', 'no_directory_creation', 'directory_mode_denial', 'directory_owner_denial', 'ancestor_denial', 'directory_symlink_denial', 'lock_mode_denial', 'lock_owner_denial', 'lock_nonempty_denial', 'lock_hardlink_denial', 'lock_symlink_denial', 'lock_fifo_denial', 'lock_directory_denial', 'unsupported_fs_denial', 'mount_teardown', 'kernel_uid_denial', 'process_lock', 'process_denial', 'process_kill', 'process_reaped', 'death_release', 'death_inode_preservation', 'final_close', 'foreign_preservation', 'fixture_teardown']
READ_CASES=['missing_denial','selector_denial','valid_read','bytes_preserved','independent_policy','inode_preserved','kernel_uid_denial','mode_denial','owner_denial','hardlink_denial','symlink_denial','fifo_denial','directory_denial','empty_denial','oversize_denial','damage_denial','newer_denial','wrong_pool_denial','controls_preserved','directory_anchor','directory_drift_denial','lock_drift_denial','lock_replacement_denial','lease_close','closed_denial','inert_denial','foreign_preservation','fixture_teardown']
ACL_CASES=['posix_job_directory_denial', 'posix_job_lock_denial', 'posix_job_record_denial', 'posix_job_record_no_repair', 'posix_job_restored_read', 'nfs4_job_directory_denial', 'nfs4_job_lock_denial', 'nfs4_job_record_denial', 'nfs4_job_record_no_repair', 'nfs4_job_restored_read', 'job_kernel_uid_denial','descriptor_denial','posix_filesystem', 'posix_capabilities', 'posix_baseline', 'posix_directory_denial', 'posix_directory_read_denial', 'posix_directory_no_repair', 'posix_ancestor_denial', 'posix_ancestor_no_creation', 'posix_lock_denial', 'posix_lock_read_denial', 'posix_lock_no_repair', 'posix_record_denial', 'posix_record_no_repair', 'posix_inheritance_denial', 'posix_inheritance_read_denial', 'posix_inheritance_no_repair', 'posix_restored_read', 'posix_lease_close', 'posix_unmount', 'posix_device_teardown', 'nfs4_filesystem', 'nfs4_capabilities', 'nfs4_baseline', 'nfs4_directory_denial', 'nfs4_directory_read_denial', 'nfs4_directory_no_repair', 'nfs4_ancestor_denial', 'nfs4_ancestor_no_creation', 'nfs4_lock_denial', 'nfs4_lock_read_denial', 'nfs4_lock_no_repair', 'nfs4_record_denial', 'nfs4_record_no_repair', 'nfs4_inheritance_denial', 'nfs4_inheritance_read_denial', 'nfs4_inheritance_no_repair', 'nfs4_restored_read', 'nfs4_lease_close', 'nfs4_unmount', 'nfs4_device_teardown', 'kernel_nfs4_grant', 'kernel_uid_denial', 'process_teardown', 'fixture_teardown']
PUBLISH_CASES=['create', 'canonical_bytes', 'private_file', 'duplicate_denial', 'revision_denial', 'invalid_denial', 'capacity_denial', 'replace', 'old_descriptor_preserved', 'new_inode', 'readback', 'stable_lock', 'no_temporary', 'uncertain_write_blocked', 'uncertain_readable', 'cleanup_uncertain', 'cleanup_write_blocked', 'serialized_revision', 'destination_drift', 'staged_inode_denial', 'staged_bytes_denial', 'directory_anchor', 'replacement_untouched', 'directory_drift', 'lock_drift', 'lease_close', 'closed_denial', 'inert_denial', 'foreign_preservation', 'fixture_teardown', 'mode_denial', 'owner_denial', 'hardlink_denial', 'symlink_denial', 'fifo_denial', 'directory_denial', 'damage_denial', 'newer_denial', 'wrong_pool_denial', 'write_failure', 'file_sync_failure', 'close_failure', 'verify_failure', 'rename_failure', 'directory_sync_uncertain']
PUBLISH_FS_CASES=['catalog_acl_denial','catalog_acl_preservation','catalog_acl_restored','ufs_publication', 'acl_denial', 'acl_preservation', 'native_enospc', 'full_denial', 'full_cleanup', 'native_readonly', 'readonly_denial', 'readonly_preservation', 'filesystem_teardown']
TESTS=['TestNativeJobRead', 'TestJobSuccessor', 'TestJobRecordCanonicalAndIndependent', 'TestJobLifecycleAndRestart', 'TestJobLifecycleDenials', 'TestJobRecordRequiredFieldsAndTypes', 'TestJobMetadataLimitsAndPrecision', 'TestJobRecordStrictDenials', 'TestJobChecksumCannotGrantLifecycle','TestNativeOwnershipCapacity','TestOwnershipPublicationCapacity','TestNativeOwnershipCatalog','TestOwnershipCatalogNames','TestOwnershipCatalogLimits','TestNativeOwnershipPublish','TestNativeOwnershipPublishFilesystem','TestOwnershipPublicationOrderAndFailures','TestOwnershipPublicationRevision','TestNativeStateACL','TestStateACLPolicy', 'TestNativeStateDirectory','TestNativeOwnershipRead','TestOwnershipRecordRoundTripAndCanonicalBytes','TestOwnershipChecksumDoesNotGrantInvalidScope','TestOwnershipRecordStrictDenials','TestOwnershipRecordRejectsInvalidPolicies','TestOwnershipRecordCapacityAndRevision','TestOwnershipReadMetadataDrift']
SUBTESTS={'TestJobRecordRequiredFieldsAndTypes':100,'TestJobRecordStrictDenials':19,'TestOwnershipChecksumDoesNotGrantInvalidScope':10,'TestOwnershipRecordStrictDenials':32,'TestOwnershipRecordRejectsInvalidPolicies':15}
RESULT='STATE_DIRECTORY_RESULT '+' '.join(c+'=pass' for c in CASES)+'\nOWNERSHIP_READ_RESULT '+' '.join(c+'=pass' for c in READ_CASES)+'\n'
RESULT+='STATE_ACL_RESULT '+' '.join(c+'=pass' for c in ACL_CASES)+'\n'
RESULT+='OWNERSHIP_PUBLISH_RESULT '+' '.join(c+'=pass' for c in PUBLISH_CASES)+'\nOWNERSHIP_PUBLISH_FS_RESULT '+' '.join(c+'=pass' for c in PUBLISH_FS_CASES)+'\n'
CATALOG_CASES=['byte_accounting', 'byte_limit', 'closed_denial', 'complete_catalog', 'count_limit', 'damage_denial', 'directory_anchor', 'directory_denial', 'directory_drift_denial', 'empty_catalog', 'empty_denial', 'exact_limits', 'fifo_denial', 'fixture_teardown', 'foreign_preservation', 'hardlink_denial', 'independent_results', 'inert_denial', 'inode_preservation', 'invalid_limits', 'lease_close', 'lock_drift_denial', 'malformed_name_denial', 'mode_denial', 'namespace_drift_denial', 'newer_denial', 'orphan_denial', 'orphan_preservation', 'oversize_denial', 'owner_denial', 'record_drift_denial', 'record_preservation', 'repeated_catalog', 'sorted_records', 'symlink_denial', 'uncertainty_preserved', 'unknown_denial', 'wrong_pool_denial']
RESULT+='OWNERSHIP_CATALOG_RESULT '+' '.join(c+'=pass' for c in CATALOG_CASES)+'\n'
CAPACITY_CASES=['create_byte_denial', 'exact_create', 'count_denial', 'replacement_reservation', 'denial_preservation', 'exact_replace', 'replacement_readback', 'stale_denial', 'invalid_limits', 'stable_lock', 'unknown_denial', 'damage_denial', 'newer_denial', 'wrong_pool_denial', 'mode_denial', 'symlink_denial', 'hardlink_denial', 'fifo_denial', 'directory_denial', 'orphan_denial', 'orphan_preservation', 'orphan_latch', 'late_unknown_denial', 'late_orphan_denial', 'late_count_denial', 'late_bytes_denial', 'late_orphan_latch', 'serialized_capacity', 'serialized_catalog', 'foreign_preservation', 'lease_close', 'fixture_teardown']
RESULT+='OWNERSHIP_CAPACITY_RESULT '+' '.join(c+'=pass' for c in CAPACITY_CASES)+'\n'
JOB_READ_CASES=['missing_denial', 'selector_denial', 'valid_read', 'bytes_preserved', 'inode_preserved', 'independent_job', 'exact_size', 'mode_denial', 'owner_denial', 'symlink_denial', 'hardlink_denial', 'fifo_denial', 'directory_denial', 'empty_denial', 'oversize_denial', 'damage_denial', 'newer_denial', 'wrong_id_denial', 'wrong_kind_denial', 'invalid_history_denial', 'controls_preserved', 'directory_anchor', 'directory_drift_denial', 'lock_drift_denial', 'lock_replacement_denial', 'lease_close', 'closed_denial', 'inert_denial', 'foreign_preservation', 'fixture_teardown']
RESULT+='JOB_READ_RESULT '+' '.join(c+'=pass' for c in JOB_READ_CASES)+'\n'
STATE_CATALOG_CASES=['byte_accounting', 'byte_limit', 'class_counts', 'closed_denial', 'directory_anchor', 'directory_drift_denial', 'empty_catalog', 'exact_limits', 'fixture_teardown', 'foreign_preservation', 'independent_results', 'inert_denial', 'inode_preservation', 'invalid_limits', 'job_damage_denial', 'job_directory_denial', 'job_empty_denial', 'job_fifo_denial', 'job_hardlink_denial', 'job_mode_denial', 'job_newer_denial', 'job_oversize_denial', 'job_owner_denial', 'job_record_drift', 'job_symlink_denial', 'job_wrong_id_denial', 'job_wrong_kind_denial', 'lease_close', 'lock_drift_denial', 'mixed_catalog', 'namespace_drift_denial', 'nonterminal_ceiling', 'nonterminal_exhaustion', 'nonterminal_limit', 'orphan_denial', 'orphan_preservation', 'ownership_damage_denial', 'ownership_limit', 'ownership_record_drift', 'record_preservation', 'repeated_catalog', 'sorted_records', 'submission_collision', 'terminal_ceiling', 'terminal_exhaustion', 'terminal_limit', 'uncertainty_preserved', 'unknown_denial']
ACL_CASES += ['posix_catalog_directory_denial', 'posix_catalog_lock_denial', 'posix_catalog_ownership_denial', 'posix_catalog_job_denial', 'posix_catalog_inheritance_denial', 'posix_catalog_restored', 'nfs4_catalog_directory_denial', 'nfs4_catalog_lock_denial', 'nfs4_catalog_ownership_denial', 'nfs4_catalog_job_denial', 'nfs4_catalog_inheritance_denial', 'nfs4_catalog_restored', 'catalog_kernel_uid_denial']
RESULT += 'STATE_CATALOG_RESULT '+' '.join(c+'=pass' for c in STATE_CATALOG_CASES)+'\n'
# Append new ACL cases to their existing result, not a second marker.
RESULT=RESULT.replace('STATE_ACL_RESULT ','STATE_ACL_RESULT '+' '.join(c+'=pass' for c in ['posix_catalog_directory_denial', 'posix_catalog_lock_denial', 'posix_catalog_ownership_denial', 'posix_catalog_job_denial', 'posix_catalog_inheritance_denial', 'posix_catalog_restored', 'nfs4_catalog_directory_denial', 'nfs4_catalog_lock_denial', 'nfs4_catalog_ownership_denial', 'nfs4_catalog_job_denial', 'nfs4_catalog_inheritance_denial', 'nfs4_catalog_restored', 'catalog_kernel_uid_denial'])+' ',1)
TESTS += ['TestNativeStateCatalog', 'TestStateCatalogNames', 'TestStateCatalogRecords', 'TestStateCatalogLimits']
JOB_PUBLICATION_CASES=['observing_publication', 'destination_ctime_denial', 'ctime_denial_preservation',  'canonical_bytes', 'cleanup_uncertain', 'close_failure', 'closed_denial', 'create', 'create_byte_denial', 'denial_preservation', 'directory_anchor', 'directory_denial', 'directory_sync_uncertain', 'duplicate_denial', 'exact_create', 'exact_replace', 'file_sync_failure', 'fixture_teardown', 'foreign_preservation', 'history_rewrite_denial', 'inert_denial', 'intent_rewrite_denial', 'invalid_denial', 'invalid_limits', 'job_exact_mixed_bytes', 'job_reserves_owner_bytes', 'late_bytes_denial', 'late_count_denial', 'late_orphan_denial', 'late_submission_denial', 'late_unknown_denial', 'lease_close', 'lock_denial', 'mixed_accounting', 'mixed_ownership_create', 'mixed_ownership_replace', 'new_inode', 'no_temporary', 'nonterminal_limit', 'old_descriptor_preserved', 'orphan_latch', 'owner_exact_mixed_bytes', 'owner_reserves_job_bytes', 'preflight_job_orphan_denial', 'preflight_ownership_denial', 'preflight_ownership_orphan_denial', 'preflight_unknown_denial', 'private_file', 'readback', 'rename_failure', 'replace', 'replacement_reservation', 'revision_denial', 'serialized_catalog', 'serialized_duplicate', 'serialized_last_slot', 'stable_lock', 'stale_denial', 'submission_collision', 'terminal_limit', 'terminal_publication', 'terminal_replay_denial', 'uncertain_job_blocked', 'uncertain_owner_blocked', 'uncertain_readable', 'unsafe_damage_denial', 'unsafe_directory_denial', 'unsafe_fifo_denial', 'unsafe_hardlink_denial', 'unsafe_mode_denial', 'unsafe_newer_denial', 'unsafe_owner_denial', 'unsafe_symlink_denial', 'unsafe_wrong_id_denial', 'unsafe_wrong_kind_denial', 'verify_failure', 'write_failure']
ACL_CASES += ['posix_job_publish_directory_denial', 'posix_job_publish_lock_denial', 'posix_job_publish_ownership_denial', 'posix_job_publish_job_denial', 'posix_job_publish_inheritance_denial', 'posix_job_publish_restored', 'nfs4_job_publish_directory_denial', 'nfs4_job_publish_lock_denial', 'nfs4_job_publish_ownership_denial', 'nfs4_job_publish_job_denial', 'nfs4_job_publish_inheritance_denial', 'nfs4_job_publish_restored', 'job_publish_kernel_uid_denial']
PUBLISH_FS_CASES += ['ufs_job_publication', 'job_full_denial', 'job_readonly_denial', 'job_readonly_preservation']
RESULT += 'JOB_PUBLICATION_RESULT '+' '.join(c+'=pass' for c in JOB_PUBLICATION_CASES)+'\n'
RESULT=RESULT.replace('STATE_ACL_RESULT ','STATE_ACL_RESULT '+' '.join(c+'=pass' for c in ['posix_job_publish_directory_denial', 'posix_job_publish_lock_denial', 'posix_job_publish_ownership_denial', 'posix_job_publish_job_denial', 'posix_job_publish_inheritance_denial', 'posix_job_publish_restored', 'nfs4_job_publish_directory_denial', 'nfs4_job_publish_lock_denial', 'nfs4_job_publish_ownership_denial', 'nfs4_job_publish_job_denial', 'nfs4_job_publish_inheritance_denial', 'nfs4_job_publish_restored', 'job_publish_kernel_uid_denial'])+' ',1)
RESULT=RESULT.replace('OWNERSHIP_PUBLISH_FS_RESULT ','OWNERSHIP_PUBLISH_FS_RESULT '+' '.join(c+'=pass' for c in ['ufs_job_publication', 'job_full_denial', 'job_readonly_denial', 'job_readonly_preservation'])+' ',1)
TESTS += ['TestNativeJobPublication', 'TestJobPublicationCapacity', 'TestMixedOwnershipPublicationCapacity']
OLD_PROPERTY_FREE_PASS=''.join('--- PASS: '+name+' (1.0s)\n' for name in TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in SUBTESTS.items() for i in range(count))
OLD_PROPERTY_FREE_RESULT=RESULT
PROPERTY_JOB_STORAGE_CASES=['baseline_rewrite_denial', 'binding_publish_denial', 'binding_read_denial', 'budget_preservation', 'canonical_bytes', 'capture_rewrite_denial', 'closeFile_failure', 'closed_denial', 'complete_read', 'create', 'create_byte_denial', 'damage_publish_denial', 'damage_read_denial', 'directory_anchor', 'directory_publish_denial', 'directory_read_denial', 'directory_sync_uncertain', 'duplicate_denial', 'empty_publish_denial', 'empty_read_denial', 'exact_create', 'exact_replace', 'exact_size', 'fifo_publish_denial', 'fifo_read_denial', 'fixture_teardown', 'foreign_preservation', 'hardlink_publish_denial', 'hardlink_read_denial', 'history_rewrite_denial', 'independent_read', 'inert_denial', 'invalid_denial', 'invalid_limits', 'late_unknown_denial', 'lease_close', 'legacy_publish_denial', 'legacy_publisher_refuses_conversion', 'legacy_read_denial', 'legacy_reader_refuses', 'legacy_submission_collision', 'missing_denial', 'mixed_accounting', 'mixed_legacy', 'mixed_owner', 'mixed_owner_byte_denial', 'mixed_owner_exact', 'mode_publish_denial', 'mode_read_denial', 'new_inode', 'newer_publish_denial', 'newer_read_denial', 'no_temporary', 'nonterminal_limit', 'observing', 'old_descriptor_preserved', 'orphan_latch', 'orphan_preservation', 'oversize_publish_denial', 'oversize_read_denial', 'owner_publish_denial', 'owner_read_denial', 'policy_rewrite_denial', 'private_file', 'property_publisher_refuses_conversion', 'property_submission_collision', 'rename_failure', 'replace', 'replacement_reservation', 'revision_denial', 'selector_denial', 'serialized_duplicate', 'session_rewrite_denial', 'stable_lock', 'staged_baseline_denial', 'stale_denial', 'symlink_publish_denial', 'symlink_read_denial', 'syncFile_failure', 'terminal', 'terminal_limit', 'terminal_replay_denial', 'uncertain_all_publishers_blocked', 'uncertain_readable', 'verify_failure', 'write_failure', 'wrong_id_publish_denial', 'wrong_id_read_denial']
ACL_CASES += ['nfs4_property_baseline', 'nfs4_property_publish_directory_denial', 'nfs4_property_publish_inheritance_denial', 'nfs4_property_publish_job_denial', 'nfs4_property_publish_lock_denial', 'nfs4_property_publish_ownership_denial', 'nfs4_property_publish_record_denial', 'nfs4_property_publish_restored', 'nfs4_property_read_directory_denial', 'nfs4_property_read_inheritance_denial', 'nfs4_property_read_lock_denial', 'nfs4_property_read_record_denial', 'nfs4_property_read_restored', 'nfs4_property_record_preserved', 'posix_property_baseline', 'posix_property_publish_directory_denial', 'posix_property_publish_inheritance_denial', 'posix_property_publish_job_denial', 'posix_property_publish_lock_denial', 'posix_property_publish_ownership_denial', 'posix_property_publish_record_denial', 'posix_property_publish_restored', 'posix_property_read_directory_denial', 'posix_property_read_inheritance_denial', 'posix_property_read_lock_denial', 'posix_property_read_record_denial', 'posix_property_read_restored', 'posix_property_record_preserved', 'property_kernel_grant', 'property_publish_kernel_uid_denial', 'property_read_kernel_uid_denial']
PUBLISH_FS_CASES += ['ufs_property_publication', 'property_full_denial', 'property_readonly_denial', 'property_readonly_preservation']
TESTS += ['TestPropertyIntentBaselineDrift', 'TestPropertyIntentCanonicalOrderingAndBounds', 'TestPropertyIntentCaptureBindingsAndPrecision', 'TestPropertyIntentCaptureRefusals', 'TestPropertyIntentRequiredFields', 'TestPropertyIntentScopePrecisionAndSessionDigest', 'TestPropertyIntentSemanticDenials', 'TestPropertyJobBindingDenials', 'TestPropertyJobBoundsAndPrecision', 'TestPropertyJobRequiredFields', 'TestPropertyJobRoundTripAndIsolation', 'TestPropertyJobSemanticDenials', 'TestPropertyJobTransitionsAndSuccessors', 'TestNativePropertyJobStorage', 'TestPropertyStateCatalog']
SUBTESTS.update({'TestPropertyIntentBaselineDrift': 7, 'TestPropertyIntentCaptureRefusals': 7, 'TestPropertyIntentRequiredFields': 741, 'TestPropertyIntentSemanticDenials': 14, 'TestPropertyJobBindingDenials': 10, 'TestPropertyJobRequiredFields': 830, 'TestPropertyJobRoundTripAndIsolation': 3, 'TestPropertyJobSemanticDenials': 10, 'TestPropertyJobTransitionsAndSuccessors': 6})
RESULT += 'PROPERTY_JOB_STORAGE_RESULT '+' '.join(c+'=pass' for c in PROPERTY_JOB_STORAGE_CASES)+'\n'
RESULT=RESULT.replace('STATE_ACL_RESULT ','STATE_ACL_RESULT '+' '.join(c+'=pass' for c in ['nfs4_property_baseline', 'nfs4_property_publish_directory_denial', 'nfs4_property_publish_inheritance_denial', 'nfs4_property_publish_job_denial', 'nfs4_property_publish_lock_denial', 'nfs4_property_publish_ownership_denial', 'nfs4_property_publish_record_denial', 'nfs4_property_publish_restored', 'nfs4_property_read_directory_denial', 'nfs4_property_read_inheritance_denial', 'nfs4_property_read_lock_denial', 'nfs4_property_read_record_denial', 'nfs4_property_read_restored', 'nfs4_property_record_preserved', 'posix_property_baseline', 'posix_property_publish_directory_denial', 'posix_property_publish_inheritance_denial', 'posix_property_publish_job_denial', 'posix_property_publish_lock_denial', 'posix_property_publish_ownership_denial', 'posix_property_publish_record_denial', 'posix_property_publish_restored', 'posix_property_read_directory_denial', 'posix_property_read_inheritance_denial', 'posix_property_read_lock_denial', 'posix_property_read_record_denial', 'posix_property_read_restored', 'posix_property_record_preserved', 'property_kernel_grant', 'property_publish_kernel_uid_denial', 'property_read_kernel_uid_denial'])+' ',1)
RESULT=RESULT.replace('OWNERSHIP_PUBLISH_FS_RESULT ','OWNERSHIP_PUBLISH_FS_RESULT '+' '.join(c+'=pass' for c in ['ufs_property_publication', 'property_full_denial', 'property_readonly_denial', 'property_readonly_preservation'])+' ',1)
PASS=''.join('--- PASS: '+name+' (1.0s)\n' for name in TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in SUBTESTS.items() for i in range(count))
OLD_RESERVATIONS_FREE_PASS=PASS
OLD_RESERVATIONS_FREE_RESULT=RESULT

RESERVATION_CASES=['byte_limit', 'closed_denial', 'complete_duplicate', 'directory_anchor', 'directory_denial', 'empty_view', 'exact_limits', 'fixture_teardown', 'foreign_preservation', 'independent_results', 'inert_denial', 'invalid_limits', 'lease_close', 'legacy_damage_denial', 'lock_denial', 'mixed_byte_accounting', 'mixed_holders', 'namespace_drift', 'no_admission', 'nonterminal_ceiling', 'nonterminal_exhaustion', 'nonterminal_limit', 'orphan_denial', 'orphan_preservation', 'ownership_budget', 'ownership_damage_denial', 'ownership_drift', 'plan_collision', 'pool_hold_denial', 'presentation_bytes', 'property_drift', 'record_damage_denial', 'record_directory_denial', 'record_empty_denial', 'record_fifo_denial', 'record_hardlink_denial', 'record_mode_denial', 'record_newer_denial', 'record_oversize_denial', 'record_owner_denial', 'record_symlink_denial', 'record_wrong_id_denial', 'record_wrong_kind_denial', 'repeated_view', 'submission_collision', 'terminal_ceiling', 'terminal_exhaustion', 'uncertainty_preserved', 'unknown_denial', 'unrelated_pool_check']
RESERVATION_ACL_CASES=['reservation_kernel_uid_denial', 'posix_reservation_baseline', 'posix_reservation_directory_denial', 'posix_reservation_lock_denial', 'posix_reservation_ownership_denial', 'posix_reservation_job_denial', 'posix_reservation_property_denial', 'posix_reservation_inheritance_denial', 'posix_reservation_restored', 'nfs4_reservation_baseline', 'nfs4_reservation_directory_denial', 'nfs4_reservation_lock_denial', 'nfs4_reservation_ownership_denial', 'nfs4_reservation_job_denial', 'nfs4_reservation_property_denial', 'nfs4_reservation_inheritance_denial', 'nfs4_reservation_restored']
TESTS += ['TestReservationBounds', 'TestReservationChecksAreReadOnly', 'TestReservationPoolHolds', 'TestReservationSubmissionChecks', 'TestReservationViewAndIsolation', 'TestReservationViewDenials', 'TestReservationStreamingBuilder', 'TestNativeJobReservations']
SUBTESTS.update({'TestReservationBounds': 2, 'TestReservationPoolHolds': 7, 'TestReservationSubmissionChecks': 13, 'TestReservationViewDenials': 8, 'TestReservationStreamingBuilder': 3})
ACL_CASES += RESERVATION_ACL_CASES
RESULT += 'JOB_RESERVATIONS_RESULT '+' '.join(c+'=pass' for c in RESERVATION_CASES)+'\n'
RESULT=RESULT.replace('STATE_ACL_RESULT ','STATE_ACL_RESULT '+' '.join(c+'=pass' for c in RESERVATION_ACL_CASES)+' ',1)
PASS=''.join('--- PASS: '+name+' (1.0s)\n' for name in TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in SUBTESTS.items() for i in range(count))

OLD_SUBMISSION_FREE_PASS=PASS
OLD_SUBMISSION_FREE_RESULT=RESULT

SUBMISSION_CASES=['canonical_bytes', 'cleanup_blocks_retry', 'cleanup_failure', 'closeFile_failure', 'closed_inert_denial', 'create', 'current_history', 'duplicate_no_write', 'exact_capacity', 'fixture_absent', 'fixture_teardown', 'full_store_retry', 'independent_result', 'invalid_capacity_denial', 'invalid_denial', 'late_bytes_denial', 'late_orphan_denial', 'late_plan_denial', 'late_pool_denial', 'late_submission_denial', 'late_unknown_denial', 'lease_close', 'legacy_retry_denial', 'nonterminal_capacity_denial', 'not_queued_denial', 'owner_bytes_capacity_denial', 'plan_reuse_denial', 'pool_hold_denial', 'private_file', 'rename_failure', 'reopen_metadata_retry', 'serialized_distinct_pools', 'serialized_last_slot', 'serialized_same_pool', 'serialized_same_submission', 'submission_reuse_denial', 'syncDirectory_blocks_retry', 'syncDirectory_failure', 'syncFile_failure', 'terminal_pool_hold', 'uncertain_visible_read', 'unrelated_pool', 'verify_failure', 'write_failure']
SUBMISSION_CASES += ['published_bytes_denial','published_inode_denial']
SUBMISSION_ACL_CASES=['submission_kernel_uid_denial', 'posix_submission_baseline', 'posix_submission_directory_denial', 'posix_submission_lock_denial', 'posix_submission_ownership_denial', 'posix_submission_job_denial', 'posix_submission_property_denial', 'posix_submission_inheritance_denial', 'posix_submission_restored', 'nfs4_submission_baseline', 'nfs4_submission_directory_denial', 'nfs4_submission_lock_denial', 'nfs4_submission_ownership_denial', 'nfs4_submission_job_denial', 'nfs4_submission_property_denial', 'nfs4_submission_inheritance_denial', 'nfs4_submission_restored']
SUBMISSION_FS_CASES=['ufs_submission_publication', 'submission_full_denial', 'submission_readonly_denial']
TESTS += ['TestNativePropertySubmission']
ACL_CASES += SUBMISSION_ACL_CASES
PUBLISH_FS_CASES += SUBMISSION_FS_CASES
RESULT += 'PROPERTY_SUBMISSION_RESULT '+' '.join(c+'=pass' for c in SUBMISSION_CASES)+'\n'
RESULT=RESULT.replace('STATE_ACL_RESULT ','STATE_ACL_RESULT '+' '.join(c+'=pass' for c in SUBMISSION_ACL_CASES)+' ',1)
RESULT=RESULT.replace('OWNERSHIP_PUBLISH_FS_RESULT ','OWNERSHIP_PUBLISH_FS_RESULT '+' '.join(c+'=pass' for c in SUBMISSION_FS_CASES)+' ',1)
PASS=''.join('--- PASS: '+name+' (1.0s)\n' for name in TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in SUBTESTS.items() for i in range(count))

SOFTUPDATES_CASES=['enabled', 'ownership_denial', 'job_denial', 'property_denial', 'submission_denial', 'retry_denial', 'read_available', 'bytes_preserved', 'no_temporary', 'records_teardown']
BACKING_TESTS=['TestNativeStateDirectory', 'TestNativeOwnershipRead', 'TestNativeOwnershipCatalog', 'TestNativeOwnershipCapacity', 'TestNativeJobRead', 'TestNativeJobPublication', 'TestNativePropertyJobStorage', 'TestNativeJobReservations', 'TestNativeStateCatalog', 'TestNativeOwnershipPublish', 'TestNativePropertySubmission', 'TestNativeStateSoftUpdates']
TESTS += ['TestNativeStateSoftUpdates']
RESULT += 'STATE_SOFTUPDATES_RESULT '+' '.join(c+'=pass' for c in SOFTUPDATES_CASES)+'\n'
RESULT += ''.join('STATE_BACKING_RESULT '+name+(' filesystem=ufs softupdates=on' if name=='TestNativeStateSoftUpdates' else ' filesystem=ufs softupdates=off' if name=='TestNativeExecutionBindingRemount' else ' filesystem=zfs softupdates=off')+' teardown=pass\n' for name in BACKING_TESTS)
PASS=''.join('--- PASS: '+name+' (1.0s)\n' for name in TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in SUBTESTS.items() for i in range(count))


OLD_CUT_FREE_PASS=PASS
OLD_CUT_FREE_RESULT=RESULT
CUT_PHASES=['before_publish','staged_before_write','after_write','after_file_sync','before_rename','after_rename','after_directory_sync','after_return']
CUT_CASES=[phase+'_'+case for phase in CUT_PHASES for case in ['barrier','writer_denial','sigkill_reaped','lock_reacquired','objects_preserved','restart_safe','retry_safe','lock_close']]+['foreign_preservation','fixture_teardown']
TESTS += ['TestNativePropertyPublicationCuts','TestRestartPageCeilingAndTraversal','TestRestartPageConcurrentReaders','TestRestartPageDenials','TestRestartPageMixedOrderAndIsolation','TestRestartPageStates']
SUBTESTS.update({'TestRestartPageStates':30,'TestRestartPageDenials':9})
BACKING_TESTS += ['TestNativePropertyPublicationCuts']
RESULT += 'PROPERTY_PUBLICATION_CUTS_RESULT '+' '.join(c+'=pass' for c in CUT_CASES)+'\n'
RESULT += 'STATE_BACKING_RESULT TestNativePropertyPublicationCuts filesystem=zfs softupdates=off teardown=pass\n'
PASS=''.join('--- PASS: '+name+' (1.0s)\n' for name in TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in SUBTESTS.items() for i in range(count))

OLD_ENROLLMENT_FREE_PASS=PASS
OLD_ENROLLMENT_FREE_RESULT=RESULT
ENROLLMENT_CASES=['first', 'exact_bytes', 'private_file', 'copies_denied', 'successor', 'old_inode', 'stable_lock', 'no_temporary', 'legacy', 'serialized_reviews', 'record_drift', 'unrelated_drift', 'unknown', 'orphan', 'queued_hold', 'terminal_hold', 'late_policy', 'late_policy_before_rename', 'late_unrelated', 'late_job', 'late_unknown', 'late_orphan', 'file_sync', 'rename', 'cleanup', 'directory_sync', 'capacity', 'expired_after_sync', 'cancelled_after_sync', 'backwards_before_rename', 'expired_before_rename', 'late_success', 'closed', 'orphan_latched', 'cleanup_blocks', 'uncertain_blocks', 'uncertain_visible', 'zero_denied', 'fixture_teardown', 'kernel_uid_denied', 'rebound_review_denied']
TESTS += ['TestNativeEnrollmentPublication']
BACKING_TESTS += ['TestNativeEnrollmentPublication']
RESULT += 'ENROLLMENT_PUBLICATION_RESULT '+' '.join(c+'=pass' for c in ENROLLMENT_CASES)+'\n'
RESULT += 'STATE_BACKING_RESULT TestNativeEnrollmentPublication filesystem=zfs softupdates=off teardown=pass\n'
PASS=''.join('--- PASS: '+name+' (1.0s)\n' for name in TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in SUBTESTS.items() for i in range(count))

OLD_CENSUS_FREE_PASS=PASS
OLD_CENSUS_FREE_RESULT=RESULT
EXECUTION_CENSUS_CASES=['empty_lock', 'published_mixed_bytes', 'same_complete_catalog', 'repeat', 'inode_preservation', 'exact_limits', 'byte_limit', 'invalid_limits', 'invalid_binding', 'binding_bytes', 'copied_handle', 'original_survives_copy', 'exact_presentation', 'published_successor', 'journal_consumes_census', 'job_drift', 'ownership_drift', 'namespace_drift', 'unknown_name', 'orphan', 'orphan_preserved', 'submission_collision', 'omission_changes_digest', 'symlink_denial', 'hardlink_denial', 'fifo_denial', 'mode_denial', 'damage_denial', 'oversize_denial', 'descriptor_anchor', 'late_lock_replacement', 'uncertainty_observed_only', 'uncertain_bytes_preserved', 'lease_close', 'closed_denial', 'inert_denial', 'foreign_preserved', 'fixture_teardown']
EXECUTION_CENSUS_ACL_CASES=['execution_census_kernel_uid_denial', 'posix_execution_census_directory_denial', 'posix_execution_census_lock_denial', 'posix_execution_census_ownership_denial', 'posix_execution_census_job_denial', 'posix_execution_census_property_denial', 'posix_execution_census_inheritance_denial', 'posix_execution_census_restored', 'nfs4_execution_census_directory_denial', 'nfs4_execution_census_lock_denial', 'nfs4_execution_census_ownership_denial', 'nfs4_execution_census_job_denial', 'nfs4_execution_census_property_denial', 'nfs4_execution_census_inheritance_denial', 'nfs4_execution_census_restored']
TESTS += ['TestNativeExecutionCensus']
BACKING_TESTS += ['TestNativeExecutionCensus']
ACL_CASES += EXECUTION_CENSUS_ACL_CASES
RESULT += 'EXECUTION_CENSUS_RESULT '+' '.join(c+'=pass' for c in EXECUTION_CENSUS_CASES)+'\n'
RESULT += 'STATE_BACKING_RESULT TestNativeExecutionCensus filesystem=zfs softupdates=off teardown=pass\n'
RESULT = RESULT.replace('STATE_ACL_RESULT ', 'STATE_ACL_RESULT '+' '.join(c+'=pass' for c in EXECUTION_CENSUS_ACL_CASES)+' ')
PASS=''.join('--- PASS: '+name+' (1.0s)\n' for name in TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in SUBTESTS.items() for i in range(count))

OLD_JOURNAL_FREE_PASS=PASS
OLD_JOURNAL_FREE_RESULT=RESULT
JOURNAL_STORE_CASES=['empty_existing_lock', 'second_writer_denial', 'no_lock_initialization', 'missing_record_denial', 'copied_close_denial', 'copied_read_denial', 'copied_publish_denial', 'noninitial_denial', 'initial_publication', 'canonical_bytes', 'private_file', 'independent_read', 'no_reinitialize', 'missing_reopen_no_initialization', 'missing_original_retirement', 'separate_namespace', 'generation_gap_denial', 'pending_publication', 'pending_readback', 'real_ledger_publication', 'actual_after_census', 'confirmed_publication', 'native_pending_publication', 'replacement_inode', 'old_descriptor_preserved', 'distinct_metadata_pending', 'real_ledger_successor', 'metadata_preserves_native', 'both_domains_readback', 'restore_barrier_publication', 'no_ordinary_rearm', 'stable_lock', 'no_temporary', 'close', 'closed_denial', 'reopen_observation', 'reopen_no_authority', 'active_reopen_no_authority', 'serialized_generation', 'symlink_denial', 'hardlink_denial', 'fifo_denial', 'mode_denial', 'owner_denial', 'damage_denial', 'oversize_denial', 'unknown_denial', 'orphan_denial', 'write_failure', 'syncFile_failure', 'closeFile_failure', 'verify_failure', 'rename_failure', 'syncDirectory_failure', 'cleanup_failure', 'lost-ack_failure', 'after-ack-drift_failure', 'late-record_failure', 'late-lock_failure', 'syncDirectory_visible', 'lost-ack_visible', 'after-ack-drift_visible', 'identical_inode_replacement', 'read_namespace_drift', 'inert_denial', 'foreign_preservation', 'fixture_teardown']
JOURNAL_CUT_CASES=['before_publish_barrier', 'before_publish_writer_denial', 'before_publish_kill_reaped', 'before_publish_lock_reacquired', 'before_publish_lock_preserved', 'before_publish_record_preserved', 'before_publish_temporary_state', 'before_publish_lease_close', 'before_publish_no_reconstructed_writer', 'staged_before_write_barrier', 'staged_before_write_writer_denial', 'staged_before_write_kill_reaped', 'staged_before_write_lock_reacquired', 'staged_before_write_lock_preserved', 'staged_before_write_record_preserved', 'staged_before_write_temporary_state', 'staged_before_write_lease_close', 'staged_before_write_no_reconstructed_writer', 'after_write_barrier', 'after_write_writer_denial', 'after_write_kill_reaped', 'after_write_lock_reacquired', 'after_write_lock_preserved', 'after_write_record_preserved', 'after_write_temporary_state', 'after_write_lease_close', 'after_write_no_reconstructed_writer', 'after_file_sync_barrier', 'after_file_sync_writer_denial', 'after_file_sync_kill_reaped', 'after_file_sync_lock_reacquired', 'after_file_sync_lock_preserved', 'after_file_sync_record_preserved', 'after_file_sync_temporary_state', 'after_file_sync_lease_close', 'after_file_sync_no_reconstructed_writer', 'before_rename_barrier', 'before_rename_writer_denial', 'before_rename_kill_reaped', 'before_rename_lock_reacquired', 'before_rename_lock_preserved', 'before_rename_record_preserved', 'before_rename_temporary_state', 'before_rename_lease_close', 'before_rename_no_reconstructed_writer', 'after_rename_barrier', 'after_rename_writer_denial', 'after_rename_kill_reaped', 'after_rename_lock_reacquired', 'after_rename_lock_preserved', 'after_rename_record_preserved', 'after_rename_temporary_state', 'after_rename_lease_close', 'after_rename_no_reconstructed_writer', 'after_directory_sync_barrier', 'after_directory_sync_writer_denial', 'after_directory_sync_kill_reaped', 'after_directory_sync_lock_reacquired', 'after_directory_sync_lock_preserved', 'after_directory_sync_record_preserved', 'after_directory_sync_temporary_state', 'after_directory_sync_lease_close', 'after_directory_sync_no_reconstructed_writer', 'after_return_barrier', 'after_return_writer_denial', 'after_return_kill_reaped', 'after_return_lock_reacquired', 'after_return_lock_preserved', 'after_return_record_preserved', 'after_return_temporary_state', 'after_return_lease_close', 'after_return_no_reconstructed_writer', 'foreign_preservation', 'fixture_teardown']
JOURNAL_ACL_CASES=['posix_journal_directory_denial', 'posix_journal_directory_restored_observation', 'posix_journal_lock_denial', 'posix_journal_lock_restored_observation', 'posix_journal_record_denial', 'posix_journal_record_restored_observation', 'posix_journal_inheritance_denial', 'posix_journal_inheritance_restored_observation', 'nfs4_journal_directory_denial', 'nfs4_journal_directory_restored_observation', 'nfs4_journal_lock_denial', 'nfs4_journal_lock_restored_observation', 'nfs4_journal_record_denial', 'nfs4_journal_record_restored_observation', 'nfs4_journal_inheritance_denial', 'nfs4_journal_inheritance_restored_observation', 'journal_kernel_grant', 'journal_kernel_uid_denial']
JOURNAL_FS_CASES=['ufs_journal_publication', 'journal_full_denial', 'journal_full_preservation', 'journal_full_cleanup', 'journal_readonly_open_denial', 'journal_readonly_preservation']
TESTS += ['TestNativeExecutionJournalStore','TestNativeExecutionJournalCuts','TestExecutionJournalNamespace','TestExecutionJournalInitialBoundary']
BACKING_TESTS += ['TestNativeExecutionJournalStore','TestNativeExecutionJournalCuts']
SUBTESTS.update({'TestExecutionJournalNamespace':15,'TestExecutionJournalInitialBoundary':6})
RESULT += 'EXECUTION_JOURNAL_STORE_RESULT '+' '.join(c+'=pass' for c in JOURNAL_STORE_CASES)+'\n'
RESULT += 'EXECUTION_JOURNAL_CUTS_RESULT '+' '.join(c+'=pass' for c in JOURNAL_CUT_CASES)+'\n'
RESULT += ''.join('STATE_BACKING_RESULT '+name+' filesystem=zfs softupdates=off teardown=pass\n' for name in ['TestNativeExecutionJournalStore','TestNativeExecutionJournalCuts'])
ACL_CASES += JOURNAL_ACL_CASES
PUBLISH_FS_CASES += JOURNAL_FS_CASES
SOFTUPDATES_CASES += ['journal_denial']
RESULT = RESULT.replace('STATE_ACL_RESULT ', 'STATE_ACL_RESULT '+' '.join(c+'=pass' for c in JOURNAL_ACL_CASES)+' ')
RESULT = RESULT.replace('OWNERSHIP_PUBLISH_FS_RESULT ', 'OWNERSHIP_PUBLISH_FS_RESULT '+' '.join(c+'=pass' for c in JOURNAL_FS_CASES)+' ')
RESULT = RESULT.replace('STATE_SOFTUPDATES_RESULT '+' '.join(c+'=pass' for c in SOFTUPDATES_CASES[:-1]), 'STATE_SOFTUPDATES_RESULT '+' '.join(c+'=pass' for c in SOFTUPDATES_CASES))
PASS=''.join('--- PASS: '+name+' (1.0s)\n' for name in TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in SUBTESTS.items() for i in range(count))


OLD_BINDING_FREE_PASS, OLD_BINDING_FREE_RESULT = PASS, RESULT
BINDING_CASES = ['native_objects','exact_descriptor_identity','live_mount_observed','repeat','actual_publication','publication_identity_stable','uncertain_observation','uncertainty_not_cleared','copied_denial','nil_denial','inert_denial','directory_mode_denial','lock_mode_denial','lock_nonempty_denial','lock_hardlink_denial','same_inode_drift','namespace_drift','restored_observation','inherited_kernel_uid_denial','child_reaped','lock_replacement_denial','close','closed_denial','reopen_identity_stable','fixture_teardown']
BINDING_REMOUNT_CASES = ['original_objects','closed_before_unmount','actual_remount','identity_preserved','close']
TESTS += ['TestNativeExecutionBindings','TestNativeExecutionBindingRemount','TestExecutionBindingIdentity','TestExecutionBindingDenials']
BACKING_TESTS += ['TestNativeExecutionBindings','TestNativeExecutionBindingRemount']
SUBTESTS.update({'TestExecutionBindingIdentity':13,'TestExecutionBindingDenials':20})
RESULT += 'EXECUTION_BINDING_RESULT '+' '.join(c+'=pass' for c in BINDING_CASES)+'\n'
RESULT += 'EXECUTION_BINDING_REMOUNT_RESULT '+' '.join(c+'=pass' for c in BINDING_REMOUNT_CASES)+'\n'
RESULT += 'STATE_BACKING_RESULT TestNativeExecutionBindings filesystem=zfs softupdates=off teardown=pass\nSTATE_BACKING_RESULT TestNativeExecutionBindingRemount filesystem=ufs softupdates=off teardown=pass\n'
PASS=''.join('--- PASS: '+name+' (1.0s)\n' for name in TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in SUBTESTS.items() for i in range(count))

class ExecutionBindingEvidenceTests(unittest.TestCase):
    def module(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m);return m

    def test_previous_evidence_cannot_qualify_bindings(self):
        m=self.module()
        with self.assertRaises(ValueError):m.validate(OLD_BINDING_FREE_PASS+OLD_BINDING_FREE_RESULT)
        self.assertEqual(m.BINDING_CASES,set(BINDING_CASES))
        self.assertEqual(m.BINDING_REMOUNT_CASES,set(BINDING_REMOUNT_CASES))
        m.validate(PASS+RESULT)

    def test_binding_cases_parents_children_and_remount_are_required(self):
        m=self.module()
        for marker,cases in [('EXECUTION_BINDING_RESULT',BINDING_CASES),('EXECUTION_BINDING_REMOUNT_RESULT',BINDING_REMOUNT_CASES)]:
            line=marker+' '+' '.join(c+'=pass' for c in cases)
            for case in cases:
                for value in ['',case+'=fail',case+'=pass '+case+'=pass']:
                    bad=marker+' '+' '.join(value if c==case else c+'=pass' for c in cases)
                    with self.subTest(case=case,value=value),self.assertRaises(ValueError):m.validate(PASS+RESULT.replace(line,bad))
        for parent in ['TestNativeExecutionBindings','TestNativeExecutionBindingRemount','TestExecutionBindingIdentity','TestExecutionBindingDenials']:
            with self.subTest(parent=parent),self.assertRaises(ValueError):m.validate((PASS+RESULT).replace('--- PASS: '+parent+' (1.0s)\n',''))
        for parent in ['TestExecutionBindingIdentity','TestExecutionBindingDenials']:
            with self.subTest(parent=parent),self.assertRaises(ValueError):m.validate((PASS+RESULT).replace('    --- PASS: '+parent+'/case_0 (0.0s)\n',''))
        with self.assertRaises(ValueError):m.validate((PASS+RESULT).replace('TestNativeExecutionBindingRemount filesystem=ufs','TestNativeExecutionBindingRemount filesystem=zfs'))

class ExecutionJournalEvidenceTests(unittest.TestCase):
    def module(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m);return m

    def test_previous_evidence_cannot_qualify_journal_persistence(self):
        with self.assertRaises(ValueError):self.module().validate(OLD_JOURNAL_FREE_PASS+OLD_JOURNAL_FREE_RESULT)
        self.assertEqual(self.module().JOURNAL_STORE_CASES,set(JOURNAL_STORE_CASES))
        self.assertEqual(self.module().JOURNAL_CUT_CASES,set(JOURNAL_CUT_CASES))
        self.module().validate(PASS+RESULT)

    def test_every_journal_control_parent_and_backing_is_required(self):
        m=self.module()
        for marker,cases in [('EXECUTION_JOURNAL_STORE_RESULT',JOURNAL_STORE_CASES),('EXECUTION_JOURNAL_CUTS_RESULT',JOURNAL_CUT_CASES)]:
            line=marker+' '+' '.join(c+'=pass' for c in cases)
            for case in cases:
                for value in ['',case+'=fail',case+'=pass '+case+'=pass']:
                    bad=marker+' '+' '.join(value if c==case else c+'=pass' for c in cases)
                    with self.subTest(case=case,value=value),self.assertRaises(ValueError):m.validate(PASS+RESULT.replace(line,bad))
        for case in JOURNAL_ACL_CASES+JOURNAL_FS_CASES+['journal_denial']:
            with self.subTest(case=case),self.assertRaises(ValueError):m.validate(PASS+RESULT.replace(case+'=pass',''))
        for parent in ['TestNativeExecutionJournalStore','TestNativeExecutionJournalCuts','TestExecutionJournalNamespace','TestExecutionJournalInitialBoundary']:
            with self.subTest(parent=parent),self.assertRaises(ValueError):m.validate((PASS+RESULT).replace('--- PASS: '+parent+' (1.0s)\n',''))
        for parent in ['TestNativeExecutionJournalStore','TestNativeExecutionJournalCuts']:
            with self.subTest(parent=parent),self.assertRaises(ValueError):m.validate((PASS+RESULT).replace('STATE_BACKING_RESULT '+parent+' filesystem=zfs softupdates=off teardown=pass\n',''))

class ExecutionCensusEvidenceTests(unittest.TestCase):
    def module(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m);return m

    def test_previous_evidence_cannot_qualify_execution_census(self):
        with self.assertRaises(ValueError):self.module().validate(OLD_CENSUS_FREE_PASS+OLD_CENSUS_FREE_RESULT)
        self.assertEqual(self.module().EXECUTION_CENSUS_CASES,set(EXECUTION_CENSUS_CASES))
        self.module().validate(PASS+RESULT)

    def test_every_census_control_parent_and_backing_is_required(self):
        m=self.module()
        line='EXECUTION_CENSUS_RESULT '+' '.join(c+'=pass' for c in EXECUTION_CENSUS_CASES)
        for case in EXECUTION_CENSUS_CASES:
            for value in ['',case+'=fail',case+'=pass '+case+'=pass']:
                bad='EXECUTION_CENSUS_RESULT '+' '.join(value if c==case else c+'=pass' for c in EXECUTION_CENSUS_CASES)
                with self.subTest(case=case,value=value),self.assertRaises(ValueError):m.validate(PASS+RESULT.replace(line,bad))
        for case in EXECUTION_CENSUS_ACL_CASES:
            with self.subTest(case=case),self.assertRaises(ValueError):m.validate(PASS+RESULT.replace(case+'=pass',''))
        for line in ['--- PASS: TestNativeExecutionCensus (1.0s)\n','STATE_BACKING_RESULT TestNativeExecutionCensus filesystem=zfs softupdates=off teardown=pass\n']:
            with self.subTest(line=line),self.assertRaises(ValueError):m.validate((PASS+RESULT).replace(line,''))

class EnrollmentPublicationEvidenceTests(unittest.TestCase):
    def module(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m);return m

    def test_previous_state_evidence_cannot_qualify_enrollment(self):
        with self.assertRaises(ValueError):self.module().validate(OLD_ENROLLMENT_FREE_PASS+OLD_ENROLLMENT_FREE_RESULT)
        self.assertEqual(self.module().ENROLLMENT_CASES,set(ENROLLMENT_CASES))
        self.module().validate(PASS+RESULT)

    def test_all_enrollment_controls_are_required(self):
        m=self.module()
        line='ENROLLMENT_PUBLICATION_RESULT '+' '.join(c+'=pass' for c in ENROLLMENT_CASES)
        for case in ENROLLMENT_CASES:
            for value in ['',case+'=fail',case+'=pass '+case+'=pass']:
                bad='ENROLLMENT_PUBLICATION_RESULT '+' '.join(value if c==case else c+'=pass' for c in ENROLLMENT_CASES)
                with self.subTest(case=case),self.assertRaises(ValueError):m.validate(PASS+RESULT.replace(line,bad))

class PublicationCutEvidenceTests(unittest.TestCase):
    def module(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m);return m

    def test_old_publication_evidence_cannot_qualify_process_cuts(self):
        with self.assertRaises(ValueError):self.module().validate(OLD_CUT_FREE_PASS+OLD_CUT_FREE_RESULT)

    def test_every_process_cut_control_and_parent_is_required(self):
        m=self.module()
        old='PROPERTY_PUBLICATION_CUTS_RESULT '+' '.join(c+'=pass' for c in CUT_CASES)
        for case in CUT_CASES:
            for value in ['',case+'=fail',case+'=pass '+case+'=pass']:
                bad='PROPERTY_PUBLICATION_CUTS_RESULT '+' '.join(value if c==case else c+'=pass' for c in CUT_CASES)
                with self.subTest(case=case,value=value),self.assertRaises(ValueError):m.validate(PASS+RESULT.replace(old,bad))
        for parent in ['TestNativePropertyPublicationCuts','TestRestartPageStates']:
            with self.subTest(parent=parent),self.assertRaises(ValueError):m.validate(PASS.replace('--- PASS: '+parent+' (1.0s)\n','')+RESULT)


class SubmissionEvidenceTests(unittest.TestCase):
    def test_old_inspection_evidence_cannot_qualify_submission_publication(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        with self.assertRaises(ValueError):m.validate(OLD_SUBMISSION_FREE_PASS+OLD_SUBMISSION_FREE_RESULT)

    def test_every_submission_assertion_and_parent_is_required(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        with self.assertRaises(ValueError):m.validate(PASS.replace('--- PASS: TestNativePropertySubmission (1.0s)\n','')+RESULT)
        for case in SUBMISSION_CASES:
            for value in ['',case+'=fail',case+'=pass '+case+'=pass']:
                bad=RESULT.replace('PROPERTY_SUBMISSION_RESULT '+' '.join(c+'=pass' for c in SUBMISSION_CASES),
                                   'PROPERTY_SUBMISSION_RESULT '+' '.join(value if c==case else c+'=pass' for c in SUBMISSION_CASES))
                with self.subTest(case=case,value=value),self.assertRaises(ValueError):m.validate(PASS+bad)
        for case in SUBMISSION_ACL_CASES+SUBMISSION_FS_CASES:
            with self.subTest(case=case),self.assertRaises(ValueError):m.validate(PASS+RESULT.replace(case+'=pass',''))


    def test_exact_backing_capabilities_and_teardown_are_required(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        for name in BACKING_TESTS:
            record='STATE_BACKING_RESULT '+name+(' filesystem=ufs softupdates=on' if name=='TestNativeStateSoftUpdates' else ' filesystem=ufs softupdates=off' if name=='TestNativeExecutionBindingRemount' else ' filesystem=zfs softupdates=off')+' teardown=pass\n'
            for replacement in ['',record+record,record.replace('teardown=pass','teardown=fail'),record.replace('softupdates=off','softupdates=on') if name!='TestNativeStateSoftUpdates' else record.replace('filesystem=ufs','filesystem=zfs')]:
                with self.subTest(name=name,replacement=replacement),self.assertRaises(ValueError):m.validate(PASS+RESULT.replace(record,replacement))
        for case in SOFTUPDATES_CASES:
            old='STATE_SOFTUPDATES_RESULT '+' '.join(c+'=pass' for c in SOFTUPDATES_CASES)
            for replacement in ['',case+'=fail',case+'=pass '+case+'=pass']:
                bad='STATE_SOFTUPDATES_RESULT '+' '.join(replacement if c==case else c+'=pass' for c in SOFTUPDATES_CASES)
                with self.subTest(case=case),self.assertRaises(ValueError):m.validate(PASS+RESULT.replace(old,bad))


class StateEvidenceTests(unittest.TestCase):
    def module(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m);return m

    def test_complete_executed_native_suite_required(self):
        m=self.module();m.validate(PASS+RESULT)
        for bad in ['',PASS,RESULT,PASS+PASS+RESULT,PASS+RESULT+RESULT,PASS+RESULT+'FAIL\n',PASS+RESULT+'--- SKIP: TestNativeStateDirectory (0.0s)\n']:
            with self.subTest(bad=bad),self.assertRaises(ValueError):m.validate(bad)

    def test_missing_duplicate_unknown_failed_or_oversized_cases_denied(self):
        m=self.module()
        for case in set(CASES+READ_CASES+ACL_CASES+PUBLISH_CASES+PUBLISH_FS_CASES+CATALOG_CASES+CAPACITY_CASES+JOB_READ_CASES+STATE_CATALOG_CASES+JOB_PUBLICATION_CASES+PROPERTY_JOB_STORAGE_CASES+RESERVATION_CASES+SUBMISSION_CASES+SOFTUPDATES_CASES):
            for bad in [RESULT.replace(case+'=pass',''),RESULT.replace(case+'=pass',case+'=fail'),RESULT.rstrip()+' '+case+'=pass\n']:
                with self.subTest(case=case),self.assertRaises(ValueError):m.validate(PASS+bad)
        for bad in [RESULT.rstrip()+' invented=pass\n',RESULT+'x'*(1<<20)]:
            with self.assertRaises(ValueError):m.validate(PASS+bad)

    def test_reader_codec_and_each_subtest_are_required(self):
        m=self.module()
        for name in TESTS:
            with self.subTest(name=name),self.assertRaises(ValueError):m.validate(PASS.replace('--- PASS: '+name+' (1.0s)\n','')+RESULT)
        for bad in [PASS.replace('    --- PASS: TestOwnershipRecordStrictDenials/case_0 (0.0s)\n',''),PASS+'    --- SKIP: child (0.0s)\n',PASS+'    --- PASS: TestOwnershipRecordStrictDenials/case_0 (0.0s)\n']:
            with self.assertRaises(ValueError):m.validate(bad+RESULT)


class CatalogEvidenceTests(unittest.TestCase):
    def test_old_catalog_free_evidence_never_qualifies_new_contract(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        old_result=RESULT[:RESULT.index('OWNERSHIP_CATALOG_RESULT ')]
        with self.assertRaises(ValueError):m.validate(PASS+old_result)


class CapacityEvidenceTests(unittest.TestCase):
    def test_old_202_assertion_evidence_never_qualifies_capacity(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        old_pass=PASS.replace('--- PASS: TestNativeOwnershipCapacity (1.0s)\n','').replace('--- PASS: TestOwnershipPublicationCapacity (1.0s)\n','')
        old_result=RESULT[:RESULT.index('OWNERSHIP_CAPACITY_RESULT ')]
        with self.assertRaises(ValueError):m.validate(old_pass+old_result)


class JobReaderEvidenceTests(unittest.TestCase):
    def test_old_234_assertion_evidence_cannot_qualify_job_reader(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        old_result=RESULT[:RESULT.index('JOB_READ_RESULT ')]
        for prefix in ['posix_','nfs4_']:
            for name in ['directory_denial','lock_denial','record_denial','record_no_repair','restored_read']:
                old_result=old_result.replace(' '+prefix+'job_'+name+'=pass','')
        old_result=old_result.replace(' job_kernel_uid_denial=pass','')
        old_pass='\n'.join(line for line in PASS.splitlines() if 'TestJob' not in line and 'TestNativeJobRead' not in line)+'\n'
        with self.assertRaises(ValueError):m.validate(old_pass+old_result)

    def test_job_schema_subtests_are_required(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        for name in ['TestJobRecordRequiredFieldsAndTypes','TestJobRecordStrictDenials']:
            child='    --- PASS: '+name+'/case_0 (0.0s)\n'
            for bad in [PASS.replace(child,''),PASS+child,PASS.replace(child,'    --- SKIP: '+name+'/case_0 (0.0s)\n')]:
                with self.subTest(name=name),self.assertRaises(ValueError):m.validate(bad+RESULT)


class MixedCatalogEvidenceTests(unittest.TestCase):
    def test_old_275_assertion_evidence_cannot_qualify_mixed_catalog(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        old_result=RESULT.split('STATE_CATALOG_RESULT ')[0]
        for prefix in ['posix_','nfs4_']:
            for name in ['directory_denial','lock_denial','ownership_denial','job_denial','inheritance_denial','restored']:
                old_result=old_result.replace(' '+prefix+'catalog_'+name+'=pass','')
        old_result=old_result.replace(' catalog_kernel_uid_denial=pass','')
        old_pass='\n'.join(line for line in PASS.splitlines() if 'TestStateCatalog' not in line and 'TestNativeStateCatalog' not in line)+'\n'
        with self.assertRaises(ValueError):m.validate(old_pass+old_result)

    def test_actual_catalog_result_has_no_placeholder_assertion(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        self.assertNotIn('job_',m.STATE_CATALOG_CASES)
        m.validate(PASS+RESULT.replace('job_=pass',''))


class JobPublicationEvidenceTests(unittest.TestCase):
    def test_old_336_assertion_evidence_cannot_qualify_publication(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        old_result=RESULT.split('JOB_PUBLICATION_RESULT ')[0]
        for prefix in ['posix_','nfs4_']:
            for name in ['directory_denial','lock_denial','ownership_denial','job_denial','inheritance_denial','restored']:
                old_result=old_result.replace(' '+prefix+'job_publish_'+name+'=pass','')
        for case in ['job_publish_kernel_uid_denial','ufs_job_publication','job_full_denial','job_readonly_denial','job_readonly_preservation']:
            old_result=old_result.replace(' '+case+'=pass','')
        new_tests=['TestNativeJobPublication','TestJobPublicationCapacity','TestMixedOwnershipPublicationCapacity']
        old_pass='\n'.join(line for line in PASS.splitlines() if not any(name in line for name in new_tests))+'\n'
        with self.assertRaises(ValueError):m.validate(old_pass+old_result)


class PropertyJobStorageEvidenceTests(unittest.TestCase):
    def test_old_property_free_evidence_is_denied(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        with self.assertRaises(ValueError):m.validate(OLD_PROPERTY_FREE_PASS+OLD_PROPERTY_FREE_RESULT)

    def test_every_property_family_and_assertion_is_required(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        for name in ['TestNativePropertyJobStorage','TestPropertyStateCatalog']+list(SUBTESTS):
            with self.subTest(name=name),self.assertRaises(ValueError):m.validate(PASS.replace('--- PASS: '+name+' (1.0s)\n','')+RESULT)
        for name in [p for p in SUBTESTS if p.startswith('TestProperty')]:
            with self.subTest(name=name),self.assertRaises(ValueError):m.validate(PASS.replace('    --- PASS: '+name+'/case_0 (0.0s)\n','')+RESULT)


class ReservationEvidenceTests(unittest.TestCase):
    def test_old_reservation_free_evidence_is_denied(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        with self.assertRaises(ValueError):m.validate(OLD_RESERVATIONS_FREE_PASS+OLD_RESERVATIONS_FREE_RESULT)

    def test_every_reservation_assertion_and_family_is_required(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m)
        for name in ['TestReservationBounds', 'TestReservationChecksAreReadOnly', 'TestReservationPoolHolds', 'TestReservationSubmissionChecks', 'TestReservationViewAndIsolation', 'TestReservationViewDenials', 'TestReservationStreamingBuilder', 'TestNativeJobReservations']:
            with self.subTest(name=name),self.assertRaises(ValueError):m.validate(PASS.replace('--- PASS: '+name+' (1.0s)\n','')+RESULT)
        for case in RESERVATION_CASES:
            for value in ['',case+'=fail',case+'=pass '+case+'=pass']:
                bad=RESULT.replace('JOB_RESERVATIONS_RESULT '+' '.join(c+'=pass' for c in RESERVATION_CASES),
                                   'JOB_RESERVATIONS_RESULT '+' '.join(value if c==case else c+'=pass' for c in RESERVATION_CASES))
                with self.subTest(case=case,value=value),self.assertRaises(ValueError):m.validate(PASS+bad)
        for case in RESERVATION_ACL_CASES:
            with self.subTest(case=case),self.assertRaises(ValueError):m.validate(PASS+RESULT.replace(case+'=pass',''))
