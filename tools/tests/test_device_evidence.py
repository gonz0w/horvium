# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest

SOURCE=Path(__file__).resolve().parents[2]
SPEC=importlib.util.spec_from_file_location('device_evidence', SOURCE/'tools/check-device-evidence.py')
RESULT='DEVICE_OBSERVATION_RESULT distinct_serial=pass duplicate_serial=pass missing_serial=pass sector_variants=pass diskinfo_correlation=pass backing_relationships=pass boot_exclusion=pass read_preservation=pass\n'
PASS='--- PASS: TestNativeDeviceFixtures (1.0s)\n'
CLEAN='DEVICE_FIXTURE_CLEANUP pass\n'
CORRELATION='DEVICE_CORRELATION_RESULT unique_serial=pass duplicate_denial=pass missing_denial=pass geometry_binding=pass\n'
ATTACHMENTS='DEVICE_ATTACHMENT_RESULT native_alias=pass partition_denial=pass blank_table_denial=pass fixture_teardown=pass foreign_preservation=pass\n'
ATTACH_PASS='--- PASS: TestNativeDiskAttachments (1.0s)\n'
SYSTEM='DEVICE_SYSTEM_USE_RESULT root_backing=pass mounted_disk=pass active_swap=pass unsupported_denial=pass fixture_teardown=pass foreign_preservation=pass\n'
SYSTEM_PASS='--- PASS: TestNativeSystemDeviceUse (1.0s)\n'

IMPORTED='DEVICE_IMPORTED_POOL_RESULT unmounted_pool=pass catalogue_binding=pass missing_denial=pass fixture_teardown=pass foreign_preservation=pass root_preservation=pass\n'
IMPORTED_PASS='--- PASS: TestNativeImportedPoolUse (1.0s)\n'

SIGNATURE_UFS='DEVICE_SIGNATURE_UFS_RESULT ufs_signature=pass probe_preservation=pass unsupported_denial=pass\n'
SIGNATURE_ZFS='DEVICE_SIGNATURE_ZFS_RESULT zfs_signature=pass removed_signature=pass probe_preservation=pass binding_denial=pass\n'
SIGNATURES=SIGNATURE_UFS+SIGNATURE_ZFS
USE_POOL='DEVICE_USE_POOL_RESULT imported_join=pass new_pool_drift=pass unchanged=pass residual_join=pass pool_removal_drift=pass missing_capture_denial=pass no_clearance=pass\n'
USE_SYSTEM='DEVICE_USE_SYSTEM_RESULT mount_join=pass swap_join=pass mount_drift=pass swap_drift=pass copy_isolation=pass use_cleanup=pass mount_removal_drift=pass swap_removal_drift=pass\n'
USES=USE_POOL+USE_SYSTEM

