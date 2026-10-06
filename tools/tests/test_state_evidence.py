# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest

ROOT=Path(__file__).resolve().parents[2]
SPEC=importlib.util.spec_from_file_location('state_evidence',ROOT/'tools/check-state-evidence.py')
CASES=['root_directory', 'private_lock', 'cloexec', 'second_writer_denial', 'close_release', 'reopen', 'stable_inode', 'first_writer_race', 'directory_sync', 'lease_close', 'path_denial', 'no_directory_creation', 'directory_mode_denial', 'directory_owner_denial', 'ancestor_denial', 'directory_symlink_denial', 'lock_mode_denial', 'lock_owner_denial', 'lock_nonempty_denial', 'lock_hardlink_denial', 'lock_symlink_denial', 'lock_fifo_denial', 'lock_directory_denial', 'unsupported_fs_denial', 'mount_teardown', 'kernel_uid_denial', 'process_lock', 'process_denial', 'process_kill', 'process_reaped', 'death_release', 'death_inode_preservation', 'final_close', 'foreign_preservation', 'fixture_teardown']
READ_CASES=['missing_denial','selector_denial','valid_read','bytes_preserved','independent_policy','inode_preserved','kernel_uid_denial','mode_denial','owner_denial','hardlink_denial','symlink_denial','fifo_denial','directory_denial','empty_denial','oversize_denial','damage_denial','newer_denial','wrong_pool_denial','controls_preserved','directory_anchor','directory_drift_denial','lock_drift_denial','lock_replacement_denial','lease_close','closed_denial','inert_denial','foreign_preservation','fixture_teardown']
ACL_CASES=['posix_job_directory_denial', 'posix_job_lock_denial', 'posix_job_record_denial', 'posix_job_record_no_repair', 'posix_job_restored_read', 'nfs4_job_directory_denial', 'nfs4_job_lock_denial', 'nfs4_job_record_denial', 'nfs4_job_record_no_repair', 'nfs4_job_restored_read', 'job_kernel_uid_denial','descriptor_denial','posix_filesystem', 'posix_capabilities', 'posix_baseline', 'posix_directory_denial', 'posix_directory_read_denial', 'posix_directory_no_repair', 'posix_ancestor_denial', 'posix_ancestor_no_creation', 'posix_lock_denial', 'posix_lock_read_denial', 'posix_lock_no_repair', 'posix_record_denial', 'posix_record_no_repair', 'posix_inheritance_denial', 'posix_inheritance_read_denial', 'posix_inheritance_no_repair', 'posix_restored_read', 'posix_lease_close', 'posix_unmount', 'posix_device_teardown', 'nfs4_filesystem', 'nfs4_capabilities', 'nfs4_baseline', 'nfs4_directory_denial', 'nfs4_directory_read_denial', 'nfs4_directory_no_repair', 'nfs4_ancestor_denial', 'nfs4_ancestor_no_creation', 'nfs4_lock_denial', 'nfs4_lock_read_denial', 'nfs4_lock_no_repair', 'nfs4_record_denial', 'nfs4_record_no_repair', 'nfs4_inheritance_denial', 'nfs4_inheritance_read_denial', 'nfs4_inheritance_no_repair', 'nfs4_restored_read', 'nfs4_lease_close', 'nfs4_unmount', 'nfs4_device_teardown', 'kernel_nfs4_grant', 'kernel_uid_denial', 'process_teardown', 'fixture_teardown']
PUBLISH_CASES=['create', 'canonical_bytes', 'private_file', 'duplicate_denial', 'revision_denial', 'invalid_denial', 'capacity_denial', 'replace', 'old_descriptor_preserved', 'new_inode', 'readback', 'stable_lock', 'no_temporary', 'uncertain_write_blocked', 'uncertain_readable', 'cleanup_uncertain', 'cleanup_write_blocked', 'serialized_revision', 'destination_drift', 'directory_anchor', 'replacement_untouched', 'directory_drift', 'lock_drift', 'lease_close', 'closed_denial', 'inert_denial', 'foreign_preservation', 'fixture_teardown', 'mode_denial', 'owner_denial', 'hardlink_denial', 'symlink_denial', 'fifo_denial', 'directory_denial', 'damage_denial', 'newer_denial', 'wrong_pool_denial', 'write_failure', 'file_sync_failure', 'close_failure', 'verify_failure', 'rename_failure', 'directory_sync_uncertain']
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
        for case in set(CASES+READ_CASES+ACL_CASES+PUBLISH_CASES+PUBLISH_FS_CASES+CATALOG_CASES+CAPACITY_CASES+JOB_READ_CASES):
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
