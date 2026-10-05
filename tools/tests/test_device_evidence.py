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

class DeviceEvidenceTests(unittest.TestCase):
    def module(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m);return m

    def test_native_result_and_owned_host_teardown_required(self):
        m=self.module();m.validate(RESULT+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+SYSTEM+SYSTEM_PASS+CLEAN)
        m.validate(RESULT+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+SYSTEM+SYSTEM_PASS, guest=True)
        with self.assertRaises(ValueError):m.validate(RESULT+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+CLEAN)
        with self.assertRaises(ValueError):m.validate(RESULT+CORRELATION+PASS+CLEAN)
        with self.assertRaises(ValueError):m.validate(RESULT+PASS+CLEAN)
        for text in ['',PASS+CLEAN,RESULT+CLEAN,RESULT+PASS,RESULT+PASS+CLEAN+'FAIL\n',RESULT+PASS+CLEAN+'--- SKIP: TestNativeDeviceFixtures (0.0s)\n',RESULT+PASS+CLEAN+CLEAN]:
            with self.subTest(text=text),self.assertRaises(ValueError):m.validate(text)

    def test_case_omission_duplication_unknown_or_failure_denied(self):
        m=self.module()
        for text in [RESULT.replace('missing_serial=pass',''), RESULT.replace('boot_exclusion=pass','boot_exclusion=fail'), RESULT.rstrip()+' invented=pass\n', RESULT+RESULT]:
            with self.subTest(text=text),self.assertRaises(ValueError):m.validate(text+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+SYSTEM+SYSTEM_PASS+CLEAN)

    def test_correlation_case_omission_duplication_or_failure_denied(self):
        m=self.module()
        for text in [CORRELATION.replace('missing_denial=pass',''),CORRELATION.replace('geometry_binding=pass','geometry_binding=fail'),CORRELATION+CORRELATION,CORRELATION.rstrip()+' invented=pass\n']:
            with self.subTest(text=text),self.assertRaises(ValueError):m.validate(RESULT+text+ATTACHMENTS+PASS+ATTACH_PASS+SYSTEM+SYSTEM_PASS+CLEAN)

    def test_attachment_result_and_executed_test_are_required(self):
        m=self.module()
        base=RESULT+CORRELATION+PASS+CLEAN+SYSTEM+SYSTEM_PASS
        for extra in [ATTACHMENTS,ATTACH_PASS,ATTACH_PASS+ATTACHMENTS.replace('partition_denial=pass',''),ATTACH_PASS+ATTACHMENTS.replace('fixture_teardown=pass','fixture_teardown=fail'),ATTACH_PASS+ATTACHMENTS+ATTACHMENTS,ATTACH_PASS+ATTACH_PASS+ATTACHMENTS]:
            with self.subTest(extra=extra),self.assertRaises(ValueError):m.validate(base+extra)

    def test_system_use_requires_actual_test_complete_cases_and_cleanup(self):
        m=self.module();base=RESULT+CORRELATION+ATTACHMENTS+PASS+ATTACH_PASS+CLEAN
        for extra in [SYSTEM,SYSTEM_PASS,SYSTEM_PASS+SYSTEM.replace('active_swap=pass',''),SYSTEM_PASS+SYSTEM.replace('fixture_teardown=pass','fixture_teardown=fail'),SYSTEM_PASS+SYSTEM+SYSTEM,SYSTEM_PASS+SYSTEM_PASS+SYSTEM,SYSTEM_PASS+SYSTEM.rstrip()+' invented=pass\n',SYSTEM+SYSTEM_PASS+'FAIL\n',SYSTEM+SYSTEM_PASS+'--- SKIP: TestNativeSystemDeviceUse (0.0s)\n']:
            with self.subTest(extra=extra),self.assertRaises(ValueError):m.validate(base+extra)
