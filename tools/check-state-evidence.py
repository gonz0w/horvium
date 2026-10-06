#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
from pathlib import Path
import re
import sys

CASES={'ancestor_denial', 'lock_owner_denial', 'lock_fifo_denial', 'mount_teardown', 'lease_close', 'directory_mode_denial', 'process_denial', 'close_release', 'final_close', 'fixture_teardown', 'directory_symlink_denial', 'second_writer_denial', 'lock_directory_denial', 'process_reaped', 'directory_owner_denial', 'lock_hardlink_denial', 'reopen', 'private_lock', 'root_directory', 'directory_sync', 'process_kill', 'death_release', 'path_denial', 'process_lock', 'death_inode_preservation', 'no_directory_creation', 'stable_inode', 'first_writer_race', 'cloexec', 'unsupported_fs_denial', 'lock_nonempty_denial', 'lock_symlink_denial', 'kernel_uid_denial', 'foreign_preservation', 'lock_mode_denial'}

READ_CASES={'missing_denial','selector_denial','valid_read','bytes_preserved','independent_policy','inode_preserved','kernel_uid_denial','mode_denial','owner_denial','hardlink_denial','symlink_denial','fifo_denial','directory_denial','empty_denial','oversize_denial','damage_denial','newer_denial','wrong_pool_denial','controls_preserved','directory_anchor','directory_drift_denial','lock_drift_denial','lock_replacement_denial','lease_close','closed_denial','inert_denial','foreign_preservation','fixture_teardown'}
ACL_CASES={'descriptor_denial','posix_directory_no_repair', 'posix_baseline', 'nfs4_capabilities', 'kernel_uid_denial', 'nfs4_lock_read_denial', 'nfs4_directory_no_repair', 'posix_inheritance_denial', 'nfs4_lock_denial', 'nfs4_lock_no_repair', 'nfs4_inheritance_read_denial', 'posix_directory_denial', 'nfs4_restored_read', 'posix_ancestor_no_creation', 'nfs4_directory_read_denial', 'nfs4_baseline', 'posix_inheritance_no_repair', 'nfs4_inheritance_no_repair', 'process_teardown', 'nfs4_directory_denial', 'posix_lock_read_denial', 'posix_record_no_repair', 'posix_restored_read', 'posix_lock_denial', 'nfs4_record_no_repair', 'posix_capabilities', 'nfs4_ancestor_denial', 'posix_unmount', 'posix_lease_close', 'posix_record_denial', 'kernel_nfs4_grant', 'nfs4_filesystem', 'posix_directory_read_denial', 'nfs4_inheritance_denial', 'nfs4_lease_close', 'fixture_teardown', 'nfs4_ancestor_no_creation', 'posix_ancestor_denial', 'nfs4_unmount', 'nfs4_record_denial', 'posix_filesystem', 'posix_inheritance_read_denial', 'posix_lock_no_repair', 'posix_device_teardown', 'nfs4_device_teardown'}
PUBLISH_CASES={'closed_denial', 'lease_close', 'directory_denial', 'cleanup_write_blocked', 'fixture_teardown', 'stable_lock', 'file_sync_failure', 'inert_denial', 'duplicate_denial', 'close_failure', 'create', 'newer_denial', 'directory_drift', 'private_file', 'invalid_denial', 'directory_sync_uncertain', 'canonical_bytes', 'destination_drift', 'uncertain_readable', 'foreign_preservation', 'serialized_revision', 'hardlink_denial', 'damage_denial', 'readback', 'uncertain_write_blocked', 'owner_denial', 'old_descriptor_preserved', 'no_temporary', 'capacity_denial', 'lock_drift', 'verify_failure', 'write_failure', 'symlink_denial', 'rename_failure', 'new_inode', 'cleanup_uncertain', 'replace', 'directory_anchor', 'wrong_pool_denial', 'replacement_untouched', 'fifo_denial', 'mode_denial', 'revision_denial'}
PUBLISH_FS_CASES={'catalog_acl_denial','catalog_acl_preservation','catalog_acl_restored','acl_preservation', 'full_denial', 'acl_denial', 'native_enospc', 'readonly_denial', 'ufs_publication', 'full_cleanup', 'filesystem_teardown', 'native_readonly', 'readonly_preservation'}
CATALOG_CASES={'inode_preservation', 'owner_denial', 'lock_drift_denial', 'orphan_preservation', 'fixture_teardown', 'exact_limits', 'inert_denial', 'complete_catalog', 'unknown_denial', 'byte_limit', 'symlink_denial', 'uncertainty_preserved', 'record_preservation', 'newer_denial', 'record_drift_denial', 'malformed_name_denial', 'namespace_drift_denial', 'mode_denial', 'wrong_pool_denial', 'closed_denial', 'byte_accounting', 'damage_denial', 'independent_results', 'invalid_limits', 'fifo_denial', 'lease_close', 'directory_drift_denial', 'foreign_preservation', 'empty_denial', 'directory_anchor', 'sorted_records', 'oversize_denial', 'empty_catalog', 'orphan_denial', 'directory_denial', 'hardlink_denial', 'count_limit', 'repeated_catalog'}
TESTS={'TestNativeOwnershipCatalog','TestOwnershipCatalogNames','TestOwnershipCatalogLimits','TestNativeOwnershipPublish','TestNativeOwnershipPublishFilesystem','TestOwnershipPublicationOrderAndFailures','TestOwnershipPublicationRevision','TestNativeStateACL','TestStateACLPolicy', 'TestNativeStateDirectory','TestNativeOwnershipRead','TestOwnershipRecordRoundTripAndCanonicalBytes','TestOwnershipChecksumDoesNotGrantInvalidScope','TestOwnershipRecordStrictDenials','TestOwnershipRecordRejectsInvalidPolicies','TestOwnershipRecordCapacityAndRevision','TestOwnershipReadMetadataDrift'}
CODEC_SUBTEST_COUNTS={'TestOwnershipChecksumDoesNotGrantInvalidScope':10,'TestOwnershipRecordStrictDenials':32,'TestOwnershipRecordRejectsInvalidPolicies':15}