class DeviceEvidenceTests(unittest.TestCase):
    def module(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m);return m

    def test_native_result_and_owned_host_teardown_required(self):
        m=self.module();m.validate(RESULT+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+SYSTEM+SYSTEM_PASS+IMPORTED+IMPORTED_PASS+SIGNATURES+USES+CLEAN)
        m.validate(RESULT+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+SYSTEM+SYSTEM_PASS+IMPORTED+IMPORTED_PASS+SIGNATURES+USES, guest=True)
        with self.assertRaises(ValueError):m.validate(RESULT+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+CLEAN)
        with self.assertRaises(ValueError):m.validate(RESULT+CORRELATION+PASS+CLEAN)
        with self.assertRaises(ValueError):m.validate(RESULT+PASS+CLEAN)
        for text in ['',PASS+CLEAN,RESULT+CLEAN,RESULT+PASS,RESULT+PASS+CLEAN+'FAIL\n',RESULT+PASS+CLEAN+'--- SKIP: TestNativeDeviceFixtures (0.0s)\n',RESULT+PASS+CLEAN+CLEAN]:
            with self.subTest(text=text),self.assertRaises(ValueError):m.validate(text)

    def test_case_omission_duplication_unknown_or_failure_denied(self):
        m=self.module()
        for text in [RESULT.replace('missing_serial=pass',''), RESULT.replace('boot_exclusion=pass','boot_exclusion=fail'), RESULT.rstrip()+' invented=pass\n', RESULT+RESULT]:
            with self.subTest(text=text),self.assertRaises(ValueError):m.validate(text+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+SYSTEM+SYSTEM_PASS+IMPORTED+IMPORTED_PASS+SIGNATURES+USES+CLEAN)

    def test_correlation_case_omission_duplication_or_failure_denied(self):
        m=self.module()
        for text in [CORRELATION.replace('missing_denial=pass',''),CORRELATION.replace('geometry_binding=pass','geometry_binding=fail'),CORRELATION+CORRELATION,CORRELATION.rstrip()+' invented=pass\n']:
            with self.subTest(text=text),self.assertRaises(ValueError):m.validate(RESULT+text+ATTACHMENTS+PASS+ATTACH_PASS+SYSTEM+SYSTEM_PASS+IMPORTED+IMPORTED_PASS+SIGNATURES+USES+CLEAN)

    def test_attachment_result_and_executed_test_are_required(self):
        m=self.module()
        base=RESULT+CORRELATION+PASS+CLEAN+SYSTEM+SYSTEM_PASS+IMPORTED+IMPORTED_PASS+SIGNATURES+USES
        for extra in [ATTACHMENTS,ATTACH_PASS,ATTACH_PASS+ATTACHMENTS.replace('partition_denial=pass',''),ATTACH_PASS+ATTACHMENTS.replace('fixture_teardown=pass','fixture_teardown=fail'),ATTACH_PASS+ATTACHMENTS+ATTACHMENTS,ATTACH_PASS+ATTACH_PASS+ATTACHMENTS]:
            with self.subTest(extra=extra),self.assertRaises(ValueError):m.validate(base+extra)

    def test_system_use_requires_actual_test_complete_cases_and_cleanup(self):
        m=self.module();base=RESULT+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+CLEAN+IMPORTED+IMPORTED_PASS+SIGNATURES+USES
        for extra in [SYSTEM,SYSTEM_PASS,SYSTEM_PASS+SYSTEM.replace('active_swap=pass',''),SYSTEM_PASS+SYSTEM.replace('fixture_teardown=pass','fixture_teardown=fail'),SYSTEM_PASS+SYSTEM+SYSTEM,SYSTEM_PASS+SYSTEM_PASS+SYSTEM,SYSTEM_PASS+SYSTEM.rstrip()+' invented=pass\n',SYSTEM+SYSTEM_PASS+'FAIL\n',SYSTEM+SYSTEM_PASS+'--- SKIP: TestNativeSystemDeviceUse (0.0s)\n']:
            with self.subTest(extra=extra),self.assertRaises(ValueError):m.validate(base+extra)

    def test_imported_pool_requires_actual_test_complete_cases_and_cleanup(self):
        m=self.module();base=RESULT+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+SYSTEM+SYSTEM_PASS+CLEAN+SIGNATURES+USES
        for extra in [IMPORTED,IMPORTED_PASS,IMPORTED_PASS+IMPORTED.replace('unmounted_pool=pass',''),IMPORTED_PASS+IMPORTED.replace('fixture_teardown=pass','fixture_teardown=fail'),IMPORTED_PASS+IMPORTED+IMPORTED,IMPORTED_PASS+IMPORTED_PASS+IMPORTED,IMPORTED_PASS+IMPORTED.rstrip()+' invented=pass\n',IMPORTED+IMPORTED_PASS+'FAIL\n',IMPORTED+IMPORTED_PASS+'--- SKIP: TestNativeImportedPoolUse (0.0s)\n']:
            with self.subTest(extra=extra),self.assertRaises(ValueError):m.validate(base+extra)

    def test_signature_results_require_complete_native_probe_cases(self):
        m=self.module();base=RESULT+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+SYSTEM+SYSTEM_PASS+IMPORTED+IMPORTED_PASS+CLEAN+USES
        for record,other,key in [(SIGNATURE_UFS,SIGNATURE_ZFS,'ufs_signature'),(SIGNATURE_ZFS,SIGNATURE_UFS,'removed_signature')]:
            for bad in ['',record.replace(key+'=pass',''),record.replace('probe_preservation=pass','probe_preservation=fail'),record+record,record.rstrip()+' invented=pass\n']:
                with self.subTest(bad=bad),self.assertRaises(ValueError):m.validate(base+other+bad)

    def test_combined_use_requires_complete_native_join_and_drift_cases(self):
        m=self.module();base=RESULT+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+SYSTEM+SYSTEM_PASS+IMPORTED+IMPORTED_PASS+SIGNATURES+CLEAN
        for record,other,key in [(USE_POOL,USE_SYSTEM,'pool_removal_drift'),(USE_SYSTEM,USE_POOL,'swap_join')]:
            for bad in ['',record.replace(key+'=pass',''),record.replace(key+'=pass',key+'=fail'),record+record,record.rstrip()+' invented=pass\n']:
                with self.subTest(bad=bad),self.assertRaises(ValueError):m.validate(base+other+bad)
