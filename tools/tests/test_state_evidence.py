# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest

ROOT=Path(__file__).resolve().parents[2]
SPEC=importlib.util.spec_from_file_location('state_evidence',ROOT/'tools/check-state-evidence.py')
CASES=['root_directory', 'private_lock', 'cloexec', 'second_writer_denial', 'close_release', 'reopen', 'stable_inode', 'first_writer_race', 'directory_sync', 'lease_close', 'path_denial', 'no_directory_creation', 'directory_mode_denial', 'directory_owner_denial', 'ancestor_denial', 'directory_symlink_denial', 'lock_mode_denial', 'lock_owner_denial', 'lock_nonempty_denial', 'lock_hardlink_denial', 'lock_symlink_denial', 'lock_fifo_denial', 'lock_directory_denial', 'unsupported_fs_denial', 'mount_teardown', 'kernel_uid_denial', 'process_lock', 'process_denial', 'process_kill', 'process_reaped', 'death_release', 'death_inode_preservation', 'final_close', 'foreign_preservation', 'fixture_teardown']
READ_CASES=['missing_denial','selector_denial','valid_read','bytes_preserved','independent_policy','inode_preserved','kernel_uid_denial','mode_denial','owner_denial','hardlink_denial','symlink_denial','fifo_denial','directory_denial','empty_denial','oversize_denial','damage_denial','newer_denial','wrong_pool_denial','controls_preserved','directory_anchor','directory_drift_denial','lock_drift_denial','lock_replacement_denial','lease_close','closed_denial','inert_denial','foreign_preservation','fixture_teardown']
ACL_CASES=['descriptor_denial','posix_filesystem', 'posix_capabilities', 'posix_baseline', 'posix_directory_denial', 'posix_directory_read_denial', 'posix_directory_no_repair', 'posix_ancestor_denial', 'posix_ancestor_no_creation', 'posix_lock_denial', 'posix_lock_read_denial', 'posix_lock_no_repair', 'posix_record_denial', 'posix_record_no_repair', 'posix_inheritance_denial', 'posix_inheritance_read_denial', 'posix_inheritance_no_repair', 'posix_restored_read', 'posix_lease_close', 'posix_unmount', 'posix_device_teardown', 'nfs4_filesystem', 'nfs4_capabilities', 'nfs4_baseline', 'nfs4_directory_denial', 'nfs4_directory_read_denial', 'nfs4_directory_no_repair', 'nfs4_ancestor_denial', 'nfs4_ancestor_no_creation', 'nfs4_lock_denial', 'nfs4_lock_read_denial', 'nfs4_lock_no_repair', 'nfs4_record_denial', 'nfs4_record_no_repair', 'nfs4_inheritance_denial', 'nfs4_inheritance_read_denial', 'nfs4_inheritance_no_repair', 'nfs4_restored_read', 'nfs4_lease_close', 'nfs4_unmount', 'nfs4_device_teardown', 'kernel_nfs4_grant', 'kernel_uid_denial', 'process_teardown', 'fixture_teardown']
PUBLISH_CASES=['create', 'canonical_bytes', 'private_file', 'duplicate_denial', 'revision_denial', 'invalid_denial', 'capacity_denial', 'replace', 'old_descriptor_preserved', 'new_inode', 'readback', 'stable_lock', 'no_temporary', 'uncertain_write_blocked', 'uncertain_readable', 'cleanup_uncertain', 'cleanup_write_blocked', 'serialized_revision', 'destination_drift', 'directory_anchor', 'replacement_untouched', 'directory_drift', 'lock_drift', 'lease_close', 'closed_denial', 'inert_denial', 'foreign_preservation', 'fixture_teardown', 'mode_denial', 'owner_denial', 'hardlink_denial', 'symlink_denial', 'fifo_denial', 'directory_denial', 'damage_denial', 'newer_denial', 'wrong_pool_denial', 'write_failure', 'file_sync_failure', 'close_failure', 'verify_failure', 'rename_failure', 'directory_sync_uncertain']
PUBLISH_FS_CASES=['ufs_publication', 'acl_denial', 'acl_preservation', 'native_enospc', 'full_denial', 'full_cleanup', 'native_readonly', 'readonly_denial', 'readonly_preservation', 'filesystem_teardown']
TESTS=['TestNativeOwnershipPublish','TestNativeOwnershipPublishFilesystem','TestOwnershipPublicationOrderAndFailures','TestOwnershipPublicationRevision','TestNativeStateACL','TestStateACLPolicy', 'TestNativeStateDirectory','TestNativeOwnershipRead','TestOwnershipRecordRoundTripAndCanonicalBytes','TestOwnershipChecksumDoesNotGrantInvalidScope','TestOwnershipRecordStrictDenials','TestOwnershipRecordRejectsInvalidPolicies','TestOwnershipRecordCapacityAndRevision','TestOwnershipReadMetadataDrift']
SUBTESTS={'TestOwnershipChecksumDoesNotGrantInvalidScope':10,'TestOwnershipRecordStrictDenials':32,'TestOwnershipRecordRejectsInvalidPolicies':15}
RESULT='STATE_DIRECTORY_RESULT '+' '.join(c+'=pass' for c in CASES)+'\nOWNERSHIP_READ_RESULT '+' '.join(c+'=pass' for c in READ_CASES)+'\n'
RESULT+='STATE_ACL_RESULT '+' '.join(c+'=pass' for c in ACL_CASES)+'\n'
RESULT+='OWNERSHIP_PUBLISH_RESULT '+' '.join(c+'=pass' for c in PUBLISH_CASES)+'\nOWNERSHIP_PUBLISH_FS_RESULT '+' '.join(c+'=pass' for c in PUBLISH_FS_CASES)+'\n'
PASS=''.join('--- PASS: '+name+' (1.0s)\n' for name in TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in SUBTESTS.items() for i in range(count))

class StateEvidenceTests(unittest.TestCase):
    def module(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m);return m

    def test_complete_executed_native_suite_required(self):
        m=self.module();m.validate(PASS+RESULT)
        for bad in ['',PASS,RESULT,PASS+PASS+RESULT,PASS+RESULT+RESULT,PASS+RESULT+'FAIL\n',PASS+RESULT+'--- SKIP: TestNativeStateDirectory (0.0s)\n']:
            with self.subTest(bad=bad),self.assertRaises(ValueError):m.validate(bad)

    def test_missing_duplicate_unknown_failed_or_oversized_cases_denied(self):
        m=self.module()
        for case in set(CASES+READ_CASES+ACL_CASES+PUBLISH_CASES+PUBLISH_FS_CASES):
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
