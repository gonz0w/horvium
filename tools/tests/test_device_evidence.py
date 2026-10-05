# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest

SOURCE=Path(__file__).resolve().parents[2]
SPEC=importlib.util.spec_from_file_location('device_evidence', SOURCE/'tools/check-device-evidence.py')
RESULT='DEVICE_OBSERVATION_RESULT distinct_serial=pass duplicate_serial=pass missing_serial=pass sector_variants=pass diskinfo_correlation=pass backing_relationships=pass boot_exclusion=pass read_preservation=pass\n'
PASS='--- PASS: TestNativeDeviceFixtures (1.0s)\n'
CLEAN='DEVICE_FIXTURE_CLEANUP pass\n'

class DeviceEvidenceTests(unittest.TestCase):
    def module(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m);return m

    def test_native_result_and_owned_host_teardown_required(self):
        m=self.module();m.validate(RESULT+PASS+CLEAN)
        m.validate(RESULT+PASS, guest=True)
        for text in ['',PASS+CLEAN,RESULT+CLEAN,RESULT+PASS,RESULT+PASS+CLEAN+'FAIL\n',RESULT+PASS+CLEAN+'--- SKIP: TestNativeDeviceFixtures (0.0s)\n',RESULT+PASS+CLEAN+CLEAN]:
            with self.subTest(text=text),self.assertRaises(ValueError):m.validate(text)

    def test_case_omission_duplication_unknown_or_failure_denied(self):
        m=self.module()
        for text in [RESULT.replace('missing_serial=pass',''), RESULT.replace('boot_exclusion=pass','boot_exclusion=fail'), RESULT.rstrip()+' invented=pass\n', RESULT+RESULT]:
            with self.subTest(text=text),self.assertRaises(ValueError):m.validate(text+PASS+CLEAN)