def validate(text):
    if len(text)>1<<20 or re.search(r'(?m)^(?:FAIL|[ \t]*--- (?:FAIL|SKIP):)',text):
        raise ValueError('state native evidence failed, skipped or oversized')
    passed=re.findall(r'(?m)^--- PASS: (Test\w+) \([0-9.]+s\)$',text)
    if len(passed)!=len(TESTS) or set(passed)!=TESTS:
        raise ValueError('complete executed lease/reader/ACL/publication/codec tests required')
    for marker,cases in [('STATE_DIRECTORY_RESULT',CASES),('OWNERSHIP_READ_RESULT',READ_CASES),('STATE_ACL_RESULT',ACL_CASES),('OWNERSHIP_PUBLISH_RESULT',PUBLISH_CASES),('OWNERSHIP_PUBLISH_FS_RESULT',PUBLISH_FS_CASES),('OWNERSHIP_CATALOG_RESULT',CATALOG_CASES)]:
        records=re.findall(marker+r' ([^\n]+)',text)
        if len(records)!=1:raise ValueError('one native result per boundary required')
        fields=records[0].split();expected={case+'=pass' for case in cases}
        if len(fields)!=len(expected) or set(fields)!=expected:
            raise ValueError('missing, duplicate, unknown or failed state assertion')
    children=re.findall(r'(?m)^[ \t]+--- PASS: (TestOwnership\w+)/(\S+) \([0-9.]+s\)$',text)
    if len(children)!=sum(CODEC_SUBTEST_COUNTS.values()) or len(set(children))!=len(children):
        raise ValueError('complete distinct codec subtests required')
    for name,count in CODEC_SUBTEST_COUNTS.items():
        if sum(parent==name for parent,_ in children)!=count:raise ValueError('incomplete codec test family')

if __name__=='__main__':
    validate(Path(sys.argv[1]).read_text())
